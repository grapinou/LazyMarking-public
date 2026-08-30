# Protection serveur de la dernière classe d’un élève

## 1. Invariant métier

L’invariant attendu est désormais garanti côté serveur :

```text
un élève doit toujours appartenir à au moins une classe
```

Une suppression réussie d’une association élève/classe doit donc laisser au moins une autre association pour le même élève et le même utilisateur.

## 2. Comportement avant correction

La page des classes d’un élève calculait `AllowedDelete` à partir du nombre de relations et masquait les liens de suppression lorsque l’élève n’avait plus qu’une classe.

`DeleteStudentClassCodeHandler` se limitait toutefois à parser `student_id` et `class_code_id`, puis appelait directement `DeleteStudentClassCodeByStudentID`. Cette requête vérifiait l’ownership de l’élève, de la classe et de la relation, mais pas l’existence d’une autre classe.

La règle n’était donc qu’une protection d’interface. Un appel direct à la route de suppression avec les bons paramètres pouvait supprimer la dernière relation et laisser un élève sans classe.

## 3. Contournement d’AllowedDelete

`AllowedDelete` demeure utile pour ne pas proposer une action impossible dans l’interface, mais une valeur de vue ne constitue pas une frontière de sécurité. Le handler peut être appelé indépendamment du template, notamment par une requête forgée ou un ancien client.

La correction ne dépend plus de `AllowedDelete`. Celui-ci reste inchangé et cohérent avec la nouvelle autorité serveur.

## 4. Solution retenue

La correction associe une classification explicite dans le handler à une garantie atomique dans la requête de mutation.

Avant la mutation, `DeleteStudentClassCodeHandler` :

1. vérifie que l’élève existe pour `user_id` avec `GetStudentByID` ;
2. vérifie que la classe existe pour `user_id` avec `GetClassCodeNameByID` ;
3. lit les classes de l’élève avec `GetAllClassCodesByStudentID` ;
4. vérifie que la relation ciblée figure bien dans cette collection ;
5. refuse immédiatement l’opération lorsqu’il ne reste qu’une relation.

La requête SQLC `DeleteStudentClassCodeByStudentID` conserve toutes ses conditions d’ownership et ajoute une condition `EXISTS` exigeant une autre relation du même élève et du même utilisateur, avec un `class_code_id` différent de la cible.

Ainsi, le handler produit une erreur métier claire dans le cas normal, tandis que le SQL reste l’autorité finale contre les appels directs et les courses concurrentes.

Si le `DELETE` affecte zéro ligne après les vérifications initiales, le handler relit les relations :

- si la relation ciblée existe encore, elle est devenue la dernière entre-temps et l’erreur métier est retournée ;
- si elle n’existe plus, le cas est classé comme ressource/relation absente et retourne 404 ;
- une erreur de cette lecture de classification retourne 500.

## 5. Atomicité et concurrence

Un simple enchaînement `SELECT/COUNT`, puis `DELETE` aurait laissé une fenêtre de course : deux requêtes auraient pu observer deux classes et supprimer chacune une relation différente.

La condition suivante est évaluée dans le même statement que le `DELETE` :

```sql
EXISTS (
    SELECT 1
    FROM student_class_codes AS remaining_relation
    WHERE remaining_relation.student_id = student_id
      AND remaining_relation.user_id = user_id
      AND remaining_relation.class_code_id <> class_code_id
)
```

SQLite sérialise les écritures et chaque statement réévalue cette condition sur l’état disponible pour sa mutation. Si deux suppressions concurrentes partent d’un élève ayant deux classes, la première peut supprimer une relation ; la seconde ne trouve alors plus d’autre relation et affecte zéro ligne. Il est donc impossible que les deux suppressions aboutissent et laissent zéro classe.

Aucune transaction applicative supplémentaire n’est nécessaire : l’atomicité utile est portée par un unique statement SQL.

## 6. Erreurs et statuts HTTP

- paramètres manquants ou non numériques : comportement existant 400 conservé ;
- élève absent ou étranger : 404 ;
- classe absente ou étrangère : 404 ;
- relation inexistante ou ne correspondant pas au contexte : 404 ;
- dernière relation de l’élève : 303 vers `ErrorMessageURL`, avec le message métier indiquant que la dernière classe ne peut pas être retirée ;
- erreur DB inattendue : 500 ;
- suppression autorisée : 303 vers la liste des classes de l’élève, comme auparavant.

Aucune absence ou erreur d’ownership n’est transformée silencieusement en succès.

## 7. Fichiers modifiés

- `db/query/student_class_codes.sql`
- `internal/db/student_class_codes.sql.go` — régénéré par SQLC, non modifié manuellement ;
- `internal/db/studentRelationshipIntegrity_test.go`
- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/handlers_test.go`
- `docs/audits/student-last-class-protection.md`

Aucun template, aucune route, aucun schéma et aucune migration n’ont été modifiés.

## 8. Tests

Les tests ajoutés ou adaptés couvrent :

- élève avec une seule classe : réponse métier 303, relation conservée et total toujours égal à un ;
- élève avec deux classes : suppression autorisée d’une seule relation et conservation de l’autre ;
- élève absent ;
- élève étranger ;
- classe absente ;
- classe étrangère ;
- relation inexistante ;
- garde SQLC : mutation à zéro ligne lorsqu’une relation est la dernière ;
- conservation des contrôles ownership de la requête.

Validations exécutées :

- `sqlc generate -f db/sqlc.yaml` : réussi ;
- `go test ./internal/handlers/studentClassCode ./internal/db` : réussi ;
- `go test ./...` : réussi ;
- `gofmt` sur les fichiers Go modifiés : appliqué ;
- `git diff --check` : réussi.

## 9. Invariants préservés

La correction conserve :

- les routes et paramètres `student_id` / `class_code_id` ;
- la méthode HTTP existante de la route ;
- les vérifications ownership ;
- l’ajout d’une association élève/classe ;
- l’appartenance possible à plusieurs classes ;
- les autres suppressions Élèves et Classes ;
- le schéma et les migrations ;
- les données de vue typées et l’UX existante.

La seule évolution métier est le refus serveur de supprimer la dernière relation, conformément à l’invariant annoncé.
