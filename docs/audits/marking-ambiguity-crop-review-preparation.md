# Préparation de la revue locale des crops d'ambiguïté

## Export réalisé

Le harness batch réel a exporté un corpus local anonyme depuis les mêmes pages
alignées non annotées que celles utilisées par `GetAnswerDetections`.

| Mesure | Résultat |
|---|---:|
| Candidats A — distance à 150 ≤ 30 | 15 |
| Candidats B — MeanGray < 75 et écart-type ≥ 80 | 1 |
| Candidats C — extrêmes inter-copies | 6 |
| Appartenances fusionnées | 2 |
| Crops uniques | 20 |
| MeanGray des crops | 59,14 à 191,23 |
| Écart-type des crops | 24,09 à 103,29 |

La catégorie C contient trois exemples représentatifs autour de la médiane de
la copie dont la médiane checked est la plus haute, et trois autour de la
médiane unchecked la plus basse. Elle reste ainsi limitée à 3 exemples par
extrême. Une détection appartenant à plusieurs catégories n'est exportée qu'une
fois ; sa catégorie est combinée dans le manifest.

## Source et cadrage

- Source crop = page aligned non annotée staged par `MarkingStudentExam` : oui.
- Scan brut utilisé : non.
- PDF corrigé utilisé : non.
- Image réécrite par `DrawMarking` utilisée : non.
- Reconstruction dédiée utilisée : non.

Le rectangle de mesure reste `centre ± radius/2`. Le crop d'inspection est
centré sur le même cercle avec un demi-côté de deux rayons, et au minimum 30
pixels, afin d'ajouter un contexte limité autour de la case sans aller chercher
un en-tête, un QR ou une zone éloignée.

## Anonymisation et artefacts locaux

Le répertoire local utilisé est :

`/home/sighto/Documents/lz_pdf_test/calibration-review`

Il est extérieur au dépôt. L'export refuse tout chemin résolu dans le dépôt et
ne nettoie que trois formes de noms strictement contrôlées :
`candidate-NNNN.png`, `manifest.csv` et `review-contact-sheet.html`. Il ne
supprime aucun autre fichier ou répertoire.

- Identités exportées : non.
- Identifiants de copie exportés : non.
- Questions ou réponses exportées : non.
- Noms de crops : `candidate-0001.png` à `candidate-0020.png`.
- Manifest créé : oui, 20 lignes de candidats.
- Contact sheet créée : oui, 20 liens PNG relatifs.
- Fichiers inattendus dans le répertoire : aucun.

Le manifest contient uniquement : `candidate`, `category`, `mean_gray`,
`detected_state`, `distance_to_150`, `stddev`, `dark_pixel_ratio`, `min_gray`,
`max_gray`, `human_label` et `human_comment`. Les deux dernières colonnes sont
vides. Aucun jugement humain n'a été prérempli.

La contact sheet affiche uniquement le numéro anonyme, la catégorie, MeanGray,
l'état automatique, la distance au seuil et l'écart-type. Elle ne porte aucun
jugement automatique.

## Impact production

- Production modifiée : non.
- `ambiguity_delta` modifié : non ; il reste `NULL`.
- Seuil 150 modifié : non.
- `GetAnswerDetections`, scores et PDF production modifiés : non.
- Données privées ajoutées à Git : non.

## Validation

- Export réel opt-in : succès, 20 crops uniques.
- Source DB privée : restée byte-for-byte inchangée.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.
