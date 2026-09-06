# Contrat de référence historique Marking

## Résultat

Le jalon introduit uniquement le contrat persistant et le resolver sécurisé
d’une référence PNG pré-QR. Il ne produit encore aucun fichier de référence et
ne modifie aucun pipeline GenerateExam ou Marking.

## Migration et schéma réel

La migration Goose `0039_add_student_exam_page_reference.sql`, postérieure à
0038, ajoute à `student_exam_page_content` cinq colonnes nullable :

- `reference_storage_key TEXT` ;
- `reference_width INTEGER` ;
- `reference_height INTEGER` ;
- `reference_dpi INTEGER` ;
- `reference_sha256 TEXT`.

Le schéma historique avait `UNIQUE(student_exam_id, page, content, user_id)`.
Cette contrainte ne rend pas `(student_exam_id, page)` unique si le JSON diffère.
Pour ne pas casser une base legacy contenant déjà de tels doublons, la migration
ne crée pas d’index unique rétroactif. Un trigger refuse en revanche tout nouveau
doublon `(student_exam_id, page, user_id)`. Les queries de référence exigent
exactement une ligne et rendent une page legacy ambiguë indisponible.

Up conserve le contenu des lignes existantes et initialise les cinq champs à
NULL. Down supprime uniquement les triggers et colonnes du jalon ; contenu, IDs
et anciennes contraintes restent présents.

## All-null / all-present et legacy

Les triggers d’insertion et de mise à jour garantissent deux états seulement :

- les cinq valeurs sont NULL : page legacy sans référence ;
- les cinq valeurs sont présentes et valides : référence attribuée.

Aucun backfill n’est exécuté. Une ligne legacy peut faire une seule transition
de NULL complet vers une référence complète. Après attribution, un trigger
interdit modification, substitution du hash ou retour à NULL. Une seconde
écriture strictement identique est acceptée comme no-op ; la query SQLC refuse
tout écrasement opportuniste.

## Storage key, dimensions, DPI et hash

La clé ne contient ni chemin absolu, ni username, ni racine serveur. Le format
contractuel est :

`references/student-exam-<student_exam_id>/page-<page_exam>.png`

La query d’écriture vérifie que les identifiants de la clé correspondent aux
paramètres ownership-aware. Le resolver les revérifie contre la ligne relue en
DB. La DB refuse formes absolues Unix, backslashes, segments `.`/`..`, doubles
séparateurs et extensions non PNG ; le resolver reste l’autorité finale de
confinement.

Largeur et hauteur doivent être strictement positives. `reference_dpi` est fixé
par CHECK de trigger à exactement 300, et la constante Go
`StudentExamPageReferenceDPI` vaut 300. Ce choix est volontaire : ces metadata
ne décrivent pas un format raster générique mais le PNG pré-QR précis produit à
300 PPI et servant de repère à `student_exam_page_content`.

Le SHA-256 représente les bytes exacts du PNG, en hexadécimal lowercase de 64
caractères. La DB valide longueur et alphabet ; le resolver recalcule le digest
sur le fichier ouvert et exige l’égalité exacte.

## Ownership DB

`SetStudentExamPageReference` fait un UPDATE seulement si :

- la page demandée existe en une seule ligne ;
- `student_exam_page_content.user_id`, `student_exam.user_id` et
  `exams_generated.user_id` correspondent au `user_id` ;
- `student_exam.exam_generated_id` désigne réellement cette génération ;
- la clé encode le `student_exam_id` et la page demandés ;
- la référence est encore NULL ou strictement identique.

`GetStudentExamPageReference` traverse la même chaîne et retourne en une lecture
generation ID, username DB, identité de page et cinq metadata. Une page absente,
étrangère ou ambiguë produit la même absence de ligne. Les triggers ownership
existants sur page et student_exam restent actifs.

## Resolver sécurisé

`ResolveStudentExamPageReference` est dans
`internal/handlers/tools/studentExamPageReference.go`. Il :

1. relit ownership, génération, username et metadata par SQLC ;
2. retourne explicitement `reference unavailable` si les metadata sont NULL ;
3. valide la clé relative et son identité student/page ;
4. reconstruit la racine avec `operationTempDir(username, exam-<generation>)` ;
5. réutilise `safePathComponent` et `ensureDirectoryTree` pour refuser les
   parents symlink/non-répertoires ;
6. applique `Lstat`, exige un fichier régulier non symlink, l’ouvre puis vérifie
   `os.SameFile` ;
7. recalcule SHA-256 sur le descripteur ouvert ;
8. décode entièrement le fichier avec `image/png` et compare largeur/hauteur.

Les erreurs internes distinguent unavailable, corrupt et unsafe sans inclure
chemin serveur, hash attendu ou information étrangère. Aucun fallback ne relit
Typst, `ref_qcm.txt`, `assets/images` ou le PDF global.

La bibliothèque PNG standard valide le format et les pixels, mais son API ne
fournit pas une preuve fiable du PPI physique du fichier. Le DPI 300 est donc une
metadata contractuelle imposée au futur producteur ; le resolver vérifie sa
valeur DB, tandis que hash, format et dimensions sont vérifiés sur les bytes.

## Contrat de coordonnées PNG / PDF

### Repère d’origine

Les coordonnées et rayons JSON de `student_exam_page_content` ont été détectés
par OpenCV sur le PNG **pré-QR** retourné par `ExportTypstToPNGs`. Leurs unités
sont des pixels de ce raster, pas des points PDF, millimètres ou pixels du scan.
Le pipeline contractuel produit ce raster à 300 PPI, soit environ 2480 × 3508
pixels pour une page A4, mais les dimensions exactes sont enregistrées page par
page.

### Transformations

Le scan élève possède son propre repère. `Homography` le projette vers la largeur
et la hauteur de la référence PNG ; les ROI utilisent ensuite directement les
coordonnées historiques. Le PNG natif conservé au prochain jalon doit être
exactement celui sur lequel les coordonnées ont été détectées : aucune
conversion PDF->PNG, aucun scale X/Y et aucun offset ne sont autorisés dans le
contrat nouveau-format.

### Ce qui est prouvé dans ce jalon

- identité DB de la page et identité encodée dans la storage key ;
- DPI contractuel égal à 300 ;
- largeur/hauteur positives et identiques au PNG décodé ;
- SHA-256 identique aux bytes ;
- fichier entièrement décodable comme PNG réel ;
- confinement dans le workspace success de la génération possédée.

Ce jalon ne prouve pas encore que GenerateExam a conservé le bon fichier,
puisqu’il n’est pas raccordé. Le prochain jalon devra calculer les metadata sur
le même path `page` utilisé immédiatement par `CircleDetection`, puis déplacer
ou copier atomiquement ces bytes sans les réencoder avant de les attacher.

### PDF legacy et vraies copies

Un PNG rasterisé depuis le PDF historique n’est pas déclaré équivalent. PDF et
PNG utilisent des unités et rasteriseurs distincts ; même à 300 PPI, dimensions,
scale, offsets, marqueurs, antialiasing et positions peuvent diverger. Aucun
facteur d’échelle silencieux n’est prévu.

Une future suite privée opt-in, jamais versionnée ni exécutée en CI, devra sur
`db_test_real.db` et plusieurs scans comparer : width/height, positions ROI,
homographie, MeanGray, detected_state, QuestionMark et score. La référence PNG
pré-QR native doit conserver strictement le repère. Le fallback PDF legacy ne
sera admissible que si les résultats finaux démontrent sa compatibilité ; une
simple faible différence de pixels ne suffira pas.

## Plusieurs pages et plusieurs jobs

Chaque page possède sa clé et son hash. Les tests résolvent deux pages du même
student_exam et prouvent que la page demandée est retournée. La relation ne
contient aucun marking_job : tous les jobs d’une génération partageront la même
référence durable sans duplication.

## Future couverture avant success

Le prochain jalon devra ajouter une finalisation Exam conditionnelle. Pour la
génération nouveau-format, elle vérifiera atomiquement qu’il n’existe aucune
ligne `student_exam_page_content` rattachée à la génération dont une des cinq
metadata est NULL, et que le nombre de pages snapshotées égale le nombre de
références complètes. Les triggers all-null/all-present simplifient cette garde,
mais la couverture et la présence des fichiers validés devront être garanties
avant `CompleteExamGeneration`. Ce blocage success n’est pas codé ici.

## Backup et lifecycle

La référence est une donnée durable de la génération. Backup et restauration
doivent traiter ensemble :

- la DB SQLite ;
- `assets/tmp/<username>/exam-<generation_id>/` ;
- le PDF final ;
- le sous-répertoire `references/`.

Le SHA-256 permet de détecter une restauration incomplète ou corrompue. Aucun
système de backup n’est ajouté dans ce jalon.

## Tests synthétiques

Les tests couvrent : migration Up/Down, contenu legacy inchangé, metadata NULL,
all-null/all-present, DPI 300, écriture et relecture exacte, ownership et page
absente, immutabilité de chaque champ, no-op identique, nouveaux doublons
refusés, doublons legacy non résolus, traversal DB, deux pages, wrong user et
wrong username, hash modifié, dimensions fausses, faux PNG avec hash exact,
metadata legacy, fichier symlink, parent symlink, clés absolues/traversal,
extension incorrecte et clé d’un autre student/page.

Aucune DB réelle, aucun scan et aucune donnée nominative ne sont utilisés.

## Périmètre et validation

- GenerateExam modifié : non.
- Marking modifié : non.
- `TypstWriter`, `ExportTypstToPNGs`, `Homography`, OpenCV et PDF : inchangés.
- SQLC : `SetStudentExamPageReference` et `GetStudentExamPageReference` seulement.
- `sqlc generate -f db/sqlc.yaml` : succès.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `db/migrations/0039_add_student_exam_page_reference.sql` ;
- `db/query/studentExamPageContent.sql` ;
- sorties SQLC correspondantes dans `internal/db/` ;
- `internal/db/studentExamPageReference_test.go` ;
- `internal/handlers/tools/studentExamPageReference.go` ;
- `internal/handlers/tools/studentExamPageReference_test.go` ;
- ce rapport.
