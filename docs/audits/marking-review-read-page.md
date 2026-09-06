# Page séquentielle Marking en lecture seule

## Route et ownership

La route ajoutée est `GET /dashboard/marking/review?job_id=...`. Elle est
enregistrée avec le routeur Marking et protégée par le middleware
d'authentification existant.

Le premier accès DB est `GetMarkingReviewSummary(marking_job_id, user_id)`. La
requête exige l'ownership et `status = 'success'`. Un job absent, cross-user,
`running` ou `failed` produit donc le même 404. Toutes les requêtes suivantes
répètent les paramètres job/utilisateur.

## Source et ordre des candidats

La file utilise directement `ListPendingMarkingReviewCandidates`. Le handler ne
recalcule ni ambiguïté, ni `MeanGray`, ni seuil, ni delta. La politique reste
celle du snapshot du job et la query exclut les copies autres que `corrected`.
Une détection non ambiguë n'est pas ajoutée artificiellement à cette file.

L'ordre SQL existant est explicite et stable : `student_exam_id`, puis
`question_index`, puis `answer_index`. La page choisit systématiquement la
première row pending. Deux GET sans review intermédiaire rendent donc le même
candidat.

## Progression

La formule affichée est `position = reviewed_candidates + 1`, sur
`total_candidates`. `pending_candidates` est également affiché comme nombre de
réponses restantes. Avec 5 candidates dont 2 déjà revues, la page affiche
« Réponse 3 sur 5 » et « 3 réponses restantes ».

## View models

Les modèles dédiés sont `MarkingReviewPageData` et
`MarkingReviewCandidateView`. Ils ne contiennent aucune row sqlc ni
`map[string]any`. Ils transportent la progression, la révision du job, le
candidat courant, l'URL du crop et l'URL de retour. Les IDs nécessaires à une
future décision et aux URLs ne sont jamais affichés comme informations métier.

## Données présentées

L'identité est décodée exclusivement depuis le `student_exam_content` historique
retourné par `GetMarkingAnswerReviewTarget`. La page affiche « prénom nom » sans
relire la table Students vivante.

`question_index` devient « Question N » en ajoutant 1. `answer_index` est
converti par un helper pur en libellé alphabétique : A à Z, puis AA, AB, etc.

Le libellé « Détection automatique » utilise uniquement `detected_state` et
affiche « cochée » ou « non cochée ». `effective_state`, `MeanGray`, threshold
et delta ne sont pas présentés.

L'image utilise uniquement
`/dashboard/marking/review/crop?job_id=...&answer_detection_id=...`. Aucun chemin
filesystem n'entre dans le view model ou le HTML.

## Files vides et legacy

`DeriveMarkingReviewStatus` est utilisé après le résumé. Pour
`no_review_needed`, `completed` et `legacy_unavailable`, la route redirige vers
`/dashboard/marking/success?job_id=...`. Une incohérence défensive où le résumé
annonce pending mais la liste est vide suit la même redirection. Aucun écran
« Réponse 0 sur 0 » n'est rendu et aucune régénération n'est déclenchée.

## HTML, responsive et accessibilité

Le template dédié est `internal/templates/marking/review.html`. Il utilise un
`container`, une grille Bootstrap centrée, une card et une image `img-fluid`
large à `width: 100%` avec une largeur maximale raisonnable. Il reste lisible
sur mobile, tablette et desktop.

La page possède un `h1`, une progression textuelle, un titre de candidat, un alt
neutre « Extrait de la case à vérifier », des libellés explicites et aucune
information portée seulement par la couleur. Elle ne propose aucun faux bouton
de validation : un message annonce que cette action viendra au jalon suivant.
Le lien « Retour aux résultats » est présent.

Aucun JavaScript n'est ajouté ; la page est entièrement server-rendered.

## Lecture seule

Le GET est strictement read-only : aucune écriture DB, aucun fichier créé,
aucun nettoyage, aucune review, aucun score et aucune régénération PDF.

## Tests

Les tests couvrent : owner + pending et premier candidat, stabilité sur deux
GET, progression, identité historique, question 1-based, labels au-delà de Z,
états cochée/non cochée, URL du crop, lien retour, absence de diagnostics,
cross-user, absent, running, failed, redirections no-review/completed/legacy et
exclusion des copies non corrected ainsi que des détections non ambiguës.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `internal/handlers/marking/handlers.go` ;
- `internal/handlers/marking/routes.go` ;
- `internal/handlers/marking/review.go` ;
- `internal/handlers/marking/review_test.go` ;
- `internal/handlers/marking/views.go` ;
- `internal/templates/data/marking.go` ;
- `internal/templates/marking/review.html` ;
- `docs/audits/marking-review-read-page.md`.
