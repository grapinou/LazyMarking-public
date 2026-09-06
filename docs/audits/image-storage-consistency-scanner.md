# Scanner de cohérence DB ↔ filesystem des images

## 1. Emplacement choisi

Le scanner se trouve dans `internal/imagestorage/scanner.go`.

Un petit package dédié évite d’ajouter une responsabilité globale de maintenance aux handlers ou au package `internal/handlers/tools`. Il reste volontairement plat : un fichier d’implémentation et un fichier de tests, sans service générique ni couche repository.

L’unique point d’entrée est `imagestorage.Scan(ctx, queries)`. Il utilise `config.ImageSavePath` et ne reçoit donc aucun chemin provenant d’un utilisateur.

## 2. Structure du résultat

`Consistency` contient exactement trois catégories :

- `Orphans []string` : fichiers réguliers présents sans référence DB ;
- `Missing []MissingImage` : noms DB sûrs sans fichier régulier correspondant ;
- `Unsafe []UnsafeEntry` : noms DB non sûrs ou entrées filesystem anormales.

`MissingImage` conserve le nom et un `ReferenceType` : image principale, image de variante, ou nom référencé par les deux tables. Cela permet de dédupliquer un nom commun aux deux tables sans perdre son origine.

`UnsafeEntry` conserve le nom, la source (`database` ou `filesystem`) et la nature (`unsafe_name`, `symlink`, `directory`, `special_file`).

## 3. Sources DB utilisées

Deux requêtes sqlc globales et strictement en lecture seule ont été ajoutées :

- `ListAllImageNames` : `SELECT image_name FROM images ORDER BY image_name` ;
- `ListAllAltImageNames` : `SELECT image_name FROM alt_images ORDER BY image_name`.

Elles n’acceptent pas de `user_id`, car le scanner établit l’état global d’un stockage physique commun. Aucune requête métier existante n’a été modifiée.

## 4. Comportement filesystem

Le scanner appelle `os.ReadDir` uniquement sur `config.ImageSavePath`. Il ne descend jamais dans un sous-répertoire.

- Un fichier régulier dont le nom est sûr rejoint l’ensemble FS.
- Un nom filesystem non sûr rejoint `Unsafe` et n’est pas comparé.
- Si `DirEntry.Info` échoue et empêche de classifier une entrée, le scan entier retourne une erreur.
- Si le chemin est illisible ou n’est pas un répertoire, le scan entier retourne une erreur.

Si `ImageSavePath` est absent :

- DB sans aucune référence : résultat vide valide ;
- au moins une référence DB, sûre ou non : erreur d’infrastructure.

Ce choix distingue une installation encore vide d’un stockage disparu alors que la DB annonce des images.

## 5. Symlinks et dossiers

Les symlinks sont détectés via le type de l’entrée avant toute inspection de cible. Ils ne sont jamais suivis et sont ajoutés à `Unsafe` avec le type `symlink`.

Les dossiers sont ajoutés à `Unsafe` avec le type `directory` et ne sont jamais parcourus. Les autres types spéciaux sont ajoutés avec le type `special_file`.

## 6. Noms DB non sûrs

Les noms DB sont validés comme composants de chemin uniques : valeur non vide, différente de `.` et `..`, égale à son basename et sans `/`, `\` ni NUL.

Cette règle reproduit au sein du package dédié la règle de sécurité existante, qui est privée au package `handlers/tools`. Une valeur telle que `../outside.png` rejoint `Unsafe` avec source `database`; elle n’est ajoutée ni à l’ensemble DB comparable ni à un chemin filesystem.

## 7. Scan incomplet

Le résultat partiel n’est jamais retourné comme valide. Toute erreur de lecture de l’une des deux requêtes DB, de `ReadDir` ou d’inspection nécessaire d’une entrée retourne une erreur et une structure vide.

Le scanner ne journalise pas un pseudo-succès et ne masque aucune de ces erreurs.

## 8. Ordre déterministe et ensembles

La comparaison est isolée dans une fonction pure. Les références DB et fichiers sont convertis en ensembles.

- les doublons dans une table ou entre tables ne produisent pas plusieurs entrées `Missing` ;
- les orphelins sont triés par nom ;
- les références manquantes sont triées par nom puis type ;
- les anomalies sont triées par nom, source puis nature.

## 9. Tests ajoutés

Onze scénarios sont couverts dans `internal/imagestorage/scanner_test.go` :

1. DB et filesystem entièrement cohérents ;
2. fichier orphelin ;
3. ligne DB dont le fichier manque ;
4. mélange images principales/variantes, plusieurs écarts, ordre et doublon inter-tables ;
5. symlink non suivi et signalé ;
6. sous-dossier non parcouru et signalé ;
7. chemin de stockage qui n’est pas un dossier : erreur, pas de résultat partiel ;
8. nom DB `../outside.png` signalé sans résolution de chemin ;
9. répertoire absent et DB vide : résultat vide ;
10. répertoire absent avec référence DB : erreur ;
11. échec de la seconde lecture DB : erreur, pas de résultat partiel.

Le scénario symlink est ignoré sur les systèmes où sa création n’est pas disponible.

## 10. SQL/sqlc

SQL/sqlc a été modifié. La commande suivante réussit :

```text
sqlc generate -f db/sqlc.yaml
```

Le diff généré ajoute uniquement `ListAllImageNames` à `internal/db/images.sql.go` et `ListAllAltImageNames` à `internal/db/altImages.sql.go`.

## 11. Absence de suppression

Le package n’appelle aucune primitive de suppression, déplacement, renommage ou écriture. Il ne contient aucune fonction de purge ou de réparation. Les seules opérations sont des SELECT, `os.ReadDir` et l’inspection des métadonnées d’entrées.

## 12. Flux utilisateur

Aucun handler, route, template, upload, traitement OpenCV, génération Typst, démarrage applicatif ou comportement utilisateur n’a été modifié. Le scanner n’est appelé par aucun flux HTTP ni par `main.go`.

## 13. Validations

- `go test ./...` : réussi.
- `go vet ./...` : réussi.
- `git diff --check` : réussi.
- `sqlc generate -f db/sqlc.yaml` : réussi.

## 14. Fichiers modifiés pour ce jalon

1. `db/query/images.sql`
2. `db/query/altImages.sql`
3. `internal/db/images.sql.go`
4. `internal/db/altImages.sql.go`
5. `internal/imagestorage/scanner.go`
6. `internal/imagestorage/scanner_test.go`
7. `docs/audits/image-storage-consistency-scanner.md`
