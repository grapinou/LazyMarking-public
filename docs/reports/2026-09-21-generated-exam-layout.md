# P5 — Lisibilité des copies générées

Date : 21 septembre 2026.

## État trouvé lors de la reprise

La reprise a commencé par `git status --short`, `git diff --stat`, `git diff`, la lecture du fichier non suivi et la recherche des rapports. Contrairement à l’hypothèse de départ, **aucune implémentation ni aucun rapport P5 n’était présent**. Le dernier commit était `f342733` (P4). P1–P4 figuraient dans l’historique.

Les seules modifications locales étaient `README.md` (55 lignes de diff) et `reset.sh` non suivi. Elles ont été conservées ; `reset.sh` n’a pas été exécuté. Aucun commit, push ou nettoyage des modifications antérieures.

Le rapport de septembre sur la consigne générale de marquage existait (`2026-09-04-student-marking-instruction.md`) : il concerne une fonctionnalité antérieure, pas P5. Aucun rapport concurrent du 20 ou 21 septembre n’a été trouvé. Le présent document est donc le seul rapport P5.

Le premier `go test ./...`, avant modification, passait. Aucun test, champ ou migration P5 n’avait déjà été ajouté. Les tests dépendant de corpus privés étaient alors ignorés faute de variables ; le test historique d’une, deux et trois pages a ensuite été exécuté explicitement avec le corpus local.

## État initial et architecture historique

| Élément | Production / consommation |
|---|---|
| Portrait individualisé | `BuildQcmStudentCtx` → `TypstWriter` → `ref_qcm.txt` |
| Aperçu QCM / question / variante | Même writer portrait, types `PreviewQCM` / `PreviewQuestion` |
| 25 paires consigne / énoncé près des sauts de page | Aucune consigne courte isolée de son énoncé |
| Aperçu paysage | `TypstWriterLandscape` + `ref_qcm_landscape.txt`, A4 paysage à deux colonnes |
| Mini-test | `TypstLandscapeContent` pour chaque exemplaire, puis `TypstWriterLandscapeAllContent` |
| Titre individualisé | `exam.Name` → `QCM.Name` → variable Typst `exam` |
| Identité | `students.first_name`, `last_name` → `StudentQCM` → variable `student` |
| Classe | Nom chargé avec ownership → `StudentQCM.ClassCodes.Name` → `classCode` |
| Page | Compteur Typst dans le repère droit ; index de page également dans le QR |
| QR | Ajout après rendu PNG, par `QrCodeMaker` puis `PasteQrCodeOnPage` |
| Consigne générale | Texte de marquage constant dans le portrait élève et chaque mini-test |
| Énoncé et réponses | Question principale ou variante choisie, réponses mélangées, image éventuelle |
| Snapshot complet | JSON `config.QCM` dans `student_exam_content`, avec identité, contenus, tags, réponses et coordonnées |
| Snapshot par page | JSON `config.PageContent` dans `student_exam_page_content` : cercles de questions et réponses |
| Référence historique | PNG natif pré-QR, dimensions, DPI et SHA-256 persistés par page |
| Correction | `MarkingStudentExam` lit le snapshot et les références persistées ; homographie, détection et score |

La génération individualisée exporte Typst à 300 ppp, détecte les cercles sur les PNG natifs, persiste ces coordonnées et références, colle le QR et transforme les images en PDF. Le PDF distribué est donc rasterisé. Une extraction `pdftotext` porte sur le PDF Typst natif ; le PDF final avec QR se contrôle visuellement et par son pipeline de détection.

Les références PNG persistées priment toujours. Leur corruption n’autorise pas un repli silencieux. Les pages anciennes dont les cinq métadonnées de référence sont toutes NULL utilisent encore une recompilation du snapshot ; ce chemin devait impérativement conserver son ancien rendu.

Le mini-test reste un document paysage à nom manuscrit, sans QR, snapshot de correction ou correction automatique. Sa consigne provient de la banque au moment de produire le PDF. Ce fonctionnement antérieur n’a pas été transformé en génération individualisée persistée.

## Cause exacte des troncatures

Aucune troncature de nom ou titre n’a été trouvée dans le parcours SQL → builders Go → littéraux Typst. Les données complètes sont transmises, avec échappement des guillemets, antislashs et caractères spéciaux. La limitation des noms de fichiers ne concerne pas le texte imprimé.

L’ancien en-tête était une grille de trois colonnes égales. L’identification disposait d’environ 63 mm et utilisait une liste en 16 points gras, plus la consigne générale. **La croissance en hauteur du cartouche faisait remonter l’en-tête Typst au-dessus du bord supérieur**, car sa position dépendait de la hauteur totale et de la marge fixe de 5 cm. Des premières lignes sortaient physiquement de la page. Le problème a été reproduit lors des compilations : la chaîne Go restait entière, mais le début du titre manquait dans `pdftotext`.

Il existait aussi un problème distinct : les aperçus QCM ne renseignaient pas `QCM.Name`, et les mini-tests n’incluaient ni le titre ni la classe. Ces informations sont maintenant alimentées depuis les requêtes déjà protégées par ownership.

## Identification des copies

Le cartouche contient, dans cet ordre :

1. le titre complet, en gras, 11 points ;
2. le libellé discret « Élève », puis prénom et nom complets en gras, 11 points ;
3. « Classe », puis son nom complet, 10 points.

La consigne générale de marquage est placée sous le cadre, à 8 points. Les libellés d’identification sont à 8 points. Les espacements verticaux sont explicites, sans la grande liste initiale.

La grille réserve 105 points au QR et 120 points aux repères droits (numéros à deux chiffres compris) ; environ 110 mm restent à l’identification. Le cadre suit sa hauteur de contenu. Les noms composés, titres et classes reviennent à la ligne sans ellipses ni découpe Go.

Le cartouche est placé indépendamment de la hauteur de la grille des repères. Typst mesure son contenu : la marge supérieure reste à 5 cm pour le cas courant ; si nécessaire, elle augmente pour garder le début des questions sous l’identification. `header-ascent` compense exactement cet agrandissement : **les coordonnées du QR et des repères ne changent pas**. Les contrôles incluent un titre répété sur plusieurs lignes, un nom de famille composé très long et une classe longue.

Le cadre est répété sur toutes les pages. Aucun déplacement ou redimensionnement du QR n’a été introduit.

## Consignes : fonctionnement historique et solution retenue

Historiquement, le modèle n’avait que `questions.content` et `alt_questions.content` pour les énoncés, sans consigne commune. Inclure une instruction dans l’énoncé restait possible, mais ne la partageait pas entre variantes et ne distinguait pas sa présentation.

La solution retenue est une **consigne facultative de famille**, portée uniquement par la question principale : `questions.instruction`.

- **Consigne générale** : indique comment marquer les réponses au stylo ; reste indépendante du contenu pédagogique.
- **Consigne de famille** : indique l’action attendue de l’élève ; facultative, commune à la principale et à ses variantes.
- **Énoncé** : données et texte propres à la question sélectionnée ; demeure obligatoire et distinct.

Les formulaires de création et d’édition de la principale proposent « Consigne commune (facultative) » et expliquent son héritage. Création et édition utilisent les mêmes requêtes propriétaires et contrôles de classification que l’énoncé. La consigne peut être modifiée ou effacée. Son texte est échappé comme l’énoncé en Typst et par les templates HTML dans le formulaire.

Les quatre builders de lecture (principale/variante, requête HTTP/contexte de génération) renseignent `config.Question.Instruction`. Une variante lit la consigne de sa principale avec le `user_id` ; aucune colonne ni édition autonome n’est ajoutée aux variantes. Les variantes conservent leurs énoncés, réponses et images.

La consigne apparaît en gras juste avant l’énoncé, dans la même cellule de question. La consigne est attachée au début de l’énoncé, et le bloc de question au début des réponses ; les questions et tables restent paginables. Une consigne vide n’affiche ni libellé superflu ni espace réservé.

## Architecture Typst finale et pagination

- `ref_qcm.txt` : nouvel en-tête portrait mesuré ; marges et repères conservés dans le cas nominal.
- `typstQuestionContent.go` : rendu commun aux questions du portrait, de l’aperçu paysage et du mini-test. La petite factorisation remplace trois copies de ce seul bloc.
- `ref_qcm_landscape.txt` : base paysage inchangée. Titre et classe sont maintenant présents ; le champ manuscrit « Prénom + Nom » est affiché une fois par exemplaire, au lieu d’une fois par question dans l’aperçu paysage.
- Les énoncés disposent d’une colonne flexible ; les réponses de deux colonnes égales. Les images conservent leur proportion et leur pourcentage de largeur dans une boîte de hauteur de 5 cm.
- La séparation des mini-tests utilise un saut de colonne faible, sans page finale vide.
- Les aperçus QCM portrait/paysage incluent désormais la consigne générale, comme leurs rendus élèves correspondants. L’aperçu administratif d’une question isolée continue à la masquer.
- L’aperçu QCM garde l’ordre de référence décidé en P4 ; la génération conserve le tirage des variantes et l’ordre individualisé existants.

Aucune refonte des templates de mémorisation, du barème, de la correction, du CSS ou de la banque communautaire.

## Snapshots et anciennes générations

`config.Question.Instruction` est sérialisé dans le JSON de `student_exam_content` avec `omitempty`. L’absence de propriété dans un ancien JSON donne une chaîne vide. La consigne est figée avec **la variante effectivement sélectionnée** et ne nécessite aucune consultation de la banque en correction.

Le JSON des nouvelles générations contient aussi `layout_version: 1`. Ce marqueur ne modifie pas les snapshots anciens. Pour leur repli sans PNG, la version absente/0 appelle `typstWriterLegacy` et `ref_qcm_legacy.txt`, qui conservent le rendu pré-P5 (en-tête et corps). La version 1 utilise le rendu P5. Les références PNG persistées restent prioritaires pour les deux versions.

`student_exam_page_content` ne reçoit pas de texte pédagogique : il conserve sa responsabilité de coordonnées/référence par page. Les nouvelles coordonnées sont détectées sur les nouvelles pages ; les anciennes lignes ne sont jamais recalculées par P5.

Le test de bout en bout modifie ensuite les énoncés et consignes de la banque, relit le snapshot inchangé, recompile son ancien texte, compare les PNG reproduits aux références persistées et corrige les pages après cette modification. Le nouveau tirage de la même famille voit bien la nouvelle consigne, contrairement à l’ancienne copie.

Limite historique conservée : une très ancienne copie dépourvue de PNG natif nécessite toujours les fichiers d’images nommés par son snapshot pour sa recompilation. P5 préserve ce chemin ; il ne peut recréer un fichier historique déjà absent. Les copies avec références natives n’ont pas cette dépendance à la banque d’images pour la correction.

## Migration et compatibilité des environnements

Nouvelle migration **0045**, additive :

```sql
ALTER TABLE questions ADD COLUMN instruction TEXT NOT NULL DEFAULT '';
```

Le défaut vide rend les données existantes compatibles. Le Down retire uniquement cette colonne. Aucune migration historique modifiée. Le code sqlc est régénéré depuis `db/sqlc.yaml`.

Les bases source accessibles par les liens de `runtime/*/db/data/app.db` ont été ouvertes en lecture seule et copiées via l’API SQLite backup. Seules les copies, sous `runtime/diagnostics/p5/migrations/`, ont été migrées.

| Environnement | Version source | Version cible | Questions | Snapshots complets / pages | Résultat |
|---|---:|---:|---:|---:|---|
| smoke | 40 | 45 | 5 | 2 / 4 | OK |
| real | 41 | 45 | 10 | 60 / 180 | OK |
| 2026-2027 | 44 | 45 | 7 | 35 / 69 | OK |

Pour chacun : montée vers 44 si nécessaire, puis 45 → 44 → 45 ; toutes les anciennes colonnes de toutes les tables métier conservent leurs valeurs, snapshots compris. `integrity_check=ok`, aucune violation de clé étrangère. Les SHA-256 des trois fichiers sources sont inchangés. Script et résultats détaillés : `runtime/diagnostics/p5/check-migration.py` et `migrations/results.json`.

Le rejeu intégral des migrations sur base neuve (`scripts/check-migrations.sh`) atteint également 45.

## QR et correction

Les fonctions de génération/collage/détection QR, détection des cercles et homographie n’ont pas été modifiées.

Contrats vérifiés sur de vraies pages à 300 ppp :

- A4 portrait : 2480 × 3508 pixels ;
- QR source : 425 pixels, recadrage de 25 pixels par bord, collage en (127, 30), paramètres inchangés ;
- carré Typst de réservation : 95 points ;
- zones QR, repères droits et pied de page : comparaison pixel par pixel pré-P5/P5 sans différence pour les numéros 1 et 10, y compris avec un titre assez long pour augmenter la marge du corps ;
- QR ajouté puis décodé avec les mêmes identifiant de copie et numéro de page ;
- détection des cercles et coordonnées persistées validées par la génération réelle ;
- copie synthétique de 3 pages, 6 questions, 48 réponses dont une variante illustrée : génération, snapshots, références, QR et correction réussis après modification de la banque ;
- copies historiques réelles d’une, deux et trois pages du corpus `testdata/problematic` : correction réussie sur copie de la base, sans modification du corpus source.

## Tests ajoutés ou adaptés

- `questionInstructionMigration_test.go` : migration additive, défaut vide, conservation du snapshot et Down.
- `questions/instruction_test.go` : création, édition, effacement, caractères spéciaux, refus de modification inter-utilisateurs et de création avec classification étrangère.
- `generatedLayout_test.go` : PDF réels courts/longs/très longs, texte complet, position des identités, plusieurs pages, énoncé dépassant une page, paysage, mini-tests, géométrie et QR réels.
- `generatedLayoutPipeline_test.go` : schéma réel migré, question principale, variante illustrée, ownership, génération de 3 pages, snapshot, modification ultérieure de la banque, reproduction des références, correction et ancien snapshot sans les nouveaux champs.
- Tests d’échappement Typst étendus aux consignes ; tests de fixtures SQL adaptés au schéma courant, sans changer leurs assertions métier.

Les tests PDF s’exécutent lorsque Typst et Poppler sont disponibles ; le test complet de génération utilise aussi Goose. Versions utilisées : Typst 0.15.1 et sqlc 1.31.1. Ces outils sont présents ici et les tests ont effectivement été exécutés, sans skip de ces nouveaux cas. Les tests privés non configurés d’autres fonctionnalités conservent leurs conditions habituelles de skip.

## Tests PDF et validations exécutées

Les artefacts de contrôle restent hors Git sous `runtime/diagnostics/p5/`. `LAZYMARKING_LAYOUT_ARTIFACTS` permet de conserver un sous-dossier distinct par exécution ; les tests ordinaires nettoient leurs propres espaces temporaires.

| Contrôle | Résultat |
|---|---|
| Nom `Jean-Baptiste Dupont-Martin`, titre complet et classe longue | PDF compilé ; texte complet extrait |
| Titre très long, nom composé étendu, classe très longue | PDF compilé ; retours à la ligne ; corps décalé sans déplacement du QR |
| Nom/titre courts, consigne absente | Une page, sans emplacement de consigne vide |
| 12 questions longues avec consignes longues et nombreuses réponses | 6 pages ; toutes les questions présentes |
| Énoncé de 90 phrases, supérieur à une page | Compilation ; 90 occurrences et fin d’énoncé présentes |
| 25 paires consigne / énoncé près des sauts de page | Aucune consigne courte isolée de son énoncé |
| Aperçu paysage | Compilation, titre et consigne présents |
| Deux mini-tests courts | Une page paysage, deux consignes, aucun feuillet final vide |
| Variante illustrée et copie finale individualisée | 3 pages générées et corrigées |
| Ancien snapshot sans consigne ni version | Compilation par le renderer conservé |
| `pdftotext` | Utilisé en mode raw pour éviter qu’un numéro de page s’intercale artificiellement dans un titre extrait |
| `go test ./...` | Réussi |
| Suites ciblées génération, examens, QCM, questions, templates et tools | Réussies |
| Test historique réel 1 / 2 / 3 pages | Réussi sur les trois cas |
| `sqlc generate -f db/sqlc.yaml` | Régénération stable |
| `git diff --check` | Réussi |

Commandes finales (journaux dans `runtime/diagnostics/p5/`) :

```sh
go test ./...
go test -count=1 ./internal/handlers/generateExams ./internal/handlers/exams ./internal/handlers/qcm ./internal/handlers/qcmPreview ./internal/handlers/qcmQuestions ./internal/handlers/questions ./internal/handlers/tools ./internal/templates/... ./internal/db
sqlc generate -f db/sqlc.yaml
git diff --check
```

Le test `TestRealStudentExamMarkingSmoke` a également été exécuté explicitement avec les trois PDF de `testdata/problematic`, une copie de sa base et les identifiants de référence configurés. Les trois sous-tests passent.

L’inspection visuelle a également révélé puis fait corriger un chevauchement dans le cas extrême : la hauteur allouée au cartouche est désormais celle mesurée, même lorsqu’elle dépasse la réservation QR. Une assertion de coordonnées vérifie que la consigne générale commence sous la dernière ligne d’identité.

L’inspection visuelle complète les assertions textuelles et géométriques : cartouche, consigne, premiers/derniers blocs de questions, variantes avec image, pagination et QR. Les PDF natifs et rasterisés sont distingués ; l’extraction de texte n’est pas présentée comme une preuve visuelle du PDF raster final.

## Points restant ouverts

Aucun blocage P5 identifié sur les cas testés. La dépendance historique aux images pour les seules copies sans référence PNG est rappelée ci-dessus ; elle précède P5. La reconnaissance de scans dégradés conserve les limites du pipeline existant, sans changement de seuils ou de politique de correction.
