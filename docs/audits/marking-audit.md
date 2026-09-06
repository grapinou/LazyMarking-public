# Audit complet — Correction / Marking

## 1. Synthèse exécutive

Le pipeline Marking est fonctionnel et comporte déjà plusieurs fondations solides : isolation des workspaces par utilisateur et job, commandes externes sans shell et avec timeout, regroupement déterministe des pages QR, lectures des contenus/points historiques, transitions DB conditionnelles, cleanup automatique sur échec, recovery au démarrage et protections de téléchargement contre traversal et symlinks.

Le domaine n'est toutefois pas prêt pour une simple refonte UX. Quatre risques P1 touchent le contrat central :

1. un job n'est lié à aucune génération et peut mélanger plusieurs évaluations possédées ;
2. la correction dépend encore du template Typst et des fichiers image vivants ;
3. les réponses détectées, scores et résultats ne sont pas persistés et les seuls PDF sont purgés après sept jours ;
4. la reconnaissance applique un seuil binaire fixe sans état ambigu ni circuit de validation.

Le verdict est donc **C — travail structurel significatif avant UX**.

Décompte :

- P0 : 0 ;
- P1 : 4 ;
- P2 : 5 ;
- P3 : 3.

## 2. Cartographie du domaine

### 2.1 Graphe DB et filesystem

```text
users
 ├─ exams
 │   ├─ qcm
 │   ├─ class_codes
 │   ├─ years
 │   └─ periods
 │
 ├─ exams_generated [running|success|failed]
 │   └─ student_exam
 │       ├─ students
 │       ├─ student_exam_content
 │       │    └─ JSON config.QCM
 │       └─ student_exam_page_content
 │            └─ JSON config.PageContent par numéro de page
 │
 └─ marking_jobs [running|success|failed]
      ├─ compteurs pages/copies
      ├─ status_pdf
      ├─ noms de deux PDF finaux
      └─ completed_at

marking_jobs n'a aucune FK vers exams_generated, exams ou student_exam.

assets/tmp/<username>/marking-<job_id>/
 ├─ pages PDF/PNG et homographies temporaires
 ├─ student-exam-<student_exam_id>.pdf temporaires
 ├─ corrected.pdf
 ├─ mark-table.pdf
 └─ corrected_NOT.pdf éventuel (créé depuis les pages restantes)
```

### 2.2 Tables réellement utilisées

| Table | Rôle Marking |
| --- | --- |
| `marking_jobs` | lifecycle, compteurs, noms des PDF finaux |
| `exams_generated` | source indirecte de l'identité historique ; consultée seulement pour la liste d'entrée |
| `exams` | nom/classe de la liste d'entrée |
| `class_codes` | nom de classe de la liste d'entrée |
| `student_exam` | identité de copie encodée dans le QR |
| `student_exam_content` | snapshot JSON complet du QCM individualisé et nombre de pages |
| `student_exam_page_content` | snapshot JSON des coordonnées questions/réponses par page |
| `students` | relation historique de `student_exam`, pas relue pour calculer la note |
| `users` | ownership, username des workspaces et recovery/purge |

Les tables vivantes `questions`, `alt_questions`, `answers`, `points` et `qcm_questions` ne sont pas lues pendant le calcul Marking.

### 2.3 Requêtes principales

- `GetExamsGeneratedSuccess` ;
- `CreateMarkingJob` ;
- `UpdateMarkingJobTotalPages`, `UpdateMarkingJobPageDone` ;
- `UpdateMarkingJobTotalExam`, `UpdateMarkingJobExamDone` ;
- `GetMarkingStatus`, `GetMarkingProgress` ;
- `CompleteMarkingJob`, `FailMarkingJob`, `DeleteMarkingJob` ;
- `GetExamAndMarkName` ;
- `ListRunningMarkingJobs`, `ListExpiredMarkingJobs` ;
- `GetStudentContentExam`, `GetPageContent`.

### 2.4 Structs métier/runtime

- `config.QrCodeInfo` : `StudentExamID`, `PageExam`, `PageName` ;
- `config.Exam` / `config.Page` : regroupement d'une copie et de ses pages ;
- `config.QCM` : snapshot de l'évaluation individualisée ;
- `config.PageContent` : géométrie historique ;
- `config.MarkExam` : résultat en mémoire d'une copie ;
- `db.MarkingJob` : état persistant du job ;
- `data.MarkingPageData` : données de vue dynamiques via `ExtraData`.

## 3. Workflow enseignant et routes

### Routes réelles

| Méthode | Route | Handler |
| --- | --- | --- |
| GET | `/dashboard/marking` | `AddPdfFormMarkingHandler` |
| POST | `/dashboard/marking/processing` | `ProcessingMarkingHandler` |
| GET | `/dashboard/marking/progress?job_id=...` | `ProgressMarkingHandler` |
| GET | `/dashboard/marking/success?job_id=...` | `SuccessMarkingProcessingHandler` |
| GET | `/dashboard/marking/servePDF?...` | `ServeFullMarkingPdfHandler` |

### Parcours observé

```text
Liste informative des générations success
        ↓
Upload d'un PDF (aucun choix de génération envoyé)
        ↓
staging OS temporaire + création marking_job
        ↓
goroutine ProcessMarking
        ↓
split PDF → PNG → QR → groupes student_exam
        ↓
reconstruction snapshot → homographie → lecture cases → score
        ↓
corrected.pdf + mark-table.pdf
        ↓
progression par meta-refresh 2 s
        ↓
page success → popups automatiques des PDF
```

La table des évaluations success est seulement informative : elle ne contient ni select, ni `exam_generated_id`, ni action par évaluation. Le POST contient uniquement `pdffile`.

## 4. Condition d'entrée

`GetExamsGeneratedSuccess` filtre correctement :

- `exams_generated.status = 'success'` ;
- ownership de `exams_generated`, `exams` et `class_codes` par `user_id`.

Mais cette condition n'est pas reliée au job créé. `CreateMarkingJob` reçoit seulement `user_id`. Les QR sont ensuite acceptés sur la base de `student_exam_id` et des snapshots possédés, sans vérifier :

- le parent `exams_generated` ;
- son statut `success` ;
- qu'il correspond à une génération sélectionnée ;
- que tous les QR du fichier appartiennent à la même génération.

Conclusion : **l'entrée effective n'est pas limitée aux générations success**. Un PDF peut mélanger plusieurs générations du même utilisateur. Un QR étranger est rejeté par les lectures ownership-aware, mais un QR possédé d'une autre génération est accepté.

## 5. Upload du scan

`ProcessingMarkingHandler` appelle `CheckPdfFile` avant la création du job :

- limite `100 << 20` sur le corps multipart complet via `MaxBytesReader` ;
- parsing multipart après installation de la limite ;
- champ attendu `pdffile` ;
- vérification des cinq octets `%PDF-` ;
- rewind obligatoire ;
- copie vers `os.CreateTemp("", "lazymarking-upload-*.pdf")`, donc nom serveur et permissions OS sûres ;
- suppression du fichier de staging à la fin de la goroutine.

L'extension et le Content-Type client ne sont pas des autorités serveur. Le pipeline `pdfseparate` valide ensuite réellement la structure. Plusieurs parties fichier ne sont pas explicitement refusées : `FormFile` prend la première et la limite globale borne l'enveloppe.

La bibliothèque `net/http` supprime les fichiers temporaires multipart en fin de requête ; le fichier de staging est explicitement fermé/supprimé. Aucun nom d'upload utilisateur n'entre dans un chemin de workspace.

Limites : aucune borne sur le nombre de pages ou la complexité du PDF, et aucune limite globale du nombre de jobs simultanés.

## 6. Conversion PDF et commandes externes

Le pipeline utilise :

- `pdfseparate` pour les pages ;
- `pdftoppm -png -singlefile` pour les PNG ;
- `pdfunite` pour les fusions ;
- `typst compile` pour reconstruire les références et produire les synthèses ;
- GoCV/OpenCV pour QR fallback, SIFT, homographie, lecture et annotation ;
- `gofpdf` pour reconvertir les PNG corrigés en PDF.

Toutes les commandes sont lancées avec `exec.CommandContext`, sans shell : les chemins ne peuvent donc pas injecter une commande. Chaque commande externe possède un timeout de deux minutes. Il n'existe toutefois pas de timeout global du job ; un document à nombreuses pages enchaîne de nombreux processus bornés individuellement.

Les workspaces sont `assets/tmp/<username>/marking-<jobID>`. Les composants sont validés, les parents symlinks sont refusés et chaque job a un identifiant distinct.

## 7. Identification des copies et ownership QR

Le QR contient seulement :

```json
{"student_exam_id": 123, "page_exam": 2}
```

`PageName` n'est jamais accepté depuis le QR : il est remplacé par le nom interne du PNG issu du split.

Après décodage :

- `GroupQrCodes` groupe par `StudentExamID` ;
- les pages sont triées par numéro puis nom ;
- `GetStudentContentExam` exige `student_exam_content.user_id` et l'existence du `student_exam` avec le même utilisateur ;
- `GetPageContent` applique la même contrainte.

Les triggers de génération garantissent à la création que `student_exam`, l'élève et `exams_generated` partagent l'utilisateur. L'isolation inter-utilisateur est donc correcte.

Lacune : les requêtes Marking ne rejoignent pas `exams_generated` pour en vérifier le statut ou l'identité attendue.

## 8. Pages mélangées, ordre et doublons

Les pages peuvent être dans n'importe quel ordre dans le PDF : le QR fournit `student_exam_id` et `page_exam`, puis le regroupement trie explicitement copies et pages.

Une copie n'est corrigée que si :

- le nombre de pages scannées égale `page_tot` ;
- le nombre de pages de référence régénérées correspond ;
- chaque numéro de `1` à `page_tot` est présent ;
- les vecteurs questions/réponses/états ont exactement les longueurs attendues.

Doublons :

- une page dupliquée augmente la cardinalité ou remplace un numéro dans la map ; les contrôles nombre/séquence font échouer la copie ;
- une copie complète dupliquée dans le même PDF produit également trop de pages et n'est pas double-comptée ;
- la même copie dans deux jobs distincts est corrigée deux fois dans deux workspaces séparés ; aucun résultat DB ne crée de conflit.

Le comportement est donc un refus de la copie dupliquée, pas un écrasement arbitraire.

## 9. Pages manquantes, QR illisibles et traitement partiel

- QR illisible : le PNG est placé dans `qrNotDetected` ;
- page manquante ou numéro incohérent : la copie est placée dans `notMarkedExams` ;
- page blanche/inconnue : QR généralement illisible, donc page restante ;
- QR étranger/inexistant : lecture snapshot en échec, donc copie non corrigée ;
- si aucune copie n'est corrigeable, le job devient `failed` ;
- si au moins une copie est corrigeable, le job peut devenir `success` malgré des pages/copies non traitées.

Les PNG restants sont transformés en `corrected_NOT.pdf` seulement lors de la consultation GET de la page success. Le tableau de synthèse mentionne les noms de fichiers QR illisibles et les copies non corrigées, mais le statut global reste `success`.

Le compteur `done_pages` n'est incrémenté que lorsque le QR est lu et l'update DB réussit ; il peut donc rester inférieur à `total_pages` sur un job finalement success.

## 10. Snapshot historique utilisé

La correction utilise principalement les snapshots :

- `student_exam_content.content` est décodé en `config.QCM` ;
- ce JSON contient le nom d'Exam, l'élève/la classe, les questions/variantes réellement choisies, l'ordre des réponses, leurs états attendus, les points et les métadonnées pédagogiques ;
- `student_exam_page_content.content` contient les coordonnées/rayons historiques par page ;
- aucune requête tardive vers `questions`, `alt_questions`, `answers`, `points` ou la composition QCM vivante n'est effectuée.

Le snapshot est néanmoins **partiel pour la capacité de correction** :

- `TypstWriter` relit le template vivant `internal/config/ref_qcm.txt` ;
- les images sont seulement référencées par leur nom dans le JSON et sont relues depuis `assets/images` ;
- une suppression d'image devenue possible après évolution de la banque, ou une évolution du template/layout, peut empêcher la reconstruction ou la rendre différente de l'original.

La formulation, les réponses, variantes, points et géométrie sont figés ; les dépendances de rendu ne le sont pas entièrement.

## 11. Lecture des réponses

Pour chaque page :

1. reconstruction d'une page de référence à 300 PPI ;
2. homographie SIFT/BFMatcher ;
3. ratio test `0.65`, minimum quatre correspondances ;
4. RANSAC avec seuil `2.0`, 10 000 itérations, confiance `0.9999` ;
5. warp vers la géométrie de référence ;
6. mesure de gris dans une ROI centrée sur chaque cercle ;
7. moyenne `< 150` → cochée (`1`), sinon non cochée (`0`).

Les ROI sont confinées à l'image et une ROI vide produit une erreur. Une rotation/déformation est traitée indirectement par l'homographie tant que SIFT trouve suffisamment de correspondances.

Il n'existe aucun état « ambigu », aucune bande de confiance autour de 150, ni validation humaine d'une case limite. Le résultat binaire alimente directement la note.

## 12. Calcul du score et stabilité des points

Pour chaque question :

- vecteur exactement égal au corrigé → totalité des points ;
- sous-ensemble non vide ne contenant que des bonnes réponses, sans dépasser leur nombre → moitié des points ;
- réponse fausse, sur-sélection ou absence de réponse → zéro.

`CountingTotalPoint` somme les scores `float64` et les totaux entiers. Aucun arrondi DB n'existe ; l'affichage utilise deux décimales dans les PDF.

Les points viennent de `question.Tags.Point.PointValue` dans le JSON historique. Ils ne sont pas relus depuis `points`. **Les points historiques sont donc stables.**

## 13. Persistance des résultats et frontière produit

La DB persiste uniquement :

- les compteurs du job ;
- `status`, `status_pdf`, `completed_at` ;
- deux chemins/noms de PDF.

Elle ne persiste pas :

- les cases détectées ;
- les réponses interprétées ;
- le score/note par élève ;
- les réussites par compétence/thème ;
- le lien du job avec une génération ;
- le détail des copies non corrigées.

Ces valeurs vivent seulement en mémoire pendant le job puis dans :

- `corrected.pdf` : copies annotées ;
- `mark-table.pdf` : notes, moyenne, écart-type, médiane, compétences et diagnostics ;
- `corrected_NOT.pdf` éventuel : pages non traitées.

Les jobs success/failed et leurs workspaces sont purgés après sept jours lors du démarrage ou d'un nouvel upload. Après purge, aucun résultat n'est recalculable depuis `marking_jobs`.

Le produit conserve donc temporairement des PDF, pas un historique de correction structuré.

## 14. PDF corrigés, chemins et téléchargement

Les noms techniques finaux sont stables et indépendants des libellés vivants :

- `corrected.pdf` ;
- `mark-table.pdf` ;
- `corrected_NOT.pdf`.

Le workspace repose sur `job_id`, sous le username. Renommer l'Exam, la classe ou l'élève n'affecte pas les noms techniques. Les libellés inclus dans les PDF viennent du snapshot.

`ServePdfNamed` protège :

- composants de chemin ;
- extension `.pdf` ;
- parents symlinks ;
- fichier régulier ;
- symlink final ;
- TOCTOU par `Lstat`, ouverture, `Stat` et `os.SameFile`.

Le handler de téléchargement ne prouve toutefois pas en DB que `operation=marking-<id>` correspond à un job success possédé ; il s'appuie sur l'isolation du répertoire dérivé du username de session. Cela bloque l'inter-utilisateur mais constitue une défense métier incomplète.

Le PDF est stable seulement jusqu'au purge de rétention ou à une perte filesystem.

## 15. Résultats de classe

Il n'existe pas de page HTML de résultats structurés. `mark-table.pdf` constitue la synthèse :

- élève et score/total ;
- moyenne, écart-type et médiane ;
- réussite globale par compétence ;
- réussite compétence par thème ;
- QR non détectés ;
- copies non corrigées.

Les sources sont les `config.MarkExam` en mémoire. Les maps de compétences sont agrégées avant production Typst. Aucune donnée n'est relue à la demande après le job.

Si plusieurs évaluations de totaux différents sont mélangées, la synthèse devient incohérente : les statistiques portent sur des scores bruts et le dénominateur global est celui du premier résultat.

## 16. Lifecycle `marking_jobs`

### Schéma final

- PK autoincrémentée ;
- FK `user_id -> users(id) ON DELETE CASCADE` ;
- compteurs nullable/default 0 ;
- `status` et `status_pdf` dans `running|success|failed` ;
- chemins nullable ;
- `completed_at` ;
- aucune FK génération, aucun UNIQUE fonctionnel.

### Transitions

```text
CreateMarkingJob → running/running
  ├─ FailMarkingJob → failed/running + completed_at
  └─ CompleteMarkingJob → success/success + PDF names + completed_at
```

Les updates de progression et transitions terminales exigent `id`, `user_id` et `status='running'`. Les lignes affectées sont contrôlées. `CompleteMarkingJob` écrit atomiquement les deux statuts et les deux noms.

`status_pdf` n'a pas de lifecycle indépendant dans le code actuel.

### Failed, retry et purge

Le workspace est automatiquement supprimé par `ProcessMarking` sur tout échec. Le job failed reste en DB :

- sa consultation via le polling le supprime par GET ;
- sinon il est purgé après la rétention ;
- un retry est toujours possible car aucune unicité ne le bloque.

## 17. Recovery serveur

`RecoverRunningMarkingJobs` est appelé au démarrage avant l'ouverture du serveur :

- chaque workspace `running` est supprimé ;
- le job est atomiquement passé à `failed` avec `completed_at` ;
- les jobs `success` sont conservés ;
- les jobs déjà `failed` ne sont pas modifiés.

La fonction est testée et idempotente. Un workspace absent est accepté.

Limite : toute erreur filesystem/DB sur un seul job remonte à `main`, qui appelle `log.Fatal`; un workspace irrécupérable peut empêcher tout le serveur de démarrer.

## 18. Polling et fermeture navigateur

La page progression utilise un meta-refresh toutes les deux secondes. Le job utilise `appCtx`, pas le contexte de la requête : fermer la page n'arrête pas le traitement.

Le polling est néanmoins **destructif** pour `failed` : `ProgressMarkingHandler` supprime la ligne via `DeleteMarkingJob` avant la redirection métier. Le cleanup filesystem n'en dépend plus, mais la conservation du diagnostic DB dépend de la visite.

Pour `success/success`, le polling redirige vers la page finale. Les GET de progression sont sinon des lectures.

## 19. Concurrence et atomicité

Chaque job a un workspace distinct et deux sémaphores internes limitent à cinq workers pages et cinq workers copies. Les slices partagées sont protégées par mutex et les erreurs critiques par `sync.Once`. Le race detector passe.

Il n'existe aucune limite globale ou par utilisateur : plusieurs uploads peuvent lancer chacun leurs propres workers et processus externes. Le même scan, la même génération et la même copie peuvent être corrigés simultanément dans des jobs indépendants.

Les résultats d'une copie ne sont pas écrits en DB, donc il n'y a pas de conflit de score ; le coût CPU/mémoire/disque est toutefois multiplié.

La finalisation construit d'abord les PDF puis exécute `CompleteMarkingJob`. Si la finalisation DB échoue, le workspace est supprimé et le job tente de passer failed. Il n'existe pas de success DB avant les deux PDF principaux.

La création du job et le staging fichier ne sont pas une transaction commune, mais les chemins d'erreur avant/après création nettoient le staging ; le recovery couvre un arrêt brutal après insertion.

## 20. Erreurs, panic et cleanup

`ProcessMarking` récupère les panic, tente `FailMarkingJob` et supprime le workspace tant que le job n'est pas complété. `MarkingFailed` réessaie avec un contexte indépendant de trois secondes lorsque `appCtx` est annulé.

Les erreurs critiques suivantes font échouer le job : split, listing, update compteurs, workers pages critiques, absence totale de QR, absence totale de copie corrigée, Typst, merge, cleanup final requis et transition DB.

Les erreurs par copie (`MarkingStudentExam`) sont converties en `notMarkedExams` et n'échouent pas le job si au moins une autre copie réussit.

Une copie success ne laisse pas de score DB partiel. Des fichiers intermédiaires peuvent rester pendant un job running brutalement interrompu, puis sont supprimés au recovery.

## 21. Suppressions et nettoyage

| Suppression | Condition/acteur | Ownership |
| --- | --- | --- |
| staging upload | fin de goroutine | fichier OS interne |
| workspace failed | defer `ProcessMarking` | username + jobID validés |
| workspace running interrompu | recovery startup | join job/user |
| job failed | GET progression | `id + user_id` |
| jobs/workspaces terminaux expirés | purge 7 jours | join job/user |
| intermédiaires | pipeline | chemins internes au workspace |

Le purge retire d'abord le workspace puis la ligne DB. Une panne DB après suppression filesystem peut laisser une ligne success sans ses PDF. Une panne filesystem arrête le purge avant suppression DB.

`LeftPages` est une mutation filesystem effectuée depuis la page GET success : il convertit les PNG restants, fusionne puis supprime PNG/PDF intermédiaires. Deux consultations concurrentes peuvent entrer en compétition.

## 22. Sécurité fichiers

Points satisfaisants :

- username et operation validés comme composants ;
- refus des parents symlinks ;
- workspaces par job ;
- noms scannés générés côté serveur ;
- nom issu du QR ignoré ;
- commandes sans shell ;
- timeouts ;
- téléchargement régulier/sans symlink/avec `SameFile` ;
- staging par `CreateTemp`.

Points à renforcer :

- preuve DB avant téléchargement ;
- limite de pages/coût global ;
- génération des pages restantes hors GET ;
- test handler upload complet sur PDF hostile/oversize et cleanup multipart.

## 23. View-data, UX et navigation

`MarkingPageData` contient encore `ExtraData map[string]any` sur les trois pages :

- liste : bool + rows SQLC `GetExamsGeneratedSuccess` directement exposées ;
- progression : `JobID` sous forme de string, compteurs/statuts dynamiques ;
- success : trois URLs dans la map.

Il n'y a pas de slices parallèles importantes, mais le contrat n'est pas typé.

UX actuelle :

- liste/table non responsive ;
- évaluations success affichées sans véritable sélection ;
- bouton « Ajouter ! » ;
- statuts DB techniques visibles ;
- progression sans barre ni contexte d'évaluation ;
- success ouvre automatiquement deux ou trois popups via JavaScript ;
- aucun bouton explicite de téléchargement/retour ;
- wording ancien et erreurs génériques ;
- drag/drop JS dépendant du DOM même lorsque l'état vide ne rend pas les éléments, ce qui peut provoquer une erreur JS dans ce cas.

## 24. N+1 et performances

Aucun N+1 de liste/view n'a été trouvé : `GetExamsGeneratedSuccess` est une seule jointure et progression/success font des lectures constantes.

Le pipeline lit un snapshot par copie et un `PageContent` par page, puis met à jour les compteurs par page/copie. Cette complexité est intrinsèque au traitement actuel et n'est pas classée comme N+1 de présentation. Une lecture groupée pourrait réduire les allers-retours, mais elle n'est pas le premier risque.

Le vrai risque performance est l'absence d'admission globale et de limite de pages/jobs.

## 25. Couverture de tests

### Couverture à forte valeur existante

- calcul full/partial/incorrect ;
- validation stricte des vecteurs ;
- ordre des pages/copies mélangées ;
- noms PDF par `student_exam_id` ;
- transitions terminales/progression et rows affected ;
- failure/panic cleanup ;
- recovery startup ;
- purge et rétention ;
- traversal, symlinks, fichier régulier, `SameFile` ;
- PDF upload rewound ;
- génération/snapshot QR DB ;
- Typst escaping ;
- smoke tests de vrais scans une/deux/trois pages et batches complets lorsqu'une DB/PDF privés sont configurés.

### Tests privés désactivés dans la suite publique

Neuf tests de scans réels ont été constatés en `skip` faute de variables `LAZYMARKING_TEST_*`. C'est approprié pour ne pas exposer de données élèves, mais la CI publique ne valide donc pas l'algorithme bout en bout sur scans réels.

### Lacunes importantes

- aucune preuve qu'un job refuse QR de plusieurs générations ;
- aucun test de statut parent `running/failed` ;
- pas de test cross-user complet du POST QR jusqu'au résultat ;
- pas de tests explicites doublon/page manquante sur le job complet ;
- pas de persistance/récupération des scores car elle n'existe pas ;
- pas de test du seuil ambigu ;
- peu de tests handlers upload/progression/success ;
- pas de concurrence de plusieurs jobs ni de double GET `LeftPages` ;
- pas de test de perte d'image/template historique.

## 26. CI

`./scripts/check.sh` et `go test -race ./...` couvrent les packages Marking et passent localement. Résultats :

- `go mod verify` : succès ;
- gofmt : succès ;
- `go vet ./...` : succès ;
- `go test ./...` : succès ;
- `go build ./...` : succès ;
- `git diff --check` : succès ;
- `go test -race ./...` : succès, aucune course détectée.

La compilation exige CGO/OpenCV. Le workflow configure CGO mais n'installe pas explicitement OpenCV, Poppler ou Typst. Les tests unitaires capables de fonctionner sans corpus passent ; les vrais scénarios Marking nécessitent les outils système et des fixtures privées et sont donc sautés en CI publique.

## 27. Checklist future sur copie historique réelle

Sans utiliser ni modifier les données réelles dans cet audit, le contrôle futur devra :

1. copier `db_test_real.db` vers un fichier temporaire en lecture/écriture isolé ;
2. exécuter `PRAGMA foreign_key_check` sur la copie ;
3. vérifier les parents/status de chaque `student_exam_id` des scans ;
4. tester séparément les corpus une/deux/trois pages ;
5. tester un paquet réellement mélangé et son ordre ;
6. tester une copie QR illisible, une page manquante et une page dupliquée ;
7. tester des classes/générations différentes dans un même paquet et confirmer le futur refus ;
8. relever réponses détectées et scores attendus, sans publier les fichiers ;
9. vérifier les trois PDF finaux et leurs pages ;
10. redémarrer sur un job running synthétique dans la copie ;
11. vérifier cleanup et rétention sur répertoires temporaires dédiés ;
12. ne jamais ajouter DB, scans ou PDF élèves à la CI publique.

## 28. Constats classés

### P0 — aucun

Aucune fuite inter-utilisateur ou exécution de commande injectée n'a été démontrée.

### P1-1 — job non lié à une génération success

**Preuve :** `marking_jobs` ne porte aucune FK génération ; le formulaire ne sélectionne rien ; le POST crée un job avec `user_id` seulement ; les snapshots ne rejoignent pas le statut parent.

**Impact enseignant :** mélange possible de copies de plusieurs évaluations, synthèse/statistiques incohérentes, correction possible hors contrat success.

**Recommandation :** ajouter une identité de génération possédée/success au contrat du POST et du job ; valider chaque QR contre cette génération.

**Taille :** M (schéma/requêtes/handlers/tests).

### P1-2 — dépendances historiques de rendu non figées

**Preuve :** QCM/géométrie sont en JSON, mais `TypstWriter` relit `ref_qcm.txt` et les images de `assets/images`.

**Impact :** une copie historique peut devenir incorrigible ou être alignée sur une référence différente après évolution/suppression.

**Recommandation :** stabiliser l'artefact de référence nécessaire à la correction (pages de référence ou dépendances/version de rendu), sans relire la banque vivante.

**Taille :** L.

### P1-3 — résultats non persistés et purge automatique

**Preuve :** aucune table de score/réponse ; `MarkExam` reste en mémoire ; PDF/workspace et job sont supprimés après sept jours.

**Impact :** perte définitive des notes/copies corrigées si elles ne sont pas téléchargées/sauvegardées ; aucun historique structuré ni reprise.

**Recommandation :** définir le contrat durable puis persister job→génération, résultats par copie et identité des artefacts ; séparer rétention temporaire et historique.

**Taille :** L.

### P1-4 — aucune gestion des cases ambiguës

**Preuve :** moyenne ROI strictement comparée au seuil fixe 150 ; sortie uniquement 0/1.

**Impact :** une marque limite peut modifier silencieusement la note sans signalement enseignant.

**Recommandation :** définir une bande d'ambiguïté/confidence, conserver la mesure et prévoir validation/relecture avant finalisation.

**Taille :** M/L avec corpus de calibration.

### P2-1 — succès partiel et compteurs incohérents

**Preuve :** une copie réussie suffit au success ; QR illisibles/pages manquantes restent hors compteurs et sont reportés tardivement.

**Impact :** l'enseignant peut interpréter « success » comme correction complète.

**Recommandation :** état explicite `completed_with_issues` ou bilan typé complet avant terminalisation.

**Taille :** M.

### P2-2 — polling failed destructif

**Preuve :** `ProgressMarkingHandler` exécute `DeleteMarkingJob` sur GET failed.

**Impact :** diagnostic supprimé selon consultation, deuxième poll en 404, lifecycle dépendant de l'UI.

**Recommandation :** GET en lecture seule ; cleanup terminal automatique/idempotent séparé.

**Taille :** S/M.

### P2-3 — génération des pages restantes dans GET success

**Preuve :** `LeftPages` convertit/fusionne/supprime des fichiers depuis le handler GET.

**Impact :** race entre onglets, success déclaré avant tous les artefacts, résultat secondaire dépendant de la visite.

**Recommandation :** produire l'artefact dans le worker avant `CompleteMarkingJob`; GET uniquement résolve/sert.

**Taille :** M.

### P2-4 — admission de charge insuffisante

**Preuve :** 100 MiB par requête mais pas de limite pages/jobs ; chaque job crée jusqu'à dix workers internes et de nombreux processus externes.

**Impact :** saturation CPU/mémoire/disque par uploads concurrents ou PDF très complexe.

**Recommandation :** limite de pages, quota jobs running par user, limite globale/queue et timeout de job.

**Taille :** M.

### P2-5 — recovery peut bloquer le démarrage global

**Preuve :** première erreur cleanup/update interrompt la boucle ; `main` appelle `log.Fatal`.

**Impact :** un workspace problématique peut rendre l'application indisponible.

**Recommandation :** traiter tous les jobs, agréger/loguer les erreurs et définir une politique non fatale ou quarantinée.

**Taille :** S/M.

### P3-1 — view-data dynamique

`MarkingPageData.ExtraData`, IDs string et rows SQLC exposées rendent les templates fragiles. Refactor typé recommandé après les contrats métier.

### P3-2 — UX legacy

Popups automatiques, statuts techniques, table non responsive, wording et navigation implicite. À traiter seulement après lifecycle/persistance.

### P3-3 — schéma de job redondant

`status_pdf` n'a pas de transition indépendante ; compteurs nullable et absence de timestamps de début rendent le contrat moins clair. Simplifier lors de l'évolution du job.

## 29. Roadmap minimale recommandée

1. **Lier un job à une génération success unique** et valider tous les QR contre elle.
2. **Définir/persister le résultat durable** : réponses/mesures, score, statut par copie, artefacts et rétention.
3. **Stabiliser la référence historique** utilisée par l'homographie (template/images/pages).
4. **Introduire la gestion d'ambiguïté** et un workflow de validation enseignant.
5. **Refondre le lifecycle** : success partiel, failed non destructif, leftover produit par worker, recovery robuste.
6. **Borner la charge** et renforcer les tests concurrence/fichiers.
7. **Typer les view-data**, puis moderniser l'UX et supprimer les popups.

## 30. Verdict

**C — travail structurel significatif avant UX.**

L'isolation utilisateur, le calcul depuis les JSON historiques, l'ordre des pages, les points et la sécurité de base des workspaces sont de bonnes fondations. Mais le domaine doit d'abord acquérir une identité de correction explicite, une persistance durable, une référence historique complète et une gestion des cas ambigus/partiels. Une refonte UX maintenant masquerait des contrats métier encore insuffisants.
