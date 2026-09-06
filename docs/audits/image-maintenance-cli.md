# Commande de maintenance du stockage des images

## 1. Syntaxe

Audit et simulation non destructive :

```text
go run ./cmd/maintenance images
```

Purge explicite des vieux fichiers orphelins :

```text
go run ./cmd/maintenance images --execute
```

Le délai de grâce peut être ajusté avec une durée Go, par exemple `--grace 48h`. Une durée négative est refusée avant le scan et avant toute suppression.

## 2. Dry-run

Sans `--execute`, la commande affiche explicitement `DRY RUN`. Elle exécute `imagestorage.Scan`, puis `imagestorage.PurgeOrphans` avec `Execute: false`. Aucun fichier et aucune ligne DB ne sont modifiés.

La simulation distingue les candidats suffisamment anciens, les fichiers récents protégés par le délai de grâce et les noms redevenus référencés lors de la revérification DB.

## 3. Mode `--execute`

La suppression physique n'est autorisée que lorsque `--execute` est présent. La commande appelle alors `imagestorage.PurgeOrphans` avec `Execute: true` et affiche les fichiers supprimés, récents, redevenus référencés et les échecs individuels.

Un échec individuel est affiché sans masquer les autres résultats. La commande retourne néanmoins un code non nul afin que l'échec soit observable par un script d'exploitation.

## 4. Délai de grâce

Le défaut est `imagestorage.DefaultOrphanGracePeriod`, soit 24 heures. L'option `--grace` utilise le parseur de durées de la bibliothèque standard. L'horloge de la petite logique de commande est remplaçable dans les tests, sans interface générale.

## 5. Sortie Orphans

La sortie donne le nombre d'orphelins et affiche leurs noms. Le résultat de purge précise ensuite lesquels sont candidats, supprimés, récents ou redevenus référencés.

## 6. Sortie Missing

Chaque référence DB sans fichier est affichée avec son type (`main_image`, `variant_image` ou les deux). La sortie précise qu'il s'agit uniquement d'un signalement et qu'aucune ligne DB n'est modifiée. La commande ne possède aucune option de suppression ou de réparation de `Missing`.

## 7. Sortie Unsafe

Chaque entrée unsafe est affichée avec son nom, sa source et sa nature. Ces entrées restent uniquement signalées : elles ne sont jamais transmises comme candidats à la suppression et aucune option ne permet de contourner cette protection.

## 8. Erreurs et codes de sortie

- `0` : audit/simulation ou purge exécuté correctement, même si des incohérences ont été signalées ;
- `1` : syntaxe invalide, ouverture DB impossible, scan incomplet, storage tree invalide, purge globale impossible ou au moins un échec individuel de suppression.

Si le scan initial échoue, `PurgeOrphans` n'est pas appelé. La validation de `Scan` refuse notamment une racine de stockage symbolique ; aucune suppression extérieure ne peut alors être déclenchée.

## 9. Accès DB réutilisé

Le chemin SQLite historique du serveur, `./db/data/app.db`, est maintenant exposé par `config.DatabasePath`. Le serveur et la commande utilisent tous deux `db.InitDB(config.DatabasePath)`. Cette centralisation ne change ni le chemin ni les options SQLite utilisés par le serveur.

## 10. Tests ajoutés

Sept tests ciblés couvrent :

1. dry-run conservant un vieil orphelin ;
2. `--execute` supprimant un vieil orphelin ;
3. conservation d'un orphelin récent ;
4. affichage de `Missing` sans mutation DB ;
5. affichage d'un dossier unsafe sans suppression ;
6. refus d'un délai de grâce négatif avant toute action ;
7. refus d'une racine de stockage symbolique, avec vérification que le fichier extérieur reste intact.

Les assertions portent sur les informations importantes et les effets, pas sur la mise en forme exacte de toute la sortie.

## 11. Absence de déclenchement automatique

Aucun handler, routeur, startup serveur, bouton ou tâche périodique n'appelle `Scan` ou `PurgeOrphans`. La purge reste uniquement accessible par l'invocation volontaire de cette commande séparée.

## 12. Invariants destructifs

La commande ne modifie aucune ligne DB. `Missing` et `Unsafe` ne peuvent jamais être supprimés. Les seuls candidats destructifs proviennent des `Orphans` validés par `imagestorage.PurgeOrphans`, avec délai de grâce, sécurité du storage tree et revérification DB.

## 13. Validations

- `go test ./...` : réussi.
- `go vet ./...` : réussi.
- `git diff --check` : réussi.

## 14. Fichiers de ce jalon

- `cmd/maintenance/main.go` : commande, parsing des options, sortie terminal et codes de retour ;
- `cmd/maintenance/main_test.go` : sept scénarios ciblés ;
- `internal/config/config.go` : chemin DB partagé sans changement de valeur ;
- `cmd/server/main.go` : utilisation de la constante partagée ;
- `docs/audits/image-maintenance-cli.md` : présent rapport.

Aucun SQL/sqlc, handler, template, primitive de stockage, routeur ou README n'a été modifié dans ce jalon.
