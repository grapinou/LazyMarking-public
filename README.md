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

```sh
go build -o app ./cmd/server
./app
```

Le serveur écoute sur `http://localhost:8080`.

## Validation

```sh
go test ./...
go vet ./...
go mod verify
```

Le scénario de peuplement local peut être lancé séparément :

```sh
go run cmd/workflow/main.go
```
