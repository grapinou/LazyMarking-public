# Rétention durable des jobs Marking success

Date : 2026-08-31

## Purge success avant/après

Avant ce jalon, `ListExpiredMarkingJobs` sélectionnait les jobs `success` et `failed` dont `completed_at` était antérieur à sept jours. `PurgeExpiredMarkingJobs` supprimait d'abord `assets/tmp/<username>/marking-<job_id>`, puis appelait le DELETE générique du job. La FK 0037 en CASCADE aurait donc supprimé toute la hiérarchie de résultats avec un success expiré.

La purge est appelée au démarrage du serveur et avant chaque nouvel upload Marking. Tous les usages ont été inspectés avant modification.

Après ce jalon, `ListExpiredFailedMarkingJobs` sélectionne exclusivement `status = 'failed'`. Aucun success n'est candidat, quel que soit son âge, sa génération nullable legacy ou la présence de résultats 0037.

## Failed retention

La durée reste exactement sept jours. La comparaison demeure stricte : `completed_at < cutoff`. Un failed récent et un failed exactement à la limite sont conservés; un failed plus ancien est purgeable.

Pour chaque failed expiré :

1. le workspace validé est supprimé;
2. `DeleteFailedMarkingJob` supprime la ligne avec `id`, `user_id` et `status='failed'`;
3. le nombre de lignes affectées doit être exactement un.

L'absence préalable du workspace est acceptée par `RemoveOperationTempDir` et ne bloque pas le DELETE. Une seconde purge est idempotente car la ligne n'est plus sélectionnée.

Si le cleanup filesystem échoue, la fonction retourne avant le DELETE. Le test utilise un workspace symlink non sûr et confirme que le job reste en DB, donc qu'une trace récupérable est conservée.

## GET failed non destructif

`ProgressMarkingHandler` ne fait plus de `DeleteMarkingJob` lorsqu'il lit un statut failed. Il conserve le message et la redirection métier existants. Deux GET successifs renvoient le même comportement et la ligne failed reste présente.

Le navigateur ne déclenche aucun cleanup filesystem. Le pipeline conserve son cleanup d'échec existant; un résidu de crash/recovery est traité plus tard par la purge failed.

## Primitive DELETE limitée par statut

`DeleteFailedMarkingJob` est une requête dédiée et ownership-aware :

- ID correct;
- user propriétaire;
- statut encore `failed` au moment du DELETE.

Elle rend atomiquement impossible la suppression d'un success même si l'état observé lors de la sélection ne correspond plus à l'état au DELETE. Les tests vérifient qu'elle affecte zéro ligne pour un success, un mauvais user, un ID déjà supprimé, et une ligne pour le failed possédé.

Le statut success est terminal dans les mutations de production actuelles. Le DELETE générique reste disponible pour une future suppression explicite/contrôlée et pour ses tests, mais aucun workflow automatique de production ne l'appelle désormais.

## Success legacy et résultats 0037

Le test central couvre simultanément :

- un success âgé de 100 jours lié à une génération success et à un `student_exam`;
- un `marking_copy_result`, un `marking_question_result` et une `marking_answer_detection` descendants;
- un workspace success avec `corrected.pdf`;
- un success legacy ancien avec `exam_generated_id NULL` et son workspace.

Après deux purges, jobs, résultats, workspaces et `corrected.pdf` existent toujours. Un success récent est également conservé. Aucun backfill legacy n'est tenté.

## Workspace success et PDF

La purge ne voyant plus aucun success, elle ne supprime jamais son workspace. `corrected.pdf`, `mark-table.pdf` et un éventuel `corrected_NOT.pdf` suivent tous cette conservation globale; aucune distinction d'artefact n'est introduite ici.

Aucun nom, contenu, mécanisme de génération/serve, `LeftPages` ou design PDF n'est modifié.

## Recovery

`RecoverRunningMarkingJobs` est inchangé : il sélectionne seulement les jobs running, supprime leur workspace, puis les passe à failed avec `completed_at`. Ils suivent ensuite la rétention failed normale et ne sont pas supprimés immédiatement.

Le test de recovery contient désormais un success avec workspace et `corrected.pdf`; deux passages de recovery conservent son statut et son fichier.

La dette P2 reste inchangée : une erreur remontée par `RecoverRunningMarkingJobs` provoque encore un `log.Fatal` au startup. Elle n'est pas traitée dans ce jalon.

## Pipeline de persistance

La persistance runtime reste non branchée. Aucun appel production à `PersistCorrectedMarkingCopy` ou aux créations de résultats SQLC n'est ajouté. Les structures runtime détaillées, OpenCV, seuil et règles de score sont inchangés dans ce jalon.

## Tests

Tests ajoutés/adaptés :

- success vieux de 100 jours conservé;
- success récent conservé;
- success legacy NULL et workspace conservés;
- hiérarchie synthétique 0037 conservée;
- workspace success et `corrected.pdf` conservés;
- failed ancien avec workspace supprimé;
- failed ancien sans workspace supprimé;
- failed récent et failed à la limite conservés;
- running non concerné par la purge;
- erreur filesystem : ligne DB conservée;
- DELETE failed limité par ownership et statut;
- deux pollings GET failed stables et non destructifs;
- recovery running inchangé;
- recovery success et son workspace préservés.

Tous les tests utilisent des données synthétiques.

## SQLC et validation

- `sqlc generate -f db/sqlc.yaml` : succès;
- `./scripts/check.sh` : succès;
- `go test -race ./...` : succès;
- `git diff --check` : succès.

Les fichiers générés n'ont pas été modifiés manuellement.

## Fichiers modifiés

- `db/query/markingJobs.sql`;
- `internal/db/markingJobs.sql.go` (généré);
- `internal/db/pipelineMutationRows_test.go`;
- `internal/handlers/tools/purgeExpiredMarkingJobs.go`;
- `internal/handlers/tools/purgeExpiredMarkingJobs_test.go`;
- `internal/handlers/tools/recoverMarkingJobs_test.go`;
- `internal/handlers/marking/handlers.go`;
- `internal/handlers/marking/handlers_test.go`;
- `docs/audits/marking-success-retention.md`.

Aucune migration, table ou commit n'a été créé.
