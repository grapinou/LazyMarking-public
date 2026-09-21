# P5.1 — Garde de migration au démarrage

Date : 21 septembre 2026.

## Problème constaté

P5 a ajouté la colonne `questions.instruction` avec la migration 0045. Le serveur a ensuite été lancé sur la base `2026-2027` encore en version 44. Le processus démarrait normalement, mais les premières pages qui lisaient les questions échouaient avec `no such column: instruction`.

La migration 0045 était correcte. Le défaut se trouvait dans le démarrage : le code supposait que la base avait déjà été migrée.

## Comportement historique

`cmd/server` appelait uniquement `db.InitDB`. Cette fonction ouvrait SQLite, activait les clés étrangères et exécutait un `Ping`. Elle ne lisait pas `goose_db_version` et n'appliquait aucune migration.

Les trois scripts `scripts/run-smoke.sh`, `scripts/run-real.sh` et `scripts/run-2026-2027.sh` :

1. créaient les liens vers la base et les ressources du runtime ;
2. construisaient le serveur ;
3. lançaient directement le binaire.

Aucun des trois n'appelait Goose. Seul `runtime/smoke` possédait un lien local vers le dossier des migrations, sans que son script l'utilise.

Les migrations étaient appliquées explicitement dans trois contextes séparés :

- la commande manuelle documentée dans le README ;
- le script de réinitialisation locale `reset.sh` ;
- les validations `scripts/check-migrations.sh` et `scripts/check.sh`.

Une installation neuve dépendait donc elle aussi d'une invocation manuelle du CLI Goose avant le premier démarrage. Le serveur ne connaissait pas la version attendue du schéma.

## Solution retenue

Les fichiers SQL versionnés de `db/migrations` sont maintenant embarqués dans le binaire avec `embed.FS`. Le serveur utilise la bibliothèque Go officielle `github.com/pressly/goose/v3`, avec le dialecte SQLite et le registre global de migrations Go désactivé.

Le chemin de démarrage devient :

```text
ouvrir SQLite et vérifier la connexion
→ construire le provider Goose depuis les migrations embarquées
→ lire la version courante et la version cible
→ refuser une base plus récente que le binaire
→ appliquer uniquement les migrations up en attente
→ vérifier que la version finale égale la version cible
→ récupérer les jobs, créer les routes et démarrer HTTP
```

Cette approche a été préférée à un appel du CLI :

- le binaire ne dépend ni d'un exécutable Goose installé, ni du répertoire courant ;
- les trois runtimes exécutent exactement le même code ;
- une base neuve et une base existante suivent le même chemin ;
- le schéma est préparé avant toute requête HTTP ou tâche de récupération ;
- les migrations exécutées sont exactement celles compilées avec cette version du serveur.

Le CLI Goose reste utile pour le développement et le rejeu de validation, mais il n'est plus requis pour lancer normalement LazyMarking.

## Migration réussie

`db.OpenMigratedDB` renvoie une connexion seulement après une migration et une vérification réussies. Son rapport contient la version initiale, la version cible et les versions appliquées.

Le serveur journalise par exemple :

```text
Database migrations: version 44 -> 45 (1 applied)
```

Une base déjà à jour produit `45 -> 45 (0 applied)`. Une base vide reçoit les migrations 1 à 45 avant le démarrage HTTP.

Aucun `down`, aucune réinitialisation et aucune suppression ad hoc ne sont exécutés par ce chemin. Les migrations continuent à bénéficier des transactions définies par Goose et par chaque fichier SQL.

## Échec de migration

Si la collecte, la lecture de version, une migration ou la vérification finale échoue :

- la connexion est fermée ;
- aucune connexion utilisable n'est rendue à `cmd/server` ;
- le serveur termine avec `Failed to prepare database schema` ;
- la récupération des jobs, les routes et `ListenAndServe` ne sont jamais atteints.

Goose valide et exécute chaque migration séquentiellement. Une migration SQL transactionnelle qui échoue est annulée. Des migrations antérieures déjà réussies peuvent rester appliquées, ce qui évite tout `down` automatique ; le prochain démarrage reprend depuis la version enregistrée après correction de la cause.

Une base dont la version est supérieure à celle embarquée est également refusée, afin qu'un ancien binaire ne serve pas un schéma qu'il ne connaît pas.

## Impact sur les scripts `run-*`

Les trois scripts restent inchangés. Ils continuent à construire le serveur puis à le lancer dans leur runtime respectif. La garde est volontairement dans l'application, ce qui couvre aussi un lancement direct du binaire et évite trois implémentations shell copiées.

Les liens `runtime/*/internal` et `runtime/smoke/migrations` ne participent pas à la migration au démarrage. Les migrations sont dans le binaire.

## Tests

Les tests de `internal/db/migrations_test.go` couvrent :

- base neuve : création et montée 0 → 45 ;
- base déjà à jour : aucune migration rejouée ;
- base synthétique en version 44 : application de 0045 et conservation d'une question existante ;
- migration volontairement invalide : aucune connexion rendue au serveur, arrêt à la dernière version réussie, transaction fautive annulée et donnée antérieure conservée ;
- copies des bases `smoke`, `real` et `2026-2027` : migration vers 45, intégrité SQLite, clés étrangères et contenu historique stable ;
- copie `2026-2027` ramenée à la version 44 uniquement dans le répertoire temporaire du test, puis remontée automatiquement à 45 ;
- empreinte de chaque base source vérifiée avant et après le test.

Les tests de copies lisent les fichiers source sous `testdata` et écrivent exclusivement dans `t.TempDir`. S'ils ne sont pas disponibles dans un autre environnement de test, ces trois sous-tests locaux sont ignorés ; les fixtures synthétiques couvrent toujours les contrats de démarrage.

Validation finale exécutée le 21 septembre 2026 :

- `go test ./...` : réussi ;
- `go test -count=1 ./cmd/server ./internal/db` : réussi ;
- `go test -v ./internal/db -run 'TestOpenMigratedDB' -count=1` : réussi, avec les trois sous-tests de runtimes exécutés ;
- `go mod verify` : réussi (`all modules verified`) ;
- `git diff --check` : réussi.

## Validation des runtimes

| Runtime | Version de la copie avant garde | Version après garde | Données stables | Intégrité / clés étrangères |
|---|---:|---:|---|---|
| `runtime/smoke` | 40 | 45 | oui | OK / OK |
| `runtime/real` | 41 | 45 | oui | OK / OK |
| `runtime/2026-2027` | 44, préparée sur la copie | 45 | oui | OK / OK |

La base source `2026-2027` était déjà à 45 lors de cette validation. Le test a exécuté le `down` 45 → 44 uniquement sur sa copie temporaire afin de reproduire précisément l'incident, puis a validé la montée automatique 44 → 45. Le code de production n'appelle jamais `Down` ou `DownTo`.

## Fichiers concernés

- `db/migrations/embed.go` : migrations SQL embarquées ;
- `internal/db/migrations.go` : ouverture, montée et vérification du schéma ;
- `internal/db/migrations_test.go` : contrats et copies des trois runtimes ;
- `cmd/server/main.go` : garde avant toute initialisation métier ou HTTP ;
- `go.mod` et `go.sum` : bibliothèque Goose ;
- `docs/reports/2026-09-21-runtime-migration-guard.md` : présent rapport.

Les migrations 0001 à 0045, les scripts de lancement, les données fonctionnelles et le code P5 de génération n'ont pas été modifiés.
