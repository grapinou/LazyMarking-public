# Schéma Marking — ambiguïtés et revue humaine

## Migration

La migration Goose `0040_add_marking_review_model.sql` ajoute le premier contrat durable de revue sans activer de workflow de production. Elle ne modifie aucune migration historique et son `Down` retire uniquement les nouvelles tables, triggers et colonnes.

## Métadonnées `marking_jobs`

Trois colonnes sont ajoutées :

- `ambiguity_delta REAL NULL`, avec `CHECK (ambiguity_delta IS NULL OR ambiguity_delta >= 0)` ;
- `review_revision INTEGER NOT NULL DEFAULT 0`, avec valeur non négative ;
- `artifacts_revision INTEGER NOT NULL DEFAULT 0`, non négative et jamais supérieure à `review_revision`.

Les jobs existants comme les nouveaux jobs créés par le pipeline actuel gardent `ambiguity_delta = NULL`. Aucun delta par défaut n'est choisi et aucun backfill n'est effectué. Une transition future `NULL -> valeur` est autorisée ; une valeur renseignée devient ensuite immutable par trigger. Les révisions restent à zéro tant qu'aucun workflow de revue/régénération n'est branché.

La politique V1 future reste centralisée conceptuellement : `abs(mean_gray - detection_threshold) <= ambiguity_delta`, bornes inclusives. Aucun booléen ambigu n'est stocké.

## Détection automatique et état effectif

`marking_answer_detections.detected_state` et `mean_gray` ne sont pas modifiés. La décision automatique originale demeure l'historique immuable.

La lecture `GetEffectiveAnswerDetection` expose :

- l'état détecté original ;
- l'état revu nullable ;
- l'état effectif dérivé par `COALESCE(reviewed_state, detected_state)` ;
- les métadonnées de revue lorsqu'elles existent.

`effective_state` n'est pas persisté comme troisième état redondant. Aucun score de question ou de copie n'est recalculé dans ce jalon.

## Table `marking_answer_reviews`

Une ligne représente la décision humaine courante pour une détection :

- PK `id` ;
- `answer_detection_id NOT NULL UNIQUE`, FK vers `marking_answer_detections`, `ON DELETE CASCADE` ;
- `reviewer_user_id NOT NULL`, FK vers `users`, `ON DELETE RESTRICT` ;
- `reviewed_state NOT NULL`, limité à `0` ou `1` ;
- `reviewed_at NOT NULL`, valeur par défaut `CURRENT_TIMESTAMP` ;
- `revision NOT NULL DEFAULT 1`, avec `revision >= 1`.

Une revue est autorisée pour toute détection possédée, indépendamment de MeanGray et d'une éventuelle ambiguïté. L'ambiguïté ne servira qu'à constituer la file automatique `pending/completed`. Une ligne dont `reviewed_state == detected_state` représente explicitement une revue confirmée ; l'absence de ligne signifie « non revue ».

La V1 conserve une seule ligne courante, pas un journal append-only. `UNIQUE(answer_detection_id)` arbitre deux créations concurrentes. La mise à jour SQLC exige la révision attendue, incrémente exactement de un et retourne le nombre de lignes affectées : zéro signifie conflit, absence ou mauvais propriétaire. Un trigger interdit également les sauts de révision lors d'un accès SQL direct.

## Ownership des reviews

Les requêtes SQLC de création, lecture et mise à jour traversent :

`answer_detection -> question_result -> copy_result -> marking_job -> user`.

Pour la V1, `reviewer_user_id` doit être l'utilisateur propriétaire du job. Des triggers `INSERT` et `UPDATE` appliquent aussi cette règle aux écritures SQL directes. L'identité de la détection d'une review est immutable. Une review Alice sur une détection Bob, ou une mutation ultérieure du reviewer vers Bob, est refusée.

## Table `marking_aligned_pages`

La table prépare la conservation future de la page alignée non annotée utilisée pour mesurer les cases :

- PK `id` ;
- `user_id NOT NULL`, FK `users`, `ON DELETE RESTRICT` ;
- `copy_result_id NOT NULL`, FK `marking_copy_results`, `ON DELETE CASCADE` ;
- `page_exam NOT NULL`, entier supérieur ou égal à 1 ;
- `storage_key NOT NULL` ;
- `width` et `height` strictement positifs ;
- `sha256 NOT NULL`, hexadécimal lowercase de 64 caractères ;
- `created_at NOT NULL` ;
- `UNIQUE(copy_result_id, page_exam)`.

La clé est relative, sans racine, username, séparateur inverse, composant `.`/`..` ou double séparateur. Le contrat SQLC et DB emploie la forme déterministe `aligned/student-exam-<student_exam_id>/page-<page_exam>.png`.

Une insertion doit viser une copie `corrected`, appartenir au même utilisateur que la copie et le job, et correspondre à exactement une page historique du `student_exam`. Les triggers réappliquent ces invariants aux accès SQL directs. Les métadonnées deviennent immutables après insertion.

Aucun fichier, resolver ou workspace aligné n'est créé dans ce jalon. Le hash représentera ultérieurement les bytes du PNG, pas son chemin.

## SQLC minimal

Les seules primitives ajoutées concernent :

- création, modification optimiste et lecture ownership-aware d'une review ;
- lecture de l'état effectif ;
- création et lecture ownership-aware des métadonnées de page alignée.

Il n'existe encore ni file UX, ni requête statistique, ni recalcul de score. Les fichiers générés ont été produits par `sqlc generate -f db/sqlc.yaml` et non modifiés manuellement.

## Legacy et migration Down

Après `Up`, les anciens jobs ont `ambiguity_delta NULL`, `review_revision = 0` et `artifacts_revision = 0`. Aucune review et aucune page alignée artificielle ne sont créées. Le test `Down` vérifie la disparition des nouvelles tables et colonnes tout en conservant le job historique et le schéma antérieur.

## Tests

Les tests synthétiques couvrent :

- valeurs initiales legacy, contraintes de révision, `artifacts_revision <= review_revision` et immutabilité du delta ;
- état effectif sans review, overrides `0 -> 1` et `1 -> 0`, et confirmation `1 -> 1` ;
- review d'une détection manifestement non ambiguë (`mean_gray = 40`) ;
- préservation de `detected_state` ;
- ownership SQLC et SQL direct, y compris mutation du reviewer ;
- unicité d'une review, révision attendue et rejet d'une révision obsolète ;
- page alignée valide, lecture ownership-aware, cross-user, copie/page absentes, doublon, clé invalide, traversal, hash invalide, dimension nulle et immutabilité.

Toutes les données sont synthétiques. Aucun scan, aucune base réelle et aucun artefact nominatif ne sont utilisés.

## Périmètre inchangé

- filesystem aligné ajouté : non ;
- scores production modifiés : non ;
- lifecycle `running/success/failed`, recovery ou purge modifié : non ;
- PDF ou `DrawMarking` modifiés : non ;
- OpenCV, MeanGray ou seuil 150 modifiés : non ;
- delta produit choisi : non.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès ;
- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés pour ce jalon

- `db/migrations/0040_add_marking_review_model.sql` ;
- `db/query/markingReviews.sql` ;
- `internal/db/markingReviews.sql.go` (généré par sqlc) ;
- `internal/db/models.go` (généré par sqlc) ;
- `internal/db/markingReviews_test.go` ;
- `docs/audits/marking-review-schema.md`.

Les autres éléments déjà présents dans le worktree ne font pas partie de ce jalon.
