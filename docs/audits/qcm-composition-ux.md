# Refonte UX de la composition d’un QCM

## Structure visuelle retenue

La page est organisée comme une composition pédagogique :

1. titre « Questions du QCM » et nom du QCM parent ;
2. compteur de questions ;
3. séquence ordonnée de cartes de questions ;
4. actions Ajouter et Aperçus ;
5. retour vers la liste des QCM.

La table technique a été remplacée par des cartes Bootstrap empilées. Le contenu long revient naturellement à la ligne et les groupes d’actions utilisent `flex-wrap`.

## Données de vue

`QCMQuestionPageData` expose désormais directement :

- `QCMQuestions` ;
- `AddQuestionsURL` ;
- `PreviewURL` ;
- `PreviewLandscapeURL`.

Le parent reste fourni par le `QCMContext` typé. Les anciennes entrées parallèles `ExtraData.QCMQuestions`, `ExtraData.Action`, `ExtraData.NoQCMQuestion` et `ExtraData.AddURL` ont été retirées de cette page.

## QCMQuestionItem

La structure typée `QCMQuestionItem` regroupe les données cohérentes d’une question de la composition : identifiant de relation, position, contenu, indicateurs première/dernière, routes Monter/Descendre et URL de retrait.

Cette structure supprime le couplage fragile entre une slice de questions et une slice d’actions indexée séparément.

## Monter et Descendre

Chaque carte contient deux formulaires `POST` explicites. Ils envoient exactement :

- `qcm_id` depuis `QCMContext.ID` ;
- `qcm_question_id` depuis l’item courant.

Les libellés textuels « Monter » et « Descendre » accompagnent les icônes décoratives.

## Bornes

Le bouton Monter est désactivé sur le premier item. Le bouton Descendre est désactivé sur le dernier. Les indicateurs `IsFirst` et `IsLast` sont construits par le handler à partir de la slice déjà ordonnée ; aucune règle métier de déplacement n’a été ajoutée.

## Retirer

L’accès à la confirmation existante est conservé sous le libellé « Retirer ». Il utilise un bouton secondaire de style outline danger, visuellement distinct des actions de déplacement sans modifier le formulaire de confirmation.

## Ajouter des questions

Une action « Ajouter des questions » utilise l’URL existante enrichie du `qcm_id`. Elle apparaît après la séquence et constitue l’action principale de l’état vide.

## Aperçus portrait et paysage

Les deux URLs sont construites dans les données de vue avec `QCMURL` et les routes Preview existantes. Les actions « Aperçu portrait » et « Aperçu paysage » apparaissent uniquement lorsque le QCM contient au moins une question. Aucun handler Preview n’a été modifié.

## État vide

Un QCM vide affiche une carte pédagogique expliquant qu’aucune question n’a encore été ajoutée et propose directement d’en ajouter depuis la banque. Aucune table ni action Preview vide n’est rendue.

## Responsive et accessibilité

- cartes empilées sans tableau multicolonne ;
- contenu long avec retour à la ligne ;
- actions pouvant s’enrouler sur plusieurs lignes ;
- boutons textuels ;
- icônes complémentaires marquées `aria-hidden` ;
- groupes d’actions nommés ;
- formulaires de déplacement strictement en POST.

## Tests ajoutés

Deux tests handler ciblés ont été ajoutés :

- composition peuplée : ordre 1, 2, 3, identifiants relationnels, première/dernière, routes de déplacement, retrait, ajout et aperçus avec le bon `qcm_id` ;
- composition vide : slice typée vide, contexte parent et URL d’ajout valides.

Ils ne figent aucune classe Bootstrap ni structure détaillée de `div`.

## Périmètre métier inchangé

Aucune règle de position, compactage, échange, génération, mélange, famille de questions, Preview backend, Typst, Exams ou Marking n’a été modifiée. Aucun SQL, fichier sqlc ou migration n’a changé.

## Validations

- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `internal/handlers/qcmQuestions/handlers.go`
- `internal/handlers/qcmQuestions/handlers_test.go`
- `internal/templates/data/qcmQuestions.go`
- `internal/templates/qcmquestions/table_qcmquestion.html`
- `docs/audits/qcm-composition-ux.md`

Les autres entrées déjà présentes dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) sont hors périmètre et n’ont pas été modifiées pour ce jalon.
