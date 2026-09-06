# Implémentation du déplacement des questions d’un QCM

## Routes ajoutées

- `POST /dashboard/qcm/qcmquestion/move-up`
- `POST /dashboard/qcm/qcmquestion/move-down`

Ces routes sont enregistrées dans le périmètre `qcmQuestions` et protégées par l’authentification existante. Aucun `GET` ne réalise de mutation.

## Handlers ajoutés

- `MoveQCMQuestionUpHandler`
- `MoveQCMQuestionDownHandler`

Les deux handlers valident la méthode et les paramètres `qcm_id` et `qcm_question_id`, puis délèguent à la même primitive transactionnelle. Après un déplacement réussi ou un no-op de borne, ils redirigent en `303` vers la composition du QCM.

## Primitive transactionnelle

La primitive privée `moveQCMQuestion` reçoit une direction typée (`moveQCMQuestionUp` ou `moveQCMQuestionDown`). Elle ouvre une transaction, charge la relation avec le triplet `user_id`, `qcm_id`, `qcm_question_id`, détermine la position adjacente, effectue l’échange, puis commit.

Toute erreur avant le commit déclenche le rollback différé. Les erreurs `sql.ErrNoRows` issues du lookup ownership-aware restent traduites en `404`; les autres erreurs restent des `500`.

## Requêtes SQL

Deux requêtes sqlc minimales ont été ajoutées :

- `GetQCMQuestionByPosition`, qui retrouve la voisine dans le même QCM possédé ;
- `MoveQCMQuestionToPosition`, mutation contrainte par `id`, `qcm_id` et `user_id`, avec vérification du QCM possédé.

Les fichiers Go sqlc ont été régénérés, sans modification manuelle.

## Stratégie d’échange

La contrainte `UNIQUE(qcm_id, position)` reste active. L’échange utilise trois mutations :

1. la relation courante passe temporairement à `MAX(position) + 1` ;
2. la voisine prend l’ancienne position de la relation courante ;
3. la relation courante prend l’ancienne position de la voisine.

La position temporaire est positive et libre puisque les positions sont compactes `1..n`. Les trois étapes appartiennent à la même transaction.

## Bornes

- Monter la première question est un no-op valide, suivi d’une redirection `303`.
- Descendre la dernière question est un no-op valide, suivi d’une redirection `303`.

Aucune écriture n’est effectuée dans ces deux cas.

## Ownership

Une relation absente, étrangère ou associée à un autre `qcm_id` n’est jamais déplacée et produit un `404`. Le lookup initial exige simultanément `user_id`, `qcm_id` et `qcm_question_id`. La recherche de la voisine et chacune des mutations restent limitées au même QCM et au même utilisateur.

## Rollback

Un test installe un trigger SQLite qui force un échec lors de la deuxième étape de l’échange. Il vérifie que les positions initiales sont intégralement restaurées, qu’aucune position temporaire ne subsiste et qu’aucune ligne n’est perdue.

## Tests DB et handlers

Cinq fonctions de test, contenant quinze sous-scénarios, ont été ajoutées :

- montée et descente au milieu de quatre questions ;
- conservation des identifiants de relation et des `question_id` ;
- bornes haute et basse ;
- déplacements dans un QCM de deux éléments ;
- rollback au milieu de l’échange ;
- handlers montée, descente et borne avec redirection `303` ;
- relation absente, mauvais parent, QCM étranger et relation étrangère avec `404` et zéro mutation.

Les assertions de composition prouvent que les positions restent exactement `1..n` après chaque opération réussie.

## Périmètre inchangé

- Aucun template n’a été modifié.
- Preview n’a pas été modifié.
- La génération réelle et son mélange n’ont pas été modifiés.
- Aucun SQL de lecture/génération existant n’a changé.
- Aucune migration n’a été ajoutée.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès.
- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `db/query/qcmquestion.sql`
- `internal/db/qcmquestion.sql.go`
- `internal/handlers/qcmQuestions/handlers.go`
- `internal/handlers/qcmQuestions/handlers_test.go`
- `internal/handlers/qcmQuestions/routes.go`
- `internal/templates/data/qcmQuestions.go`
- `docs/audits/qcm-question-move-implementation.md`

Les autres entrées visibles dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) préexistaient à ce jalon et n’ont pas été modifiées pour cette implémentation.
