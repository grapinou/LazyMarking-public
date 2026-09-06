# Pages alignées Marking — production

## Contrat livré

Pour chaque copie corrigée, `MarkingStudentExam` conserve désormais chaque PNG
issu de `Homography` après le calcul des détections et avant que `DrawMarking` ne
réécrive l'image. Cette page non annotée est donc exactement le repère raster
dans lequel les centres, rayons et `MeanGray` ont été mesurés.

La page est d'abord copiée dans le staging propre à la copie :

`<workspace>/.aligned-staging/student-exam-<student_exam_id>/page-<page_exam>.png`

Après la persistance transactionnelle du résultat `corrected`, le pipeline
publie chaque page sous la clé durable :

`aligned/student-exam-<student_exam_id>/page-<page_exam>.png`

Le workspace reste celui du job `assets/tmp/<username>/marking-<job_id>/`. Les
pages de copies différentes et les pages d'une même copie ne peuvent donc pas
se remplacer.

## Fidélité et publication

`StoreMarkingAlignedPage` ne réencode pas l'image. Il ouvre le fichier régulier
staged, valide le PNG et ses dimensions, copie ses octets dans un temporaire en
calculant SHA-256, synchronise ce fichier puis publie son inode par hard-link.
Cette publication est atomique sur le filesystem du workspace et ne remplace
jamais une destination existante. Une destination préexistante n'est acceptée
que si son contenu possède exactement le même hash.

Les métadonnées DB (`storage_key`, largeur, hauteur et SHA-256) sont attachées
après la publication. Le resolver vérifie à chaque lecture l'ownership complet,
la clé canonique, les répertoires sans symlink, le fichier régulier, son hash,
son format PNG et ses dimensions.

## Ownership et confinement

Avant toute publication, `ValidateMarkingAlignedPageTarget` exige que la copie :

- appartienne à l'utilisateur et au job demandés ;
- corresponde au `student_exam` demandé ;
- soit terminale avec l'outcome `corrected` ;
- possède exactement la page demandée dans `student_exam_page_content`.

La source doit se trouver directement dans le staging déterministe de cette
copie. La destination est reconstruite depuis une clé relative dont chaque
composant est validé. Les sources, parents ou destinations symlink, les chemins
absolus, traversals et clés non canoniques sont refusés.

`ResolveMarkingAlignedPage` recoupe en outre le username fourni avec celui du
propriétaire DB du job. Une demande cross-user, cross-job, cross-copy ou
cross-student ne révèle aucun chemin et retourne une indisponibilité.

## Finalisation et atomicité

Il n'existe pas de transaction commune entre SQLite et le filesystem. Le
protocole est donc : résultat `corrected` transactionnel, publication d'une page,
métadonnée DB, suppression du staging correspondant, puis page suivante.

`CompleteMarkingJobWithResults` interdit désormais la transition vers `success`
si une copie `corrected` n'a pas exactement `expected_pages` pages alignées ou
si une page attendue de `student_exam_page_content` n'est pas attachée. Une
erreur de fichier ou de metadata laisse le job non-success ; le pipeline le fait
ensuite suivre son lifecycle d'échec existant. Une publication filesystem sans
metadata peut être rejouée uniquement avec les mêmes octets.

Les outcomes `incomplete`, `not_seen` et `error` ne requièrent aucune page
alignée : aucune détection pédagogique exploitable n'est enregistrée pour ces
copies.

## Lifecycle

Le cleanup normal retire chaque fichier de staging après sa publication mais
conserve le sous-répertoire durable `aligned/` avec le PDF corrigé. Les jobs
`success` restent exclus de la purge. Les workspaces des jobs `failed` expirés
sont supprimés, et les lignes `marking_aligned_pages` correspondantes sont
supprimées par cascade avec la hiérarchie de résultats.

Les jobs historiques déjà terminés ne sont pas modifiés. La nouvelle exigence
de couverture ne concerne que le chemin de finalisation nouveau-format encore
`running`.

## Couverture de tests

Les tests couvrent notamment :

- identité byte-for-byte avant annotation et stabilité des détections ;
- égalité du SHA-256 fichier / metadata et validation des dimensions ;
- pages distinctes entre pages et copies ;
- corruption, metadata incohérente et fichier manquant ;
- ownership erroné, traversal et symlinks source ou destination ;
- refus de finaliser une copie corrected sans pages alignées ;
- conservation des pages d'un job success et cascade/purge d'un job failed.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.

## Fichiers du jalon

- `db/query/markingJobs.sql`
- `db/query/markingReviews.sql`
- fichiers SQLC générés correspondants dans `internal/db/`
- `internal/config/config.go`
- `internal/handlers/tools/markingAlignedPage.go`
- `internal/handlers/tools/markingAlignedPage_test.go`
- `internal/handlers/tools/markingStudentExam.go`
- `internal/handlers/tools/processExamsConcurrently.go`
- tests de finalisation DB et de purge
- ce rapport

Aucune migration supplémentaire n'est nécessaire :
`0040_add_marking_review_model.sql` fournit déjà la table, les contraintes,
l'ownership, l'immuabilité et les cascades requis.
