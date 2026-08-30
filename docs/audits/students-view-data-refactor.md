# Refactor des données de vue de la section Élèves

Date : 30 août 2026

## 1. Objectif du jalon

Ce jalon avait pour objectif de supprimer les données de vue non typées de la section Élèves et de construire un contrat explicite entre les handlers et les six templates du CRUD/import.

Il prépare la future modernisation UX de cette section, sans modifier son apparence de manière volontaire et sans toucher aux règles métier, aux routes, aux requêtes SQLC, au SQL ou aux migrations.

## 2. État avant modification

`StudentPageData` exposait un champ :

```go
ExtraData map[string]any
```

Les six templates Élèves lisaient leurs données par des clés dynamiques :

- `table_students.html` : `Students`, `Action`, `ClassCodes`, `CurrentClassFilter`, `NoStudent`, `NoClassCode` ;
- `add_form_student.html` : `ClassCodes` ;
- `edit_form_student.html` : `StudentID`, `FirstName`, `LastName` ;
- `delete_form_student.html` : `Student`, `StudentID` ;
- `add_csv_form_student.html` : `ClassCodes` ;
- `delete_form_all_students.html` : `ClassCodeID`, `ClassCodeName`.

Dans la liste, les élèves et leurs actions étaient construits dans deux slices parallèles. Le template devait associer chaque élève à son entrée `Action` par son index. Le handler transformait directement les lignes SQLC en `config.Student` et `config.ClassCode`, puis exposait ces structures au template.

Cette organisation présentait plusieurs limites :

- une clé mal orthographiée ou une valeur de mauvais type n'était détectée qu'au rendu ;
- le compilateur ne pouvait pas vérifier le contrat handler/template ;
- les slices parallèles pouvaient désynchroniser un élève et ses URLs ;
- les templates dépendaient de structures conçues pour la configuration ou la persistance plutôt que pour l'interface ;
- l'ajout ou la suppression d'une donnée nécessitait une recherche manuelle des clés dynamiques ;
- les tests de données de vue étaient moins directs.

L'ancien regroupement conservait aussi dans une map des pointeurs vers des éléments d'une slice extensible. Une réallocation de cette slice pouvait rendre ces pointeurs obsolètes lorsque les lignes de plusieurs élèves et classes étaient intercalées. Le nouveau builder utilise des index stables.

## 3. Fichiers modifiés

### Données de vue

- `internal/templates/data/student.go`

### Handlers et builders

- `internal/handlers/students/handlers.go`
- `internal/handlers/students/viewData.go`

### Tests

- `internal/handlers/students/viewData_test.go`

### Templates adaptés

- `internal/templates/students/table_students.html`
- `internal/templates/students/add_form_student.html`
- `internal/templates/students/edit_form_student.html`
- `internal/templates/students/delete_form_student.html`
- `internal/templates/students/add_csv_form_student.html`
- `internal/templates/students/delete_form_all_students.html`

## 4. Structures de vue créées

### `StudentClassOption`

Représente une classe dans l'interface avec uniquement son `ID int64` et son `Name`. Elle est utilisée pour les classes affichées sur un élève, les filtres, les sélecteurs de formulaire et l'action de suppression par classe. Elle évite d'exposer le propriétaire ou d'autres détails DB inutiles au template.

### `StudentListItem`

Représente une ligne fonctionnelle de la liste Élèves. Elle transporte :

- l'ID, le prénom et le nom de l'élève ;
- sa collection typée de classes ;
- son URL d'édition ;
- son URL de suppression ;
- son URL de gestion des classes.

Les données et les actions d'un même élève sont ainsi regroupées dans un seul objet.

### `StudentListData`

Regroupe tout le contexte propre à la page de liste :

- les `StudentListItem` ;
- les classes disponibles pour le filtre et la suppression groupée ;
- le filtre de classe courant ;
- `NoStudents` ;
- `NoClasses`.

Ces deux booléens conservent explicitement les états que le template affichait auparavant avec `NoStudent` et `NoClassCode`.

### `StudentFormData`

Contient les classes disponibles pour les formulaires d'ajout manuel et d'import CSV. La même structure reflète leur besoin commun sans exposer `db.ClassCode`.

### `StudentContext`

Représente l'élève parent des pages Edit et Delete avec :

- `ID int64` ;
- `FirstName` ;
- `LastName`.

Il remplace les clés dispersées et la structure DB directement fournie au template.

### `StudentClassDeleteData`

Représente la classe ciblée par la confirmation de suppression de tous ses élèves avec son ID typé et son nom.

### `StudentPageData`

`StudentPageData` porte désormais des champs explicites :

```go
List        StudentListData
Form        StudentFormData
Student     StudentContext
ClassDelete StudentClassDeleteData
```

Les routes générales, routes Élèves et titre de page restent inchangés.

## 5. Nouvelle organisation des données

### Liste et regroupement des classes

`buildStudentListItems` reçoit les lignes produites par `GetStudentsWithClasses`. Il construit une slice de `StudentListItem` dans l'ordre de première apparition des élèves et conserve une map `student_id -> index de slice`.

Lorsqu'une nouvelle ligne concerne un élève déjà rencontré, sa classe est ajoutée directement à `items[index].Classes`. Cette stratégie reste correcte même si les lignes SQL de deux élèves sont intercalées et même si la slice est réallouée.

Plusieurs classes sont représentées par :

```go
Classes []StudentClassOption
```

Si le LEFT JOIN renvoie un élève sans classe, l'item est tout de même créé et reçoit une slice vide non nil. Les champs SQL nullables ne sont jamais exposés au template.

### URLs par élève

Lors de la création du premier item d'un élève, le builder calcule à partir de son ID :

- `EditURL` ;
- `DeleteURL` ;
- `StudentClassCodesURL`.

Le template itère uniquement sur `.List.Items` et utilise les URLs du même item. La slice parallèle `Action` et l'appel `index` ont disparu.

### Filtre et états vides

Le paramètre `class_filter` lu par le handler est conservé dans `StudentListData.CurrentClassFilter`. Le template continue de sélectionner l'option correspondante.

Les états sont dérivés des collections construites :

- `NoStudents` vaut vrai lorsque la liste d'items est vide ;
- `NoClasses` vaut vrai lorsque la liste de classes disponibles est vide.

Les slices `Items` et `Classes` sont initialisées comme slices vides non nil, ce qui donne un contrat stable au template et aux tests.

### Formulaires et contextes

`buildStudentFormPageData` convertit les `db.ClassCode` chargées par les requêtes existantes en `StudentClassOption`. Il alimente sans distinction métier le formulaire d'ajout et celui d'import CSV.

`buildStudentContextPageData` convertit l'élève possédé chargé par le handler en `StudentContext` pour Edit ou Delete.

`buildStudentClassDeletePageData` fournit l'ID `int64` et le nom de la classe à la confirmation de suppression groupée.

Les handlers conservent leurs lectures, validations et redirections ; ils délèguent seulement l'assemblage des données de vue à ces fonctions.

## 6. Suppression d'ExtraData

Le résultat est complet pour la section Élèves :

- `StudentPageData.ExtraData` a été supprimé ;
- aucun des six templates Élèves ne référence `ExtraData` ;
- `StudentPageData` et les structures utilisées par ces templates ne contiennent ni `any` ni `map[string]any` ;
- aucune structure `db.Student`, `db.ClassCode`, ligne SQLC ou structure `config.Student` n'est directement exposée aux templates ;
- l'ancien `StudentActionURLs` et les slices parallèles associées ont été supprimés.

Les types DB/SQLC restent légitimement utilisés à la frontière des builders, côté Go, puis sont convertis en modèles de vue.

## 7. Comportement métier préservé

Aucune règle métier n'a été modifiée concernant :

- la création d'un élève ;
- l'obligation de disposer d'au moins une classe avant l'accès aux formulaires Add/CSV ;
- la transaction atomique création élève + association à la classe ;
- l'édition du prénom et du nom ;
- le filtrage par classe ;
- la suppression individuelle ;
- la suppression des élèves d'une classe et la gestion des élèves multi-classes ;
- l'import et la validation CSV ;
- l'ownership utilisateur ;
- les relations élève/classe.

Les méthodes HTTP, noms de champs de formulaire, routes et redirections sont inchangés.

Aucun fichier SQL, aucune requête SQLC, aucun fichier généré, aucune migration et aucun élément du schéma n'ont été modifiés.

## 8. Tests

Le fichier `internal/handlers/students/viewData_test.go` couvre les contrats de vue suivants.

### Regroupement et liste

- regroupement de plusieurs classes sous un seul élève ;
- lignes de classes intercalées entre plusieurs élèves ;
- conservation de l'ordre de première apparition ;
- ID, prénom et nom corrects ;
- URLs Edit/Delete/Gestion des classes correctes pour l'élève ;
- élève sans classe issu du LEFT JOIN ;
- slice de classes vide non nil dans ce cas ;
- filtre de classe courant ;
- état liste vide ;
- état aucune classe disponible ;
- slices de liste et de classes vides non nil.

### Formulaires et contextes

- conversion des classes en options de formulaire ;
- formulaire avec classes ;
- formulaire sans classe avec slice vide non nil ;
- données Edit ;
- données Delete individuel ;
- ID et nom de la classe pour la suppression groupée.

### Templates et garde-fou

- rendu sans erreur du template de liste ;
- rendu sans erreur du formulaire Add ;
- rendu sans erreur du formulaire Edit ;
- rendu sans erreur du formulaire Delete ;
- rendu sans erreur du formulaire CSV ;
- rendu sans erreur de la suppression par classe ;
- vérification par réflexion que `StudentPageData` ne possède plus de champ `ExtraData` ;
- lecture des six templates et échec du test si `ExtraData` y réapparaît.

### Résultats des validations

- fichiers Go modifiés passés par `gofmt` : succès ;
- tests ciblés `go test ./internal/handlers/students -count=1` : succès ;
- tests complémentaires `go test ./internal/handlers/students ./internal/templates/data` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès.

## 9. Bénéfices architecturaux

Ce jalon apporte :

- un contrat clair et documenté entre handlers et templates ;
- la détection à la compilation des champs supprimés, renommés ou mal typés ;
- des templates plus simples, sans clés dynamiques ni association par index ;
- une séparation explicite entre les lignes DB et les besoins de l'interface ;
- des IDs restant des `int64` jusqu'à la construction des URLs ou au rendu des formulaires ;
- une construction de liste robuste pour les relations multi-classes ;
- des fonctions pures directement testables ;
- une base stable pour modifier ultérieurement la présentation sans toucher aux requêtes ou au métier.

## 10. Suite logique

Le prochain jalon recommandé est la modernisation UX de la section Élèves, en commençant par la liste. Son modèle de vue est maintenant typé, les items regroupent leurs actions et les états nécessaires sont explicites.

Cette modernisation UX n'est pas réalisée dans le présent jalon.

## Fichier créé par ce rapport

- `docs/audits/students-view-data-refactor.md`

Aucun fichier du refactor existant n'a été modifié pendant la rédaction de ce rapport et aucun commit n'a été créé.
