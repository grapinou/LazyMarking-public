# Calibration réelle étendue de l'ambiguïté Marking

## Synthèse

La seconde calibration opt-in a traité 3 PDF de paquets complets, soit 121
pages, 75 copies découvertes et 1 890 cases. Les 75 copies ont été analysées ;
aucune n'a été ignorée. Les retries du lecteur production ont finalement décodé
tous les QR : zéro page est restée illisible.

Contrairement au premier corpus, celui-ci contient des cas proches du seuil
historique 150 : 15 détections à distance ≤ 30, dont 3 à distance ≤ 5. Il
contient également une case très sombre mais fortement hétérogène selon une
catégorie exploratoire basée sur l'écart-type de sa ROI.

La prochaine étape recommandée est une inspection locale de crops anonymes :
les 15 cas à distance ≤ 30, plus les cas sombres à forte variance. Cette charge
est assez petite pour obtenir une vérité terrain humaine avant de choisir un
`ambiguity_delta`. Aucun delta n'est activé par ce jalon.

## Harness et confidentialité

`TestRealMarkingAmbiguityCalibrationBatches` lit tous les PDF directement
présents dans `LAZYMARKING_CALIBRATION_DIR`. Pour chaque PDF, il réutilise les
primitives réelles : split PDF, rasterisation, lecture QR, `GroupQrCodes`, tri
de sécurité effectué par `MarkingStudentExam`, snapshots de la DB historique,
références historiques, homographie et `GetAnswerDetections`.

La DB source est hashée, copiée dans un répertoire temporaire, migrée uniquement
sur cette copie, puis hashée de nouveau à la fermeture. Le test réel a confirmé
qu'elle est restée byte-for-byte inchangée.

La sortie verbose et ce rapport ne contiennent que des agrégats. Aucun PDF,
scan, crop, nom, texte, réponse ou identifiant de copie n'est écrit dans le
dépôt. Le test est skipped lorsque `LAZYMARKING_CALIBRATION_DIR` ou
`LAZYMARKING_TEST_DB` manque.

## Volumétrie et copies non exploitables

| Mesure | Nombre |
|---|---:|
| PDF vus | 3 |
| Pages vues | 121 |
| Copies découvertes | 75 |
| Copies analysées | 75 |
| Copies ignorées | 0 |
| Pages QR finalement illisibles | 0 |
| Copies absentes de la DB | 0 |
| Copies incomplètes | 0 |
| Erreurs homographie ou pipeline | 0 |
| Cases analysées | 1 890 |

Une erreur de QR, une copie absente ou incomplète reste locale à son élément et
n'aurait pas fait échouer le reste du corpus. Une erreur d'intégrité du harness,
de la DB copiée ou de la correspondance entre ROI et détection reste fatale.

## Distribution MeanGray

Les quantiles utilisent une interpolation linéaire. Les valeurs sont arrondies
à deux décimales dans le rapport.

| Population | N | Min | Max | Moyenne | Médiane | p1 | p5 | p10 | p25 | p75 | p90 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Toutes | 1 890 | 18,18 | 255,00 | 188,47 | 252,99 | 26,08 | 33,91 | 39,25 | 68,77 | 253,50 | 253,93 | 254,20 | 254,72 |
| Checked | 600 | 18,18 | 147,63 | 54,02 | 46,47 | 22,57 | 28,69 | 31,45 | 37,29 | 63,80 | 91,12 | 103,25 | 126,15 |
| Unchecked | 1 290 | 154,01 | 255,00 | 251,00 | 253,33 | 187,51 | 247,59 | 252,50 | 252,97 | 253,68 | 254,09 | 254,37 | 254,80 |

Les groupes restent globalement bimodaux, mais la zone centrale n'est plus
vide : le maximum checked est 147,63 et le minimum unchecked 154,01. Le plus
petit écart au seuil vaut 2,37.

## Histogramme détaillé

| Bande MeanGray | Cases | Bande MeanGray | Cases |
|---|---:|---|---:|
| [0, 20[ | 2 | [75, 100[ | 63 |
| [20, 40[ | 198 | [100, 110[ | 15 |
| [40, 60[ | 226 | [110, 120[ | 14 |
| [60, 75[ | 73 | [120, 130[ | 4 |
| [130, 135[ | 2 | [135, 140[ | 1 |
| [140, 145[ | 0 | [145, 150[ | 2 |
| [150, 155[ | 1 | [155, 160[ | 0 |
| [160, 165[ | 0 | [165, 170[ | 0 |
| [170, 180[ | 5 | [180, 200[ | 30 |
| [200, 225[ | 10 | [225, 240[ | 8 |
| [240, 256[ | 1 236 |  |  |

## Distance au seuil 150

| Min | Max | Moyenne | Médiane | p1 | p5 | p10 | p25 | p75 | p90 | p95 | p99 |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 2,37 | 131,82 | 99,40 | 103,33 | 31,05 | 57,06 | 83,24 | 102,70 | 103,94 | 110,75 | 116,09 | 123,92 |

| Distance maximale | Détections | Pourcentage |
|---|---:|---:|
| ≤ 2 | 0 | 0 % |
| ≤ 5 | 3 | 0,16 % |
| ≤ 10 | 3 | 0,16 % |
| ≤ 15 | 4 | 0,21 % |
| ≤ 20 | 6 | 0,32 % |
| ≤ 25 | 10 | 0,53 % |
| ≤ 30 | 15 | 0,79 % |
| ≤ 40 | 43 | 2,28 % |
| ≤ 50 | 74 | 3,92 % |
| ≤ 75 | 147 | 7,78 % |

## Bandes candidates et charge par copie

| Delta | Candidats | Part | Checked | Unchecked | Moyenne/copie | Médiane/copie | Maximum/copie | Copies à zéro |
|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| 5 | 3 | 0,16 % | 2 | 1 | 0,04 | 0 | 2 | 73 / 75 |
| 10 | 3 | 0,16 % | 2 | 1 | 0,04 | 0 | 2 | 73 / 75 |
| 15 | 4 | 0,21 % | 3 | 1 | 0,05 | 0 | 2 | 73 / 75 |
| 20 | 6 | 0,32 % | 5 | 1 | 0,08 | 0 | 3 | 72 / 75 |
| 25 | 10 | 0,53 % | 8 | 2 | 0,13 | 0 | 3 | 70 / 75 |
| 30 | 15 | 0,79 % | 9 | 6 | 0,20 | 0 | 3 | 68 / 75 |

Même delta 30 ne transforme pas la revue en vérification générale : moins de
1 % des cases, 0,20 case par copie en moyenne, au plus 3 sur une copie et 68
copies sur 75 sans aucune candidate. Ces chiffres décrivent seulement la charge
de proximité au seuil, pas la qualité du rappel des erreurs humaines.

## Distribution inter-copies

| Métrique par copie | Copies | Min | Max | Médiane | p1 | p5 | p10 | p25 | p75 | p90 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| Médiane checked | 75 | 25,65 | 131,45 | 50,84 | 28,28 | 30,50 | 33,48 | 38,89 | 65,65 | 91,84 | 99,72 | 115,65 |
| Médiane unchecked | 75 | 190,17 | 254,32 | 253,32 | 196,62 | 244,44 | 252,98 | 253,19 | 253,58 | 253,82 | 253,89 | 254,09 |
| Distance minimale à 150 | 75 | 2,37 | 102,92 | 79,84 | 3,48 | 23,29 | 30,28 | 48,05 | 96,55 | 102,31 | 102,65 | 102,85 |

La variation inter-copies est significative. Certaines copies ont des médianes
checked beaucoup plus claires, et quelques médianes unchecked sont nettement
plus sombres que le mode principal proche de 253. Un seuil global peut donc être
plus sensible aux conditions de scan sur une fraction du corpus. Ce constat ne
modifie pas l'algorithme dans ce jalon.

## Métriques ROI exploratoires

Le harness relit chaque PNG aligné non annoté staged par `MarkingStudentExam` et
utilise exactement le rectangle `centre ± radius/2`. Le MeanGray production
reste canonique. Le calcul Go du gris est seulement recoupé à une tolérance de
1,1 niveau, qui couvre la différence d'arrondi avec OpenCV.

Pour chaque ROI, le test calcule sans influencer la classification :

- écart-type des niveaux de gris ;
- proportion de pixels strictement sous 128 ;
- minimum et maximum.

| Métrique ROI | Min | Max | Moyenne | Médiane | p90 | p95 | p99 |
|---|---:|---:|---:|---:|---:|---:|---:|
| Écart-type | 0,00 | 103,29 | 10,09 | 1,51 | 29,85 | 47,36 | 89,92 |
| Proportion sombre | 0,00 | 1,00 | 0,31 | 0,00 | 1,00 | 1,00 | 1,00 |
| Minimum | 0 | 255 | 166,13 | 244 | 252 | 253 | 255 |
| Maximum | 32 | 255 | 215,46 | 255 | 255 | 255 | 255 |

Une catégorie exploratoire « MeanGray < 75 et écart-type ≥ 80 » trouve 1 cas
sombre fortement hétérogène. Ces bornes ne constituent ni un algorithme proposé
ni un seuil calibré ; elles servent uniquement à prouver qu'une rature ou un
trait irrégulier peut former une catégorie d'inspection distincte de la
proximité à 150.

## Catégories pour inspection humaine

Trois catégories techniques peuvent alimenter une petite revue locale :

- A — les 15 détections à distance ≤ 30 de 150 ;
- B — les cases très sombres à forte variance, dont 1 cas avec les bornes
  exploratoires actuelles ;
- C — les copies aux médianes checked très hautes ou unchecked très basses,
  sélectionnées par quantiles inter-copies plutôt que par identité.

Un futur export opt-in peut résoudre la page alignée, découper la ROI avec une
marge et écrire seulement `candidate-0001.png`, `candidate-0002.png`, etc. dans
un `t.TempDir()` ou un répertoire privé hors dépôt. Un manifeste local peut
contenir le numéro séquentiel, MeanGray, état automatique, distance, écart-type,
proportion sombre et min/max. Il ne doit contenir ni identité, ni réponse
attendue, ni texte pédagogique.

## Interprétation et recommandation

MeanGray reste adapté à la **présélection des cas proches du seuil** : la charge
est faible et le corpus fournit maintenant un petit échantillon exploitable.
MeanGray seul n'est toutefois **pas suffisant pour couvrir toutes les formes de
doute visuel**, notamment les ratures très sombres et hétérogènes qui restent
loin de 150.

Aucun delta définitif ne doit être choisi sans vérité terrain. L'étape suivante
est une crop review locale ciblée sur la catégorie A à delta 30, complétée par
les catégories B et C. Les annotations humaines permettront ensuite de comparer
5/10/15/20/25/30 en termes de rappel utile et de charge, puis seulement de
recommander une première valeur de production.

## Validation et absence d'impact production

- `ambiguity_delta` production modifié : non ; il reste `NULL`.
- Seuil historique 150 modifié : non.
- `GetAnswerDetections`, `CountingPoints`, scores et PDF production modifiés :
  non.
- Métriques ROI supplémentaires : test opt-in uniquement.
- Données privées ajoutées au dépôt : non.
- Test réel exécuté : oui.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.
