# Réduction de la charge de revue du détecteur hybride

Date : 2026-09-04

## Résumé et recommandation

Le job hybride 6eB analysé est le `marking_job_id=10`, version `hybrid-historical-v2-frozen-1`, politique `detector-agreement-v1`. Il contient 30 entrées de copies, dont 27 corrigées et 3 `not_seen`, soit 1 080 cases effectivement détectées. Les 178 candidats annoncés sont retrouvés exactement : ils représentent 16,481 % des cases et concernent 20 copies. Tous sont du type `historique=0 / V2=1`.

Les désaccords sont très majoritairement des signaux forts : 155/178 activent simultanément le gris et la couleur, 10/178 la couleur seule et 13/178 le gris seul. Sur les 112 désaccords du job couverts par une vérité terrain existante, les 112 sont de vraies marques ; les 66 autres ne sont pas annotés et ne doivent pas être présentés comme vrais positifs.

La règle simple la plus prudente observée est : **dans un désaccord, accepter automatiquement V2 si `color_signal=1`, sinon conserver la revue**. Elle ne modifie ni V2 ni ses seuils ; elle ne fait qu'étudier un triage en aval. Sur le job 10, elle réduirait la charge de 178 à **13 revues (1,204 % des cases), sur 3 copies**. Elle accepte 112 vrais positifs parmi les cas annotés et aucun faux positif connu, mais aussi 53 cas non annotés dont la vérité reste inconnue. Sur le développement et le hold-out, elle laisse en revue les six faux positifs V2 connus et accepte automatiquement 111 vrais désaccords.

Cette séparation est prometteuse, pas encore une preuve suffisante pour modifier la production. La recommandation est de retenir cette règle comme candidate du prochain jalon, puis de faire adjuger en aveugle les 178 cas — en priorité les 66 sans vérité terrain — et de la valider prospectivement sous la consigne nominale. La politique fondée sur la seule intensité grise descend à 3 revues sur ce job, mais son niveau de preuve est trop faible pour la priorité de fiabilité demandée.

## Sources et méthode

L'analyse est en lecture seule et utilise exclusivement :

- la base réelle locale `testdata/real/app.db` ;
- les métriques persistées de `marking_answer_detections` pour le job 10 ;
- `runtime/diagnostics/answer-detection-validation/ground_truth.csv` ;
- `runtime/diagnostics/answer-detection-v2/holdout-ground-truth.csv` et `holdout-measurements.csv` ;
- `runtime/diagnostics/answer-detection-v2/development-measurements.csv` ;
- les classifications visuelles déjà documentées dans les rapports antérieurs.

Le script et les sorties détaillées restent privés et ignorés par Git :

- `runtime/diagnostics/hybrid-review-load-analysis/analyze.py` ;
- `runtime/diagnostics/hybrid-review-load-analysis/job-10-disagreements.csv` ;
- `runtime/diagnostics/hybrid-review-load-analysis/summary.txt`.

Le CSV contient les 178 lignes, avec `student_exam_id`, question, réponse, états historique et V2, `mean_gray`, `dark_ratio`, `chroma_ratio`, les deux sous-signaux, le motif de revue et, lorsqu'elle existe déjà, la vérité humaine. Aucun seuil ni paramètre de V2 n'a été recalibré.

## Job analysé et répartition de la charge

| Élément | Valeur |
|:--|--:|
| `marking_job_id` | 10 |
| Statut persisté | `success` |
| Version d'algorithme | `hybrid-historical-v2-frozen-1` |
| Entrées de copies | 30 |
| Copies corrigées / `not_seen` | 27 / 3 |
| Cases détectées | 1 080 |
| Candidats à revue | 178 |
| Part des cases | 16,481 % |
| Copies concernées | 20 |

Le statut technique `success` signifie que le traitement est achevé ; le contrôle de cycle de vie précédent garantit séparément que les désaccords non résolus empêchent la présentation d'un résultat définitif.

### Candidats par copie

| `student_exam_id` | Revues | `student_exam_id` | Revues | `student_exam_id` | Revues |
|--:|--:|--:|--:|--:|--:|
| 31 | 10 | 41 | 10 | 51 | 4 |
| 32 | 8 | 42 | 0 | 52 | 10 |
| 33 | 10 | 43 | 10 | 53 | 9 |
| 34 | 10 | 44 | 12 | 54 | 0 |
| 35 | 10 | 45 | 8 | 55 | 0 |
| 36 | 7 | 46 | 9 | 56 | 0 |
| 37 | 9 | 47 | 0 | 57 | 0 |
| 38 | 0 | 48 | 2 | 58 | 0 |
| 39 | 0 | 49 | 0 | 59 | 10 |
| 40 | 10 | 50 | 10 | 60 | 10 |

| Tranche | Copies |
|:--|--:|
| 0 revue | 10 |
| 1–2 revues | 1 |
| 3–5 revues | 1 |
| 6–10 revues | 17 |
| Plus de 10 | 1 |

La charge n'est donc pas une longue traîne de quelques cas ambigus : 18 copies ont au moins six revues et une en a douze.

## Profil des 178 désaccords

| Type de V2 positif | Nombre | Part |
|:--|--:|--:|
| Gris et couleur | 155 | 87,08 % |
| Gris seul | 13 | 7,30 % |
| Couleur seule | 10 | 5,62 % |

| Mesure | Min | P10 | P25 | Médiane | P75 | P90 | Max |
|:--|--:|--:|--:|--:|--:|--:|--:|
| `mean_gray` | 150,300 | 160,208 | 177,085 | 200,860 | 220,055 | 232,377 | 246,404 |
| `dark_ratio` | 0,000 | 0,178 | 0,401 | 0,734 | 0,964 | 1,000 | 1,000 |
| `chroma_ratio` | 0,000 | 0,203 | 0,667 | 0,995 | 1,000 | 1,000 | 1,000 |

La médiane très élevée des deux ratios et les 155 doubles signaux expliquent le caractère inutilement massif de la revue actuelle sur cette classe sans consigne d'instrument.

## Vérité terrain et comparaison aux cas connus

La vérité existante couvre 18 des 30 copies du job et 112 des 178 désaccords. Tous ces 112 cas sont réellement cochés. Il reste **66 désaccords non annotés**, concentrés sur sept copies ; aucune politique ne peut honnêtement compter leurs acceptations comme vrais positifs.

### Distributions des désaccords vrais et faux connus

| Jeu / vérité | N | `dark_ratio` min / P25 / médiane / P75 / max | `chroma_ratio` min / P25 / médiane / P75 / max | Signaux |
|:--|--:|:--|:--|:--|
| Développement, vrais positifs | 60 | 0,000 / 0,406 / 0,791 / 0,970 / 1,000 | 0,000 / 0,548 / 0,995 / 1,000 / 1,000 | 46 deux, 10 gris, 4 couleur |
| Développement, faux positifs | 4 | 0,122 / 0,223 / 0,244 / 0,244 / 0,533 | 0 / 0 / 0 / 0 / 0 | 4 gris seuls |
| Hold-out, vrais positifs | 66 | 0,000 / 0,345 / 0,685 / 0,949 / 1,000 | 0,000 / 0,757 / 1,000 / 1,000 / 1,000 | 56 deux, 5 gris, 5 couleur |
| Hold-out, faux positifs | 2 | 0,102 / 0,102 / 0,102 / 0,289 / 0,289 | 0 / 0 / 0 / 0 / 0 | 2 gris seuls |

Les vrais et faux cas se recouvrent sur `dark_ratio` : un seuil gris ne crée donc pas de séparation structurelle. En revanche, les six faux positifs V2 connus sont tous achromatiques, tandis que 111/126 vrais désaccords des deux jeux ont un signal couleur. Cette propriété motive la politique C, mais six négatifs restent un échantillon trop petit pour conclure qu'un signal couleur exclut tout futur faux positif.

### Quatre erreurs demandées

| Cas | `dark_ratio` | `chroma_ratio` | Signaux V2 | Effet des politiques étudiées |
|:--|--:|--:|:--|:--|
| 9 / Q4 / A2, FP | 0,102 | 0,000 | gris seul | reste en revue A, B et C |
| 17 / Q6 / A3, FP | 0,289 | 0,000 | gris seul | reste en revue A, B et C |
| 46 / Q5 / A2, FN extrême | 0,010 | 0,000 | aucun | invisible pour toutes |
| 46 / Q6 / A3, FN extrême | 0,000 | 0,000 | aucun | invisible pour toutes |

Les deux faux négatifs extrêmes sont des accords négatifs, pas des désaccords. Aucun triage des désaccords ne peut les révéler. Le développement comporte de même un faux négatif invisible ; le hold-out en comporte deux.

## Politiques de triage

Les politiques sont évaluées uniquement sur les désaccords. Une acceptation automatique signifie ici « accepter l'état positif de V2 » dans la simulation analytique.

- Politique actuelle : tout désaccord est revu.
- Politique A : automatique si les deux sous-signaux V2 sont positifs.
- Politique B : automatique si le signal couleur est positif ou si `dark_ratio` est strictement supérieur à `0,5329949`, maximum des quatre FP du développement. Ce seuil est dérivé du développement et non de ce job ; il sert seulement de comparaison descriptive.
- Politique C : automatique si le signal couleur est positif ; tous les désaccords gris seuls restent revus.

### Charge et vérité connue sur le job 10

| Politique | Auto | Revues | % des 1 080 cases | Copies en revue | TP auto connus | FP auto connus | Auto sans vérité | Erreurs invisibles connues |
|:--|--:|--:|--:|--:|--:|--:|--:|:--|
| Actuelle | 0 | 178 | 16,481 % | 20 | 0 | 0 | 0 | FN 46/Q5/A2 et 46/Q6/A3 |
| A, deux signaux | 155 | 23 | 2,130 % | 8 | 103 | 0 | 52 | mêmes deux FN |
| B, couleur ou gris très fort | 175 | 3 | 0,278 % | 2 | 112 | 0 | 63 | mêmes deux FN |
| C, signal couleur | 165 | 13 | 1,204 % | 3 | 112 | 0 | 53 | mêmes deux FN |

Pour C, les 13 revues se trouvent sur les copies 35 (1), 37 (2) et 60 (10). Ces treize cas sont précisément les désaccords `grayscale_only`. Pour B, seules trois de ces cases restent en revue : 35/Q8/A2 et 60/Q4/A3, Q8/A4. Les indices de question/réponse sont présentés ici en base 1.

### Contrôle sur les corpus annotés

| Jeu | Politique | Revues | TP auto | FP auto | TP restant en revue | FP restant en revue |
|:--|:--|--:|--:|--:|--:|--:|
| Développement, 64 désaccords | Actuelle | 64 | 0 | 0 | 60 | 4 |
|  | A | 18 | 46 | 0 | 14 | 4 |
|  | B | 11 | 53 | 0 | 7 | 4 |
|  | C | 14 | 50 | 0 | 10 | 4 |
| Hold-out, 68 désaccords | Actuelle | 68 | 0 | 0 | 66 | 2 |
|  | A | 12 | 56 | 0 | 10 | 2 |
|  | B | 6 | 62 | 0 | 4 | 2 |
|  | C | 7 | 61 | 0 | 5 | 2 |

Les trois variantes gardent les six FP connus en revue. A est très interprétable mais dépasse légèrement la cible de 20 sur le stress opérationnel. B donne la charge minimale, mais son seuil est appuyé sur seulement quatre FP de développement et les distributions grises se chevauchent ; un artefact sombre futur pourrait être accepté. C atteint la cible opérationnelle sans introduire cette hypothèse supplémentaire sur l'intensité grise.

## Lecture par domaine d'utilisation

La passation 6eB n'avait aucune consigne et ne permet pas toujours d'inférer l'instrument après numérisation. Le découpage suivant est donc descriptif, non une nouvelle vérité terrain :

- proche du nominal documenté : copie 42, traits bleus forts ;
- visible mais non assuré nominal : copies 31, 32, 33, 34, 38, 39, 43, 44, 47, 48, 50, 51, 58 et 59, couvrant notamment rouge, vert, violet, crayon ou marques bleues claires ;
- stress documenté : copies 45, 46 et 53 ;
- non classé faute de description préalable suffisante : les 12 autres copies.

| Groupe descriptif | Désaccords | Actuelle | A | B | C |
|:--|--:|--:|--:|--:|--:|
| Proche nominal documenté | 0 | 0 | 0 | 0 | 0 |
| Visible, non assuré nominal | 86 | 86 | 4 | 0 | 0 |
| Stress | 26 | 26 | 5 | 0 | 0 |
| Non classé | 66 | 66 | 14 | 3 | 13 |

Les nombres sont des revues, pas des erreurs. Le résultat nul du seul cas 6eB proche du nominal est cohérent avec la future consigne — stylo bille bleu/noir, feutre foncé, marque clairement visible — mais une seule copie ne permet aucune estimation fiable de charge nominale. Sur le hold-out post-hoc antérieur, les deux faux positifs nominaux sont gris seuls : la politique C les garderait tous deux en revue. Une classe prospective sous consigne est indispensable pour estimer si l'objectif de moins de 10–20 validations est durable.

## Risques et limites

- 66/178 cas du job n'ont pas de vérité humaine existante. Les accepter automatiquement dans une simulation ne prouve pas leur correction.
- Les six FP connus proviennent de peu de copies et sont tous achromatiques. Une trace colorée, un défaut de scan ou un élément imprimé coloré pourrait invalider la séparation observée.
- La politique C accepterait les signaux `color_only`, y compris les marques très pâles hors domaine nominal. Cela améliore la charge mais ne résout pas les accords négatifs extrêmes.
- B exploite une frontière numérique du développement ; malgré l'absence de FP observé, elle est plus fragile et ressemble davantage à un nouveau seuil de confiance à valider.
- Le découpage de domaine est post-hoc et incomplet. Il ne doit pas servir à annoncer une performance nominale garantie.
- Les métriques de cases d'une même copie ne sont pas indépendantes ; un style ou un artefact peut affecter plusieurs réponses à la fois.

## Conclusion et prochain jalon

La politique actuelle est opérationnellement trop coûteuse sur ce stress test. La politique A réduit déjà la charge de 87 %, mais la politique C offre le meilleur compromis simple et conservateur observé : **13 revues au lieu de 178**, sur seulement trois copies, tout en laissant en revue chacun des six faux positifs V2 connus. B descend à trois revues, mais son gain supplémentaire ne justifie pas le risque d'une frontière grise apprise sur quatre faux positifs.

Le prochain jalon recommandé est une **validation de la politique C sans changement immédiat de production** :

1. faire relire en aveugle les 178 désaccords du job 10 et figer leur vérité, en contrôlant particulièrement les 66 actuellement non annotés ;
2. vérifier explicitement l'absence de faux positif parmi les 165 cas que C accepterait ;
3. constituer un corpus prospectif d'environ 30 élèves sous la consigne nominale annoncée ;
4. mesurer par copie la charge, la précision des acceptations automatiques et les accords négatifs faux ;
5. seulement si ces critères sont satisfaits, spécifier un jalon séparé d'implémentation versionnée et auditable du triage.

Aucun code Go, SQL, schéma, scoring, seuil V2 ou composant UX n'a été modifié dans ce jalon.
