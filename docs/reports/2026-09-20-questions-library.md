# P2 — Mes questions

## État initial

`GET /dashboard/questions` affichait déjà les familles personnelles, leurs
variantes intégralement dépliées et six actions par famille. Le titre « Banque
de questions », les six prérequis de création toujours affichés et les paramètres
placés avant les questions orientaient cette vue vers la préparation et la gestion.
Les caractéristiques des questions n'étaient pas affichées, sans recherche ni filtres.

Les routes GET/POST de création et d'édition existent dans
`internal/handlers/questions/routes.go`. Le formulaire de création est déjà
distinct à `/dashboard/questions/add`. Les mutations redirigent vers la liste.

Le modèle associe une question principale à une matière, un niveau, un thème,
une compétence, une difficulté et un barème. Les variantes dans `alt_questions`
référencent leur question principale et disposent de leurs réponses et images.
`questionfamilies.Build` regroupe ces variantes sans modifier le modèle métier.
Les lectures SQL contrôlent le propriétaire des questions, des variantes et des
caractéristiques. Les QCM référencent les questions principales ; la génération
utilise les lectures existantes pour sélectionner une formulation et construire
les questions. Ces relations et cette génération restent inchangées.

## Problème UX

La consultation était noyée sous les aides à la création et les formulations
alternatives. Il manquait les informations de classement pour retrouver rapidement
une question parmi plusieurs dizaines.

## Architecture retenue

Les URLs et handlers publics sont conservés. `TableQuestionsHandler` enrichit
les familles avec `GetFilteredQuestions`, requête SQL/sqlc existante utilisée
pour récupérer les caractéristiques avec isolation utilisateur. Trois lectures
en lot sont effectuées, sans requête par question.

`library.go` filtre en mémoire les seules familles personnelles chargées.
Cette approche convient à quelques dizaines de questions et ne nécessite ni
migration, ni nouvelle requête SQL, ni régénération sqlc. Le modèle des familles
est réutilisé. Les actions Modifier, Réponses, Variantes, Image, Aperçu et
Supprimer conservent leurs URLs.

## Page « Mes questions »

URL : `/dashboard/questions`. La navigation et la carte du dashboard portent
le même intitulé. Le bouton principal est « Ajouter une question ».

Chaque carte affiche l'énoncé, la matière, le niveau, le thème, la compétence
et le nombre de variantes. Le tri existant par identifiant décroissant conserve
les plus récentes en premier. Le compteur indique le nombre trouvé sur le total.
Les paramètres restent accessibles dans un panneau replié.

La recherche GET `q` porte sur l'énoncé principal, les variantes et les
caractéristiques affichées. Elle ignore casse et accents via la décomposition
Unicode NFD. Les filtres exacts `subject`, `level`, `theme` se combinent
avec la recherche ; leurs options sont les libellés distincts de la banque
personnelle, triés. Les critères restent visibles et peuvent être réinitialisés.

Une recherche sans résultat affiche un message distinct de la banque vide.
Une banque vide propose « Vous n’avez encore aucune question » et le CTA
« Créer ma première question ».

La mise en page conserve Bootstrap : cartes sans largeur fixe, textes longs
avec retour à la ligne, filtres empilés sur mobile et sur trois colonnes sur
desktop, actions pouvant revenir à la ligne. Pas de changement CSS global.

## Création

`GET /dashboard/questions/add` réutilise le formulaire sous le titre
« Ajouter une question », avec un lien « Retour à Mes questions » et Annuler.
Le POST existant crée la question puis redirige vers Mes questions.
Si des caractéristiques obligatoires manquent, le parcours de préparation
existant est conservé : redirection vers le premier paramètre manquant, puis
liens Suivant. Aucun nouveau formulaire ni éditeur n'est créé.

## Variantes

Le compteur ne compte que les formulations alternatives possédées par
l'utilisateur, sans inclure la question principale. Un panneau natif
`details/summary`, fermé par défaut, permet de les consulter. Le lien Variantes
ouvre leur gestion existante. La recherche d'une variante retrouve sa famille.

## Tests

`TestPersonalLibraryAndCreation` exerce les handlers authentifiés avec SQLite :
état vide personnel même si un autre utilisateur possède des questions ;
GET du formulaire ; création et redirection vers la liste ; métadonnées ;
deux variantes propres ; exclusion des variantes étrangères et des variantes
attachées à un parent étranger ; recherche sans accents et par variante ;
trois filtres séparés et combinés ; résultats vides et isolation utilisateur.

Les tests existants de mutations, ownership et regroupement des variantes sont
conservés. Validation (cache Go dans `/tmp/lazymarking-go-cache`) :

- `go test ./...` : succès.
- `go test ./internal/handlers/questions ./internal/questionfamilies ./internal/templates/data -count=1` : succès.
- `git diff --check` : succès.

Aucun commit ni push. Les modifications préexistantes de README.md et le
fichier local reset.sh sont conservés.

## Points restant ouverts

Une pagination et un filtrage SQL pourront être ajoutés si les bibliothèques
atteignent plusieurs milliers de questions. Aucun état actif/archivé ni
statistique d'utilisation n'est inventé à partir du modèle actuel.
Le contrôle responsive repose sur la structure Bootstrap et les templates ;
aucune session de validation visuelle dans un navigateur n'a été réalisée.
