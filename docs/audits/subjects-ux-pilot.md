# Pilote UX — Matières

## Résumé

Le référentiel Matières utilise désormais un motif entièrement orienté enseignant : une liste responsive d'unités pédagogiques, des actions textuelles, des formulaires centrés et une confirmation destructive explicite. Aucun comportement métier, SQL ou FK n'a été modifié.

## Liste

La page « Matières » présente :

- une courte explication du rôle du référentiel ;
- l'action principale « Ajouter une matière » ;
- une carte responsive par matière ;
- les actions textuelles « Modifier » et « Supprimer » ;
- le retour « Retour à la banque de questions ».

L'ancienne table, la colonne `Edit/Sup`, les boutons icon-only et les tableaux parallèles ont été supprimés.

## View-data typé

`SubjectListItem` contient l'ID `int64`, le nom et les URLs Edit/Delete correspondant à la même matière. `SubjectPageData.SubjectItems` remplace les anciennes clés `Subjects` et `Action`.

`SubjectContext` contient l'ID `int64` et le nom de la matière pour Edit et Delete. `CancelURL` porte explicitement la destination de retour vers la liste Matières.

Le petit helper pur `SubjectURL` construit les URLs portant `subject_id`. Il reste propre au domaine Subjects ; aucune abstraction générique de référentiel n'a été créée.

Il ne reste aucun usage d'`ExtraData` dans `SubjectPageData`, les handlers ou les templates Subjects. Les anciennes clés `NoSubject`, `Subjects`, `Action`, `SubjectID` et `Subject` ont été retirées.

## État vide

Une slice vide non-nil représente l'absence de matières. Le template affiche une carte pédagogique indiquant qu'aucune matière n'a encore été créée et propose directement « Ajouter une matière ». Aucune table vide n'est rendue.

## Ajouter

Le formulaire « Ajouter une matière » comporte uniquement le champ « Nom de la matière », avec un exemple Physique, puis les actions « Ajouter » et « Annuler ». Annuler revient à la liste Matières.

Le JavaScript qui supprimait les guillemets a été retiré. Aucune validation métier n'a été déplacée vers le navigateur.

## Modifier

Le formulaire « Modifier la matière » reçoit un `SubjectContext` ownership-validé, préremplit le nom courant et conserve exactement `subject_id` et `new_subject`. Les actions sont « Enregistrer » et « Annuler », ce dernier revenant à la liste Matières.

## Supprimer

La confirmation « Supprimer la matière » affiche clairement le nom concerné et demande : « Voulez-vous supprimer cette matière ? ». Elle précise qu'une matière utilisée par une question ne peut pas être supprimée.

Le formulaire conserve POST et `subject_id`. Le bouton « Supprimer » est destructif et « Annuler » revient à la liste Matières. La FK et le handler sécurisé restent seuls responsables de l'autorisation réelle de suppression.

## Apostrophes et guillemets

Les handlers continuent à appliquer `TrimSpace`, sans filtrage de caractères. Les tests prouvent que l'ajout et la modification conservent exactement des noms comme `L'étude "physique"` et `L'étude "des forces"`.

## Responsive et accessibilité

Le pilote utilise uniquement Bootstrap : cartes empilées, `flex-wrap`, largeurs de formulaire raisonnables et `text-break`. Tous les champs ont un couple `for`/`id`, les actions sont textuelles et les icônes complémentaires sont décoratives avec `aria-hidden="true"`.

## Tests ajoutés ou adaptés

- deux matières produisent deux `SubjectListItem` correctement associés ;
- ID, nom, EditURL et DeleteURL sont vérifiés pour chaque item ;
- la liste vide produit une slice vide propre ;
- Add fournit la bonne `CancelURL` ;
- Edit et Delete fournissent le bon `SubjectContext` et la bonne `CancelURL` ;
- Add et Edit conservent apostrophes et guillemets ;
- tous les tests existants d'ownership, rows affected, FK utilisée et erreur DB restent passants.

## Règles métier

Aucune règle métier n'a changé. Les requêtes Subjects, l'ownership, les rows affected, la classification SQLite, les FK Questions et les contrats de suppression sont inchangés. Aucun autre référentiel n'a été modifié.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `internal/handlers/subjects/handlers.go` ;
- `internal/handlers/subjects/handlers_test.go` ;
- `internal/templates/data/subject.go` ;
- `internal/templates/subjects/table_subjects.html` ;
- `internal/templates/subjects/add_form_subject.html` ;
- `internal/templates/subjects/edit_form_subject.html` ;
- `internal/templates/subjects/delete_form_subject.html` ;
- `docs/audits/subjects-ux-pilot.md`.

## Recommandation

Le motif est réutilisable conceptuellement pour Thèmes, Niveaux, Compétences et Difficultés : item de liste typé, contexte typé, URL d'annulation, cartes responsive et mêmes actions. Points peut reprendre la même hiérarchie visuelle et les mêmes contrats de navigation, avec son contrôle numérique et ses options propres plutôt qu'un champ texte artificiellement uniformisé.
