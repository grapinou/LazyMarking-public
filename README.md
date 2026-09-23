# LazyMarking

## Prérequis

- Go 1.25 ou plus récent avec CGO ; OpenCV (`libopencv-dev` et `opencv4.pc`) pour GoCV.
- Typst, SQLite, Goose, sqlc pour régénérer les requêtes SQL.
- Poppler : `pdfseparate`, `pdftoppm`, `pdfunite`, `pdfinfo` et `pdftotext` (paquet `poppler-utils` sur Debian/Ubuntu).
- `openssl` pour les scripts de lancement et une clé CSRF distincte.

Le contrôle commun est `./scripts/check.sh` ; il rejoue toutes les migrations sur une base vierge, puis exécute `go vet`, les tests et le build. `go test -race ./...` reste une vérification séparée.

## Configuration

Créer `.env` à la racine du dépôt, hors Git :

```dotenv
SESSION_KEY=<valeur aléatoire stable d'au moins 32 caractères>
APP_BASE_URL=http://localhost:8080
# SMTP_* seulement si la réinitialisation de mot de passe est utilisée.
```

Générer `SESSION_KEY` avec `openssl rand -hex 32`, puis conserver cette valeur entre les redémarrages et avec les sauvegardes. Une rotation déconnecte les sessions P4 en cours et empêche la réimpression à l'identique des anciens coupons ; les codes déjà imprimés restent vérifiables avec leur hash conservé en base.

Pour les liens et QR P4/P5, `APP_BASE_URL` doit être l'origine effectivement accessible aux élèves. Seul `localhost` peut utiliser HTTP ; une origine publique doit être en HTTPS. `localhost` dans un QR ne fonctionne pas sur le téléphone d'un élève. Les scripts ci-dessous déterminent `SESSION_SECURE` à partir du schéma de cette URL : utiliser le même accès HTTPS côté professeur lorsque la valeur est `true`.

Pour lancer directement depuis la racine, fournir également `SESSION_SECURE` (`false` en HTTP local, `true` en HTTPS) et une `CSRF_AUTH_KEY` distincte de 32 octets, par exemple `openssl rand -hex 16`.

## Environnements locaux

Les données d'essai sont dans `testdata/` et les fichiers d'exécution dans `runtime/` ; ces dossiers sont exclus de Git. Les scripts construisent le serveur, préparent les liens vers la base et `.env`, puis démarrent depuis le bon répertoire :

```sh
./scripts/run-smoke.sh
./scripts/run-real.sh
./scripts/run-2026-2027.sh
```

Le serveur écoute sur le port 8080. Arrêter l'instance active avant d'en lancer une autre. Les scripts appliquent automatiquement les migrations au démarrage ; faire une sauvegarde avant de les utiliser avec de vraies données. Ils génèrent une nouvelle clé CSRF à chaque lancement : un formulaire déjà ouvert doit être rechargé après un redémarrage.

## Données à conserver

Une évaluation ne tient pas dans le seul fichier SQLite. Pour chaque environnement, conserver ensemble :

- `testdata/<environnement>/app.db` (ou une sauvegarde SQLite cohérente) ;
- `runtime/<environnement>/assets/images` (images de questions) ;
- `runtime/<environnement>/assets/tmp`, en particulier les références PNG `exam-*` et les scans alignés `marking-*` nécessaires aux corrections et à P4 ;
- `.env`, protégé comme un secret ;
- les scans originaux conservés sous `testdata/<environnement>/scans` lorsqu'ils existent.

Arrêter le serveur avant de copier ces éléments dans un répertoire privé. Vérifier la sauvegarde avec `sqlite3 <copie>/app.db 'PRAGMA integrity_check;'`. Le dossier `assets/tmp` contient aussi des fichiers provisoires ; il ne faut donc pas l'effacer globalement entre génération et correction.

## Validation et diagnostics

```sh
./scripts/check.sh
go test -count=1 ./...
go test -race ./...
```

Les tests sur de vrais scans historiques utilisent des variables `LAZYMARKING_TEST_*` et sont volontairement ignorés en leur absence. Les chemins et identifiants de ces corpus restent locaux. Le peuplement de démonstration peut être lancé séparément avec `go run ./cmd/workflow`.

Voir [le rapport de consolidation pré-V1](docs/reports/2026-09-23-pre-v1-consolidation-classroom-readiness.md) pour la checklist du test en classe.
