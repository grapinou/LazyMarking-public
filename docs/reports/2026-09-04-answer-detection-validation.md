# Validation indépendante de la détection des réponses

Date : 2026-09-04

## Résumé

La règle candidate a été évaluée sans modifier ses paramètres après le début de la validation : pixels sombres définis par `gray < 220`, puis case déclarée cochée si `dark_ratio >= 0.10`. B05 et B08, qui avaient servi au choix de cette règle, sont conservées comme calibration historique et exclues des résultats principaux.

Le jeu indépendant comprend 16 copies, 8 issues de 6eA (job 7) et 8 de 6eB (job 8), soit 640 cases. La vérité terrain visuelle contient 172 cases cochées et 468 cases vides. La méthode actuelle obtient 119 VP, 468 VN, 0 FP et 53 FN. La candidate obtient 169 VP, 460 VN, 8 FP et 3 FN.

La candidate améliore fortement le rappel, de 69,19 % à 98,26 %, mais la séparation n'est pas parfaite. Les huit faux positifs sont principalement liés à l'anneau imprimé ou à des marques parasites qui entrent dans les coins de la ROI carrée. Les trois faux négatifs sont des traits rose ou bleu extrêmement pâles. La validation est donc insuffisante pour remplacer directement la méthode de production, mais suffisamment encourageante pour un prototype contrôlé lors d'un prochain jalon.

## Sources et périmètre

Toutes les mesures et toutes les annotations ont été réalisées à partir des pages alignées non annotées persistées :

- job 7 : `runtime/real/assets/tmp/prof-6e-test/marking-7/aligned/` ;
- job 8 : `runtime/real/assets/tmp/prof-6e-test/marking-8/aligned/`.

Le PDF corrigé n'a pas été utilisé. Les positions et rayons viennent des snapshots persistés (`student_exam_page_content`). Aucun état détecté, score ou corrigé n'a servi à établir la vérité terrain.

## Sélection des copies

Les identifiants ci-dessous sont uniquement des `student_exam_id`. Aucune donnée personnelle n'est incluse dans les artefacts versionnés.

| Jeu | Classe/job | `student_exam_id` | Motif de sélection |
|---|---:|---:|---|
| Calibration | 6eB / 8 | 31 | B05, traits bleus fins et clairs, échec connu de la méthode actuelle |
| Calibration | 6eB / 8 | 38 | B08, coches au crayon plus sombres, comparaison historique |
| Validation | 6eA / 7 | 26 | copie ayant produit une ambiguïté et marques de force variable |
| Validation | 6eA / 7 | 8 | plusieurs coches faibles sous-détectées par la méthode actuelle |
| Validation | 6eA / 7 | 29 | sélections multiples et marque parasite |
| Validation | 6eA / 7 | 14 | sélections multiples et anneaux fortement présents dans la ROI |
| Validation | 6eA / 7 | 11 | copie bien reconnue par la méthode actuelle |
| Validation | 6eA / 7 | 19 | une coche faible et une marque parasite |
| Validation | 6eA / 7 | 3 | sélections multiples et anneaux proches de la ROI |
| Validation | 6eA / 7 | 15 | sélections multiples, copie bien reconnue |
| Validation | 6eB / 8 | 44 | traits bleus clairs, copie fortement sous-détectée |
| Validation | 6eB / 8 | 32 | marques rouges, copie fortement sous-détectée |
| Validation | 6eB / 8 | 51 | marques vertes, dont une question laissée vide |
| Validation | 6eB / 8 | 58 | marques rouges, copie bien reconnue et cas précédemment ambigu |
| Validation | 6eB / 8 | 45 | traits roses extrêmement pâles et une question vide |
| Validation | 6eB / 8 | 53 | traits bleus extrêmement pâles et une question vide |
| Validation | 6eB / 8 | 47 | traits violets forts, copie bien reconnue |
| Validation | 6eB / 8 | 42 | traits bleus forts et sélections multiples |

Cet échantillon privilégie la diversité des instruments, de l'intensité, des sélections multiples, des ambiguïtés historiques et de la qualité de reconnaissance actuelle. Il ne s'agit pas d'un tirage aléatoire représentatif de toute la population.

## Construction de la vérité terrain

Une planche de contact de 40 crops a été générée pour chacune des 18 copies. Chaque case est uniquement légendée avec `student_exam_id`, `question_index` et `answer_index`. Aucune prédiction de LazyMarking, valeur numérique, réponse attendue ou note n'est affichée sur ces planches, afin de limiter le biais de confirmation.

Les cases ont ensuite été classées visuellement en `human_checked=1` ou `human_checked=0`. Les questions vides et les sélections multiples ont été conservées comme telles : la vérité terrain porte sur chaque case, jamais sur la correction de la réponse. Le fichier local `ground_truth.csv` contient les 720 cases des jeux de calibration et de validation, ainsi que la provenance nécessaire à la reproductibilité.

Les planches et le CSV restent sous `runtime/diagnostics/answer-detection-validation/`, hors Git.

## Méthodes évaluées

Pour chaque réponse, la ROI de production est conservée exactement : carré centré sur la position persistée, de demi-largeur `radius / 2` avec division entière. La conversion RGB vers gris utilise `0.299 R + 0.587 G + 0.114 B`.

- méthode actuelle : `mean_gray < 150` ;
- candidate figée : proportion des pixels tels que `gray < 220`, puis `dark_ratio >= 0.10`.

Ni `220`, ni `0.10`, ni la géométrie de ROI n'ont été modifiés après inspection du jeu indépendant.

## Résultats principaux

| Méthode | VP | VN | FP | FN | Précision | Rappel | Spécificité | Erreur globale |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Actuelle, `mean_gray < 150` | 119 | 468 | 0 | 53 | 100,00 % | 69,19 % | 100,00 % | 8,28 % |
| Candidate, `dark_ratio >= 0.10` | 169 | 460 | 8 | 3 | 95,48 % | 98,26 % | 98,29 % | 1,72 % |

La candidate récupère 50 des 53 faux négatifs de la méthode actuelle. En contrepartie, elle introduit huit faux positifs absents de la baseline.

### Résultats par copie

Chaque cellule est présentée sous la forme `VP/VN/FP/FN`.

| `student_exam_id` | Job | Cases cochées | Actuelle | Candidate |
|---:|---:|---:|---:|---:|
| 3 | 7 | 12 | 12/28/0/0 | 12/26/2/0 |
| 8 | 7 | 10 | 3/30/0/7 | 10/30/0/0 |
| 11 | 7 | 10 | 10/30/0/0 | 10/30/0/0 |
| 14 | 7 | 13 | 13/27/0/0 | 13/24/3/0 |
| 15 | 7 | 14 | 14/26/0/0 | 14/26/0/0 |
| 19 | 7 | 10 | 9/30/0/1 | 10/29/1/0 |
| 26 | 7 | 11 | 9/29/0/2 | 11/29/0/0 |
| 29 | 7 | 11 | 11/29/0/0 | 11/28/1/0 |
| 32 | 8 | 10 | 2/30/0/8 | 10/30/0/0 |
| 42 | 8 | 12 | 12/28/0/0 | 12/28/0/0 |
| 44 | 8 | 12 | 0/28/0/12 | 12/28/0/0 |
| 45 | 8 | 9 | 0/31/0/9 | 8/30/1/1 |
| 47 | 8 | 10 | 10/30/0/0 | 10/30/0/0 |
| 51 | 8 | 9 | 4/31/0/5 | 9/31/0/0 |
| 53 | 8 | 9 | 0/31/0/9 | 7/31/0/2 |
| 58 | 8 | 10 | 10/30/0/0 | 10/30/0/0 |

### Contrôle du jeu de calibration

Ces résultats ne sont pas inclus dans les performances principales.

| Copie | `student_exam_id` | Méthode actuelle | Candidate |
|---|---:|---:|---:|
| B05 | 31 | 0/30/0/10 | 10/30/0/0 |
| B08 | 38 | 10/30/0/0 | 10/30/0/0 |

Le résultat historique de 20 VP et 60 VN sans erreur sur B05+B08 est donc reproduit, mais il ne préjuge pas du résultat indépendant.

## Distribution de `dark_ratio`

| Vérité terrain | N | Min | P1 | P5 | P10 | P25 | Médiane | P75 | P90 | Max |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Cochée | 172 | 0,000 | 0,0149 | 0,2224 | 0,4585 | 0,8581 | 0,9848 | 1,000 | 1,000 | 1,000 |
| Vide | 468 | 0,000 | 0,000 | 0,000 | 0,000 | 0,000 | 0,000 | 0,000 | 0,000 | 0,455 |

Le minimum des cases cochées est `0.000`; le maximum des cases vides est `0.455`. Les distributions se chevauchent donc largement dans leurs extrêmes. La majorité des cases est très bien séparée, mais `dark_ratio` seul ne fournit pas une frontière absolue sur ce corpus.

## Analyse détaillée des erreurs de la candidate

Les crops correspondants ont été exportés localement dans `candidate-errors/`.

| Type | ID | Q | A | Page | Centre `(x,y)`, rayon | `mean_gray` | `dark_ratio` | Observation visuelle |
|---|---:|---:|---:|---:|---|---:|---:|---|
| FP | 29 | 6 | 1 | 2 | `(790,1574)`, 20 | 232,05 | 0,180 | rature ou trace sombre à l'intérieur de la case ; la case n'est pas cochée selon la lecture globale |
| FP | 14 | 9 | 0 | 3 | `(193,1357)`, 20 | 201,35 | 0,308 | anneau imprimé très épais pénétrant les coins de la ROI carrée |
| FP | 14 | 9 | 1 | 3 | `(561,1357)`, 20 | 221,36 | 0,185 | même effet d'anneau, moins marqué |
| FP | 14 | 9 | 2 | 3 | `(193,1519)`, 20 | 167,07 | 0,455 | anneau épais/déformé capté par la ROI ; intérieur visuellement vide |
| FP | 19 | 5 | 2 | 2 | `(193,977)`, 20 | 184,75 | 0,348 | point sombre et anneau irrégulier dans une case non sélectionnée |
| FP | 3 | 9 | 0 | 3 | `(193,1032)`, 20 | 231,40 | 0,158 | anneau imprimé légèrement décentré dans la ROI carrée |
| FP | 3 | 9 | 2 | 3 | `(193,1194)`, 19 | 226,69 | 0,179 | anneau imprimé légèrement décentré dans la ROI carrée |
| FP | 45 | 8 | 0 | 2 | `(194,2822)`, 20 | 239,53 | 0,113 | signal juste au-dessus du seuil, dû à l'anneau/au bruit très pâle ; question visuellement laissée vide |
| FN | 45 | 4 | 2 | 2 | `(193,813)`, 20 | 253,60 | 0,000 | traits roses verticaux extrêmement pâles, visibles dans le contexte mais aucun pixel sous 220 dans la ROI |
| FN | 53 | 4 | 0 | 1 | `(193,3121)`, 19 | 246,41 | 0,006 | trace cyan très pâle et diffuse |
| FN | 53 | 9 | 2 | 3 | `(193,1355)`, 19 | 244,01 | 0,019 | trait cyan très pâle ; signal chromatique visible mais presque absent en gris sous 220 |

Les faux positifs ne sont pas de simples valeurs juste au-dessus de `0.10` : sept sur huit vont de `0.158` à `0.455`. De même, les trois faux négatifs sont très loin sous le seuil. Cela indique deux limites structurelles plutôt qu'une mauvaise localisation fine du seuil : ROI carrée sensible à l'anneau, et perte du signal de traits colorés extrêmement pâles lors du passage en gris.

## Bande d'ambiguïté

Une bande étroite autour de `0.10` n'est pas justifiée par ces données. À titre descriptif, sans changer la décision :

- `[0.08, 0.12]` contient 2 cases et seulement 1 des 11 erreurs ;
- `[0.05, 0.15]` contient 8 cases et seulement 1 erreur ;
- `[0.05, 0.20]` contient 14 cases et 5 erreurs.

Même la bande la plus large de cet examen manque les trois faux négatifs et plusieurs faux positifs. Proposer maintenant une bande comme mécanisme suffisant donnerait donc une fausse impression de sécurité. Une future ambiguïté devrait probablement combiner plusieurs signaux, par exemple la distance au seuil, la couleur et la géométrie du signal sombre.

## Limites

- L'échantillon est raisonné et limité à deux jobs, deux formulaires et 16 copies indépendantes ; il n'est pas statistiquement représentatif de tous les scanners, stylos ou impressions.
- La vérité terrain a été transcrite par inspection des pages alignées et non des originaux papier. Les trois traces les plus pâles restent intrinsèquement difficiles à juger, même si leur continuité et leur couleur dans les vues de contexte conduisent à les classer cochées.
- Plusieurs faux positifs apparaissent sur les mêmes zones de page et la même copie. Les 640 cases ne sont donc pas des observations totalement indépendantes.
- L'évaluation conserve volontairement la ROI carrée actuelle. Elle ne mesure pas encore l'effet d'un masque circulaire intérieur qui exclurait mieux l'anneau.
- Aucun résultat de scoring n'a été calculé ou utilisé.

## Conclusion et recommandation

La règle figée `gray < 220` et `dark_ratio >= 0.10` est nettement plus robuste que la moyenne de gris pour les traits clairs : elle réduit les erreurs de 53 à 11 et porte le rappel de 69,19 % à 98,26 %. Elle généralise bien au-delà de B05/B08 sur plusieurs couleurs et intensités.

La validation reste toutefois **insuffisante pour une bascule directe en production**, car huit faux positifs sélectionnent des cases visuellement vides et trois traits très pâles restent invisibles. Elle est **suffisamment robuste pour un prototypage de production contrôlé**.

Le prochain jalon devrait, sans toucher au scoring :

1. implémenter la métrique candidate derrière une fonction isolée et testable, en conservant strictement `220` et `0.10` comme référence ;
2. comparer sur le même corpus une ROI circulaire intérieure ou un carré légèrement plus petit afin d'exclure l'anneau, avec un nouveau jeu tenu à l'écart pour éviter une seconde calibration sur ces erreurs ;
3. ajouter un signal couleur simple pour les traits rose/cyan dont le niveau de gris reste presque blanc ;
4. prévoir des tests de non-régression sur les 11 cas limites avant toute activation dans le workflow réel.

## Fichiers créés

Fichier versionné :

- `docs/reports/2026-09-04-answer-detection-validation.md`.

Artefacts privés, ignorés par Git :

- `runtime/diagnostics/answer-detection-validation/prepare.py` ;
- `runtime/diagnostics/answer-detection-validation/evaluate.py` ;
- 18 planches `student-exam-*-contact-sheet.png` ;
- `ground_truth.csv` ;
- `evaluated-cases.csv` ;
- `metrics.csv` ;
- `results.txt` ;
- 11 crops sous `candidate-errors/`.

## Commandes exécutées

```text
python3 runtime/diagnostics/answer-detection-validation/prepare.py
python3 runtime/diagnostics/answer-detection-validation/evaluate.py
python3 <analyse descriptive des quantiles et bandes à partir de evaluated-cases.csv>
git check-ignore -v runtime/diagnostics/answer-detection-validation/ground_truth.csv
git status --short
git diff --check
```

## Validations

- 18 copies préparées, 720 cases au total ; 2 copies/80 cases de calibration et 16 copies/640 cases de validation indépendante.
- Les planches ne montrent aucune décision automatique et ne contiennent aucune donnée personnelle.
- Les sources ouvertes par les scripts sont exclusivement les pages `aligned` non annotées des jobs 7 et 8.
- Les seuils évalués sont restés fixés à `220` et `0.10`.
- Les artefacts privés sont placés sous `runtime/` et ignorés par Git.
- `git check-ignore -v` confirme leur exclusion par la règle `.gitignore:37:/runtime/`.
- `git status --short` ne montre aucun artefact de validation sous `runtime/`. Il montre le rapport via le répertoire non suivi `docs/reports/`, ainsi que des modifications et fichiers non suivis préexistants, laissés intacts.
- `git diff --check` termine sans erreur. Le rapport non suivi a également été contrôlé séparément avec `git diff --no-index --check` : aucune erreur.
- Aucun fichier Go, code de production, schéma, base de données, scoring ou UX n'a été modifié.
- Aucun commit n'a été effectué.
