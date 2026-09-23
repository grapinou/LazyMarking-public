# C1.1 — Finalisation de l'état pré-test

23 septembre 2026.

## État initial et contenu figé

La branche `main` était à `c1e19e8`, alignée avec `origin/main`, avec des modifications suivies et des ajouts non suivis issus de P4, P4.5, P5 et C1. L'inventaire a identifié :

- P4 : migration 0048, accès individuels temporaires, coupons/QR, PDF individuel, routes et tests ;
- P4.5 : parseur `internal/mathcontent`, validation et rendu Typst, formulaires, aperçus et tests ;
- P5 : migration 0049, requêtes sqlc, decks, rendu mathématique, templates professeur/élève et tests ;
- C1 : scripts de lancement, prérequis CI, README et rapport de consolidation.

Les migrations existantes n'ont pas été modifiées pendant C1.1. Une régénération sqlc dans un répertoire temporaire a produit **35 fichiers identiques** aux sources générées présentes. Les templates et les rapports sont des fichiers texte. Un test QR P5 intermittent a été stabilisé en décodant le PNG généré en mode QR pur : le décodeur de scans manquait parfois un code pourtant lisible. Le QR produit pour les utilisateurs et les workflows métier n'ont pas changé.

`reset.sh`, ancien script local de réinitialisation sans lien avec cette séquence, est resté à sa place et a été exclu **localement** via `.git/info/exclude` ; il n'est pas dans le commit. Aucun fichier temporaire ou artefact de test n'a été ajouté.

## Contrôles de confidentialité

L'index du commit de référence contenait 63 fichiers, tous textuels. Aucun `.env`, fichier runtime, base SQLite, scan, PDF de copie ou image de données réelles n'y figurait. `testdata/`, `runtime/` et `.env` restent ignorés. Le README courant et les fichiers candidats ne contiennent plus les anciens mots de passe de démonstration documentés auparavant. Leur présence dans l'historique Git ancien n'a pas été réécrite : **ces mots de passe doivent être considérés comme compromis et changés avant exposition publique**.

## Validation finale

| Commande / contrôle | Résultat |
| --- | --- |
| `go test ./...` | Réussi |
| `go test -count=1 ./...` | Réussi après correction du test QR intermittent |
| `go test -race ./...` | Réussi |
| `git diff --check` et contrôle de l'index | Réussis |
| `./scripts/check.sh` | Réussi : modules, migrations 0001→0049, formatage, `go vet`, tests, build et diff |
| `go test -count=20 ./internal/handlers/training -run TestCollectiveTrainingFlow` | Réussi, 20 répétitions |
| Régénération sqlc hors du dépôt | 35 fichiers identiques |
| Audit des chemins et secrets de l'index | Aucun fichier ou ancien mot de passe interdit |

Le premier passage sans cache a exposé l'intermittence du décodeur QR du test ; le passage final, après le correctif limité au test, est vert. Aucun changement fonctionnel n'a été introduit.

## Commit et publication

- Commit de référence : `11451fcf896f08f456649a5e1fd0d54757854ea2`
- Message : `Complete pre-V1 classroom readiness`
- Branche et remote : `main` → `origin/main` (`https://github.com/grapinou/LazyMarking.git`)
- Push du commit de référence : réussi (`c1e19e8..11451fc`).
- Ce rapport est ajouté dans un second commit documentaire afin de pouvoir citer le SHA exact du commit de référence sans auto-référence.
- État Git final vérifié après publication du rapport : `git status --short` vide ; `main` alignée avec `origin/main`.
- Volontairement non versionnés : `.env`, bases et corpus de `testdata/`, `runtime/` (images, scans, PDF, diagnostics), et `reset.sh`.

NEXT PRE-TEST ACTIONS

1. Configurer une URL HTTPS accessible aux élèves.
2. Changer les anciens mots de passe de démonstration.
3. Répéter sauvegarde + restauration.
4. Faire une répétition physique impression → scan → correction → P4/P5 sur téléphone.
