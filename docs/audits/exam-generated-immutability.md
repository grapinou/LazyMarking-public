# Immutabilité d'une évaluation générée

## 1. Contrat avant / après

Avant, `UpdateExam` permettait de modifier tous les champs d'une évaluation même lorsqu'une ligne `exams_generated` existait.

Après ce jalon :

- sans génération, l'évaluation reste entièrement modifiable;
- avec une génération `running`, `success` ou `failed`, aucune modification métier n'est autorisée;
- après suppression explicite de la génération par un workflow existant, l'évaluation redevient naturellement modifiable.

## 2. Champs figés

Les cinq champs métier modifiables sont protégés ensemble :

- `name`;
- `qcm_id`;
- `class_code_id`;
- `period_id`;
- `year_id`.

Leur modification reste atomique : aucun sous-ensemble ne peut changer sur un Exam généré.

## 3. Autorité finale

La garantie finale est portée directement par la requête SQL `UpdateExam` :

```sql
AND NOT EXISTS (
    SELECT 1
    FROM exams_generated eg
    WHERE eg.exam_id = exams.id
      AND eg.user_id = :user_id
)
```

Cette condition fait partie du même statement que l'UPDATE. SQLite sérialise les écritures : si la génération est créée avant l'UPDATE, celui-ci affecte zéro ligne; si l'UPDATE gagne la course, la génération ultérieure voit déjà le nouvel état, qui devient son contexte historique. Il n'existe donc aucune fenêtre où une génération créée pourrait être suivie d'une mutation de l'Exam.

Le handler effectue en plus une prévalidation `GetExamByID` puis `ExamHasGeneration` pour fournir une réponse métier explicite. Si l'UPDATE retourne zéro ligne après cette prévalidation, il revérifie la génération afin de distinguer la course d'un parent ou d'une cible invalide.

## 4. Migration

Aucune migration n'est nécessaire. La protection est un invariant de mutation atomique et ne modifie ni schéma, ni données, ni trigger. `sqlc generate` a régénéré la requête Go.

## 5. GET Edit généré

`EditFormExamHandler` conserve le lookup ownership-aware : absent ou étranger retourne 404. Pour un Exam possédé avec génération, le formulaire n'est pas rendu et le handler redirige en 303 vers `ErrorMessageURL` avec :

> Cette évaluation a déjà été générée et ne peut plus être modifiée.

Un Exam libre continue à rendre le formulaire normalement.

## 6. POST Edit généré

`EditExamHandler` valide d'abord la cible possédée puis l'absence de génération. Une génération déjà présente produit le même 303 sans appeler l'UPDATE. Si elle apparaît après le précheck, l'UPDATE atomique affecte zéro ligne et la revérification produit le même message métier.

Une erreur DB non liée à une contrainte métier retourne désormais 500. Les contraintes SQLite de nom vide/unicité conservent la redirection métier existante, avec un wording corrigé pour parler d'évaluation.

## 7. Exam libre

Un Exam sans génération peut toujours changer simultanément son nom, son QCM, sa classe, sa période et son année. Les contrôles ownership-aware des nouveaux parents et de la cible restent dans la même requête SQL.

## 8. Course génération / UPDATE

Un hook interne minimal de synchronisation de test crée déterministement une génération après le précheck et avant l'UPDATE. Le test vérifie :

- UPDATE refusé;
- cinq champs inchangés;
- génération présente;
- réponse 303 métier.

La condition SQL, et non le hook ou le précheck, assure la protection en production.

## 9. Génération failed et cleanup

Une génération `failed` présente interdit l'édition comme les deux autres statuts. Après sa suppression explicite, le test confirme que l'édition redevient possible. Aucun nouveau workflow de cleanup n'a été ajouté.

## 10. Ownership

Les conditions `exams.id`, `exams.user_id` et les vérifications ownership des quatre parents sont intactes. `ExamHasGeneration` reste filtré par `exam_id` et `user_id`. Les tests existants absent, étranger et parents étrangers continuent à passer.

## 11. Périmètre inchangé

- protection de suppression du jalon précédent : inchangée;
- création Exam : inchangée;
- tables de snapshots : inchangées;
- individualisation : inchangée;
- génération : inchangée;
- Marking : inchangé.

## 12. Tests

Ajouts/adaptations :

- DB table-driven : chacun des cinq champs est immuable avec une génération;
- DB : édition complète autorisée après cleanup;
- GET libre : formulaire rendu;
- GET généré : 303 pour `running`, `success` et `failed`;
- POST généré : aucune mutation;
- génération failed puis cleanup : édition réautorisée;
- course déterministe génération/UPDATE : refus atomique;
- erreur DB réelle : 500;
- test de succès existant renforcé pour modifier les cinq champs;
- tests absence, étranger et ownership existants conservés.

## 13. Validations

- `sqlc generate -f db/sqlc.yaml` : réussi;
- `go test ./...` : réussi;
- `go vet ./...` : réussi;
- `git diff --check` : réussi.

## 14. Fichiers modifiés

- `db/query/exams.sql`;
- `internal/db/exams.sql.go` (généré par sqlc);
- `internal/db/examRelationshipIntegrity_test.go`;
- `internal/handlers/exams/handlers.go`;
- `internal/handlers/exams/handlers_test.go`;
- `docs/audits/exam-generated-immutability.md`.

Aucune migration, aucun template et aucun fichier de génération/Marking/individualisation n'a été modifié.
