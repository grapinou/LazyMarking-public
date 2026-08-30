# Gestion de l’erreur DB de la liste Élèves

## 1. Défaut initial

`TableStudentsHandler` appelait `GetStudentsWithClasses`, capturait son erreur, mais la branche `err != nil` était vide. Le handler poursuivait ensuite son exécution avec des lignes absentes ou incomplètes.

## 2. Comportement avant correction

Une panne de la première lecture DB pouvait atteindre la récupération des classes puis le renderer. Selon le résultat du second appel, elle pouvait ainsi être présentée comme une liste Élèves vide ou partielle au lieu d’une panne serveur déterministe.

Cette situation brouillait la distinction entre l’état métier « aucun élève » et une indisponibilité technique de la base.

## 3. Correction appliquée

La branche d’erreur de `GetStudentsWithClasses` :

- journalise maintenant l’erreur avec le contexte `TableStudentsHandler -> GetStudentsWithClasses DB error` ;
- répond avec HTTP `500 Internal Server Error` ;
- interrompt immédiatement le handler avec `return` ;
- empêche donc tout appel à `ListClassCodesByUser`, à la construction du view-model et au renderer.

Le chemin sans erreur reste inchangé : filtre `class_filter`, regroupement multi-classe, représentation d’un élève sans classe, états vides et rendu typé conservent leur fonctionnement existant.

## 4. Comportement HTTP attendu

- succès des lectures : rendu normal de la liste Élèves ;
- erreur `GetStudentsWithClasses` : HTTP 500, sans redirection, 404 ni état vide métier ;
- l’erreur DB brute est journalisée côté serveur mais n’est pas exposée dans la réponse utilisateur.

## 5. Logging

Le log de la première lecture nomme explicitement `GetStudentsWithClasses`.

La seconde lecture du même handler, `ListClassCodesByUser`, possédait déjà une gestion correcte avec log, HTTP 500 et retour immédiat. Son message de log mentionnait toutefois par erreur `GetStudentsWithClasses` ; son contexte a été corrigé en `ListClassCodesByUser` sans modifier son comportement.

## 6. Autres lectures du même handler vérifiées

`TableStudentsHandler` ne réalise que deux accès DB :

1. `GetStudentsWithClasses` ;
2. `ListClassCodesByUser` pour alimenter le filtre.

Après correction, les deux erreurs interrompent le handler et produisent un HTTP 500. Aucune autre erreur DB silencieuse n’a été trouvée dans ce handler. Aucun autre handler du module n’a été audité ou modifié dans ce jalon.

## 7. Fichiers modifiés

- `internal/handlers/students/handlers.go`
- `internal/handlers/students/handlers_test.go`
- `docs/audits/students-list-db-error-handling.md`

Les autres modifications préexistantes du working tree n’ont pas été touchées.

## 8. Test ajouté

`TestTableStudentsReturnsInternalServerErrorWhenStudentQueryFails` ferme volontairement la connexion SQLite de test avant l’appel au handler. Cette panne déterministe fait échouer `GetStudentsWithClasses`.

Le test vérifie :

- une réponse HTTP 500 ;
- l’absence de rendu HTML normal de la page Élèves ;
- l’absence de l’état métier « Aucun élève ».

La couverture existante des builders et templates continue de protéger le chemin normal, notamment le regroupement multi-classe et les états de liste.

## 9. Résultats des vérifications

- `go test ./internal/handlers/students` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès ;
- SQLC : non exécuté, aucun SQL ni fichier SQLC n’ayant changé.

## 10. Invariants préservés

Aucun changement n’a été apporté au filtre, au regroupement des élèves, aux relations multi-classes, au traitement du `LEFT JOIN`, aux données de vue, aux routes, aux templates, au SQL, à SQLC, aux migrations, au schéma ou à l’ownership.

Les autres constats de l’audit Élèves / Classes restent hors périmètre.

## 11. Statut du constat P2

**Résolu.** Une panne de `GetStudentsWithClasses` ne peut plus être confondue avec une liste vide ou partielle : elle est journalisée et renvoyée immédiatement sous forme de HTTP 500.
