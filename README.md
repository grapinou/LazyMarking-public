# LazyMarking

## Prérequis

- Go 1.25
- SQLite et Goose pour les migrations
- Typst
- Poppler (`pdfseparate`, `pdftoppm` et `pdfunite`)
- OpenCV, requis par GoCV

Sous Debian/Ubuntu, Poppler s'installe avec `sudo apt install poppler-utils`.

## Configuration

Créer un fichier `.env` (non versionné) avec au minimum :

```dotenv
# Obligatoire, 32 caractères minimum. Exemple de génération : openssl rand -hex 32
SESSION_KEY=
# true lorsque l'application est servie en HTTPS
SESSION_SECURE=false
APP_BASE_URL=http://localhost:8080

# Requis pour la réinitialisation de mot de passe
SMTP_FROM=
SMTP_PASSWORD=
SMTP_HOST=
SMTP_PORT=
```

L'application refuse de démarrer avec une clé de session absente ou trop courte. `SESSION_SECURE` doit être explicitement défini ; utiliser `true` en HTTPS et réserver `false` au développement HTTP local.

## Base de données

```sh
goose -dir db/migrations sqlite3 db/data/app.db up
```

Pour régénérer les accès SQL après une modification des requêtes :

```sh
cd db
sqlc generate
```

## Exécution

Les environnements locaux de test utilisent les données de `testdata/` et les fichiers d'exécution de `runtime/`. Ces dossiers restent hors Git.

### Smoke test

```sh
./scripts/run-smoke.sh
```

Identifiant : `smoke-prof`
Mot de passe : `SmokeTest-2026!`

### Corpus réel 6e

```sh
./scripts/run-real.sh
```

Identifiant : `prof-6e-test`
Mot de passe : `SixiemeTest-2026!`

Le serveur écoute sur `http://localhost:8080`.

## Validation

Pour lancer l'ensemble des vérifications du projet :

```sh
./scripts/check.sh
```

Le scénario de peuplement local peut être lancé séparément :

```sh
go run cmd/workflow/main.go
```

### Corpus de PDF problématiques

```sh
export LAZYMARKING_TEST_DB="./testdata/problematic/app.db"
export LAZYMARKING_TEST_USER_ID="1"

export LAZYMARKING_TEST_PDF_1_PAGE="./testdata/problematic/63_1_page.pdf"
export LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE="728"

export LAZYMARKING_TEST_PDF_2_PAGES="./testdata/problematic/6e1_2_pages.pdf"
export LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES="14"

export LAZYMARKING_TEST_PDF="./testdata/problematic/51_3_pages.pdf"
export LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES="366"
```
