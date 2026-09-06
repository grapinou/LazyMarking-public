# Refonte de la liste des évaluations

## Requête et état de génération

La requête `GetExamsAllInfos` a été modifiée. Sa jointure existante Exam + QCM + classe + année + période est conservée et complétée par un `LEFT JOIN exams_generated`. Elle expose deux valeurs nullables : `generation_id` et `generation_status`.

L'absence de ligne liée représente un brouillon. Les valeurs `running`, `success` et l'éventuel état résiduel `failed` sont transmises sans construire de logique métier complexe en SQL.

Le JOIN porte à la fois sur `exams_generated.exam_id = exams.id` et `exams_generated.user_id = :user_id`. Les filtres ownership existants sur l'Exam, le QCM, la classe, l'année et la période sont inchangés. La contrainte unique `(exam_id, user_id)` garantit au plus une génération jointe par Exam.

## Choix de generation_id

L'ID de génération est exposé parce qu'il permet deux actions existantes et utiles sans lookup secondaire :

- `running` : ouvrir la route existante de progression ;
- `success` : ouvrir cette même route, qui rend ensuite la page de succès donnant accès au PDF.

L'URL existante nécessite `exam_generated_id`; elle reçoit aussi `exam_id` et `generation_started=1`, conformément au flux de génération actuel. Aucune route n'a été créée.

## ExamListItem

`ExamListItem` contient maintenant un `ExamGenerationStatus` ciblé avec quatre valeurs : `draft`, `running`, `success` et `failed`. Il ne repose pas sur une collection de booléens potentiellement contradictoires.

Les URLs restent portées par leur propre item et sont exposées uniquement lorsque l'action est cohérente avec l'état :

- Draft : génération complète, mini test, modification et suppression ;
- Running : progression seulement ;
- Success : accès aux copies via la page de succès seulement ;
- Failed résiduel : aucune action, avec indication que le nettoyage automatique est en cours.

Les URLs de modification, suppression et génération restent vides pour running, success et failed. Les templates ne fabriquent aucune URL et aucune slice parallèle n'a été réintroduite.

## Présentation

La table multicolonne a été remplacée par une grille responsive de cartes Bootstrap. Le nom est l'information principale ; QCM, classe, année et période sont présentés comme métadonnées secondaires. Un badge distingue Brouillon, Génération en cours, Générée et Échec de génération.

Les actions sont textuelles. Pour un brouillon, « Générer les copies » est l'action principale ; « Mini test », « Modifier » et « Supprimer » sont secondaires. Running propose « Voir la progression ». Success propose « Voir les copies ». Aucun bouton impossible de mutation ou de nouvelle génération n'est présenté pour running ou success.

L'état vide explique qu'aucune évaluation n'existe encore et propose « Créer une évaluation ». Les accès Années et Périodes sont conservés comme navigation secondaire de configuration. Aucun accès Classes n'existait sur cette page avant le jalon ; aucun accès fictif n'a été ajouté.

## N+1 et règles métier

Aucun N+1 n'a été introduit : la liste exécute toujours un unique `GetExamsAllInfos`. L'ID et le statut de génération proviennent du même `LEFT JOIN`.

Aucune règle métier n'a été modifiée. `CreateExam`, `UpdateExam`, `DeleteExam`, `ExamHasGeneration`, le cleanup failed, le preflight, la génération, l'individualisation et Marking sont inchangés.

## Tests

Les tests SQL couvrent :

- brouillon sans génération ;
- running ;
- success ;
- failed résiduel ;
- génération étrangère sans influence sur un Exam possédé ;
- une seule ligne retournée par Exam.

Les tests du view-model couvrent :

- actions complètes d'un brouillon ;
- progression seule pour running ;
- accès aux copies seul pour success ;
- aucune action pour failed ;
- absence des actions interdites ;
- association correcte des métadonnées et URLs entre plusieurs Exams ;
- liste vide propre ;
- rendu du template avec les données typées.

Aucune classe Bootstrap n'est testée.

## Génération sqlc et validations

- `sqlc generate -f db/sqlc.yaml` : exécuté avec succès ;
- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `db/query/exams.sql` ;
- `internal/db/exams.sql.go` généré par sqlc ;
- `internal/db/examRelationshipIntegrity_test.go` ;
- `internal/templates/data/exam.go` ;
- `internal/handlers/exams/handlers.go` ;
- `internal/handlers/exams/handlers_test.go` ;
- `internal/templates/exams/table_exams.html` ;
- `docs/audits/exam-list-ux.md`.
