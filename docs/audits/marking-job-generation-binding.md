# Liaison d'un job de correction à une génération

Date : 2026-08-31

## 1. Contrat avant/après

Avant, `marking_jobs` ne mémorisait que `user_id`; un `student_exam_id` possédé suffisait donc à charger les snapshots, même s'il provenait d'une autre génération du même utilisateur.

Après, toute nouvelle correction porte `exam_generated_id`. La création exige une génération existante, possédée et `success`. Chaque QR est accepté uniquement si son `student_exam` appartient au même utilisateur et à la génération persistée par le job.

## 2. Migration

La migration Goose `0036_bind_marking_jobs_to_exam_generation.sql` ajoute `marking_jobs.exam_generated_id`, sa FK et deux triggers d'ownership (INSERT et UPDATE). Up et Down sont couverts par un test d'exécution SQLite.

## 3. Compatibilité des anciens marking_jobs

La colonne est nullable uniquement pour préserver honnêtement les lignes historiques, sans backfill arbitraire ni suppression. Ces lignes restent compatibles avec les mises à jour de statut, recovery et purge. Elles ne sont jamais proposées comme nouvelle correction liée.

## 4. FK choisie

La FK cible `exams_generated(id)` avec `ON DELETE RESTRICT`. Une génération réussie est un historique déjà protégé contre la suppression de son `Exam`; une correction liée ne doit donc pas perdre son autorité parent. La purge d'un job reste possible et libère ensuite la relation. Il n'y a pas de `UNIQUE(exam_generated_id)`.

## 5. Ownership relationnel

Les triggers refusent une génération absente ou appartenant à un autre utilisateur. Le trigger INSERT refuse aussi explicitement `NULL`; le trigger UPDATE empêche de délier ou de réaffecter un job existant. Les lignes legacy NULL peuvent néanmoins continuer à changer de statut car ces triggers ne s'exécutent que lors d'une modification de `user_id` ou `exam_generated_id`.

## 6. Validation status success

Le handler vérifie le statut avant de valider/stager le PDF. La requête de création répète atomiquement `id + user_id + status = 'success'`. Les générations running et failed ne créent aucun job. Dans le modèle actuel, `success` est terminal : aucune requête ne ramène une génération réussie vers running ou failed.

## 7. Formulaire/POST

Le formulaire existant contient maintenant un `select` HTML requis alimenté par les générations success déjà chargées. Le POST multipart transmet `exam_generated_id` et `pdffile`, sans route ni JavaScript supplémentaire.

Un ID absent/invalide renvoie 400, une génération absente ou étrangère renvoie 404, et une génération possédée mais non success renvoie 409. Aucune information sur le propriétaire étranger n'est révélée.

## 8. CreateMarkingJob

`CreateMarkingJob` reçoit `user_id` et `exam_generated_id` et utilise `INSERT ... SELECT ... WHERE` avec ownership et statut. Une sélection invalide produit zéro ligne retournée (`sql.ErrNoRows`); le job n'est jamais créé avant validation.

## 9. Scope QR

`ValidateMarkingJobStudentExam` joint `marking_jobs` et `student_exam` sur `exam_generated_id` et `user_id`, puis vérifie job, utilisateur et `student_exam_id`. Cette primitive est appelée immédiatement après décodage JSON du QR, avant ajout aux données agrégées et avant toute lecture de snapshot.

## 10. Mélange génération A/B

Une page B dans un job A échoue avec l'erreur générique `ErrQrOutsideMarkingGeneration`. Le pipeline reçoit une erreur critique, marque le job failed et ne produit aucune synthèse agrégeant A et B.

## 11. Cross-user

Un QR de Bob dans le job d'Alice suit exactement le même refus générique qu'un ID absent ou hors génération. Aucun snapshot ni détail de Bob n'est chargé ou exposé.

## 12. Pages mélangées same-generation

Le regroupement et son tri existants n'ont pas été modifiés. Plusieurs élèves et pages de la génération attendue restent regroupés par `student_exam_id`, puis triés par page, quel que soit l'ordre du PDF. Le test existant de regroupement mélangé continue de passer.

## 13. Lifecycle inchangé

Les états running, success et failed, ainsi que recovery et purge, sont inchangés. Aucun nouvel état n'a été ajouté.

## 14. Recovery/purge

Les requêtes `ListRunningMarkingJobs`, `ListExpiredMarkingJobs`, recovery et purge ne dépendent pas de la nouvelle colonne. Elles fonctionnent donc pour les lignes legacy NULL et les nouvelles lignes liées, sans dépendre de la présence du PDF uploadé.

## 15. Algorithme inchangé

Aucun seuil, traitement SIFT, homographie, RANSAC, calcul full/half/zero, statistique, Typst ou traitement d'image n'a été modifié.

## 16. Persistance des résultats inchangée

Aucune table ni persistance de notes, réponses détectées ou résultats n'a été ajoutée.

## 17. Tests

Tests ajoutés/adaptés :

- migration Up/Down, conservation legacy NULL, lifecycle legacy et `ON DELETE RESTRICT`;
- création possédée success, running, failed, étrangère et absente;
- interdiction d'un INSERT direct NULL ou cross-user;
- scope QR same-generation, autre génération du même utilisateur, cross-user et absent;
- handler multipart : success, ID absent/invalide, génération étrangère, running et failed;
- contrôle de la génération persistée par le handler;
- maintien du test existant sur l'ordre mélangé des élèves/pages.

## 18. sqlc

`sqlc generate -f db/sqlc.yaml` : succès. Les fichiers générés n'ont pas été modifiés manuellement.

## 19. check

`./scripts/check.sh` : succès (modules, formatage, vet, tests, build et `git diff --check`).

## 20. race

`go test -race ./...` : succès.

## 21. Fichiers modifiés

- `db/migrations/0036_bind_marking_jobs_to_exam_generation.sql`
- `db/query/examsGenerated.sql`
- `db/query/markingJobs.sql`
- `internal/db/examsGenerated.sql.go` (généré)
- `internal/db/markingJobs.sql.go` (généré)
- `internal/db/models.go` (généré)
- `internal/db/markingJobGenerationBinding_test.go`
- `internal/handlers/marking/handlers.go`
- `internal/handlers/marking/handlers_test.go`
- `internal/handlers/tools/markingGenerationScope.go`
- `internal/handlers/tools/markingGenerationScope_test.go`
- `internal/handlers/tools/processPagesConcurrently.go`
- `internal/templates/marking/table_marking.html`
- `docs/audits/marking-job-generation-binding.md`

Aucun commit n'a été créé et aucune donnée réelle ou fixture privée n'a été touchée.
