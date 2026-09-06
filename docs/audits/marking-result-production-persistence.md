# Persistance de production des résultats Marking

## Contrat livré

Un job créé par l'upload porte désormais `result_schema_version = 1`,
`marking_algorithm_version = "1"` et le seuil réellement utilisé par le
runtime, `detection_threshold = 150.0`. Ces quatre éléments d'identité avec
`exam_generated_id` sont immuables en base. Les anciennes lignes conservent
leurs métadonnées `NULL` et ne sont pas backfillées.

La liste attendue des copies vient de la relation persistante du job vers sa
génération, puis de `student_exam` et `student_exam_content`. Elle ne dépend pas
des QR reçus.

## Écriture des outcomes

- `corrected` : `DetailedResult` est adapté par
  `MarkingCopyResultToPersistedInput`, puis copie, questions et détections sont
  écrites atomiquement par `PersistCorrectedMarkingCopyWithQueries`. Le
  `MeanGray`, les états, les indices zéro-based et les demi-points sont
  conservés.
- `incomplete` : une copie reconnue avec pages manquantes, hors plage ou
  dupliquées est terminalisée sans score avec `page_set_incomplete`.
- `error` : une copie dont le jeu de pages est complet mais dont le traitement
  local échoue est terminalisée sans score avec `copy_processing_error`.
- `not_seen` : après les groupes QR, chaque copie attendue absente est écrite
  avec zéro page détectée et `no_qr_pages`.

Les codes sont courts et stables. Aucun chemin serveur ni détail d'erreur brut
n'est persisté. Les QR illisibles restent dans le mécanisme PDF/filesystem
existant ; ils conduisent indirectement à `not_seen` ou `incomplete`. Un QR
cross-user ou cross-generation reste une erreur critique et ne déclenche aucune
lecture de snapshot étrangère.

## Finalisation et atomicité

`CompleteMarkingJobWithResults` est l'unique chemin de production nouveau-format
vers `success`. Son `UPDATE` atomique exige : job possédé et `running`,
métadonnées exactes, une ligne terminale pour chaque `student_exam` attendu et,
pour chaque résultat `corrected`, le nombre exact de questions du snapshot et
le nombre exact de détections de chaque question. Les contraintes ownership et
unicité empêchent qu'un résultat extérieur ou doublon satisfasse la couverture.

Le travail image et Typst reste hors transaction. Chaque copie corrected est
écrite dans une transaction courte. Les PDF sont produits avant l'UPDATE final.
Une erreur de persistance, de PDF ou de finalisation fait passer le job à
`failed`. Des résultats déjà écrits peuvent rester temporairement sous ce job :
le recovery conserve ces enfants en passant `running` à `failed`, puis la purge
des failed de plus de sept jours les supprime par cascade. Les jobs success et
leurs workspaces restent exclus de la purge.

Un success avec issues pédagogiques est permis : toutes les copies ont un
outcome terminal, mais certaines peuvent être `incomplete`, `not_seen` ou
`error`. Aucun état `completed_with_issues` n'est ajouté. Les compteurs de
progression gardent leur ancien sens (groupes QR et copies corrected) et ne
présentent pas encore ce bilan riche.

## Compatibilité et périmètre inchangé

Les jobs legacy à métadonnées `NULL` restent lisibles, récupérables, conservés
s'ils sont success et purgeables s'ils sont failed. Le chemin générique legacy
existe encore, mais n'est plus appelé par `ProcessMarking`.

Le score n'est calculé qu'une fois dans le runtime existant. Le seuil, SIFT,
l'homographie, RANSAC, le dessin, les PDF et les statistiques runtime n'ont pas
changé. Les statistiques ne sont pas lues depuis la DB. La référence historique
Typst/images reste un P1 ouvert ; la persistance structurée ne rend donc pas
encore les PDF honnêtement régénérables.

## Migration et SQLC

La migration `0038_add_marking_job_result_metadata.sql` ajoute trois colonnes
nullable, valide leur présence groupée et leur domaine, protège leur immutabilité
ainsi que celle de la génération, et possède un Down ciblé. SQLC fournit la
création nouveau-format, la liste des copies attendues, la couverture, les
outcomes terminaux et la finalisation conditionnelle. Les fichiers générés ont
été produits exclusivement par `sqlc generate -f db/sqlc.yaml`.

## Tests et validation

Les tests synthétiques couvrent : métadonnées upload, Up/Down et legacy NULL,
immutabilité, success avec deux corrected et un not_seen, couverture manquante,
enfants corrected incomplets, cross-generation, deux jobs sur la même copie,
round-trip détaillé avec MeanGray, incomplete/error et pages dupliquées,
résultats partiels conservés au recovery, cascade complète d'un failed purgé et
préservation des success.

- `sqlc generate -f db/sqlc.yaml` : succès
- `./scripts/check.sh` : succès
- `go test -race ./...` : succès
- `git diff --check` : succès

## Fichiers modifiés par ce jalon

- `db/migrations/0038_add_marking_job_result_metadata.sql`
- `db/query/markingJobs.sql`
- `db/query/markingResults.sql`
- fichiers SQLC générés dans `internal/db/`
- `internal/db/markingResultsRepository.go`
- tests DB de génération, schéma et persistance de production
- handler d'upload et son test
- constante de seuil et constantes de version runtime
- pipeline `processMarking` / `processExamsConcurrently` et leurs tests
- tests de recovery et purge
- ce rapport

Les autres fichiers déjà modifiés ou non suivis dans le worktree ne font pas
partie de ce jalon.
