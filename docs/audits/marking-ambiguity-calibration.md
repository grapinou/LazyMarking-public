# Calibration réelle de l'ambiguïté Marking

## Résultat exécutif

Le harness réel opt-in a analysé 3 copies, 6 pages et 86 cases. Sur ce corpus,
les valeurs forment deux groupes entièrement séparés : 29 cases classées
`checked` entre 22,25 et 74,90, et 57 cases classées `unchecked` entre 252,24
et 255,00. La zone comprise entre 74,90 et 252,24 est vide ; le seuil historique
150 se trouve donc dans un intervalle vide de 177,34 niveaux de gris.

Aucune case n'est située à moins de 30 points du seuil. Les deltas 5, 10, 15 et
20 produisent par conséquent exactement le même résultat : zéro proposition de
revue. Ce corpus confirme que la distance à 150 est techniquement exploitable
comme critère de présélection, mais il ne contient précisément aucun cas proche
du seuil permettant de calibrer la largeur de la bande.

Décision : **données insuffisantes, inspection visuelle requise avant choix**.
Il faut d'abord élargir le corpus afin d'obtenir des cas proches de 150, puis
inspecter humainement un échantillon de crops. Aucun `ambiguity_delta` n'est
activé en production.

## Corpus et méthode

`TestRealMarkingAmbiguityCalibration` réutilise `MarkingStudentExam` sur les
trois fixtures privées opt-in existantes (une, deux et trois pages). Il lit les
`AnswerDetections` du `DetailedResult` produit par le pipeline réel. Il ne
duplique ni l'homographie, ni `GetAnswerDetections`, ni la classification.

Le test agrège immédiatement les données en mémoire. Sa sortie contient
uniquement des nombres globaux : aucun chemin, identifiant de copie, nom, texte,
réponse ou scan. Les données privées et les sorties individuelles ne sont pas
écrites dans le dépôt.

Les quantiles utilisent une interpolation linéaire entre les observations
triées. Les nombres du présent rapport sont arrondis à deux décimales lorsque
la précision supplémentaire n'apporte rien à la décision.

| Mesure | Valeur |
|---|---:|
| Copies | 3 |
| Pages | 6 |
| Cases | 86 |
| Checked selon l'algorithme | 29 |
| Unchecked selon l'algorithme | 57 |

## Distribution MeanGray

| Population | Min | Max | Moyenne | Médiane | p1 | p5 | p10 | p25 | p75 | p90 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Toutes (86) | 22,25 | 255,00 | 182,25 | 252,98 | 25,47 | 29,52 | 31,95 | 51,48 | 253,32 | 253,65 | 253,84 | 254,30 |
| Checked (29) | 22,25 | 74,90 | 42,71 | 38,20 | 23,31 | 26,81 | 28,92 | 31,55 | 51,46 | 62,61 | 65,95 | 72,42 |
| Unchecked (57) | 252,24 | 255,00 | 253,24 | 253,24 | 252,29 | 252,47 | 252,60 | 252,99 | 253,51 | 253,76 | 253,90 | 254,54 |

### Histogramme anonymisé

| Bande MeanGray | Cases | Bande MeanGray | Cases |
|---|---:|---|---:|
| [0, 20[ | 0 | [120, 130[ | 0 |
| [20, 40[ | 16 | [130, 140[ | 0 |
| [40, 60[ | 9 | [140, 145[ | 0 |
| [60, 80[ | 4 | [145, 150[ | 0 |
| [80, 100[ | 0 | [150, 155[ | 0 |
| [100, 120[ | 0 | [155, 160[ | 0 |
| [160, 170[ | 0 | [170, 180[ | 0 |
| [180, 200[ | 0 | [200, 220[ | 0 |
| [220, 240[ | 0 | [240, 256[ | 57 |

La distribution observée est nettement bimodale : un groupe sombre de 29
observations sous 75 et un groupe clair de 57 observations au-dessus de 252.
Il n'existe aucune observation entre les deux groupes. Cela quantifie une
séparation très forte sur ce corpus, sans constituer une vérité terrain sur la
correction des états automatiques.

## Distance au seuil 150

`distance = abs(MeanGray - 150)`.

| Min | Max | Moyenne | Médiane | p1 | p5 | p10 | p25 | p75 | p90 | p95 | p99 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 75,10 | 127,75 | 104,60 | 103,30 | 82,62 | 91,01 | 99,49 | 102,86 | 103,85 | 118,05 | 120,48 | 124,53 |

| Distance maximale | Cases | Pourcentage |
|---|---:|---:|
| ≤ 2 | 0 | 0 % |
| ≤ 5 | 0 | 0 % |
| ≤ 10 | 0 | 0 % |
| ≤ 15 | 0 | 0 % |
| ≤ 20 | 0 | 0 % |
| ≤ 25 | 0 | 0 % |
| ≤ 30 | 0 | 0 % |

Il n'existe aucune valeur exactement égale à 150, ni même à distance ≤ 0,5.
Le cas contractuel reste inchangé : une valeur exactement égale à 150 serait
`unchecked` avec l'algorithme historique et appartiendrait à toute bande de
delta positif ou nul.

## Bandes candidates et charge utilisateur

| Delta | Cases proposées | Part totale | Checked | Unchecked | Moyenne par copie | Copies sans revue |
|---:|---:|---:|---:|---:|---:|---:|
| 5 | 0 | 0 % | 0 | 0 | 0,00 | 3 / 3 |
| 10 | 0 | 0 % | 0 | 0 | 0,00 | 3 / 3 |
| 15 | 0 | 0 % | 0 | 0 | 0,00 | 3 / 3 |
| 20 | 0 | 0 % | 0 | 0 | 0,00 | 3 / 3 |

Ces résultats montrent une charge nulle sur le corpus, mais ne permettent pas
de départager les candidats. Choisir 5, 10, 15 ou 20 à partir de ces seules
données serait arbitraire.

## Variation inter-copies

La médiane des `MeanGray unchecked` a pu être calculée anonymement pour les
trois copies. Elle varie seulement de 253,05 à 253,71, avec une médiane
inter-copies de 253,17 : amplitude 0,67. Aucune variation inter-copies
significative de la luminosité des cases claires n'est visible dans ce petit
échantillon.

Cette observation ne démontre pas qu'un seuil global est robuste à tous les
scanners ou toutes les conditions d'acquisition : trois copies provenant de
fixtures limitées ne couvrent vraisemblablement pas cette diversité.

## Anomalies

Toutes les 86 valeurs sont finies et comprises dans `[0, 255]`. Aucun état
inattendu, aucune valeur exactement à 150 et aucune valeur quasi égale à 150
n'ont été observés.

## Limites méthodologiques

`detected_state` est produit par la comparaison historique à 150 ; ce n'est pas
une annotation humaine indépendante. MeanGray seul ne permet donc pas de dire
qu'une détection est correcte, et ce rapport ne mesure ni faux positif ni faux
négatif. En particulier, l'absence de valeurs proches du seuil ne prouve pas
qu'un delta donné « détecte toutes les erreurs ».

Le corpus de 3 copies est suffisant pour vérifier la collecte et constater une
bimodalité locale, mais insuffisant pour calibrer une bande destinée à des
conditions réelles variées. Les quatre candidats ayant le même résultat nul,
aucune optimisation charge/rappel n'est possible ici.

## Calibration en deux étapes

### Étape A — distribution statistique

Étendre le test opt-in à des lots privés plus nombreux et variés, en conservant
uniquement les mêmes agrégats anonymes. Rechercher en priorité des acquisitions
dont la distance minimale à 150 descend sous 30, puis comparer la charge par
copie et le nombre de copies sans revue pour chaque candidat.

### Étape B — vérification visuelle humaine

Lorsque des cas proches existent, un test ou petit outil strictement local peut :

1. résoudre la page via `ResolveMarkingAlignedPage` ;
2. retrouver le cercle dans le snapshot de page et appliquer une marge autour
   de sa ROI ;
3. écrire les crops dans un `t.TempDir()` ou un répertoire explicitement ignoré ;
4. utiliser des noms séquentiels anonymes et un manifeste local limité à
   `MeanGray`, état automatique et distance au seuil ;
5. supprimer le répertoire après inspection.

Aucun crop, manifeste privé ou identifiant ne doit entrer dans Git. Une UX web
production n'est ni nécessaire ni souhaitée pour cette calibration.

## Recommandation finale

La distance à 150 semble **suffisamment discriminante pour une première file de
revue, avec réserve** : elle sépare mathématiquement la proximité au seuil et le
corpus observé montre deux modes très éloignés. Elle ne prédit toutefois pas à
elle seule les erreurs réelles.

Recommandation : **données insuffisantes, inspection visuelle requise avant
choix**. Aucun delta définitif n'est recommandé tant qu'un corpus plus large ne
contient pas de valeurs proches du seuil et que leurs crops n'ont pas été
inspectés humainement.

## Validation et absence d'impact production

- `ambiguity_delta` production modifié : non ; il reste `NULL`.
- Seuil historique 150 modifié : non.
- `GetAnswerDetections`, `CountingPoints`, scores et PDF modifiés : non.
- Données privées ajoutées au dépôt : non.
- Test ajouté : `TestRealMarkingAmbiguityCalibration`, opt-in et skipped sans
  toutes les variables privées.
- Test réel exécuté : oui, sur 3 copies et 6 pages.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.
