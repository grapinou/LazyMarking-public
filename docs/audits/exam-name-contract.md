# Contrat du nom des évaluations

## Résultat

Avant ce jalon, les handlers Create et Edit transmettaient le nom reçu sans normalisation. Un nom blanc dépendait du `CHECK` SQLite, Create présentait toute erreur SQL comme un doublon ou un nom vide, et Edit présentait toute contrainte SQLite de cette manière.

Après ce jalon, chaque nouvelle écriture applicative applique `strings.TrimSpace` au nom avant `CreateExam` ou `UpdateExam`. Un nom vide après cette normalisation est refusé explicitement par le serveur, sans mutation SQL. Aucun autre caractère n'est retiré : les apostrophes et guillemets sont conservés exactement.

## Create

`AddExamHandler` récupère et normalise désormais le nom avant le parsing des quatre parents. Si le résultat est vide, il redirige en 303 vers `ErrorMessageURL` avec une erreur métier et n'appelle pas `CreateExam`. Les quatre parents restent obligatoires et parsés comme auparavant ; leur ownership reste vérifié par le SQL existant. Les brouillons à QCM vide ou classe vide restent autorisés.

Une violation SQLite structurée `SQLITE_CONSTRAINT_UNIQUE` produit une redirection métier 303 indiquant que l'évaluation existe déjà ou que la combinaison est utilisée. Toute autre erreur de `CreateExam` produit HTTP 500. La classification ne parse aucun message SQLite.

## Edit

`EditExamHandler` conserve d'abord la lecture ownership-aware de la cible et la vérification d'immutabilité d'un Exam généré. Le nom est ensuite normalisé et le blanc refusé avant le parsing des parents et avant `UpdateExam`. L'UPDATE atomique avec `NOT EXISTS`, la vérification après zéro ligne et le hook de test de la course génération/UPDATE restent inchangés.

Une violation SQLite structurée `SQLITE_CONSTRAINT_UNIQUE` produit la même erreur métier 303 que Create. Toute autre erreur de `UpdateExam` produit HTTP 500. Une cible ou un parent absent/étranger conserve le contrat 404. Un Exam généré conserve sa redirection métier 303 d'immutabilité.

## Unicité et défense en profondeur

La contrainte `UNIQUE(name, qcm_id, class_code_id, user_id)` n'a pas été modifiée. Année et période ne participent toujours pas à l'unicité. Après normalisation applicative, `Contrôle` et `  Contrôle  ` entrent donc naturellement en collision pour les mêmes QCM, classe et utilisateur.

Le `CHECK length(trim(name)) > 0` existant reste inchangé comme défense en profondeur. La validation serveur explicite traite le cas métier normal sans dépendre de ce CHECK.

## Données historiques et GET Edit

Aucune migration n'a été créée et aucun nom historique n'est normalisé automatiquement. Cela évite toute collision silencieuse entre des valeurs historiques telles que `Examen` et ` Examen `. Le GET Edit reste inchangé et présente donc le nom réellement stocké, espaces compris, jusqu'à une modification explicite par l'utilisateur.

## Classification SQLite

Le helper existant `IsSQLiteForeignKeyConstraint` n'a pas été détourné. Un helper ciblé `IsSQLiteUniqueConstraint` a été ajouté ; il utilise `errors.As` et `sqlite3.ErrConstraintUnique`. Il refuse notamment de classifier un CHECK ou une panne non SQLite comme doublon.

## Périmètre inchangé

L'immutabilité des Exams générés est inchangée et ses protections restent actives. Aucun code de génération, QCM, Marking, cleanup, preflight, worker ou individualisation n'a été modifié. Aucun template, JavaScript, view-data ou SQL n'a été modifié.

## Tests

Les tests ajoutés ou adaptés couvrent :

- TrimSpace en Create et Edit ;
- conservation exacte des apostrophes et guillemets après trim ;
- refus d'un nom uniquement blanc sans création ni modification ;
- doublon Create après trim avec une seule ligne finale ;
- doublon Edit après trim sans mutation ;
- erreur DB réelle Create en HTTP 500 ;
- erreur DB réelle Edit en HTTP 500 via le test existant ;
- classification ciblée UNIQUE, y compris erreur enveloppée, et rejet des CHECK/erreurs non SQLite.

Les tests existants d'immutabilité d'un Exam généré, de course génération/UPDATE, d'ownership et de parents étrangers restent passants.

## Validations

- `sqlc generate` : non exécuté, car aucun SQL n'a été modifié ;
- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `internal/handlers/exams/handlers.go` ;
- `internal/handlers/exams/handlers_test.go` ;
- `internal/handlers/tools/sqliteErrors.go` ;
- `internal/handlers/tools/sqliteErrors_test.go` ;
- `docs/audits/exam-name-contract.md`.
