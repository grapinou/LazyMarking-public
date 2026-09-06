# Régénération des artefacts Marking après review

## Résultat

- Service : `RegenerateMarkingArtifacts`.
- Source autoritative : résultats persistés, reviews, snapshots historiques et pages alignées durables.
- Déclenchement : uniquement lorsque `artifacts_revision < review_revision`.
- No-op : oui lorsque les deux révisions sont égales.
- Migration : non.

## Reconstruction

`corrected.pdf` est reconstruit copie par copie depuis les PNG de
`marking_aligned_pages`. Chaque page est résolue avec les contrôles existants
d'ownership, de chemin, de hash et de dimensions, puis copiée dans un répertoire
temporaire avant annotation. Les PNG alignés durables ne sont jamais modifiés.

Les états de question et les scores proviennent de la DB. Les réponses utilisées
pour vérifier leur cohérence sont les `effective_state`, soit
`COALESCE(reviewed_state, detected_state)`. La règle partagée
`internal/markingscoring.ScoreQuestion` est réappliquée comme invariant : une
incohérence entre snapshots, effective states et résultats persistés interrompt
la génération. Aucun OpenCV de détection, aucune homographie et aucun
`GetAnswerDetections` ne sont relancés. `DrawMarking` reste utilisé uniquement
pour le rendu sur une copie temporaire de la page alignée.

`mark-table.pdf` est reconstruit à partir des scores effectifs des copies, des
snapshots élèves/QCM et des résultats de question effectifs. Les fonctions
existantes de moyenne, médiane, écart-type, compétences et thèmes sont
réutilisées sans changement de formule. Les copies non corrigées sont
reconstruites depuis leur outcome et leur snapshot. Les noms techniques des
pages dont le QR était illisible ne sont pas persistés dans le modèle
autoritatif ; cette ancienne liste runtime reste donc vide lors d'une
régénération, tandis que les copies non corrigées restent listées.

`corrected_NOT.pdf` contient uniquement les pages/copies non corrigées, sur
lesquelles aucune answer review n'est possible. Il est indépendant des reviews
et n'est ni régénéré ni inclus dans `artifacts_revision`.

## Publication filesystem

Les deux PDF sont d'abord produits intégralement dans un répertoire temporaire
confiné au workspace du job. Ils doivent être des fichiers réguliers, non vides,
non symlinkés, sous ce répertoire, avec un en-tête PDF valide.

La publication multi-fichiers ne peut pas être atomique au sens strict. Le
protocole adopté est donc détectable et restaurable :

1. sauvegarde par rename des deux artefacts canoniques ;
2. rename des deux nouveaux PDF vers leurs noms canoniques ;
3. UPDATE conditionnel de `artifacts_revision` avec l'ownership, le statut
   `success`, la review revision attendue et l'ancienne artifacts revision ;
4. suppression des sauvegardes seulement après succès DB.

Une publication partielle ou un échec DB déclenche une restauration best-effort
des deux anciens PDF. Dans tous les cas, `artifacts_revision` n'avance pas si les
deux fichiers n'ont pas été publiés. Si une review concurrente change
`review_revision` pendant la génération, l'UPDATE affecte zéro ligne, les
anciens artefacts sont restaurés et le job reste détectablement stale. Une
interruption brutale peut laisser des temporaires ou sauvegardes, mais ne peut
pas déclarer le job current ; une nouvelle régénération reste possible.

Un échec de régénération ne modifie pas le lifecycle automatique `success` et
est exposé par une erreur applicative typée. Les jobs legacy sans politique de
review ou sans artefacts/pages alignées requis sont refusés sans fallback vers
le scan brut ou l'ancien `corrected.pdf`.

## Tests

- current vers no-op sans accès filesystem ;
- publication des deux PDF puis avancement de révision ;
- conflit concurrent avec restauration des deux PDF ;
- échec de génération laissant les canoniques et la DB inchangés ;
- nettoyage des temporaires ;
- ownership et optimistic locking SQL de la révision d'artefacts ;
- résolution existante des pages alignées manquantes, corrompues, cross-user,
  aux mauvaises dimensions, au mauvais hash et aux chemins dangereux ;
- cohérence effective state / scoring vérifiée par le générateur.

## Validations

- `sqlc generate -f db/sqlc.yaml` : réussi.
- `./scripts/check.sh` : réussi.
- `go test -race ./...` : réussi.
- `git diff --check` : réussi.

## Fichiers modifiés

- `db/query/markingReviews.sql`
- `internal/db/markingReviews.sql.go` (généré par sqlc)
- `internal/db/markingReviews_test.go`
- `internal/handlers/tools/markingArtifactsRegeneration.go`
- `internal/handlers/tools/markingArtifactsGenerator.go`
- `internal/handlers/tools/markingArtifactsRegeneration_test.go`
- `docs/audits/marking-artifacts-regeneration.md`

PDF production modifié pendant ce jalon : non. UX modifiée : non. Lifecycle
modifié : non. Migration ajoutée : non.
