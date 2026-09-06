# Refactor des données de vue du CRUD Exams

## Types créés

`ExamListItem` regroupe désormais, pour une même évaluation, son ID, son nom, les noms du QCM, de la classe, de l'année et de la période, ainsi que les quatre actions réellement présentes : modification, suppression, génération complète et mini test.

`ExamContext` contient l'ID `int64` et le nom d'un Exam précis. Il est utilisé naturellement par les pages Edit et Delete.

`ExamFormData` regroupe les quatre collections ownership-aware réellement retournées par sqlc : `[]db.GetAllQCMRow`, `[]db.ClassCode`, `[]db.Year` et `[]db.Period`. Il contient aussi le nom et les quatre IDs `int64` sélectionnés.

## Liste

Le handler de `/dashboard/exams` continue d'appeler une seule fois `GetExamsAllInfos`. Un builder pur transforme chaque ligne jointe en un `ExamListItem`, avec les données et URLs du même Exam. Les anciennes slices parallèles `Exams` et `Action` ont été supprimées.

Une liste SQL vide devient une slice `Items` vide et non nil. Aucun lookup par Exam et aucun N+1 n'ont été introduits.

La requête actuelle ne fournit aucun état de génération. Aucun statut n'a donc été inventé et aucune requête supplémentaire n'a été ajoutée. Une distinction « jamais généré / généré » demanderait une évolution SQL ultérieure si l'UX en a besoin.

## Formulaire Create

Le GET Create construit un `ExamFormData` avec les quatre collections existantes. Le nom et les sélections restent à leur valeur zéro : aucune sélection artificielle n'est créée. `CancelURL` pointe vers la liste Exams.

Les quatre requêtes ownership-aware restent inchangées. Les règles autorisant un QCM vide ou une classe vide comme brouillon et le POST Create restent inchangés.

## Formulaire Edit

Le GET Edit construit un `ExamContext` avec l'ID et le nom stockés, ainsi qu'un `ExamFormData` contenant le nom actuel et les IDs courants de QCM, classe, année et période. `CancelURL` pointe vers la liste Exams.

La vérification `ExamHasGeneration` reste placée avant le chargement et le rendu du formulaire : un Exam généré produit toujours la redirection métier et aucun formulaire n'est rendu. Les contrats absent/étranger en 404 restent inchangés.

## Formulaire Delete

Le GET Delete fournit maintenant un `ExamContext` et `CancelURL`. Le template utilise l'ID `int64` et le nom de ce contexte. Le POST Delete et ses règles métier n'ont pas été modifiés.

## ExtraData et templates

`ExamPageData.ExtraData` et `ExamActionURLs` ont été supprimés. Il ne reste aucun usage fonctionnel d'`ExtraData` dans le CRUD Exams. Les quatre templates CRUD ont seulement été adaptés aux nouveaux champs typés ; aucune refonte visuelle ou suppression du JavaScript historique n'a été réalisée.

`GenerateExamPageData` et son `ExtraData` n'ont pas été touchés.

## URLs

Un helper pur ciblé assemble les URLs existantes avec un `exam_id` décimal. Aucune route fictive et aucun routeur générique n'ont été ajoutés.

## Règles métier et SQL

Aucune règle métier n'a été modifiée. `CreateExam`, `UpdateExam`, `DeleteExam`, `ExamHasGeneration`, l'immutabilité, le cleanup failed, le preflight, la génération, l'individualisation et Marking sont inchangés.

Aucun SQL ni aucune migration n'ont été modifiés. `sqlc generate` n'était donc pas requis.

## Tests

Les tests de données ajoutés couvrent :

- plusieurs lignes transformées en plusieurs `ExamListItem` ;
- association correcte ID/nom/QCM/classe/année/période ;
- association des quatre URLs au bon item ;
- slice de liste vide propre ;
- quatre collections du formulaire Create ;
- absence de sélection artificielle en Create ;
- `ExamContext`, nom et quatre sélections courantes en Edit ;
- `ExamContext` en Delete ;
- `CancelURL` pour Create, Edit et Delete.

La construction des items est pure et n'effectue aucune requête ; le handler conserve l'unique appel joint à `GetExamsAllInfos`. Les tests HTTP existants de l'immutabilité, des Exams absents/étrangers et du CRUD restent passants.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès ;
- `sqlc generate` : non exécuté, aucun SQL modifié.

## Fichiers modifiés par ce jalon

- `internal/templates/data/exam.go` ;
- `internal/handlers/exams/handlers.go` ;
- `internal/handlers/exams/handlers_test.go` ;
- `internal/templates/exams/table_exams.html` ;
- `internal/templates/exams/add_form_exam.html` ;
- `internal/templates/exams/edit_form_exam.html` ;
- `internal/templates/exams/delete_form_exam.html` ;
- `docs/audits/exam-view-data-refactor.md`.
