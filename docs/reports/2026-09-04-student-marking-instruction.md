# Consigne de marquage sur les QCM générés

Date : 2026-09-04

## Résumé

La consigne suivante est désormais affichée sur les documents réellement destinés à être remplis par les élèves :

> Répondez au stylo bleu ou noir. Cochez nettement la case choisie.

Elle apparaît dans l'en-tête fixe de chaque page d'un examen QCM standard et sous les champs d'identité de chaque mini-QCM paysage. Elle n'est pas rendue dans les previews administratives.

Le contrôle sur le QCM 6eB de référence conserve trois pages. Les 50 zones contenant les cercles de questions et de réponses sont identiques pixel pour pixel aux références antérieures. Les repères, la zone réservée au QR code et le QR effectivement ajouté et décodé restent valides.

## Templates audités

### Template standard

`internal/config/ref_qcm.txt` est la base commune copiée par `TypstWriter`. Ses usages sont :

- `config.ExamQCM` : examen individualisé destiné à l'élève ;
- `config.PreviewQCM` : preview administrative du QCM ;
- `config.PreviewQuestion` : preview administrative d'une question ou variante.

La génération d'examen par `BuildQcmStudentCtx` appelle `TypstWriter` avec `ExamQCM`, exporte les pages, détecte les cercles sur les pages natives, ajoute ensuite le QR code et persiste les références de correction.

Le template commun contient désormais un bloc conditionnel piloté par `show_marking_instruction`. `TypstWriter` le fixe à `true` uniquement pour `ExamQCM`, et à `false` pour les previews. Les previews administratives conservent donc leur rendu sans consigne élève.

### Paysage et mini-QCM

`internal/config/ref_qcm_landscape.txt` est utilisé par deux chemins :

- `TypstWriterLandscape` : preview paysage administrative ;
- `TypstWriterLandscapeAllContent` associé à `TypstLandscapeContent` : mini-QCM paysage imprimable, composé pour les élèves.

Le template paysage de base n'a pas été modifié. La consigne est ajoutée par `TypstLandscapeContent`, donc seulement dans chaque contenu élève du mini-QCM. La preview paysage administrative ne la reçoit pas.

Le mini-QCM n'appartient pas au pipeline de correction automatique : il ne génère ni QR individualisé, ni snapshots de cercles, ni références de pages consommées par l'homographie. Il a néanmoins été compilé et contrôlé visuellement.

## Emplacement et rendu

### Examen standard

La consigne se trouve dans le cartouche central de l'en-tête, après le nom de l'examen, l'identité et la classe. Elle utilise une taille de 8 points et un poids normal, contre 16 points gras pour les informations principales.

Ce choix la rend lisible mais secondaire. L'en-tête se trouve dans la marge supérieure fixe de 5 cm : son enrichissement ne participe pas au flux des questions et ne réduit pas leur espace de composition. La zone QR reste dans la colonne gauche et les repères de page dans la colonne droite.

La consigne est répétée sur chaque page, comme les autres éléments d'en-tête. Cela évite qu'une page séparée ou distribuée isolément perde l'instruction.

### Mini-QCM paysage

La consigne apparaît immédiatement sous « Prénom + Nom » et « Classe », avant la première question de chaque exemplaire. Elle utilise également 8 points. Ce placement est naturel pour un document court remis à l'élève et ne concurrence pas les questions.

## Pagination et mise en page

Trois documents ont été générés sous `runtime/diagnostics/student-marking-instruction/` :

| Contrôle | Format | Pages | Résultat visuel |
|:--|:--|--:|:--|
| `short.pdf` | A4 portrait, une question | 1 | consigne lisible, aucun chevauchement |
| `class-6e.pdf` | A4 portrait, QCM réel 6eB représentatif | 3 | en-têtes, questions et pagination cohérents |
| `control_miniqcm_landscape.pdf` | A4 paysage, deux contenus élève | 3 | consigne lisible dans chaque contenu |

Le QCM 6eB antérieur comportait trois pages ; la nouvelle génération en comporte toujours trois. Pour le mini-QCM, une compilation privée sans la ligne de consigne comporte également trois pages : aucune page supplémentaire n'est créée.

L'inspection visuelle des premières et dernières pages ne montre aucun chevauchement entre la consigne, l'identité, les repères, les questions ou les réponses.

## QR code et repères

La génération standard a été contrôlée à deux niveaux :

1. les zones de la page native qui contiennent le carré réservé au QR, les repères droits et le pied de page sont strictement identiques aux références persistées antérieures ;
2. un QR de contrôle `{student_exam_id: 31, page_exam: 1}` a été créé, collé par le pipeline existant, puis décodé avec succès avec exactement ces valeurs.

La consigne est située dans la colonne centrale ; elle ne masque ni ne déplace la zone QR de gauche.

## Cases et contrat de détection

La nouvelle page 6eB a été comparée aux PNG natifs pré-QR persistés pour `student_exam_id=31`. Pour les trois pages :

- dimensions inchangées : 2 480 × 3 508 pixels à 300 ppp ;
- 50 crops contrôlés autour de tous les cercles de questions et réponses ;
- différence maximale observée dans ces crops : **0 niveau de pixel** ;
- positions et rayons issus de `student_exam_page_content` toujours superposables exactement.

Le changement ne touche aucune structure Go de cercle, aucun calcul de coordonnées, aucune détection, aucune homographie et aucun snapshot. Le corps étant inchangé et placé sous une marge fixe, les données `CircleValidated` et les références de correction restent valides.

Les examens déjà générés restent compatibles : leurs snapshots et références persistés ne sont ni modifiés ni régénérés par ce jalon. Les nouveaux examens reçoivent simplement la consigne dans leur en-tête.

## Tests

Les tests ciblés vérifient :

- l'activation de `show_marking_instruction` pour `ExamQCM` ;
- la présence du texte souhaité dans le document élève standard ;
- la désactivation du rendu pour une preview administrative de question ;
- la présence de la consigne dans le contenu du mini-QCM paysage ;
- les contrats d'échappement Typst existants.

Les validations exécutées sont :

| Validation | Résultat |
|:--|:--|
| Tests Typst ciblés | succès |
| Compilation Typst des trois contrôles | succès |
| Export PNG à 300 ppp | succès |
| Comparaison géométrique aux références 6eB | 50/50 crops identiques |
| Ajout et décodage d'un QR réel | succès |
| `go test ./...` | succès |
| `./scripts/check.sh` | succès |
| `git diff --check` | succès |

## Fichiers modifiés

- `internal/config/ref_qcm.txt` : bloc conditionnel de consigne dans l'en-tête ;
- `internal/handlers/tools/typstWriter.go` : activation uniquement pour les examens élèves ;
- `internal/handlers/tools/typstLandscapeContent.go` : consigne du mini-QCM ;
- `internal/handlers/tools/typstProducerEscape_test.go` : tests de présence et de portée ;
- `docs/reports/2026-09-04-student-marking-instruction.md` : présent rapport.

Les scripts, PDF, PNG, sources Typst et résultats de comparaison utilisés pour le contrôle restent sous `runtime/diagnostics/student-marking-instruction/` et sont ignorés par Git.

## Points d'attention

- La consigne prépare le domaine nominal mais ne garantit pas que chaque élève la suivra ; le prochain test réel devra encore mesurer la charge de revue.
- Le texte est répété sur les pages standard. Cette répétition est volontaire pour les feuilles séparables.
- Le mini-QCM paysage peut naturellement paginer différemment selon le nombre et la longueur des questions ; sur le cas représentatif contrôlé, la consigne n'ajoute aucune page.
- Aucun changement n'a été apporté au détecteur, à la politique de confiance, aux seuils, au scoring, à la base, aux migrations ou aux écrans d'administration.
