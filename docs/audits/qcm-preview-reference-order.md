# Preview QCM en ordre pédagogique de référence

## 1. Primitive choisie

La fonction explicite suivante a été ajoutée dans `internal/handlers/tools` :

```go
func GetQCMQuestionsAnswersInReferenceOrder(
    userID, qcmID int64,
    r *http.Request,
    queries *db.Queries,
) ([]config.Question, error)
```

Elle charge les IDs via `GetQCMQuestionsIDs`, déjà triés par `qcm_questions.position`, puis les transmet directement au worker indexé. Elle ne possède ni booléen de stratégie, ni enum, ni abstraction générique.

Les petites fonctions privées `getQCMQuestionIDs` et `buildQCMQuestionsForRequest` évitent de dupliquer la lecture et la construction entre le mode historique et le mode de référence.

## 2. Comportement historique mélangé

`GetQCMQuestionsAnswers` conserve son contrat : lecture des IDs, appel à `shuffleQCMQuestionIDs`, puis construction indexée. `GetQCMQuestionsAnswersCtx` et ses callers sont inchangés.

`BuildQcmStudentCtx` et `GenerateMiniPDFHandler` continuent donc à produire une permutation aléatoire par élève. Le pipeline concurrent préserve ensuite exactement cette permutation.

## 3. Preview de référence

Portrait et paysage appellent tous deux `GetQCMQuestionsAnswersInReferenceOrder`. Aucun shuffle des familles n'est effectué.

Un test utilise volontairement les relations :

```text
question 30 → position 1
question 10 → position 2
question 20 → position 3
```

Les workers terminent dans l'ordre forcé `20,10,30`, mais la sortie reste `30,10,20`. Le test installe également un hook de shuffle qui modifierait l'ordre s'il était appelé et vérifie zéro appel.

## 4. Validation explicite du parent

Le helper local `loadPreviewQCMQuestions` est partagé par les deux handlers. Avant toute lecture de relation ou création de workspace, il appelle :

```go
GetQCMNameByID(qcmID, userID)
```

L'erreur passe par `tools.HandleOwnedLookupError`. Le parent n'est plus déduit d'une liste d'IDs potentiellement vide.

## 5. QCM absent

Un QCM inexistant retourne HTTP 404. La primitive de construction n'est pas appelée et aucun workspace n'est créé.

## 6. QCM étranger

Un QCM appartenant à un autre utilisateur retourne également HTTP 404, sans construction ni workspace. Ce comportement suit la convention ownership-aware du projet et ne révèle pas l'existence de la ressource.

## 7. QCM vide

Un QCM possédé dont la liste de questions est vide est distingué du 404 : le handler redirige en HTTP 303 vers `ErrorMessageURL` avec un message demandant d'ajouter au moins une question avant l'aperçu.

Aucune nouvelle page HTML n'a été créée.

## 8. Question sans réponse

`ErrQuestionWithNoAnswer` conserve sa traduction historique : redirection HTTP 303 vers `ErrorMessageURL`. `BuildQuestion`, son retry et les règles de validité des formulations n'ont pas changé.

## 9. Portrait et paysage

Les deux orientations partagent validation parentale, ordre de référence, cas vide et erreurs. Seule leur écriture Typst reste différente.

Les workspaces `preview-UUID`, la purge différée, le nettoyage sur erreur, la conservation après succès et le service PDF sont inchangés.

## 10. Aléatoire restant

Ce jalon retire uniquement le mélange de l'ordre des familles dans le Preview. Restent inchangés :

- le choix aléatoire entre question principale et variante ;
- le retry d'une formulation sans réponse ;
- le mélange des réponses.

## 11. Tests

Quatre tests ont été ajoutés :

1. ordre de référence `30,10,20`, achèvement inverse et zéro shuffle ;
2. QCM absent/étranger : 404 avant construction ;
3. portrait et paysage : même chemin de construction de référence pour un parent possédé ;
4. QCM possédé vide : redirection métier 303, distincte du 404.

Deux tests existants du pipeline mélangé ont été adaptés à la fixture dont IDs et positions diffèrent. Ils vérifient explicitement que le mode historique appelle toujours le shuffle et conserve la permutation choisie.

Les tests de Preview remplacent seulement la fonction de construction, ce qui évite d'invoquer Typst ou le filesystem tout en exerçant réellement les handlers et leur lookup DB.

## 12. Génération réelle

Aucun fichier Exams, GenerateExams, Marking ou `student_exam_content` n'a été modifié. La génération réelle utilise toujours :

```text
GetQCMQuestionsIDs
  → ShuffleSlice
  → workers indexés
```

## 13. Race detector

`go test -race ./internal/handlers/tools/...` : réussi.

## 14. Validations

- `go test ./...` : réussi ;
- `go vet ./...` : réussi ;
- `git diff --check` : réussi.

## 15. Fichiers modifiés

- `internal/handlers/tools/getQCMQuestionsAnswers.go` ;
- `internal/handlers/tools/getQCMQuestionsAnswers_test.go` ;
- `internal/handlers/qcmPreview/handlers.go` ;
- `internal/handlers/qcmPreview/handlers_test.go` ;
- `docs/audits/qcm-preview-reference-order.md`.

Aucun SQL, sqlc, migration, template, Bootstrap, Typst, workspace, génération réelle, QCMContext ou réglage utilisateur n'a été modifié. Aucun commit n'a été créé.
