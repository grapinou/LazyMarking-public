# Correction du contrat Points

## Résumé

Le contrat final est `point_value INTEGER NOT NULL CHECK (point_value >= 1)`. Il ne comporte aucune limite haute métier : les valeurs supérieures à 100 restent acceptées par les handlers et la base, même si l'interface actuelle propose principalement les valeurs 1 à 100.

## Migration

La migration `0034_require_positive_point_values.sql` reconstruit la table `points` avec le `CHECK (point_value >= 1)` en conservant les colonnes, les IDs, les valeurs, les propriétaires, l'unicité `(point_value, user_id)` et la clé étrangère vers `users`.

Avant de désactiver temporairement les clés étrangères et avant toute reconstruction, la migration copie les valeurs existantes dans une table temporaire portant une contrainte nommée. Une valeur historique inférieure ou égale à zéro provoque donc un échec explicite `migration_0034_point_value_must_be_positive`. Aucune valeur pédagogique n'est corrigée silencieusement et l'ancienne table reste intacte.

La migration Down restaure le contrat précédent sans `CHECK`. Les tests vérifient également ce chemin.

## Préservation des questions

La relation `questions.point_id` n'est pas modifiée. Un test de migration réel conserve une ligne `points` d'ID 12, sa valeur 101 et une question référençant toujours l'ID 12. `PRAGMA foreign_key_check` reste sans erreur après Up et Down.

## Validation des handlers

`AddPointHandler` et `EditPointHandler` rejettent désormais explicitement les entiers inférieurs à 1 avec HTTP 400 avant toute mutation. Les erreurs de parsing, l'ownership, l'unicité et les autres comportements existants restent inchangés. Aucune limite haute n'a été ajoutée : 101 est accepté par le handler et 1000 est accepté par la base dans les tests.

## Formulaire Edit

Un `PointFormData` ciblé fournit au template l'ID, la valeur courante et les options. La valeur stockée est marquée `selected`. Si une valeur historique valide dépasse 100, elle est ajoutée aux options afin qu'une ouverture puis une soumission sans changement la conserve.

Une édition sans changement d'un point valant 5 conserve la valeur 5. Le caractère partagé du référentiel Points n'a pas été modifié.

## Tests

Tests de migration ajoutés :

- application réelle de Up sur l'ancien schéma ;
- préservation des IDs, valeurs et références des questions ;
- acceptation de 1 et des valeurs supérieures à 100 ;
- rejet de 0 et des valeurs négatives ;
- échec explicite sur une donnée historique invalide avant reconstruction ;
- application réelle de Down et restauration de l'ancien contrat.

Tests handlers ajoutés :

- Add accepte 1, 100 et 101 ;
- Add rejette 0 et -1 ;
- le GET Edit fournit la valeur courante 5, ainsi qu'une valeur courante 101 hors plage UI ;
- le POST Edit sans changement conserve 5 ;
- Edit rejette 0 et -1.

## Périmètre

Aucun comportement des matières, thèmes, niveaux, compétences ou difficultés n'a été modifié. Aucun comportement de suppression ni aucune apparence générale des pages Points n'a été refondu.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès, aucun fichier généré modifié ;
- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `db/migrations/0034_require_positive_point_values.sql` ;
- `internal/db/pointsContractMigration_test.go` ;
- `internal/handlers/points/handlers.go` ;
- `internal/handlers/points/handlers_test.go` ;
- `internal/templates/data/point.go` ;
- `internal/templates/points/edit_form_point.html` ;
- `docs/audits/points-contract-fix.md`.
