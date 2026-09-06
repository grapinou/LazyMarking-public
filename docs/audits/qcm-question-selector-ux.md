# Refonte UX du sélecteur de questions QCM

## Structure visuelle

La page est organisée en trois zones :

1. en-tête « Ajouter des questions » avec le nom du QCM parent ;
2. carte compacte de filtres ;
3. liste responsive de familles de questions sélectionnables.

Les actions finales sont « Ajouter les questions sélectionnées » et « Annuler ». L’ancienne table multicolonne et son CSS local ont été supprimés au profit de cartes Bootstrap empilées.

## Filtres

Les six filtres existants sont conservés sans changement de paramètres ni de sémantique :

- matière (`subjectID`) ;
- thème (`themeID`) ;
- niveau (`yearLevelID`) ;
- compétence (`skillID`) ;
- difficulté (`difficultyID`) ;
- points (`pointID`).

Le formulaire reste en GET, avec `qcm_id` caché. Les valeurs sélectionnées sont préservées par les champs typés du view-model.

## Réinitialisation des filtres

L’action « Réinitialiser les filtres » utilise une URL construite par `QCMURL`. Elle revient au même sélecteur en conservant uniquement `qcm_id` et ne dépend d’aucun JavaScript.

## Affichage QuestionFamily

Chaque `QuestionFamily` est affichée dans une carte. La question principale porte :

- l’unique checkbox `question_ids` ;
- son contenu ;
- les métadonnées déjà chargées : matière, thème, niveau, compétence, difficulté et points.

Aucune requête supplémentaire ni N+1 n’a été introduit.

## Variantes

Les variantes déjà présentes dans `QuestionFamily.Variants` apparaissent sous « Variantes possibles » comme enfants informatifs. Elles n’ont ni checkbox, ni champ POST, ni action propre.

## Sélection et zéro sélection

Le formulaire d’ajout reste un POST vers la route existante et envoie `qcm_id` ainsi que les `question_ids` cochés. Le bouton est un vrai submit et aucun JavaScript de sélection n’a été ajouté.

Le comportement backend d’un POST sans sélection n’a pas été modifié : il reste le no-op transactionnel existant suivi de la redirection vers la composition.

## Navigation Annuler

Annuler utilise `Selector.TableURL`, construit avec `QCMRoutes.AddQuestionURL` et le bon `qcm_id`. La destination est donc « Questions du QCM », jamais Mes QCM.

## États vides

Les données GET permettent de distinguer deux situations sans requête supplémentaire :

- sans filtre actif : « Aucune question disponible à ajouter » ;
- avec au moins un filtre actif : « Aucune question ne correspond aux critères sélectionnés » avec action Reset.

Les questions déjà dans le QCM restent exclues et ne sont pas réaffichées comme désactivées.

## Données de vue

La structure ciblée `QCMQuestionSelectorData` a été ajoutée à `QCMQuestionPageData`. Elle regroupe uniquement les données du sélecteur :

- URLs GET, POST, Reset et Annuler ;
- collections des six filtres ;
- IDs sélectionnés ;
- `QuestionFamilies` ;
- indicateur `HasActiveFilters`.

Les anciennes clés `ExtraData` correspondantes ont été retirées du handler et du template. `QCMContext` reste le contexte parent typé.

## Contrat main/variant

`questionfamilies.Build`, `Selectable`, l’association main/variant et l’exclusion des questions déjà composées n’ont pas été modifiés. Les tests vérifient que chaque principale exposée reste sélectionnable et que chaque variante reste rattachée à son parent, sans sélection indépendante.

## Règles métier

Aucune règle métier, requête SQL, transaction d’ajout, position, tri des IDs, déplacement, suppression, Preview, worker, mélange, Exam ou Marking n’a changé.

## Tests

Deux fonctions de test, couvrant trois scénarios, ont été ajoutées :

- view-model complet : QCMContext, URLs, Reset limité à `qcm_id`, six filtres sélectionnés, principales sélectionnables et variantes enfants ;
- aucune question disponible sans filtre ;
- résultat vide provoqué par des filtres actifs.

Les tests existants continuent de couvrir l’exclusion des questions déjà ajoutées, les métadonnées des principales et la conservation de toutes leurs variantes.

## Validations

- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `internal/handlers/qcmQuestions/handlers.go`
- `internal/handlers/qcmQuestions/handlers_test.go`
- `internal/templates/data/qcmQuestions.go`
- `internal/templates/qcmquestions/add_form_qcm_question.html`
- `docs/audits/qcm-question-selector-ux.md`

Les autres entrées visibles dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) préexistaient à ce jalon et n’ont pas été modifiées pour cette refonte.
