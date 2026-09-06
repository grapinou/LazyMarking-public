# Review Marking interactive — POST et PRG

## Route et formulaire

La route ajoutée est `POST /dashboard/marking/review/apply`, enregistrée sous le
middleware d'authentification existant. Le formulaire server-rendered transmet :

- `job_id` ;
- `answer_detection_id` ;
- `reviewed_state` ;
- `expected_review_revision` ;
- `expected_answer_review_revision`.

`reviewed_state` est strictement limité à `0` (« Non cochée ») et `1`
(« Cochée »). Les deux radios possèdent name, value, id et label, et aucune
valeur n'est présélectionnée : le professeur confirme explicitement ce qu'il
voit. L'attribut HTML `required` empêche un submit normal sans choix, tandis que
le handler reste autoritatif et refuse également toute absence ou valeur
invalide.

## Optimistic locking

Le GET transporte `JobRevision` et une `AnswerReviewRevision` nullable dans le
view model. Elles sont uniquement rendues dans des champs hidden et ne sont
jamais affichées comme contenu utilisateur.

Le service actuel représente l'absence de review par
`ExpectedAnswerReviewRevision == 0`, alors qu'une révision persistée commence à
`1`. Pour conserver une représentation de formulaire non ambiguë, l'absence est
envoyée comme chaîne vide ; une révision présente doit être strictement
positive. Le POST convertit explicitement la chaîne vide vers la sentinelle du
service. Une chaîne `"0"` est refusée.

Le POST ne recharge aucune révision. Il transmet exactement les versions vues
par le professeur à `ApplyMarkingAnswerReview`.

## Application métier et ownership

Le handler réalise uniquement auth, parsing, validation, appel du service,
gestion d'erreur et redirection. `ApplyMarkingAnswerReview` reste l'unique
primitive pour l'insertion/update de review, l'état effectif, le recalcul de la
question et de la copie, `review_revision`, `artifacts_revision` et
l'optimistic locking.

Le handler ne contient aucun scoring et aucun UPDATE direct. La frontière
d'ownership autoritative du service refuse de façon indistinguable les jobs ou
detections inconnus, cross-user, cross-job, running, failed ou non-corrected.
Le POST ne filtre pas l'ambiguïté et accepte donc une détection owned/corrected
non ambiguë ; seule la file GET automatique demeure ambiguity-only.

## Confirmation, override et PRG

Confirmation et override utilisent le même formulaire et le même endpoint. Le
service décide seul si l'état effectif change et quelles conséquences appliquer
aux scores et aux révisions.

Après succès, le handler répond `303 See Other` vers
`/dashboard/marking/review?job_id=...`. Le GET recharge le premier candidat
pending suivant. Un refresh du GET ne rejoue aucune mutation.

Après la dernière candidate, le même PRG est conservé : le GET `/review`
constate une file vide et redirige vers `/dashboard/marking/success?job_id=...`.
Aucune régénération n'est déclenchée. Les règles existantes peuvent donc laisser
les artefacts stale lorsque l'état effectif a changé.

## Conflits et erreurs

`ErrMarkingReviewConflict` produit un `303` vers la page review avec
`notice=conflict`. Le GET construit alors une notice locale typée :
« Cette correction a été modifiée dans un autre onglet. La page a été
actualisée. » La décision stale n'est jamais réappliquée et le service rollbacke
toute écriture tentée.

`ErrMarkingReviewUnavailable` produit un 404 sans révéler l'existence d'une
ressource étrangère. Les erreurs de formulaire produisent un 400 en français et
les erreurs internes un 500 générique sans détail DB.

Ce mécanisme de notice reste volontairement local. Aucun système flash
transverse ni mécanisme CSRF propre à Marking n'est ajouté. La dette CSRF reste
globale à l'application, conformément à l'audit UX.

## Hors périmètre

- scoring présent dans le handler : **non** ;
- PDF régénérés : **non** ;
- filesystem modifié : **non** ;
- OpenCV, ROI ou crop modifiés : **non** ;
- migration ajoutée : **non**.

## Tests

Les tests couvrent :

- aucun radio présélectionné, labels, bouton submit et champs hidden ;
- révisions job et answer nullable transportées ;
- validation de `reviewed_state` absent, non numérique, `-1` et `2` ;
- IDs et révisions invalides ;
- confirmation auto unchecked / humain unchecked avec score conforme au
  service, review créée et PRG ;
- override unchecked vers checked avec recalcul du service ;
- retry idempotent avec les versions valides ;
- passage au candidat suivant et refresh GET non mutatif ;
- dernière candidate, redirection résultat et artefacts stale ;
- conflit global concurrent, rollback, redirection et notice ;
- cross-user, cross-job, inconnu, running et failed ;
- review directe d'une détection non ambiguë.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers créés ou modifiés

- `internal/handlers/marking/review.go` ;
- `internal/handlers/marking/reviewApply.go` ;
- `internal/handlers/marking/review_test.go` ;
- `internal/handlers/marking/routes.go` ;
- `internal/templates/data/marking.go` ;
- `internal/templates/marking/review.html` ;
- `docs/audits/marking-review-post-prg.md`.
