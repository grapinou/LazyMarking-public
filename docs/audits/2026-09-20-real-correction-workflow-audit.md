# Audit du cycle réel de correction — 20 septembre 2026

## État initial

### Création et préparation

Une évaluation est créée depuis `/dashboard/exams`, puis sa génération est
lancée par `POST /dashboard/exams/generate`. `GenerateExamsHandler` crée une
ligne `exams_generated` à l'état `running`, puis une ligne `student_exam` et les
snapshots `student_exam_content` / `student_exam_page_content` pour chaque
élève. Les références PNG de chaque page sont enregistrées dans le workspace
de la génération et décrites en base. La génération ne passe à `success` que si
la couverture de références est complète et non ambiguë.

Les artefacts nécessaires à une correction moderne sont donc : la génération
`success`, les snapshots complets, les références de pages avec leurs
métadonnées d'intégrité et le PDF distribué aux élèves.

### Import, détection et correction

Les routes du parcours sont enregistrées dans
`internal/handlers/marking/routes.go` :

| Méthode et route | Handler | Rôle |
|---|---|---|
| `GET /dashboard/marking` | `AddPdfFormMarkingHandler` | choix de la génération et dépôt d'un lot |
| `POST /dashboard/marking/processing` | `ProcessingMarkingHandler` | admission du PDF, création du job, lancement asynchrone |
| `GET /dashboard/marking/progress` | `ProgressMarkingHandler` | progression du job |
| `GET /dashboard/marking/success` | `SuccessMarkingProcessingHandler` | résultat, revue et artefacts |
| `GET /dashboard/marking/review` | `MarkingReviewHandler` | décision humaine séquentielle |
| `POST /dashboard/marking/review/apply` | `ApplyMarkingReviewHandler` | persistance et recalcul atomique |
| `GET /dashboard/marking/review/crop` | `MarkingReviewCropHandler` | crop sécurisé depuis la page alignée |
| `POST /dashboard/marking/artifacts/regenerate` | `RegenerateMarkingArtifactsHandler` | régénération des PDF après revue |
| `GET /dashboard/marking/servePDF` | `ServeFullMarkingPdfHandler` | téléchargement contrôlé des artefacts |

Chaque dépôt crée un `marking_job` indépendant, lié à une seule
`exams_generated`. Le PDF est découpé, converti en images, puis les QR sont lus
en parallèle. Un QR qui appartient à un autre utilisateur ou à une autre
génération fait échouer le lot. Les QR valides sont groupés par
`student_exam_id`; les pages manquantes, dupliquées ou hors plage produisent un
outcome `incomplete`.

Une copie complète est alignée sur ses références historiques, ses réponses
sont détectées et son résultat est persisté atomiquement dans
`marking_copy_results`, `marking_question_results` et
`marking_answer_detections`. Les pages alignées non annotées sont conservées et
contrôlées par taille et SHA-256. Une erreur locale sur une copie complète
produit `error`. Une copie attendue sans QR dans le lot produit `not_seen`.

`CompleteMarkingJobWithResults` ne permet le passage à `success` que lorsque
toutes les copies attendues ont un outcome terminal et que chaque copie
`corrected` possède exactement ses questions, réponses et pages alignées.

### Vérification humaine

La politique hybride marque les désaccords de détecteurs avec
`review_reason = detector_disagreement` et ne leur attribue pas
`automatic_state`. Une ligne `marking_answer_reviews` représente la décision
humaine. `ApplyMarkingAnswerReview` verrouille par révision, recalcule la
question et la copie dans la même transaction, puis avance
`marking_jobs.review_revision`. Les PDF deviennent obsolètes si l'état effectif
change et `artifacts_revision` ne rejoint la révision courante qu'après une
régénération réussie.

Avant ce chantier, l'écran ne proposait que les candidats encore en attente.
Le service savait modifier une décision existante, mais aucun parcours normal
ne permettait de la rouvrir.

### Résultats, fichiers et reprise

Les résultats et artefacts étaient affichés lot par lot. `corrected.pdf` et le
tableau des notes étaient masqués pendant une revue hybride ou lorsqu'ils
étaient obsolètes. `corrected_NOT.pdf` restait indépendant des décisions de
revue.

Le workspace d'un job réussi est `assets/tmp/<utilisateur>/marking-<job>`.
L'arrêt du navigateur ne coupe pas le traitement serveur, mais l'interface ne
listait aucun job : sans conserver son URL, l'utilisateur ne pouvait pas
retrouver la progression ou le résultat. Au démarrage du serveur, les jobs
restés `running` sont volontairement convertis en `failed` et leur workspace
partiel est nettoyé.

## Problèmes identifiés

### P1 — aucun résultat courant pour plusieurs imports

- **Scénario :** lot principal corrigé, puis import d'une ou plusieurs copies
  de rattrapage.
- **Cause :** chaque job persistait à nouveau un outcome pour toute la classe ;
  les élèves absents du nouveau PDF devenaient `not_seen` dans ce job, et les
  écrans ne lisaient qu'un job.
- **Conséquence :** les anciennes corrections n'étaient pas supprimées, mais
  aucun écran ne réunissait les présents et les rattrapages. Le tableau du
  dernier lot pouvait donner l'impression que les copies déjà corrigées
  avaient disparu.
- **Gravité :** haute.

### P1 — parcours introuvable après fermeture de l'écran

- **Scénario :** import lancé, onglet fermé, retour ultérieur dans LazyMarking.
- **Cause :** l'URL portant `job_id` était le seul point d'accès au job.
- **Conséquence :** progression, revue ou résultat existaient toujours, mais
  n'étaient plus découvrables dans l'interface.
- **Gravité :** haute.

### P1 — lot sans aucune copie corrigée traité comme panne technique

- **Scénario :** aucun QR lisible, ou toutes les copies reconnues sont
  incomplètes / en erreur.
- **Cause :** `ProcessMarking` échouait explicitement lorsque `qrDatas` ou
  `markExams` était vide ; `TypstBuildMarkTable` supposait au moins une copie.
- **Conséquence :** les outcomes utiles pouvaient avoir été persistés, mais le
  job finissait `failed`, le workspace et les pages de contrôle étaient perdus,
  et l'utilisateur recevait un message d'échec générique.
- **Gravité :** haute.

### P2 — résultat présenté comme terminé avant revue

- **Scénario :** un job technique est `success`, mais une ou plusieurs
  réponses hybrides attendent une décision.
- **Cause :** le titre annonçait « Correction terminée ». La base contient un
  score provisoire nécessaire au recalcul, sans vue cumulée capable de le
  masquer copie par copie.
- **Conséquence :** les PDF étaient correctement bloqués, mais le vocabulaire
  et l'absence d'un état courant par élève restaient ambigus.
- **Gravité :** moyenne.

### P2 — décision humaine non corrigeable depuis l'interface

- **Scénario :** le professeur valide la mauvaise case puis revient sur le
  résultat.
- **Cause :** le GET de revue redirigeait dès qu'il ne restait aucun candidat
  pending.
- **Conséquence :** le service transactionnel supportait l'override, mais il
  était inaccessible dans le workflow réel.
- **Gravité :** moyenne.

## Invariants retenus

1. Un job d'import est un lot historique indépendant et n'écrase jamais un
   autre job.
2. L'état courant d'une copie utilise sa correction réussie la plus récente.
   Un `not_seen`, `incomplete` ou `error` ultérieur ne dégrade pas une
   correction déjà acquise.
3. Si aucune correction n'existe pour une copie, son incident terminal le plus
   récent constitue son état courant.
4. Un nouvel import de la même génération peut ajouter une ou plusieurs copies
   sans invalider les résultats précédents.
5. Une note dont au moins une réponse attend une revue n'est pas affichée comme
   définitive.
6. Une décision humaine est atomique avec le recalcul de la question et de la
   copie ; le bilan courant lit immédiatement ce score recalculé.
7. Une décision existante peut être rouverte, corrigée et entraîne la même
   invalidation / régénération des artefacts qu'une première décision.
8. Un lot dont toutes les copies sont `not_seen`, `incomplete` ou `error` est un
   résultat pédagogique terminal, pas une panne technique, si le pipeline et
   la persistance ont fonctionné.
9. Un job en cours, en revue ou réussi doit rester accessible depuis le point
   d'entrée Correction.
10. Les transitions critiques restent ownership-aware, atomiques et protégées
    par leurs contraintes d'unicité ou révisions optimistes.
11. Le dépôt répété du même PDF crée volontairement un nouveau lot : l'import
    n'est pas déclaré idempotent, car un second dépôt peut être un rattrapage ou
    une nouvelle acquisition. Le calcul du bilan, la confirmation identique
    d'une revue et la régénération déjà à jour sont idempotents.

## Modifications apportées

### Bilan cumulé et imports successifs

`db/query/markingWorkflow.sql` ajoute deux lectures sans nouvelle table :

- l'historique récent des jobs possédés ;
- l'état courant de chaque `student_exam` pour la génération d'un job.

La sélection classe d'abord les corrections réussies, puis prend la plus
récente. Ainsi, les `not_seen` structurels des lots de rattrapage ne masquent
pas une correction antérieure. Une correction plus récente remplace bien une
ancienne correction. Les jobs `running` et `failed`, potentiellement partiels,
sont exclus du bilan.

`workflowView.go` transforme ces lignes en modèles de vue typés. La page de
résultat affiche désormais un bilan cumulé par élève, l'import source et le
score courant. Une copie avec revue pending affiche « Après vérification » à la
place de la note.

### Reprise visible

La page `/dashboard/marking` liste les vingt jobs récents. Selon l'état réel,
elle propose la progression, la reprise de la vérification ou le résultat. Les
jobs interrompus restent visibles et demandent explicitement un nouvel import.

### Revue révisable

`MarkingReviewHandler` peut charger un candidat déjà revu, vérifie toujours son
appartenance au job et à l'utilisateur, préremplit la décision actuelle et
offre une navigation entre les décisions. Le POST existant reste l'unique
écriture et conserve le verrouillage optimiste. Après une modification, les
PDF sont régénérés ou restent marqués obsolètes avec le retry existant.

### Lots sans copie corrigée

`ProcessMarking` ne transforme plus l'absence de QR ou de copie corrigée en
échec artificiel. Il persiste la couverture terminale, produit les artefacts de
synthèse et le PDF des pages à contrôler, puis utilise la finalisation normale.
`TypstBuildMarkTable` accepte maintenant une liste de notes vide.

Aucune migration n'a été ajoutée : le correctif repose sur les tables et
colonnes additives existantes jusqu'à `0044`. Les fichiers SQLC ont été
régénérés depuis les requêtes sources.

## Tests

Les tests ajoutés ou étendus couvrent :

- lot principal puis rattrapage ;
- plusieurs vagues successives avec `not_seen` et `incomplete` ;
- conservation des corrections précédentes ;
- exclusion d'un job échoué du résultat courant ;
- revue pending qui masque la note ;
- décision humaine appliquée puis visible immédiatement dans le bilan ;
- historique de jobs `running`, `success` avec revue et `failed` ;
- réouverture et correction d'une décision existante ;
- finalisation DB d'un lot entièrement `not_seen` ;
- génération du tableau Typst lorsqu'aucune copie n'est corrigée.

Validation exécutée :

- `go test ./...` : succès ;
- `go test ./internal/handlers/marking ./internal/handlers/tools ./internal/db -count=1` : succès ;
- copies temporaires de `testdata/smoke/app.db`, `testdata/real/app.db` et
  `testdata/2026-2027/app.db` migrées jusqu'à la version 44 : succès, originaux
  non modifiés ;
- `git diff --check` : succès.

Les tests d'intégration sur scans réels restent opt-in et n'ont pas été lancés
automatiquement : ils nécessitent leurs variables d'environnement et, selon le
cas, les corpus privés attendus.

## Résultat final

Le workflow nominal conserve ses garanties de génération, détection,
persistance et revue. Il accepte désormais un lot ne produisant aucune note
sans confondre incident pédagogique et panne technique.

Les imports successifs d'une même génération forment un historique et un bilan
courant cohérent. Le cas absent puis rattrapage ajoute la nouvelle correction
au bilan sans toucher aux anciennes. Les notes en attente de revue sont
explicitement non définitives, et les décisions humaines peuvent être
corrigées avec recalcul et artefacts cohérents.

Un utilisateur qui ferme son onglet retrouve les jobs récents et reprend le bon
écran sans reconstruire ses données.

## Points restant à traiter

### Reprise après arrêt brutal du serveur

Le PDF source n'est pas conservé durablement. Après un arrêt brutal, le
recovery existant transforme le job `running` en `failed`, nettoie son
workspace et l'interface demande un nouvel import. Une reprise exacte au milieu
du pipeline nécessiterait de conserver le PDF source, de rendre le nettoyage
des résultats partiels transactionnel et de définir une politique de rétention.
Ce comportement est maintenant visible et sans ambiguïté, mais il justifie un
chantier de recovery dédié si la reprise sans réimport après crash devient une
exigence d'exploitation.

### Artefact PDF cumulé

Le bilan cumulé fiable est disponible dans l'interface. Les PDF restent des
artefacts de chaque lot historique. Produire un tableau de notes PDF consolidé
sur plusieurs jobs demanderait un générateur d'artefacts au niveau de la
génération et une provenance explicite ; ce n'est pas nécessaire au parcours
corrigé ici, mais peut devenir utile pour l'export administratif.
