# Revue Marking et recalcul transactionnel effectif

## Service créé

`ApplyMarkingAnswerReview` est une primitive applicative sans handler HTTP. Elle
reçoit l'utilisateur, le job, la détection, le nouvel état humain, la révision
attendue de la review de réponse et la `review_revision` attendue du job.

Toute l'opération s'exécute dans une transaction SQLite unique : validation du
scope, création ou modification de review, reconstruction des états effectifs,
recalcul de la question, recomposition du score de copie, publication des
révisions puis commit. Toute erreur provoque un rollback.

## Préconditions et ownership

La cible SQL exige simultanément :

- job appartenant à l'utilisateur et `status = success` ;
- answer detection rattachée à ce job ;
- copy result `outcome = corrected` ;
- snapshot `student_exam_content` du même propriétaire ;
- `reviewed_state` dans `{0,1}` validé par le service et le schéma.

Les jobs running/failed, les copies incomplete/not_seen/error, les utilisateurs
étrangers et les answer detections d'un autre job sont refusés sans révéler la
cible. L'écriture ne contient volontairement aucun critère d'ambiguïté : une
review hors de la bande automatique reste autorisée.

## États automatique, humain et effectif

Le contrat reste strict :

`effective_state = COALESCE(reviewed_state, detected_state)`

`detected_state` n'est jamais mis à jour et reste la décision automatique
historique immutable. Une confirmation humaine identique est une vraie review.
Une review existante reçoit un UPDATE ownership-aware avec sa révision attendue;
une première review reçoit un INSERT.

Si une review existante reçoit exactement le même état avec les mêmes versions
attendues, le service retourne un no-op : aucune révision, date ou note n'est
modifiée. Une version job ou answer stale reste un conflit, même pour cette
demande idempotente.

## Règle de notation unique et snapshot

La règle pure d'une question est extraite dans
`markingscoring.ScoreQuestion`. `CountingPoints`, utilisé par la correction
automatique, appelle désormais cette fonction; le recalcul de review appelle la
même fonction. Les cas correct, partial et incorrect ne sont donc pas dupliqués.

Le service décode exclusivement le QCM snapshoté dans
`student_exam_content.content`. Il récupère les réponses attendues et les points
de la question par `question_index`, puis relit toutes ses détections dans
l'ordre `answer_index` avec leur état effectif. Aucune table vivante Question,
Answer ou Point n'intervient.

Un test de confirmation force un recalcul depuis les états effectifs et vérifie
que la note persistée reste identique au résultat automatique. Le résultat
automatique original reste toujours reconstructible depuis les
`detected_state`, le snapshot historique et cette même règle; aucune colonne
`original_score` n'est ajoutée.

## Recalcul question et copie

Seule la question concernée reçoit un nouvel état et de nouveaux
`score_half_units`; son `question_index`, ses `total_points` et ses détections
originales restent inchangés. Les CHECK existants continuent d'imposer :

- incorrect : 0 half-unit ;
- partial : `total_points` half-units ;
- correct : `2 * total_points` half-units.

Le score de copie est ensuite recomposé avec la somme de **tous** les
`marking_question_results`. Aucun calcul différentiel ancien score ± delta
n'est utilisé. Les reviews précédentes restent donc intégrées lors de chaque
recalcul, et une modification de Q2 laisse Q1 inchangée.

## Optimistic locking

Deux niveaux sont appliqués :

- la révision de `marking_answer_reviews` empêche deux éditions concurrentes de
  la même réponse ;
- `marking_jobs.review_revision` est publiée en dernier par un UPDATE
  ownership-aware, `status = success`, limité à la révision globale attendue.

Si l'UPDATE global n'affecte pas exactement une ligne, toute la transaction — y
compris une review déjà insérée et les scores recalculés — est rollbackée. Deux
réponses différentes éditées depuis la même version job ne peuvent donc pas
committer silencieusement. Un retry avec la nouvelle version réussit.

## `review_revision` et fraîcheur des artefacts

Chaque décision humaine non idempotente incrémente `review_revision` une fois.
`artifacts_revision` suit exactement la règle monotone convenue :

- si l'état effectif change, elle ne bouge pas ;
- si l'état effectif ne change pas et que les artefacts étaient current avant
  la transaction, elle avance avec la nouvelle `review_revision` ;
- si les artefacts étaient déjà stale, une confirmation ne les rattrape pas.

Une confirmation `detected=1, reviewed=1` garde donc les artefacts current. Un
override `detected=0, reviewed=1` les rend stale. Un changement effectif qui
conserve le même score de question les rend également stale, car le marquage du
PDF peut être faux même si le total est identique. Revenir ultérieurement vers
l'état automatique ne rattrape jamais automatiquement une révision stale.

Aucun PDF n'est régénéré dans ce jalon.

## Summary et lifecycle

Après commit, les queries existantes candidats/pending/summary reflètent
naturellement la review. Une review non ambiguë n'entre toujours pas dans leurs
compteurs automatiques.

Le lifecycle `running / success / failed` ne change pas. La review et la
fraîcheur des artefacts restent des dimensions orthogonales; aucun statut review
redondant n'est persisté.

## Tests

Les tests synthétiques couvrent :

- régression 154,01 : auto unchecked, review checked, question/copie
  recalculées, pending consommé, artefacts stale ;
- confirmation sans changement et retry idempotent ;
- review non ambiguë MeanGray 40 avec recalcul mais hors compteur candidat ;
- correct/partial/incorrect et conversions exactes en half-units ;
- changement effectif conservant le même score, mais rendant les artefacts
  stale ;
- plusieurs questions et somme complète de copie ;
- plusieurs reviews successives utilisant tous les états effectifs ;
- stale restant stale après une confirmation ;
- ownership, cross-job et workflows non-success/non-corrected ;
- conflit answer, conflit global entre réponses différentes et retry ;
- rollback total après insertion/recalcul tentés mais publication globale
  refusée ;
- équivalence du recalcul sans override effectif avec le résultat automatique.

## Impact et migration

- PDF modifié : non.
- UX, route ou handler web modifié : non.
- OpenCV, seuil 150 ou ambiguity delta modifié : non.
- Migration ajoutée : non.
- Migration 0040 modifiée : non.

## Validation

- `sqlc generate -f db/sqlc.yaml` : succès.
- `./scripts/check.sh` : succès.
- `go test -race ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés

- `db/query/markingReviews.sql`
- `internal/db/markingReviews.sql.go` — généré
- `internal/db/markingAnswerReviewService.go`
- `internal/db/markingAnswerReviewService_test.go`
- `internal/markingscoring/question.go`
- `internal/markingscoring/question_test.go`
- `internal/handlers/tools/countingPoints.go`
- ce rapport
