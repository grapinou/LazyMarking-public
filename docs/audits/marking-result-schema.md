# Schéma initial de persistance des résultats Marking

Date : 2026-08-31

## Périmètre

Ce jalon crée uniquement le schéma, l'intégrité DB, les requêtes SQLC et une primitive transactionnelle testable pour les résultats de copie. Il ne branche pas le pipeline OpenCV et ne modifie ni algorithme, seuil, PDF, UX, recovery, purge ou rétention.

## Tables créées

### `marking_copy_results`

Une ligne représente le résultat terminal d'un `student_exam` dans une tentative `marking_job`.

- `id INTEGER PRIMARY KEY AUTOINCREMENT`;
- `user_id INTEGER NOT NULL`;
- `marking_job_id INTEGER NOT NULL`;
- `student_exam_id INTEGER NOT NULL`;
- `outcome TEXT NOT NULL`;
- `expected_pages INTEGER NOT NULL`;
- `detected_pages INTEGER NOT NULL`;
- `score_half_units INTEGER NULL`;
- `total_points INTEGER NULL`;
- `failure_code TEXT NULL`;
- `failure_detail TEXT NULL`;
- `completed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP`.

`outcome` accepte seulement `corrected`, `incomplete`, `not_seen`, `error`. `expected_pages >= 1` et `detected_pages >= 0`. La borne `detected_pages <= expected_pages` n'est volontairement pas imposée : le contrat actuel peut observer des pages dupliquées avant de classer la copie non corrigée.

Une copie `corrected` exige score et total, `total_points >= 1`, `0 <= score_half_units <= 2 * total_points`, et aucune information de failure. Les trois autres outcomes exigent score et total NULL; ils ne reçoivent donc jamais une note artificielle. Leur failure reste nullable tant que la taxonomie n'est pas définie.

`completed_at` est toujours non NULL : aucune ligne de construction n'est représentée.

### `marking_question_results`

- `id INTEGER PRIMARY KEY AUTOINCREMENT`;
- `copy_result_id INTEGER NOT NULL`;
- `question_index INTEGER NOT NULL`, zéro-based et correspondant à `qcm.Questions[question_index]`;
- `state TEXT NOT NULL` parmi `incorrect`, `partial`, `correct`;
- `score_half_units INTEGER NOT NULL`;
- `total_points INTEGER NOT NULL`.

Les CHECK imposent `question_index >= 0`, `total_points >= 1`, puis exactement :

- incorrect : `score_half_units = 0`;
- partial : `score_half_units = total_points`;
- correct : `score_half_units = 2 * total_points`.

Un trigger interdit également d'ajouter ou de déplacer une question sous un résultat copie non `corrected`.

### `marking_answer_detections`

- `id INTEGER PRIMARY KEY AUTOINCREMENT`;
- `question_result_id INTEGER NOT NULL`;
- `answer_index INTEGER NOT NULL`, zéro-based et correspondant à `qcm.Questions[question_index].Answers[answer_index]`;
- `detected_state INTEGER NOT NULL`, limité à 0/1;
- `mean_gray REAL NOT NULL`, borné entre 0 et 255 inclus.

L'état attendu n'est pas dupliqué. Il reste dans `student_exam_content`, indexé par question/réponse. Aucune FK ne cible les tables vivantes questions, réponses, alternatives ou points.

## Unicités

- `UNIQUE(marking_job_id, student_exam_id)` : une copie a au plus un résultat par tentative;
- aucune unicité sur `student_exam_id` seul : plusieurs jobs peuvent corriger la même copie;
- `UNIQUE(copy_result_id, question_index)`;
- `UNIQUE(question_result_id, answer_index)`.

## FK et suppressions

- `marking_copy_results.user_id -> users.id ON DELETE RESTRICT`;
- `marking_copy_results.marking_job_id -> marking_jobs.id ON DELETE CASCADE`;
- `marking_copy_results.student_exam_id -> student_exam.id ON DELETE RESTRICT`;
- questions vers résultat copie en CASCADE;
- détections vers résultat question en CASCADE.

Le RESTRICT sur `student_exam` empêche un résultat orphelin. Une génération reste déjà protégée par la FK RESTRICT du `marking_job`; le test confirme qu'elle ne peut pas disparaître avec le job/résultat présent.

La suppression explicite d'un job supprime toute sa hiérarchie de résultats. Avec la purge applicative actuelle, ce CASCADE supprimerait aussi les résultats d'un job success. Cela est accepté uniquement parce que le pipeline ne crée encore aucune de ces lignes. Le branchement production devra être précédé ou accompagné du jalon rétention qui empêchera la purge automatique des jobs success durables.

## Ownership

`CreateMarkingCopyResult` est un `INSERT ... SELECT` qui joint `marking_jobs` et `student_exam` et impose simultanément :

- `copy_result.user_id = marking_job.user_id`;
- `student_exam.user_id = marking_job.user_id`;
- `student_exam.exam_generated_id = marking_job.exam_generated_id`.

Des triggers INSERT et UPDATE répètent cette preuve pour bloquer les écritures SQL directes et les réaffectations incohérentes. Les tests couvrent job Alice/copie Bob, job A/copie B du même Alice, user Alice/job Bob, INSERT direct et UPDATE direct.

`user_id` n'est pas répété dans `marking_question_results` ni `marking_answer_detections`. Leur chaîne parent est l'autorité; les lectures SQLC traversent `marking_copy_results` et filtrent son `user_id`. Cela évite des identités redondantes et de nouveaux triggers sans perdre le scope utilisateur.

## SQLC

Requêtes ajoutées :

- `CreateMarkingCopyResult`, ownership-aware;
- `CreateMarkingQuestionResult`;
- `CreateMarkingAnswerDetection`;
- `GetMarkingCopyResult`;
- `ListMarkingQuestionResults`;
- `ListMarkingAnswerDetections`.

Aucune API de statistiques ou de pipeline n'est ajoutée. `sqlc generate -f db/sqlc.yaml` réussit et les fichiers générés n'ont pas été modifiés manuellement.

## Atomicité synthétique

`PersistCorrectedMarkingCopy` reçoit une petite structure dédiée, sans logique OpenCV et distincte de `config.MarkExam`. Il vérifie que score/total copie correspondent aux sommes des questions, puis écrit dans une transaction courte :

1. le résultat copie terminal corrected;
2. toutes les questions;
3. toutes les détections.

La transaction commit seulement après le dernier enfant. Le test provoque un échec sur la dernière détection (`mean_gray = 300`) et vérifie zéro ligne dans les trois tables.

## Legacy et metadata job

La migration ne backfill aucun résultat. Elle fonctionne avec un `marking_job` legacy dont `exam_generated_id` est NULL et le conserve à l'identique après Up puis Down. Down supprime seulement triggers et trois nouvelles tables.

`result_schema_version`, `marking_algorithm_version` et `detection_threshold` ne sont pas ajoutés. Le meilleur moment est le branchement réel de la persistance, afin qu'un job portant ces métadonnées garantisse effectivement l'usage du nouveau pipeline et non l'ancien comportement.

## Pipeline production

Pipeline modifié : non. En particulier, aucun changement dans `ProcessMarking`, `MarkingStudentExam`, `GetAnswersState`, `CountingPoints`, `DrawMarking`, `TypstWriter`, `LeftPages`, `CompleteMarkingJob`, purge ou recovery.

## Tests synthétiques

Les tests couvrent :

- migration Up/Down et job legacy NULL;
- corrected avec deux questions, plusieurs réponses, indices, états, scores, totaux et moyennes relus exactement;
- correct, partial, incorrect valides et combinaisons state/score incohérentes refusées;
- incomplete, not_seen et error sans score, avec `completed_at`;
- score sur non-corrected refusé, failure sur corrected refusée;
- pages attendues/détectées et score copie hors bornes refusés;
- `detected_state` et `mean_gray` hors bornes refusés;
- ownership cross-user, cross-generation et user/job, via SQLC et SQL direct;
- unicités copie/question/réponse;
- même `student_exam` accepté dans deux jobs;
- rollback intégral de la hiérarchie;
- suppression `student_exam`/génération refusée;
- suppression explicite job avec CASCADE descendant.

Aucune donnée, copie ou base réelle n'est utilisée.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès;
- `./scripts/check.sh` : succès;
- `go test -race ./...` : succès;
- `git diff --check` : succès.

## Fichiers modifiés

- `db/migrations/0037_create_marking_results.sql`;
- `db/query/markingResults.sql`;
- `internal/db/markingResults.sql.go` (généré);
- `internal/db/models.go` (généré);
- `internal/db/markingResultsRepository.go`;
- `internal/db/markingResults_test.go`;
- `docs/audits/marking-result-schema.md`.

Aucun commit n'a été créé.
