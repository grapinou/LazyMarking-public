# Expériences de reconnaissance de coches

Date : 2026-09-04

## Conclusion

La meilleure métrique observée est la **proportion de pixels dont le niveau de gris est inférieur à 220 dans la ROI intérieure actuelle**, avec décision « cochée » à partir de **10 %** de pixels concernés.

Sur les 80 cases étudiées, cette règle obtient `20 VP`, `60 VN`, `0 FP`, `0 FN`. Elle reconnaît les dix coches de B05, y compris la réponse humainement fausse Q4/A1, et les dix coches de B08. Elle ne transforme aucune des 60 cases vides en coche. Elle reste parfaite avec les ROI testées à `0,40 r`, `0,50 r`, `0,60 r`, carrées ou circulaires.

Ce résultat est expérimental : deux copies seulement ne suffisent pas à fixer un seuil de production. La recommandation est de prototyper cette métrique, sans toucher au scoring, puis de la valider sur un corpus annoté plus large comprenant scans bruités, couleurs différentes, ratures, anneaux décentrés et cases faiblement remplies.

## Sources et méthodologie

Toutes les mesures proviennent exclusivement des pages alignées non annotées du job 8 :

- B05 : `runtime/real/assets/tmp/prof-6e-test/marking-8/aligned/student-exam-31/page-{1,2,3}.png` ;
- B08 : `runtime/real/assets/tmp/prof-6e-test/marking-8/aligned/student-exam-38/page-{1,2,3}.png`.

Les centres et rayons viennent des `student_exam_page_content` persistés. Le mapping global question/réponse/page est celui vérifié dans le diagnostic B05 précédent. Ni `corrected (1).pdf`, ni les pages annotées du job 9 n'ont servi aux mesures.

L'échantillon contient 80 cases : 20 cochées et 60 vides. Les métriques sont évaluées au niveau de la case, indépendamment du corrigé et du score.

Les niveaux de gris expérimentaux utilisent la conversion interprétable `0,299 R + 0,587 G + 0,114 B`, équivalente à l'usage grayscale actuel à l'arrondi près.

## Vérité terrain

La vérité terrain porte sur « cochée/non cochée », pas sur « correcte/incorrecte ».

### B05

Inspection visuelle sans ambiguïté des aligned pages non annotées :

| Question | Case cochée | Correspond au corrigé ? |
|---:|---:|:--|
| Q0 | A1 | oui |
| Q1 | A3 | oui |
| Q2 | A3 | oui |
| Q3 | A2 | oui |
| Q4 | A1 | non, le snapshot attend A3 |
| Q5 | A3 | oui |
| Q6 | A3 | oui |
| Q7 | A3 | oui |
| Q8 | A1 | oui |
| Q9 | A3 | oui |

Il y a donc 10 cases cochées et 30 vides. La cible de reconnaissance est bien 10 coches, même si le score humain correspondant est 9/10.

### B08

Les coches grises sont pleines et visuellement nettes sur les trois pages. La vérité terrain complète retenue est :

| Question | Case cochée |
|---:|---:|
| Q0 | A1 |
| Q1 | A3 |
| Q2 | A2 |
| Q3 | A1 |
| Q4 | A0 |
| Q5 | A3 |
| Q6 | A0 |
| Q7 | A0 |
| Q8 | A3 |
| Q9 | A2 |

B08 contient également 10 cases cochées et 30 cases vides.

## ROI testées

Pour chaque centre et rayon persisté, les variantes suivantes ont été évaluées :

- carré intérieur de demi-côté `0,40 r`, `0,50 r` et `0,60 r` ;
- disque intérieur de rayon `0,40 r`, `0,50 r` et `0,60 r` ;
- rectangle exact de production `center ± floor(radius/2)` pour confirmer les résultats du candidat final.

`0,50 r` correspond approximativement à la ROI courante. Aucun test n'a agrandi la zone jusqu'au rayon complet de l'anneau.

L'effet de l'anneau devient néanmoins visible à `0,60 r` : avec une ROI carrée, le P5 minimal d'une case vide de B08 descend à `194,25`. C'est le seul faux positif observé pour `P5 < 220`, et un avertissement contre les percentiles trop extrêmes associés à une ROI élargie.

## Métriques évaluées

### Baseline

- moyenne de gris dans la ROI exacte actuelle ;
- règle de production `mean_gray < 150`.

### Percentiles sombres

- P5, P10, P20 et P25 ;
- seuils testés : `80, 100, 120, 140, 160, 180, 200, 220`.

### Proportions sombres

- proportions de pixels `<180`, `<200` et `<220` ;
- proportions minimales testées : `1 %, 2 %, 5 %, 10 %, 20 %, 30 %`.

### Contraste local

Le fond est mesuré dans une couronne située entre `1,30 r` et `1,75 r`, donc à l'extérieur de l'anneau imprimé. Ont été comparés :

- médiane du fond moins moyenne intérieure ;
- moyenne du fond moins moyenne intérieure ;
- médiane du fond moins P5/P10/P20/P25 intérieur.

La médiane du fond est privilégiée pour résister à un fragment de texte ou à une poussière dans la couronne.

### Couleur

- saturation moyenne et P90 de saturation ;
- excès de bleu moyen et P90 : `B - (R+G)/2` ;
- distance RGB moyenne au blanc.

## Résultats globaux

Résultats sur les 80 cases, avec la ROI carrée `0,50 r`, sauf baseline qui utilise le rectangle exact de production :

| Méthode | Paramètre | VP | VN | FP | FN | Observation |
|:--|:--|---:|---:|---:|---:|:--|
| moyenne actuelle | `<150` | 10 | 60 | 0 | 10 | perd toutes les coches B05 |
| moyenne recalibrée | `<238` | 20 | 60 | 0 | 0 | intervalle séparateur observé, mais seuil ajusté sur 2 copies |
| P5 | `<220` | 20 | 60 | 0 | 0 | sensible à l'anneau à `0,60 r` carré : 1 FP |
| P10 | `<220` | 20 | 60 | 0 | 0 | séparation large sur les six ROI |
| P20 | `<220` | 20 | 60 | 0 | 0 | séparation large sur les six ROI |
| P25 | `<220` | 20 | 60 | 0 | 0 | marge B05 plus faible que P10/P20 |
| proportion `<180` | `≥10 %` | 19 | 60 | 0 | 1 | une coche B05 trop claire |
| proportion `<200` | `≥10 %` | 20 | 60 | 0 | 0 | parfait sur les six ROI |
| proportion `<220` | `≥10 %` | 20 | 60 | 0 | 0 | parfait, meilleure marge B05 |
| contraste médiane-fond / moyenne | `≥20` | 20 | 60 | 0 | 0 | parfait, mais dépend du voisinage externe |
| saturation moyenne | `≥0,01` | 20 | 60 | 0 | 0 | B08 gris proche de la limite couleur |
| excès bleu moyen | `≥2` | 20 | 60 | 0 | 0 | B08 gris proche de la limite ; dépend de la couleur |

Les VP/VN/FP/FN ci-dessus utilisent les seuils explicites de la grille, pas seulement un seuil optimisé continu.

## Distributions numériques pertinentes

### Rectangle exact de production

| Copie / classe | moyenne gris | P10 | proportion `<220` |
|:--|:--|:--|:--|
| B05 cochée | `174,052..224,355` | `136,431..185,826` | `37,2 %..98,2 %` |
| B05 vide | `253,605..254,910` | `251..255` | `0 %` |
| B08 cochée | `98,586..144,072` | `81,933..121,576` | `93,2 %..100 %` |
| B08 vide | `252,028..255` | `250..255` | `0 %..1,8 %` |

Pour la règle recommandée, la plus faible coche contient `37,2 %` de pixels sous 220 tandis que la case vide la plus bruitée n'en contient que `1,8 %`. Le seuil expérimental à `10 %` laisse donc une marge observée de `27,2` points de pourcentage côté coche et de `8,2` points côté vide.

### Effet des tailles et formes de ROI sur la proportion `<220`

| ROI | B05 cochées min–max | B05 vides min–max | B08 cochées min–max | B08 vides min–max | résultat à `≥10 %` |
|:--|:--|:--|:--|:--|:--|
| carré `0,40 r` | `44,9–100 %` | `0 %` | `92,6–100 %` | `0 %` | 20 VP, 0 FP/FN |
| disque `0,40 r` | `47,1–100 %` | `0 %` | `90,9–100 %` | `0 %` | 20 VP, 0 FP/FN |
| carré `0,50 r` | `37,2–98,3 %` | `0 %` | `93,5–100 %` | `0–1,8 %` | 20 VP, 0 FP/FN |
| disque `0,50 r` | `42,1–100 %` | `0 %` | `92,4–100 %` | `0 %` | 20 VP, 0 FP/FN |
| carré `0,60 r` | `32,5–90,8 %` | `0–4,0 %` | `94,2–100 %` | `0–6,1 %` | 20 VP, 0 FP/FN |
| disque `0,60 r` | `38,2–100 %` | `0 %` | `92,7–100 %` | `0–0,9 %` | 20 VP, 0 FP/FN |

La ROI circulaire limite mieux l'entrée de l'anneau lorsque la taille augmente. Toutefois, la ROI actuelle suffit : aucun changement de géométrie n'est nécessaire pour obtenir le meilleur résultat observé.

### Percentiles, ROI carrée `0,50 r`

| Percentile | B05 cochées | B05 vides | B08 cochées | B08 vides |
|:--|:--|:--|:--|:--|
| P5 | `130,748..177,190` | `249..254` | `74,836..117,238` | `249..255` |
| P10 | `136,431..185,827` | `251..255` | `79,144..121,576` | `250..255` |
| P20 | `150,537..201,290` | `252..255` | `85,523..125,922` | `252..255` |
| P25 | `155,861..207,410` | `253..255` | `87,124..127,802` | `253..255` |

P10 et P20 séparent clairement l'échantillon. P5 détecte les traits les plus fins mais devient plus vulnérable à quelques pixels de l'anneau, tandis que P25 perd de la marge sur la coche B05 la plus faible.

### Contraste local, ROI carrée `0,50 r`

Pour `médiane(fond) - moyenne(intérieur)` :

| Copie / classe | intervalle |
|:--|:--|
| B05 cochée | `28,645..78,948` |
| B05 vide | `-3,568..0,395` |
| B08 cochée | `108,416..153,216` |
| B08 vide | `-2,958..2,972` |

Un seuil de 20 est parfait ici. Cette métrique normalise utilement l'exposition, mais sa couronne extérieure peut rencontrer du texte, une rature ou un défaut de page dans un corpus plus varié. Elle est donc retenue comme métrique secondaire à réévaluer, pas comme premier candidat.

### Couleur, ROI carrée `0,50 r`

| Métrique | B05 cochée | B05 vide | B08 cochée | B08 vide |
|:--|:--|:--|:--|:--|
| saturation moyenne | `0,177..0,565` | `0` | `0,019..0,096` | `0` |
| excès bleu moyen | `23,811..80,511` | `0` | `2,111..7,630` | `0` |

La couleur caractérise très bien B05, mais B08 est presque achromatique. Une règle couleur seule serait fragile pour le crayon gris, les scans désaturés ou d'autres encres. Puisque les métriques grayscale séparent déjà parfaitement ces copies, la couleur n'est pas nécessaire comme décision principale. Elle pourrait devenir un signal auxiliaire après validation sur un corpus plus large.

## Analyse par copie

### B05

Baseline : `0 VP`, `30 VN`, `0 FP`, `10 FN`. Toutes les coches sont perdues.

Proportion `<220 ≥10 %`, ROI exacte actuelle : `10 VP`, `30 VN`, `0 FP`, `0 FN`. La coche la moins dense reste à `37,2 %`, bien au-dessus des cases vides à `0 %`.

La métrique reconnaît Q4/A1 comme cochée, ce qui est volontaire : elle ne connaît pas le corrigé. Le scoring futur conserverait la responsabilité de constater que cette sélection ne correspond pas à A3.

### B08

Baseline : `10 VP`, `30 VN`, `0 FP`, `0 FN`.

Proportion `<220 ≥10 %`, ROI exacte actuelle : résultat identique, `10 VP`, `30 VN`, `0 FP`, `0 FN`. Les coches pleines donnent `93,2–100 %` de pixels sous 220 ; les vides restent à `0–1,8 %`.

## Faux positifs et faux négatifs

La méthode recommandée n'en produit aucun sur cet échantillon.

Cas limites observés avec d'autres paramètres :

- baseline : 10 FN, tous sur B05 ;
- proportion `<180 ≥10 %` : 1 FN B05 pour toutes les formes/tailles testées ;
- `P5 <220` avec carré `0,60 r` : 1 FP B08 causé par l'approche de l'anneau imprimé ;
- aucune des ROI `0,40–0,60 r` ne provoque de FP avec la règle proportion `<220 ≥10 %`.

## Méthode et paramètres recommandés pour un futur jalon

Premier candidat à implémenter dans un prototype/test, pas directement en production :

1. conserver le rectangle exact actuel `center ± floor(radius/2)` ;
2. convertir en gris comme aujourd'hui ;
3. calculer `dark_ratio = count(gray < 220) / pixel_count` ;
4. candidat de décision : `dark_ratio >= 0,10` ;
5. persister ou exposer temporairement `dark_ratio` dans les tests de calibration afin de définir une future bande ambiguë sur cette métrique ;
6. comparer obligatoirement au P10 `<220` et au contraste `médiane(fond)-moyenne(intérieur)` sur le corpus élargi.

Pourquoi ce choix : il détecte un trait fin, exige plus que quelques pixels isolés, reste interprétable, n'utilise pas la couleur, ne nécessite pas d'agrandir la ROI et dispose de la meilleure marge observée entre coches B05 et cases vides.

## Risques et limites

- Seulement deux copies et 80 cases : les performances parfaites ne constituent pas une estimation fiable du taux d'erreur futur.
- Les seuils 220/10 % ont été évalués sur le même petit jeu qui sert à conclure ; une validation indépendante est indispensable.
- Une poussière, une rature centrale, un toner irrégulier ou une compression sévère pourraient augmenter la proportion sombre d'une case vide.
- Une coche encore plus fine que B05 pourrait rester sous 10 %.
- Un décalage d'homographie supérieur à ceux observés pourrait déplacer le trait hors de la ROI actuelle.
- Les métriques de contraste dépendent du choix de la couronne et peuvent rencontrer le texte voisin.
- Les métriques couleur dépendent du scanner et ne couvrent pas correctement tous les crayons gris.
- L'étude ne mesure aucune note finale et ne valide aucun changement de scoring.

## Recommandation pour le prochain jalon

Construire un corpus de vérité terrain au niveau case à partir de plusieurs copies 6eA/6eB, incluant explicitement coches pâles, pleines, croix, ratures et cases vides bruitées. Évaluer la règle proposée sans réajuster ses paramètres, puis choisir seuil et bande ambiguë sur un jeu de calibration distinct du jeu de validation. Une modification de production ne devrait être décidée qu'après cette validation hors échantillon.

## Fichiers créés

Versionné :

- `docs/reports/2026-09-04-answer-detection-experiments.md`.

Hors Git, sous `runtime/diagnostics/answer-detection/` :

- `analyze.py` : extraction et évaluation reproductible ;
- `features.csv` : caractéristiques par case/ROI ;
- `fixed-grid.csv` : résultats de la grille explicite ;
- `optimized-thresholds.csv` : seuil séparateur optimal par métrique, utilisé uniquement pour étudier la séparation ;
- `results.txt` : sortie synthétique brute.

Aucun fichier Go ou fichier de production n'a été créé ou modifié.

## Commandes exécutées

```sh
python3 runtime/diagnostics/answer-detection/analyze.py \
  > runtime/diagnostics/answer-detection/results.txt

sqlite3 testdata/real/app.db \
  "SELECT content FROM student_exam_content WHERE student_exam_id IN (31,38);"

sqlite3 testdata/real/app.db \
  "SELECT page,content FROM student_exam_page_content
   WHERE student_exam_id IN (31,38) ORDER BY student_exam_id,page;"

git status --short
git diff --check
```

## Validations

- source des mesures contrôlée : uniquement `marking-8/aligned/student-exam-31` et `student-exam-38` ;
- cardinalité contrôlée par le script : 40 cases par copie ;
- outils temporaires placés sous `runtime/diagnostics/`, répertoire ignoré par Git ;
- aucune utilisation de `corrected (1).pdf` dans le script d'analyse ;
- aucune modification de production, base, scoring, workflow ou UX ;
- `git status --short` et `git diff --check` exécutés après finalisation.
