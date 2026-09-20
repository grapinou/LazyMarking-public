# P1.2 — Accès aux copies manquantes et bilan pédagogique cumulé

## Problèmes observés

P1.1 avait établi une page de bilan par `exams_generated`, une sélection
commune des résultats courants et un PDF généré à la demande. Cette architecture
est conservée, ainsi que les jobs et leurs documents indépendants.

Deux limites subsistaient :

- dans la page principale Correction, l'évaluation était identifiable mais
  les actions visibles privilégiaient le bilan et les documents des jobs ;
  l'accès au dépôt prérempli de rattrapage demandait un détour ;
- le PDF cumulé présentait correctement les états individuels mais avait perdu
  les statistiques pédagogiques du tableau PDF historique.

L'audit a porté sur les routes et tests P1.1, `table_marking.html`,
`generation_results.html`, les modèles de navigation, le générateur historique
`TypstBuildMarkTable`, son template et ses producteurs de statistiques.
Les modifications P1/P1.1 non commitées ont été conservées. `README.md` et
`reset.sh`, préexistants et hors chantier, n'ont pas été modifiés.

## UX finale

Dans `/dashboard/marking`, chaque ligne d'évaluation générée propose maintenant :

- **Voir les résultats** ;
- **Ajouter les copies manquantes**.

Le bouton du bilan de l'évaluation reste présent, au-dessus des résultats.
Les actions sont toujours attachées à une génération précise. Aucune nouvelle
route de rattrapage n'a été créée : le lien existant
`/dashboard/marking?exam_generated_id=…` ouvre le dépôt avec un champ caché et
le nom de l'évaluation affiché. Dans ce formulaire prérempli, les listes
d'autres évaluations et des anciens imports sont masquées pour laisser le
choix du PDF et le retour au bilan.

Le POST conserve ses contrôles d'ownership et de génération réussie. Le
traitement crée normalement un nouveau job, passe par les éventuelles revues
et revient au bilan de la même génération. Le tri français nom/prénom du
tableau et du PDF est inchangé.

L'**historique des imports récents** reste en dessous des actions par
évaluation, avec des boutons secondaires. L'historique propre au bilan et
les liens « Documents de cet import » sont conservés. Les PDF historiques
ne sont ni fusionnés ni supprimés. « Import source » reste du texte.

## Ancien bilan : contenu réellement retrouvé

Sources :

- `internal/config/ref_mark_table.txt` ;
- `internal/handlers/tools/typstBuildMarkTable.go` ;
- `computeStatMarking.go`, `getThemeSkill.go`, `agregateThemeSkill.go` ;
- appels depuis `ProcessMarking` et `defaultMarkingArtifactsGenerator.Generate`.

| Section historique | Calcul ou contenu |
|---|---|
| Moyenne | moyenne arithmétique des scores des `MarkExam` du lot |
| Médiane | score central, ou moyenne des deux scores centraux |
| Écart-type | écart-type de population, division par l'effectif |
| Notes | nom de l'élève et score sur le barème |
| Compétences globales | somme des points obtenus / somme des points possibles, par compétence |
| Compétences par thème | même ratio, par couple thème-compétence |
| Informations de correction | pages sans QR détecté et élèves non corrigés |

Il n'existait pas de tableau de réussite par question, ni de distribution,
minimum ou maximum dans ce document. La section par question est donc un
ajout P1.2 ; les métriques de notes et de compétences sont restaurées.

Les copies non corrigées étaient listées séparément et n'étaient pas ajoutées
comme des zéros à `markExams`. Les résultats automatiques pouvaient être
provisoires ; les protections de revue et d'artefacts intervenaient ensuite
sur leur publication. Le bilan pédagogique courant exclut explicitement
chaque copie dont une revue reste en attente.

## Architecture statistique

### Une seule sélection et un état cohérent

`ListCurrentExamResultsForGeneration` conserve les règles P1 : correction
réussie prioritaire sur un incident ultérieur, correction réussie la plus
récente, exclusion des jobs dont `status` ou `status_pdf` n'est pas `success`.

Cette même requête charge maintenant, pour chaque résultat sélectionné :

- le snapshot `student_exam_content` de son élève, limité à l'utilisateur ;
- les résultats persistés de ses questions : index, état, demi-points et barème.

Les questions sont renvoyées en JSON par une sous-requête corrélée au
`copy_result_id`. Les notes, les états de revue et le détail pédagogique sont
ainsi lus dans **la même instruction SQL**. Une modification transactionnelle
de revue ne peut pas produire une moyenne utilisant l'ancien score et des
statistiques de question utilisant le nouveau.

`loadMarkingGeneration` transmet ces mêmes lignes à la vue individuelle et à
`buildMarkingPedagogicalSummary`. Il n'y a aucune nouvelle sélection de jobs,
aucun accès à la banque de questions actuelle et aucun nouveau calcul de
correction des réponses.

### Population et moyenne

Une copie participe aux statistiques si elle est `corrected`, sans candidat
pending, avec une note et un barème positif. La disponibilité d'une note
finalisée est partagée avec la vue individuelle via `hasFinalMarkingScore`.

| État courant | Moyenne, médiane, écart-type | Questions et compétences |
|---|---|---|
| corrigée sans pending | incluse | incluse si les détails sont exploitables |
| revue pending | exclue en totalité | exclue en totalité |
| `not_seen` | exclue, jamais assimilée à zéro | exclue |
| `incomplete` | exclue, jamais assimilée à zéro | exclue |
| `error` | exclue, jamais assimilée à zéro | exclue |
| note absente ou barème non positif | exclue | exclue |

Les fonctions historiques `Mean`, `Median` et `StdDev` sont réutilisées.
Les demi-points persistés sont convertis en points sans arrondi intermédiaire ;
l'affichage utilise deux décimales françaises.

En cas de barèmes différents, les statistiques de notes sont présentées
séparément pour chaque barème, avec leur effectif. Aucune moyenne de notes
incomparables ou conversion implicite sur 20 n'est faite. Sans note exploitable,
le PDF indique que les statistiques sont indisponibles.

### Réussite par question et variantes

Les questions et les réponses sont mélangées à la génération. Leur position
sur une copie ne constitue donc pas une identité commune.
`BuildQuestionCtx` conserve dans chaque snapshot `Tags.MainQuestionID`, y
compris lorsqu'une variante est tirée. Il conserve aussi l'énoncé, l'image,
les réponses attendues et les tags historiques.

Le regroupement utilise :

1. l'identité de famille issue du snapshot ;
2. une signature du contenu, de l'image, des tags/barème et des réponses
   attendues triées.

Les positions des cercles, symboles et permutations des réponses sont exclus
de la signature. Deux familles distinctes ne sont jamais fusionnées, même
avec le même énoncé et la même compétence. Deux variantes distinctes d'une
famille apparaissent séparément. Si l'identité de famille manque dans un ancien
snapshot, seuls des contenus et métadonnées identiques sont regroupés.

Le PDF affiche un repère de question, une version lorsqu'il y en a plusieurs
et un extrait de l'énoncé. Ces repères sont explicitement ceux du bilan, pas
les numéros imprimés sur chaque copie. Pour chaque ligne :

- **réussite** = points obtenus / points possibles, crédit partiel inclus ;
- nombre de réponses entièrement réussies ;
- nombre d'occurrences évaluées dans la population incluse.

Le pourcentage mesure donc les points obtenus, et non seulement la proportion
de réponses intégralement correctes. Les barèmes restent pris en compte.

### Compétences et couverture historique

Les résultats par question sont associés au snapshot par leur index, puis
convertis en `config.QuestionMark`. `GetThemeSkill` et `AgregateThemeSkill`
effectuent les agrégations historiques inchangées. Le ratio de réussite est
factorisé dans `MarkingSuccessPercentage`, également utilisé par l'ancien
générateur local. Les tags manquants ne produisent pas de catégories anonymes.

Avant d'utiliser les détails d'une copie, le code vérifie la couverture des
questions, les index uniques, les états/demi-points, les barèmes du snapshot
et l'égalité des sommes avec la note de copie. Un détail absent ou incohérent
ne bloque pas le bilan : la note persistée reste dans les statistiques de
notes, mais cette copie est exclue des statistiques de questions et de
compétences. Le PDF affiche la couverture détaillée et explique cette exclusion.

## PDF final

Le document contient, dans cet ordre :

1. évaluation, classe, génération et compteurs de copies ;
2. résumé pédagogique : population retenue, exclusions, moyenne, médiane,
   écart-type, effectifs et éventuels barèmes distincts ;
3. réussite par question/version avec couverture et définition du taux ;
4. réussite des compétences globales ;
5. réussite des compétences par thème ;
6. résultats individuels, états et notes, dans le tri français P1.1.

Les sections de compétences sans tags exploitables sont omises. Le HTML garde
son tableau compact ; les informations communes viennent toujours des mêmes
résultats courants.

Garanties P1.1 conservées : génération à la demande identifiée par
`exam_generated_id`, `Cache-Control: no-store`, aucun cache persistant,
workspace isolé `bilan-<uuid>`, nettoyage normal et après erreur, purge existante,
timeout, annulation via contexte, échappement de tous les textes Typst,
en-têtes de tableau répétés sur les pages suivantes.

Un nouveau téléchargement reflète un nouvel import réussi ou une décision
humaine modifiée. Les révisions et régénérations des PDF par job restent
indépendantes et inchangées.

## Composants principaux modifiés

- `table_marking.html` : accès direct et hiérarchie des actions ;
- `markingWorkflow.sql` et sa sortie sqlc : détails pédagogiques dans la lecture
  courante existante ;
- `generation.go`, `workflowView.go`, `pedagogicalSummary.go` : modèle partagé,
  population finalisée et agrégations ;
- `internal/templates/data/marking.go` : vues statistiques typées ;
- `markingGenerationPDF.go` : sections pédagogiques ;
- `computeStatMarking.go`, `typstBuildMarkTable.go` : ratio de réussite partagé ;
- tests P1.1 étendus et `pedagogicalSummary_test.go`.

## Tests

Les fixtures et helpers HTTP/SQLite/PDF existants sont réutilisés.

- Navigation : boutons avant l'historique, bon `exam_generated_id`, aucune
  génération étrangère ou non prête, formulaire prérempli sans choix parasite.
- Premier lot : 10/20 et 14/20 donnent 12/20, médiane 12, écart-type 2 ;
  réussite globale et par thème-compétence de 60 %.
- Rattrapage : ajout de 18/20, moyenne de 14/20 ; les anciens `not_seen` et
  `incomplete` ne masquent pas les corrections acquises.
- Pending : exclusion complète de la copie, y compris pour ses autres questions.
- Revue confirmée puis modifiée : actualisation de la note, de la moyenne
  (13,33/20 après modification) et de la première question (83,33 %).
- Questions déterministes : cinq questions avec des taux de 100 %, 100 %,
  75 %, 25 % et 0 % dans le premier lot ; ordre des questions inversé sur une copie.
- Variantes : même famille avec contenus distincts séparés ; réponses mélangées
  regroupées ; familles différentes non fusionnées ; fallback historique prudent.
- Données anciennes : absence/incohérence de détail, note conservée et couverture
  réduite explicite ; barèmes différents séparés ; population vide sans faux zéro.
- Tests P1.1 enrichis : moyenne après correction plus récente, exclusion des jobs
  failed/running et conservation des anciens scores.
- PDF : compilation réelle et extraction du texte à chaque étape ; document vide,
  soixante lignes de statistiques de questions et cent élèves, en-têtes répétés,
  échappement, timeout/contexte et nettoyage conservés. L'annulation est testée.

### Validation technique

- `go test ./...` : succès.
- `go test ./internal/handlers/marking ./internal/handlers/tools ./internal/db -count=1` : succès.
- Tests ciblés de pédagogie et de compilation/extraction PDF : succès, Typst et
  `pdftotext` présents et utilisés.
- `sqlc generate -f db/sqlc.yaml` : succès.
- `git diff --check` : succès.

Les suites privées historiques de scans restent opt-in et sont sautées lorsque
leurs variables `LAZYMARKING_TEST_*` et de calibration ne sont pas renseignées.
Les tests PDF P1.2 ont bien été exécutés ; les scans terrain n'ont pas été
réimportés automatiquement.

## P1.3 — Identification du bilan

Le titre principal HTML identifie directement la génération sous la forme
`{classe} — {nom de l'évaluation}`, sans répéter le libellé générique du bilan.
Il accepte le retour à la ligne pour rester lisible avec un intitulé long.

Le PDF autonome conserve le titre `Bilan de l'évaluation`, suivi de la même
identité `{classe} — {nom de l'évaluation}`. Son contenu pédagogique, ses
compteurs, ses états et ses résultats individuels sont inchangés.

Le téléchargement utilise `{classe}-{évaluation}-bilan.pdf`, normalisé en
minuscules ASCII : accents décomposés, espaces et caractères spéciaux remplacés
par un seul tiret, séparateurs de début et de fin supprimés. Le symbole `°` est
omis afin que `n°2` devienne `n2`. Le nom est borné à 120 caractères avant
l'extension, tout en conservant le suffixe `-bilan.pdf`. Si aucune métadonnée
n'est exploitable, le fallback est `evaluation-bilan.pdf`.

Les tests couvrent le titre HTML, les deux titres extraits du PDF réel, le
`Content-Disposition`, le type PDF et `no-store`, les accents et caractères
spéciaux (`6e B — Électricité & énergie`), les métadonnées vides et la longueur
maximale du nom.

## Compatibilité

Aucune migration, colonne ou table ajoutée. Aucune nouvelle dépendance.
Les bases existantes utilisent les snapshots et résultats persistés actuels.

Les nouvelles lectures ont été exécutées sur des sauvegardes SQLite temporaires
des trois bases `testdata/smoke`, `testdata/real` et `testdata/2026-2027`, après
application des seules migrations déjà présentes jusqu'à la version 44.
Elles passent sur toutes les générations réussies de ces copies.

Sur la copie 2026-2027, la génération 2 comporte 34 états d'élèves, dont trente
notes finalisées avec détails complets pour six familles de questions. Le
contrôle des données donne une moyenne de 3,73/6 pour ces trente copies.
Ce contrôle de données ne constitue pas une réexécution de détection des scans.

Les empreintes SHA-256 des bases sources et de `lot-1.pdf` / `lot-2.pdf` sont
inchangées. Les trois runtimes existants n'ont pas été modifiés. Aucun commit
ni push n'a été effectué.

### Procédure terrain courte

Utiliser une copie isolée du runtime et une sauvegarde SQLite indépendante,
avec les artefacts de génération et images copiés. Ne pas lancer un script qui
relie ce runtime à la base source. Pour partir d'un premier lot vierge, utiliser
une sauvegarde prise avant les imports, puisque les résultats existants sont
volontairement conservés.

1. Déposer `testdata/2026-2027/scans/lot-1.pdf`, terminer les revues, ouvrir les
   résultats et télécharger le bilan pédagogique.
2. Depuis Correction ou le bilan, cliquer « Ajouter les copies manquantes » ;
   vérifier l'évaluation déjà sélectionnée et déposer `lot-2.pdf`.
3. Terminer les revues, télécharger à nouveau le bilan : vérifier les anciennes
   notes, les ajouts, les effectifs, la moyenne et les statistiques de questions.
4. Modifier une décision humaine et retélécharger pour vérifier la mise à jour.
5. Ouvrir séparément les documents des deux imports dans l'historique.

## Points restant ouverts

Le replay navigateur des deux scans réels n'a pas été exécuté dans ce chantier ;
la procédure ci-dessus permet cette validation terrain complémentaire.
Les helpers de scans historiques restent ciblés sur leurs anciennes fixtures,
comme documenté dans P1.1. Aucun blocage d'implémentation P1.2 identifié.
