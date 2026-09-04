# Validation aveugle finale de la politique C

Date : 2026-09-04

## Conclusion

**La politique C est validée sur le job réel 6eB pour un prochain jalon d'implémentation versionnée.**

Les 66 désaccords jusque-là dépourvus de vérité terrain ont été relus sur des planches aveugles. Les 53 cas que C accepterait automatiquement sont tous réellement cochés. Parmi les 13 cas que C conserverait en revue, 12 sont cochés et un est vide.

La vérité complète des 178 désaccords du job 10 est donc : **177 cases cochées et 1 case vide**. La politique C accepte automatiquement **165/165 vraies marques**, n'accepte aucune case vide et conserve en revue les treize cas `grayscale_only`, dont l'unique faux positif. Sa valeur prédictive positive observée est de **100 %**.

La charge passe de 178 à **13 revues**, soit **1,204 % des 1 080 cases**, réparties sur trois copies. Ce résultat répond au critère principal du jalon sans chercher à supprimer les treize revues restantes.

## Périmètre

L'analyse porte exclusivement sur :

- `marking_job_id=10` ;
- 30 entrées de copies, dont 27 effectivement corrigées et 3 `not_seen` ;
- 1 080 cases détectées ;
- 178 désaccords `historique=0 / V2=1`.

Les 112 désaccords déjà annotés ont été repris sans nouvelle lecture. Leur cohérence a été vérifiée par clé `(student_exam_id, question_index, answer_index)` ; aucun conflit n'a été trouvé. Les 66 autres cas ont reçu une nouvelle vérité humaine.

Tous les artefacts privés sont sous `runtime/diagnostics/hybrid-policy-c-validation/`, donc hors Git :

- trois planches du groupe critique ;
- une planche du groupe revue ;
- `ground_truth.csv`, contenant les 66 nouvelles annotations ;
- `review-case-metrics.csv`, produit seulement après annotation ;
- les scripts locaux de préparation et d'évaluation ;
- `results.txt`.

## Méthode d'annotation aveugle

### Constitution des groupes

Les 178 désaccords persistés ont été joints aux deux vérités terrain existantes. Les 112 clés déjà connues ont été exclues des planches. Les 66 clés restantes ont été séparées avant présentation :

- groupe critique : 53 cas que C accepterait automatiquement ;
- groupe revue : 13 cas que C laisserait à l'opérateur.

Les groupes n'ont pas été mélangés, conformément au protocole demandé.

### Présentation

Chaque crop montre la case et un voisinage restreint. La seule légende visible est :

- `student_exam_id` ;
- `question_index` ;
- `answer_index`.

Les planches n'affichent ni métrique, ni signal, ni état d'un détecteur, ni décision de C, ni correction pédagogique, ni score. Le CSV initial a été créé avec `human_checked` vide. Les métriques ont été réassociées uniquement après l'enregistrement des 66 décisions visuelles.

### Résultat de la nouvelle annotation

| Groupe | Cas annotés | Cochés | Vides |
|:--|--:|--:|--:|
| Critique, accepté par C | 53 | 53 | 0 |
| Revue, conservé par C | 13 | 12 | 1 |
| **Total nouveau** | **66** | **65** | **1** |

L'unique case vide est `student_exam_id=35 / question_index=7 / answer_index=1`. La planche montre un anneau imprimé sans marque humaine à l'intérieur.

## Vérité complète des 178 désaccords

| Origine de la vérité | Cas | Cochés | Vides |
|:--|--:|--:|--:|
| Vérité terrain préexistante | 112 | 112 | 0 |
| Nouvelle annotation aveugle | 66 | 65 | 1 |
| **Total** | **178** | **177** | **1** |

La table exhaustive par clé est reproductible localement par la réunion des vérités préexistantes et de `runtime/diagnostics/hybrid-policy-c-validation/ground_truth.csv`. Comme les 177 cas positifs partagent la même valeur, l'énumération complète est équivalente à : toutes les clés du CSV exhaustif des désaccords valent `1`, sauf `35/7/1`, qui vaut `0`.

## Résultat final de la politique C

La politique évaluée ne change pas V2 : parmi les seuls désaccords `historique=0 / V2=1`, `color_signal=1` est accepté et `color_signal=0` reste en revue.

| Mesure | Politique actuelle | Politique C |
|:--|--:|--:|
| Acceptations automatiques parmi les désaccords | 0 | 165 |
| Revues | 178 | 13 |
| Vrais positifs automatiques | 0 | 165 |
| Faux positifs automatiques | 0 | **0** |
| VPP des acceptations automatiques | sans objet | **100 %** |
| Vraies coches restant en revue | 177 | 12 |
| Cases vides restant en revue | 1 | 1 |
| Taux de revue sur 1 080 cases | 16,481 % | **1,204 %** |
| Copies concernées | 20 | **3** |

La réponse à la question centrale est donc **non** : sur ce job, C n'accepte automatiquement aucune case humainement vide. Les 165 acceptations automatiques sont toutes des marques réelles.

## Charge par copie

| `student_exam_id` | Revues C | `student_exam_id` | Revues C | `student_exam_id` | Revues C |
|--:|--:|--:|--:|--:|--:|
| 31 | 0 | 41 | 0 | 51 | 0 |
| 32 | 0 | 42 | 0 | 52 | 0 |
| 33 | 0 | 43 | 0 | 53 | 0 |
| 34 | 0 | 44 | 0 | 54 | 0 |
| 35 | 1 | 45 | 0 | 55 | 0 |
| 36 | 0 | 46 | 0 | 56 | 0 |
| 37 | 2 | 47 | 0 | 57 | 0 |
| 38 | 0 | 48 | 0 | 58 | 0 |
| 39 | 0 | 49 | 0 | 59 | 0 |
| 40 | 0 | 50 | 0 | 60 | 10 |

Vingt-sept copies ne nécessitent aucune revue. Les trois autres en nécessitent respectivement 1, 2 et 10. La charge totale de treize validations est compatible avec la cible opérationnelle annoncée pour cette phase.

## Analyse des treize cas restant en revue

Les indices ci-dessous sont les indices persistés, en base zéro. Les métriques n'ont été consultées qu'après la décision humaine.

| Copie | Question | Réponse | Vérité | `mean_gray` | `dark_ratio` | `chroma_ratio` |
|--:|--:|--:|:--|--:|--:|--:|
| 35 | 7 | 1 | vide | 222,650 | 0,101523 | 0,000000 |
| 37 | 7 | 0 | cochée | 171,893 | 0,720812 | 0,020305 |
| 37 | 8 | 1 | cochée | 178,130 | 0,690355 | 0,000000 |
| 60 | 0 | 1 | cochée | 199,780 | 0,634518 | 0,000000 |
| 60 | 1 | 1 | cochée | 196,380 | 0,624365 | 0,000000 |
| 60 | 2 | 3 | cochée | 204,368 | 0,568528 | 0,000000 |
| 60 | 3 | 2 | cochée | 216,827 | 0,355932 | 0,000000 |
| 60 | 4 | 3 | cochée | 207,758 | 0,568528 | 0,000000 |
| 60 | 5 | 0 | cochée | 203,113 | 0,604061 | 0,000000 |
| 60 | 6 | 2 | cochée | 197,843 | 0,666667 | 0,000000 |
| 60 | 7 | 3 | cochée | 220,235 | 0,385787 | 0,000000 |
| 60 | 8 | 2 | cochée | 198,035 | 0,609137 | 0,000000 |
| 60 | 9 | 0 | cochée | 185,105 | 0,734463 | 0,000000 |

Le faux positif présente un signal gris à peine supérieur au seuil V2 (`dark_ratio=0,101523`) et aucune chromaticité. Les douze vraies marques restantes sont achromatiques ou presque ; dix appartiennent à une même copie utilisant des croix grises/noires. Ce sont exactement les cas que la règle C veut soumettre à l'humain. Aucun nouveau seuil n'est recherché pour les supprimer.

## Faux négatifs hors désaccord

La validation des désaccords ne change pas les accords négatifs. Sur les copies du job disposant d'une vérité complète, trois marques connues restent classées `historique=0 / V2=0` :

| Copie | Question | Réponse | Statut |
|--:|--:|--:|:--|
| 45 | 4 | 2 | faux négatif invisible |
| 46 | 5 | 2 | faux négatif extrême déjà documenté |
| 46 | 6 | 3 | faux négatif extrême déjà documenté |

Ces cas ne sont pas causés par C et ne peuvent pas être exposés par une politique limitée aux désaccords. Les deux cas de la copie 46 restent les stress tests rose extrêmement pâle déjà connus. Le cas de la copie 45 est également une marque très pâle documentée dans le corpus de développement.

Pour les douze copies sans vérité exhaustive sur leurs accords, cette analyse ne permet pas d'affirmer l'absence de tout autre faux négatif hors désaccord.

## Limites

- La validation porte sur un seul scan réel de 6eB, volontairement difficile et réalisé sans consigne d'instrument.
- Les 165 succès ne constituent pas 165 observations indépendantes : plusieurs partagent une copie et un style de marquage.
- Les six faux positifs V2 antérieurement connus, plus le nouveau faux positif 35/7/1, sont tous `grayscale_only`. Cela soutient C, mais ne prouve pas qu'un futur artefact coloré est impossible.
- La vérité exhaustive des accords n'existe que pour 18 des 30 copies ; le chiffre de trois faux négatifs hors désaccord est donc un minimum connu sur le périmètre annoté, pas une garantie globale.
- La future charge nominale doit encore être mesurée prospectivement sous la consigne stylo bille bleu/noir, feutre foncé et marque clairement visible.

## Recommandation du prochain jalon

Le prochain jalon recommandé est **l'implémentation minimale, versionnée et auditable de la politique C**, avec les garde-fous suivants :

1. ne modifier ni les détecteurs ni leurs seuils ;
2. créer une nouvelle version de politique, sans réinterpréter les jobs hybrides existants ;
3. persister explicitement la décision de confiance fondée sur `color_signal` et son motif ;
4. conserver tous les désaccords `grayscale_only` dans la revue humaine ;
5. maintenir la revue comme autorité finale et le blocage de finalisation tant que ces cas ne sont pas résolus ;
6. ajouter des tests couvrant notamment un faux positif gris, un vrai positif gris et les chemins sans revue ;
7. poursuivre en parallèle une validation prospective sous consigne nominale, avec suivi du taux de revue et des éventuels faux positifs colorés.

Aucun code Go, aucune migration, aucune requête SQL de production, aucun détecteur, seuil, scoring ou composant UX n'a été modifié pendant ce jalon.
