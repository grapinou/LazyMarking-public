# Refonte UX de la liste des QCM

## Structure visuelle

La table CRUD a été remplacée par une grille responsive de cartes Bootstrap. Chaque carte représente un QCM avec :

- son nom ;
- son nombre de questions ;
- l’action principale « Gérer les questions » ;
- les aperçus portrait et paysage ;
- les actions secondaires Modifier et Supprimer.

L’en-tête « Mes QCM » porte l’action « Créer un QCM ». Les cartes s’affichent sur une colonne en largeur réduite et deux colonnes sur grand écran.

## QCMListItem

La structure typée `QCMListItem` a été créée. Elle regroupe l’ID, le nom, le compteur et toutes les URLs d’un QCM. `QCMPageData.QCMItems` remplace, pour la liste, les anciennes slices parallèles `ExtraData.QCM` et `ExtraData.Action`.

## Stratégie de comptage

`GetAllQCM` retourne désormais le compteur dans la même requête que la liste. Elle utilise un `LEFT JOIN` de `qcm` vers `qcm_questions`, suivi d’un `COUNT(qcm_questions.id)` et d’un groupement par QCM.

Le `LEFT JOIN` conserve les QCM vides avec un compteur à zéro. La jointure exige également la cohérence de `user_id`, et le filtre principal reste ownership-aware sur `qcm.user_id`.

## N+1

Il n’existe aucun N+1 : le handler effectue un seul appel à `GetAllQCM`, quel que soit le nombre de QCM affichés. Il ne charge pas les questions pour les compter.

## Action principale

« Gérer les questions » conduit à la page de composition avec le bon `qcm_id`. Elle est visuellement prioritaire sur Modifier et Supprimer.

## Aperçus

Les aperçus portrait et paysage restent disponibles comme actions secondaires pour les QCM contenant des questions. Ils sont présentés désactivés pour un QCM vide, sans modifier les handlers Preview.

## État vide

La table vide a été remplacée par une carte pédagogique : elle indique qu’aucun QCM n’a encore été créé, explique brièvement la prochaine étape et propose directement « Créer un QCM ».

## SQL et sqlc

`db/query/qcm.sql` a été modifié uniquement pour enrichir `GetAllQCM` avec le compteur agrégé. `internal/db/qcm.sql.go` a été régénéré avec sqlc ; aucune modification manuelle du code généré et aucune migration n’ont été nécessaires.

## Tests

Trois tests ont été ajoutés :

- requête agrégée : ownership, plusieurs QCM, plusieurs questions et compteurs indépendants ;
- view-model de liste : ID, nom, compteur et cinq URLs propres à chaque QCM ;
- liste vide : slice typée non nil et vide.

Les tests portent sur les données et les contrats d’URL, pas sur les classes Bootstrap ou la structure détaillée du HTML.

## Périmètre inchangé

L’ordre, `QCMQuestionItem`, Monter/Descendre, la composition, Preview, la génération, `ShuffleSlice`, les variantes, les réponses, Exams et Marking n’ont pas été modifiés.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès.
- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `db/query/qcm.sql`
- `internal/db/qcm.sql.go`
- `internal/db/qcmRelationshipIntegrity_test.go`
- `internal/handlers/qcm/handlers.go`
- `internal/handlers/qcm/handlers_test.go`
- `internal/templates/data/qcm.go`
- `internal/templates/qcm/table_qcm.html`
- `docs/audits/qcm-list-ux.md`

Les autres entrées déjà présentes dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) sont hors périmètre et n’ont pas été modifiées pour ce jalon.
