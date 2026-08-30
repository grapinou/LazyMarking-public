# Classification des erreurs DB de création et d’édition Élèves / Classes

## 1. Défaut avant correction

Plusieurs mutations transformaient toute erreur DB en erreur de saisie : doublon, nom vide ou relation déjà existante. Une panne SQLite, un trigger ou une contrainte sans rapport pouvait donc être masqué derrière un message métier trompeur.

Le jalon distingue désormais les contraintes métier précisément attendues des erreurs techniques, sans modifier les contraintes ni les règles applicatives.

## 2. Handlers concernés

L’audit ciblé a identifié les mutations suivantes :

| Handler | Requête SQLC | Classification avant |
|---|---|---|
| `AddStudentHandler` | `CreateStudentAndReturnID` | toute erreur = doublon ou nom vide |
| `EditStudentHandler` | `UpdateStudent` | toute erreur = doublon ou nom vide |
| `AddClassCodeHandler` | `CreateClassCode` | toute erreur = doublon ou classe vide |
| `EditClassCodeHandler` | `UpdateClassCode` | toute erreur = doublon ou champ vide |
| `AddStudentClassCodeHandler` | `CreateStudentWithClassCode` | toute erreur = relation en doublon |

La seconde mutation de `AddStudentHandler`, `CreateStudentWithClassCode`, produisait déjà un HTTP 500 pour une erreur DB et utilisait le contrat zéro ligne pour une ressource absente ou étrangère. Elle n’avait donc pas le défaut de classification générique, mais son rollback a été vérifié.

Les suppressions n’ont pas été modifiées.

## 3. Contraintes DB réellement observées

Le schéma final impose :

### `students`

- `CHECK(length(trim(first_name)) > 0)` ;
- `CHECK(length(trim(last_name)) > 0)` ;
- `UNIQUE(user_id, first_name, last_name)` ;
- FK `user_id`, qui n’est pas une erreur de saisie prénom/nom.

### `class_codes`

Après la migration de portée utilisateur :

- `CHECK(length(trim(name)) > 0)` ;
- `UNIQUE(name, user_id)` ;
- FK `user_id`.

### `student_class_codes`

- `UNIQUE(student_id, class_code_id, user_id)` ;
- FK vers élève, classe et utilisateur ;
- insertion conditionnelle ownership-aware qui retourne zéro ligne lorsque l’élève ou la classe n’appartient pas à l’utilisateur.

Des tests utilisant réellement `github.com/mattn/go-sqlite3` confirment les codes étendus :

- doublon : `SQLITE_CONSTRAINT_UNIQUE` ;
- valeur refusée par un `CHECK` : `SQLITE_CONSTRAINT_CHECK`.

Les tests de triggers produisent une autre catégorie et confirment qu’elle n’est pas confondue avec ces deux contrats.

## 4. Stratégie de classification

Le helper existant `tools.IsSQLiteUniqueConstraint` est réutilisé. Un helper symétrique et ciblé, `tools.IsSQLiteCheckConstraint`, a été ajouté.

Les deux helpers reposent sur `errors.As` et `sqlite3.Error.ExtendedCode`. Aucun message SQLite n’est analysé et le code générique `SQLITE_CONSTRAINT` ne suffit jamais à classifier une erreur métier.

La politique est :

- `UNIQUE` attendue par la mutation → redirection métier précise ;
- `CHECK` attendue par les mutations de nom → redirection métier précise ;
- zéro ligne sur les mutations ownership-aware → contrat 404 existant ;
- toute autre erreur → log contextualisé et HTTP 500.

## 5. Students Add / Edit

### Ajout

`CreateStudentAndReturnID` classe maintenant :

- `UNIQUE` → « Cet élève existe déjà. » ;
- `CHECK` → « Le prénom et le nom de l’élève doivent être renseignés. » ;
- autre erreur → log `AddStudentHandler -> CreateStudentAndReturnID DB error` et HTTP 500.

La transaction création élève + première relation reste inchangée. Une contrainte métier ou une erreur technique avant commit déclenche le rollback. Une erreur inattendue de `CreateStudentWithClassCode` après création de l’élève est également testée : aucune ligne partielle ne subsiste.

### Édition

`UpdateStudent` applique les mêmes deux catégories et retourne HTTP 500 pour les autres erreurs. Le contrôle `rows == 0` continue de produire le comportement ownership/absence existant.

## 6. Classes Add / Edit

`CreateClassCode` et `UpdateClassCode` classent désormais :

- `UNIQUE` → « Cette classe existe déjà. » ;
- `CHECK` → « Le nom de la classe doit être renseigné. » ;
- autre erreur → log avec le handler et la requête SQLC, puis HTTP 500.

L’édition absente ou étrangère conserve son contrat zéro ligne/404. La suppression et sa classification FK restent hors périmètre et inchangées.

## 7. Ajout relation élève/classe

`AddStudentClassCodeHandler` ne transforme plus toute erreur en doublon :

- `SQLITE_CONSTRAINT_UNIQUE` → « Cette classe est déjà associée à cet élève. » ;
- élève ou classe absent/étranger → insertion conditionnelle à zéro ligne puis 404, comme auparavant ;
- toute autre erreur → log `AddStudentClassCodeHandler -> CreateStudentWithClassCode DB error` et HTTP 500.

Les routes, paramètres `student_id`/`class_code_id` et règles ownership restent identiques.

## 8. Distinction métier / erreur technique

Les réponses métier utilisent la convention existante : HTTP 303 vers `ErrorMessageURL`, sans exposer SQL, code SQLite ni nom de contrainte.

Les erreurs techniques sont journalisées avec le handler, la requête SQLC et l’erreur originale, puis renvoyées sous forme de HTTP 500. Les tests injectent des triggers `RAISE(ABORT)` pour démontrer qu’une autre contrainte n’est ni un doublon ni un nom invalide.

## 9. Transactions et rollback

`AddStudentHandler` conserve son `BeginTx`, son `defer tx.Rollback()`, son `WithTx` et son commit final.

Les tests couvrent :

- doublon et `CHECK` avant création complète : aucune relation créée ;
- panne inattendue lors de la création de l’élève : aucune ligne créée ;
- panne inattendue lors de l’ajout de la première classe après insertion de l’élève : rollback de l’élève et absence de relation ;
- échec ownership de la première relation : test de rollback préexistant toujours passant.

## 10. Logging

Seules les erreurs inattendues sont journalisées comme pannes DB. Chaque log modifié cite le handler et la requête concernée :

- `CreateStudentAndReturnID` ;
- `UpdateStudent` ;
- `CreateClassCode` ;
- `UpdateClassCode` ;
- `CreateStudentWithClassCode`.

Les violations `UNIQUE`/`CHECK` attendues sont traitées comme refus métier sans exposer leur détail technique.

## 11. Fichiers modifiés

- `internal/handlers/tools/sqliteErrors.go`
- `internal/handlers/tools/sqliteErrors_test.go`
- `internal/handlers/students/handlers.go`
- `internal/handlers/students/handlers_test.go`
- `internal/handlers/classCodes/handlers.go`
- `internal/handlers/classCodes/handlers_test.go`
- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/handlers_test.go`
- `docs/audits/student-class-db-error-classification.md`

Les autres modifications préexistantes du working tree n’ont pas été touchées.

## 12. Tests et résultats

La couverture ajoutée vérifie :

- Students Add : `UNIQUE`, `CHECK`, erreur inattendue, cohérence et rollback de la première relation ;
- Students Edit : `UNIQUE`, `CHECK`, erreur inattendue, ligne originale conservée ;
- Classes Add : `UNIQUE`, `CHECK`, erreur inattendue ;
- Classes Edit : `UNIQUE`, `CHECK`, erreur inattendue, absence/ownership à zéro ligne ;
- relation élève/classe : doublon, élève/classe absent ou étranger, erreur inattendue ;
- helpers : erreurs construites, erreurs enveloppées et codes réellement renvoyés par le driver pour `UNIQUE` et `CHECK`.

Résultats :

- `go test ./internal/handlers/students ./internal/handlers/classCodes ./internal/handlers/studentClassCode ./internal/handlers/tools ./internal/db` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès ;
- `sqlc generate` non exécuté : aucune requête SQL n’a changé.

## 13. Invariants préservés

Aucun changement n’a été apporté aux migrations, au schéma, au SQL métier, aux fichiers SQLC générés, à `TrimSpace`, aux contraintes, à l’unicité, à l’ownership, aux routes, aux paramètres HTTP, aux templates, aux transactions ou aux suppressions.

La troncature CSV, le JavaScript historique des classes, les performances N+1 et les autres modules restent hors périmètre.

## 14. Statut du P2

**Résolu.** Toutes les mutations Add/Edit ciblées distinguent maintenant les contraintes métier précises des erreurs DB inattendues, lesquelles restent visibles comme HTTP 500 et ne sont plus masquées par un message de saisie.
