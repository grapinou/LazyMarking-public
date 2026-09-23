# C1 — Consolidation pré-V1 et préparation du test en classe

23 septembre 2026. État audité : `c1e19e8ca1d70a9e461e5af1acabb2a20fea3d74` et arborescence de travail P4/P4.5/P5 **non commise**. L'arborescence était déjà largement modifiée au début de C1 ; aucune modification préalable n'a été effacée ni aucune base réelle migrée directement pendant cet audit.

## État général

La chaîne est `questions → QCM/variantes → génération PDF et student_exam_content → import/scans → correction/revue → lecture cumulative P3 → coupons/PDF P4 → sélection P3/deck P5`. Les rapports P3, P4, P4.5 et P5 du dépôt ont été relus. P3 utilise la dernière correction définitive de chaque copie, exclut les revues ouvertes des statistiques, regroupe les variantes par famille et classe les familles « à retravailler » sous 40 %. P5 réutilise cette classification et fige ses cartes à la création du brouillon, puis leur rendu Typst à la publication.

Les migrations 0001 à 0049 se rejouent sur une base vide. Le test de migration a également mis à niveau **des copies** des bases locales : smoke 40→49, real 41→49 et 2026-2027 48→49, avec conservation des données préexistantes. Les tests couvrent aussi les transitions 44 à 49 une par une. Les migrations déjà présentes n'ont pas été modifiées. Le serveur migre au démarrage, d'où l'obligation de sauvegarder avant le premier lancement de chaque environnement.

Les dépendances nécessaires à la répétition locale sont disponibles : Go/CGO et OpenCV, Goose, Typst, SQLite, Poppler (`pdftotext`, `pdfinfo`, `pdftoppm`, `pdfseparate`, `pdfunite`), OpenSSL et Chromium. `sqlc` est présent pour régénérer les requêtes, mais n'est pas nécessaire à l'exécution du binaire. Le serveur utilise des chemins relatifs à son répertoire courant : les scripts de lancement se placent dans `runtime/<environnement>`, lient `db/data/app.db` à `testdata/<environnement>/app.db`, et lient `internal` pour les templates. Le serveur écoute sur 8080.

La CI installe OpenCV et Goose et lance `scripts/check.sh` ainsi que le race detector. L'audit a révélé que ses deux jobs omettaient Typst alors qu'un test P5 en dépend obligatoirement : une exécution locale avec Typst retiré du `PATH` a reproduit un échec de publication HTTP 422. La CI a été corrigée pour installer Typst 0.15.1 (version testée ici) et Poppler dans les deux jobs. Chromium reste facultatif et absent de la CI ; le parcours navigateur a été exécuté localement.

## Parcours validés

| Zone | Vérification et résultat | Limite |
| --- | --- | --- |
| Génération | Test réel de génération, variante, image, QR techniques, snapshot immuable, PDF multipage et matrices de pagination. PDF physique-chimie inspecté visuellement : `rho=m/V`, `E_c`, `60 "mL"`, racine, unités, accents, cases QCM et nom composé visibles. | Impression papier non faite. |
| Correction et revue | Scénarios d'import, résultat cumulé, correction automatique et décision humaine, PDF corrigé, revue confirmée/modifiée et exclusion des revues ouvertes réussis. | Pas de scan matériel neuf pendant C1 ; les tests sur corpus privé historique restent conditionnels aux variables `LAZYMARKING_TEST_*`. |
| Rattrapage | Fixture avec plusieurs élèves et une absence : l'import tardif ajoute la copie, ne remplace pas les autres corrections valides, actualise P3 et ouvre une fenêtre P4 propre à l'absent. | Refaire avec les vrais scans des deux classes. |
| P3 | Taux et familles calculés sur les résultats courants, variantes regroupées, revue ouverte exclue, propriétaire vérifié. | Le seuil pédagogique n'a pas été modifié. |
| P4 | Coupon et QR réellement produits, code personnel, cookie limité au token, PDF individuel annoté, absence d'accès au PDF collectif ou à une autre copie, expiration, révocation/réouverture et rattrapage testés. La page publique de saisie du code a été ouverte et inspectée dans Chromium en fenêtre étroite ; elle ne montre pas d'identité avant code. | La saisie complète du code et le téléchargement sur un **autre téléphone physique** restent à refaire avec l'URL de classe. |
| P4.5 | Les formulaires, l'aide et le parseur ont été audités. Des tests HTTP refusent la syntaxe dangereuse ou invalide ; un vrai PDF Typst avec formules a été compilé et inspecté. | La prise en main réelle de la barre d'outils par un professeur reste un point terrain. |
| P5 | Création depuis une génération propriétaire, présélection P3, ajout/retrait/ordre, variantes, snapshot après changement de banque, états, token/QR, QCM simple et multiple, correction exacte des ensembles, score local, recommencement, confidentialité, isolation et CSRF passent. Chromium a suivi trois réponses jusqu'à `3 / 3` et recommencé ; à 500 px CSS, aucun débordement horizontal. | Chromium headless borne la fenêtre demandée de 390 px à 500 px ; test sur vrai téléphone restant. Aucun stockage de réponse individuelle n'a été observé. |
| Redémarrage | Serveur réel lancé deux fois sur une **copie isolée** de smoke, migration 40→49, deck public retrouvé à la même URL après redémarrage, intégrité SQLite `ok`. | Ce scénario insère une carte de test en base après migration ; il complète les tests HTTP du parcours professeur, sans les remplacer. |

La répétition transversale est constituée de fixtures complémentaires : génération PDF et snapshots ; correction/revue/rattrapage/P3/P4 ; P3→P5 et navigateur. Elles couvrent les cas demandés (plusieurs élèves, absence, variante, QCM simple/multiple, Typst), mais **aucun unique test automatisé ne simule un scan physique puis l'intégralité de la chaîne jusqu'au téléphone**. Cette dernière répétition est l'action pré-test décrite ci-dessous.

## Problèmes trouvés

| Sévérité | Zone | Problème | Correction / décision | État |
| --- | --- | --- | --- | --- |
| **BLOCKER pré-test** | Réseau public | `.env` pointe encore vers une URL HTTP locale ; un QR `localhost` ne marche pas depuis un téléphone. P4/P5 exigent HTTPS pour toute origine non locale. | Configurer une origine HTTPS accessible aux élèves, régler `APP_BASE_URL` sur cette origine, vérifier QR et accès P4/P5 depuis le réseau utilisé en classe. Aucun déploiement externe n'a été lancé. | **À faire avant classe** |
| **HIGH corrigé** | Scripts / P4-P5 | `run-smoke.sh` et `run-real.sh` ne liaient pas `.env` et ne créaient pas `assets/tmp` ; les trois scripts forçaient `SESSION_SECURE=false`. | Liens et répertoire ajoutés ; `SESSION_SECURE` suit maintenant le schéma de `APP_BASE_URL`. Les trois scripts ont été vérifiés en HTTP et HTTPS dans des racines factices, puis `bash -n`. | Corrigé |
| **HIGH pré-test** | Comptes | La version précédente du README contenait des identifiants de démonstration en clair. | Retirés du README de travail. Avant toute exposition d'une base contenant ces comptes, réinitialiser leurs mots de passe ou utiliser une base sans ces comptes ; l'historique Git peut encore porter l'ancien texte. | **À faire avant exposition** |
| **MEDIUM** | Sauvegarde | `runtime` contient des liens vers une base située sous `testdata` ; copier seulement `runtime` manquerait SQLite. `assets/tmp` conserve aussi les références et scans utiles à P4. | Procédure explicite ci-dessous et README. Répétition d'une sauvegarde SQLite cohérente + images/tmp sur copie smoke : intégrité `ok`. | Documenté |
| **HIGH corrigé** | CI | Sans Typst, le test P5 échoue réellement à la publication ; la CI n'installait pas ce prérequis. | Installer Typst 0.15.1 et Poppler dans les jobs check et race. Échec reproduit localement sans Typst ; la CI distante n'a pas encore tourné avec ce changement. | Correctif ajouté |
| **MEDIUM** | Couverture navigateur | Chromium est présent ici mais absent de la CI ; son test s'y saute. | Garder une répétition navigateur locale avant les deux évaluations ; envisager Chromium dans la CI après le terrain. | Documenté |
| **LOW** | Tests Typst | `LAZYMARKING_LAYOUT_ARTIFACTS` placé dans `/tmp` échoue parce que Typst limite l'accès au projet par sa racine. | Relance réussie avec `runtime/diagnostics/c1-layout` (chemin interne au dépôt). | Documenté |
| **LOW / observation** | UX | Libellés des statuts de correction, bibliothèque de questions, boutons monter/descendre, encadrés Typst. | Observer avec les professeurs ; pas de retouche graphique sans blocage constaté. | Après terrain |

Aucun panic ni erreur ignorée manifestement bloquante n'a été rencontré sur les parcours joués. La recherche ciblée a trouvé `applyHybridPolicy` avec un `panic` sur violation de constante interne de politique ; les chemins externes utilisent la version qui retourne une erreur. Quelques écritures HTTP et fermetures de fichiers ignorent volontairement une erreur après rendu ; aucun cas plausible de perte de résultat n'a été démontré. Pas de refactor global.

## Configuration et données avant le test

- Garder `SESSION_KEY` aléatoire (au moins 32 caractères), privée et **stable** dans `.env`. Une rotation coupe les sessions/cookies P4 en cours et empêche de réimprimer les anciens codes à l'identique ; les codes imprimés restent vérifiables par leur hash en base. Les scripts créent une clé CSRF distincte à chaque démarrage : recharger les formulaires déjà ouverts après redémarrage.
- Régler `APP_BASE_URL=https://<origine réellement joignable>` sans chemin de proxy inattendu ; ouvrir **la même origine HTTPS** côté professeur, car `SESSION_SECURE=true` sur ces scripts. Vérifier le certificat, le réseau des élèves et les QR sur un téléphone hors session professeur. Une URL locale ne suffit qu'aux essais sur la machine.
- Prévoir de l'espace : `assets/tmp` occupe environ 2,3 Go pour real, 242 Mo pour 2026-2027 et 26 Mo pour smoke lors de cet audit ; 358 Go étaient libres sur la machine. Ne pas purger `assets/tmp` globalement entre génération, correction et restitution.
- Sauvegarder ensemble `testdata/<env>/app.db`, `runtime/<env>/assets/images`, `runtime/<env>/assets/tmp`, `.env` et, si disponibles, les scans originaux. Les résultats P3/P5 et les accès P4/P5 sont dans SQLite ; les PDF de copie P4 exigent aussi les références/pages sous `assets/tmp`. La simple copie de `runtime/<env>` **ne suffit pas** car `app.db` y est un lien symbolique.

Exemple simple **serveur arrêté**, avec `env_name=2026-2027` et `dest` sur un support privé différent :

```sh
env_name=2026-2027
dest=/chemin/prive/sauvegarde-lazymarking
install -d -m 700 "$dest"
cp -a "testdata/$env_name/app.db" "$dest/app.db"
cp -a "runtime/$env_name/assets" "$dest/assets"
cp -a .env "$dest/.env"
chmod 600 "$dest/.env"
sqlite3 "$dest/app.db" 'PRAGMA integrity_check;'
```

La sortie doit être `ok`. Copier également les scans originaux s'ils sont conservés hors de ces chemins. Tester une restauration sur une copie avant de dépendre de la sauvegarde pour les vraies classes.

## Checklist terrain courte

### Avant la 6e

- [ ] Sauvegarde et restauration testées ; place disque et imprimante/scanner vérifiés.
- [ ] Base et liste d'élèves de **la bonne classe** vérifiées, avec accents, homonymes, absent possible et ordre d'impression.
- [ ] Une copie témoin imprimée contrôlée : QR techniques, cases, pagination et consignes.
- [ ] Serveur relancé après sauvegarde ; compte professeur, `SESSION_KEY` et URL HTTPS stable vérifiés.
- [ ] Un coupon P4 témoin et un deck P5 témoin ouverts depuis un téléphone indépendant ; code P4, PDF, QR, maths, réponse P5 et score vérifiés.

### Avant la seconde

- [ ] Même contrôles de sauvegarde, réseau et scanner, sur la **bonne génération** et la bonne classe.
- [ ] PDF témoin avec `$rho = m / V$`, `$E_c = 1/2 m v^2$` et `$V = 60 "mL"$` inspecté avant impression.
- [ ] QCM à plusieurs réponses et variante présents dans un aperçu ; consigne commune et barème relus.
- [ ] Un élève absent simulé : premier lot corrigé, puis import tardif ; P3 et fenêtre P4 de cette copie vérifiés sans altérer les autres.
- [ ] Terminer toutes les revues avant publication des copies/decks et conserver les scans originaux.

Pendant les deux séances, noter le temps et les hésitations sur l'import, la revue, les statuts « corrigé / partiel / non corrigé », les boutons de réordonnancement QCM, la navigation dans la bibliothèque de questions, la lisibilité des encadrés Typst et l'usage mobile P4/P5. Ces points se tranchent mieux avec les enseignants et les élèves qu'avec un changement préventif de l'interface.

## Feature freeze

Jusqu'aux tests : accepter les corrections de bug, sécurité, données, messages bloquants, fiabilité et infrastructure indispensable. Reporter les nouvelles fonctionnalités, le suivi élève, les statistiques individuelles, les refactors esthétiques, les changements graphiques majeurs et la containerisation tant qu'aucun besoin de déploiement précis n'est démontré.

## Commandes et résultats

| Commande / répétition | Résultat |
| --- | --- |
| `go test ./...` | Réussi. |
| `go test -count=1 ./...` | Réussi, sans cache. |
| `go test -race ./...` | Réussi. |
| `git diff --check` | Réussi. |
| `./scripts/check.sh` | Réussi : `go mod verify`, Goose 0001→0049, gofmt, `go vet ./...`, tests, build et diff. |
| Tests ciblés `internal/db` | Base vierge, transitions 44→49 et copies smoke/real/2026-2027 : réussis. |
| Tests ciblés `internal/handlers/tools` | Génération/PDF/Typst, coupons et copie individuelle : réussis ; PDF de physique inspecté visuellement. |
| Tests ciblés `internal/handlers/marking` | Correction cumulative, revue, rattrapage/P3, accès P4 et séparation des copies : réussis. |
| Tests ciblés `internal/handlers/training` | P5 et Chromium, dont fenêtre étroite sans débordement : réussis. |
| `go build -o /tmp/lazymarking-c1-server ./cmd/server` | Réussi ; serveur démarré deux fois sur base copiée, accès P5 inchangé. |
| `bash -n scripts/run-*.sh` et simulation HTTP/HTTPS | Réussis ; `.env` et cookie Secure cohérents. |
| `PATH` sans Typst, test P5 ciblé | Échec HTTP 422 reproduit ; a motivé l'ajout de Typst/Poppler aux deux jobs CI. |
| Sauvegarde SQLite + images/tmp sur copie smoke | `PRAGMA integrity_check=ok`. |

Le premier essai du test d'artefacts PDF avec un répertoire `/tmp` extérieur à la racine Typst a échoué ; la même vérification a réussi sous `runtime/diagnostics/c1-layout`. Le premier essai de redémarrage utilisait une clé CSRF de test de longueur incorrecte ; la validation du serveur l'a rejetée comme prévu, puis l'essai corrigé a réussi.

Principaux fichiers touchés dans C1 : `scripts/run-smoke.sh`, `scripts/run-real.sh`, `scripts/run-2026-2027.sh`, `.github/workflows/ci.yml`, `README.md`, `internal/handlers/training/handlers_test.go` et ce rapport. Aucun modèle métier ni migration n'a été changé dans C1.

## Conclusion

**READY WITH CONDITIONS**

Le logiciel et ses tests locaux sont suffisamment stables pour une répétition en conditions proches de la classe. Avant d'exposer les vraies évaluations, il reste **trois actions concrètes** : (1) fournir une URL HTTPS réellement accessible et vérifier P4/P5 depuis le réseau des élèves ; (2) sauvegarder puis restaurer un environnement complet avec SQLite, images, références/scans et clé ; (3) supprimer ou changer les comptes dont les anciens identifiants de démonstration étaient documentés. La répétition physique impression → scan → correction → téléphone doit ensuite être faite sur quelques copies témoin, dont une absence/rattrapage, avant la première séance.
