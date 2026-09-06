# Purge prudente des fichiers image orphelins

## 1. API choisie

La primitive se trouve dans `internal/imagestorage/purge.go` :

```go
func PurgeOrphans(ctx context.Context, queries *db.Queries, options PurgeOptions) (PurgeResult, error)
```

`PurgeOptions` contient :

- `Execute bool` : seule la valeur explicite `true` autorise une suppression ;
- `GracePeriod time.Duration` ;
- `Now time.Time` pour une horloge déterministe.

`DefaultPurgeOptions()` retourne une configuration non destructive avec un délai de grâce de 24 heures.

## 2. Dry-run

Le comportement par défaut est non destructif : la valeur zéro de `Execute` est `false`. En dry-run, le scanner, l’inspection filesystem et la revérification DB sont exécutés, mais `os.Remove` ne peut pas être appelé.

Les orphelins trouvés apparaissent dans `Candidates`; les candidats récents et ceux redevenus référencés sont classés séparément.

## 3. Délai de grâce

`DefaultOrphanGracePeriod` vaut 24 heures. Une durée différente, y compris zéro, peut être fournie explicitement.

La règle frontière est : suppression autorisée si `modtime <= now - gracePeriod`, donc si `age >= gracePeriod`. Un fichier strictement plus récent est ajouté à `SkippedRecent`.

Une durée négative est rejetée avant le scan.

## 4. Horloge et testabilité

Les tests fournissent une valeur fixe dans `PurgeOptions.Now` et positionnent les mtimes avec `os.Chtimes`. Si `Now` est nul en exploitation, la primitive utilise une seule valeur `time.Now()` capturée au début de l’opération.

Aucune interface Clock n’a été créée.

## 5. Source des candidats

La purge appelle exclusivement `imagestorage.Scan` et copie uniquement `Consistency.Orphans` dans `Candidates`.

- `Missing` n’est jamais utilisé pour supprimer ou modifier quoi que ce soit ;
- `Unsafe` n’est jamais candidat ;
- aucune seconde comparaison DB ↔ filesystem n’a été créée.

## 6. Revérification DB avant suppression

Une requête sqlc globale minimale a été ajoutée :

```sql
ImageNameIsReferenced(image_name)
```

Elle teste avec un seul `EXISTS`/`UNION ALL` si le nom figure dans `images` ou `alt_images`, sans filtre utilisateur.

Tous les candidats éligibles sont revérifiés avant la première suppression. En mode destructif, le même contrôle est répété immédiatement avant chaque `os.Remove`. Une erreur de la phase DB préalable annule l’opération avant toute suppression.

## 7. Protection TOCTOU

Un fichier vu comme orphelin par `Scan` mais référencé avant la phase destructive rejoint `SkippedReferenced` et reste présent. La revérification répétée réduit également la fenêtre entre préparation et suppression.

Comme pour toute coordination SQLite/filesystem sans transaction commune, une fenêtre infinitésimale subsiste entre le dernier SELECT et `os.Remove`; aucune atomicité distribuée n’est prétendue.

## 8. Symlinks, dossiers et noms non sûrs

Avant toute inspection et de nouveau avant suppression, le nom doit satisfaire `filesafety.IsSafePathComponent`.

Le chemin est construit uniquement avec `config.ImageSavePath + nom sûr`. `os.Lstat` est utilisé pour ne pas suivre les symlinks. Seul un fichier encore régulier est admissible. Le fichier est réinspecté juste avant suppression afin de détecter un changement intervenu après le scan.

Symlinks, dossiers, entrées spéciales et noms DB dangereux restent hors des candidats fournis par le scanner et ne sont jamais supprimés.

## Protection du storage root

Avant la correction, `os.ReadDir(config.ImageSavePath)` pouvait suivre un storage root symbolique. Le `os.Lstat` appliqué ensuite à `assets/images/nom` ne protégeait que le dernier composant : avec `assets/images -> /outside`, la purge pouvait donc inspecter puis supprimer `/outside/nom`.

Une primitive neutre `filesafety.ValidateDirectoryTree` inspecte désormais chaque composant du chemin avec `os.Lstat` et refuse tout symlink, fichier, entrée spéciale ou composant absent. Elle ne résout jamais les liens symboliques.

- `Scan` valide le storage root avant `os.ReadDir` et retourne une erreur sans résultat partiel si le root existe mais est non sûr. Son comportement historique pour un répertoire absent avec DB vide reste un résultat vide en lecture seule.
- `PurgeOrphans`, y compris en dry-run, exige toujours un storage root réel et existant avant même d’appeler `Scan`. Un root absent ou non sûr produit une erreur et aucune suppression.
- Le helper historique `ensureDirectoryTree` délègue à cette primitive canonique lorsqu’il valide un arbre existant sans création.

Les tests créent un fichier ancien dans un répertoire extérieur puis remplacent `assets/images` par un symlink vers ce répertoire. Scan et Purge retournent une erreur, la primitive de suppression n’est jamais appelée et le fichier extérieur reste intact.

## 9. Échec individuel

Un échec `os.Remove` ajoute `PurgeFailure{Name, Err}` à `Failed`, puis la boucle continue. Le résultat est trié par nom.

Le test `a.png / b.png / c.png` injecte un échec sur `b.png` et vérifie que `a.png` et `c.png` sont néanmoins supprimés.

## 10. Scan incomplet

Si `Scan` échoue, `PurgeOrphans` retourne immédiatement une erreur et un résultat vide. Aucun appel de suppression n’a lieu.

Une erreur pendant la phase initiale de revérification DB globale retourne également avant toute suppression. Une erreur d’une revérification finale locale empêche la suppression du seul fichier concerné et est enregistrée dans `Failed`.

## 11. Résultat structuré

`PurgeResult` contient cinq catégories déterministes :

- `Candidates []string` ;
- `Deleted []string` ;
- `SkippedRecent []string` ;
- `SkippedReferenced []string` ;
- `Failed []PurgeFailure`.

Les slices sont triées ; les catégories de skip sont dédupliquées. Chaque échec conserve le nom et l’erreur exploitable.

## 12. Tests ajoutés

Onze scénarios déterministes de purge sont couverts :

1. dry-run par défaut sur orphelin ancien ;
2. purge réelle au seuil exact du délai de grâce ;
3. orphelin récent conservé ;
4. fichier référencé jamais candidat ;
5. ligne DB sans fichier jamais modifiée ;
6. symlink, dossier et nom DB dangereux jamais supprimés ;
7. référence créée après le scan détectée ;
8. échec individuel avec poursuite sur les autres fichiers ;
9. échec du scan sans aucun appel de suppression ;
10. échec de revérification DB sans aucun appel de suppression.
11. storage root symbolique refusé sans toucher au fichier extérieur.

Le scanner possède en complément un scénario dédié vérifiant qu’un storage root symbolique retourne une erreur sans résultat partiel. Il compte désormais 12 scénarios ; purge et scanner totalisent donc 23 scénarios.

## 13. SQL/sqlc

SQL/sqlc a été modifié uniquement pour `ImageNameIsReferenced`. Aucune migration n’a été ajoutée.

```text
sqlc generate -f db/sqlc.yaml : réussi
```

Le diff généré ajoute seulement la méthode correspondante dans `internal/db/images.sql.go`.

## 14. Aucune ligne DB supprimée

La primitive n’exécute que les lectures de `Scan` et `ImageNameIsReferenced`. Elle ne contient aucun INSERT, UPDATE ou DELETE et ne tente jamais de réparer `Missing`.

## 15. Aucun déclenchement utilisateur

Aucune route, page, commande CLI, tâche périodique ou exécution au démarrage n’appelle la purge. `main.go`, les handlers et les templates sont inchangés.

## 16. Validations

- `go test ./...` : réussi.
- `go vet ./...` : réussi.
- `git diff --check` : réussi.
- `sqlc generate -f db/sqlc.yaml` : réussi.

## 17. Fichiers modifiés pour ce jalon

1. `db/query/images.sql`
2. `internal/db/images.sql.go`
3. `internal/imagestorage/purge.go`
4. `internal/imagestorage/purge_test.go`
5. `docs/audits/image-orphan-purge.md`
6. `internal/filesafety/directory_tree.go`
7. `internal/handlers/tools/safeDirectory.go`
8. `internal/imagestorage/scanner.go`
9. `internal/imagestorage/scanner_test.go`
