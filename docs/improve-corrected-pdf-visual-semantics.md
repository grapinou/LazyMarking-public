# Amélioration de la lisibilité du PDF corrigé

## Problème UX

Le PDF corrigé représentait seulement le corrigé attendu : cercle vert sur une bonne réponse et croix rouge sur chaque mauvaise réponse. Il ne montrait pas le vecteur de réponses effectivement attribué à l'élève après détection et, le cas échéant, review manuelle.

Cette convention rendait un score nul difficile à comprendre : la bonne réponse pouvait être entourée en vert tandis que l'élève avait effectivement sélectionné une autre proposition, sans que cette sélection soit visible.

## Convention graphique retenue

| Réponse attendue | Réponse effective | Rendu |
|---:|---:|---|
| 1 | 1 | cercle vert extérieur et cercle bleu intérieur |
| 1 | 0 | cercle vert extérieur uniquement |
| 0 | 1 | cercle bleu intérieur et croix rouge |
| 0 | 0 | aucune marque |

Le vert indique exclusivement la réponse correcte attendue. Le bleu indique exclusivement une réponse considérée cochée par l'élève. La croix rouge signale uniquement une mauvaise proposition effectivement cochée.

Les symboles et scores des questions (`correct`, `partial`, `incorrect`, puis `x.xx/total`) restent inchangés et proviennent toujours de `config.QuestionMark`.

## Architecture du rendu

La signature de `DrawMarking` reçoit désormais deux vecteurs distincts, `expectedAnswers` et `effectiveAnswers`, en plus des positions des réponses.

Une petite fonction pure `answerMarks` transforme chaque couple `(expected, effective)` en trois décisions graphiques : réponse attendue, réponse effective et sélection incorrecte. Cette séparation permet de tester les quatre cas sans test pixel fragile et laisse à `DrawMarking` uniquement les opérations OpenCV.

Le cercle attendu est dessiné à l'extérieur de la bulle en vert. Le cercle effectif est plus petit et bleu. Pour une mauvaise sélection, la croix rouge dépasse légèrement le cercle bleu afin que les deux informations restent visibles.

## Prise en compte des réponses effectives

Le rendu ne recalcule et ne modifie aucun état. Il reçoit les mêmes réponses effectives que le scoring :

```text
effective = COALESCE(reviewed_state, detected_state)
```

Après review, une décision humaine remplace donc visuellement la détection automatique pour la réponse concernée. Les réponses sans review conservent leur état détecté.

Les réponses attendues restent utilisées uniquement pour indiquer le corrigé. Les réponses effectives restent utilisées par `ScoreQuestion` et servent maintenant aussi au marqueur bleu.

## Génération initiale

`MarkingStudentExam` possédait déjà :

- `answersQCM`, le vecteur global des réponses attendues ;
- `answersState`, le vecteur global des états détectés de l'élève.

Pour chaque page, les deux vecteurs sont désormais consommés selon `len(page.Content.Answers)` puis transmis séparément à `DrawMarking`. Le calcul des questions et leurs positions reste inchangé.

## Régénération après review

`regenerateCorrectedCopy` construit déjà les vecteurs effectifs par question à partir de `ListEffectiveMarkingAnswersForArtifacts`. Ces vecteurs sont aplatis dans l'ordre historique question/réponse, parallèlement au vecteur attendu.

Le curseur de snapshots transmet pour chaque page :

- la tranche de `QuestionMark` correspondant aux cercles de questions présents ;
- la tranche des réponses attendues correspondant aux cercles de réponses présents ;
- la tranche des réponses effectives correspondant aux mêmes cercles de réponses.

Les contrôles existants de cohérence avec les scores persistés restent inchangés.

## Protection contre les questions sur plusieurs pages

Le découpage des réponses demeure indépendant du découpage des questions. Le curseur conserve :

- un `questionOffset`, avancé par le nombre de cercles de questions de la page ;
- un `answerOffset`, avancé par le nombre de cercles de réponses de la page.

Le même `answerOffset` découpe les deux vecteurs parallèles attendu/effectif, avec des contrôles de dépassement séparés. Une page peut ainsi contenir la fin des réponses d'une question sans contenir son cercle, puis le début d'une autre question. L'ancienne hypothèse « réponses de la page = réponses des questions présentes sur la page » n'est pas réintroduite.

Le test de pagination utilise notamment une première page contenant deux réponses, suivie d'une page contenant les réponses restantes, et vérifie que les deux vecteurs sont restitués intégralement dans leur ordre propre.

## Fichiers modifiés

- `internal/handlers/tools/drawMarking.go` : nouvelle sémantique graphique et signature distinguant attendu/effectif.
- `internal/handlers/tools/drawMarking_test.go` : tests purs des quatre combinaisons graphiques.
- `internal/handlers/tools/markingStudentExam.go` : transmission des états détectés au rendu initial.
- `internal/handlers/tools/markingArtifactsGenerator.go` : transmission des états effectifs après review et maintien des offsets indépendants.
- `internal/handlers/tools/markingArtifactsGenerator_test.go` : couverture parallèle des vecteurs attendu/effectif à travers les pages.
- `docs/improve-corrected-pdf-visual-semantics.md` : présent rapport.

## Tests ajoutés ou modifiés

`TestAnswerMarks` couvre :

- bonne réponse cochée ;
- bonne réponse non cochée ;
- mauvaise réponse cochée ;
- mauvaise réponse non cochée.

`TestMarkingSnapshotCursorPagination` vérifie maintenant simultanément le vecteur attendu et le vecteur effectif pour la pagination classique, une question traversant deux pages et une page contenant des réponses sans question.

Les tests d'incohérence de snapshots continuent de protéger les dépassements et consommations incomplètes.

## Validation

- `go test ./internal/handlers/tools` : réussi.
- `go test ./...` : réussi.
- `git diff --check` : réussi.
- Recherche de tous les appels à `DrawMarking` : seuls les deux appels attendus existent et utilisent la nouvelle signature.

## Vérification visuelle

Un PNG synthétique sans donnée personnelle a été généré hors du dépôt avec les quatre combinaisons. La vérification visuelle confirme :

- deux cercles concentriques vert/bleu pour une bonne réponse cochée ;
- cercle vert seul pour une bonne réponse non cochée ;
- cercle bleu et croix rouge nettement visibles pour une mauvaise réponse cochée ;
- aucune marque pour une mauvaise réponse non cochée.

L'artefact temporaire de vérification se trouve à `/tmp/corrected-pdf-visual-semantics.png` et ne fait pas partie du dépôt.

## Limites ou améliorations futures

Le rendu repose encore uniquement sur les couleurs et les formes, sans légende intégrée dans chaque PDF. Les formes distinctes permettent déjà la lecture des cas et évitent de dépendre seulement de la couleur. Une légende explicite pourrait être envisagée séparément après retour d'usage, sans modifier la règle de scoring.
