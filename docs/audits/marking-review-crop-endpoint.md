# Endpoint de crop pour la revue Marking

## Primitive de mapping

`ResolveMarkingReviewCandidateROI` est la primitive applicative unique. Elle
reçoit le contexte, les queries, l'utilisateur de session, son username, le job
et la détection. Elle retourne un DTO `MarkingReviewCandidateROI`, jamais une
row sqlc, avec le chemin résolu de la page alignée, `page_exam`, le centre, le
rayon et les dimensions de l'image.

La source du mapping est exclusivement historique :

- `marking_answer_detections`, `marking_question_results` et
  `marking_copy_results` persistés ;
- `student_exam_content` ;
- `student_exam_page_content` ;
- `marking_aligned_pages`.

Aucune question, réponse ou composition QCM vivante n'est jointe.

## Règle d'offset et ROI

La fonction pure `resolveHistoricalAnswerROI` centralise la règle. Elle convertit
`question_index / answer_index` en offset global à partir du QCM snapshot, puis
parcourt les réponses de `student_exam_page_content` dans l'ordre strict des
pages. C'est le même ordre que le runtime, qui concatène les détections page par
page. Elle refuse les indexes invalides, les pages non consécutives et une
couverture de réponses incohérente.

`MarkingAnswerMeasurementRect` centralise la ROI de mesure exacte : centre
`± radius/2`, clampée aux dimensions. `GetAnswerDetections` utilise désormais
ce helper sans changement de formule, de classification ou d'OpenCV.

## Page alignée durable

La page finale est résolue par `ResolveMarkingAlignedPage`. Les contrôles
existants restent l'unique frontière : ownership, job/copie/page, storage key,
confinement de chemin, refus des symlinks, fichier régulier, SHA-256, dimensions
et décodage PNG. Il n'existe aucun fallback vers un scan brut, un PDF corrigé,
une image reconstruite ou une autre source.

## Crop d'inspection

`BuildMarkingReviewCrop` décode la page alignée et encode le crop PNG entièrement
en mémoire. La marge centralisée utilise un demi-côté
`max(2 * radius, 30 px)`. Le rectangle est intersecté avec les bornes de l'image,
ce qui couvre les bords gauche, droit, haut et bas sans rectangle invalide.

Le crop reste local autour de la case et ne contient pas volontairement une
grande portion de feuille. Aucun chemin filesystem n'est envoyé au client. Le
fichier durable n'est ni annoté ni modifié et aucun crop persistant n'est créé.

## Endpoint et sécurité

La route ajoutée est :

`GET /dashboard/marking/review/crop?job_id=...&answer_detection_id=...`

Elle utilise l'authentification de session et exige un job possédé en état
`success`, une détection appartenant exactement à ce job et une
`marking_copy_result` avec outcome `corrected`. Cross-user, inconnu, cross-job,
job non réussi, copie non corrigée ou mapping inutilisable sont présentés de
façon indistinguable par un 404.

Le resolver ne filtre volontairement pas l'ambiguïté : toute
`answer_detection` possédée et corrigée est supportée.

Une réponse réussie possède :

- `Content-Type: image/png` ;
- `Content-Disposition: inline; filename=review-crop.png` ;
- `X-Content-Type-Options: nosniff` ;
- `Cache-Control: no-store`.

Les erreurs utilisateur ne contiennent aucun détail technique.

## Hors périmètre confirmé

- fallback utilisé : **non** ;
- filesystem persistant ajouté : **non** ;
- score modifié : **non** ;
- PDF modifié : **non** ;
- migration ajoutée : **non** ;
- `ambiguity_delta` ou `detected_state` modifié : **non**.

## Tests

Les tests couvrent :

- première et dernière réponse de la première question ;
- première réponse de la question suivante et frontières entre pages ;
- plusieurs pages et plusieurs questions ;
- indexes de question et de réponse invalides ;
- fidélité du centre, rayon et rectangle de mesure, avec comparaison du
  `MeanGray` d'un PNG synthétique ;
- crop normal et clamp aux quatre bords ;
- dimensions, PNG décodable et source durable inchangée ;
- owner et détection non ambiguë : 200 `image/png` ;
- cross-user, cross-job, running, failed et non-corrected : refus ;
- page absente, hash invalide et PNG corrompu : erreur propre ;
- headers `no-store` et `nosniff`.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `db/query/markingReviews.sql` ;
- `internal/db/markingReviews.sql.go` ;
- `internal/handlers/marking/routes.go` ;
- `internal/handlers/marking/reviewCrop.go` ;
- `internal/handlers/marking/reviewCrop_test.go` ;
- `internal/handlers/tools/getAnswersState.go` ;
- `internal/handlers/tools/markingReviewCrop.go` ;
- `internal/handlers/tools/markingReviewCrop_test.go` ;
- `internal/templates/data/marking.go` ;
- `docs/audits/marking-review-crop-endpoint.md`.
