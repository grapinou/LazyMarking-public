# Refactor des données de vue Classes et Classes d’un élève

## 1. Objectif

Ce jalon remplace les données de vue dynamiques des modules de gestion des classes et des classes associées à un élève par des contrats Go explicites. Il prolonge le travail réalisé sur la section Élèves sans moderniser l’apparence des pages.

Le périmètre est strictement limité aux PageData, aux builders de vue, à leur alimentation par les handlers, aux adaptations mécaniques des templates et aux tests correspondants.

## 2. État avant modification

### Gestion des classes

`ClassCodePageData` exposait `ExtraData map[string]any`. La liste y plaçait :

- les `db.ClassCode` issus de SQLC ;
- un booléen `NoClassCode` ;
- une slice séparée de `ClassCodeActionURLs`.

Le template associait chaque classe à ses URLs Edit/Delete par son index dans les deux slices parallèles. Les formulaires Edit et Delete lisaient leur ID et leur nom depuis des clés dynamiques `ClassCodeID` et `ClassCode`.

Le formulaire Add n’utilisait aucune donnée dynamique, mais partageait le même PageData faible.

### Classes d’un élève

`StudentClassCodePageData` exposait lui aussi `ExtraData map[string]any`. La liste recevait séparément :

- l’élève sous forme de `db.Student` ;
- les classes sous forme de `config.ClassCode` ;
- une slice d’actions `StudentClassCodeActionURLs` alignée par index ;
- l’URL d’ajout ;
- le booléen global `AllowedDelete`.

Le formulaire d’ajout exposait directement des lignes SQLC `ListClassCodesNotAssignedToStudentRow` ainsi que l’ID élève sous forme de chaîne.

Cette organisation rendait les erreurs de clé ou d’index détectables uniquement à l’exécution et couplait directement les templates aux représentations DB/configuration.

## 3. Fichiers modifiés

### Données de vue

- `internal/templates/data/classCode.go`
- `internal/templates/data/studentClassCode.go`

### Builders et handlers

- `internal/handlers/classCodes/handlers.go`
- `internal/handlers/classCodes/viewData.go`
- `internal/handlers/classCodes/viewData_test.go`
- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/viewData.go`
- `internal/handlers/studentClassCode/viewData_test.go`

### Templates

- `internal/templates/classcodes/table_class_codes.html`
- `internal/templates/classcodes/edit_form_class_code.html`
- `internal/templates/classcodes/delete_form_class_code.html`
- `internal/templates/studentClassCodes/table_student_class_codes.html`
- `internal/templates/studentClassCodes/add_form_student_class_code.html`

`internal/templates/classcodes/add_form_class_code.html` a été audité et testé, mais n’a nécessité aucune adaptation : il utilisait déjà uniquement les routes typées et aucune clé `ExtraData`.

## 4. Structures typées créées

### Classes

- `ClassCodeContext` transporte l’ID `int64` et le nom d’une classe précise pour Edit/Delete.
- `ClassCodeListItem` regroupe l’ID, le nom, `EditURL` et `DeleteURL` d’une même classe.
- `ClassCodeListData` contient la slice typée des items et l’état `NoClasses`.
- `ClassCodePageData` expose désormais `List` et `ClassCode` en plus des routes et du titre.

### Classes d’un élève

- `StudentClassContext` transporte l’ID `int64`, le prénom et le nom de l’élève parent.
- `StudentClassListItem` regroupe l’ID et le nom d’une classe associée avec sa propre `DeleteURL`.
- `StudentClassListData` regroupe le contexte élève, les relations affichées, l’URL d’ajout, `AllowedDelete` et l’état `NoClasses`.
- `StudentClassFormData` regroupe le contexte élève et les classes disponibles sous forme de `[]StudentClassOption`.
- `StudentClassCodePageData` expose désormais `List` et `Form`.

Ces structures décrivent les besoins des interfaces, sans exposer directement de struct SQLC, de struct DB ou de struct de configuration aux templates.

## 5. Suppression d’ExtraData

`ExtraData` a été supprimé de `ClassCodePageData` et `StudentClassCodePageData`.

Aucun des six templates audités sous `internal/templates/classcodes/` et `internal/templates/studentClassCodes/` ne référence désormais `ExtraData`. Les deux fichiers PageData ne contiennent plus `map[string]any` ni `any`.

Les IDs restent des `int64` dans les données de vue. Leur conversion en chaîne est limitée à la construction des URLs côté Go.

## 6. Remplacement des slices parallèles

Les types `ClassCodeActionURLs` et `StudentClassCodeActionURLs` ont été supprimés.

Pour la liste des classes, chaque `ClassCodeListItem` porte directement son `EditURL` et son `DeleteURL`. Pour les relations élève/classe, chaque `StudentClassListItem` porte directement sa `DeleteURL`.

Les templates n’utilisent plus `index` pour faire correspondre une collection métier à une collection d’actions. Les builders construisent les items complets dans un seul parcours et pré-calculent les URLs avec les routes existantes et des paramètres échappés.

## 7. Représentation d’AllowedDelete

Le comportement existant a été conservé exactement : `AllowedDelete` est une décision globale de la page, vraie uniquement lorsque l’élève appartient à plus d’une classe.

Le builder `buildStudentClassListPageData` dérive donc :

```text
AllowedDelete = len(Items) > 1
```

Le template continue d’afficher un tiret à la place de chaque action de suppression lorsque ce booléen est faux. Lorsqu’il est vrai, chaque relation expose sa propre URL correctement associée.

Une incohérence préexistante a été observée pendant l’audit : cette protection de cardinalité est appliquée par la vue, mais `DeleteStudentClassCodeHandler` ne revérifie pas lui-même que l’élève conservera au moins une classe. Une requête forgée directement vers le handler pourrait donc contourner l’indisponibilité visuelle. Conformément au périmètre exclusivement view-data, ce point n’a pas été corrigé et devra faire l’objet d’un jalon métier ciblé.

## 8. Comportement métier préservé

Les handlers continuent d’effectuer les mêmes lectures ownership-aware et les mutations existantes n’ont pas été modifiées. Le formulaire d’ajout de relation récupère maintenant la valeur du `db.Student` déjà lue pour la vérification ownership afin de construire le contexte typé ; cela n’ajoute aucune requête.

Aucun changement n’a été apporté à :

- la création, la normalisation, l’unicité ou l’édition d’une classe ;
- la protection FK lors de la suppression d’une classe ;
- l’association élève/classe ;
- la suppression d’une association ;
- la représentation UI de l’obligation d’avoir au moins une classe ;
- l’appartenance à plusieurs classes ;
- l’ownership ;
- les routes, méthodes et noms de paramètres ;
- SQL, SQLC, schéma ou migrations.

La liste des classes d’un élève conserve ses lectures actuelles, y compris les recherches de nom par classe. Leur optimisation éventuelle nécessiterait une évolution SQL hors de ce jalon.

## 9. Tests et résultats

### Gestion des classes

Les tests couvrent :

- plusieurs classes avec ID et nom exacts ;
- une liste vide avec slice non nil et `NoClasses` ;
- les URLs Edit/Delete portées par le bon item ;
- le contexte sans classe précise du formulaire Add ;
- les contextes typés Edit et Delete ;
- le rendu des quatre templates ;
- la conservation de `class_code`, `class_code_id` et `new_class_code` ;
- l’absence d’`ExtraData`, de `any`, d’ancien type d’actions parallèle et d’association template par index.

### Classes d’un élève

Les tests couvrent :

- le contexte élève ID/prénom/nom ;
- plusieurs classes associées ;
- les URLs de suppression attachées au bon item ;
- l’URL d’ajout avec le bon élève ;
- `AllowedDelete=false` pour une seule classe ;
- `AllowedDelete=true` pour plusieurs classes ;
- le cas vide avec slice non nil ;
- plusieurs classes disponibles dans le formulaire d’ajout ;
- aucune classe disponible dans le builder ;
- le rendu des deux templates ;
- la non-exposition d’une suppression dans le rendu mono-classe ;
- la conservation de `student_id` et `class_code_id` ;
- l’absence d’`ExtraData`, de `any`, d’ancien type d’actions parallèle et d’association par index.

### Résultats

- `go test ./internal/handlers/classCodes ./internal/handlers/studentClassCode` : réussi ;
- `go test ./...` : réussi ;
- `git diff --check` : réussi ;
- `gofmt` appliqué à tous les fichiers Go modifiés ou créés.

## 10. Bénéfices architecturaux

Le refactor apporte :

- un contrat compilable entre handlers et templates ;
- la suppression des assertions dynamiques et risques de clés manquantes ;
- la suppression des erreurs d’alignement entre slices parallèles ;
- des URLs explicitement rattachées à leur item ;
- des templates indépendants des types DB/SQLC/config ;
- des builders purs testables sans serveur ni base ;
- une base stable pour les prochains jalons UX.

## 11. Suite logique

La suite recommandée est :

1. moderniser l’UX de la gestion des classes ;
2. moderniser ensuite l’UX de la gestion des classes d’un élève.

Ces redesigns ne sont pas réalisés dans le présent jalon. La vérification serveur de la cardinalité avant suppression de la dernière relation devra être traitée séparément comme correction métier, avant ou avec le second jalon selon la priorité retenue.
