# LazyMarking

## Mémo de commande

- go build -o app ./cmd && ./app

- git add cmd/ internal/ templates/ go.mod README.md .gitignore

- git status

- git commit -m ""

- git push

Avec le go.sum quand il sera présent :

- git add cmd/ internal/ templates/ db/ go.mod go.sum README.md .gitignore

## Mémo goose :

- goose -dir db/migrations sqlite3 db/data/app.db up (depuis root projet)

- sqlc generate depuis le dosser db

## Mémo go install

go get github.com/mattn/go-sqlite3

go mod tidy
