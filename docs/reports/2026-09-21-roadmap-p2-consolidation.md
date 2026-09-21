# Roadmap P2 — consolidation fonctionnelle

Date de l'audit : 21 septembre 2026.

Ce document concerne le nouveau jalon « Roadmap P2 » de consolidation. Il ne
renomme et ne réinterprète aucun des anciens jalons P2, P5, P6.1 ou P6.2.

## 1. Baseline

### Dépôt et état initial

- branche : `main` ;
- commit initial et public de référence :
  `ca882f88ab1808be1dda7818b8e07ae20a288e6e` — `Add shared QCM library` ;
- remotes présents : `origin` vers le dépôt LazyMarking et `public` vers
  `grapinou/LazyMarking-public` ;
- working tree initial : `README.md` modifié et `reset.sh` non suivi. Ces deux
  changements préexistaient à l'audit, sont étrangers à P2 et n'ont été ni
  modifiés, ni supprimés, ni ajoutés au commit ;
- aucun écart entre `HEAD` et le commit public indiqué dans la demande.

### Environnement local

L'environnement de validation possède Go 1.26.5, OpenCV 4.12.0 exposé par
`pkg-config`, Typst 0.15.1, `pdftotext` 24.02.0, Goose 3.27.3 et sqlc 1.31.1.
Les tests de génération Typst, de rasterisation, de QR et de correction ont donc
pu être réellement exécutés ; ils ne sont pas seulement des assertions sur des
chaînes de templates.

### État initial des validations

Avant la modification :

- `./scripts/check.sh` réussissait entièrement : modules, replay des migrations
  0001 à 0047, formatage, `go vet`, tests, build et `git diff --check` ;
- `go test -race ./...` réussissait localement ;
- les tests fonctionnels déjà présents couvraient les bibliothèques partagées,
  les snapshots, la génération/correction et les imports cumulatifs.

### État initial de la CI

Le run GitHub Actions du commit initial échouait dans les jobs `check` et
`race`, après le replay réussi des migrations. Le runner Ubuntu possédait
`pkg-config` mais pas le fichier `opencv4.pc` ni les bibliothèques de
développement OpenCV ; la compilation de `gocv` échouait donc avant les tests.
Run observé :
<https://github.com/grapinou/LazyMarking-public/actions/runs/35646182577>.

La correction minimale installe `libopencv-dev`, sans recommandations, dans les
deux jobs avant toute commande Go. Les commandes communes restent
`./scripts/check.sh` et `go test -race ./...` ; aucun package et aucun test
OpenCV n'est exclu. Le workflow a été chargé avec PyYAML et ses deux séquences
de jobs ont été inspectées. Une exécution distante ne sera possible qu'après
publication du commit.

## 2. Cartographie des parcours audités

### Questions

Le parcours principale → réponses → variantes → réponses/images de variante →
édition → « Mes questions » → composition QCM a été relu dans les handlers,
requêtes sqlc, contraintes d'ownership et tests HTTP. La consigne est portée par
la principale, héritée par toutes les variantes, rendue dans les aperçus et
figée dans le snapshot de génération.

### QCM

Le parcours création → ajout de familles → positions → réordonnancement →
aperçu portrait/paysage → examen utilise des relations `qcm_questions`
propriétaires et ordonnées. Les mutations sont transactionnelles lorsque
plusieurs positions doivent changer. Les tests vérifient les limites, les IDs
étrangers et le rollback d'un échange incomplet.

### Partage

Les parcours Alice/Bob des familles et des QCM ont été exécutés. Ils couvrent
le privé par défaut, la publication explicite et idempotente, l'aperçu complet,
la copie, le retrait immédiat, la suppression/modification de la source et
l'indépendance de la copie. Un QCM partagé ouvre uniquement une lecture
contextuelle des familles réellement présentes dans sa composition ; il ne les
publie pas dans la bibliothèque de questions.

### Examen

Le parcours QCM → examen → classe/élèves → génération individualisée persiste
un `student_exam_content` complet, les coordonnées par page, les références
PNG et les QR. Un test réel génère et corrige trois pages, six questions et 48
réponses, dont une variante illustrée, puis modifie la banque et démontre que
le snapshot et la correction historiques restent inchangés.

### Correction

Les tests couvrent l'import initial, les décisions automatiques et humaines,
les états en attente, la régénération des artefacts, un import de rattrapage et
le bilan cumulatif. Les résultats antérieurs restent présents ; le résumé
retient l'état final courant par élève sans modifier les lignes des imports
précédents. Les PDF du premier lot, du rattrapage et des révisions ont été
produits et extraits avec `pdftotext`.

## 3. Défauts trouvés et corrections

### CI incapable de compiler `gocv`

- **Symptôme** : les jobs `check` et `race` échouaient avec « Package
  `opencv4` was not found ».
- **Cause** : `.github/workflows/ci.yml` installait Go et Goose mais aucune
  dépendance système OpenCV sur `ubuntu-latest`.
- **Gravité** : importante ; la validation distante ne pouvait atteindre ni les
  tests ordinaires ni le détecteur de races.
- **Correction** : installation de `libopencv-dev` dans chacun des deux jobs.
- **Fichier** : `.github/workflows/ci.yml`.
- **Validation** : syntaxe YAML chargée, commandes Go inchangées, compilation
  locale complète avec `gocv` et OpenCV disponible.

### Matrice récente de migrations insuffisamment explicite

- **Symptôme** : le démarrage neuf, 44 → 47, les trois copies de runtime,
  l'échec d'une migration et une DB plus récente étaient couverts, mais aucun
  test unique ne prouvait 44/45/46/47 → 47 avec un historique complet de
  génération et correction.
- **Cause** : les tests P5.1, P6.1 et P6.2 avaient évolué séparément.
- **Gravité** : importante pour la non-régression ; aucune corruption observée.
- **Correction** : ajout d'une matrice synthétique contenant question, QCM,
  examen, génération, élève, snapshots complet et par page, job, copie
  corrigée, résultat par question et détection. Chaque transition contrôle les
  données, le privé par défaut, les partages déjà explicites, la version,
  `integrity_check` et `foreign_key_check`.
- **Fichier** : `internal/db/migrations_test.go`.
- **Tests** : `TestOpenMigratedDBRecentTransitionsPreserveHistoryAndPrivacy`
  et ajout de l'intégrité au cas base neuve/base à jour.

### Défauts écartés après audit

Les handlers de fichiers privés ne posent pas tous localement leurs en-têtes,
mais les routes sont enveloppées par `AuthMiddleware`, qui impose
`Cache-Control: no-store`, puis par `SecurityHeaders`, qui impose notamment
`X-Content-Type-Options: nosniff`. Il n'y avait donc pas de défaut de cache à
corriger dans chaque helper.

Aucune autre perte de données, fuite d'ownership, copie partielle, erreur 500
reproductible ou incohérence de statut n'a été trouvée. Aucun remaniement
fonctionnel n'a été entrepris.

## 4. Migrations et démarrage

`cmd/server` appelle `OpenMigratedDB` avant de construire les queries et avant
d'enregistrer ou servir les routes. Les migrations sont embarquées depuis
`db/migrations`, exclusivement appliquées vers le haut et bornées par un délai
de deux minutes. Une erreur ferme la connexion et arrête le serveur. Une base
plus récente que le binaire est refusée.

Résultats :

| Source | Destination | Migrations appliquées | Résultat |
| --- | --- | --- | --- |
| vide | 47 | 1 à 47 | données/schéma valides, intégrité OK |
| 44 | 47 | 45, 46, 47 | historique conservé, aucun partage implicite |
| 45 | 47 | 46, 47 | historique conservé, aucun partage implicite |
| 46 | 47 | 47 | partage question conservé, aucun QCM publié implicitement |
| 47 | 47 | aucune | idempotent, partages explicites conservés |
| 48 | refus | aucune | aucun handle utilisable retourné |

L'injection d'une migration SQL invalide prouve que sa transaction est annulée,
que la migration précédente reste intacte et qu'aucune connexion utilisable
n'est rendue au serveur.

Les fixtures ont été ouvertes en lecture seule pour inventorier leurs versions,
puis copiées dans des répertoires temporaires par les tests :

- `testdata/smoke/app.db` : 40 → 47 ;
- `testdata/real/app.db` : 41 → 47 ;
- `testdata/2026-2027/app.db` : 47 → 47.

Leur empreinte source est contrôlée avant/après, les données métier stables sont
comparées, les partages sont inchangés et les PRAGMA d'intégrité et de clés
étrangères passent. Aucune base source n'a été migrée par cet audit.

## 5. Multi-utilisateur et sécurité

Les tests Alice/Bob vérifient les accès GET et les mutations POST pour les
principales, réponses, variantes, réponses alternatives, images, QCM et
relations ordonnées. Connaître un ID étranger conduit à un 404 ou à un refus
explicite, sans utiliser de `user_id` fourni par le formulaire. L'identité vient
de la session validée.

Les opérations mutantes sont en POST et traversent le middleware CSRF global.
L'inventaire teste les 76 formulaires mutables et les parcours de partage
testent également l'absence de jeton. Les IDs vides, négatifs, invalides ou hors
plage sont refusés. Les accès anonymes aux bibliothèques et aperçus partagés
sont redirigés vers l'authentification.

Les images personnelles sont autorisées par la référence DB et le propriétaire,
jamais par le seul nom. Les images partagées sont résolues après contrôle du
partage, de la famille, de la composition QCM et de la variante. Les helpers de
fichiers rejettent composants de chemin dangereux, traversées, symlinks,
répertoires substitués et fichiers non réguliers. Les réponses authentifiées ne
sont pas mises en cache ; les routes d'images partagées ajoutent en plus leur
propre `private, no-store`.

## 6. Bibliothèques partagées

### Familles de questions

L'absence de ligne dans `question_shares` signifie privé. Seul le propriétaire
peut publier/retirer, avec `ON CONFLICT DO NOTHING` pour l'idempotence. La copie
duplique la principale, la consigne, les réponses et états, toutes les variantes,
leurs réponses, leurs images, les classifications et le barème. Elle reste
privée et porte une provenance informative indépendante de la source.

Les classifications textuelles équivalentes par espaces, casse, accents et
normalisation Unicode sont réutilisées chez le destinataire ; leur libellé
affiché n'est pas réécrit et aucune équivalence sémantique n'est inventée.

### QCM

`qcm_shares` est également privé par absence. Une lecture contextuelle vérifie
simultanément QCM partagé, auteur commun, relation et famille. Une famille privée
du QCM reste invisible par les routes personnelles et par la bibliothèque des
familles. Une famille hors composition et une variante d'une autre famille sont
refusées.

La copie utilise une transaction globale : QCM, familles distinctes, fichiers,
positions, relations et provenances, puis un seul commit. Les collisions de nom
produisent `— copie`, puis un suffixe numérique. Les tests injectent des erreurs
sur une famille tardive, une image, une relation, le commit et un conflit de
snapshot SQLite ; aucune ligne ni aucun fichier partiel ne subsiste.

### Fichiers et concurrence

Chaque image est ouverte sans suivre de symlink, copiée avec un UUID, créée en
exclusif, synchronisée et fermée avant insertion de sa référence. Toute erreur
ordinaire déclenche rollback et suppression des fichiers créés. SQLite en mode
WAL refuse avec `SQLITE_BUSY_SNAPSHOT` la promotion en écriture d'une vue
devenue obsolète ; une nouvelle tentative copie ensuite un état complet et
cohérent.

Un arrêt brutal du processus entre la création physique d'un fichier et le
rollback peut laisser un fichier sans référence. Ce n'est pas une dépendance ou
une corruption de copie ; le scanner/purger d'images existant traite ces
orphelins après délai de grâce. Une atomicité absolue entre SQLite et le système
de fichiers nécessiterait une architecture de stockage différente et reste hors
périmètre.

## 7. Examens, snapshots et correction

La génération sérialise le contenu réellement tiré, y compris consigne,
énoncé/variante, réponses mélangées, caractéristiques, barème, identité élève et
`layout_version`. Les coordonnées par page et les références PNG persistées
priment lors de la correction. Modifier ensuite la question, le QCM ou l'ordre
ne réécrit aucun snapshot.

Les snapshots anciens sans `layout_version` restent sur le renderer legacy.
Les snapshots actuels avec références natives ne dépendent plus d'une
recompilation de la banque pour l'alignement. La limite historique connue reste
qu'un très ancien snapshot sans référence PNG native doit encore retrouver les
fichiers d'images qu'il nomme pour une recompilation.

La correction associe chaque résultat à un `student_exam`, donc à une génération
et un élève, puis à un `question_index` du snapshot. Les états effectifs suivent
la priorité revue humaine, décision automatique, état historique. Les imports
successifs restent des jobs distincts avec leur nom de source ; le résumé
cumulatif conserve les copies antérieures et ajoute les élèves vus lors du
rattrapage. Les états corrigé, non vu, en revue et correction partielle restent
distincts.

## 8. Préparation des chantiers futurs, sans implémentation

### Analyse pédagogique par question

Déjà disponible : examen/génération, élève, index de question, contenu figé,
réponses attendues réellement mélangées, états finaux revus, barème figé et
`tags.main_question_id`. Il est donc possible de relier un résultat final à la
famille mère et au contexte historique sans relire la banque actuelle.

Trou de traçabilité : le snapshot ne conserve pas l'ID stable de la variante
effectivement présentée. Son contenu est bien figé et les rapports actuels
distinguent les contenus, mais deux variantes de même texte ou une analyse
longitudinale par variante ne peuvent pas être attribuées sans ambiguïté. Pour
de futures statistiques durables, une extension additive des nouveaux
snapshots avec un identifiant de version source sera probablement utile ; les
anciens snapshots devront garder un fallback par contenu/famille. Aucune
statistique ni couleur de réussite n'a été ajoutée ici.

### Restitution individuelle sans compte élève

Déjà disponible : identité de l'élève, copie individualisée, snapshot, pages,
résultat final, réponses détectées/revues, score et provenance du job. Le modèle
peut produire une vue limitée à un `student_exam`.

Manquent volontairement : un mécanisme public opaque, révocable et non
énumérable ; sa durée de vie ; l'autorisation de téléchargement ; et la
politique de cache/publication de l'artefact individuel. Il faudra empêcher
qu'un token ou un ID permette de changer d'élève, de génération ou d'enseignant.
Aucun portail, token ou QR de restitution n'a été créé.

### Entraînement collectif

Les familles, variantes, réponses, consignes, images et classifications sont
réutilisables. Les résultats corrigés permettent déjà de remonter aux familles
mères. Restent à définir dans un chantier futur : modèle de deck, règles de
sélection, snapshot ou stratégie de copie, sessions et résultats
d'entraînement. L'absence d'ID de variante dans les anciens snapshots limite
les sélections fines par variante. Aucun deck n'a été créé.

## 9. Dette et limites connues

### Critique

Aucune limite critique reproductible ne reste ouverte après l'audit.

### Important

1. La correction CI est cohérente et minimale, mais le run GitHub distant ne
   peut être observé avant publication de ce commit.
2. Les snapshots ne portent pas l'identifiant stable de la variante tirée ; ce
   point concerne surtout les futures analyses longitudinales.
3. SQLite et le système de fichiers ne forment pas une transaction distribuée.
   Les échecs gérés sont nettoyés et le purger couvre les orphelins après crash.

### Confort / futur

1. Plusieurs chemins de runtime sont relatifs au répertoire de lancement. Les
   scripts `run-*` se placent explicitement dans le bon runtime ; un futur
   packaging devra rendre ce contrat explicite.
2. Les très anciens snapshots sans références PNG natives gardent leur
   dépendance historique aux images nécessaires à leur recompilation.

## 10. Validation finale

Commandes de baseline et ciblées exécutées avec succès pendant l'audit :

```text
./scripts/check.sh
go test -race ./...
go test -count=1 ./internal/db ./internal/handlers/tools ./internal/handlers/qcm
go test -count=1 ./internal/sharedlibrary ./internal/handlers/questions ./internal/handlers/qcm ./internal/handlers/qcmQuestions ./internal/handlers/qcmPreview ./internal/httpsecurity ./internal/imagestorage
go test -count=1 ./internal/handlers/generateExams ./internal/handlers/marking ./internal/handlers/tools
go test -count=1 -run <matrice migrations> -v ./internal/db
go test -count=1 -run <parcours partagés et rollback> -v ./internal/handlers/questions ./internal/handlers/qcm ./internal/sharedlibrary
go test -count=1 -run <snapshots/génération/correction cumulative> -v ./internal/handlers/tools ./internal/handlers/marking
```

La validation finale complète a ensuite été exécutée avec les commandes exactes
demandées :

```text
./scripts/check.sh
go test -count=1 ./...
go test -race ./...
go build ./...
git diff --check
```

Résultats : toutes ces commandes réussissent. `./scripts/check.sh` rejoue les 47
migrations puis réussit `go mod verify`, `gofmt`, `go vet`, `go test`, le build
et le contrôle des espaces. La passe sans cache réussit tous les packages, dont
les tests réels de génération/correction. La passe `-race` réussit sans course
détectée ; son package le plus long, `internal/handlers/tools`, termine en
76,384 s. `go build ./...` et `git diff --check` réussissent séparément.

La CI distante reste à exécuter après push ; aucun succès non exécuté n'est
revendiqué. sqlc n'a pas été régénéré car ni les requêtes SQL, ni le schéma, ni
les fichiers générés n'ont changé.

## 11. État Git de livraison

Le commit de consolidation porte le message `Consolidate LazyMarking workflows`
et a pour parent `ca882f88ab1808be1dda7818b8e07ae20a288e6e`. Son SHA est fourni par le
`git log -1 --oneline` final : un commit ne peut pas contenir sa propre empreinte
sans la modifier.

Les changements inclus sont limités à la dépendance CI OpenCV, aux tests de
migration et à ce rapport. `README.md` et `reset.sh` restent hors commit et
doivent encore apparaître dans `git status --short` après livraison.

## Décision

**P2 : READY WITH KNOWN LIMITATIONS.**

Les validations locales, migrations, parcours partagés, génération réelle,
snapshots et correction cumulative sont verts. La CI dispose maintenant de la
dépendance nécessaire pour compiler `gocv`. Les limites restantes sont
documentées, sans effet démontré sur les fonctions actuelles : validation
distante en attente du push, identité stable de variante pour les futures
analyses, et atomicité inter-systèmes compensée par le nettoyage existant.
