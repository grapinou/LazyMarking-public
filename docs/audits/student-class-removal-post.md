# Retrait d’une relation élève/classe par confirmation GET et mutation POST

## 1. Défaut avant correction

La route `/dashboard/students-classcodes/delete` était enregistrée uniquement en `GET` et appelait directement `DeleteStudentClassCodeHandler`. Le lien « Retirer » de la liste pouvait donc supprimer immédiatement une relation `student_class_codes` par simple navigation.

Cette implémentation était incorrecte indépendamment de la protection `SameSite=Strict` de la session : un GET doit rester sans effet de bord et peut être activé involontairement, préchargé ou appelé directement dans une session valide.

## 2. Nouvelle séparation GET / POST

La même URL conserve son contrat de routage, mais les méthodes ont désormais des responsabilités distinctes :

- `GET` appelle `DeleteFormStudentClassCodeHandler`, valide le contexte puis rend une confirmation sans mutation ;
- `POST` appelle `DeleteStudentClassCodeHandler` et réalise seul le retrait ;
- les autres méthodes sont rejetées par le routeur avec `405 Method Not Allowed`.

Le POST lit les paramètres existants `student_id` et `class_code_id` dans le formulaire. La redirection après succès revient à la page des classes du même élève.

## 3. Données de vue de confirmation

Le type `StudentClassRelationDeleteData` a été ajouté au modèle typé `StudentClassCodePageData`. Il transporte :

- le contexte élève typé (`StudentClassContext`) ;
- la classe ciblée (`StudentClassOption`) ;
- `ActionURL` ;
- `ReturnURL` ;
- `CanDelete`.

Les IDs restent des `int64`, les URLs sont construites côté Go et aucune donnée SQLC, `ExtraData`, `map[string]any` ou reconstruction de route n’est exposée au template.

Le nouveau template `delete_form_student_class_code.html` affiche l’élève et la classe, précise que ni l’élève ni la classe ne seront supprimés, et propose « Retirer de la classe » puis « Annuler ».

## 4. Parcours utilisateur

Le parcours est maintenant :

1. « Classes de l’élève » → « Retirer » ;
2. confirmation GET en lecture seule ;
3. POST explicite « Retirer de la classe » ;
4. retour à la page des classes du même élève après succès.

« Annuler » revient directement à cette même page sans mutation.

## 5. Protection de la dernière classe

Les trois niveaux de protection sont cohérents :

- la liste n’affiche pas de lien actif lorsque `AllowedDelete` vaut faux ;
- un GET direct vers la confirmation détecte la dernière relation, affiche « Impossible de retirer la dernière classe de l’élève. » et ne rend aucun formulaire destructif ;
- un POST direct reste contrôlé par le handler et par `DeleteStudentClassCodeByStudentID`, dont la condition atomique exige qu’une autre relation existe.

La requête conditionnelle SQL existante demeure l’autorité contre une course entre suppressions concurrentes. Sa classification zéro ligne reste inchangée : dernière classe encore présente → erreur métier ; relation disparue ou contexte invalide → 404 selon les contrôles ownership-aware.

## 6. Ownership et appels forgés

Le GET et le POST vérifient tous deux :

- l’élève par `(student_id, user_id)` ;
- la classe par `(class_code_id, user_id)` ;
- la présence de la relation dans les classes de cet élève et de cet utilisateur.

Un élève absent ou étranger, une classe absente ou étrangère, ou une relation inexistante produit un `404`. Un paramètre absent ou non numérique produit un `400`. Aucun de ces chemins ne modifie la base.

## 7. Fichiers modifiés

- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/handlers_test.go`
- `internal/handlers/studentClassCode/routes.go`
- `internal/handlers/studentClassCode/viewData.go`
- `internal/handlers/studentClassCode/viewData_test.go`
- `internal/handlers/studentClassCode/views.go`
- `internal/templates/data/studentClassCode.go`
- `internal/templates/studentClassCodes/delete_form_student_class_code.html`
- `docs/audits/student-class-removal-post.md`

Les modifications préexistantes sans rapport dans le working tree n’ont pas été touchées.

## 8. Tests

La couverture ajoutée ou adaptée vérifie :

- rendu de la confirmation GET avec élève, classe, action POST, IDs et retour corrects ;
- conservation de toutes les relations après le GET ;
- retrait POST d’une seule relation lorsque l’élève en possède deux ;
- maintien de l’autre relation et redirection vers le bon élève ;
- dernière classe : GET informatif sans formulaire actif et POST forgé refusé ;
- élève/classe absents ou étrangers et relation inexistante sur GET et POST ;
- paramètres absents ou invalides en `400` ;
- méthode non déclarée en `405` ;
- rendu typé du nouveau template et garde-fou contre `ExtraData`.

Résultats :

- `go test ./internal/handlers/studentClassCode ./internal/db` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès.

## 9. Invariants préservés

Aucun changement n’a été apporté au schéma, aux migrations, au SQL, aux fichiers générés SQLC, aux routes nominales, à l’ownership ni à la règle « un élève doit toujours appartenir à au moins une classe ».

La mutation atomique `DeleteStudentClassCodeByStudentID` n’a pas été modifiée. Les autres constats de l’audit de clôture Élèves / Classes restent hors périmètre.

## 10. Résultat

Le bloquant est **résolu** : un GET vers le parcours de retrait ne modifie plus jamais la base, et la suppression effective exige désormais une confirmation suivie d’un POST tout en conservant les protections serveur existantes.
