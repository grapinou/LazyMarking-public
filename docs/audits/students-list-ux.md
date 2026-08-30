# Modernisation UX de la liste Élèves

## 1. Objectif

Ce jalon modernise uniquement l’interface de la liste Élèves. Il s’appuie sur le modèle de données de vue typé livré au jalon précédent (`StudentListData`, `StudentListItem` et `StudentClassOption`) afin d’améliorer la lisibilité, la hiérarchie des actions, les états vides et l’utilisation sur écran étroit.

Le périmètre ne comprend aucun changement métier, aucune nouvelle route et aucune évolution de la persistance.

## 2. État avant modification

La page présentait plusieurs éléments hérités de l’ancienne interface :

- le titre visible était « Table des élèves », accompagné du texte minimal « Gestion des élèves. » ;
- les trois actions principales « Nom des classes », « CSV » et « Elève » avaient la même apparence et un ordre peu adapté au parcours courant ;
- le filtre de classe reposait sur un élément `<select>` stylé comme un bouton ;
- le tableau séparait les actions dans deux colonnes, « Edit/Sup » et « Edit », en plus de la colonne de classe ;
- la suppression de tous les élèves d’une classe était mélangée aux actions ordinaires en haut de page et se déclenchait dès le changement du select ;
- la suppression individuelle utilisait un symbole `☠` sans libellé explicite ;
- l’état vide se limitait à « Pas d'élève pour l'instant ... », sans explication ni parcours de démarrage.

## 3. Fichiers modifiés

- `internal/templates/students/table_students.html`
- `internal/handlers/students/viewData_test.go`

Aucun fichier Go de production n’a été modifié : les données de vue typées existantes suffisaient.

## 4. Nouvelle organisation de la page

La page utilise désormais un conteneur Bootstrap et un en-tête responsive comprenant :

- le titre principal « Élèves » ;
- un sous-titre expliquant la gestion des élèves, de leur appartenance aux classes et des imports ;
- trois actions clairement hiérarchisées :
  1. « Ajouter un élève », action principale ;
  2. « Importer un CSV », action secondaire ;
  3. « Gérer les classes », action de configuration secondaire.

Le contenu est ensuite organisé en blocs distincts : filtre, liste ou état vide, puis actions avancées. Les routes proviennent toujours de `StudentRoutes.AddURL`, `StudentRoutes.AddCSVURL` et `StudentRoutes.ClassCodesURL`.

## 5. Filtre par classe

Le filtre conserve une soumission HTTP GET et le paramètre `class_filter`. La valeur de `StudentListData.CurrentClassFilter` sélectionne toujours l’option courante parmi les `StudentListData.Classes`.

Le contrôle est maintenant un véritable champ de formulaire Bootstrap : label associé « Filtrer par classe », identifiant explicite et classe `form-select`. L’option « Toutes les classes » conserve une valeur vide.

Lorsque JavaScript est disponible, l’attribut `onchange` soumet immédiatement le formulaire, comme auparavant. Le bloc `<noscript>` fournit uniquement un bouton « Filtrer » aux navigateurs sans JavaScript ; il ne change ni la méthode GET, ni le paramètre envoyé, ni le traitement du handler.

Le filtre n’est pas rendu lorsque `StudentListData.NoClasses` est vrai.

## 6. Liste des élèves

La liste est placée dans un conteneur `table-responsive` et le tableau est réduit à trois colonnes sémantiques :

- « Élève » ;
- « Classe(s) » ;
- « Actions ».

Chaque ligne consomme directement un `StudentListItem`. Les classes, déjà regroupées dans `StudentListItem.Classes`, sont affichées sous forme de badges Bootstrap et peuvent se répartir sur plusieurs lignes. Un élève associé à plusieurs classes affiche donc un badge par `StudentClassOption`. Lorsque la slice est vide, la cellule affiche le texte neutre « Aucune classe ».

Aucun identifiant technique ni structure de base de données n’est exposé dans le template.

## 7. Actions par élève

Toutes les actions d’un élève sont regroupées dans une unique zone responsive :

- « Modifier » utilise `StudentListItem.EditURL` ;
- « Classes » utilise `StudentListItem.StudentClassCodesURL` ;
- « Supprimer » utilise `StudentListItem.DeleteURL`.

Les deux actions ordinaires utilisent de petits boutons outline secondaires. La suppression utilise un bouton outline danger, mais reste une action textuelle explicite. Les trois URLs pré-calculées par le view-model sont conservées sans reconstruction dans le template.

## 8. Suppression massive

La suppression de tous les élèves d’une classe a été déplacée dans une section distincte « Actions avancées », sous la liste ou l’état vide.

Le formulaire conserve :

- la méthode GET ;
- l’action `StudentRoutes.DeleteAllStudentURL` ;
- le paramètre `class_code_id` ;
- les classes disponibles issues de `StudentListData.Classes`.

L’utilisateur choisit une classe puis active le bouton explicite « Supprimer les élèves d’une classe ». Cette requête ouvre toujours la page de confirmation existante. Aucun POST et aucune suppression ne sont exécutés directement depuis la liste.

La section entière est masquée lorsqu’aucune classe n’est disponible.

## 9. États vides

### Aucun élève avec des classes disponibles

La page affiche un véritable état vide intitulé « Aucun élève ». Il explique que l’enseignant peut commencer par un ajout manuel ou un import CSV, puis présente les actions « Ajouter un élève » et « Importer un CSV ». Le filtre et la section d’actions avancées restent disponibles puisque des classes existent.

### Aucune classe disponible

Une alerte explique qu’une classe doit d’abord être créée pour rattacher des élèves et met en avant « Gérer les classes ». Dans l’état vide, le texte rappelle ce prérequis et ne propose comme action contextuelle que la gestion des classes.

Le filtre et la suppression massive ne sont pas rendus dans ce cas. Cette présentation respecte la règle métier existante imposant une classe avant la création d’un élève, sans déplacer ni dupliquer cette validation dans le template ou le handler.

## 10. Responsive et accessibilité

Les améliorations comprennent :

- un en-tête et des groupes d’actions utilisant `flex-wrap` ;
- un conteneur `table-responsive` autour du tableau ;
- des boutons textuels qui restent compréhensibles sans icône ;
- des icônes décoratives uniquement en complément des actions principales, avec `aria-hidden="true"` ;
- des labels reliés aux deux selects par `for` et `id` ;
- des en-têtes de colonnes dotés de `scope="col"` et le nom de l’élève utilisé comme en-tête de ligne avec `scope="row"` ;
- des titres de section reliés par `aria-labelledby` ;
- un libellé accessible contextualisé pour les actions de chaque élève ;
- la suppression du symbole destructif ambigu `☠` au profit du texte « Supprimer ».

## 11. Tests

Les tests de rendu ajoutés dans `viewData_test.go` couvrent :

- plusieurs élèves dans une même liste ;
- plusieurs classes pour un élève ;
- un élève sans classe et le libellé « Aucune classe » ;
- la conservation du filtre courant ;
- l’état sans classe ;
- l’état sans élève lorsque des classes sont disponibles ;
- les URLs `EditURL`, `StudentClassCodesURL` et `DeleteURL` portées par chaque élève ;
- les routes d’ajout, d’import CSV et de gestion des classes ;
- le formulaire GET de suppression massive vers la confirmation existante ;
- la conservation des paramètres `class_filter` et `class_code_id` ;
- l’absence des anciens libellés « Table des élèves » et « Edit/Sup » ;
- l’absence du symbole `☠` et de son ancienne entité HTML.

Résultats des validations du jalon :

- tests ciblés Élèves (`go test ./internal/handlers/students`) : réussis ;
- suite complète (`go test ./...`) : réussie ;
- `git diff --check` : réussi ;
- `gofmt` appliqué au fichier Go de test modifié.

## 12. Invariants

Ce jalon n’a apporté aucun changement aux éléments suivants :

- règles métier de la section Élèves ;
- handlers métier ;
- routes et paramètres attendus ;
- requêtes SQL ;
- code SQLC ;
- migrations et schéma ;
- traitement de l’import CSV ;
- contrôles d’ownership ;
- relations entre élèves et classes ;
- comportement effectif des suppressions.

La modification porte uniquement sur le template de liste et ses tests de rendu.

## 13. Suite logique

Le prochain jalon recommandé est la modernisation UX des formulaires Élèves, maintenant que la liste et les données de vue sont stabilisées :

- ajout manuel ;
- édition ;
- import CSV ;
- suppression individuelle ;
- suppression de tous les élèves d’une classe.

Cette suite n’est pas réalisée dans le présent jalon.
