# P1.1 — Bilan cumulé et rattrapages

## État initial

Le chantier P1 avait conservé un `marking_job` indépendant pour chaque dépôt
et ajouté une lecture cumulée par `student_exam` pour une même
`exams_generated`. Les anciennes corrections survivaient aux imports suivants,
les notes avec revue en attente étaient masquées et les décisions humaines
pouvaient être rouvertes.

Le bilan était cependant intégré à la page de résultat d'un job. Le lien PDF
utilisait le `mark_table_name` de ce job : après `lot-1.pdf` puis `lot-2.pdf`,
l'écran réunissait les corrections, mais le PDF ne contenait que celles du lot
consulté. L'entrée de correction imposait aussi de sélectionner à nouveau la
génération pour ajouter des copies. Le tri SQL utilisait `COLLATE NOCASE`, qui
ne traite pas les accents comme une collation française. « Import source »
pointait vers une autre page portant le même bilan cumulé.

| Problème | Conséquence | Gravité |
|---|---|---|
| PDF local présenté auprès du bilan global | document incomplet pour l'évaluation | haute |
| Navigation centrée sur le job | rattrapage perçu comme une nouvelle correction indépendante | moyenne |
| Tri ASCII et affichage prénom/nom | recherche d'un élève peu naturelle | moyenne |
| Lien « Import source » ambigu | navigation sans information utile supplémentaire | faible |

Référence historique :
[audit P1](../audits/2026-09-20-real-correction-workflow-audit.md).

## État trouvé lors de cette reprise

La reprise a commencé par `git status --short`, `git diff --stat`, `git diff`
et la lecture des fichiers pertinents non suivis. Aucun rapport P1.1 n'existait
dans `docs/reports/` ou `docs/audits/`.

Étaient déjà implémentés :

- la page de bilan identifiée par la génération et son historique d'imports ;
- le téléchargement d'un PDF cumulé calculé à la demande ;
- une source commune de données HTML/PDF ;
- le formulaire de rattrapage avec génération préremplie ;
- le retour du traitement vers le bilan, et l'accès explicite aux documents
  historiques de chaque import ;
- le tri français commun et « Import source » en texte ;
- les premiers tests de cycle cumulé, ownership, revue et tri ;
- le nettoyage des espaces temporaires de bilan par la purge existante.

Les derniers tests ciblés passaient. Il restait la validation complète, les
cas de compatibilité et de rendu supplémentaires, ainsi que ce rapport.
L'architecture a été conservée. La reprise a complété les tests des anciens
jobs sans génération, de l'historique dépassant vingt imports, des PDF vides et
multipages, de l'échappement du texte et du nettoyage après annulation. Le tri
SQL redondant a été retiré pour conserver une seule règle de présentation.

Les modifications P1 déjà présentes ont été conservées. Les changements
préexistants de `README.md` et le fichier non suivi `reset.sh` n'ont pas été
modifiés par ce chantier. Aucun commit ni push n'a été effectué.

## Architecture finale

```text
exams_generated
├── bilan courant HTML
├── bilan-evaluation.pdf, produit à la demande
└── historique des marking_job
    ├── import 1 : résultats locaux, revues, PDF de correction
    └── import 2 : résultats locaux, revues, PDF de correction
```

### Lectures et routes

| Entrée | Fonction |
|---|---|
| `GET /dashboard/marking/results?exam_generated_id=…` | bilan de l'évaluation |
| `GET /dashboard/marking/results/pdf?exam_generated_id=…` | téléchargement du bilan cumulé |
| `GET /dashboard/marking?exam_generated_id=…` | dépôt des copies supplémentaires pour cette génération |
| `POST /dashboard/marking/processing` | création habituelle d'un nouveau job |
| `GET /dashboard/marking/success?job_id=…` | redirection vers le bilan de sa génération |
| `GET /dashboard/marking/success?job_id=…&import=1` | documents et revue propres à cet import |

`GetMarkingGeneration` contrôle l'appartenance de la génération, de son examen
et de sa classe, et exige une génération réussie. Les deux handlers de bilan
appellent `loadMarkingGeneration`, qui utilise
`ListCurrentExamResultsForGeneration`, puis `buildMarkingExamSummary`.
Le générateur PDF reçoit ce même modèle de présentation ; il ne sélectionne
ni les jobs ni les corrections.

Les requêtes sont définies dans `db/query/markingWorkflow.sql` ; le fichier
sqlc a été régénéré. Les anciens points d'entrée Go par job restent de simples
adaptateurs vers la lecture par génération, sans deuxième requête métier.

### Choix de la correction courante

La règle P1 est conservée :

1. seuls les jobs dont `status` et `status_pdf` sont `success` participent ;
2. les résultats sont regroupés par `student_exam_id` pour cette génération ;
3. une copie `corrected` est prioritaire sur `not_seen`, `incomplete` ou `error` ;
4. entre corrections réussies, le job ayant le plus grand identifiant gagne,
   conformément à la notion de récence déjà utilisée ; l'identifiant du
   résultat départage une éventuelle égalité ;
5. sans correction réussie, l'incident terminal le plus récent est affiché ;
6. les candidats de revue non résolus de la copie sélectionnée masquent sa
   note, même si un score provisoire est enregistré.

Une nouvelle correction avec revue pending devient donc l'état courant « À
vérifier ». Les jobs échoués ou partiels n'y contribuent pas. Modifier une
ancienne correction qui n'est plus courante ne remplace pas une correction
réussie plus récente.

### Invariants

- Les jobs et leurs résultats historiques ne sont ni fusionnés ni écrasés.
- Un rattrapage conserve la génération et ajoute un nouvel import.
- Une absence ou un incident ultérieur ne dégrade pas une correction acquise.
- HTML et PDF partagent sélection, états, notes masquées et ordre des élèves.
- Aucune note pending n'est présentée comme définitive.
- Les lectures et les dépôts restent limités à l'utilisateur connecté.
- Un téléchargement du bilan ne change aucune donnée ni révision de job.

## UX

### Ajouter les copies manquantes

Le bouton du bilan ouvre le formulaire existant avec `exam_generated_id`.
Le nom de l'évaluation et sa classe sont affichés ; le sélecteur est remplacé
par un champ caché. L'utilisateur choisit seulement son PDF. Un retour direct
au bilan est également proposé.

Le champ caché n'est pas une protection : le POST conserve sa propre
vérification d'ownership et de statut. Le traitement crée un nouveau job,
passe par la progression et les éventuelles revues. Le résultat normal
redirige vers le bilan de la même génération. Une erreur de régénération des
documents locaux conserve l'écran d'alerte et son action de réessai.

### Navigation et historique

Le bilan est accessible depuis les générations disponibles et les imports
récents. L'historique de l'évaluation est secondaire et contient tous ses
imports ; la liste générale reste limitée aux vingt plus récents.

Chaque import indique sa date de fin lorsqu'elle existe, son fichier source,
ses compteurs et l'action appropriée : progression, vérification ou documents.
La page historique précise que ses PDF concernent uniquement cet import.
Les anciens jobs sans génération conservent leur page et leurs liens PDF.

« Import source » est une information non cliquable (`Import N`). Les actions
utiles se trouvent dans l'historique.

### Tri alphabétique

`buildMarkingExamSummary` applique une seule collation française
(`golang.org/x/text/collate`, dépendance déjà présente), sans distinction de
casse, sur le nom puis le prénom. Les accents et les formes Unicode
équivalentes sont pris en charge. Les espaces aux extrémités sont ignorés ;
les noms de famille absents viennent en dernier. À nom et prénom équivalents,
`student_exam_id` garantit un ordre déterministe.

La liste triée est partagée par HTML et PDF, sans tri supplémentaire dans le
générateur ou les templates. Le tri SQL ASCII a été supprimé. Un nom totalement
vide est affiché « Élève sans nom ».

## PDF cumulé et artefacts

### Artefact d'évaluation

`BuildMarkingGenerationPDF` produit `bilan-evaluation.pdf` à chaque demande.
Le document indique l'évaluation, la classe et la génération, puis les mêmes
élèves, états, notes et imports sources que le tableau. Les absents ou copies
incomplètes restent des lignes sans note, conformément au bilan P1 ; une ligne
ne signifie donc pas nécessairement une copie corrigée.

Le rendu Typst utilise le compilateur existant. Les textes passent par
`typstStringLiteral`, les en-têtes de tableau se répètent sur les pages et les
PDF vides contiennent un message explicite. Les compilations ont le timeout
existant et suivent l'annulation de la requête.

### Provenance, invalidation et régénération

L'URL, le contrôle d'accès et les données du bilan sont identifiés par
`exams_generated_id`, jamais par « le dernier job ». Il n'existe ni table
d'artefacts cumulés ni fichier de bilan durable à invalider.

Chaque téléchargement relit l'état courant et répond avec
`Cache-Control: no-store`. Un nouvel import réussi ou un recalcul transactionnel
après revue est donc pris en compte au téléchargement suivant. Un PDF déjà
téléchargé reste naturellement un instantané ; il faut le télécharger à
nouveau après une modification.

Les fichiers de compilation sont isolés dans
`assets/tmp/<utilisateur>/bilan-<uuid>/`, puis supprimés après lecture ou erreur.
La purge éphémère existante reconnaît aussi ces dossiers : les résidus d'un
arrêt brutal deviennent éligibles après une heure.

### Artefacts historiques des imports

Les fichiers existants de `assets/tmp/<utilisateur>/marking-<job>/` restent
inchangés :

- `corrected.pdf` : copies annotées du lot ;
- `corrected_NOT.pdf`, lorsqu'il existe : pages non corrigées ;
- `mark-table.pdf` / `mark_table_name` : tableau des notes local ;
- pages alignées et données nécessaires aux crops de revue.

Leur invalidation et leur régénération utilisent toujours `review_revision`
et `artifacts_revision`. Le bilan à la demande n'attend pas qu'un PDF local
obsolète soit régénéré : il utilise les scores déjà recalculés en base.

## Tests

| Test | Couverture |
|---|---|
| `TestCumulativeResultsAndPDFThroughImportsAndHumanDecisions` | premier lot, rattrapage, `not_seen` ultérieur, revue pending, décision modifiée, correction plus récente, exclusion des jobs failed/running, conservation des scores historiques |
| `TestCumulativeNavigationAndOwnership` | action du bilan, génération préremplie, redirections progression/résultat, documents par import, génération vide, refus des générations étrangères/en cours/inexistantes, ancien job sans génération |
| `TestBuildMarkingExamSummaryFrenchOrderAndMissingNames` | accents, casse, Unicode composé/décomposé, prénom, blancs, noms absents, égalités, stabilité indépendante de l'ordre d'entrée |
| `TestGenerationImportHistoryIsCompleteAndScoped` | plus de vingt imports par évaluation, limite générale de vingt, isolation entre générations/utilisateurs |
| `TestMarkingGenerationPDFEmptyAndMultipage` | compilation réelle, document vide, cent élèves, en-têtes répétés, texte échappé, annulation et nettoyage |
| Tests de purge éphémère étendus | dossiers `bilan-<uuid>` anciens supprimés, récents conservés, autres workspaces préservés |

Le test de cycle compile puis extrait réellement les PDF avec `pdftotext` à
chaque étape. Il vérifie les noms, leur ordre, les états et les scores du
modèle partagé avec l'écran. Typst et `pdftotext` étaient disponibles : ces
assertions ont été exécutées. Sur une installation sans ces outils, le test
de cycle conserve les assertions HTTP/DB et signale l'extraction non exécutée ;
le test de rendu est explicitement sauté.

Les tests P1 de résultat courant et les tests existants de POST d'import,
ownership, revue transactionnelle et artefacts ont été conservés.
`TestProcessingMarkingHandlerRequiresOwnedSuccessfulGeneration` vérifie déjà
que le POST conserve la génération choisie et refuse une génération étrangère.

### Validation exécutée

- `go test ./...` : succès.
- `go test ./internal/handlers/marking ./internal/handlers/tools ./internal/db -count=1` : succès.
- Suites ciblées P1.1 avec compilation/extraction PDF : succès.
- `sqlc generate -f db/sqlc.yaml` : succès.
- `git diff --check` : succès.

Les tests privés `TestReal*` ont été explicitement recensés avec `-v` : ils
sont sautés faute des variables `LAZYMARKING_TEST_DB`,
`LAZYMARKING_TEST_USER_ID`, `LAZYMARKING_TEST_BATCH_PDFS`, des chemins/identifiants
des corpus historiques, ou des variables de calibration correspondantes.
Le test privé de pipeline QR exige également les trois corpus historiques.
Ils ne sont pas comptés comme une validation effective des scans réels.

## Compatibilité

Aucune migration ajoutée ou modifiée, aucune nouvelle colonne ou table.
`golang.org/x/text` garde la même version ; son utilisation directe est
simplement déclarée dans `go.mod`.

Contrôle local réalisé sur des sauvegardes SQLite temporaires, ouvertes depuis
les bases sources en lecture seule, puis migrées avec les migrations existantes :

| Source | Version source | Version de la copie | Résultat |
|---|---:|---:|---|
| `testdata/smoke/app.db` | 40 | 44 | lectures bilan/historique/génération réussies |
| `testdata/real/app.db` | 41 | 44 | lectures bilan/historique/génération réussies |
| `testdata/2026-2027/app.db` | 44 | 44 | lectures bilan/historique/génération réussies |

Sur la copie 2026-2027, la génération 2 a trois imports et 34 états d'élèves,
dont 30 corrigés et aucune copie pending. Les trois requêtes fonctionnent
sur cet état réel persistant. Ce contrôle SQL ne constitue pas une nouvelle
exécution du pipeline de scans.

Les empreintes SHA-256 des trois bases sources et de `lot-1.pdf` / `lot-2.pdf`
étaient identiques avant et après cette validation. Les dossiers
`runtime/smoke`, `runtime/real` et `runtime/2026-2027` n'ont pas été modifiés.

### Validation manuelle des deux PDF terrain

Les fichiers sont présents dans `testdata/2026-2027/scans/`. Le helper privé
`openCopiedHistoricalDBWithConnection` vise actuellement exactement la version
39 ; il n'est pas adapté tel quel à cette base en version 44 et ne couvre pas
le nouveau parcours HTTP cumulé. Il n'a pas été modifié pour ce chantier.

Procédure sans modification des sources :

1. Préparer une copie isolée du runtime 2026-2027. Remplacer le lien de base
   par une vraie sauvegarde SQLite indépendante ; copier aussi les artefacts
   de génération et les images nécessaires. Ne pas utiliser le script
   `run-2026-2027.sh` pour cet essai isolé : il pointe vers la base source.
2. Démarrer le serveur depuis cette copie, avec une session distincte et les
   templates/code à jour. Pour observer un premier import vierge, utiliser une
   sauvegarde de cette génération prise avant correction ; autrement les
   corrections historiques restent volontairement cumulées.
3. Déposer `lot-1.pdf`, terminer les revues puis ouvrir le bilan et son PDF.
4. Cliquer « Ajouter les copies manquantes », vérifier l'évaluation affichée,
   déposer `lot-2.pdf` et terminer les éventuelles revues.
5. Vérifier le retour au bilan, la conservation des notes précédentes, les
   ajouts, l'ordre nom/prénom et la concordance avec le nouveau PDF.
6. Depuis l'historique, ouvrir séparément les documents des deux imports.
7. Modifier une décision humaine, puis retélécharger le bilan et vérifier le
   score actualisé. Les documents locaux suivent leur régénération habituelle.

## Résultat final

Le bilan et son PDF appartiennent à l'évaluation. L'ajout de copies conserve
automatiquement sa génération et revient au même bilan. L'historique et les
documents de chaque import restent accessibles séparément. Les tests
automatisés du P1.1 et les lectures sur copies des bases existantes passent.

## Points restant ouverts

- La reprise exacte d'un traitement après arrêt brutal du serveur reste le
  sujet de recovery identifié par P1 ; elle n'est pas modifiée ici.
- L'adaptation des helpers privés historiques aux corpus modernes et une
  automatisation navigateur du scénario terrain complet peuvent faire l'objet
  d'un chantier de tests dédié. La procédure terrain ci-dessus n'a pas été
  exécutée pendant cette validation.
