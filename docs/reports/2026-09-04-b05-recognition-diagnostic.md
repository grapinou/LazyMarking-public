# Diagnostic de reconnaissance — copie 6eB B05

Date : 2026-09-04

## Résumé

Le résultat `0 / 10` de B05 ne vient ni du barème, ni du snapshot, ni d'une permutation des réponses. Pour le job courant, les 40 détections valent `0`. Les dix marques visibles de B05 sont fines et bleu clair ; leurs `mean_gray` dans la ROI de production sont tous supérieurs au seuil strict de `150` (`174,515` à `224,465`). Aucune n'entre dans la zone ambiguë `[145, 155]`. Le score `0 / 10` est donc la conséquence déterministe de ces états effectifs tous nuls.

Le contrôle visuel donne neuf réponses justes sur dix : Q4 est cochée sur « ouvert » alors que le snapshot attend « fermé ». La note humaine cohérente avec le snapshot est donc `9 / 10`.

Un second problème de provenance affecte spécifiquement le dernier job : les pages alignées persistées du job 9 contiennent déjà les annotations d'une correction antérieure (croix rouges, scores et cercles verts). Le job 9 a donc reçu `testdata/real/scans/corrected (1).pdf`, ou un fichier byte-à-byte/logiquement équivalent, et non le scan vierge. Cette conclusion repose sur ses 12 pages, leur séquence QR exacte B05/B02/B01/B08 et la présence des annotations avant `DrawMarking`. La base ne persiste toutefois ni le nom ni le hash du fichier uploadé ; le nom ne peut donc pas être démontré par une colonne d'audit.

Le scan brut non annoté correspondant est `testdata/real/scans/6eB.pdf` : B05 se trouve aux pages physiques 31–33 et B08 aux pages 79–81, confirmées par décodage des QR. Les `mean_gray` demandés ci-dessous sont exclusivement ceux persistés depuis les `marking_aligned_pages` du job 9, pas des mesures faites sur `corrected (1).pdf`.

## Identifiants et état persisté

### Job courant 6eB

| Champ | Valeur |
|:--|:--|
| `marking_job_id` | `9` |
| `user_id` | `1` |
| `exam_generated_id` | `2` |
| `status` / `status_pdf` | `success` / `success` |
| pages / copies reconnues | `12` / `4` |
| `result_schema_version` | `1` |
| `marking_algorithm_version` | `1` |
| `detection_threshold` | `150.0` |
| `ambiguity_delta` | `5.0` |
| zone ambiguë inclusive | `[145.0, 155.0]` |
| `review_revision` / `artifacts_revision` | `0` / `0` |
| fin du job | `2026-09-04 13:24:38` |

### B05

| Champ | Valeur |
|:--|:--|
| élève | `Élève B05`, classe `6eB` |
| `student_exam_id` | `31` |
| `marking_copy_result.id` | `241` |
| outcome | `corrected` |
| pages attendues / détectées | `3 / 3` |
| `score_half_units` | `0` |
| total | `10` points |
| score affiché | `0 / 10` |
| failure | aucune |
| fin de copie | `2026-09-04 13:24:36` |

Les trois pages alignées persistées sont les IDs `643`, `644`, `645`, aux dimensions `2480 × 3508`, avec respectivement les SHA-256 `e52052b2ff8673eb54499e8624dd7cd22da7996227500548afd2ce13688e8287`, `8ded477c522449a0372dd90ff6ad867073d7c09ea143e1fceab32f0b53eb16c5` et `3c68503d1c10d4f746c7d51da11344a4d411efd1b843c215c7aad9607ae9838d`.

### Copie de comparaison B08

| Champ | Valeur |
|:--|:--|
| `student_exam_id` | `38` |
| `marking_copy_result.id` | `242` |
| outcome | `corrected` |
| score | `14` demi-points, soit `7 / 10` |
| pages attendues / détectées | `3 / 3` |

## Reconstitution question par question de B05

`Attendus` est le vecteur du snapshot dans l'ordre des `answer_index`. Il n'existe aucune revue sur le job 9 : `reviewed_state` vaut donc « — » et `effective_state = detected_state` partout.

| Q | Pts | Attendus | A | `mean_gray` | détecté | revu | effectif | état Q | score Q |
|---:|---:|:---:|---:|---:|---:|:---:|---:|:---|---:|
| 0 | 1 | `0100` | 0 | 254.068 | 0 | — | 0 | incorrect | 0/1 |
| 0 | 1 | `0100` | 1 | 174.515 | 0 | — | 0 | incorrect | 0/1 |
| 0 | 1 | `0100` | 2 | 253.585 | 0 | — | 0 | incorrect | 0/1 |
| 0 | 1 | `0100` | 3 | 254.248 | 0 | — | 0 | incorrect | 0/1 |
| 1 | 1 | `0001` | 0 | 254.608 | 0 | — | 0 | incorrect | 0/1 |
| 1 | 1 | `0001` | 1 | 254.065 | 0 | — | 0 | incorrect | 0/1 |
| 1 | 1 | `0001` | 2 | 254.361 | 0 | — | 0 | incorrect | 0/1 |
| 1 | 1 | `0001` | 3 | 194.310 | 0 | — | 0 | incorrect | 0/1 |
| 2 | 1 | `0001` | 0 | 254.176 | 0 | — | 0 | incorrect | 0/1 |
| 2 | 1 | `0001` | 1 | 254.639 | 0 | — | 0 | incorrect | 0/1 |
| 2 | 1 | `0001` | 2 | 254.361 | 0 | — | 0 | incorrect | 0/1 |
| 2 | 1 | `0001` | 3 | 183.750 | 0 | — | 0 | incorrect | 0/1 |
| 3 | 1 | `0010` | 0 | 251.787 | 0 | — | 0 | incorrect | 0/1 |
| 3 | 1 | `0010` | 1 | 210.018 | 0 | — | 0 | incorrect | 0/1 |
| 3 | 1 | `0010` | 2 | 214.370 | 0 | — | 0 | incorrect | 0/1 |
| 3 | 1 | `0010` | 3 | 190.507 | 0 | — | 0 | incorrect | 0/1 |
| 4 | 1 | `0001` | 0 | 208.978 | 0 | — | 0 | incorrect | 0/1 |
| 4 | 1 | `0001` | 1 | 195.675 | 0 | — | 0 | incorrect | 0/1 |
| 4 | 1 | `0001` | 2 | 251.102 | 0 | — | 0 | incorrect | 0/1 |
| 4 | 1 | `0001` | 3 | 254.015 | 0 | — | 0 | incorrect | 0/1 |
| 5 | 1 | `0001` | 0 | 253.864 | 0 | — | 0 | incorrect | 0/1 |
| 5 | 1 | `0001` | 1 | 254.265 | 0 | — | 0 | incorrect | 0/1 |
| 5 | 1 | `0001` | 2 | 252.773 | 0 | — | 0 | incorrect | 0/1 |
| 5 | 1 | `0001` | 3 | 196.477 | 0 | — | 0 | incorrect | 0/1 |
| 6 | 1 | `0001` | 0 | 253.948 | 0 | — | 0 | incorrect | 0/1 |
| 6 | 1 | `0001` | 1 | 254.183 | 0 | — | 0 | incorrect | 0/1 |
| 6 | 1 | `0001` | 2 | 254.293 | 0 | — | 0 | incorrect | 0/1 |
| 6 | 1 | `0001` | 3 | 196.430 | 0 | — | 0 | incorrect | 0/1 |
| 7 | 1 | `0001` | 0 | 254.691 | 0 | — | 0 | incorrect | 0/1 |
| 7 | 1 | `0001` | 1 | 254.192 | 0 | — | 0 | incorrect | 0/1 |
| 7 | 1 | `0001` | 2 | 253.648 | 0 | — | 0 | incorrect | 0/1 |
| 7 | 1 | `0001` | 3 | 224.465 | 0 | — | 0 | incorrect | 0/1 |
| 8 | 1 | `0100` | 0 | 253.733 | 0 | — | 0 | incorrect | 0/1 |
| 8 | 1 | `0100` | 1 | 193.707 | 0 | — | 0 | incorrect | 0/1 |
| 8 | 1 | `0100` | 2 | 254.020 | 0 | — | 0 | incorrect | 0/1 |
| 8 | 1 | `0100` | 3 | 254.252 | 0 | — | 0 | incorrect | 0/1 |
| 9 | 1 | `0001` | 0 | 253.893 | 0 | — | 0 | incorrect | 0/1 |
| 9 | 1 | `0001` | 1 | 254.513 | 0 | — | 0 | incorrect | 0/1 |
| 9 | 1 | `0001` | 2 | 254.102 | 0 | — | 0 | incorrect | 0/1 |
| 9 | 1 | `0001` | 3 | 200.165 | 0 | — | 0 | incorrect | 0/1 |

Lecture synthétique des marques visibles :

| Q | réponse visiblement cochée | attendu | `mean_gray` | décision |
|---:|---:|---:|---:|:--|
| 0 | A1 | A1 | 174.515 | faux négatif |
| 1 | A3 | A3 | 194.310 | faux négatif |
| 2 | A3 | A3 | 183.750 | faux négatif |
| 3 | A2 | A2 | 214.370 | faux négatif |
| 4 | A1 | A3 | 195.675 | non détectée, mais réponse humainement fausse |
| 5 | A3 | A3 | 196.477 | faux négatif |
| 6 | A3 | A3 | 196.430 | faux négatif |
| 7 | A3 | A3 | 224.465 | faux négatif |
| 8 | A1 | A1 | 193.707 | faux négatif |
| 9 | A3 | A3 | 200.165 | faux négatif |

## Vérification du mapping

Le snapshot B05 contient 10 questions et 40 réponses. Les `marking_answer_detections` contiennent exactement 10 `question_index` consécutifs (`0..9`) et, pour chacun, quatre `answer_index` consécutifs (`0..3`). Les réponses du snapshot, les détections et les positions de `page_content` ont donc la même cardinalité.

Le parcours réel est : pages triées par `page_exam`, puis réponses dans l'ordre stocké dans chaque `page_content`, concaténées dans un vecteur global. `BuildMarkingCopyResult` redécoupe ce vecteur par le nombre de réponses de chaque question du snapshot.

Les offsets observés pour B05 sont :

| Page | indices globaux | correspondance |
|---:|:---|:---|
| 1 | `0..17` | Q0 A0 à Q3 A3, puis Q4 A0–A1 |
| 2 | `18..33` | Q4 A2–A3, Q5/Q6/Q7 complets, puis Q8 A0–A1 |
| 3 | `34..39` | Q8 A2–A3, puis Q9 complet |

Q4 et Q8 passent donc effectivement d'une page à l'autre. Ce n'est pas un décalage : les minima/traits visibles se retrouvent aux bons couples question/réponse, y compris Q4 A1 sur la page 1, Q5 A3 sur la page 2 et Q9 A3 sur la page 3. L'ordre vertical des questions, l'ordre gauche-droite puis haut-bas des réponses, les transitions de page et les offsets sont cohérents.

Le snapshot est également le bon : libellés, variante des réponses, identité B05 et réponses attendues correspondent aux pages. Q4 attend bien « fermé » (A3), alors que l'élève a marqué « ouvert » (A1).

## Images, ROI et homographie

Les artefacts hors Git sont sous `runtime/diagnostics/b05/` :

- `b05/page-{1,2,3}-roi-overview.png` : pages alignées du job 9, cercle nominal vert, ROI exacte rouge et étiquette `Q/A/mean_gray` ;
- `b05/crops/qXX-aY.png` : 40 crops B05, avec la ROI exacte en rouge ;
- `b08/...` : mêmes artefacts pour B08 ;
- `raw-identification/` : rendus temporaires et cartes QR ayant servi à identifier les PDF/pages.

La ROI de production est exactement le carré centré `center ± radius/2`. Avec des rayons de 19 ou 20 pixels, sa taille est donc généralement `18 × 18` ou `20 × 20` pixels. Sur B05, le trait bleu est fin, clair et parfois principalement vertical ; la ROI contient beaucoup de fond blanc. Les crops montrent directement que les marques sont présentes tout en donnant une moyenne supérieure à 150.

L'homographie n'apparaît pas défaillante. Sur les pages non annotées du job 8, une estimation du centre des anneaux noirs dans un voisinage local donne pour B05 un décalage médian d'environ `(+0,16 px, -0,08 px)`, avec des extrêmes d'environ `-2,43..+1,70 px` en X et `-2,51..+1,70 px` en Y. B08 est du même ordre. Il n'existe donc pas de translation ou déformation capable d'expliquer dix faux négatifs.

Attention de provenance : les aligned pages du job 9 montrent les annotations d'une correction précédente. La méthode `StageMarkingAlignedPage` copie pourtant l'homographie avant l'appel à `DrawMarking`; ces annotations étaient donc déjà dans l'entrée du job. La comparaison avec le job 8, dont les aligned pages B05 sont non annotées, confirme ce point.

## Analyse numérique et comparaison B08

Pour B05/job 9 : 40 détections, toutes `detected_state=0`, minimum `174,515`, maximum `254,691`, moyenne `235,969`. Il y a zéro détection dans `[145,155]`, donc aucune ambiguïté possible selon la règle persistée.

Pour B08/job 9 :

| classe LazyMarking | N | min | max | moyenne |
|:--|---:|---:|---:|---:|
| cochée (`1`) | 10 | 81.552 | 132.273 | 109.650 |
| non cochée (`0`) | 30 | 253.740 | 255.000 | 254.377 |

B08 remplit largement les cases au crayon gris, avec une surface sombre dense au centre. Les deux groupes sont nettement séparés par le seuil 150. B05 utilise au contraire un trait bleu clair et peu couvrant : les valeurs des marques visibles se placent entre 174,515 et 224,465, donc du côté « vide » sans être proches du seuil.

Les jobs 2 à 8, réalisés sur les pages non annotées, donnent exactement le même résultat B05 : minimum `173,558`, zéro case détectée et score nul. Le job 9 annoté modifie certaines mesures, mais ne crée donc pas la cause initiale. Il la reproduit.

## Hypothèses

### Confirmées

1. **Faux négatifs dus à la décision sur `mean_gray`** : confirmé. Les neuf réponses humainement justes sont visibles mais ont toutes une moyenne supérieure à 150 ; elles deviennent `0`, puis les dix questions deviennent `incorrect`, puis le total devient `0`.
2. **Autre cause de données sur le job courant** : confirmé. Le job 9 retraite une sortie annotée de correction. C'est un problème réel de provenance de ce run, mais pas la cause historique du zéro de B05 puisque les jobs 2–8 sur scan vierge donnaient déjà zéro.

### Non confirmée mais contributrice possible

3. **ROI trop petite** : la ROI `radius/2` échantillonne une petite zone centrale et les traits B05 sont fins ; cela contribue vraisemblablement à une moyenne claire. Ce diagnostic ne teste aucun nouveau rectangle et ne permet donc pas de conclure qu'un agrandissement est la bonne correction.

### Écartées par les données disponibles

4. **Homographie/décalage géométrique majeur** : écarté ; centres à quelques pixels au plus, sans biais systématique significatif.
5. **Ordre ou mapping incorrect** : écarté ; cardinalités, offsets, passages de page et correspondances visuelles sont cohérents.
6. **Mauvais corrigé/snapshot** : écarté ; identité, questions, variantes et réponses attendues correspondent à B05.
7. **Erreur du scoring** : écartée ; avec 40 états effectifs à zéro face à dix questions ayant chacune une réponse attendue, `0 / 10` est le résultat prévu du barème actuel.

## Cause racine retenue

La cause racine du faux `0 / 10` est une incompatibilité entre le style de marquage réel de B05 (trait bleu clair, fin et peu couvrant) et la caractéristique de reconnaissance actuelle (moyenne de gris d'une petite ROI centrale, seuil fixe strict `< 150`). Les marques restent très au-dessus du seuil et très loin de la bande ambiguë, ce qui explique simultanément les faux négatifs et l'absence de proposition de revue.

Le retraitement d'un PDF annoté est une anomalie indépendante du job 9 et doit être évité lors des prochains essais, mais il n'explique pas à lui seul le cas B05.

## Recommandation pour le prochain jalon

Conserver B05 et B08 comme paire de référence et évaluer, hors production puis par tests reproductibles, des caractéristiques plus robustes à la couleur et au faible taux de remplissage : distribution/pixels sombres, contraste relatif avec l'anneau ou le voisinage, composante couleur/saturation, et variantes de ROI. Toute proposition devra être comparée sur les aligned pages non annotées du job 8 et vérifier les faux positifs, notamment les anneaux imprimés. Il faut aussi empêcher ou signaler l'upload d'une sortie déjà annotée avant de refaire une campagne de calibration.

## Commandes exécutées

Principales lectures SQL :

```sh
sqlite3 testdata/real/app.db \
  "SELECT ... FROM marking_jobs WHERE id=9;"
sqlite3 testdata/real/app.db \
  "SELECT ... FROM marking_copy_results ... WHERE marking_job_id=9;"
sqlite3 testdata/real/app.db \
  "SELECT ... FROM marking_question_results
   JOIN marking_answer_detections ...
   LEFT JOIN marking_answer_reviews ... WHERE copy_result_id=241;"
sqlite3 testdata/real/app.db \
  "SELECT page,content FROM student_exam_page_content
   WHERE student_exam_id IN (31,38) ORDER BY student_exam_id,page;"
```

Identification et inspection :

```sh
pdfinfo testdata/real/scans/*.pdf
pdftoppm -r 300 -png testdata/real/scans/6eB.pdf ...
go run ./runtime/diagnostics/decodeqr ...
sha256sum runtime/real/assets/tmp/prof-6e-test/marking-9/aligned/student-exam-31/page-*.png
python3 runtime/diagnostics/b05/build_roi_diagnostics.py
```

## Fichiers créés ou modifiés

- créé et versionnable : `docs/reports/2026-09-04-b05-recognition-diagnostic.md` ;
- créés hors Git : `runtime/diagnostics/b05/` (crops, overviews, rendus PDF et utilitaires temporaires) ;
- aucun fichier de production, migration, schéma, scoring, seuil, ROI ou UX modifié.

## Validation

Aucun code permanent n'a été ajouté. Les contrôles applicables au livrable sont :

- intégrité des aligned pages B05 : les trois SHA-256 recalculés correspondent aux valeurs persistées ;
- cardinalité : 40 réponses snapshot = 40 positions de page = 40 détections ;
- revues job 9 : `0` ; ambiguïtés B05 dans `[145,155]` : `0` ;
- identification QR du brut : B05 pages 31–33 et B08 pages 79–81 de `6eB.pdf` ;
- `git diff --check` : à exécuter après finalisation du rapport.

Les tests Go et `scripts/check.sh` ne sont pas requis puisqu'aucun code permanent n'a été modifié.
