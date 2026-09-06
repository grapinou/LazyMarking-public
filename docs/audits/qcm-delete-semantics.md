# Sémantique de suppression d’un QCM

## Migration créée

La migration Goose `0033_cascade_qcm_composition_delete.sql` reconstruit `qcm_questions` afin de modifier uniquement la règle de suppression de sa FK vers `qcm`.

## FK qcm_questions.qcm_id avant et après

- Avant 0033 : `REFERENCES qcm(id)` sans cascade, soit `NO ACTION` dans SQLite.
- Après 0033 : `REFERENCES qcm(id) ON DELETE CASCADE`.

La composition appartient désormais au QCM : supprimer un QCM supprimable entraîne atomiquement la suppression de ses relations `qcm_questions`.

## FK question_id

`qcm_questions.question_id` reste `REFERENCES questions(id) ON DELETE RESTRICT`. La suppression d’un QCM ne supprime jamais une question de la banque.

## FK exams.qcm_id

La migration ne touche pas `exams`. Sa FK reste `REFERENCES qcm(id) ON DELETE RESTRICT`. Un examen protège donc toujours le QCM utilisé.

## Reconstruction de table

Les migrations Up et Down conservent :

- `id INTEGER PRIMARY KEY AUTOINCREMENT` ;
- `qcm_id`, `question_id`, `user_id` et `position` non nuls ;
- `CHECK(position >= 1)` ;
- `UNIQUE(qcm_id, question_id)` ;
- `UNIQUE(qcm_id, position)` ;
- la FK `user_id` vers `users` ;
- tous les IDs, parents et positions existants.

## Triggers ownership

Après chaque reconstruction, `qcm_questions_owner_insert` et `qcm_questions_owner_update` sont recréés avec la logique exacte de 0030/0032 : le QCM et la question doivent appartenir au `user_id` de la relation. Les tests vérifient leur présence et leur activité après Up et Down.

## Comportement Down

Le Down reconstruit le schéma 0032 avec `position` et ses deux contraintes uniques, mais restaure `qcm_id REFERENCES qcm(id)` sans cascade. Les données, IDs, positions et triggers sont conservés. Une suppression du QCM parent redevient refusée lorsqu’une relation existe.

## Suppression avec composition et conservation des questions

Le test métier supprime un QCM possédé contenant plusieurs relations. Le QCM et ses seules relations disparaissent ; toutes les questions de banque restent présentes. Une question partagée avec un second QCM et la relation de ce second QCM restent intactes.

Un QCM vide reste supprimable.

## QCM utilisé par un examen et atomicité

Un test exécute un vrai `DELETE` sur un QCM possédant simultanément une composition et un examen. La FK Exams refuse le statement. Le test vérifie ensuite que le QCM, toutes ses relations et l’examen sont encore présents : la cascade n’a laissé aucun état partiel.

## Stratégie du handler

Le handler appelle d’abord la requête ownership-aware `QCMHasExams` :

- parent absent ou étranger : `sql.ErrNoRows`, traduit en `404` ;
- QCM protégé : aucune tentative de `DELETE`, redirection `303` vers `ErrorMessageURL` avec le message « Ce QCM est utilisé par une évaluation et ne peut pas être supprimé. » ;
- QCM supprimable : `DeleteQCM` reste contraint par `id` et `user_id`, puis redirection `303` vers Mes QCM.

La prévalidation et le DELETE ne sont volontairement pas entourés d’une transaction applicative : la vérification fournit le message métier, tandis que la FK SQLite reste l’autorité atomique finale. Une référence Exam créée entre les deux opérations fera échouer le DELETE sans état partiel.

## Détection SQLite de la course résiduelle

Une erreur du DELETE est classée avec le type structuré `sqlite3.Error` et l’extended code `sqlite3.ErrConstraintForeignKey`. Aucun texte d’erreur n’est parsé. Cette branche produit la même redirection métier que la prévalidation ; toute autre erreur DB reste une erreur serveur.

## Ownership

`QCMHasExams` vérifie d’abord l’existence du QCM pour le couple `qcm_id/user_id`, puis ne compte que les examens du même utilisateur. `DeleteQCM` conserve son filtre `id/user_id` et son contrôle du nombre de lignes affectées. Les triggers protègent séparément l’ownership des relations de composition.

## Tests migration

Deux tests appliquent réellement le SQL 0033 à un schéma pré-0033 :

- Up : données, IDs, positions, contraintes, FKs, triggers, cascade, conservation des questions, QCM vide, isolation, question partagée et atomicité Cascade/Restrict ;
- Down : données, positions, contraintes, FKs et triggers conservés, contrat sans cascade restauré.

## Tests DB et handler

Trois fonctions de tests handler ont été ajoutées :

- succès avec cascade, QCM absent, QCM étranger et QCM protégé ;
- simulation déterministe d’une violation FK apparue après la prévalidation ;
- preuve qu’un QCM détecté comme protégé ne déclenche aucun appel à `DeleteQCM`.

Le fixture ownership existant a été adapté pour fournir la table `exams` requise par la nouvelle prévalidation.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès.
- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `db/migrations/0033_cascade_qcm_composition_delete.sql`
- `db/query/qcm.sql`
- `internal/db/qcm.sql.go`
- `internal/db/qcmDeleteCascadeMigration_test.go`
- `internal/handlers/qcm/handlers.go`
- `internal/handlers/qcm/handlers_test.go`
- `docs/audits/qcm-delete-semantics.md`

Les autres entrées visibles dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) préexistaient à ce jalon et n’ont pas été modifiées pour cette implémentation.
