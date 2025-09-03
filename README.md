# LazyMarking

## dépendance à typst

- avoir typst installer car utilisation d'execution de commande typst en directe

- avoir poppler également d'installer : sudo apt install poppler-utils

## Mémo de commande

- go build -o app ./cmd/server && ./app

- git add cmd/ internal/ templates/ go.mod README.md .gitignore

- git status

- git commit -m ""

- git push

Avec le go.sum quand il sera présent :

- git add cmd/ internal/ db/ go.mod go.sum README.md .gitignore

## Mémo goose :

- goose -dir db/migrations sqlite3 db/data/app.db up (depuis root projet)

- sqlc generate depuis le dosser db

## Mémo go install

go get github.com/mattn/go-sqlite3

go mod tidy

## pour le workflow

go run cmd/workflow/main.go
