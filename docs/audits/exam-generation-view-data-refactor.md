# Refactor des données de vue de génération Exam

## Usages ExtraData avant le refactor

La page de progression recevait quatre clés :

- `Status`, produit par `GetExamStatus` et affiché dans le template ;
- `Processed`, produit par `GetExamGeneratedProgress` et affiché ;
- `Total`, produit par la même requête et affiché ;
- `ExamGeneratedID`, construit comme string depuis le paramètre HTTP mais non consommé par le template.

La page de succès recevait deux clés :

- `Status`, fixé à `success` et affiché ;
- `PdfURL`, construit depuis l'ID de génération et les noms vivants immuables de l'Exam et de la classe, puis utilisé par l'ouverture du PDF.

`GenerateExamPageData.Routes` est nécessaire au layout Dashboard. `GenerateExamRoutes` était porté par le PageData sans être consommé directement par les templates de génération.

## Types et champs créés

`GenerateExamPageData` contient désormais uniquement des champs typés : `PageTitle`, `Routes`, `Context`, `Progress` et `Success`.

`ExamGenerationContext` regroupe l'identité disponible du workflow : `GenerationID int64`, puis `ExamName` et `ClassName` lorsqu'ils sont déjà lus sur le chemin success.

`ExamGenerationProgress` contient `Status`, `ProcessedStudents`, `TotalStudents` et `ProgressURL`.

`ExamGenerationSuccessData` contient `Status`, `CopiesURL` et `ExamsURL`.

L'ancienne valeur morte `ExamGeneratedID` string a été supprimée. L'ID réellement conservé dans les données de vue reste un `int64`.

## Progression typée

Le handler running conserve ses deux lectures existantes : statut, puis compteurs. Il construit un view-model explicite sans requête de nom supplémentaire. Le nom de l'Exam n'est donc pas affiché pendant le polling, afin de ne pas ajouter une lecture à chaque refresh.

Le polling reste un refresh toutes les deux secondes. Son URL est désormais explicite dans `ProgressURL` et consommée par la balise meta ; elle conserve les paramètres `exam_generated_id`, `exam_id` et `generation_started` du workflow.

## Succès typé

Le chemin success conserve la lecture existante `GetExamNameAndClassCodeName`. Les noms sont placés dans le contexte et affichés. L'immutabilité d'un Exam généré garantit désormais leur stabilité applicative ; aucun snapshot ou SQL supplémentaire n'est créé.

`CopiesURL` pointe vers la route PDF existante avec l'opération et le nom de fichier calculés côté Go. `ExamsURL` fournit un retour explicite vers la liste des évaluations. Aucune route nouvelle n'a été ajoutée.

## URLs centralisées

Deux helpers purs et ciblés construisent les URLs : progression et copies PDF. Les templates ne concatènent aucun ID ni paramètre. Le redirect initial après création de la génération réutilise le helper de progression ; cette adaptation est mécanique et ne modifie aucune transition du lifecycle.

## Failed après cleanup

Le chemin failed ne rend pas de `GenerateExamPageData`. Une ligne encore `failed` et une génération déjà supprimée continuent toutes deux d'utiliser la redirection métier existante. Le signal `generation_started=1`, la validation ownership de l'Exam et les contrats 400/404 existants sont conservés.

Aucun cleanup n'a été ajouté au GET et aucune représentation artificielle d'un état DB supprimé n'a été créée dans le view-model.

## ExtraData, requêtes et périmètre

`ExtraData` a été supprimé de `GenerateExamPageData` et il ne reste aucun usage fonctionnel dans les handlers ou templates generateExam. Les autres PageData du dépôt ne sont pas concernés.

Aucune requête n'a été ajoutée et aucun N+1 n'a été introduit. Running conserve les lectures du statut et des compteurs ; success conserve les lectures du statut et des noms.

Le lifecycle, la génération interne, les workers, l'individualisation et Marking n'ont pas été modifiés. Aucun SQL, aucune migration et aucun fichier sqlc n'ont changé.

## Tests

Les tests ajoutés couvrent :

- contexte running avec `GenerationID int64` ;
- statut, nombre traité et total ;
- URL de progression et conservation des paramètres du cleanup ;
- plusieurs couples génération/Exam sans mélange d'IDs ;
- contexte success avec noms Exam/classe ;
- URL PDF, opération et nom de fichier ;
- URL de retour vers Exams ;
- smoke tests de rendu des templates running et success.

Les tests existants conservent la couverture de l'échec nettoyé, de la ligne failed transitoire non supprimée par GET, des accès inconnus sans signal, de l'ownership et des erreurs HTTP.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès ;
- `sqlc generate` : non exécuté, aucun SQL modifié.

## Fichiers modifiés par ce jalon

- `internal/templates/data/generateExam.go` ;
- `internal/handlers/generateExams/handlers.go` ;
- `internal/handlers/generateExams/viewData_test.go` ;
- `internal/templates/generateExam/processing_students.html` ;
- `internal/templates/generateExam/success_processing.html` ;
- `docs/audits/exam-generation-view-data-refactor.md`.
