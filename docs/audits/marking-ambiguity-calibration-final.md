# Calibration finale de l'ambiguïté Marking

## Conclusion

La revue humaine ciblée soutient **`ambiguity_delta = 5` comme première valeur
de production**. Les trois cas situés à distance ≤ 5 du seuil auraient tous été
présentés, dont l'unique divergence auto/humain observée. La charge mesurée sur
le corpus complet aurait été de 3 contrôles sur 1 890 cases, soit 0,16 % et
0,04 contrôle par copie.

Cette recommandation porte sur une première file de revue prudente, pas sur une
modification du seuil ni sur un nouvel algorithme de détection. Son niveau de
confiance est modéré : le signal observé est directement favorable à delta 5,
mais la revue de 20 crops est ciblée et ne mesure pas une sensibilité générale.

## Labels humains

Les 20 crops anonymes ont tous reçu un label :

| Label humain | Nombre |
|---|---:|
| `clear_checked` | 12 |
| `clear_unchecked` | 8 |
| `ambiguous` | 0 |
| `crossed_out` | 0 |
| `other` | 0 |

Aucun crop n'a donc été jugé intrinsèquement ambigu ou raturé dans cet
échantillon.

## Matrice automatique contre humain

Pour cette matrice, `clear_checked` est la classe humaine positive et
`clear_unchecked` la classe humaine négative.

|  | Humain checked | Humain unchecked | Total auto |
|---|---:|---:|---:|
| Auto checked | 11 | 0 | 11 |
| Auto unchecked | 1 | 8 | 9 |
| Total humain | 12 | 8 | 20 |

- Divergences auto/humain : 1.
- Faux positifs automatiques : 0.
- Faux négatifs automatiques : 1.

La divergence est `candidate-0014` : MeanGray 154,01, état automatique
`unchecked`, label humain `clear_checked`. Elle se situe à 4,01 points du seuil.
Les deux autres cas à distance ≤ 5 sont automatiquement et humainement
`checked`. Il ne faut pas en déduire que le seuil 150 doit changer : une seule
divergence ciblée ne permet pas de recalibrer la décision binaire historique.

## Comparaison des bandes

Les colonnes « divergences capturées » et « manquées » portent uniquement sur
l'unique divergence connue parmi les 20 crops revus. « Contrôles sans
modification » compte les candidats dont l'état automatique concorde déjà avec
le label humain. La précision de file est définie ici comme
`divergences / candidats` ; ce n'est pas la précision statistique du
classificateur.

| Delta | Candidats | Divergences capturées | Divergences manquées | Contrôles sans modification | Précision de file | Charge / 1 890 | Moyenne / 75 copies |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 5 | 3 | 1 | 0 | 2 | 33,33 % | 0,16 % | 0,04 |
| 10 | 3 | 1 | 0 | 2 | 33,33 % | 0,16 % | 0,04 |
| 15 | 4 | 1 | 0 | 3 | 25,00 % | 0,21 % | 0,05 |
| 20 | 6 | 1 | 0 | 5 | 16,67 % | 0,32 % | 0,08 |
| 25 | 10 | 1 | 0 | 9 | 10,00 % | 0,53 % | 0,13 |
| 30 | 15 | 1 | 0 | 14 | 6,67 % | 0,79 % | 0,20 |

Delta 10 n'apporte aucun cas supplémentaire par rapport à delta 5. À partir de
15, chaque élargissement augmente la charge sans trouver de divergence humaine
supplémentaire dans le corpus revu. Delta 5 est donc la plus petite bande testée
qui capture la divergence connue, et elle offre le meilleur compromis observé.

## Apport des catégories B et C

La catégorie B, « MeanGray < 75 et écart-type ≥ 80 », contient un cas. Son état
automatique `checked` concorde avec le label humain `clear_checked`. Divergence
supplémentaire au-delà de delta 5 : **non**.

La catégorie C contient six appartenances issues des extrêmes inter-copies,
avec deux recouvrements dans le corpus fusionné. Tous les crops concernés
concordent avec leur état automatique ; l'un d'eux appartient déjà à delta 5.
Divergence supplémentaire au-delà de delta 5 : **non**.

Ces résultats ne justifient pas de transformer l'écart-type, la proportion de
pixels sombres ou les extrêmes inter-copies en règles production. Ils restent
des pistes exploratoires sans preuve de gain sur cette revue.

## Biais et limites

Les 20 crops ne constituent pas un échantillon aléatoire des 1 890 cases. Ils
ont été volontairement sélectionnés parce qu'ils étaient proches du seuil,
sombres et hétérogènes, ou issus d'extrêmes inter-copies. En conséquence :

- le taux de divergence de la file ne mesure pas le taux d'erreur global ;
- zéro divergence manquée signifie seulement que l'unique divergence **connue
  dans le corpus revu** est capturée ;
- aucune sensibilité générale ne peut être estimée sans annoter aussi un
  échantillon indépendant hors des catégories ciblées ;
- l'absence de rature parmi 20 crops ne réfute pas l'existence de ratures très
  sombres dans d'autres copies.

La revue apporte néanmoins une preuve produit concrète : une bande ±5 aurait
soumis le faux négatif observé tout en imposant une charge extrêmement faible
sur les 75 copies.

## Recommandation concrète

**Activer `ambiguity_delta = 5` lors d'un jalon production distinct.**

Cette activation devra conserver le seuil historique 150 et la décision
automatique existante, et seulement alimenter la file de revue pour les cas à
distance ≤ 5. Il faudra ensuite mesurer la charge réelle et les corrections
humaines sur davantage de jobs. Une calibration ultérieure pourra comparer une
bande plus large si de nouvelles divergences apparaissent hors ±5.

Étape suivante : implémenter séparément l'activation production de delta 5 avec
tests d'ownership, de persistance et de lifecycle de revue, sans modifier le
score tant qu'aucune décision humaine n'est appliquée.

## Absence d'impact de ce jalon

- Production modifiée : non.
- `ambiguity_delta` activé : non ; il reste `NULL`.
- Seuil 150 modifié : non.
- `GetAnswerDetections` modifié : non.
- Migration ajoutée : non.
- Code de test modifié pour cette analyse finale : non.
- Données privées ajoutées au dépôt : non.
- Commit créé : non.
