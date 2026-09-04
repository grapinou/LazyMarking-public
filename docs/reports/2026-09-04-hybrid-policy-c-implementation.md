# Implémentation versionnée de la politique C

Date : 2026-09-04

## Résumé

La politique C est implémentée sous la version `detector-color-confidence-v1`. Les détecteurs restent identifiés par `hybrid-historical-v2-frozen-1` et aucun paramètre de détection n'a changé.

Les nouveaux jobs utilisent la nouvelle politique. Les jobs existants `detector-agreement-v1` conservent leur comportement : tout désaccord reste une revue. Les jobs historiques sans version continuent à utiliser la bande d'ambiguïté historique.

La décision automatique C est persistée dans `automatic_state`. Son origine est reconstruite sans ambiguïté à partir de la version de politique, de `historical_state`, `v2_state` et `color_signal`. Aucune colonne de motif supplémentaire n'a été ajoutée. Une migration remplace les contraintes d'insertion afin de vérifier cette cohérence en fonction de la politique du job.

La validation privée du job 10 reproduit exactement **165 acceptations automatiques, 13 revues et 3 copies concernées**, avec 0 faux positif automatiquement accepté dans la vérité complète. B05 possède dix désaccords avec les deux signaux positifs ; ses dix marques deviennent automatiques sous C et son résultat pédagogique attendu reste 9/10.

## Architecture retenue

Le chemin reste séparé en quatre responsabilités :

1. le détecteur historique calcule `mean_gray` et son état ;
2. V2 calcule indépendamment ses ratios, sous-signaux et état ;
3. une fonction de politique versionnée transforme ces signaux en `automatic_state` ou en obligation de revue ;
4. le scoring consomme uniquement l'état effectif, jamais une métrique visuelle.

La fonction de politique accepte explicitement une version. Le wrapper employé par les nouveaux traitements sélectionne `detector-color-confidence-v1`. La validation de cohérence utilise la même fonction de décision, ce qui évite deux interprétations divergentes dans le code Go.

Deux vecteurs d'états restent distincts :

- `answerDetectionStates` conserve l'état historique pour les adaptateurs de compatibilité ;
- `answerDetectionScoringStates` prend `automatic_state` lorsqu'il existe, sinon l'état historique provisoire pour un désaccord en attente.

Cette distinction est nécessaire parce que `detected_state` conserve volontairement le sens historique.

## Versions

| Élément | Version |
|:--|:--|
| Schéma de résultat | `2` |
| Algorithme de détection | `hybrid-historical-v2-frozen-1` |
| Ancienne politique conservée | `detector-agreement-v1` |
| Nouvelle politique par défaut | `detector-color-confidence-v1` |

Le handler de création de job utilisait déjà une constante de politique. Cette constante pointe maintenant vers C ; aucune mise à jour des lignes existantes n'est effectuée.

## Décision exacte par cas

| Historique | V2 | Couleur | `automatic_state` | Revue | Provenance reconstruite |
|--:|--:|:--:|:--:|:--:|:--|
| 0 | 0 | false | 0 | non | accord des détecteurs |
| 1 | 1 | false ou true | 1 | non | accord des détecteurs |
| 0 | 1 | true | 1 | non | confiance couleur V2 |
| 0 | 1 | false | `NULL` | oui | désaccord sans confiance couleur |
| 1 | 0 | false | `NULL` | oui | désaccord inverse |

Dans les deux cas revus, `review_reason='detector_disagreement'`. Le cas inverse 1/0 ne reçoit aucune préférence automatique.

Pour `detector-agreement-v1`, les deux formes de désaccord conservent `automatic_state=NULL` et `review_reason='detector_disagreement'`, y compris lorsque `color_signal=1`.

## Persistance et auditabilité

### Choix de schéma

Aucune nouvelle colonne n'est nécessaire. Pour un job C :

- `historical_state=0`, `v2_state=1`, `color_signal=1`, `automatic_state=1`, sans motif de revue, identifie nécessairement une décision `v2_color_confidence` ;
- un accord est identifié par l'égalité des deux états ;
- une revue est identifiée par `automatic_state=NULL` et `review_reason='detector_disagreement'` ;
- `review_policy_version` fixe l'interprétation durable de ces champs.

Ajouter un texte de motif automatique aurait dupliqué une information déjà déterministe. La contrainte SQL vérifie désormais la relation avec le job parent, ce qui empêche par exemple de persister silencieusement une acceptation couleur C sous un job `detector-agreement-v1`.

### Migration 0043

`0043_add_color_confidence_review_policy.sql` :

- autorise les créations de jobs `detector-color-confidence-v1` avec le même ensemble complet de paramètres V2 ;
- conserve l'autorisation de `detector-agreement-v1` et des jobs historiques sans métadonnées hybrides ;
- remplace le trigger de détection par une contrainte dépendant de la politique du job ;
- autorise sous C le seul désaccord automatique `0/1` avec `color_signal=1` ;
- exige une revue pour `0/1` sans couleur et pour `1/0` ;
- conserve le trigger d'immutabilité existant des versions et paramètres.

La migration est additive du point de vue des données : aucune ligne, colonne ou valeur historique n'est supprimée ou réécrite. Son retour arrière restaure les contraintes de la version 0042.

## État effectif et scoring

Le calcul initial d'un nouveau job utilise le vecteur d'états automatiques :

- accord : état commun ;
- confiance couleur C : état V2 positif ;
- revue requise : état historique uniquement comme état provisoire non publiable.

Les requêtes de recalcul et d'artefacts utilisent maintenant l'ordre explicite :

```sql
COALESCE(reviewed_state, automatic_state, detected_state)
```

Cet ordre garantit :

1. l'autorité finale de la décision humaine ;
2. la conservation d'une acceptation couleur automatique lors du recalcul d'une question ou de la régénération des PDF ;
3. le repli historique pour les anciens jobs et pour l'état provisoire d'un désaccord non résolu.

Les fonctions pédagogiques reçoivent uniquement un vecteur binaire. `mean_gray`, `dark_ratio`, `chroma_ratio`, `grayscale_signal` et `color_signal` ne sont jamais transmis au calcul des points.

## Sélection de revue

Les quatre requêtes qui identifient ou comptent les candidats reconnaissent maintenant deux politiques hybrides :

- `detector-agreement-v1` ;
- `detector-color-confidence-v1`.

Dans les deux cas, la file sélectionne `review_reason='detector_disagreement'`. La différence est décidée et persistée en amont : sous C, un désaccord couleur n'a aucun motif de revue, tandis que les deux désaccords à vérifier le conservent.

La branche `review_policy_version IS NULL` et son critère `mean_gray ± ambiguity_delta` restent inchangés pour les jobs historiques.

## Cycle de vie

Les protections précédentes sont conservées et testées avec les deux politiques hybrides :

- un candidat non résolu maintient le statut de revue en attente ;
- les liens d'artefacts définitifs sont masqués ;
- le téléchargement direct est refusé ;
- la régénération est refusée tant que `pending_candidates > 0` ;
- la dernière décision humaine recalcule la question et la copie avec les états effectifs ;
- elle avance la révision de revue, déclenche la régénération puis synchronise la révision des artefacts ;
- un job sans candidat obtient le statut « aucune revue nécessaire » et expose directement ses résultats normaux.

Le changement de `COALESCE` est essentiel pour que les réponses automatiques couleur d'une même question restent positives lors de la revue d'une autre réponse.

## Fichiers et requêtes modifiés

### Production

- `internal/handlers/tools/getAnswersState.go` : décision versionnée C et vecteur d'état pour le scoring ;
- `internal/handlers/tools/markingRuntimeResult.go` : constantes de versions et validation de cohérence par politique ;
- `internal/handlers/tools/markingStudentExam.go` : consommation de l'état automatique effectif ;
- `db/migrations/0043_add_color_confidence_review_policy.sql` : contraintes versionnées ;
- `db/query/markingReviews.sql` : reconnaissance de C et priorité à `automatic_state` ;
- `internal/db/markingReviews.sql.go` : code régénéré par sqlc.

### Tests

- `internal/handlers/tools/answerDetectionHybrid_test.go` ;
- `internal/db/markingReviews_test.go` ;
- `internal/db/markingResults_test.go` ;
- `internal/handlers/marking/hybridLifecycle_test.go` ;
- adaptations minimales des schémas synthétiques dans les tests de revue et de crop.

### Requêtes affectées

La sélection est étendue dans :

- `ListMarkingReviewCandidates` ;
- `ListPendingMarkingReviewCandidates` ;
- `GetMarkingReviewSummary` ;
- `GetMarkingArtifactsRegenerationTarget`.

La priorité `reviewed → automatic → historical` est appliquée dans :

- `GetEffectiveAnswerDetection` ;
- `ListMarkingReviewCandidates` ;
- `GetMarkingAnswerReviewTarget` ;
- `ListEffectiveQuestionAnswersForReview` ;
- `ListEffectiveMarkingAnswersForArtifacts`.

## Tests ajoutés ou étendus

La couverture vérifie :

- les cinq décisions demandées, dont les deux variantes de l'accord positif ;
- l'acceptation automatique du seul désaccord couleur 0/1 ;
- la revue du faux positif de forme gris seul ;
- la revue prudente du cas 1/0 ;
- le rejet d'une version inconnue ;
- le comportement inchangé de `detector-agreement-v1`, même avec un signal couleur ;
- la sélection SQL exacte des deux candidats C parmi les six formes synthétiques ;
- la lecture effective d'une décision couleur avec `detected_state=0` et `automatic_state=1` ;
- l'emploi de cet état effectif dans la liste utilisée par le recalcul ;
- les protections de téléchargement avec les deux versions hybrides ;
- la résolution du dernier candidat, la régénération et la synchronisation des révisions avec les deux versions ;
- la migration 0043 dans les bases synthétiques ;
- le comportement historique, déjà couvert, de la bande d'ambiguïté et des jobs sans version.

Les tests du détecteur figé existants restent inchangés et passent. Aucun seuil ni calcul de pixels n'a été modifié.

## Validation privée

### Job 10

La base `testdata/real/app.db` a été interrogée en lecture seule. Une copie privée sous `runtime/diagnostics/hybrid-policy-c-implementation/app.db` a servi à tester la migration ; l'original n'a pas été modifié.

| Mesure | Résultat |
|:--|--:|
| Désaccords historiques du job 10 | 178 |
| Acceptés automatiquement par C | 165 |
| Candidats C | 13 |
| Copies avec candidat | 3 |
| TP automatiques selon la vérité complète | 165 |
| FP automatiques | 0 |
| Vraies coches en revue | 12 |
| Cases vides en revue | 1 |

Le job 10 reste persisté sous `detector-agreement-v1` et n'a pas été réinterprété. Le faux positif `35/7/1`, `historique=0 / V2=1 / color_signal=0`, reste candidat à revue sous C.

### B05

Les dix marques de B05 (`student_exam_id=31`) ont toutes :

- `historical_state=0` ;
- `v2_state=1` ;
- `grayscale_signal=1` ;
- `color_signal=1`.

C les accepte donc toutes automatiquement et ne crée aucune revue pour cette copie. La reconnaissance correspond à la vérité visuelle 10/10. Le scoring pédagogique inchangé continue à considérer Q4 comme fausse, d'où le résultat attendu **9/10**. Aucune règle spéciale B05 n'existe.

### Corpus privés existants

L'exécution du code Go sur les images privées reproduit les décisions figées :

- hold-out : historique `96 VP / 476 VN / 0 FP / 68 FN`, V2 `162 VP / 474 VN / 2 FP / 2 FN` ; C produit 7 revues ;
- développement : historique `130 VP / 528 VN / 0 FP / 62 FN`, V2 `191 VP / 524 VN / 4 FP / 1 FN` ; C produit 14 revues ;
- B05 : historique 0/10, V2 10/10.

Les six faux positifs V2 de ces deux corpus restent en revue. Les accords négatifs extrêmes des copies 45 et 46 restent inchangés, conformément au périmètre.

### Nouveau job réel

La création de tout nouveau job renseigne désormais `detector-color-confidence-v1`. La migration a été appliquée avec succès sur une copie de la base réelle, puis testée en descente et remontée. Le scan `testdata/real/scans/6eB.pdf` peut donc être soumis comme nouveau job sans réinterpréter le job 10. Aucun nouveau job n'a été écrit dans la base réelle pendant cette validation ; les métriques identiques du job 10 fournissent le contrôle déterministe attendu de 13 candidats sur 3 copies.

## Résultats des validations

| Validation | Résultat |
|:--|:--|
| `sqlc generate -f db/sqlc.yaml` | succès |
| `go test ./...` | succès |
| `./scripts/check.sh` | succès |
| Replay migrations jusqu'à 0043 | succès |
| Migration 0043 down puis up sur copie privée | succès |
| `git diff --check` | succès |

## Points d'attention

- La validation de C reste concentrée sur les corpus disponibles ; un futur artefact coloré pourrait encore produire un faux positif.
- Le job 10 ne doit pas être mis à jour en base : son comportement historique reste celui de `detector-agreement-v1`.
- `detected_state` demeure l'état historique par contrat. Toute nouvelle lecture d'état effectif doit respecter la priorité `reviewed_state`, `automatic_state`, `detected_state`.
- Un désaccord sans état automatique possède toujours un score technique provisoire avant revue ; les protections de cycle de vie empêchent sa publication comme résultat définitif.
- La charge nominale future reste à mesurer sous consigne contrôlée.

## Recommandation du prochain jalon

Le prochain jalon recommandé est un **essai opérationnel prospectif de la politique versionnée**, sur un nouveau job et sous consigne élève nominale : stylo bille bleu ou noir, feutre foncé, marque clairement visible dans la case.

Il devra mesurer avant toute nouvelle évolution :

1. le nombre de revues par classe et par copie ;
2. la vérité humaine de toutes les acceptations automatiques échantillonnées ou contrôlées ;
3. tout faux positif coloré ;
4. les accords négatifs manqués ;
5. le bon déroulement réel de la dernière revue et de la publication des artefacts.

Les améliorations UX, consignes imprimées, renommage PDF et prévention du réupload restent hors de ce jalon.
