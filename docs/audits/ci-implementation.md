# Implémentation de la CI LazyMarking

## Audit de l'environnement

Le module déclare Go 1.25 dans `go.mod`. L'environnement local utilisé pour ce jalon exécute Go 1.26.5 sur Linux amd64.

Le projet dépend directement de `github.com/mattn/go-sqlite3`, qui utilise CGO. L'environnement local a `CGO_ENABLED=1` et GCC 13.3. Un essai ciblé avec `CGO_ENABLED=0 go test ./internal/db` échoue dès la compilation des tests SQLite ; CGO est donc explicitement activé dans les jobs CI.

`go test ./...` et `go build ./...` réussissent avec la toolchain locale standard. Aucun outil système supplémentaire n'a été nécessaire. Le workflow `ubuntu-latest` n'installe donc aucun paquet : il utilise le compilateur C standard du runner hébergé.

Aucun workflow GitHub Actions ni script commun de validation n'existait avant ce jalon. Le README documentait séparément `go test ./...`, `go vet ./...` et `go mod verify`; `reset.sh` construisait le serveur local et lançait sqlc, mais ce script local non suivi n'est pas une base CI reproductible.

## Script commun

Le fichier exécutable `scripts/check.sh` a été créé avec Bash et `set -euo pipefail`. Il se replace à la racine du dépôt, ce qui permet de le lancer simplement avec `./scripts/check.sh`.

Les étapes, dans l'ordre, sont :

1. `go mod verify` ;
2. vérification gofmt de tous les fichiers `.go`, sans modification automatique ;
3. `go vet ./...` ;
4. `go test ./...` ;
5. `go build ./...` ;
6. `git diff --check`.

Chaque étape affiche un en-tête lisible. La vérification gofmt utilise la liste des fichiers trouvés dans le dépôt en excluant seulement `.git`. Un fichier temporaire volontairement mal formaté a fait échouer le script avec le statut 1 et le chemin concerné ; ce fichier a ensuite été retiré sans laisser de modification.

## Workflow GitHub Actions

`.github/workflows/ci.yml` crée le workflow `CI` avec les déclencheurs :

- tous les `push` ;
- toutes les `pull_request` ;
- lancement manuel `workflow_dispatch`.

Les permissions sont limitées à `contents: read`. Aucun droit d'écriture n'est accordé. Une concurrence par workflow et référence annule les runs devenus inutiles après un nouveau push sur la même branche.

Le job principal `check` utilise `ubuntu-latest`, `actions/checkout@v6`, `actions/setup-go@v6`, `go-version-file: go.mod`, le cache intégré de setup-go et `CGO_ENABLED=1`. Il exécute uniquement `./scripts/check.sh`, sans recopier ses commandes. Son timeout est de 15 minutes.

La source de version Go en CI est donc le directive `go 1.25` du `go.mod`; setup-go résout la version corrective disponible correspondante.

## Race detector

`go test -race ./...` réussit localement en environ 10 secondes, sans course détectée. Il est retenu dans un job CI séparé `race`, également sous `ubuntu-latest`, Go issu de `go.mod`, cache activé, CGO activé et timeout de 15 minutes.

Le job race ne duplique aucune autre validation et peut s'exécuter en parallèle du job principal. Le race detector n'est pas intégré à `scripts/check.sh`, afin que la validation locale commune reste rapide.

## Reproductibilité, données et secrets

La CI ne référence ni `app`, ni `reset.sh`, ni une DB locale, ni un workspace personnel. Les commandes reposent uniquement sur le code du dépôt et les bases temporaires/fixtures synthétiques déjà créées par les tests.

Aucune ancienne base réelle, copie PDF scannée réelle ou donnée élève réelle n'a été utilisée ou ajoutée. Aucun GitHub Secret ni service externe n'est nécessaire.

## Validations

- `bash -n scripts/check.sh` : succès ;
- `./scripts/check.sh` : succès ;
- gofmt check négatif volontaire : succès, le script retourne 1 ;
- `go mod verify` : succès ;
- `go vet ./...` : succès ;
- `go test ./...` : succès ;
- `go build ./...` : succès ;
- `git diff --check` : succès ;
- `go test -race ./...` : succès, environ 10 secondes ;
- parsing YAML de `.github/workflows/ci.yml` avec le module Python déjà disponible : succès.

## Problèmes découverts

Aucun problème applicatif, échec de test, erreur vet, erreur build ou course de données n'a été découvert.

CGO désactivé n'est pas un environnement supporté pour la suite actuelle : les tests DB ne compilent pas, notamment `internal/db/referenceDeleteIntegrity_test.go`, car les types complets du driver SQLite ne sont pas disponibles. Cette contrainte est traitée par `CGO_ENABLED=1` dans la CI et ne nécessite aucun correctif métier.

## Fichiers modifiés par ce jalon

- `scripts/check.sh` ;
- `.github/workflows/ci.yml` ;
- `docs/audits/ci-implementation.md`.
