# Protection de l'historique lors de la suppression d'une évaluation

## 1. Contrat avant / après

Avant, supprimer une ligne `exams` supprimait automatiquement sa ligne `exams_generated`, puis toutes les copies et tous leurs contenus par cascades successives.

Après ce jalon :

- une évaluation sans aucune génération reste supprimable;
- l'existence d'une génération `running`, `success` ou `failed` protège l'évaluation;
- une génération reste explicitement supprimable par les workflows de cleanup existants, avec cascade sur sa propre descendance.

## 2. Migration créée

`db/migrations/0035_protect_generated_exam_history.sql` reconstruit uniquement `exams_generated`. Toutes les colonnes, valeurs, IDs, defaults, CHECK, FK utilisateur et `UNIQUE(exam_id, user_id)` sont préservés.

Le Down reconstruit la même table avec le contrat historique et préserve également les données.

## 3. FK avant / après

```text
avant : exams_generated.exam_id -> exams.id ON DELETE CASCADE
après : exams_generated.exam_id -> exams.id ON DELETE RESTRICT
```

Les cascades `exams_generated -> student_exam -> student_exam_content / student_exam_page_content` sont inchangées.

## 4. Triggers recréés

La reconstruction supprime les triggers attachés à l'ancienne table. La migration recrée donc, en Up comme en Down, sans modifier leur logique :

- `generated_exams_owner_insert`;
- `generated_exams_owner_update`.

Les tests vérifient leur présence et leur efficacité après les deux directions.

## 5. Exam sans génération

Le DELETE DB réussit. Le handler effectue le lookup ownership-aware, constate l'absence de génération, supprime exactement une ligne puis redirige en 303 vers la liste Exams.

## 6. Exam avec génération

Le DELETE DB direct échoue par FK. Le handler évite normalement cette tentative grâce à `GetExamByID` puis `ExamHasGeneration`, et redirige vers l'erreur métier pour les trois statuts possibles.

## 7. Chaîne historique préservée

Le test de migration construit une chaîne complète :

```text
Exam
  -> exams_generated
     -> student_exam
        -> student_exam_content
        -> student_exam_page_content
```

Après une tentative de suppression refusée, les cinq lignes, leurs IDs et leurs contenus sont toujours présents. L'échec du statement ne laisse aucun état partiel.

## 8. Cleanup explicite d'une génération

La suppression directe de `exams_generated` fonctionne toujours et cascade vers la copie, son snapshot et ses pages, sans supprimer l'Exam. Les tests existants de récupération des générations `running` continuent également à passer.

## 9. Prévalidation du handler

`DeleteExamHandler` charge d'abord l'Exam avec `GetExamByID(id,user_id)`. Une cible absente ou étrangère reste donc un 404. La nouvelle requête `ExamHasGeneration(exam_id,user_id)` est globale à la génération concernée mais toujours ownership-aware.

Si elle retourne vrai, aucun `DeleteExam` n'est appelé.

## 10. Gestion de la race FK

Une petite fonction interne injectable uniquement pour synchroniser le test insère déterministement une génération après le précheck et avant le DELETE. La FK RESTRICT bloque alors le DELETE; `tools.IsSQLiteForeignKeyConstraint` traduit cette branche vers le même message métier. L'Exam et la génération restent présents.

La prévalidation améliore donc la réponse utilisateur, tandis que la FK reste l'autorité finale.

## 11. Erreur DB non-FK

Une erreur SQLite non liée à une FK est provoquée déterministement après le précheck. Elle reste une réponse HTTP 500 et n'est pas présentée comme une évaluation générée.

## 12. Message métier

> Cette évaluation a déjà été générée et ne peut plus être supprimée.

La réponse est un 303 vers `ErrorMessageURL`.

## 13. Édition et individualisation

- règles d'édition Exam modifiées : **non**;
- individualisation modifiée : **non**;
- Marking modifié : **non**.

## 14. Tests

Ajouts/adaptations :

- migration Up : données, IDs, FK RESTRICT, triggers, Exam libre, chaîne historique atomiquement protégée, cascade interne;
- migration Down : données, FK CASCADE restaurée, triggers et ancienne cascade;
- handler : protection pour `running`, `success` et `failed`;
- handler : course entre précheck et DELETE;
- handler : erreur DB non-FK;
- fixtures handler adaptées au schéma protecteur;
- tests existants absence, ownership, succès et recovery conservés.

## 15. Validations

- `sqlc generate -f db/sqlc.yaml` : réussi;
- `go test ./...` : réussi;
- `go vet ./...` : réussi;
- `git diff --check` : réussi.

## 16. Fichiers modifiés

- `db/migrations/0035_protect_generated_exam_history.sql`;
- `db/query/exams.sql`;
- `internal/db/exams.sql.go` (généré par sqlc);
- `internal/db/examHistoryProtectionMigration_test.go`;
- `internal/handlers/exams/handlers.go`;
- `internal/handlers/exams/handlers_test.go`;
- `docs/audits/exam-delete-history-protection.md`.
