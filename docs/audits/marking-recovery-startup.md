# Audit du recovery Marking au démarrage

Date : 2026-09-01

## Comportement avant modification

`RecoverRunningMarkingJobs` chargeait les jobs `running` avec
`ListRunningMarkingJobs`, puis supprimait leur workspace et appelait
`FailMarkingJob`. L'absence d'un workspace était déjà un succès et le second
appel au recovery était déjà sans effet. Les jobs `success` et `failed`
n'étaient pas listés.

La fonction retournait toutefois immédiatement à la première erreur de cleanup,
d'UPDATE ou de nombre de lignes affectées. Dans `cmd/server/main.go`, toute cette
catégorie d'erreur arrivait au même `log.Fatal` que l'échec du listing. Un seul
ancien workspace invalide ou un seul UPDATE défaillant pouvait donc empêcher le
traitement des jobs suivants et le démarrage du serveur.

## Contrat après modification

`RecoverRunningMarkingJobs` renvoie maintenant un `MarkingRecoveryResult` :
nombre de jobs trouvés, jobs récupérés, échecs de cleanup et échecs de
transition. Deux catégories d'erreurs sont distinguées :

- l'échec de `ListRunningMarkingJobs` empêche de commencer et reste une erreur
  globale ; le startup conserve volontairement son `log.Fatal` dans ce cas ;
- une erreur propre à un job est comptée et loguée côté serveur, puis le
  recovery poursuit avec le job suivant et retourne normalement au startup.

Le startup journalise un résumé sans donnée élève. Les IDs de job ne sont
utilisés que dans les logs techniques individuels, conformément aux conventions
serveur existantes.

## Cleanup filesystem et transition failed

Le cleanup continue d'utiliser exclusivement `RemoveOperationTempDir`. Son
contrat de confinement, de validation des composants, de refus des symlinks et
d'absence considérée comme un succès n'a pas été modifié.

Si le cleanup échoue, le recovery tente malgré tout `FailMarkingJob`. Un job
interrompu ne reste donc pas artificiellement `running` uniquement à cause d'un
résidu filesystem. `FailMarkingJob` demeure la primitive existante : l'UPDATE
est conditionné par `id`, `user_id` et `status = 'running'`. Une erreur ou zéro
ligne affectée est loguée et n'empêche pas les jobs suivants.

## Lifecycle et idempotence

Seuls les jobs listés comme `running` sont traités. Les jobs `success` et
`failed`, leurs workspaces et leurs artefacts ne sont pas touchés. Aucun statut
n'a été ajouté. Après un premier passage réussi, le job est `failed` et n'est
plus listé au second passage : le recovery est idempotent.

Un workspace absent ou déjà supprimé reste accepté. Un workspace symlinké est
refusé par le confinement, l'erreur est observable, la transition `failed` est
quand même tentée et les jobs suivants continuent. Une disparition ou une
transition concurrente produit zéro ligne affectée, traité comme un échec local
observable et non comme une panne globale.

## Startup et log.Fatal restant

Une erreur isolée n'est plus retournée au startup et ne peut donc plus déclencher
son `log.Fatal`. Le `log.Fatal("Failed to recover interrupted marking jobs")`
reste uniquement pour l'erreur globale de listing. Ce choix conserve un arrêt
sûr lorsque la DB est indisponible ou illisible au point qu'aucun recovery ne
peut commencer.

## Résidus de régénération review

Les `.review-artifacts-*` et `.review-backup` des jobs `success` ne sont pas
traités dans ce jalon. Les staging dirs sont normalement nettoyés par `defer`,
et le protocole de publication utilise les backups pour restaurer la paire de
PDF en cas d'échec. Après un crash brutal, supprimer automatiquement ces
backups sans connaître l'étape exacte de publication pourrait détruire la seule
copie restaurable. Cette question reste une dette séparée mineure (P3), à
traiter avec un protocole explicite de reconnaissance/restauration, sans la
mélanger au recovery des jobs `running`.

## Tests

Les tests couvrent :

- cleanup et transition d'un job interrompu, conservation de ses résultats
  partiels et idempotence sur deux appels ;
- absence de workspace ;
- trois jobs `running`, dont un cleanup échoue sur un symlink : les trois
  transitions sont tentées et les jobs suivants sont traités ;
- échec DB injecté sur la transition du job central : les autres jobs passent
  `failed` ;
- échec complet du listing : erreur globale retournée ;
- absence de modification des jobs et workspaces `success` et `failed`.

## Changements exclus

- migration : non ;
- SQL/sqlc : non ;
- scoring, OpenCV, handlers HTTP et UX : non ;
- purge/retention générale : non.

## Validations

- `./scripts/check.sh` : succès (inclut replay Goose jusqu'à 0040, formatage,
  vet, tests, build et `git diff --check`) ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès dans `check.sh`, puis relancé séparément en fin de
  validation.

## Fichiers modifiés ou créés

- `cmd/server/main.go` ;
- `internal/handlers/tools/recoverMarkingJobs.go` ;
- `internal/handlers/tools/recoverMarkingJobs_test.go` ;
- `docs/audits/marking-recovery-startup.md` (rapport local non commité).
