# Modernisation UX des classes d’un élève

## 1. Objectif

Ce jalon modernise uniquement l’interface du module `studentClassCodes`, qui permet de consulter les classes d’un élève, d’ajouter une association et de retirer une association lorsque l’invariant métier l’autorise.

Il s’appuie sur les données de vue typées et sur la protection serveur atomique déjà en place. Aucun changement n’est apporté aux mutations, aux routes ou à la persistance.

## 2. UX avant modification

La liste affichait le titre « Gestion des classes d'un élève », deux paragraphes séparés sur la règle métier et le lien anglais « Back to students ». La table utilisait une colonne « Sup », un bouton constitué uniquement d’une icône de corbeille et un simple tiret lorsque le retrait était interdit.

Le formulaire d’ajout utilisait des espacements imbriqués, un label non associé, le bouton « Ajouter ! », aucun contexte élève visible et aucune action Annuler. Lorsque toutes les classes étaient déjà associées, le GET redirigeait vers un ancien message d’erreur au lieu de présenter un état informatif.

## 3. Fichiers modifiés

- `internal/templates/studentClassCodes/table_student_class_codes.html`
- `internal/templates/studentClassCodes/add_form_student_class_code.html`
- `internal/templates/data/studentClassCode.go`
- `internal/handlers/studentClassCode/viewData.go`
- `internal/handlers/studentClassCode/viewData_test.go`
- `internal/handlers/studentClassCode/handlers.go`
- `docs/audits/student-class-codes-ux.md`

Le changement dans le handler concerne uniquement le rendu du GET Add quand aucune classe n’est disponible. Le POST et les règles métier sont inchangés.

## 4. Contexte élève

La page principale utilise désormais le titre « Classes de l’élève » et affiche immédiatement le prénom et le nom provenant de `StudentClassListData.Student`. Le sous-titre explique que la page gère les classes auxquelles cet élève est rattaché.

Le formulaire Add affiche également le contexte provenant de `StudentClassFormData.Student`, sous le titre « Ajouter une classe à l’élève ».

Aucun ID technique n’est présenté à l’utilisateur.

## 5. Nouvelle liste des classes

La page est organisée dans un conteneur Bootstrap avec :

- l’action principale « Ajouter une classe » utilisant directement `List.AddURL` ;
- l’action secondaire « Retour aux élèves » utilisant `Routes.StudentURL` ;
- un rappel concis de la règle métier dans une alerte légère ;
- une table responsive à deux colonnes, « Classe » et « Actions ».

Chaque ligne utilise un `StudentClassListItem`. Le nom est affiché depuis `ClassName` et l’action éventuelle utilise la `DeleteURL` déjà calculée côté Go. Aucune URL n’est reconstruite dans le template.

## 6. Comportement AllowedDelete

`AllowedDelete` reste une aide de présentation dérivée du nombre de classes associées. Il ne remplace pas la protection atomique du serveur.

Lorsque `AllowedDelete` est vrai, chaque relation affiche un bouton outline danger « Retirer ». Ce terme décrit la suppression de l’association élève/classe, et non la suppression de l’élève ou de la classe générale.

Lorsque `AllowedDelete` est faux, aucun lien actif n’est rendu. La cellule affiche « Classe obligatoire » avec une explication accessible, et un texte sous la table indique : « Impossible de retirer la dernière classe de l’élève. »

## 7. Cas dernière classe

Avec une seule classe, le template n’expose aucune `DeleteURL`. L’utilisateur comprend pourquoi l’action n’est pas disponible, sans déclencher volontairement une erreur serveur.

La vue reste défensive uniquement : même en cas d’appel forgé, la requête SQLC conditionnelle et le handler empêchent atomiquement de retirer la dernière relation. Cette protection métier n’a pas été modifiée dans ce jalon.

## 8. Cas aucune classe

Bien que l’invariant rende ce cas anormal pour les nouvelles mutations, `StudentClassListData.NoClasses` peut représenter une donnée historique incohérente.

La page affiche alors un état explicite « Aucune classe associée », précise que l’élève n’est actuellement rattaché à aucune classe et met en avant « Ajouter une classe ». Aucune relation n’est créée automatiquement et l’incohérence n’est pas masquée.

## 9. Formulaire Add

Le formulaire présente :

- le titre « Ajouter une classe à l’élève » ;
- le prénom et le nom de l’élève ;
- le sous-titre « Sélectionnez une classe supplémentaire pour cet élève. » ;
- un vrai groupe Bootstrap avec le label associé « Classe » ;
- uniquement les options de `StudentClassFormData.Classes`.

Le contrat existant est conservé : POST vers `StudentClassCodeRoutes.AddURL`, champ caché `student_id`, select `class_code_id` requis. Les actions sont « Ajouter la classe » et « Annuler ».

Pour éviter une reconstruction d’URL dans le template, `StudentClassFormData` expose désormais `ReturnURL`. Le builder la calcule vers la page des classes de l’élève avec son `student_id`.

## 10. Cas aucune classe disponible à ajouter

Lorsque `Form.Classes` est vide, le GET Add rend désormais la page au lieu de rediriger vers l’ancien message d’erreur.

Le template affiche : « Toutes les classes disponibles sont déjà associées à cet élève. » Il ne rend ni formulaire POST, ni select vide, ni bouton d’ajout impossible. La seule action est « Retour aux classes de l’élève », via `Form.ReturnURL`.

Cette évolution est purement une présentation du même état : aucune association supplémentaire n’est autorisée et le POST conserve toutes ses validations.

## 11. Wording retenu pour le retrait

Le libellé retenu est « Retirer ».

Il est court dans la table et décrit correctement l’effet réel de `DeleteStudentClassCodeByStudentID` : la relation entre cet élève et cette classe est retirée. La classe générale et l’élève ne sont pas supprimés.

## 12. Accessibilité et responsive

Les améliorations comprennent :

- conteneurs Bootstrap et carte de formulaire de largeur raisonnable ;
- en-têtes et groupes d’actions avec `flex-wrap` ;
- contexte élève visible et compatible avec les contenus longs ;
- `table-responsive` ;
- en-têtes de colonnes avec `scope="col"` et classes avec `scope="row"` ;
- label relié au select avec `for` et `id` ;
- boutons et liens textuels ;
- icône décorative marquée `aria-hidden="true"` ;
- explication visible et texte supplémentaire pour lecteur d’écran lorsque le retrait est interdit ;
- danger communiqué par le verbe « Retirer », pas seulement par la couleur.

## 13. Tests

Les tests couvrent désormais :

- prénom et nom de l’élève sur la liste et le formulaire ;
- plusieurs classes avec leurs noms ;
- association de chaque `DeleteURL` au bon item ;
- conservation de `List.AddURL` et du retour vers Élèves ;
- bouton « Ajouter une classe » ;
- présence de « Retirer » lorsque `AllowedDelete=true` ;
- absence de lien destructif, « Classe obligatoire » et explication complète lorsque `AllowedDelete=false` ;
- état défensif sans classe et action d’ajout ;
- formulaire Add : route/méthode POST, `student_id`, `class_code_id`, options, bouton et annulation ;
- calcul et rendu de `Form.ReturnURL` ;
- état sans classe disponible : message, retour et absence de select/POST/bouton impossible ;
- absence de « Gestion des classes d'un élève », « Back to students », colonne « Sup » et ancien tiret isolé.

Résultats :

- `go test ./internal/handlers/studentClassCode` : réussi ;
- `go test ./...` : réussi ;
- `git diff --check` : réussi ;
- `gofmt` appliqué aux fichiers Go modifiés.

## 14. Invariants métier préservés

Aucun changement n’a été apporté à :

- l’invariant « au moins une classe » ;
- la protection atomique SQLC ;
- les contrôles ownership ;
- l’ajout et la suppression des relations ;
- la création des classes ;
- les routes, méthodes ou paramètres HTTP ;
- SQL, SQLC, schéma ou migrations.

Le module demeure fondé sur les PageData typées, sans `ExtraData` ni reconstruction de route dans les templates.

## 15. Suite logique

Le prochain jalon recommandé est un audit de clôture du workflow Élèves, couvrant la cohérence entre Élèves, Classes, relations élève/classe et leurs protections serveur.

Cet audit n’est pas réalisé dans le présent jalon.
