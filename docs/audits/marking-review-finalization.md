# Finalisation de la revue Marking

## Moment de régénération

Le POST de revue appelle et termine `ApplyMarkingAnswerReview` avant toute
tentative PDF. Il relit ensuite la file via
`ListPendingMarkingReviewCandidates`, le read model déjà utilisé par le GET.

- s'il reste un candidat, il répond `303` vers `/dashboard/marking/review` et
  ne régénère rien ;
- après la dernière candidate, il appelle une fois
  `RegenerateMarkingArtifacts`, puis répond `303` directement vers la page
  résultat.

Le service existant reste l'unique implémentation de la génération. Son cas
`artifacts_revision == review_revision` est un no-op idempotent.

## Erreur et conservation de la DB

La review, l'état effectif, les résultats question/copie et
`review_revision` sont commités avant l'appel PDF. Une erreur de génération ne
peut donc pas les rollbacker. Le handler ne modifie pas le statut du job et ne
met pas à jour `artifacts_revision` directement : le job reste `success` et les
artefacts restent stale.

Une erreur ou un conflit de génération redirige vers la page résultat avec
`notice=artifacts_failed`. La notice française indique : « Les réponses sont
enregistrées, mais les PDF n'ont pas pu être actualisés. » Aucun détail interne
n'est exposé.

## Retry et ownership

La route authentifiée `POST /dashboard/marking/artifacts/regenerate` reçoit
uniquement `job_id`, appelle `RegenerateMarkingArtifacts`, puis applique PRG
vers `/dashboard/marking/success?job_id=...`.

Le service recharge lui-même la cible avec le couple utilisateur/job et exige
un job `success`. Il protège ainsi les jobs absents, cross-user, running,
failed et legacy. `ErrMarkingArtifactsUnavailable` est rendu comme un 404 sans
fuite d'existence. Les conflits et erreurs de publication conservent l'état
stale et redirigent avec la même notice de retry.

Le handler ne contourne ni la capture des révisions ni l'UPDATE conditionnel
du service. Une review concurrente empêche donc la déclaration incorrecte des
artefacts comme current et laisse le retry disponible.

## Page résultat et artefacts

Le view model typé expose une action de régénération uniquement pour un job
moderne `completed` dont les artefacts sont stale. Dans cet état :

- les liens finaux `corrected.pdf` et `mark-table.pdf` restent masqués ;
- un formulaire POST propose « Actualiser les PDF » ou, après erreur,
  « Réessayer l'actualisation des PDF » ;
- `corrected_NOT.pdf` reste accessible séparément et n'est pas régénéré.

Après succès ou no-op current, la page recalcule toujours la fraîcheur depuis
`artifacts_revision == review_revision`, affiche les deux PDF finaux et masque
le bouton retry. Un job legacy garde son contrat historique et ne reçoit pas
un bouton impossible.

Les anciens fichiers stale ne sont pas supprimés. Les handlers n'ajoutent
aucune stratégie filesystem : seul le protocole temp/backup/rename/restore de
`RegenerateMarkingArtifacts` est utilisé.

## Tests

Les tests ajoutés ou adaptés couvrent :

- candidat non final sans appel de régénération ;
- dernière candidate avec override, appel unique, artefacts current et PRG
  résultat ;
- dernière candidate avec confirmation et finalisation idempotente ;
- échec après la dernière décision, review persistée, job `success` et
  révisions stale ;
- retry réussi et retry current no-op ;
- nouvel échec/conflit avec notice retry ;
- formulaire retry invalide ;
- indisponibilité cross-user, running, failed, legacy et inconnue rendue 404 ;
- bouton retry réservé à `completed + stale` ;
- notice d'échec, liens finaux stale masqués et `corrected_NOT.pdf`
  indépendant.

## Périmètre et migrations

- scoring modifié : **non** ;
- OpenCV, ROI ou crop modifiés : **non** ;
- migration ajoutée : **non** ;
- génération PDF dupliquée dans un handler : **non**.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès.

## Fichiers créés ou modifiés

- `internal/handlers/marking/artifactsRegenerate.go` (créé) ;
- `internal/handlers/marking/artifactsRegenerate_test.go` (créé) ;
- `internal/handlers/marking/reviewApply.go` ;
- `internal/handlers/marking/review_test.go` ;
- `internal/handlers/marking/handlers.go` ;
- `internal/handlers/marking/resultViewData_test.go` ;
- `internal/handlers/marking/routes.go` ;
- `internal/templates/data/marking.go` ;
- `internal/templates/marking/success_marking_processing.html` ;
- `docs/audits/marking-review-finalization.md` (créé, documentation locale).
