# Classification des erreurs de suppression des parents d'Exam

## Résumé

Les suppressions de Classes, Années et Périodes distinguent désormais une violation de clé étrangère d'une autre erreur SQLite. Aucun schéma, aucune requête SQL et aucun comportement Exam n'ont été modifiés.

## Domaines adaptés

### Classe

`DeleteClassCodeHandler` conserve l'authentification, le parsing, la suppression ownership-aware, le contrôle du nombre de lignes et la redirection vers la liste.

- classe libre : suppression puis HTTP 303 vers la liste des classes ;
- classe absente ou étrangère : zéro ligne, donc HTTP 404 ;
- contrainte FK : HTTP 303 vers `ErrorMessageURL` avec le message « Cette classe est utilisée par une évaluation ou contient encore des élèves et ne peut pas être supprimée. » ;
- autre erreur DB : HTTP 500.

Le message mentionne aussi les élèves car `student_class_codes.class_code_id` protège réellement une classe en plus de `exams.class_code_id`.

### Année

`DeleteYearHandler` applique le même contrat. Le message FK est : « Cette année est utilisée par une évaluation et ne peut pas être supprimée. »

### Période

`DeletePeriodHandler` applique le même contrat. Le message FK est : « Cette période est utilisée par une évaluation et ne peut pas être supprimée. »

## Classification SQLite

Les trois handlers réutilisent exclusivement `internal/handlers/tools.IsSQLiteForeignKeyConstraint`. Ce helper effectue une classification structurée avec `errors.As` et les codes étendus du driver SQLite. Aucun texte d'erreur n'est parsé et aucun second helper n'a été créé.

## Intégrité et ownership

La matrice de test utilise les FK réelles du domaine Exam :

- `exams.class_code_id -> class_codes.id ON DELETE RESTRICT` ;
- `exams.year_id -> years.id ON DELETE RESTRICT` ;
- `exams.period_id -> periods.id ON DELETE RESTRICT`.

Pour chacun des trois parents, elle prouve qu'une valeur libre est supprimable et qu'une valeur utilisée est refusée. Après le refus, l'Exam et le parent sont toujours présents : aucune cascade ni mutation partielle n'intervient.

Les DELETE existants restent filtrés par `id` et `user_id`. Une ressource étrangère affecte zéro ligne et retourne 404 sans révéler son existence.

## Erreur DB non-FK

Un trigger de test déterministe provoque une erreur SQLite non-FK lors du DELETE. Pour les trois handlers, la réponse est HTTP 500, sans redirection vers le message « utilisée », et aucune ligne n'est supprimée.

## Tests

Une fonction de test table-driven a été ajoutée avec trois domaines et cinq scénarios par domaine :

- succès ;
- absent ;
- étranger ;
- utilisé par un Exam ;
- erreur DB non-FK.

Elle vérifie également les destinations des redirections, les messages fonctionnels, la conservation des parents et la conservation de l'Exam.

## Validation

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

Aucun `sqlc generate` n'était nécessaire : aucun SQL n'a changé.

## Fichiers modifiés

- `internal/handlers/classCodes/handlers.go` ;
- `internal/handlers/years/handlers.go` ;
- `internal/handlers/periods/handlers.go` ;
- `internal/handlers/tools/examParentDeleteHandlers_test.go` ;
- `docs/audits/exam-parent-delete-error-handling.md`.

Aucune migration n'a été créée. Les handlers Exams, QCM, génération et Marking n'ont pas été modifiés.
