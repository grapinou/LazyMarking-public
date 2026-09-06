# Référence historique Marking — production GenerateExam

## Contrat implémenté

Chaque page d'une nouvelle génération conserve comme donnée durable le fichier PNG natif pré-QR retourné par `ExportTypstToPNGs`. Dans `BuildQcmStudentCtx`, `page` est le chemin de ce fichier et `pageName` son nom dans le workspace `assets/tmp/<username>/exam-<generation_id>/`.

`PasteQrCodeOnPage(tempDir, qrName, pageName)` lit ce PNG et écrit un fichier distinct préfixé `qr_`; il ne remplace pas la source. `CircleDetection(tempDir, pageName)` et tous les appels à `CircleDetectionAnswer(tempDir, pageName, ...)` lisent donc le même PNG pré-QR. La référence est stockée après la détection et l'insertion de `student_exam_page_content`, avant la conversion du PNG QR en PDF et avant le cleanup.

## Stockage byte-for-byte

`StoreStudentExamPageReference` :

- vérifie par la DB l'utilisateur, la génération, le `student_exam` et la page ;
- exige que la source soit un fichier régulier non-symlink situé directement dans le workspace de la génération ;
- lit largeur et hauteur avec `png.DecodeConfig`, sans réencodage ;
- copie les octets source avec `io.Copy` tout en calculant SHA-256 ;
- écrit dans un temporaire du répertoire destination, le synchronise et le ferme ;
- publie l'inode complet atomiquement avec un hard-link, puis retire le nom temporaire et synchronise le répertoire.

Le hard-link remplit ici le rôle d'une publication atomique sur le même filesystem tout en apportant une propriété que `os.Rename` n'offre pas : il refuse de remplacer une destination existante. Si la destination existe déjà, seuls des octets de hash strictement identique sont acceptés ; une référence différente provoque une erreur. Aucun `image.Encode`, `png.Encode`, `imwrite`, redimensionnement ou recompilation Typst n'intervient en production.

Le SHA-256 hexadécimal attaché à la DB est calculé sur les octets source copiés. Le resolver relit ensuite la destination et exige le même hash, un PNG réellement décodable et les mêmes dimensions. Les tests prouvent `bytes(source) == bytes(destination)` et `SHA(source) == SHA(destination) == reference_sha256`. Une modification ou suppression ultérieure de la source n'affecte pas la référence.

## Métadonnées et organisation

La clé déterministe est :

`references/student-exam-<student_exam_id>/page-<page_exam>.png`

Elle reste relative et confinée sous `assets/tmp/<username>/exam-<generation_id>/`. Les métadonnées 0039 sont attachées uniquement après publication du fichier : clé, largeur, hauteur, DPI contractuel 300 et SHA-256. `SetStudentExamPageReference` reste ownership-aware et l'immuabilité DB interdit le remplacement ultérieur.

Le DPI 300 est le contrat du producteur. Il n'est pas prétendu relu depuis un chunk PNG lorsque celui-ci ne porte pas une information PPI fiable ; le hash, le format et les dimensions sont vérifiés sur le fichier.

## Couverture et finalisation

`GetExamGenerationReferenceCoverage` compte les lignes `student_exam_page_content`, les lignes entièrement référencées et les couples `(student_exam_id, page)` ambigus. `ListExamGenerationPageReferences` fournit la liste ownership-aware à valider.

Avant la transition terminale, `ValidateExamGenerationReferences` exige :

- au moins une page historique ;
- autant de références complètes que de pages ;
- aucune ambiguïté de page ;
- une liste sans doublon et de cardinalité identique ;
- la résolution sécurisée de chaque fichier, donc confinement, absence de symlink, fichier régulier, SHA-256, format PNG et dimensions valides.

`CompleteExamGenerationWithReferences` réalise ensuite un UPDATE SQL atomiquement limité à une génération possédée encore `running`, avec au moins une page, aucune metadata incomplète et aucune page ambiguë. Une référence manquante ou corrompue bloque donc `success`. Les générations legacy déjà `success` ne sont ni relues ni rétrogradées.

Il n'existe pas d'atomicité commune entre SQLite et le filesystem. Le protocole réel est : publication atomique du fichier, metadata DB, production et fusion du PDF, cleanup des intermédiaires, validation exhaustive des références, puis UPDATE conditionnel vers `success`. Tout échec avant ce point laisse la génération non-success ; le cleanup/recovery existant supprime ensuite le workspace entier et ses références partielles.

## Cleanup et lifecycle

Le cleanup de succès ne parcourt que les fichiers racine correspondant à `*.png` et `*.typ`, ainsi que la liste explicite des PDF intermédiaires. Le sous-répertoire `references/` et le PDF global final sont conservés. Un test protège explicitement ce contrat.

Le cleanup d'échec et le recovery suppriment le workspace complet. Les tests couvrent une référence partielle dans une génération interrompue : elle disparaît avec le workspace, tandis que la génération suit son lifecycle existant vers `failed`. Une génération `running` présente au démarrage n'est pas reprise vers `success` sans références.

Le contenu, l'ordre, la pagination, les QR et le nom du PDF global ne sont pas modifiés.

## Contrat de coordonnées PNG / PDF

`student_exam_page_content` contient des coordonnées pixel mesurées par `CircleDetection` et `CircleDetectionAnswer` sur le PNG pré-QR natif. La référence durable est une copie exacte des octets de ce même raster, avec largeur, hauteur, DPI contractuel et hash enregistrés.

Aucun PDF n'est rasterisé, aucun repère PDF n'est utilisé, et aucun scaling ou offset n'est appliqué. Ce jalon prouve l'identité byte-for-byte entre le repère de détection produit et le repère durable. Le jalon de consommation Marking devra encore valider avec les scans privés opt-in la stabilité de l'homographie, des ROI, des MeanGray, des états détectés, des `QuestionMark` et du score. Ces données privées ne sont ni utilisées ni versionnées ici.

## Impact Marking

Marking n'est pas modifié. `MarkingStudentExam`, `Homography`, `GetAnswerDetections`, `ProcessMarking` et la reconstruction Typst actuelle restent inchangés. Aucun fallback Typst ou PDF n'est ajouté.

## Tests

Les tests synthétiques couvrent :

- conservation exacte des octets, hashes et dimensions ;
- indépendance après modification de la source ;
- destination identique idempotente et destination différente refusée ;
- plusieurs `student_exam` et plusieurs pages, avec clés distinctes ;
- référence DB manquante, fichier manquant et fichier corrompu ;
- ambiguïté historique de page refusée ;
- finalisation valide vers `success` ;
- conservation des références et du PDF final lors du cleanup de succès ;
- suppression des références partielles lors du cleanup/recovery d'échec.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés pour ce jalon

- `db/query/examsGenerated.sql`
- `db/query/studentExamPageContent.sql`
- `internal/db/examsGenerated.sql.go` (généré par sqlc)
- `internal/db/studentExamPageContent.sql.go` (généré par sqlc)
- `internal/handlers/generateExams/examGenerationCompleted.go`
- `internal/handlers/generateExams/examGenerationReferences_test.go`
- `internal/handlers/generateExams/finalizeExamGeneration_test.go`
- `internal/handlers/generateExams/handlers.go`
- `internal/handlers/tools/buildQcmStudentCtx.go`
- `internal/handlers/tools/cleanupFailedExamGeneration_test.go`
- `internal/handlers/tools/recoverExamGenerations_test.go`
- `internal/handlers/tools/storeStudentExamPageReference.go`
- `internal/handlers/tools/studentExamPageReference_test.go`
- `internal/handlers/tools/validateExamGenerationReferences.go`
- `docs/audits/marking-reference-production.md`

Aucune migration nouvelle n'a été nécessaire : le schéma 0039 suffit.
