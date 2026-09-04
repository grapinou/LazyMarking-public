# Implémentation du détecteur hybride de réponses

Date : 2026-09-04

## Résumé

La détection hybride est implémentée pour les nouveaux jobs sous la version `hybrid-historical-v2-frozen-1` et la politique de revue `detector-agreement-v1`. Le détecteur historique et V2 sont exécutés indépendamment. Un accord produit un `automatic_state` persistant ; un désaccord laisse cet état automatique à `NULL`, porte le motif `detector_disagreement` et devient obligatoirement candidat à la revue existante.

La migration 0042 est additive. Les anciennes lignes gardent les nouveaux champs à `NULL`, les anciens jobs restent sélectionnés par leur bande historique `mean_gray ± ambiguity_delta`, et aucune détection ancienne n'est recalculée ou réinterprétée.

Le code Go reproduit exactement les 640 décisions historiques et les 640 décisions V2 du hold-out figé. Il retrouve la matrice attendue : historique `96 VP / 476 VN / 0 FP / 68 FN`, V2 `162 VP / 474 VN / 2 FP / 2 FN`, et 68 désaccords. B05 conserve 0/10 au détecteur historique mais obtient 10/10 au V2, donc ses dix marques deviennent des revues obligatoires ; avec les décisions humaines correctes, le scoring pédagogique existant produit 9/10, Q4 restant fausse.

## Architecture existante auditée

Avant modification, le chemin pertinent était :

`page alignée` → `GetAnswerDetections` → moyenne de la ROI carrée → `detected_state` → résultats question/copie persistés → sélection par bande d'ambiguïté → `marking_answer_reviews` → `COALESCE(reviewed_state, detected_state)` → recalcul par le moteur de scoring existant.

- `GetAnswerDetections` ouvrait l'image en niveaux de gris, mesurait la moyenne de la ROI carrée historique et appliquait le seuil strict `< 150`.
- `marking_answer_detections` conservait `detected_state` et `mean_gray`.
- `marking_jobs` conservait déjà `result_schema_version`, `marking_algorithm_version`, `detection_threshold` et `ambiguity_delta`.
- les candidats à revue étaient les détections dont la moyenne se trouvait dans la bande `threshold ± ambiguity_delta` ; les anciennes revues restaient attachées à l'identifiant immuable de détection.
- l'état effectif était et reste `COALESCE(marking_answer_reviews.reviewed_state, marking_answer_detections.detected_state)`.
- le recalcul récupère uniquement le vecteur d'états effectifs, appelle le moteur pédagogique existant, met à jour la question puis la somme de la copie. Il ne lit aucune métrique visuelle.

## Architecture hybride

L'unité de reconnaissance reste dans `internal/handlers/tools/getAnswersState.go`, sans logique distribuée dans les handlers. Elle produit trois résultats cohérents :

1. historique : `mean_gray` et état `< 150` ;
2. V2 : `dark_ratio`, `chroma_ratio`, les deux sous-signaux et leur OR ;
3. politique : comparaison des deux états, présence éventuelle d'un état automatique et motif de revue.

Le handler de création se limite à associer au job l'identité et les paramètres figés. La persistance transforme ensuite le résultat structuré en colonnes auditables. Les requêtes de revue choisissent leur critère selon `review_policy_version`.

## Détecteur historique

La mesure historique est inchangée : même chargement OpenCV en niveaux de gris, même `MarkingAnswerMeasurementRect`, même moyenne de ROI et même comparaison stricte `mean_gray < 150`. `GetAnswersState` reste un adaptateur compatible qui renvoie le vecteur conservateur historique ; aucun appelant historique n'obtient silencieusement la décision V2.

## Détecteur V2 figé

Les constantes codées et persistées sont exactement :

| Paramètre | Valeur |
|:--|--:|
| rayon de ROI | `0.40 × rayon` |
| pixel sombre | `gray < 220` |
| signal gris | `dark_ratio >= 0.10` |
| pixel chromatique | `max(R,G,B)-min(R,G,B) > 12` |
| signal couleur | `chroma_ratio >= 0.05` |
| décision V2 | `grayscale_signal OR color_signal` |

Le masque parcourt les coordonnées entières vérifiant `dx² + dy² <= (0.40r)²`. La conversion est `0.299R + 0.587G + 0.114B`. OpenCV fournit les pixels en BGR ; l'implémentation les remet explicitement dans l'ordre RGB avant les calculs. Aucun paramètre n'a été recalibré.

## Politique accord/désaccord

| Historique | V2 | `automatic_state` | Revue |
|--:|--:|--:|:--|
| 0 | 0 | 0 | non |
| 1 | 1 | 1 | non |
| 0 | 1 | `NULL` | oui |
| 1 | 0 | `NULL` | oui |

`detected_state` reste non nullable pour compatibilité avec le modèle de résultat et reçoit, pour un nouveau job, l'état historique conservateur. En cas de désaccord, il s'agit seulement du support provisoire avant revue : `automatic_state=NULL` rend explicite qu'aucune décision automatique définitive n'existe, `review_reason` impose la présence dans la file, et l'UX signale le job comme en attente. Aucune préférence automatique n'est attribuée à historique ou V2.

Une fois la revue saisie, `reviewed_state` demeure l'autorité finale. Le dernier candidat traité déclenche la régénération existante des artefacts à partir des états effectifs.

## Migration et modèle de persistance

La migration additive `0042_add_hybrid_answer_detection.sql` ajoute au job :

- `review_policy_version` ;
- les cinq paramètres V2 (`v2_roi_radius_ratio`, seuil pixel sombre, seuil ratio sombre, seuil pixel chromatique, seuil ratio chromatique).

Elle ajoute à chaque détection :

- `historical_state` et le `mean_gray` déjà existant ;
- `v2_state`, `dark_ratio`, `chroma_ratio`, `grayscale_signal`, `color_signal` ;
- `automatic_state`, nullable seulement lors d'un désaccord ;
- `review_reason`, égal à `detector_disagreement` pour un désaccord.

Tous les nouveaux champs sont nullables afin de préserver les données antérieures. Des contraintes et triggers vérifient qu'une ligne est soit entièrement historique, soit un enregistrement hybride complet et cohérent. Les paramètres de job sont complets ou tous absents, validés dans leurs domaines et immuables après création. Aucune donnée historique n'est supprimée.

Versions des nouveaux jobs :

- schéma de résultat : `2` ;
- algorithme : `hybrid-historical-v2-frozen-1` ;
- politique : `detector-agreement-v1`.

## Requêtes SQL et compatibilité historique

Deux créations explicites évitent de changer le contrat des helpers historiques : `CreateHybridMarkingJob` et `CreateHybridMarkingAnswerDetection`. Les requêtes anciennes restent disponibles pour les tests, outils et lignes de format antérieur.

Les trois lectures de file — liste complète, liste en attente et résumé — appliquent :

- job hybride : `review_reason = 'detector_disagreement'` ;
- job antérieur sans `review_policy_version` : bande historique `ABS(mean_gray-detection_threshold) <= ambiguity_delta`.

L'heuristique expérimentale V2 à 103 alertes n'est présente nulle part dans le code. Pour les nouveaux jobs, `ambiguity_delta=0` est seulement conservé afin de satisfaire le contrat historique de disponibilité de la revue ; il ne participe jamais à la sélection grâce au branchement explicite sur la politique.

## Scoring

Aucune règle pédagogique et aucun fichier du package `markingscoring` n'ont été modifiés. Le calcul continue à recevoir exclusivement un tableau binaire : état persistant pour une décision automatique, puis `reviewed_state` via `COALESCE` lorsqu'un humain intervient. `mean_gray`, les ratios, les sous-signaux et les versions ne sont jamais transmis au calcul des points.

Les tests transactionnels existants de revue, recalcul question/copie, idempotence, concurrence et régénération continuent de passer. Un nouveau test vérifie aussi qu'un désaccord `0/1` revu à 1 remplace l'état provisoire 0 dans l'état effectif.

## Fichiers modifiés

Fichiers fonctionnels et de schéma :

- `db/migrations/0042_add_hybrid_answer_detection.sql` ;
- `db/query/markingJobs.sql`, `markingResults.sql`, `markingReviews.sql` ;
- fichiers sqlc régénérés sous `internal/db/` ;
- `internal/config/config.go` ;
- `internal/db/markingResultsRepository.go` ;
- `internal/handlers/tools/getAnswersState.go` ;
- `internal/handlers/tools/markingRuntimeResult.go` ;
- `internal/handlers/marking/handlers.go`.

Tests adaptés ou ajoutés :

- `internal/handlers/tools/answerDetectionHybrid_test.go` ;
- `internal/db/markingResults_test.go` ;
- `internal/db/markingReviews_test.go` ;
- `internal/handlers/marking/handlers_test.go` ;
- `internal/handlers/marking/review_test.go`.

Le script local de validation se trouve sous `runtime/diagnostics/answer-detection-hybrid-implementation/validate.go` et reste ignoré. Aucun crop ni scan réel n'a été ajouté à Git. Des modifications préexistantes hors de ce périmètre ont été laissées intactes.

## Tests automatisés

Les tests synthétiques couvrent :

- le seuil historique strict à 150 via les tests existants ;
- case blanche et marque noire ;
- marques bleue et verte, et signal coloré clair ;
- seuils pixel stricts `gray < 220` et `chroma > 12` ;
- seuils de ratio inclusifs `>= 0.10` et `>= 0.05` sur un disque de 49 pixels ;
- exclusion d'un pixel sombre hors du disque `0.40r` ;
- combinaison OR des signaux ;
- les quatre combinaisons de politique, dont `historique=1/V2=0` ;
- sélection SQL des seuls désaccords, y compris les deux directions ;
- remplacement humain de l'état provisoire et compatibilité de l'ancien mécanisme de revue.

Résultats :

- `sqlc generate -f db/sqlc.yaml` : succès ;
- `go test ./...` : succès pour tous les packages ;
- `./scripts/check.sh` : succès (`go mod verify`, replay Goose jusqu'à 0042, formatage, vet, tests, build et contrôle du diff) ;
- `git diff --check` : succès.

## Validation privée

La validation a exécuté directement `GetAnswerDetections` sur les pages alignées privées, en utilisant les coordonnées et rayons déjà présents dans les CSV locaux. Les originaux et les artefacts expérimentaux n'ont pas été modifiés.

### Hold-out figé

Les 640 décisions sont identiques, case par case, à `holdout-measurements.csv` :

| Méthode | VP | VN | FP | FN |
|:--|--:|--:|--:|--:|
| Historique Go | 96 | 476 | 0 | 68 |
| V2 Go | 162 | 474 | 2 | 2 |

La politique produit exactement 68 désaccords. Elle retrouve donc la simulation documentée : après revue parfaite, 162 VP, 476 VN, 0 FP et 2 FN.

Les quatre erreurs V2 ont été vérifiées implicitement par la comparaison case par case :

- copie 9 / Q4 / A2 : `historique=0`, `V2=1`, donc revue ;
- copie 17 / Q6 / A3 : `historique=0`, `V2=1`, donc revue ;
- copie 46 / Q5 / A2 : `historique=0`, `V2=0`, donc reste invisible ;
- copie 46 / Q6 / A3 : `historique=0`, `V2=0`, donc reste invisible.

Les sept copies nominales sont incluses dans cette comparaison exhaustive ; aucune marque nominale n'est manquée et les deux seuls désaccords nominaux sont les deux cases vides ci-dessus.

### Corpus de développement et validation antérieure

Sur les 720 cases privées, V2 reproduit `191 VP / 524 VN / 4 FP / 1 FN`. Le détecteur historique exécuté par le code de production donne `130 VP / 528 VN / 0 FP / 62 FN` et 65 désaccords.

Un écart historique unique avec le CSV expérimental a été identifié : copie 51 / Q9 / A1, moyenne OpenCV de production `149.970000`, donc positive selon le seuil strict, alors que l'artefact Python l'avait enregistrée négative avec une conversion RGB flottante légèrement différente. Cet écart ne touche ni V2 ni le hold-out et ne constitue pas un changement du détecteur historique : l'implémentation conserve précisément la lecture OpenCV déjà utilisée en production.

### B05 et B08

B05 : les dix marques humainement visibles donnent `historique=0` et `V2=1`. Elles deviennent dix candidats de revue sans règle spéciale. Après validation humaine des dix états, le moteur pédagogique existant conserve neuf réponses correctes ; Q4/A1 reste sélectionnée face à A3 attendue, soit **9/10**.

B08 est incluse dans le même corpus de calibration : ses dix marques sombres restent détectées par les deux lectures et ne créent pas de revue inutile.

## Points d'attention

- Un désaccord possède nécessairement un `detected_state` technique à cause du schéma historique non nullable, mais `automatic_state=NULL`, le motif persistant et le statut de revue empêchent de le présenter comme définitif.
- Les deux marques rose extrêmement pâle de la copie 46 restent invisibles aux deux méthodes ; elles sont hors du domaine nominal annoncé mais restent des cas de stress utiles.
- Les paramètres sont figés et auditables, mais l'échantillon nominal reste petit et défini post-hoc.
- La qualité et le temps de revue humaine ne sont pas encore mesurés prospectivement.
- `ambiguity_delta` subsiste pour la compatibilité de l'ancien cycle de revue ; il n'est pas un critère des jobs hybrides.
- La consigne élève, le renommage de PDF et la prévention de réupload d'un corrigé restent volontairement hors périmètre.

## Recommandation pour le prochain jalon

Le prochain jalon doit être une validation prospective opérationnelle de cette version figée, sous consigne « stylo bille bleu ou noir, feutre foncé, marque clairement visible dans la case ». Il devra mesurer le taux de désaccord, le temps et les erreurs de revue, la spécificité finale, les résultats par scanner et la complétude du parcours jusqu'aux artefacts régénérés.

La consigne imprimée pourra ensuite faire l'objet du jalon UX séparé prévu. Aucun recalibrage ne doit être entrepris à partir des quatre erreurs du hold-out avant la constitution d'un nouveau jeu de développement distinct.

Aucun commit n'a été effectué.
