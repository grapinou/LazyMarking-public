# UX de progression et de succès de génération Exam

## Progression

La page de progression a été refaite sous forme de carte Bootstrap centrale et responsive. Le titre visible est « Génération des copies » et le texte explique que les copies individualisées sont en cours de préparation, sans vocabulaire de worker, goroutine ou Typst.

Le compteur affiche directement `ProcessedStudents` et `TotalStudents` sous la forme « X copies préparées sur Y ». Les valeurs restent brutes dans le view-model.

Une barre de progression Bootstrap a été ajoutée. La méthode de présentation pure `ExamGenerationProgress.Percentage` retourne un entier borné entre 0 et 100. Elle retourne 0 si le total est nul ou négatif et évite donc toute division par zéro ; elle ne modifie aucune donnée métier.

La barre expose `role="progressbar"`, un libellé accessible et des valeurs `aria-valuenow`, `aria-valuemin` et `aria-valuemax` cohérentes. Si le total est nul, la barre reste neutre et aucun pourcentage textuel trompeur n'est affiché.

## Polling et actions running

Le polling reste assuré par la même balise meta refresh toutes les deux secondes. Elle continue d'utiliser `ProgressURL`, qui conserve `exam_generated_id`, `exam_id` et `generation_started`. Aucun JavaScript de polling n'a été ajouté.

La page précise qu'elle se met à jour automatiquement et que l'enseignant peut la quitter, le job étant déjà exécuté en arrière-plan par le serveur indépendamment du navigateur.

La seule action proposée est « Retour aux évaluations ». `ExamsURL` a été ajouté mécaniquement au view-model de progression depuis la route Dashboard existante, sans requête ni nouveau contrat. Aucune action Modifier, Supprimer ou Annuler la génération n'est exposée.

## Succès

La page de succès a été refaite sous forme de carte responsive avec le titre « Copies générées ». Elle affiche les valeurs déjà disponibles `ExamName` et `ClassName`, sans charger de donnée supplémentaire et sans montrer l'ID de génération comme information visible.

L'action principale « Ouvrir le PDF des copies » utilise directement `Success.CopiesURL`. L'action secondaire « Retour aux évaluations » utilise `Success.ExamsURL`. Aucune URL n'est reconstruite dans le template.

## Popup et JavaScript

L'ancien script `window.onload` ouvrait automatiquement une popup vers le PDF et obligeait à expliquer les blocages de popup du navigateur. Il a été supprimé au profit du lien explicite vers le PDF, ouvert à la demande dans un nouvel onglet avec `rel="noopener"`.

Aucun JavaScript ne reste dans les templates progression et succès. Le polling continue exclusivement avec la balise meta refresh.

## Failed et périmètre

Le workflow failed est inchangé : aucune page failed n'a été créée, aucun DELETE ni cleanup filesystem n'a été ajouté au GET, et les redirections métier existantes restent actives après cleanup ou face à une ligne failed transitoire.

La mini génération n'a pas été touchée. Le lifecycle, la génération interne, l'individualisation et Marking n'ont pas été modifiés. Aucun SQL, migration ou fichier sqlc n'a changé.

## Tests

Les tests ajoutés ou adaptés couvrent :

- calcul du pourcentage avec total nul, données négatives, progression normale, fin et dépassement ;
- présence du compteur X/Y dans le rendu running ;
- utilisation de `ProgressURL` ;
- texte d'actualisation automatique ;
- retour vers Exams ;
- absence d'actions impossibles pendant running ;
- affichage de `ExamName` et `ClassName` au succès ;
- présence de `CopiesURL` et `ExamsURL` ;
- absence de popup automatique et de libellé d'ID technique.

Les tests existants failed, absent et étranger restent inchangés et passants.

## Validations

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès ;
- `sqlc generate` : non exécuté, aucun SQL modifié.

## Fichiers modifiés par ce jalon

- `internal/templates/generateExam/processing_students.html` ;
- `internal/templates/generateExam/success_processing.html` ;
- `internal/templates/data/generateExam.go` ;
- `internal/templates/data/generateExam_test.go` ;
- `internal/handlers/generateExams/handlers.go` ;
- `internal/handlers/generateExams/viewData_test.go` ;
- `docs/audits/exam-generation-ux.md`.
