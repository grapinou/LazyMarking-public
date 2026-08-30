# Suppression du N+1 de lecture des classes d’un élève

## 1. Comportement avant correction

`TableStudentClassCodesHandler` chargeait les classes associées en trois étapes :

1. `GetStudentByID` vérifiait l’existence et l’ownership de l’élève ;
2. `GetAllClassCodesByStudentID` retournait les IDs de ses relations ;
3. une boucle appelait `GetClassCodeNameByID` pour chaque ID.

Le view-model final était correct, mais la liste nécessitait `2 + N` requêtes SQL pour `N` classes. La requête d’IDs ne possédait aucun `ORDER BY` explicite.

## 2. Nombre et type de requêtes avant

Pour une page contenant `N` classes :

- 1 lecture ownership-aware de l’élève ;
- 1 lecture des `class_code_id` associés ;
- `N` lectures unitaires ownership-aware du nom de classe.

Total : **`2 + N` lectures**.

## 3. Nouvelle requête SQL

La requête SQLC ciblée `ListStudentClassCodesWithNames` a été ajoutée à `db/query/student_class_codes.sql`.

Elle retourne directement :

- `class_code_id` ;
- `class_code_name`.

Elle joint `students`, `student_class_codes` et `class_codes`, puis applique un ordre explicite `ORDER BY student_class_codes.id ASC`.

L’ancien chemin n’avait pas d’ordre garanti. L’ordre par ID de relation rend déterministe l’ordre d’association/insertion généralement observé jusque-là, sans tri alphabétique visible nouveau.

## 4. Ownership

La requête protège les trois niveaux :

- `students.id = student_id` et `students.user_id = user_id` ;
- la relation doit avoir le même `student_id` et le même `user_id` que l’élève ;
- la classe doit avoir le même ID que la relation et le même `user_id` que l’élève.

Le test DB injecte volontairement :

- une relation vers une classe étrangère ;
- une relation portant un `user_id` étranger ;
- un élève étranger possédant une relation incohérente ;
- un ID d’élève absent.

Aucune de ces données n’est exposée. Seules les classes possédées et correctement reliées sont retournées, dans l’ordre déterministe prévu.

## 5. Comportement du handler

Le handler conserve d’abord `GetStudentByID`, qui maintient le contrat existant : élève absent ou étranger → 404.

Il appelle ensuite une seule fois `ListStudentClassCodesWithNames`. Une erreur de cette lecture est journalisée avec le nom exact de la requête et produit HTTP 500 ; elle n’est pas transformée en état vide.

La boucle de lectures `GetClassCodeNameByID` a disparu. La seule boucle restante transforme en mémoire les lignes déjà complètes en `StudentClassListItem`.

Un garde-fou structurel isole le code source de `TableStudentClassCodesHandler` et vérifie :

- la présence de `ListStudentClassCodesWithNames` ;
- l’absence de `GetAllClassCodesByStudentID` ;
- l’absence de `GetClassCodeNameByID`.

## 6. Nombre et type de requêtes après

Pour une page contenant n’importe quel nombre de classes :

- 1 lecture ownership-aware de l’élève ;
- 1 lecture jointe ownership-aware de toutes ses classes avec IDs et noms.

Total : **2 lectures constantes**. Aucune requête supplémentaire n’est exécutée par classe.

## 7. Compatibilité du view-data

`buildStudentClassListPageData` reçoit les lignes SQLC uniquement dans la couche Go du handler/builder et les convertit immédiatement en view-model typé.

Les templates continuent à recevoir exclusivement :

- `StudentClassContext` ;
- `StudentClassListItem` ;
- `StudentClassListData`.

Chaque item conserve son ID, son nom et sa propre `DeleteURL`. `AddURL`, `AllowedDelete`, `NoClasses`, la protection de la dernière classe et l’état défensif sans classe restent identiques. Aucune struct SQLC, `ExtraData`, `any` ou reconstruction d’URL ne fuit dans le template.

Les templates n’ont pas été modifiés.

## 8. Tests

La couverture ajoutée ou adaptée vérifie :

- plusieurs classes, leurs IDs, leurs noms et leur ordre ;
- exclusion d’un élève étranger, d’une relation étrangère et d’une classe étrangère ;
- élève absent → aucun résultat SQL ;
- URLs de retrait attachées au bon item ;
- `AllowedDelete` avec plusieurs classes ;
- protection avec une seule classe ;
- état défensif avec zéro classe ;
- erreur DB de la nouvelle lecture → HTTP 500 ;
- absence structurelle du chemin N+1 dans le handler ;
- rendu des templates typés inchangé.

Résultats :

- `go test ./internal/db ./internal/handlers/studentClassCode` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès.

## 9. Fichiers modifiés

- `db/query/student_class_codes.sql`
- `internal/db/student_class_codes.sql.go` — généré par SQLC
- `internal/db/studentRelationshipIntegrity_test.go`
- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/handlers_test.go`
- `internal/handlers/studentClassCode/viewData.go`
- `internal/handlers/studentClassCode/viewData_test.go`
- `docs/audits/student-class-n-plus-one.md`

Les autres modifications préexistantes du working tree n’ont pas été touchées.

## 10. Fichiers SQLC régénérés

Commande exécutée :

```text
sqlc generate -f db/sqlc.yaml
```

Le seul fichier généré modifié est `internal/db/student_class_codes.sql.go`. Il contient exclusivement la constante SQL, les paramètres, la ligne résultat et la méthode générée de `ListStudentClassCodesWithNames`.

`GetAllClassCodesByStudentID` et `GetClassCodeNameByID` n’ont pas été supprimées : elles restent utilisées par les parcours de suppression, de confirmation, de génération ou par d’autres modules.

## 11. Invariants préservés

Aucun changement n’a été apporté au schéma, aux migrations, aux règles métier, aux contraintes, à l’ownership, aux routes, aux méthodes HTTP, aux templates, à la confirmation GET/POST, à la suppression atomique, à l’invariant de dernière classe, à l’import CSV, aux guillemets ou aux autres modules.

## 12. Statut du P3

**Résolu.** La page des classes d’un élève utilise désormais un nombre constant de lectures SQL, sans affaiblir l’ownership ni modifier son comportement fonctionnel ou son view-model.
