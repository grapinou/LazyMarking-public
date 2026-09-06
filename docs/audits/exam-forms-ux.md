# Refonte UX des formulaires Exams

## Formulaire Create

Le formulaire Create a été refait sous forme de carte Bootstrap de largeur raisonnable. Il présente clairement une évaluation comme l'association d'un nom, d'un QCM, d'une classe, d'une année et d'une période. L'ordre des champs est : Nom de l'évaluation, QCM, Classe, Année, Période.

Les noms POST existants (`exam`, `qcm_id`, `class_code_id`, `year_id`, `period_id`) sont inchangés. L'action principale est « Créer l'évaluation » et l'action secondaire « Annuler » utilise `CancelURL` vers `/dashboard/exams`.

## Prérequis et collections vides

Les collections existantes de `ExamFormData` sont utilisées sans nouvelle requête ni nouveau booléen. Si une collection obligatoire est vide, un message ciblé explique le prérequis manquant : QCM, classe, année ou période. Les routes déjà présentes sont utilisées pour QCM, Années et Périodes. `ExamPageData` ne porte pas naturellement la route de gestion des classes ; le message Classe renvoie donc verbalement vers la gestion des élèves sans fabriquer d'URL.

Le bouton de création est dérivé directement de la présence des quatre collections dans le template et n'est pas rendu si une FK obligatoire ne peut pas être choisie. Le backend reste l'autorité. Les QCM sans question et classes sans élève restent sélectionnables comme brouillons : aucune règle métier n'a été déplacée dans Create.

## Formulaire Edit

Le formulaire Edit a été refait avec la même structure et le titre « Modifier l'évaluation ». `ExamContext` affiche le nom courant et fournit l'ID `int64` au champ caché. `ExamFormData` fournit le nom et les sélections exactes de QCM, classe, année et période.

Les actions sont « Enregistrer » et « Annuler ». Annuler retourne à la liste Évaluations via `CancelURL`. L'interdiction d'accès pour un Exam généré reste assurée par le handler existant avant tout rendu.

## Formulaire Delete

Le formulaire Delete affiche « Supprimer l'évaluation », le nom provenant de `ExamContext`, la question « Voulez-vous supprimer cette évaluation ? » et précise qu'une évaluation déjà générée ne peut pas être supprimée. Il ne prétend pas que les copies seront supprimées.

Les actions sont « Supprimer » et « Annuler ». Le POST et le champ `exam_id` sont inchangés ; Annuler retourne à `/dashboard/exams`. Un brouillon reste accessible et supprimable, tandis que le POST d'un Exam généré reste refusé par les protections existantes.

## Nom et JavaScript

L'ancien JavaScript `removeForbiddenCharacters` et les attributs `oninput` ont été supprimés de Create et Edit. Aucun filtrage frontend des apostrophes ou guillemets n'est réintroduit. Le backend conserve TrimSpace, le refus du blanc, la conservation des apostrophes/guillemets, la classification UNIQUE et HTTP 500 pour les vraies erreurs DB.

## Wording, responsive et accessibilité

Les anciens textes « Modifier la question », « Es-tu sur » et « C'est mon dernier mot » ont été retirés. Tout le wording visible est en français et utilise Évaluation, QCM, Classe, Année, Période, Créer, Enregistrer, Supprimer et Annuler.

Les formulaires utilisent des champs empilés, une grille responsive légère pour Année/Période, des actions `flex-wrap`, des labels reliés par `for`/`id` et des protections pour les contenus longs. Aucun CSS complexe n'a été ajouté.

## Données de vue et périmètre

`ExamFormData` et `ExamContext` sont conservés sans second modèle de formulaire. Aucun ID string et aucun `ExtraData` ne sont réintroduits dans le CRUD Exams. `GenerateExamPageData` est hors périmètre et inchangé.

Aucune règle métier, requête SQL, migration, génération, logique Marking, cleanup, preflight ou individualisation n'a été modifiée. Aucun `sqlc generate` n'était nécessaire.

## Tests

Les tests existants couvrent toujours :

- les quatre collections Create et l'absence de sélection artificielle ;
- `ExamContext`, le nom et les quatre sélections Edit ;
- `CancelURL` pour Create, Edit et Delete ;
- TrimSpace et la conservation des apostrophes/guillemets en Create/Edit ;
- le refus d'Edit pour un Exam généré ;
- le refus du POST Delete pour un Exam généré ;
- les comportements POST Create/Edit/Delete existants.

Les tests ajoutés couvrent les quatre variantes de collection obligatoire vide, leur message pédagogique, l'absence du bouton Create dans cet état et l'accès au formulaire Delete d'un brouillon. Aucun snapshot HTML ni aucune classe CSS ne sont testés.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès ;
- `sqlc generate` : non exécuté, aucun SQL modifié dans ce jalon.

## Fichiers modifiés par ce jalon

- `internal/templates/exams/add_form_exam.html` ;
- `internal/templates/exams/edit_form_exam.html` ;
- `internal/templates/exams/delete_form_exam.html` ;
- `internal/handlers/exams/handlers_test.go` ;
- `docs/audits/exam-forms-ux.md`.
