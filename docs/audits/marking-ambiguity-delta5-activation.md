# Activation de la politique de revue Marking delta 5

## Valeur activée et origine

Les nouveaux jobs Marking nouveau-format snapshotent désormais
`ambiguity_delta = 5.0` en même temps que `detection_threshold = 150.0`, la
version de schéma résultat et la version d'algorithme.

`MarkingAmbiguityDelta` est documentée comme la première valeur calibrée sur un
corpus réel : 75 copies, 121 pages et 1 890 cases, puis une revue humaine ciblée
de 20 crops. La bande ±5 capturait l'unique divergence auto/humain observée avec
3 candidats, soit 0,16 % des cases. Cette valeur est une politique produit
versionnée, pas une propriété mathématique universelle de MeanGray.

## Snapshot des nouveaux jobs et legacy

`CreateMarkingJob` écrit le delta fourni dans `marking_jobs` au moment de la
création. Le handler de production lui fournit exclusivement la constante
`MarkingAmbiguityDelta`. La contrainte et le trigger 0040 rendent ensuite toute
valeur non-NULL immutable.

Les jobs existants conservent `ambiguity_delta IS NULL`. Aucun backfill n'est
effectué. NULL signifie que le job ne possédait aucune politique de revue
snapshotée ; il n'est jamais interprété comme zéro et ne produit aucune file
automatique artificielle.

## Détection automatique inchangée

Le seuil historique reste 150 et `GetAnswerDetections` reste inchangé :

- `mean_gray < 150` donne `checked` ;
- `mean_gray >= 150` donne `unchecked`.

Le delta ne modifie jamais `detected_state`. La régression synthétique 154,01
reste donc automatiquement unchecked tout en étant candidate à la revue.

## Fonction pure V1

`IsMarkingDetectionAmbiguous(meanGray, threshold, ambiguityDelta)` centralise le
contrat Go :

`abs(meanGray - threshold) <= ambiguityDelta`

La borne est inclusive. MeanGray et threshold doivent être finis et compris
dans `[0,255]`; le delta doit être fini et positif ou nul. Les tests couvrent
144,99, 145, 149,99, 150, 154,01, 155 et 155,01 pour threshold 150 et delta 5,
ainsi que NaN, infinis et domaines invalides.

## Read model SQL

Toutes les lectures applicatives exigent un job possédé avec `status =
'success'`. Un job running ou failed n'expose donc pas ses résultats partiels
comme une file de revue normale. Le lifecycle technique `running / success /
failed` n'est pas modifié et reste orthogonal au lifecycle de revue.

### Candidats

`ListMarkingReviewCandidates` utilise exclusivement
`marking_jobs.detection_threshold` et `marking_jobs.ambiguity_delta`, jamais les
constantes courantes de l'application. La formule SQL est la même borne
inclusive que la fonction pure et n'est active que lorsque le delta snapshoté
est non-NULL.

Chaque ligne fournit : détection, copy result, student exam, indices question
et réponse snapshotés, état détecté, état reviewed nullable, état effectif,
MeanGray, threshold, delta, ainsi que l'identifiant, le numéro de page et la clé
de la page alignée. La page est retrouvée depuis la répartition historique des
questions dans `student_exam_page_content`, sans jointure vers les tables
vivantes Question/Answer.

### Pending

`ListPendingMarkingReviewCandidates` applique le même scope et la même formule,
puis conserve seulement les détections sans `marking_answer_review`. Une review
qui confirme `detected_state` et une review override retirent toutes deux la
ligne du pending.

Une review humaine peut toujours cibler une détection non ambiguë. Elle reste
valide et consultable via le modèle effectif existant, mais elle n'entre ni dans
la file automatique ni dans son résumé.

### Résumé et statut dérivé

`GetMarkingReviewSummary` retourne `ambiguity_delta`, `total_candidates`,
`reviewed_candidates` et `pending_candidates`, avec ownership et statut success.
`DeriveMarkingReviewStatus` produit sans colonne redondante :

- delta NULL : `legacy_unavailable` ;
- delta actif et zéro candidat : `no_review_needed` ;
- au moins un pending : `pending` ;
- tous les candidats revus : `completed`.

Un job peut ainsi être `status = success` et avoir une review `pending`.

## Lifecycle, scores et artefacts

Ce jalon est un snapshot de politique et un read model uniquement :

- score modifié : non ;
- recalcul des question/copy results : non ;
- `review_revision` modifié : non, il reste normalement 0 ;
- `artifacts_revision` modifié : non ;
- PDF modifié ou régénéré : non ;
- OpenCV, homographie ou `GetAnswerDetections` modifié : non ;
- production des pages alignées modifiée : non ;
- UX, route ou handler de décision humaine ajouté : non.

## Migration et SQLC

Aucune migration n'est ajoutée et la migration 0040 n'est pas modifiée. Son
delta nullable immutable, ses reviews et ses révisions suffisent au contrat.
Les queries sont ajoutées au fichier Marking review existant et les fichiers Go
sont générés par `sqlc generate -f db/sqlc.yaml`, sans édition manuelle.

## Tests

Les tests synthétiques couvrent :

- snapshot 150 / 5 et versions lors d'un nouvel upload ;
- immutabilité DB ultérieure du delta ;
- fonction pure, bornes inclusives et entrées invalides ;
- équivalence des frontières SQL 145–155 avec la fonction pure ;
- régression 154,01 unchecked et ambiguë ;
- candidats/pending et page alignée historique ;
- review confirmée, override et état effectif ;
- review non ambiguë autorisée mais exclue du compteur ;
- résumé pending puis completed ;
- legacy NULL indisponible sans candidats ;
- refus des lectures cross-user ;
- absence de lecture normale pour un job non-success ;
- `review_revision` et `artifacts_revision` inchangés.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés

- `db/query/markingJobs.sql`
- `db/query/markingReviews.sql`
- `internal/db/markingJobs.sql.go` — généré
- `internal/db/markingReviews.sql.go` — généré
- `internal/db/markingReviewReadModel.go`
- tests DB de création, read model et lifecycle review
- `internal/handlers/marking/handlers.go`
- `internal/handlers/marking/handlers_test.go`
- `internal/handlers/tools/markingAmbiguity.go`
- `internal/handlers/tools/markingAmbiguity_test.go`
- ce rapport
