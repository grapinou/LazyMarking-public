# Classification métier de la suppression d’un élève protégé par l’historique

## 1. Protection historique existante

La migration `0024_create_student_exam.sql` définit `student_exam.student_id` comme clé étrangère vers `students.id`, sans cascade de suppression. Cette FK empêche la disparition d’un élève dont l’identité est utilisée par une copie générée.

Cette protection est volontaire et reste inchangée : les lignes `student_exam` et leurs descendants historiques ne doivent jamais être altérés pour permettre une suppression d’élève.

## 2. Défaut avant correction

`DeleteStudentHandler` et la première phase de `DeleteAllStudentsHandler` transmettaient toute erreur de suppression comme une panne technique HTTP 500.

Une violation de la FK historique était donc correctement refusée par SQLite, mais présentée à l’utilisateur comme une erreur serveur générique, sans expliquer le motif métier du refus.

## 3. Suppression individuelle

La suppression continue d’exécuter la requête ownership-aware `DeleteStudent`. Si SQLite retourne la violation FK structurée correspondant à la référence `student_exam`, le handler redirige désormais vers la page d’erreur métier avec le message :

> Cet élève ne peut pas être supprimé car il est déjà associé à une évaluation générée.

L’élève et la ligne `student_exam` restent présents. Un élève sans historique est toujours supprimé normalement, puis le navigateur est redirigé vers la liste Élèves. Les ressources absentes ou étrangères conservent la classification existante par nombre de lignes affectées.

## 4. Suppression des élèves d’une classe

Le comportement existant reste en deux phases dans une transaction :

1. suppression des élèves qui n’appartiennent qu’à la classe ciblée ;
2. détachement de cette classe pour les élèves multi-classes.

Si la première phase rencontre un élève mono-classe référencé dans `student_exam`, SQLite refuse le `DELETE`. Le handler redirige alors vers l’erreur métier :

> Impossible de supprimer les élèves de cette classe car au moins un élève est déjà associé à une évaluation générée.

Aucune suppression partielle n’est introduite. Les élèves multi-classes continuent seulement d’être détachés dans le cas normal.

## 5. Atomicité et rollback

`DeleteAllStudentsHandler` conserve sa transaction existante et son `defer tx.Rollback()`.

Lorsqu’une FK historique bloque la suppression :

- l’instruction SQLite qui cible les élèves mono-classe échoue atomiquement ;
- le handler quitte avant la phase de détachement ;
- le rollback restaure toute éventuelle modification transactionnelle ;
- tous les élèves et toutes les relations de la classe restent présents ;
- `student_exam` reste intact.

Le test de régression inclut un élève protégé, un autre élève mono-classe supprimable et un élève multi-classes afin de vérifier l’absence de résultat partiel.

## 6. Classification de la FK connue

La classification repose sur `sqlite3.Error` via `errors.As`, sans analyse de texte de message SQLite.

Le helper local `isStudentExamForeignKeyConstraint` accepte uniquement le code étendu `SQLITE_CONSTRAINT_FOREIGNKEY`. Ce choix est volontairement plus étroit que le helper FK générique du projet :

- `student_exam.student_id` utilise l’action SQLite par défaut `NO ACTION` et produit ce code explicite ;
- `SQLITE_CONSTRAINT_TRIGGER` n’est pas accepté, afin qu’un trigger sans rapport ne soit pas présenté comme une protection d’historique ;
- dans le schéma final, `student_exam` est la seule FK entrante non-cascade vers `students` susceptible de refuser ces `DELETE FROM students`.

La classification est appliquée uniquement à `DeleteStudent` et à `DeleteStudentsOnlyInOneClass`, c’est-à-dire aux mutations qui suppriment réellement des élèves. Une erreur lors du détachement des relations multi-classes n’est pas reclassifiée.

## 7. Message utilisateur et logging

Les refus métier utilisent la convention existante : redirection HTTP 303 vers `ErrorMessageURL` avec un `errormessage` encodé.

L’erreur SQLite brute n’est pas exposée à l’utilisateur. Les refus attendus ne sont pas journalisés comme des pannes critiques. Les erreurs non reconnues restent journalisées avec le nom précis du handler et de la requête, puis produisent HTTP 500.

Les anciens libellés de log de la suppression par classe ont également été rendus fidèles à `DeleteAllStudentsHandler`, sans changement fonctionnel.

## 8. Erreurs DB inattendues

Une contrainte déclenchée artificiellement et sans rapport avec `student_exam` reste une erreur HTTP 500 lors de la suppression individuelle. Le test transactionnel existant confirme également qu’un échec inattendu pendant le détachement multi-classe reste un HTTP 500 avec rollback.

Les erreurs de début ou de commit de transaction restent des erreurs techniques HTTP 500.

## 9. Fichiers modifiés

- `internal/handlers/students/handlers.go`
- `internal/handlers/students/handlers_test.go`
- `docs/audits/student-delete-history-protection.md`

Les autres changements préexistants du working tree n’ont pas été modifiés.

## 10. Tests

Les tests ajoutés ou adaptés couvrent :

- suppression individuelle protégée : redirection métier, élève conservé et `student_exam` conservé ;
- suppression individuelle non protégée : suppression et redirection normales ;
- erreur de contrainte inattendue : HTTP 500 et élève conservé ;
- suppression par classe avec historique : rollback complet des élèves mono-classe, de l’élève multi-classes et de toutes leurs relations ;
- conservation de la ligne `student_exam` ;
- suppression par classe normale : élève mono-classe supprimé, élève multi-classes conservé et seulement détaché ;
- erreur inattendue de détachement : HTTP 500 et rollback, via le test préexistant adapté à la FK cascade réaliste.

Résultats :

- `go test ./internal/handlers/students ./internal/db` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès ;
- `sqlc generate` non exécuté : aucune requête SQL n’a changé.

## 11. Invariants préservés

Aucun changement n’a été apporté aux migrations, au schéma, à la FK `student_exam`, aux règles `ON DELETE`, au SQL, à SQLC, aux routes, aux méthodes HTTP, aux templates, à l’ownership ou aux règles de suppression.

L’historique généré reste protégé, la suppression par classe reste transactionnelle et les autres constats de l’audit Élèves / Classes restent hors périmètre.

## 12. Statut du P2

**Résolu.** La protection historique continue d’empêcher toute suppression destructive, mais ce refus connu est désormais présenté comme une erreur métier claire ; les autres erreurs DB restent des HTTP 500.
