# Classification des erreurs de suppression des référentiels

## Classification avant et après

Avant ce jalon, les handlers de suppression Matières, Thèmes, Niveaux, Compétences, Difficultés et Points redirigeaient toute erreur SQL vers le message indiquant que la valeur était utilisée par une question. Une erreur d'infrastructure ou une autre erreur SQLite était donc présentée à tort comme une violation de relation.

Désormais :

- une suppression possédée et libre redirige en HTTP 303 vers la liste du référentiel ;
- zéro ligne affectée (valeur absente ou étrangère) reste traité par `HandleOwnedMutationRows` en HTTP 404 ;
- une violation structurée de la FK `questions.<reference_id>` redirige en HTTP 303 vers `ErrorMessageURL` ;
- toute autre erreur SQL produit HTTP 500.

Aucun texte d'erreur SQLite n'est analysé.

## Helper commun SQLite

`internal/handlers/tools/sqliteErrors.go` introduit :

```go
func IsSQLiteForeignKeyConstraint(err error) bool
```

Le helper utilise `errors.As` et `sqlite3.Error.ExtendedCode`. Il reconnaît `sqlite3.ErrConstraintForeignKey`. Les tests sur le schéma réel ont également établi qu'une FK déclarée explicitement `ON DELETE RESTRICT` est rapportée par SQLite/mattn avec `sqlite3.ErrConstraintTrigger` (1811), et non 787. Ce code est donc reconnu pour les suppressions protégées du schéma LazyMarking. Une erreur SQL non-FK déterministe reste classée séparément et produit HTTP 500.

La classification locale du handler QCM a été remplacée par ce helper commun. Aucun autre comportement QCM n'a changé et les tests QCM existants restent inchangés et passants.

## Handlers adaptés

Les branches `err != nil` ont été adaptées dans :

- `DeleteSubjectHandler` ;
- `DeleteThemeHandler` ;
- `DeleteYearLevelHandler` ;
- `DeleteSkillHandler` ;
- `DeleteDifficultyHandler` ;
- `DeletePointHandler`.

Le parsing, l'authentification, les paramètres ownership-aware, les requêtes `Delete*`, le contrôle du nombre de lignes et les redirections de succès sont inchangés.

## Contrat DB et conservation des questions

Une matrice DB couvre les six FK `ON DELETE RESTRICT` :

- une valeur libre est supprimée ;
- une valeur étrangère affecte zéro ligne ;
- une valeur utilisée produit une contrainte SQLite ;
- après le refus, la question, son identifiant de référence et la valeur du référentiel sont tous conservés.

Aucune suppression ne cascade vers `questions`.

## Tests handlers

Une matrice transversale exécute les six handlers avec une base SQLite contrôlée. Pour chaque référentiel, elle couvre :

- succès et bonne destination HTTP 303 ;
- valeur absente en HTTP 404 ;
- valeur étrangère en HTTP 404 et conservation de sa ligne ;
- valeur utilisée en HTTP 303 vers `ErrorMessageURL`, sans mutation ;
- erreur DB non-FK en HTTP 500, sans redirection métier et sans mutation.

L'erreur non-FK est provoquée de façon déterministe par un trigger de test qui appelle une fonction SQLite inexistante. Elle n'est pas confondue avec la contrainte de relation.

Les tests unitaires du helper couvrent aussi une erreur FK directe, une erreur enveloppée, le code réellement émis par `RESTRICT`, une autre contrainte SQLite, une erreur non-SQLite et `nil`.

## SQL et migrations

Aucune requête SQL/sqlc n'a été ajoutée ou modifiée. Aucune requête de comptage préalable n'est utilisée : la FK reste l'autorité atomique. Aucune migration n'a été créée.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `internal/handlers/tools/sqliteErrors.go` ;
- `internal/handlers/tools/sqliteErrors_test.go` ;
- `internal/handlers/tools/referenceDeleteHandlers_test.go` ;
- `internal/db/referenceDeleteIntegrity_test.go` ;
- `internal/handlers/subjects/handlers.go` ;
- `internal/handlers/themes/handlers.go` ;
- `internal/handlers/yearlevels/handlers.go` ;
- `internal/handlers/skills/handlers.go` ;
- `internal/handlers/difficulties/handlers.go` ;
- `internal/handlers/points/handlers.go` ;
- `internal/handlers/qcm/handlers.go` ;
- `docs/audits/reference-delete-error-handling.md`.
