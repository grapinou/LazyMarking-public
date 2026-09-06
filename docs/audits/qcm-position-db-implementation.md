# Implémentation DB de la position des questions d'un QCM

## 1. Migration créée

La migration Goose `db/migrations/0032_add_qcm_question_position.sql` ajoute l'ordre pédagogique sans modifier les migrations historiques.

Elle possède des sections Up et Down complètes et conserve les IDs relationnels ainsi que toutes les colonnes historiques.

## 2. Reconstruction de la table

SQLite ne permet pas d'ajouter proprement toutes les contraintes voulues sur une colonne backfillée. La migration crée donc une table temporaire avec le contrat final, copie les données, supprime l'ancienne table puis renomme la nouvelle.

Le schéma final conserve :

- `id INTEGER PRIMARY KEY AUTOINCREMENT` ;
- `qcm_id` obligatoire, FK vers `qcm(id)` sans cascade ;
- `question_id` obligatoire, FK vers `questions(id) ON DELETE RESTRICT` ;
- `user_id` obligatoire, FK vers `users(id)` ;
- `UNIQUE(qcm_id, question_id)`.

La migration Down reconstruit symétriquement le schéma sans `position`.

## 3. Backfill

Les positions existantes sont attribuées avec :

```sql
ROW_NUMBER() OVER (PARTITION BY qcm_id ORDER BY id)
```

Chaque QCM reçoit indépendamment les positions `1..n`. Les `id`, `qcm_id`, `question_id` et `user_id` sont copiés explicitement. Le test de migration utilise volontairement des `question_id` qui ne suivent pas les IDs relationnels afin de prouver que le backfill dépend bien de `qcm_questions.id`.

## 4. Contraintes finales

La colonne suit le contrat :

```sql
position INTEGER NOT NULL CHECK(position >= 1)
```

La table ajoute :

```sql
UNIQUE(qcm_id, position)
```

La position 1 reste autorisée dans deux QCM différents. Une position inférieure à 1, une position dupliquée dans le même QCM et un doublon QCM/question sont refusés.

## 5. Triggers ownership

La reconstruction supprime les triggers attachés à l'ancienne table. Les migrations Up et Down recréent donc explicitement, avec la logique de `0030` inchangée :

- `qcm_questions_owner_insert` ;
- `qcm_questions_owner_update`.

Les tests appliquent réellement la migration à un ancien schéma possédant ces triggers, vérifient leur présence après reconstruction, puis démontrent que QCM étranger, question étrangère et updates forgés restent refusés.

## 6. Requêtes SQL modifiées

- `GetAllQuestionsByQCMID` retourne `position` et trie par position ascendante ;
- `GetQCMQuestionsIDs` trie désormais par position ascendante ;
- `CreateQCMQuestion` calcule et insère la position suivante ;
- `DeleteQCMQuestion` reste ownership-aware et est entouré par les nouvelles lectures/mutations de compactage ;
- `GetQCMQuestionPosition` lit la position et le maximum du QCM possédé ;
- `MoveQCMQuestionPositionsToTemporaryRange` réalise la première phase sûre ;
- `CompactQCMQuestionPositions` ramène les positions dans la séquence compacte.

Les fichiers Go sqlc et le modèle `QcmQuestion.Position` ont été régénérés, jamais modifiés manuellement.

## 7. `CreateQCMQuestion`

La position est calculée dans l'unique `INSERT SELECT` :

```sql
COALESCE((SELECT MAX(position) FROM qcm_questions WHERE qcm_id = :qcm_id), 0) + 1
```

La première question reçoit 1. Les contrôles d'existence/ownership du QCM et de la question restent dans le même statement ; les triggers et les deux contraintes uniques restent une défense supplémentaire.

## 8. Ajout multiple

`AddQCMQuestionHandler` convertit les IDs puis applique `slices.Sort` avant d'ouvrir/exécuter le lot transactionnel. L'ordre implicite des checkboxes ne devient donc pas un contrat métier.

Chaque insertion prend la prochaine position. Un lot `9,4,7` est ajouté comme `4,7,9`. Une erreur ou un élément étranger rollbacke toujours tout le lot.

## 9. Suppression et compactage

`DeleteQCMQuestionHandler` reçoit maintenant la connexion DB afin d'exécuter dans une seule transaction :

1. lookup ownership-aware de la relation, de sa position et du maximum ;
2. DELETE ownership-aware ;
3. déplacement des positions supérieures vers `position + ancienMax` ;
4. retour vers `position - ancienMax - 1` ;
5. vérification du nombre de lignes aux deux phases ;
6. commit.

La plage temporaire se situe strictement au-dessus de l'ancien maximum, respecte `CHECK(position >= 1)` et évite toute collision avec `UNIQUE(qcm_id, position)`. Une suppression en dernière position ne nécessite aucun UPDATE. Toute erreur rollbacke également le DELETE.

## 10. Handlers adaptés

- `AddQCMQuestionHandler` trie le lot, sans modifier sa transaction ni ses contrats POST ;
- `DeleteQCMQuestionHandler` effectue lookup, suppression et compactage dans une transaction ;
- la route POST de retrait utilise le wrapper existant `HandlerWithDBAndConn`.

Les comportements absent, étranger et mauvais parent restent des 404. Aucun formulaire ou template n'a changé.

## 11. Tests de migration

Deux tests ont été ajoutés :

- montée réelle depuis un état pré-0032 avec deux QCM, IDs préservés, backfill déterministe, contraintes, FKs et triggers ownership ;
- descente réelle vérifiant la disparition de `position` et la recréation du trigger ownership.

Il ne s'agit pas uniquement d'un test d'un schéma final reconstruit à la main : le SQL exact de la nouvelle migration est exécuté.

## 12. Tests DB et handlers

Cinq autres tests ont été ajoutés :

- ajout en positions successives et indépendance de deux QCM ;
- lectures de composition et d'IDs dans l'ordre des positions ;
- ajout handler d'un lot désordonné, résultat trié et positionné ;
- suppressions dernière, milieu et première avec compactage ;
- échec simulé du compactage, avec rollback du DELETE et des positions.

Le test handler existant des relations absentes/étrangères/mauvais parent a été adapté à la transaction avec connexion. Les fixtures existantes ont reçu la colonne et les contraintes finales. Le test existant de lot mixte continue de prouver le rollback complet.

Au total : 7 tests ajoutés et 1 test comportemental adapté, en plus des fixtures mises à niveau.

## 13. Pipeline concurrent

`GetQCMQuestionsAnswers`, `GetQCMQuestionsAnswersCtx`, `BuildQuestion`, `BuildQuestionCtx`, les workers, channels et résultats n'ont pas été modifiés.

## 14. Preview

Les handlers et routes `qcmPreview` sont inchangés. Le preview conserve temporairement son comportement actuel.

## 15. Génération réelle

`GetQCMQuestionsIDs` lit désormais les IDs dans l'ordre pédagogique, puis le `ShuffleSlice` existant continue immédiatement à les mélanger. La génération réelle reste donc mélangée. Exams, Marking, Typst et `student_exam_content` sont inchangés.

## 16. Génération sqlc

`sqlc generate -f db/sqlc.yaml` : réussi. Le diff généré est limité au modèle `QcmQuestion`, aux rows de composition et aux requêtes de position/compactage attendues.

## 17. Validations

- `go test ./...` : réussi ;
- `go vet ./...` : réussi ;
- `git diff --check` : réussi.

## 18. Fichiers modifiés pour ce jalon

- `db/migrations/0032_add_qcm_question_position.sql` ;
- `db/query/qcmquestion.sql` ;
- `internal/db/models.go` (généré) ;
- `internal/db/qcmquestion.sql.go` (généré) ;
- `internal/db/qcmPositionMigration_test.go` ;
- `internal/db/qcmPositionQueries_test.go` ;
- `internal/db/qcmRelationshipIntegrity_test.go` ;
- `internal/handlers/qcmQuestions/handlers.go` ;
- `internal/handlers/qcmQuestions/handlers_test.go` ;
- `internal/handlers/qcmQuestions/routes.go` ;
- `docs/audits/qcm-position-db-implementation.md`.

Aucun SQL Exams, template, preview, pipeline concurrent, Typst, router global ou QCMContext n'a été modifié. Aucun commit n'a été créé.
