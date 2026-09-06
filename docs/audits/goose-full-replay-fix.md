# Correction du replay complet Goose

## Échec reproduit avant correction

Le défaut a été reproduit avec le binaire installé :

```text
goose version: v3.27.3
```

Commande exécutée sur une base SQLite temporaire neuve :

```text
goose -dir db/migrations sqlite <temp>/before.db up
```

Les migrations 0001 à 0035 ont réussi. La migration
`0036_bind_marking_jobs_to_exam_generation.sql` a échoué sur le premier trigger
avec :

```text
failed to execute SQL query "CREATE TRIGGER ... BEGIN
    SELECT RAISE(...);": SQL logic error: incomplete input (1)
```

## Cause exacte

Un trigger SQLite contient un point-virgule interne avant son `END;`. Sans
directive Goose, le parser traite ce point-virgule comme la fin de la requête et
envoie un `CREATE TRIGGER ... BEGIN ...;` incomplet à SQLite.

Les migrations 0037, 0038 et 0039 contenaient la même forme de trigger. La 0040
utilisait déjà correctement `-- +goose StatementBegin` et
`-- +goose StatementEnd`.

## Modifications des migrations

Les blocs `CREATE TRIGGER ... BEGIN ... END;` ont uniquement été encadrés par
les directives de parsing Goose dans :

- `0036_bind_marking_jobs_to_exam_generation.sql` : 2 triggers ;
- `0037_create_marking_results.sql` : 4 triggers ;
- `0038_add_marking_job_result_metadata.sql` : 3 triggers ;
- `0039_add_student_exam_page_reference.sql` : 4 triggers.

Aucun nom de table, colonne, trigger ou contrainte n'a changé. Aucune expression
SQL, condition, action de trigger, section Down ou logique métier n'a changé.
Les directives sont des commentaires pour SQLite et servent uniquement au
découpage effectué par Goose.

**SQL métier historique modifié : non.**

Les sections Down concernées contiennent seulement des `DROP TRIGGER`,
`DROP TABLE` et `ALTER TABLE ... DROP COLUMN` simples ; elles ne nécessitent pas
de bloc StatementBegin/End.

## Replay fresh et status

Après correction, le replay complet sur une nouvelle base temporaire réussit :

```text
goose: successfully migrated database to version: 40
```

`goose status` liste toutes les migrations 0001 à 0040 comme appliquées. La
dernière migration atteinte est `0040_add_marking_review_model.sql`.

Des cycles indépendants ont également été exécutés sur des bases neuves pour
chaque migration modifiée :

```text
up-to 36 -> down -> up-to 36 : succès, version 36
up-to 37 -> down -> up-to 37 : succès, version 37
up-to 38 -> down -> up-to 38 : succès, version 38
up-to 39 -> down -> up-to 39 : succès, version 39
```

## Compatibilité des bases existantes

Goose v3.27.3 enregistre dans `goose_db_version` l'identifiant de version,
`is_applied` et le timestamp ; aucun checksum du contenu du fichier migration
n'est stocké. Une seconde commande `up` sur une base déjà à la version 40 répond
`no migrations to run` et ne rejoue donc pas 0036–0039.

Une base de production ayant déjà appliqué ces versions reste compatible et ne
doit effectuer aucune action particulière. Les directives n'altèrent pas le SQL
qui avait été exécuté antérieurement.

**Base déjà migrée compatible : oui.**

## Protection automatisée

Le nouveau script `scripts/check-migrations.sh` :

1. exige le binaire `goose` ;
2. crée un répertoire et une DB via `mktemp` ;
3. applique toutes les migrations avec Goose ;
4. compare la version appliquée à la dernière migration présente ;
5. affiche le status Goose ;
6. nettoie le répertoire temporaire via un trap.

`scripts/check.sh` appelle ce contrôle dans le workflow normal. La CI installe
explicitement Goose v3.27.3 avant de lancer le script partagé :

```text
go install github.com/pressly/goose/v3/cmd/goose@v3.27.3
```

Cette version n'est pas une nouvelle dépendance applicative : c'est l'outil de
migration déjà documenté dans le README, fixé dans la CI à la version qui a
reproduit et validé le défaut. Aucun fichier de DB n'est créé dans le dépôt.

Données privées nécessaires : **non**.

## Validations

- échec 0036 avant correction : reproduit avec Goose v3.27.3 ;
- fresh `up` 0001 → 0040 : succès ;
- fresh `status` : toutes les migrations appliquées ;
- cycles Down/Up 0036, 0037, 0038 et 0039 : succès ;
- seconde commande `up` sur version 40 : no-op ;
- `scripts/check-migrations.sh` : succès ;
- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés ou créés

- `.github/workflows/ci.yml` ;
- `db/migrations/0036_bind_marking_jobs_to_exam_generation.sql` ;
- `db/migrations/0037_create_marking_results.sql` ;
- `db/migrations/0038_add_marking_job_result_metadata.sql` ;
- `db/migrations/0039_add_student_exam_page_reference.sql` ;
- `scripts/check.sh` ;
- `scripts/check-migrations.sh` (créé) ;
- `docs/audits/goose-full-replay-fix.md` (créé, rapport local).

Aucun handler, service, query SQLC, template ou runtime Marking n'a été modifié.
