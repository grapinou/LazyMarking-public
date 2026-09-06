# Audit final de clôture — Marking

Date de vérification : 1er septembre 2026.

Cet audit est fondé sur le dépôt actuel, pas sur les conclusions des audits
précédents. Il a relu les migrations 0027 à 0040, les queries SQLC Marking, le
pipeline de production, les services de review et d'artefacts, les handlers,
les templates et les tests. Aucun code ni schéma n'a été modifié.

## 1. Synthèse et verdict

Les quatre P1 historiques du premier audit sont fermés : le job est lié à une
génération possédée et success, les références visuelles modernes sont
historiques, les résultats détaillés sont persistés, et les ambiguïtés disposent
d'une review humaine transactionnelle suivie d'une régénération contrôlée.

Le cœur métier n'a plus de P1. Il reste toutefois cinq sujets P2 de clôture :
l'admission de charge, le recovery fatal au démarrage, le replay Goose cassé,
le contrat moderne de `corrected_NOT.pdf`, et l'absence de CSRF explicite à
trancher selon le déploiement. Ces sujets ne fragilisent pas le calcul ou
l'historique d'un job moderne déjà réussi, mais empêchent de déclarer toute la
refonte terminée sans réserve.

**Verdict Marking : B.**

- P1 restants : **0** ;
- P2 restants : **5** ;
- P3 restants : **3** ;
- chantier structurellement terminé : **non au sens de la clôture produit** ;
- architecture métier/historique structurellement solide : **oui** ;
- passage à une courte phase de dettes de clôture : **oui** ;
- test réel complet ensuite : **oui, recommandé**.

## 2. Architecture vérifiée de bout en bout

| Étape | Source d'autorité et persistance | Ownership / intégrité | Tests et legacy |
|---|---|---|---|
| Génération Exam | `exams_generated`, `student_exam`, `student_exam_content` et `student_exam_page_content` ; le QCM individualisé et sa géométrie sont snapshotés. | Les relations génération, élève et user sont protégées par requêtes et triggers. Une génération ne devient success qu'avec une couverture complète de références modernes. | Tests de génération, références et recovery Exam. Les anciennes pages ont des métadonnées de référence toutes NULL. |
| Références visuelles | PNG pré-QR natif 300 PPI conservé sous `assets/tmp/<user>/exam-<generation>/references/...`, avec storage key, dimensions, DPI et SHA-256 dans `student_exam_page_content`. | Resolver ownership-aware ; clé exacte, confinement, parents non symlinkés, fichier régulier, identité du fichier ouvert, hash, décodage PNG et dimensions. | Tests octets, corruption, traversal, symlink, mauvais owner et couverture. Le fallback Typst n'est autorisé que si les cinq métadonnées legacy sont toutes NULL ; une référence moderne corrompue ne fallback jamais. |
| Admission scan | POST multipart sélectionnant explicitement `exam_generated_id` ; corps limité à 100 Mio et fichier stagé sous un nom OS. | `GetExamStatus` puis `CreateMarkingJob` exigent owner + success ; la requête d'insertion répète le contrôle. | Tests handler génération absente, étrangère et non-success. Pas de limite de pages ni de jobs concurrents. |
| Job | `marking_jobs` persiste génération, lifecycle, compteurs, versions d'algorithme, seuil, delta et révisions. | Trigger de génération possédée ; identité génération/version/seuil immutable. | Tests de création, transitions conditionnelles et métadonnées. Les jobs antérieurs à 0036/0038 restent nullable/legacy. |
| QR / regroupement | QR fournit `student_exam_id` et `page_exam`; le nom du fichier provient du split serveur. Regroupement et tri déterministes. | `ValidateQrCodeForMarkingJob` joint job, user, génération et `student_exam`; un QR cross-generation ou cross-user est refusé. | Tests de scope et regroupement. Aucun ID QR n'est une frontière autonome. |
| Homographie | Scan + PNG historique moderne ; géométrie de `student_exam_page_content`. | La référence est résolue et validée avant OpenCV. | Tests synthétiques et corpus réel opt-in. Legacy seulement : reconstruction Typst/image vivante lorsque toutes les métadonnées sont NULL. |
| Page alignée | PNG non annoté issu de l'homographie, stocké durablement sous le workspace Marking ; metadata dans `marking_aligned_pages`. | Cible corrected, job/copie/user/page exacts, storage key canonique, symlink/path confinement, fichier régulier, hash, PNG et dimensions ; metadata immutable. | Tests multi-pages/copies, bytes, détection, corruption, mauvais scope, traversal et metadata. Aucun fallback vers scan/PDF. |
| Détections et résultats | `marking_copy_results`, `marking_question_results`, `marking_answer_detections` ; score exact en half-units, état, `mean_gray`, index et outcomes persistés. Versions, seuil et delta résident sur le job. | Triggers de génération/user ; questions seulement sous copie corrected ; unicités par copie/question/réponse. | Repository transactionnel par copie, couverture terminale exigée avant passage success, tests DB et production synthétique. |
| Ambiguïtés | Politique snapshotée `ABS(mean_gray-threshold) <= ambiguity_delta`; delta moderne 5. Read models DB ordonnés par copie, question, réponse. | Job success + owner + corrected + aligned page. | Tests bornes, summary, ordre et legacy delta NULL. Aucune nouvelle lecture MeanGray dans les handlers. |
| Review humaine | `marking_answer_reviews` séparée ; `detected_state` reste intact, `effective_state = COALESCE(reviewed_state, detected_state)`. | Service exige owner, job success, detection du job et copie corrected. Review non ambiguë permise par le service/POST. Double optimistic locking réponse + job. | Tests confirmation, override, non-ambiguë, idempotence, concurrence, rollback et handlers. Legacy sans delta redirigé vers le résultat. |
| Recalcul | `ApplyMarkingAnswerReview` décode le snapshot, recalcule la question avec `markingscoring.ScoreQuestion`, puis somme toutes les questions de la copie. | Transaction SQLite unique : review, question, copie et `review_revision` commitent ou rollbackent ensemble. | Tests correct/partial/incorrect, half-units, plusieurs questions/reviews, conflit global et rollback. |
| Régénération | `corrected.pdf` vient des pages alignées copiées en staging et des effective states ; `mark-table.pdf` vient des résultats DB et snapshots. | Resolver aligned complet ; invariants scoring revérifiés ; aucune homographie/redétection. Publication filesystem à deux fichiers avec backups/restore, puis UPDATE DB conditionnel. | Tests génération, échec, PDF invalide, conflit concurrent, restauration et no-op current. Ce protocole est détectable/restaurable, **pas atomique multi-fichier**. |
| Résultat professeur | View models typés pour résultat/review ; fraîcheur calculée par égalité des révisions ; PDF stale masqués ; retry POST. | Tous les read models et actions utilisent user + job ; cross-user devient 404. | Tests lifecycle, stale/current, review/crop/POST/retry. Les pages upload/progress restent sur `ExtraData`. |

## 3. Vérification des anciens P1

### 3.1 Job et génération — fermé

`ProcessingMarkingHandler` exige un `exam_generated_id` positif, possédé et
`success`. `CreateMarkingJob` est un `INSERT ... SELECT` avec le même scope, et
la migration 0036 ajoute les triggers d'ownership. Chaque QR est ensuite validé
par `ValidateMarkingJobStudentExam` contre la génération attachée au job. Les
QR étrangers, inexistants et d'une autre génération du même propriétaire sont
donc refusés.

### 3.2 Historique moderne — fermé pour l'autorité métier

La correction initiale moderne utilise le snapshot QCM, la géométrie snapshotée
et les PNG historiques. Elle ne relit ni QCM, Question, Answer, image ni template
d'examen vivants. La régénération post-review utilise les snapshots, résultats
DB et pages alignées ; elle ne relance aucune détection et ne relit aucune image
pédagogique.

Les gabarits génériques qui composent le sommaire et le tableau PDF, ainsi que
l'outil Typst installé, restent des dépendances de rendu au moment d'une
régénération. Leur changement peut modifier la présentation ou faire échouer la
publication, mais ne change ni les scores ni l'autorité DB, et l'échec laisse les
artefacts stale avec retry. C'est une dette P3 de reproductibilité visuelle, pas
le P1 historique initial portant sur la référence d'homographie.

### 3.3 Persistance — fermé

Sont persistés : outcomes de copie, pages attendues/détectées, scores exacts en
half-units, points totaux, résultats par question, détections par réponse,
`detected_state`, `mean_gray`, version de schéma, version d'algorithme, threshold,
ambiguity delta, snapshots QCM/page, références historiques et pages alignées.
Le passage success exige une ligne terminale pour chaque copie attendue, des
enfants complets pour chaque copie corrected et toutes ses aligned pages.

La production n'est pas une transaction globale sur tout le job : chaque copie
corrected est persistée dans une transaction SQLite, les outcomes non corrected
sont insérés séparément, les fichiers alignés sont publiés ensuite, et la
transition terminale vérifie la couverture complète. En cas d'échec intermédiaire
le job devient failed ; il ne peut pas devenir faussement success.

### 3.4 Review humaine — fermé

`detected_state` n'est jamais mis à jour. La décision humaine est séparée et
l'effective state est dérivé dans les queries. Le service permet une review de
toute détection owned/corrected, même non ambiguë ; seule la file automatique est
limitée par la politique d'ambiguïté.

`ApplyMarkingAnswerReview` applique, dans une transaction SQLite unique : cible
et versions attendues, INSERT/UPDATE de review, recomposition du vecteur effectif,
score partagé, résultat question, somme de copie et révision globale. Les
conflits réponse/job rollbackent tout. Une répétition identique aux versions
valides est un no-op ; une version stale ne devient jamais last-write-wins.

### 3.5 Artefacts — fermé

`review_revision` et `artifacts_revision` sont orthogonales au lifecycle
running/success/failed. Current signifie strictement égalité ; un changement
effectif laisse les artefacts stale. La dernière candidate déclenche une seule
tentative de `RegenerateMarkingArtifacts`, et le retry utilise le même service.

Le corrected PDF est redessiné sur des copies temporaires des aligned pages ; le
mark table est reconstruit depuis DB/snapshots. Aucune redétection OpenCV n'est
faite. Une review concurrente fait échouer l'UPDATE conditionnel et restaure au
mieux l'ancienne paire. Un échec ne rollbacke jamais review/score et ne transforme
pas le job success en failed.

**P1 métier/historique/sécurité restant : non.**

## 4. Sécurité historique et endpoints

### Garanties réelles

- Ownership est répété aux frontières job, génération, QR, résultats, review,
  crop et régénération.
- Une détection cross-job est refusée par la chaîne job/copy/question/detection.
- Références Exam et aligned pages imposent une clé structurée exacte, un
  workspace dérivé de l'utilisateur et de l'entité, des parents non symlinkés,
  un fichier régulier, l'identité entre `Lstat` et fichier ouvert, SHA-256,
  décodage PNG et dimensions.
- Le crop ne vient que d'une aligned page validée, est borné à un contexte local,
  encodé en mémoire, `no-store` et `nosniff`.
- Les POST review/regenerate reposent sur la session puis sur les services
  ownership-aware. Les erreurs cross-user/inconnues sont des 404.
- Les erreurs utilisateur sont génériques ; paths, SQL, hash et stack traces ne
  sont pas rendus. Les logs techniques contiennent parfois paths et IDs, mais ne
  sont pas exposés dans les pages.

### `ServePDF`

`/dashboard/marking/servePDF` ne vérifie pas `job_id` en DB. Il reçoit
`operation` et `file`, puis confine la lecture à
`assets/tmp/<session-username>/<operation>/<file>`. Chaque composant est simple,
l'extension doit être PDF, les parents symlinkés et fichiers non réguliers sont
refusés, et le fichier ouvert doit être le même que celui inspecté.

Le confinement cross-user et traversal est robuste. Le risque réel est un scope
trop large **à l'intérieur du workspace du propriétaire** : une URL forgée peut
viser un autre PDF connu de ses propres opérations, y compris un artefact stale
que l'UX ne propose plus. Ce n'est pas un contournement d'ownership et ne justifie
pas P1/P2. Un endpoint `job_id + artifact enum + ownership DB + current` reste
une amélioration **P3**.

## 5. Audit des GET Marking

| GET | DB | Filesystem | Conclusion |
|---|---|---|---|
| `/dashboard/marking` | Liste des générations | Aucun | Lecture seule. |
| `/dashboard/marking/progress` | Statut et compteurs | Aucun | Lecture seule ; meta-refresh client. |
| `/dashboard/marking/success` | Job, summary, artefacts | `Lstat` seulement via `MarkingArtifactExists` | Lecture seule ; aucun `LeftPages`. |
| `/dashboard/marking/review` | Summary, candidats, snapshot | Aucun | Lecture seule. |
| `/dashboard/marking/review/crop` | Mapping owned | Lit/décode aligned PNG, encode en mémoire | Lecture seule. |
| `/dashboard/marking/servePDF` | Aucune DB | Lit et sert un PDF confiné | Lecture seule. |

**GET Marking mutatifs restants : 0.**

## 6. Admission, performance et quotas

Le corps multipart est limité à 100 Mio avec `MaxBytesReader`, le magic `%PDF-`
est vérifié, et les commandes externes ont chacune un timeout de deux minutes.
Les workers pages et copies sont limités à cinq **par job**.

Il n'existe en revanche aucune borne sur : nombre de pages, dimensions/complexité
après rasterisation, nombre de copies, durée globale du job, mémoire totale,
nombre de jobs simultanés par utilisateur ou global, ni quota disque. Plusieurs
jobs de 100 Mio peuvent donc lancer simultanément pdfseparate, rasterisation
PNG, QR, SIFT/homographie et Typst. Le contexte d'application permet l'arrêt du
pipeline, mais les opérations GoCV ne constituent pas un budget CPU/mémoire.

Sur une classe normale, le sémaphore par job limite le pic local et les tests
réels montrent que le pipeline fonctionne. Sous upload pathologique ou usages
concurrents, le serveur peut néanmoins devenir indisponible. Dette **P2** :
ajouter une limite de pages après split, une admission globale/per-user et un
budget/documentation d'exploitation avant exposition non contrôlée.

## 7. Recovery et lifecycle

`RecoverRunningMarkingJobs` supprime le workspace de chaque job running puis le
passe failed. Il est idempotent et conserve les jobs success. Mais la première
erreur de liste, de suppression ou d'UPDATE est retournée ; `cmd/server/main.go`
l'envoie à `log.Fatal`. Un seul workspace symlinké, illisible ou une erreur DB
peut donc empêcher toute l'application de démarrer. Dette **P2**.

Le recovery ne répare pas les `.review-artifacts-*` ou `.review-backup` laissés
par un crash de régénération sur un job success. Le protocole de révision garde
l'incohérence détectable et le retry reste possible, mais un nettoyage ciblé au
démarrage serait utile. Cela est inclus dans le jalon recovery, sans nécessiter
de nouveau lifecycle.

Le lifecycle technique nécessaire reste `running / success / failed`. Le review
status est correctement dérivé (`legacy_unavailable`, `no_review_needed`,
`pending`, `completed`) et la fraîcheur est l'égalité des révisions. `status_pdf`
est aujourd'hui redondant : la production le passe success dans le même UPDATE
que le job, et les reviews ne le modifient pas. Il reste utilisé par progress et
success pour la compatibilité historique. Le supprimer n'apporte pas assez de
valeur immédiate pour justifier une migration.

## 8. UX, view models et erreurs

Résultat et review ont des view models dédiés, sans row SQLC brute ni
`map[string]any`. Les IDs et révisions sont seulement dans URL/champs hidden.
Le vocabulaire est français, les PDF sont ouverts par action explicite, il n'y
a aucun popup automatique ni JavaScript de review, et les pages utilisent des
titres, labels et Bootstrap responsive.

Deux pages restent sur `data.MarkingPageData.ExtraData` :

- upload : `NoExamGenerated` et une slice de rows SQLC
  `GetExamsGeneratedSuccessRow` ;
- progress : job ID sous forme de chaîne, compteurs, `status` et `status_pdf`.

La page progress affiche encore les valeurs techniques anglaises sous
« Status » et « Status création pdf finaux ». L'upload utilise un JavaScript
drag/drop utile mais améliorable au clavier ; le backend reste l'autorité. Ces
points sont **P3** : un petit view-model/UX cleanup, pas une dette de sécurité.

Les handlers anciens renvoient encore plusieurs messages anglais génériques
(`Something went wrong`, `DB error`, `Invalid form`). Ils n'exposent aucun
détail sensible ; leur qualité éditoriale est P3.

## 9. Contrat de `corrected_NOT.pdf`

`corrected_NOT.pdf` est indépendant des answer reviews et des deux révisions.
La page résultat le sert séparément s'il existe ; la régénération review ne le
touche pas. Les compteurs `incomplete`, `error` et `not_seen` proviennent des
résultats persistés et restent disponibles même lorsque les PDF review-sensitive
sont stale.

Le code de création `LeftPages` existe et est testé, mais n'est appelé par aucun
chemin de production actuel. Depuis que le GET success est devenu read-only,
`ProcessMarking` laisse les PNG QR illisibles dans le workspace sans construire
le PDF. Les jobs historiques pour lesquels le GET ancien l'avait déjà créé le
conservent ; les nouveaux jobs ne le produisent normalement pas. De plus, les
noms techniques des pages QR illisibles ne sont pas persistés dans le modèle de
régénération du mark table, et le scan uploadé est supprimé à la fin du job.

La correction et les scores ne sont pas perdus, mais l'enseignant peut perdre
l'accès pratique aux pages rejetées après une correction moderne. Dette **P2** :
générer `corrected_NOT.pdf` dans le pipeline POST/background avant completion,
ou formaliser explicitement un autre artefact durable. Aucun GET ne doit redevenir
mutatif.

## 10. Compatibilité legacy

| Fallback | Raison et sécurité | Conclusion |
|---|---|---|
| Références de page toutes NULL | Générations antérieures à 0039 : reconstruction via snapshot + template/images vivants. Déclenchée uniquement sur absence complète ; corruption moderne refuse sans fallback. | Compatibilité légitime tant que ces jobs doivent rester corrigeables. Non reproductible si assets vivants changent ; documenter la durée de support. |
| `ambiguity_delta IS NULL` | Jobs antérieurs à la review : aucune file inventée, PDF historique accessible. | Compatibilité sûre et légitime. |
| Absence d'aligned pages | Les jobs legacy n'ont ni crop ni régénération post-review. Aucun fallback vers corrected PDF ou scan brut. | Refus sûr ; conserver. |
| Métadonnées résultat/version NULL | Jobs anciens sans résultats détaillés. Les nouvelles créations doivent fournir un ensemble complet. | Compatibilité DB nécessaire. |
| PDF historiques | Résultat legacy servi selon son ancien contrat et confinement username/workspace. | Légitime ; endpoint strict reste P3. |

## 11. Replay des migrations

La dette Goose est réelle et reproductible. Avec Goose v3.27.3 et une DB
temporaire neuve, la commande documentée :

`goose -dir db/migrations sqlite <temp>/replay.db up`

applique 0001 à 0035 puis échoue sur
`0036_bind_marking_jobs_to_exam_generation.sql` avec `incomplete input` lors du
premier `CREATE TRIGGER`. La migration 0036 ne possède pas les blocs
`StatementBegin/StatementEnd` ajoutés plus tard dans 0040. `db.InitDB` n'exécute
aucune migration ; le harness privé contourne le parseur Goose en exécutant le
SQL Up directement sur une copie.

Impact :

- installation neuve suivant le README : **cassée** ;
- CI actuelle : non détecté, car elle compile/teste sans replay complet Goose ;
- production déjà migrée : non affectée tant que son schéma est déjà à jour ;
- harness historique : contournement local explicite.

Dette **P2 prioritaire avant toute nouvelle installation/release**. La correction
doit être minimale et accompagnée d'un test CI de replay complet ; elle ne
demande aucun changement métier Marking.

## 12. Tests, CI et données privées

### Couverture actuelle

- unitaires : scoring, half-units, ambiguïté, offsets ROI, clamp crop, vecteurs ;
- DB/migrations : génération/job, résultats, constraints, ownership, révisions,
  effective state et read models ;
- services transactionnels : confirmation, override, idempotence, concurrence,
  rollback et copie multi-question ;
- filesystem : références et aligned pages, symlink/traversal/hash/dimensions,
  cleanup failed/purge, publication/rollback des deux PDF ;
- handlers : admission generation, lifecycle, résultat, review GET/POST, crop,
  retry, headers et ownership ;
- intégration synthétique : production détaillée et couverture terminale ;
- scans réels opt-in : une/deux/trois pages, batches, équivalence des références
  et calibration ambiguity ;
- race : workflow CI dédié `go test -race ./...`.

Les tests synthétiques sont suffisants pour protéger les invariants centraux.
Les trous à forte valeur sont : replay Goose complet en CI, admission d'un PDF
à trop grand nombre de pages, recovery poursuivant après une erreur isolée,
création moderne de corrected_NOT, et un scénario réel end-to-end incluant
upload → résultats → review → régénération.

La CI exécute `./scripts/check.sh` et `go test -race ./...` avec un timeout de
15 minutes. Les tests réels sont opt-in via variables d'environnement et se
skipent sans DB/PDF privés ; la CI publique n'en dépend pas.

`git ls-files` ne contient aucun `.db`, `.sqlite`, PDF, PNG/JPEG de scan ou crop
privé. Les DB, sorties Marking et corpus réel présents localement sont couverts
par `.gitignore` (`db/data/*.db`, `/assets/`, `.real-marking-integration-*`).
Aucun nom élève, scan privé, DB privée ou crop privé n'est versionné selon cet
inventaire. Il faut conserver cette discipline pour le futur test réel.

## 13. Transactions et cohérence

- Production : transaction SQLite par copie corrected ; inserts terminaux
  séparés ; publication aligned page fichier puis metadata ; transition job
  success conditionnelle sur la couverture complète. Ce n'est pas une
  transaction globale DB + filesystem.
- Review : transaction SQLite unique et véritablement atomique pour review,
  question, copie et révision job.
- Régénération : génération en staging, renames de deux fichiers avec backups,
  restauration best-effort, puis UPDATE DB optimiste. La paire de fichiers
  n'est **pas atomique** ; la révision rend l'état observable et retryable.

## 14. CSRF

Aucun token/middleware CSRF explicite n'est présent. Les cookies de session sont
`HttpOnly`, configurables `Secure` et `SameSite=Strict`, ce qui réduit fortement
les soumissions cross-site classiques, mais n'est pas un mécanisme CSRF complet
pour toutes les topologies (contenu same-site, sous-domaines, mauvaise
configuration HTTPS). La dette est globale à l'application, pas propre aux POST
review/regenerate.

Classement **P2 conditionnel** : à traiter avant une V1 Internet multi-utilisateur
ou exposée à du contenu same-site non maîtrisé ; acceptable après V1 seulement
pour un déploiement local/contrôlé explicitement assumé avec HTTPS et
`SESSION_SECURE=true`.

## 15. Contrat de robustesse d'un job moderne success

- Questions ou QCM modifiés ensuite : aucun effet sur résultat/review ; snapshot
  historique utilisé.
- Template d'examen modifié : aucun effet sur la référence ou homographie ; PNG
  historique utilisé. Les templates génériques de sommaire/table peuvent changer
  la présentation d'une future régénération.
- Image pédagogique supprimée : aucun effet sur correction moderne ni
  régénération post-review ; les pixels nécessaires sont déjà dans les références
  puis aligned pages.
- Redémarrage : job success et artefacts conservés. Un job running est nettoyé
  puis failed, mais une erreur de ce recovery peut bloquer le démarrage complet.
- Review : décision, effective state, question, copie et révision commitent
  atomiquement ; la dernière décision tente la régénération.
- Review concurrente : optimistic locking answer + job ; aucune écriture stale.
- Génération PDF échoue : reviews/scores/job success conservés, artefacts stale,
  anciens PDF non présentés comme finaux, notice et retry.
- Aligned page corrompue : hash/PNG/dimensions font échouer crop/régénération ;
  aucun fallback moins sûr et DB reste autoritative, mais il n'existe pas de
  restauration automatique de cette source durable.
- Accès au job d'un autre utilisateur : read models/actions retournent 404 ; les
  fichiers sont isolés par le username de session.

## 16. Classement final

| Priorité | Sujet | État | Impact | Action recommandée |
|---|---|---|---|---|
| CLOSED | Liaison job/génération et QR | Fermé | Plus de mélange cross-generation/user. | Conserver tests et triggers. |
| CLOSED | Historique moderne, résultats et pages alignées | Fermé | Autorité DB/snapshots, indépendance des tables vivantes. | Test réel périodique. |
| CLOSED | Review, recalcul et concurrence | Fermé | Décisions humaines exactes, transactionnelles et idempotentes. | Conserver service unique. |
| CLOSED | Artefacts review-sensitive | Fermé | Stale/current, regen sans OpenCV, retry et conflit protégés. | Conserver protocole détectable ; ne pas le qualifier d'atomique. |
| P2 | Admission et capacité | Ouvert | DoS CPU/RAM/disque par pages/jobs non bornés. | Limite pages + admission globale/per-user + budget job. |
| P2 | Recovery startup | Ouvert | Une erreur isolée peut déclencher `log.Fatal`; résidus de regen non réconciliés. | Recovery best-effort par job, logs/alerte, cleanup sûr des résidus. |
| P2 | Replay Goose 0036 | Ouvert, reproduit | Installation neuve documentée impossible ; CI aveugle. | Corriger le découpage Goose et ajouter un replay fresh DB en CI. |
| P2 | `corrected_NOT.pdf` moderne | Ouvert | Pages rejetées non proposées après un nouveau job ; noms QR illisibles non persistés. | Produire durablement hors GET ou formaliser un nouvel artefact. |
| P2 | CSRF global | Ouvert, dépend du déploiement | POST authentifiés protégés seulement par SameSite Strict. | Protection globale avant exposition Internet ; décision documentée sinon. |
| P3 | `ServePDF` | Robuste mais large | Accès possible aux propres PDF connus hors enum/job/current. | Endpoint job + enum + ownership/fraîcheur. |
| P3 | Upload/progress view models et UX | Legacy de présentation | ExtraData/row SQLC, statuts anglais, erreurs génériques. | Petit cleanup typé et français après clôture. |
| P3 | Reproductibilité/entretien artefacts | Partiel | Templates génériques vivants, backups/temp de crash, logs paths/IDs. | Versionner/documenter renderer et nettoyage ciblé, sans refonte métier. |
| POST-V1 | Journal append-only de review / observabilité | Non requis V1 | Une seule review courante et métriques limitées. | Ajouter seulement si besoin d'audit métier/support. |

## 17. Roadmap courte

### Jalon A — déployabilité

Réparer le replay Goose 0036 et l'ajouter à la CI ; rendre le recovery Marking
best-effort sans `log.Fatal` pour une erreur de workspace isolée.

### Jalon B — capacité et pages rejetées

Ajouter borne de pages/admission de jobs, puis produire le contrat durable de
`corrected_NOT.pdf` dans le pipeline non-GET.

### Jalon C — sécurité de déploiement et preuve réelle

Décider/implémenter la protection CSRF globale selon la cible, puis exécuter un
test privé end-to-end complet incluant review et régénération. Les P3 peuvent
attendre après V1.

## 18. Validations lecture seule

À la rédaction initiale du rapport :

- replay Goose v3.27.3 sur DB temporaire : **échec reproduit à 0036** ;
- `./scripts/check.sh` : **succès** ;
- `go test -race ./...` : **succès** ;
- `git diff --check` : **succès**.

Seul `docs/audits/marking-final-closure-audit.md` a été créé par cet audit.
