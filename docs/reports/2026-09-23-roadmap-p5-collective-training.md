# P5 — Entraînements collectifs ciblés (23 septembre 2026)

## Audit préalable

- **P3** : `buildMarkingPedagogicalSummary` lit les résultats courants de `ListCurrentExamResultsForGeneration`. Il exclut les copies sans note finale ou sans détail cohérent, regroupe les questions par `main_question_id` (avec identité prudente pour les anciens snapshots), distingue les variantes, puis calcule un taux de réussite pondéré par les points, crédit partiel compris. `pedagogicalClassification` classe une famille « À retravailler » sous **40 %**, « Intermédiaire » jusqu'à 60 % inclus. P5 appelle cette fonction de P3 et utilise sa classification ; il ne recalcule aucun score de difficulté.
- **Modèle des questions** : `questions` porte matière, thème, niveau, compétence, difficulté, points, consigne et énoncé. `answers.state` vaut 0 ou 1, sans unicité de la valeur 1 : plusieurs réponses correctes sont permises. `alt_questions` et `alt_answers` représentent les variantes d'une famille. Le tirage et l'ordre des propositions varient selon la copie ; P3 neutralise cet ordre pour reconnaître les variantes.
- **Copies et snapshots** : `student_exam_content.content` contient un `config.QCM` JSON immuable pour chaque copie. `config.Question.Tags` conserve l'identifiant de famille et, sur les copies récentes, le type et l'identifiant exacts de variante. Les résultats courants sont liés à ce snapshot, plutôt qu'à la bibliothèque modifiable.
- **P4** : les coupons créent un token aléatoire de 32 octets encodé en base64 URL sans remplissage (43 caractères), mais ajoutent un code personnel et une expiration. Les QR de P4 utilisent `github.com/skip2/go-qrcode`. P5 réutilise cette bibliothèque et le format aléatoire, avec une nouvelle table et la route `/train/{token}` ; aucun code ni lien P4 n'intervient.
- **P4.5** : `internal/mathcontent.Parse` distingue texte littéral et fragments `$...$`, et refuse les formes dangereuses. P5 appelle directement ce parseur et compile uniquement les fragments autorisés avec le binaire Typst déjà utilisé pour les PDF. Aucun langage mathématique ou moteur JavaScript supplémentaire n'a été ajouté.
- **Interface existante** : le dashboard Bootstrap, `RenderMergeTemplate`, la session professeur et le middleware CSRF global servent aussi aux pages P5.

## Données et composition

La migration `0049_collective_training.sql` ajoute deux tables : `training_decks` (propriétaire, génération source facultative, titre, token, état) et `training_cards` (ordre, sélection, référence informative à la question/variante source, contenu compact JSON et rendu HTML précompilé). La référence source ne sert jamais à afficher un deck publié et ne contraint pas la suppression de la bibliothèque. Les requêtes sont définies dans `db/query/training.sql` et générées avec sqlc. La configuration sqlc renomme le modèle généré P4 `StudentCopyAccessRow` pour éviter la collision avec le type enrichi déjà utilisé par P4.

Depuis l'analyse P3, « Créer un entraînement » charge uniquement les résultats d'une génération réussie appartenant au professeur. Chaque variante observée dans les copies corrigées et cohérentes devient une carte candidate. Les cartes des **familles** classées « À retravailler » par P3 sont présélectionnées. Les autres variantes observées ainsi que les questions et variantes textuelles actuelles de la bibliothèque du professeur, limitées aux mêmes thèmes, restent disponibles à cocher. Cette recherche bornée par thème (40 familles au plus par thème) réutilise les requêtes existantes de propriété, questions et réponses ; elle n’attribue aucun score P3 aux cartes supplémentaires. Le professeur peut modifier le titre, inclure ou retirer des cartes et saisir leur ordre. En absence de famille faible, le brouillon s'ouvre sans carte présélectionnée, avec une invitation au choix manuel. Aucune publication vide n'est acceptée. Les questions illustrées et les QCM textuels incomplets ne sont pas proposés : cette V1 ne publie pas d'image privée de la bibliothèque.

Chaque carte contient seulement consigne, énoncé et propositions avec leurs états corrects. Le contenu est copié depuis le snapshot de correction **à la création du brouillon**. Au moment de la publication, chaque fragment mathématique des cartes sélectionnées est validé et rendu en SVG, puis `rendered_json` est figé. Les modifications ultérieures d'une question, d'une variante ou d'une copie source ne changent donc pas le deck. Une erreur de parse ou de compilation Typst retourne un message avec la carte et empêche la publication. Le texte normal reste du HTML échappé ; seules les formules deviennent des images SVG intégrées, avec texte alternatif. Le rendu est effectué à la publication, pas à chaque visite élève.

## Cycle de vie et partage

Le cycle est `draft → published → closed → published`. Seul le brouillon est éditable. Un token cryptographiquement aléatoire est créé par deck ; l'URL ne contient pas l'ID SQL et reste identique lors d'une réouverture. Le brouillon et le deck fermé répondent 404 sur la route publique, y compris sur l'endpoint de correction. La page professeur affiche l'état, la composition, le lien public, une copie du lien et un QR unique lorsqu'il est publié. `APP_BASE_URL` doit être configuré avec une URL HTTPS publique (HTTP n'est accepté que pour localhost) avant publication. Le QR encodé dans la page a été réellement décodé dans le test et comparé au lien affiché.

Les pages professeur sont sous `login.CheckAuth`. Chaque lecture et mutation de deck filtre par `user_id`. Les formulaires POST passent par le middleware CSRF existant ; le test vérifie le refus sans token. Le token public P5 est indépendant des accès P4.

## Parcours élève

La page `/train/{token}` est autonome et adaptée aux petits écrans : accueil, une carte à la fois, choix accessibles au clavier, validation, retour correct/incorrect textuel, question suivante, score `réussites / cartes` et pourcentage, puis « Recommencer ». Une carte avec une seule bonne réponse emploie des radios ; plusieurs bonnes réponses emploient des cases à cocher. Le serveur compare **exactement** l'ensemble choisi à l'ensemble correct. Une sélection partielle ou contenant un mauvais choix est incorrecte. Les bonnes réponses sont renvoyées uniquement après validation, pour cette carte. Le navigateur verrouille les choix, calcule son score localement et le remet à zéro au redémarrage. L'ordre est fixe dans cette V1.

La page publique ne contient pas le corrigé dans son HTML initial, ni nom, prénom, identifiant de copie, résultat individuel ou lien P4. L'endpoint `/check` ne fait que lire le snapshot publié et renvoyer la correction ; il ne crée ni tentative, ni cookie d'identité, ni historique. Le cookie CSRF technique protège la requête POST sans identifier un élève.

## Tests et validation

Les tests P5 couvrent la création depuis une génération propriétaire et le refus d'une génération étrangère, la présélection P3 et le cas sans difficulté, la modification de composition et d'ordre, l’ajout d’une question et d’une variante de la bibliothèque, le snapshot face à une modification de question source, les états et la réouverture, le token inconnu, les QCM à une et plusieurs réponses, les sélections partielles et excessives, l'interdiction de publier une formule dangereuse, les quatre exemples Typst (`rho`, `E_c`, unité `mL`, `sqrt`), l'absence de données élève dans le HTML public, l'absence de modification de base lors des réponses, le QR extrait de la page professeur et décodé, l'isolation entre professeurs et le CSRF. Un scénario Chromium headless suit réellement démarrage → trois corrections → score `3 / 3` → recommencement.

Commandes exécutées :

- `go test ./...` : réussi.
- `go test -count=1 ./...` : réussi.
- `go test -race ./...` : réussi.
- `git diff --check` : réussi.
- `./scripts/check.sh` : réussi.
- `go test ./internal/handlers/training -count=1` et scénario Chromium : réussis.

Principaux fichiers : `db/migrations/0049_collective_training.sql`, `db/query/training.sql`, `internal/db/training.sql.go`, `internal/handlers/marking/trainingCandidates.go`, `internal/handlers/training/`, `internal/handlers/tools/trainingMath.go`, `internal/templates/training/`, ainsi que les liens dans le bilan P3, le dashboard et l'enregistrement des routes serveur.

Limites assumées : uniquement les QCM textuels observés dans les copies corrigées ou présents dans la bibliothèque du même thème ; pas d'images privées, de thèmes supplémentaires hors de cette sélection, de question libre corrigée automatiquement, de personnalisation par élève, de stockage de score, d'adaptation ni de répétition espacée. Le choix de cartes reste collectif et rapide.
