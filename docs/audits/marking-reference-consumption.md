# Consommation de la référence historique par Marking

## Ancien chemin

`MarkingStudentExam` lisait le snapshot `student_exam_content`, reconstruisait chaque copie avec `TypstWriter`, invoquait `ExportTypstToPNGs` à 300 PPI, puis associait les PNG reconstruits aux scans triés avant `Homography`. Cette reconstruction dépendait de `internal/config/ref_qcm.txt`, des images vivantes, des polices et de la version courante de Typst.

Les snapshots `student_exam_content` et `student_exam_page_content` restent lus : ils portent toujours les réponses attendues, les points/tags historiques et les coordonnées pixel des ROI.

## Nouveau chemin

Après le tri existant des scans par `page_exam`, `resolveMarkingPageReferences` traite chaque couple `(student_exam_id, page_exam)` :

1. il appelle `ResolveStudentExamPageReference` avec l'utilisateur authentifié ;
2. le resolver relit l'identité et les metadata ownership-aware depuis la DB ;
3. il vérifie confinement, symlinks, fichier régulier, SHA-256, PNG et dimensions ;
4. son chemin validé est passé directement à `Homography` dans le même ordre que les scans.

Le chemin nouveau-format n'appelle ni `TypstWriter`, ni `ExportTypstToPNGs`, ni `ref_qcm.txt`, ni les images vivantes. Le PDF Exam n'est jamais utilisé comme référence.

## Critère nouveau-format et legacy

Le choix ne dépend jamais de l'absence du fichier :

- les cinq metadata 0039 présentes imposent le resolver ; toute absence de fichier, corruption, dimension incohérente, hash invalide ou chemin unsafe est une erreur d'intégrité, sans fallback ;
- les cinq metadata NULL sur une page possédée identifient explicitement une page legacy et autorisent temporairement l'ancien rendu Typst ;
- une metadata partielle, une page absente/ambiguë ou un mauvais owner n'est pas legacy et est refusé.

Pour un nouveau-format, `ErrMarkingHistoricalReference` est traité comme une erreur critique par `ProcessExamsConcurrently`. Le job ne peut donc pas être finalisé `success` en transformant une corruption d'artefact en simple outcome pédagogique par copie.

Le fallback PDF legacy reste entièrement ouvert et n'est ni implémenté ni validé ici.

## Repère géométrique et Homography

Le resolver fournit exactement le PNG pré-QR 300 PPI dont les octets ont servi à `CircleDetection` et `CircleDetectionAnswer`. Aucun resize, raster PDF, réencodage, changement de DPI, scale ou offset n'est ajouté.

Le bloc algorithmique de `Homography` est inchangé : lecture couleur, grayscale, adaptive threshold, SIFT, BF/KNN, ratio 0,65, RANSAC et warp restent identiques. L'adaptation mécanique accepte un chemin absolu de référence validé et produit un nom de sortie unique contenant `student-exam-<id>` afin d'éviter les collisions entre copies concurrentes dont les références se nomment toutes `page-N.png`.

## Cleanup Marking

Pour un nouveau-format, aucun `.typ` ni PNG de reconstruction n'est créé, et aucune référence durable n'entre dans la liste de suppression du workspace Marking. Les scans, homographies, PDF de copie et autres temporaires historiques gardent leur lifecycle existant.

Pour une page legacy NULL, le `.typ` et les PNG reconstruits sont encore produits puis supprimés comme auparavant.

## Multi-pages et performance

Les scans sont toujours triés par `page_exam`; la référence de chaque page est résolue explicitement avec ce numéro. Les tests couvrent des pages fournies initialement dans un ordre arbitraire et vérifient l'association page 1/page 2.

Le chemin nouveau-format supprime une compilation Typst et une rasterisation complète par copie. Son coût supplémentaire est la validation du fichier durable (lecture/hash/décodage), nécessaire à l'intégrité historique et nettement plus faible que la reconstruction Typst attendue.

## Tests synthétiques

La couverture ajoute :

- deux pages durables résolues dans l'ordre sans aucun appel au renderer legacy ;
- metadata présente et fichier corrompu refusés sans fallback ;
- mauvais utilisateur refusé sans fallback ;
- metadata toutes NULL déclenchant explicitement le seul fallback historique ;
- conservation du chemin exact fourni par le resolver ;
- comparaison déterministe ancien chemin relatif / référence absolue de mêmes octets à travers `Homography`, `MeanGray`, `detected_state`, `QuestionMark` et score ;
- nom d'homographie nouveau-format unique par `student_exam`.

Le seuil reste `MarkingDetectionThreshold = 150.0`. `GetAnswerDetections`, `CountingPoints`, `CountingTotalPoint`, l'adaptateur de persistance et les règles d'outcome ne sont pas modifiés.

## Suite réelle opt-in

`TestRealHistoricalReferencePathEquivalence` réutilise le mécanisme privé existant. Il :

1. copie la DB configurée dans un fichier temporaire et applique les migrations uniquement à cette copie ;
2. exécute le chemin legacy sur les scans ;
3. reconstruit le même raster Typst dans un workspace utilisateur isolé ;
4. stocke exactement ces octets avec `StoreStudentExamPageReference` dans la copie DB/workspace ;
5. supprime les sources reconstruites ;
6. réexécute Marking via le resolver sur une nouvelle extraction du même scan ;
7. compare score/total et toute la hiérarchie de questions, donc `MeanGray`, `detected_state`, `QuestionMark` et score par question.

Le test ne journalise que les IDs techniques et les compteurs. Il ne modifie jamais la DB originale et ne copie aucun scan ou résultat nominatif dans Git.

Variables requises :

- `LAZYMARKING_TEST_DB`
- `LAZYMARKING_TEST_USER_ID`
- `LAZYMARKING_TEST_PDF_1_PAGE`
- `LAZYMARKING_TEST_STUDENT_EXAM_ID_1_PAGE`
- `LAZYMARKING_TEST_PDF_2_PAGES`
- `LAZYMARKING_TEST_STUDENT_EXAM_ID_2_PAGES`
- `LAZYMARKING_TEST_PDF`
- `LAZYMARKING_TEST_STUDENT_EXAM_ID_3_PAGES`

Commande :

```sh
go test ./internal/handlers/tools -run '^TestRealHistoricalReferencePathEquivalence$' -v -count=1
```

Le harness copie d'abord la DB source ouverte en lecture seule, calcule son SHA-256, applique les sections `Up` jusqu'à 0039 uniquement à la copie, puis crée les queries utilisées par Marking. Une assertion précoce exige une version de migration 39, les cinq colonnes `reference_*` via `PRAGMA table_info(student_exam_page_content)`, et des valeurs NULL pour toutes les lignes legacy. Le SHA-256 de la DB source est revérifié à la fermeture.

`db.InitDB` n'exécute aucune migration : c'était la cause du premier échec du harness. La CLI Goose ne peut par ailleurs pas parser les triggers de l'ancienne migration 0036 dépourvue de blocs `StatementBegin/End`. Le harness applique donc, sur la copie uniquement, le SQL `Up` de chaque migration manquante avec le driver SQLite et enregistre les versions dans `goose_db_version`. Aucune migration historique ni DB source n'est modifiée.

La validation privée a été exécutée sur trois copies techniques :

- `student_exam_id=728` : 1 page, 4 questions ;
- `student_exam_id=14` : 2 pages, 6 questions ;
- `student_exam_id=366` : 3 pages, 13 questions.

L'ID initialement configuré pour le paquet deux pages était 12. Le décodage des deux QR du PDF a identifié le groupe technique 14 avec les pages 1 et 2 ; l'ID 12 correspond à une autre copie présente en DB. Le message intermédiaire `QR code not found` provient d'une tentative du premier décodeur, mais le fallback QR existant retrouve bien les deux pages. Aucun algorithme QR n'a été modifié.

Sur les 3 copies et 6 pages, les deux homographies réussissent. La comparaison profonde de tous les résultats par question établit une égalité exacte des MeanGray, `detected_state`, `QuestionMark`, scores par question et scores totaux. Aucun écart numérique n'a été observé et aucune tolérance n'a été introduite.

Cette validation compare uniquement l'ancien raster Typst aux mêmes octets consommés via le resolver. Elle ne prétend pas valider `PDF -> PNG` pour le legacy.

## Validation standard

- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.
- validation privée : succès, 3 copies / 6 pages, résultats strictement identiques.

## Fichiers modifiés pour ce jalon

- `internal/handlers/tools/markingPageReferences.go`
- `internal/handlers/tools/markingPageReferences_test.go`
- `internal/handlers/tools/markingStudentExam.go`
- `internal/handlers/tools/processExamsConcurrently.go`
- `internal/handlers/tools/homography.go`
- `internal/handlers/tools/homography_test.go`
- `internal/handlers/tools/realMarking_integration_test.go`
- `docs/audits/marking-reference-consumption.md`

Aucune migration, requête SQLC, modification de PDF, recalibrage OpenCV ou changement de persistance des résultats n'a été ajouté.
