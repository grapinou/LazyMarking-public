# Contrat moderne de corrected_NOT.pdf

Date : 2026-09-01

## Comportement avant modification

`ProcessMarking` séparait l'upload en `page-N.pdf`, rasterisait chaque page en
`page-N.png`, puis supprimait les PDF séparés. Une page correctement rattachée à
une copie et corrigée voyait son PNG source supprimé par `MarkingStudentExam`.
Les PNG sans QR et ceux d'une copie dont la correction n'aboutissait pas
restaient dans le workspace.

`LeftPages` globait ensuite historiquement tous les `*.png`, les convertissait
en PDF, les fusionnait sous un nom dérivé de `corrected.pdf`, puis supprimait
PNG et PDF intermédiaires. Depuis la suppression de cet appel dans le GET
`/dashboard/marking/success`, aucun chemin moderne ne l'appelait. Le GET était
bien read-only, mais `corrected_NOT.pdf` n'était plus produit.

Le glob historique était en outre trop implicite pour le pipeline moderne : un
échec partiel peut laisser des PNG dérivés (homographie/références), qui ne sont
pas des pages originales du scan.

## Définition exacte d'une page rejetée

Une left page moderne est exclusivement un PNG original `page-N.png` issu de
la page `N` du PDF scanné et appartenant à l'une des catégories suivantes :

- QR non détecté ou non décodable, tel que renvoyé par
  `ProcessPagesConcurrently` ;
- QR valide et groupe connu, mais copie absente de la liste `markExams`
  effectivement corrigée (cas incomplete/error).

Les PNG dérivés ne sont jamais sélectionnés. Les outcomes DB
`incomplete/error/not_seen` restent un résumé de copies distinct : le nombre de
pages du PDF de récupération n'est pas dérivé de ces compteurs.

## Source et résolution

`ResolveMarkingRejectedScanPages` construit d'abord un catalogue fermé depuis
les noms `page-N.pdf` produits par `pdfseparate`. Il associe à chaque entrée son
PNG original attendu et son numéro numérique de page. Les noms QR illisibles et
les pages des groupes non corrigés ne peuvent sélectionner qu'une entrée de ce
catalogue ; tout nom extérieur provoque un échec du job.

La liste est triée par entier `N`, et non par glob lexical. L'ordre reste donc
`page-1`, `page-2`, …, `page-10`, y compris au-delà de neuf pages.

## Primitive et publication

`BuildUncorrectedPagesPDF` reçoit des `MarkingRejectedScanPage` typées. Elle :

1. ne fait rien lorsque la liste est vide ;
2. valide le workspace avec les helpers de confinement existants ;
3. exige pour chaque source un fichier régulier non symlinké, directement dans
   le workspace et nommé exactement `page-N.png` ;
4. crée un staging `.corrected-not-*` dans ce workspace ;
5. convertit les PNG sans les déplacer ni les supprimer ;
6. fusionne les PDF intermédiaires dans l'ordre numérique ;
7. valide le fichier régulier, son confinement, sa taille minimale et l'en-tête
   `%PDF-` avec la validation existante ;
8. publie par `os.Rename` sous le nom canonique `corrected_NOT.pdf` ;
9. supprime toujours le staging.

Le nom canonique reste celui déjà attendu par `MarkingArtifactExists`, le view
model résultat et `/dashboard/marking/servePDF`. Aucune persistance DB de nom
déterministe n'est nécessaire.

## Moment de génération et lifecycle

La résolution et la génération ont lieu après :

- QR et regroupement ;
- corrections de copies et persistance des résultats ;
- création de `corrected.pdf` ;
- compilation de `mark-table.pdf`, dernier consommateur de `qrNotDetected` ;

mais avant `CompleteMarkingJobWithResults`, seul passage terminal vers
`success`.

Si des pages rejetées existent et que leur résolution, conversion, fusion,
validation ou publication échoue, `ProcessMarking` appelle `MarkingFailed` et
retourne avant l'UPDATE terminal. Le `defer` existant supprime alors tout le
workspace : il ne peut rester ni faux `success`, ni PDF canonique partiel.

Après publication, les PNG de staging restants ne servent plus au QR, à
l'homographie, à la correction ou au mark table. Le pipeline tente donc de les
supprimer. Une erreur de ce cleanup secondaire est loguée mais n'invalide pas
la correction complète puisque l'artefact durable est déjà publié.

## Aucun leftover et durabilité

Sans page rejetée, la primitive retourne un nom vide et ne crée aucun PDF vide.
Un workspace de job repose sur un identifiant DB unique ; si un artefact
canonique existe néanmoins avant publication, la primitive refuse de
l'écraser, empêchant de présenter un résidu comme résultat neuf.

Sur succès, le workspace n'est pas supprimé. `corrected_NOT.pdf` n'appartient
pas à la liste des PDF/Typst temporaires nettoyés et survit donc à la suppression
de l'upload staged système ainsi qu'au cleanup des PNG. Les jobs failed gardent
le cleanup existant ; aucune politique de purge/rétention n'a changé.

## Page résultat, review et legacy

Le GET résultat reste strictement read-only : il appelle seulement
`MarkingArtifactExists`, qui valide par `Lstat` un fichier régulier non symlinké.
Présent, le fichier produit le bouton « Copies / pages non corrigées » ; absent,
aucune URL n'est exposée.

`RegenerateMarkingArtifacts` reste inchangé et ne gère que `corrected.pdf` et
`mark-table.pdf`. `corrected_NOT.pdf` ne dépend ni de `review_revision`, ni de
`artifacts_revision`, ni des décisions humaines.

Les jobs historiques déjà dotés du fichier canonique continuent à être servis.
Il n'existe ni backfill ni régénération legacy.

## Tests

- aucune page rejetée : aucun PDF vide ;
- sélection explicite d'un QR illisible et de toutes les pages d'une copie non
  corrigée, sans inclure une copie corrigée ;
- pages 1, 10 et 12 triées selon l'ordre original ;
- trois PNG colorés publiés dans cet ordre, puis extraits du PDF pour vérifier
  le contenu et l'ordre ;
- PDF canonique non vide avec en-tête `%PDF-` et sources inchangées pendant la
  construction ;
- source manquante ou extérieure : erreur, aucun canonique partiel, aucun
  staging restant ;
- `MarkingArtifactExists` détecte un fichier durable sans modifier contenu,
  taille ou date ;
- view model résultat : artefact absent, aucune URL non-corrected ; artefact
  présent et PDF review-sensitive stale, URL non-corrected toujours disponible.

La position de l'appel avant `CompleteMarkingJobWithResults` est la frontière
qui garantit qu'un échec de génération ne peut produire un faux succès.

## Changements exclus

- migration : non ;
- SQL/sqlc : non ;
- QR, homographie, OpenCV, MeanGray, scoring, ambiguity et review : non ;
- `RegenerateMarkingArtifacts` : non modifié ;
- GET résultat : non modifié.

## Validations

- `./scripts/check.sh` : succès, replay Goose 0001–0040 inclus ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès dans `check.sh`, puis relancé séparément en fin de
  validation.

## Fichiers modifiés ou créés

- `internal/handlers/tools/convertPngToPdf.go` ;
- `internal/handlers/tools/markingUncorrectedPages.go` ;
- `internal/handlers/tools/markingUncorrectedPages_test.go` ;
- `internal/handlers/tools/processMarking.go` ;
- `internal/handlers/marking/resultViewData_test.go` ;
- `docs/audits/marking-corrected-not-modern.md` (rapport local non commité).
