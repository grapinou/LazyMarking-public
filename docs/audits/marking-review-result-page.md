# Page résultat Marking — premier jalon de revue

## Périmètre livré

La route `GET /dashboard/marking/success?job_id=...` rend désormais une page
résultat typée, ownership-aware et en lecture seule. Aucun écran ou handler de
revue, aucun POST, aucune régénération, aucun changement de scoring ou OpenCV et
aucune migration ne sont ajoutés.

## View models

Les types dédiés créés sont :

- `MarkingResultPageData` ;
- `MarkingReviewStatusView` ;
- `MarkingArtifactLinksView` ;
- `MarkingNonCorrectedSummaryView` ;
- `NoticeView`.

Ils ne contiennent aucune row sqlc et aucune classe Bootstrap. `ExtraData` est
retiré de la page résultat : **oui**. Il reste utilisé par les autres pages
Marking, qui sont hors périmètre de ce jalon.

## Ownership et lifecycles

Le handler lit d'abord le job avec `(job_id, user_id de session)`. Un job absent
ou appartenant à un autre utilisateur retourne le même 404. Les requêtes de
résumé, de révisions et de copies non corrigées répètent cette frontière
d'ownership.

Le lifecycle technique existant `running / success / failed` reste inchangé.
La page résultat refuse `running` et `failed` selon sa convention historique de
404. Un job techniquement réussi dont la génération PDF n'est pas terminée est
redirigé vers la progression.

Pour un job réussi, `GetMarkingReviewSummary` et
`DeriveMarkingReviewStatus` alimentent exclusivement le lifecycle de revue :

- `no_review_needed` : « Aucune réponse à vérifier » ; PDF finaux seulement si
  les révisions sont current ;
- `pending` : compteur séparé des réponses ambiguës et action « Vérifier les
  réponses » vers la future route GET de revue ;
- `completed`, current : « Toutes les réponses ont été vérifiées » et PDF
  finaux disponibles ;
- `completed`, stale : réponses enregistrées, actualisation nécessaire, aucun
  lien final vers `corrected.pdf` ou `mark-table.pdf` ;
- `legacy_unavailable` : message dédié, sans faux compteur à zéro, et maintien
  des PDF historiques.

Aucun nouveau statut n'est persisté.

## Artefacts et copies non corrigées

Pour les jobs modernes, les liens finaux ne sont construits que lorsque
`artifacts_revision == review_revision`. Les anciens fichiers stale ne sont pas
supprimés. Les URLs utilisent uniquement l'opération et les noms d'artefacts
issus du job possédé.

Le PDF `corrected_NOT.pdf`, lorsqu'il existe comme fichier régulier attendu dans
le workspace possédé, est présenté séparément sous « Copies / pages non
corrigées ». Sa disponibilité ne dépend pas des révisions de review.

Une requête typée compte séparément les outcomes `incomplete`, `error` et
`not_seen`. Ces compteurs ne sont jamais additionnés aux réponses ambiguës.
Limite conservée : le contrat historique ne stocke pas le nom de
`corrected_NOT.pdf` en DB ; il est donc dérivé du nom de `corrected.pdf`, puis
vérifié en lecture seule dans le workspace attendu.

## UX, responsive et accessibilité

La page utilise le Bootstrap existant avec `container`, cards et boutons
explicites, ainsi qu'une disposition de boutons adaptée au mobile. Elle possède
un `h1`, des sections titrées par des `h2`, des textes explicites et aucune
information portée uniquement par la couleur. Aucun ID brut ou statut technique
anglais n'est présenté.

Les ouvertures automatiques de PDF et tout le JavaScript associé ont été
supprimés. Le `console.log` a été supprimé. L'utilisateur ouvre chaque document
explicitement.

## Mutation filesystem du GET

Mutation filesystem dans le GET actuel : **non**. Avant ce jalon, le handler
appelait `CreateOperationTempDir` puis `LeftPages`, qui pouvait créer le PDF des
pages restantes, convertir des PNG et supprimer des fichiers pendant un GET.
Cet appel a été remplacé par une vérification `Lstat` en lecture seule du PDF
historique attendu. Aucune génération, suppression ou régénération n'est
effectuée.

## Tests

Les tests couvrent :

- job réussi avec `no_review_needed` ;
- `pending` et URL de la future revue ;
- `completed` avec artefacts current ;
- `completed` avec artefacts stale ;
- legacy ;
- job cross-user et job absent, indistinguables ;
- absence explicite des liens finaux stale et présence des liens current ;
- indépendance de `corrected_NOT.pdf` et résumé typé des trois outcomes ;
- jobs `running` et `failed` refusés par la page résultat.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `db/query/markingReviews.sql` ;
- `internal/db/markingReviews.sql.go` ;
- `internal/handlers/marking/handlers.go` ;
- `internal/handlers/marking/handlers_test.go` ;
- `internal/handlers/marking/resultViewData_test.go` ;
- `internal/handlers/marking/views.go` ;
- `internal/handlers/tools/markingArtifact.go` ;
- `internal/templates/data/marking.go` ;
- `internal/templates/marking/success_marking_processing.html` ;
- `docs/audits/marking-review-result-page.md`.
