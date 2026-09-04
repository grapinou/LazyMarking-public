# Évaluation d'une politique hybride de détection et de revue

Date : 2026-09-04

## Résumé exécutif

La politique étudiée conserve la production comme décision automatique, utilise V2 comme seconde lecture et envoie uniquement les désaccords `production=0 / V2=1` en revue humaine. Sur le hold-out de 640 cases, elle produit **68 alertes (10,625 %)**, portant sur 68 instances de question et 10 copies. Parmi elles, **66 sont réellement cochées et 2 réellement vides** : la valeur prédictive positive de l'alerte est donc de **97,06 %**, et 66 des 68 faux négatifs de production sont récupérables.

Avec une revue humaine supposée parfaite, le résultat simulé est de **162 VP, 476 VN, 0 FP et 2 FN**, soit 100 % de précision, 98,78 % de rappel, 100 % de spécificité et 0,3125 % d'erreur. Les deux erreurs restantes sont les deux marques rose extrêmement pâle de la copie 46 ; production et V2 les classent toutes deux vides, elles restent donc invisibles.

La charge globale est concentrée hors du domaine nominal. Sur les 280 cases nominales, seules les deux cases vides faussement positives de V2 partent en revue : **2 cases, 0,714 %, 2 questions et 2 copies**. La revue parfaite les rejette et conserve les performances sans erreur de la production actuelle sur ce sous-groupe.

**Recommandation : politique hybride**, plutôt qu'activation directe de V2. Elle exploite presque tout le gain de rappel de V2 sans accepter ses faux positifs comme décisions automatiques, et elle est nettement plus ciblée que la règle d'ambiguïté à 103 alertes. Cette recommandation désigne la meilleure première stratégie parmi celles évaluées ; le petit corpus et la définition post-hoc du domaine nominal imposent néanmoins une validation prospective avant une activation générale.

## Périmètre et méthode

Tous les calculs reposent exclusivement sur les artefacts existants sous `runtime/diagnostics/answer-detection-v2/`, principalement `holdout-measurements.csv`, `development-measurements.csv`, `holdout-results.txt` et `v2-frozen.json`. Aucun détecteur, seuil, paramètre, code de production, état persistant, scoring ou donnée de vérité terrain n'a été modifié. La simulation est purement analytique.

Le hold-out contient 16 copies, 640 cases, 164 cases humainement cochées et 476 humainement vides. Les catégories post-hoc reprises sans changement du rapport V2 sont :

- nominal : copies 1, 2, 9, 12, 17, 20 et 23 ;
- non conforme mais raisonnablement visible : copies 33, 34, 39, 43, 48, 50 et 59 ;
- stress tests : copies 28 et 46.

Dans les tableaux de matrice, la « précision de l'interprétation » est la proportion compatible avec l'interprétation portée par le groupe : vide pour `0/0`, cochée pour les groupes contenant une décision positive. Pour `0/1`, elle correspond donc aussi à la valeur prédictive positive de l'alerte V2. Le groupe vide `1/0` n'a pas de précision définie.

Pour la charge, une « question concernée » est une instance `(student_exam_id, question_index)`, et non le seul indice de question commun aux formulaires. Les 10 indices logiques Q0 à Q9 sont représentés parmi les alertes globales.

## Matrice complète production / V2 / vérité humaine

### Hold-out

| Production | V2 | Interprétation | Cases | Humainement cochées | Humainement vides | Précision de l'interprétation |
|--:|--:|:--|--:|--:|--:|--:|
| 0 | 0 | automatiquement vide | 476 | 2 | 474 | 99,58 % |
| 0 | 1 | alerte « cochée » / revue | 68 | 66 | 2 | 97,06 % |
| 1 | 0 | désaccord inverse | 0 | 0 | 0 | non définie |
| 1 | 1 | automatiquement cochée | 96 | 96 | 0 | 100,00 % |
| **Total** |  |  | **640** | **164** | **476** |  |

Il n'existe donc aucun cas D (`production=1 / V2=0`) sur le hold-out. Aucun traitement arbitraire n'est nécessaire ni déduit de ces données. Si ce cas apparaît prospectivement, la politique devra être complétée après analyse dédiée ; la présente évaluation ne l'assimile ni à une acceptation ni à une revue.

### Développement, contrôle de cohérence

| Production | V2 figé | Interprétation | Cases | Humainement cochées | Humainement vides | Précision de l'interprétation |
|--:|--:|:--|--:|--:|--:|--:|
| 0 | 0 | automatiquement vide | 525 | 1 | 524 | 99,81 % |
| 0 | 1 | alerte « cochée » / revue | 64 | 60 | 4 | 93,75 % |
| 1 | 0 | désaccord inverse | 0 | 0 | 0 | non définie |
| 1 | 1 | automatiquement cochée | 131 | 131 | 0 | 100,00 % |
| **Total** |  |  | **720** | **192** | **528** |  |

Le développement confirme la structure observée : V2 est un sur-ensemble des positifs de production, et les désaccords sont fortement enrichis en marques réelles. Ce résultat est seulement descriptif, puisque ce jeu a servi au développement de V2.

## Politique hybride évaluée

La simulation applique exactement les règles demandées :

1. `production=1` : cochée automatiquement ;
2. `production=0 / V2=1` : revue humaine ;
3. `production=0 / V2=0` : vide automatiquement ;
4. `production=1 / V2=0` : analyse séparée, sans décision présupposée — aucun cas observé ici.

La revue ne change pas le détecteur. Elle remplace uniquement, dans la simulation, la décision des 68 alertes par la vérité humaine.

## Charge exacte de revue

| Domaine | Cases du domaine | Cases revues | Part des cases | Questions concernées `(copie, Q)` | Copies concernées | Vérifications moyennes par copie concernée |
|:--|--:|--:|--:|--:|--:|--:|
| Nominal | 280 | 2 | 0,714 % | 2 | 2 | 1,00 |
| Non conforme mais raisonnablement visible | 280 | 52 | 18,571 % | 52 | 6 | 8,67 |
| Stress tests | 80 | 14 | 17,500 % | 14 | 2 | 7,00 |
| **Hold-out complet** | **640** | **68** | **10,625 %** | **68** | **10** | **6,80** |

Chaque alerte du hold-out appartient à une instance de question différente : les nombres de cases et de questions concernées sont donc identiques. Six des sept copies « non conformes visibles » sont concernées ; la septième est entièrement traitée sans revue. Dans le nominal, la charge se limite à une vérification sur chacune des copies 9 et 17.

La moyenne de 6,80 est calculée sur les 10 copies effectivement concernées, conformément à la demande. Rapportée à l'ensemble des 16 copies, la charge serait de 4,25 vérifications par copie, mais cette seconde valeur ne décrit pas la session d'un opérateur ayant effectivement une alerte.

## Qualité des candidats à revue

Parmi les 68 cas `production=0 / V2=1` :

- 66 sont réellement cochés ;
- 2 sont réellement vides ;
- la valeur prédictive positive est `66/68`, soit **97,06 %** ;
- ils couvrent **66/68 faux négatifs de production**, soit **97,06 %** ;
- les deux alertes non confirmées sont précisément les FP V2 nominaux : copie 9 / Q4 / A2 et copie 17 / Q6 / A3.

Par domaine :

| Domaine | Alertes | Réellement cochées | Réellement vides | VPP |
|:--|--:|--:|--:|--:|
| Nominal | 2 | 0 | 2 | 0,00 % |
| Non conforme mais visible | 52 | 52 | 0 | 100,00 % |
| Stress tests | 14 | 14 | 0 | 100,00 % |
| **Total** | **68** | **66** | **2** | **97,06 %** |

La VPP nominale n'est pas un signal de mauvaise charge : elle signifie ici que la production avait déjà 0 erreur nominale et que les deux revues servent exactement de garde-fou contre les deux faux positifs qu'introduirait une activation directe de V2.

### Comparaison avec la règle d'ambiguïté V2 à 103 alertes

| Critère | Désaccord hybride `P=0/V2=1` | Ambiguïté V2 actuelle |
|:--|--:|--:|
| Alertes | 68 (10,625 %) | 103 (16,09 %) |
| Alertes humainement cochées | 66 | 101 |
| Alertes humainement vides | 2 | 2 |
| Faux négatifs de production couverts | 66/68 (97,06 %) | 10/68 (14,71 %) |
| Erreurs propres à V2 couvertes | 2/4 (50,00 %) | 2/4 (50,00 %) |
| Alertes nominales | 2/280 (0,714 %) | 77/280 (27,50 %) |

Les 103 ambiguïtés comprennent 91 cases `production=1 / V2=1` déjà correctement cochées, 10 désaccords hybrides réellement cochés et les 2 FP V2. Autrement dit, seulement 12 des 68 alertes hybrides sont aussi ambiguës. La règle d'ambiguïté dépense 35 revues supplémentaires, mais couvre 56 faux négatifs de production de moins. Sa VPP brute « marque réellement présente » (`101/103 = 98,06 %`) est élevée précisément parce qu'elle alerte surtout sur des marques évidentes déjà correctement reconnues ; ce n'est pas une bonne mesure de son utilité marginale. Pour corriger les erreurs de production, le désaccord hybride est beaucoup plus ciblé.

## Résultat simulé après revue humaine parfaite

| Périmètre | VP | VN | FP | FN | Précision | Rappel | Spécificité | Taux d'erreur |
|:--|--:|--:|--:|--:|--:|--:|--:|--:|
| Nominal | 75 | 205 | 0 | 0 | 100,00 % | 100,00 % | 100,00 % | 0,00 % |
| Non conforme visible | 68 | 212 | 0 | 0 | 100,00 % | 100,00 % | 100,00 % | 0,00 % |
| Stress tests | 19 | 59 | 0 | 2 | 100,00 % | 90,48 % | 100,00 % | 2,50 % |
| **Hold-out complet** | **162** | **476** | **0** | **2** | **100,00 %** | **98,78 %** | **100,00 %** | **0,3125 %** |

Par rapport à la production, les erreurs passent de 68 à 2, soit une réduction de **97,06 %**. Par rapport à l'activation directe de V2, les deux FP sont éliminés par la revue, tandis que les deux FN communs subsistent : les erreurs passent de 4 à 2. Cette spécificité de 100 % est conditionnelle à l'hypothèse analytique de revue humaine parfaite ; elle ne prédit pas le taux d'erreur réel des opérateurs.

Sur le développement, la même simulation descriptive donnerait 191 VP, 528 VN, 0 FP et 1 FN : précision 100 %, rappel 99,48 %, spécificité 100 % et erreur 0,139 %, avec 64 revues sur 720 cases (8,89 %), 61 instances de question et 11 copies concernées.

## Erreurs restant invisibles

Les seules erreurs restantes sont dans le groupe `production=0 / V2=0` :

| Copie / case | Catégorie | Vérité | Production | V2 | Revue hybride | Résultat final |
|:--|:--|--:|--:|--:|:--|:--|
| 46 / Q5 / A2 | Stress test | cochée | 0 | 0 | non | faux négatif |
| 46 / Q6 / A3 | Stress test | cochée | 0 | 0 | non | faux négatif |

Oui, les deux FN stress-test explicitement demandés restent non détectés. Le premier est un remplissage rose extrêmement pâle et diffus ; le second une trace rose encore plus faible, proche du fond du scan. Puisque les deux détecteurs rendent la même décision négative, une politique fondée uniquement sur leur désaccord ne peut pas les exposer à l'opérateur. La règle d'ambiguïté V2 actuelle ne les expose pas non plus.

Il n'existe aucune autre erreur invisible sur ce hold-out. Le groupe `0/0` contient 474 vrais vides et ces 2 faux négatifs.

## Analyse opérationnelle

### 1. Réduction des erreurs et volume de revue

Oui. Sous l'hypothèse de revue parfaite, la politique élimine 66 des 68 erreurs de production, pour 68 vérifications sur 640 cases. La charge globale de 10,625 % reste notable, mais 66 revues sur 68 conduisent à une correction réelle. Elle est principalement causée par des instruments hors domaine nominal ou par les stress tests.

### 2. Charge dans le domaine nominal

Elle est très faible sur cet échantillon : 2/280 cases (0,714 %), deux questions sur deux copies, une vérification par copie concernée. Ces deux vérifications empêchent précisément les deux FP de V2. La petitesse et la définition post-hoc du sous-groupe interdisent toutefois d'en faire une estimation populationnelle définitive.

### 3. Comparaison avec une activation directe de V2

La politique hybride est préférable comme première activation opérationnelle. V2 direct donne 162 VP, 474 VN, 2 FP et 2 FN ; l'hybride avec revue parfaite donne les mêmes VP et FN mais restaure les deux VN, donc aucun FP. Elle évite qu'une seconde lecture plus sensible modifie automatiquement une réponse vide. Son coût est une file de revue et la dépendance à la qualité humaine.

### 4. Conservation de la spécificité de production

Oui dans cette simulation : les positifs de production sont tous vrais, aucun désaccord inverse n'existe, et les deux propositions positives erronées de V2 sont rejetées en revue. La spécificité finale reste donc à 100 %, contre 99,58 % pour V2 direct. Ce résultat doit être revalidé avec des erreurs de revue réelles et avec une règle explicite si un futur cas `production=1 / V2=0` apparaît.

### 5. Pertinence comme première stratégie de déploiement

Oui, à condition de la déployer de façon contrôlée et observable. Elle est conservatrice, réversible conceptuellement, explicable à l'opérateur et nettement plus efficiente que l'ambiguïté actuelle. Elle convient particulièrement à une phase où les élèves reçoivent la consigne nominale, car la charge nominale observée est faible. Elle ne remplace pas un corpus prospectif : le hold-out ne compte que 16 copies, les catégories sont post-hoc et aucune erreur humaine de revue n'est mesurée.

## Avantages et inconvénients

Avantages :

- récupère 97,06 % des FN de production sans accepter automatiquement les FP de V2 ;
- conserve analytiquement la spécificité de 100 % de la production ;
- alertes très actionnables globalement : 97,06 % entraînent une correction ;
- charge nominale observée de seulement 0,714 % ;
- 35 alertes de moins que l'ambiguïté V2 actuelle, avec une bien meilleure couverture des FN de production ;
- aucune modification du scoring et séparation claire entre décision automatique et intervention humaine.

Inconvénients et risques :

- 10,625 % des cases sont revues sur le corpus complet, et environ 18 % dans chacun des deux groupes hors nominal ;
- les deux marques extrêmement pâles restent invisibles, comme avec V2 direct et avec l'ambiguïté actuelle ;
- la performance finale dépend de la disponibilité et de la justesse des opérateurs ;
- aucun cas `production=1 / V2=0` n'a permis de valider le traitement du cas D ;
- le domaine nominal est post-hoc et petit ; les chiffres ne suffisent pas à garantir la charge future ;
- cette évaluation ne mesure ni temps par vérification, ni ergonomie, ni erreurs de revue, ni impact au niveau d'une copie complète.

## Recommandation et prochain jalon

Parmi les trois choix proposés, la recommandation est **la politique hybride**. Une activation directe de V2 ferait entrer deux faux positifs nominaux dans les décisions persistées. Conserver la production seule laisserait 68 marques non détectées. Déclarer simplement la validation insuffisante ignorerait qu'une stratégie conservatrice mesurable réduit déjà les erreurs de 97,06 % tout en limitant la charge nominale observée à 0,714 %. La bonne lecture est donc : politique hybride comme cible du prochain essai, mais pas généralisation sans validation prospective.

Le prochain jalon recommandé est une **validation prospective opérationnelle sous consignes élèves nominales**, sans recalibrer V2 sur le présent hold-out :

1. figer à l'avance la politique, y compris un traitement explicite et prudent d'un éventuel cas `production=1 / V2=0` ;
2. collecter un nouveau corpus sous consigne « stylo bille bleu ou noir, feutre foncé, marque clairement visible », avec plusieurs scanners et conditions d'impression ;
3. mesurer la matrice, la charge par copie, le temps de revue, les erreurs humaines de revue et les performances finales ;
4. conserver les deux FN de la copie 46 et les deux FP nominaux comme cas de non-régression, sans les employer à recalibrer les paramètres ;
5. fixer avant observation des critères d'acceptation portant au minimum sur spécificité finale, rappel, taux de revue nominal et erreurs au niveau question/copie.

Ce jalon n'a effectué aucun commit et s'arrête à la rédaction du présent rapport.
