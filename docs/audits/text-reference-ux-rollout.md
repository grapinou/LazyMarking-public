# Déploiement UX des référentiels textuels

## Résumé

Le motif validé sur Matières a été appliqué explicitement à Thèmes, Niveaux, Compétences et Difficultés. Chaque domaine conserve ses propres types, helpers, routes, handlers et templates. Aucune abstraction CRUD générique, aucune migration et aucun changement SQL ou métier n'ont été introduits.

## Thèmes

- liste responsive en cartes, avec actions textuelles Modifier/Supprimer ;
- `ThemeListItem` et `ThemeContext` créés ;
- helper local `ThemeURL` ;
- Add, Edit et Delete refaits avec le vocabulaire « Thème » ;
- Annuler revient à la liste Thèmes ;
- retour « Retour à la banque de questions » ;
- état vide pédagogique et action Ajouter ;
- confirmation : « Un thème utilisé par une question ne peut pas être supprimé. » ;
- ancien JavaScript supprimant les guillemets retiré ;
- aucun `ExtraData` restant ;
- tests : items/URLs, liste vide, contextes, CancelURL, apostrophes et guillemets.

## Niveaux

- liste responsive en cartes ;
- `YearLevelListItem` et `YearLevelContext` créés ;
- helper local `YearLevelURL` ;
- Add, Edit et Delete refaits ;
- tout le vocabulaire visible emploie « Niveau/Niveaux », jamais « Classe/Classes » ;
- Annuler revient à la liste Niveaux ;
- retour Banque et état vide harmonisés ;
- confirmation : « Un niveau utilisé par une question ne peut pas être supprimé. » ;
- JavaScript de filtrage retiré ;
- aucun `ExtraData` restant ;
- tests : items/URLs, liste vide, contextes, CancelURL, apostrophes et guillemets.

## Compétences

- liste responsive en cartes ;
- `SkillListItem` et `SkillContext` créés ;
- helper local `SkillURL` ;
- Add, Edit et Delete refaits ;
- Annuler revient à la liste Compétences ;
- retour Banque et état vide harmonisés ;
- confirmation : « Une compétence utilisée par une question ne peut pas être supprimée. » ;
- JavaScript de filtrage retiré ;
- aucun `ExtraData` restant ;
- tests : items/URLs, liste vide, contextes, CancelURL, apostrophes et guillemets.

## Difficultés

- liste responsive en cartes ;
- `DifficultyListItem` et `DifficultyContext` créés ;
- helper local `DifficultyURL` ;
- Add, Edit et Delete refaits ;
- tous les textes visibles emploient correctement « Difficulté/Difficultés » ;
- les formes fautives `difficultée` et `difficil` ont disparu du périmètre ;
- Annuler revient à la liste Difficultés ;
- retour Banque et état vide harmonisés ;
- confirmation : « Une difficulté utilisée par une question ne peut pas être supprimée. » ;
- JavaScript de filtrage retiré ;
- aucun `ExtraData` restant ;
- tests : items/URLs, liste vide, contextes, CancelURL, apostrophes et guillemets.

## Motif commun respecté

Les quatre domaines suivent conceptuellement le pilote Subjects :

- liste typée sans slices parallèles ni booléen `NoX` ;
- cartes Bootstrap responsive, `flex-wrap` et `text-break` ;
- actions textuelles et icônes décoratives `aria-hidden` ;
- état vide piloté naturellement par une slice vide ;
- contexte typé avec ID `int64` et nom ;
- URL d'annulation typée ;
- labels associés par `for`/`id` ;
- confirmations sobres expliquant la protection RESTRICT ;
- aucune table CRUD technique et aucun JavaScript local.

Les seules différences sont les noms de types/routes/champs POST et le vocabulaire pédagogique propre à chaque domaine. Elles sont intentionnelles. Aucun type, template ou handler générique n'a été créé.

## Apostrophes et guillemets

Les tests de chaque domaine confirment que l'ajout et la modification conservent exactement les apostrophes et guillemets. Les handlers gardent leur `TrimSpace`, leur unicité par utilisateur, leur ownership et leur gestion des lignes affectées.

## Règles métier

Aucune règle métier n'a été modifiée. Les requêtes SQL, FK Questions, suppressions sécurisées, classification SQLite et redirections métier existantes sont inchangées. Points et ses fichiers n'ont pas été modifiés.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

### Thèmes

- `internal/handlers/themes/handlers.go` ;
- `internal/handlers/themes/handlers_test.go` ;
- `internal/templates/data/theme.go` ;
- `internal/templates/themes/table_themes.html` ;
- `internal/templates/themes/add_form_theme.html` ;
- `internal/templates/themes/edit_form_theme.html` ;
- `internal/templates/themes/delete_form_theme.html`.

### Niveaux

- `internal/handlers/yearlevels/handlers.go` ;
- `internal/handlers/yearlevels/handlers_test.go` ;
- `internal/templates/data/yearlevel.go` ;
- `internal/templates/yearlevels/table_yearlevels.html` ;
- `internal/templates/yearlevels/add_form_yearlevel.html` ;
- `internal/templates/yearlevels/edit_form_yearlevel.html` ;
- `internal/templates/yearlevels/delete_form_yearlevel.html`.

### Compétences

- `internal/handlers/skills/handlers.go` ;
- `internal/handlers/skills/handlers_test.go` ;
- `internal/templates/data/skill.go` ;
- `internal/templates/skills/table_skills.html` ;
- `internal/templates/skills/add_form_skill.html` ;
- `internal/templates/skills/edit_form_skill.html` ;
- `internal/templates/skills/delete_form_skill.html`.

### Difficultés

- `internal/handlers/difficulties/handlers.go` ;
- `internal/handlers/difficulties/handlers_test.go` ;
- `internal/templates/data/difficulty.go` ;
- `internal/templates/difficulties/table_difficulties.html` ;
- `internal/templates/difficulties/add_form_difficulty.html` ;
- `internal/templates/difficulties/edit_form_difficulty.html` ;
- `internal/templates/difficulties/delete_form_difficulty.html`.

### Rapport

- `docs/audits/text-reference-ux-rollout.md`.
