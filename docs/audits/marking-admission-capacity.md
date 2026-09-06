# Admission et capacité Marking

Date : 2026-09-01

## Limites avant ce jalon

`ProcessingMarkingHandler` appliquait `http.MaxBytesReader` et
`ParseMultipartForm` avec une limite de 100 Mio. Après contrôle de la génération
possédée et `success`, il vérifiait les cinq octets `%PDF-`, copiait l'upload
dans un fichier temporaire système, créait immédiatement un `marking_job`
`running`, puis lançait `ProcessMarking` en goroutine.

`ProcessMarking` créait ensuite le workspace du job, chargeait le snapshot des
copies attendues et lançait `pdfseparate`. Le nombre de fichiers séparés était
écrit dans le job, sans limite. Les pages étaient ensuite rasterisées et
traitées par QR/OpenCV avec cinq workers par job, puis les copies par cinq
workers par job. Plusieurs jobs pouvaient multiplier ces deux pools sans borne
globale.

Les commandes `pdfseparate`, `pdftoppm`, Typst et `pdfunite` possèdent un timeout
individuel de deux minutes. Le pipeline complet n'avait et n'a pas de deadline
globale.

## Limites retenues

- upload HTTP brut : `MaxMarkingUploadBytes = 100 << 20`, soit 100 Mio, valeur
  existante conservée et désormais nommée ;
- pages : `MaxMarkingPDFPages = 500` ;
- pipelines Marking lourds simultanés : `MaxConcurrentMarkingJobs = 2`, global
  au processus ;
- dimension physique déclarée d'une page :
  `MaxMarkingPDFPageDimensionPoints = 2000` points par côté (environ 70,6 cm).

La limite de 500 pages permet encore, par exemple, cent copies de cinq pages,
donc une charge très supérieure à une classe ordinaire, tout en bornant la
multiplication des rasterisations et traitements OpenCV. Deux jobs globaux
représentent au plus deux pools de cinq workers pages/copies : ce compromis est
prudent pour une V1 single-instance sans introduire de queue.

Une borne per-user n'a pas été ajoutée. La borne globale protège déjà la
ressource critique pour le volume d'utilisateurs visé ; ajouter une comptabilité
par utilisateur complexifierait le contrat sans augmenter la capacité sûre.

## Comptage et moment du rejet

Après staging de l'upload, le handler tente immédiatement d'acquérir un slot,
puis appelle `pdfinfo -f 1 -l 501 -box`. La commande est limitée à 30 secondes.
Son résultat fournit le nombre total de pages et les dimensions des pages
jusqu'à la borne utile. La sortie stderr et les chemins ne sont jamais transmis
à l'utilisateur.

Les contrôles suivants ont donc lieu avant `CreateMarkingJob`, `pdfseparate`,
`pdftoppm`, QR, SIFT, homographie et tout autre traitement OpenCV :

- PDF structurellement lisible par Poppler ;
- nombre de pages positif et inférieur ou égal à 500 ;
- MediaBox positive et inférieure ou égale à 2000 points sur chaque côté pour
  toutes les pages admises.

Un rejet de page count/dimension ne crée aucun job DB et supprime le fichier de
staging. Il ne peut pas laisser de job fantôme `running`.

La dimension physique ne constitue pas un détecteur complet de bombe PDF : une
page A4 peut embarquer une image interne très compressée et très grande. La
borne de taille brute, le timeout Poppler, la limite de pages et la concurrence
globale réduisent le risque, sans tenter de réimplémenter un parseur PDF.

## Capacité simultanée et libération

Le sémaphore est process-local, non persistant et non bloquant. Quand ses deux
slots sont occupés, le handler répond immédiatement HTTP 503 avec :

> Le serveur traite déjà plusieurs corrections. Veuillez réessayer dans
> quelques instants.

Il n'existe ni attente HTTP indéfinie ni queue cachée. Le slot couvre
l'inspection `pdfinfo` et toute la goroutine de traitement. Avant transfert à la
goroutine, un `defer` le libère sur tout retour anticipé (PDF rejeté, erreur DB,
etc.). Après transfert, la goroutine le libère par `defer` après succès, échec,
annulation ou panic récupérée par `ProcessMarking`. La closure de libération est
elle-même idempotente pour éviter qu'un double cleanup libère le slot d'un autre
job.

Au redémarrage, le sémaphore repart vide. C'est volontaire : les anciens jobs
`running` sont d'abord convertis en `failed` par le recovery startup, et aucune
capacité persistante n'est nécessaire.

## Durée globale

Aucun timeout global n'a été ajouté. Une vraie correction proche de 500 pages
peut légitimement dépasser une durée arbitraire, tandis que certaines opérations
GoCV ne sont pas préemptables finement par contexte. Les subprocess restent
bornés individuellement, l'arrêt applicatif annule le contexte orchestration,
et la limite de deux jobs borne la multiplication de charge. Une deadline
globale reste un paramètre d'exploitation à calibrer sur corpus réel plutôt
qu'une valeur artificielle introduite ici.

## Disque et cleanup

L'upload staged est supprimé sur tous les rejets et à la fin de la goroutine.
Le workspace d'un job échoué est supprimé par `ProcessMarking`; le recovery
startup traite les interruptions ; les jobs failed résiduels sont purgés après
sept jours. Les workspaces `success` conservent volontairement les pages
alignées et artefacts historiques. Il n'existe toujours pas de quota disque
global : la rétention historique doit être dimensionnée/surveillée en
exploitation, mais les chemins d'échec normaux ne croissent pas sans cleanup.

## Erreurs utilisateur

Les catégories sont séparées avec des textes français sans stderr, SQL, stack
trace ni chemin :

- upload supérieur à 100 Mio : HTTP 413, fichier trop volumineux ;
- multipart ou PDF invalide : HTTP 400 ;
- plus de 500 pages : HTTP 422, trop de pages pour une seule correction ;
- page supérieure à la dimension admise : HTTP 422 ;
- deux jobs déjà admis : HTTP 503, serveur temporairement occupé.

## Tests

- PDF synthétique à 499 pages : admis ;
- PDF synthétique à exactement 500 pages : admis ;
- PDF synthétique à 501 pages : rejeté avant création du job ;
- page synthétique dépassant 2000 points : rejetée ;
- inspection sans propagation d'un path/diagnostic technique ;
- `MaxBytesReader` renvoie bien `http.MaxBytesError` au dépassement ;
- deux slots admis, troisième refusé immédiatement ;
- libération idempotente après échec et nouvelle admission possible ;
- handler : capacité pleine -> 503 et aucun job ;
- handler : dépassement pages -> 422 et aucun job ;
- chemin normal : job créé et pipeline asynchrone lancé.

L'absence de job DB sur les rejets de pages/capacité garantit aussi qu'aucune
étape lourde ou OpenCV ne peut être lancée dans ces cas : le lancement de la
goroutine se trouve strictement après `CreateMarkingJob`.

## Changements exclus

- migration : non ;
- SQL/sqlc : non ;
- QR, SIFT, homographie, MeanGray, scoring et review : non ;
- queue persistante ou architecture distribuée : non.

## Validations

- `./scripts/check.sh` : succès, replay Goose 0001–0040 inclus ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès dans `check.sh`, puis relancé séparément en fin de
  validation.

## Fichiers modifiés ou créés

- `internal/handlers/marking/admission.go` ;
- `internal/handlers/marking/admission_test.go` ;
- `internal/handlers/marking/handlers.go` ;
- `internal/handlers/marking/handlers_test.go` ;
- `docs/audits/marking-admission-capacity.md` (rapport local non commité).
