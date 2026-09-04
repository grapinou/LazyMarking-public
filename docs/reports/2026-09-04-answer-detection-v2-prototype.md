# Prototype du détecteur de réponses V2

Date : 2026-09-04

## Résumé exécutif

Le V2 figé a été évalué sur un hold-out de 16 copies et 640 cases, entièrement séparé de son jeu de développement. Il obtient 162 VP, 474 VN, 2 FP et 2 FN, soit 98,78 % de précision, 98,78 % de rappel, 99,58 % de spécificité et 0,625 % d'erreur. Il améliore nettement la production actuelle (10,63 % d'erreur) et V1 (1,88 %), sans avoir été recalibré après observation du hold-out.

Une analyse visuelle post-hoc, ajoutée après l'évaluation et sans modification de V2, répartit les copies selon leur style dominant : 7 copies nominales (marques sombres, 280 cases), 7 non conformes mais raisonnablement visibles (280 cases) et 2 stress tests très pâles (80 cases). V2 donne respectivement 0,71 %, 0 % et 2,50 % d'erreur. Dans le domaine nominal, son rappel est de 100 %, avec deux faux positifs dus à des traces ou irrégularités dans des cases humainement vides.

Le signal d'ambiguïté expérimental n'est pas prêt pour la production : il désigne 103/640 cases (16,09 %), ne couvre que 2 des 4 erreurs et imposerait 101 revues inutiles. Dans le domaine nominal, il désigne même 77/280 cases (27,50 %). Il ne doit donc pas être activé tel quel.

Conclusion : les résultats justifient une implémentation V2 isolée et testable lors du prochain jalon, suivie d'une validation opérationnelle, mais pas une bascule aveugle ni l'activation de la règle d'ambiguïté actuelle. Les quatre erreurs doivent devenir des cas de non-régression. Aucun code de production ni paramètre expérimental n'a été modifié pendant le présent travail.

## Périmètre et artefacts

L'analyse réutilise exclusivement les artefacts privés déjà présents sous `runtime/diagnostics/answer-detection-v2/` :

- `development-measurements.csv` et `development-results.txt` ;
- `v2-frozen.json` ;
- `holdout-ground-truth.csv`, `holdout-measurements.csv`, `holdout-metrics.csv` et `holdout-results.txt` ;
- les 16 planches de contact `holdout-student-exam-*-contact-sheet.png` ;
- les quatre crops existants sous `errors/` ;
- les scripts historiques `develop.py`, `prepare_holdout.py` et `evaluate_holdout.py`, consultés pour documenter la méthode.

Les images sources sont les pages alignées non annotées des jobs 7 et 8. Aucun PDF corrigé, résultat de scoring ou réponse attendue n'a servi à déterminer si une case était cochée. Le présent jalon de reprise n'a relancé ni apprentissage, ni calibration, ni évaluation génératrice d'artefacts.

## Méthodologie

La vérité terrain est binaire et porte sur chaque case : une case visuellement marquée est `human_checked=1`, indépendamment de la bonne réponse ; une case visuellement vide est `0`. Les questions sans sélection et les sélections multiples sont conservées. Les planches ayant servi à la transcription montrent les crops et leurs coordonnées logiques, mais aucune prédiction automatique.

Trois méthodes sont comparées sur les mêmes cases :

- **production actuelle** : moyenne de gris de la ROI carrée `< 150` ;
- **V1** : au moins 10 % des pixels de la ROI carrée sous le niveau de gris 220 ;
- **V2 figé** : combinaison d'un signal de gris et d'un signal chromatique dans une ROI circulaire intérieure, décrite exactement ci-dessous.

Les conventions sont : VP = cochée humaine détectée, VN = vide humain rejeté, FP = vide détecté comme coché, FN = cochée non détectée. La précision vaut `VP/(VP+FP)`, le rappel `VP/(VP+FN)`, la spécificité `VN/(VN+FP)` et le taux d'erreur `(FP+FN)/N`.

## Séparation développement / hold-out et gel de V2

Le développement comprend 18 copies et 720 cases issues du corpus de validation antérieur : 192 cochées et 528 vides. Le hold-out comprend 16 autres copies (`student_exam_id` 1, 2, 9, 12, 17, 20, 23, 28, 33, 34, 39, 43, 46, 48, 50 et 59), soit 640 cases : 164 cochées et 476 vides.

La séquence matérielle conservée constitue la preuve du gel préalable :

1. `development-measurements.csv` et `development-results.txt` ont été produits à 16:38:33 ;
2. `v2-frozen.json` a été créé à 16:39:03 et déclare explicitement `frozen_before_holdout_annotation_and_evaluation: true` ;
3. `holdout-ground-truth.csv` puis `holdout-measurements.csv` ont été produits à 16:40:33, et `holdout-results.txt` à 16:40:34.

Le script historique d'évaluation charge la configuration figée avant de calculer le hold-out et commente que la vérité terrain a été transcrite depuis des planches sans prédiction après la création du fichier de gel. L'empreinte SHA-256 actuelle de `v2-frozen.json` est `3944f206b9b3b2507a259f2c269b0c00a56deefab5dc1b8db46ef187c5851db5`. Ce fichier et tous les paramètres qu'il contient ont été laissés intacts.

## Règle V2 exacte et figée

Pour une case de centre `(x,y)` et de rayon `r`, V2 utilise le disque centré défini par `dx² + dy² <= (0,40 r)²`. Pour chaque pixel RGB du disque :

- niveau de gris : `gray = 0,299 R + 0,587 G + 0,114 B` ;
- chroma : `max(R,G,B) - min(R,G,B)`.

Les signaux sont ensuite :

- `grayscale_signal = proportion(gray < 220) >= 0,10` ;
- `color_signal = proportion(chroma > 12) >= 0,05` ;
- décision V2 : `grayscale_signal OR color_signal`.

La règle d'ambiguïté expérimentale, également figée, est :

`grayscale_signal != color_signal OR (v1_square_signal AND NOT grayscale_signal)`

où `v1_square_signal` est la règle V1 sur la ROI carrée. Les comparaisons strictes sur les pixels (`< 220`, `> 12`) et inclusives sur les proportions (`>= 0,10`, `>= 0,05`) font partie de la définition.

## Résultats sur le développement

| Méthode | VP | VN | FP | FN | Précision | Rappel | Spécificité | Erreur |
|:--|--:|--:|--:|--:|--:|--:|--:|--:|
| Production actuelle | 131 | 528 | 0 | 61 | 100,00 % | 68,23 % | 100,00 % | 8,47 % |
| V1 | 189 | 520 | 8 | 3 | 95,94 % | 98,44 % | 98,48 % | 1,53 % |
| V2 retenu puis figé | 191 | 524 | 4 | 1 | 97,95 % | 99,48 % | 99,24 % | 0,69 % |

La grille conservée dans `development-results.txt` montre l'exploration des géométries et seuils uniquement sur le développement. La combinaison finalement figée correspond à `circle_0.40`, `chroma > 12`, proportion chromatique `>= 0.05`, avec le signal gris `gray < 220` à proportion `>= 0.10`. Le hold-out n'a servi à aucun choix de paramètre.

## Résultats finaux sur le hold-out

Le hold-out contient exactement 640 cases, dont 164 cochées et 476 vides.

| Méthode | VP | VN | FP | FN | Précision | Rappel | Spécificité | Erreur |
|:--|--:|--:|--:|--:|--:|--:|--:|--:|
| Production actuelle | 96 | 476 | 0 | 68 | 100,00 % | 58,54 % | 100,00 % | 10,63 % |
| V1 | 154 | 474 | 2 | 10 | 98,72 % | 93,90 % | 99,58 % | 1,88 % |
| **V2 figé** | **162** | **474** | **2** | **2** | **98,78 %** | **98,78 %** | **99,58 %** | **0,625 %** |

V2 récupère 66 des 68 faux négatifs de la production actuelle. Par rapport à V1, il récupère les 8 faux négatifs supplémentaires sans introduire de faux positif supplémentaire. Les deux FP de V1 restent des FP de V2.

## Analyse des quatre erreurs V2

La catégorie ci-dessous vient de l'analyse visuelle post-hoc par style dominant de copie, jamais de l'issue de V2. `mean` désigne la moyenne de gris carrée, `ds` la proportion sombre carrée, `dc` la proportion sombre du disque V2 et `c` la proportion chromatique du disque V2.

| Type | Copie / case | Catégorie | `mean` | `ds` | `dc` | `c` | Ambiguë | Observation visuelle |
|:--|:--|:--|--:|--:|--:|--:|:--:|:--|
| FP | 9 / Q4 / A2 | Nominale | 200,305 | 0,275 | 0,102 | 0,000 | Oui | Case vide selon la lecture humaine et le style cohérent de la copie. L'anneau est irrégulier/épaissi à gauche et une petite trace noire se trouve au-dessus ; le disque contient juste assez de noir pour franchir 10 %. |
| FP | 17 / Q6 / A3 | Nominale | 221,290 | 0,278 | 0,289 | 0,000 | Oui | Case non sélectionnée, mais présence de petits traits/points sombres et d'une forte irrégularité de l'anneau dans le quadrant inférieur droit. Ce bruit central sombre déclenche le signal gris. |
| FN | 46 / Q5 / A2 | Stress test | 244,661 | 0,005 | 0,010 | 0,000 | Non | Remplissage rose extrêmement pâle, diffus et peu contrasté ; visible en contexte, mais presque aucun pixel n'est sous 220 et la chroma mesurée ne dépasse pas le seuil requis. |
| FN | 46 / Q6 / A3 | Stress test | 251,150 | 0,003 | 0,000 | 0,000 | Non | Trace rose encore plus faible, proche du fond du scan ; aucun signal gris ou couleur suffisant dans le disque. |

Les deux FP sont couverts par l'ambiguïté parce que le signal gris est vrai alors que le signal couleur est faux. Les deux FN ne sont pas couverts : les deux signaux sont faux et V1 est faux. Les erreurs montrent donc deux limites distinctes : artefacts sombres au centre pour les FP, et marques chromatiques presque effacées par le scan pour les FN.

## Analyse de l'ambiguïté

| Indicateur | Résultat |
|:--|--:|
| Cases ambiguës | 103 / 640 |
| Taux global | 16,09 % |
| Erreurs V2 couvertes | 2 / 4 (50,00 %) |
| Cases correctes inutilement revues | 101 / 636 |
| Part des alertes correspondant réellement à une erreur | 2 / 103 (1,94 %) |

Cette règle n'est ni une bande de confiance calibrée ni une estimation probabiliste. Elle signale surtout un désaccord entre signaux. Sur les marques noires ou grises, le signal de gris est fréquemment vrai tandis que la chroma est naturellement faible : un grand nombre de réponses évidentes deviennent donc « ambiguës » sans être erronées.

Dans la catégorie nominale retenue ci-dessous, 77/280 cases sont ambiguës, soit 27,50 %. Elle couvre bien les deux erreurs nominales, mais envoie aussi 75 décisions correctes en revue. Un mécanisme qui fait vérifier plus d'un quart des cases nominales tout en manquant les deux FN pâles ne remplit pas un objectif raisonnable de réduction de charge. **Il ne faut pas activer cette règle d'ambiguïté en production dans son état actuel.**

## Analyse post-hoc du domaine nominal

### Statut et critères visuels

Cette analyse est explicitement **post-hoc** : le domaine produit a été précisé après le gel et l'évaluation du hold-out. Elle sert à décrire les résultats existants, pas à sélectionner des paramètres, redéfinir la vérité terrain ou exclure des cas du résultat principal.

Les critères ont été établis avant de compter les performances par catégorie, à partir des planches et de la vérité terrain humaine :

1. **Nominal** : style dominant composé de marques noires ou bleu très sombre, nettes, opaques et clairement distinctes du papier, compatible visuellement avec stylo bille bleu/noir ou feutre foncé.
2. **Non conforme mais raisonnablement visible** : marques nettement visibles et intentionnelles, mais couleur claire/non prévue (vert, cyan, orange), remplissage gris, ou instrument impossible à identifier avec assez de certitude pour affirmer la conformité.
3. **Extrême / stress test** : marques très pâles, diffuses ou proches du fond, dont la lecture exige le contexte et qui correspondent aux cas annoncés comme rose clair, cyan très clair ou crayon très léger.

Le scan ne permet pas d'identifier sûrement un instrument. Les catégories qualifient donc l'apparence numérisée et le style dominant de la copie. Toutes les cases, y compris vides, héritent de cette catégorie afin de calculer précision, spécificité et erreur sans classer artificiellement les cases vides sur leur absence de marque.

### Classement prudent des copies

| Catégorie | `student_exam_id` | Justification visuelle synthétique |
|:--|:--|:--|
| Nominal | 1, 2, 9, 12, 17, 20, 23 | Marques sombres, franches et aisément visibles ; l'instrument exact n'est pas démontrable, mais l'apparence est compatible avec la consigne future. |
| Non conforme mais visible | 33, 34, 39, 43, 48, 50, 59 | Remplissages verts, cyan, orange ou gris clairement visibles. Pour 39 et 48 notamment, crayon/feutre et couleur d'origine ne peuvent pas être affirmés depuis le scan ; classement prudent hors nominal. La copie 59 est bleue mais visuellement claire/remplie, donc conservée prudemment hors nominal. |
| Stress test | 28, 46 | Traces extrêmement faibles : gris très léger pour 28, rose très pâle et parfois à peine discernable pour 46. |

Ce classement n'affirme donc pas que les élèves ont employé un instrument précis ; il mesure la conformité visuelle au signal attendu après numérisation.

### Performances par catégorie

| Catégorie | Copies | Cases | Cochées / vides | Méthode | VP | VN | FP | FN | Précision | Rappel | Spécificité | Erreur | Ambiguës |
|:--|--:|--:|:--|:--|--:|--:|--:|--:|--:|--:|--:|--:|--:|
| Nominal | 7 | 280 | 75 / 205 | Production | 75 | 205 | 0 | 0 | 100,00 % | 100,00 % | 100,00 % | 0,00 % | — |
| Nominal | 7 | 280 | 75 / 205 | V1 | 75 | 203 | 2 | 0 | 97,40 % | 100,00 % | 99,02 % | 0,71 % | — |
| **Nominal** | **7** | **280** | **75 / 205** | **V2** | **75** | **203** | **2** | **0** | **97,40 %** | **100,00 %** | **99,02 %** | **0,71 %** | **77 (27,50 %)** |
| Non conforme visible | 7 | 280 | 68 / 212 | Production | 16 | 212 | 0 | 52 | 100,00 % | 23,53 % | 100,00 % | 18,57 % | — |
| Non conforme visible | 7 | 280 | 68 / 212 | V1 | 64 | 212 | 0 | 4 | 100,00 % | 94,12 % | 100,00 % | 1,43 % | — |
| **Non conforme visible** | **7** | **280** | **68 / 212** | **V2** | **68** | **212** | **0** | **0** | **100,00 %** | **100,00 %** | **100,00 %** | **0,00 %** | **15 (5,36 %)** |
| Stress test | 2 | 80 | 21 / 59 | Production | 5 | 59 | 0 | 16 | 100,00 % | 23,81 % | 100,00 % | 20,00 % | — |
| Stress test | 2 | 80 | 21 / 59 | V1 | 15 | 59 | 0 | 6 | 100,00 % | 71,43 % | 100,00 % | 7,50 % | — |
| **Stress test** | **2** | **80** | **21 / 59** | **V2** | **19** | **59** | **0** | **2** | **100,00 %** | **90,48 %** | **100,00 %** | **2,50 %** | **11 (13,75 %)** |

La lecture principale est double. Premièrement, V2 ne manque aucune marque nominale sur cet échantillon ; ses seules erreurs nominales sont deux cases vides bruitées. Deuxièmement, ses deux FN appartiennent tous deux à une même copie stress test rose extrêmement pâle, explicitement hors de l'exigence nominale envisagée. Cela contextualise les performances sans retrancher ces erreurs du score officiel de 0,625 %.

Les résultats catégoriels ne sont pas des estimations populationnelles : les groupes ne contiennent que 2 à 7 copies, la sélection initiale recherchait la diversité et plusieurs cases d'une même copie partagent le même instrument et le même scan.

## Limites

- Le hold-out est indépendant du développement mais raisonné, limité à 16 copies, deux jobs et deux formulaires ; il n'est pas un échantillon aléatoire de tous les usages futurs.
- La vérité terrain et la classification post-hoc reposent sur les scans alignés, pas sur les originaux papier. Les marques les plus pâles sont intrinsèquement difficiles à juger.
- Une seule personne a effectué la transcription visuelle ; aucun accord inter-annotateurs n'est mesuré.
- Les cases d'une copie ne sont pas indépendantes : instrument, geste, impression, scanner et alignement sont partagés.
- Le classement nominal repose sur l'apparence. Un stylo bille, un feutre ou un crayon ne peuvent pas toujours être distingués avec certitude après numérisation.
- Le sous-groupe nominal est défini après observation du corpus. Ses chiffres sont descriptifs et ne remplacent pas une nouvelle validation prospective sous consigne contrôlée.
- Les deux FP montrent que même une ROI circulaire intérieure peut contenir un anneau déformé, une poussière ou une trace parasite. Les deux FN montrent que le signal couleur actuel ne garantit pas la détection des couleurs presque blanches.
- Le prototype mesure la présence d'une marque, pas la validité pédagogique de la sélection, le score final, l'UX de revue ou le coût opérationnel.
- Le taux d'ambiguïté mesure une règle heuristique précise ; il ne doit pas être interprété comme une probabilité d'erreur.

## Conclusion sur une implémentation en production

V2 est suffisamment supérieur aux méthodes comparées pour justifier son **implémentation technique lors d'un prochain jalon**, derrière une unité isolée, testable et réversible. Sur le hold-out complet, il divise l'erreur de V1 par trois et celle de la production actuelle par dix-sept environ ; dans le domaine nominal post-hoc, il atteint 100 % de rappel et 0,71 % d'erreur.

Ces résultats ne justifient toutefois pas une activation sans garde-fous. Le corpus reste petit, le sous-groupe nominal est post-hoc, deux cases vides sont faussement sélectionnées et la règle d'ambiguïté actuelle produit une charge de revue disproportionnée tout en ratant la moitié des erreurs. L'opportunité recommandée est donc : implémenter V2 fidèlement, le valider prospectivement et opérationnellement, puis décider séparément de l'activation. Ne pas déployer la règle d'ambiguïté actuelle.

## Recommandations pour le prochain jalon

1. Implémenter la règle V2 figée dans une fonction isolée, sans modifier la ROI, les seuils ni la combinaison des signaux, avec tests reproduisant les métriques du fichier de gel.
2. Ajouter les quatre crops d'erreur comme cas de non-régression documentés, sans les utiliser pour recalibrer V2.
3. Constituer un nouveau corpus prospectif après communication de la consigne « stylo bille bleu ou noir, feutre foncé, marque clairement visible », et enregistrer si possible l'instrument déclaré ou observé avant numérisation.
4. Prévoir plusieurs scanners, qualités d'impression et alignements, ainsi qu'une annotation indépendante ou un arbitrage pour les cas pâles.
5. Mesurer séparément les performances copie par copie et dans le domaine nominal préétabli ; conserver aussi des stress tests, mais ne pas confondre leurs objectifs avec l'exigence nominale.
6. Concevoir et calibrer une nouvelle stratégie de revue sur un jeu de développement distinct, puis la geler avant un nouveau hold-out. Mesurer couverture des erreurs, taux de revue et valeur prédictive des alertes ; ne pas reprendre telle quelle l'heuristique actuelle à 103 alertes.
7. Valider enfin l'intégration complète : absence d'impact sur le scoring hors détection, observabilité des décisions, UX de correction manuelle, possibilité de retour arrière et tests de non-régression de production.

## Traçabilité de la reprise

Le rapport est le seul fichier créé par cette reprise. Tous les artefacts sous `runtime/` ont été conservés sur place et restent privés. Aucun code de production, paramètre V2, fichier de vérité terrain, mesure, crop ou planche de contact n'a été modifié ; aucun recalibrage et aucun commit n'ont été effectués.
