# Refactor du contexte QCM

## Définition et emplacement

`QCMContext` est défini dans `internal/templates/data/qcm.go`, à côté des routes et données de vue propres au QCM :

```go
type QCMContext struct {
    ID   int64
    Name string
}
```

L'identifiant reste typé en `int64`. Aucun package, service, repository ou loader supplémentaire n'a été créé.

## PageData migrées

Deux structures reçoivent désormais un champ `QCMContext QCMContext` :

- `QCMPageData`, utilisé ici par les formulaires de modification et de suppression du QCM ;
- `QCMQuestionPageData`, utilisé par la composition, le sélecteur et la confirmation de retrait.

La liste des QCM conserve sa collection de `db.Qcm` : elle n'a pas de parent QCM unique et n'a donc pas été convertie artificiellement.

## Handlers constructeurs

Cinq handlers GET construisent le contexte :

1. `EditFormQCMHandler` ;
2. `DeleteFormQCMHandler` ;
3. `TableQCMQuestionsHandler` ;
4. `AddFormQCMQuestionHandler` ;
5. `DeleteFormQCMQuestionHandler`.

Chaque contexte réutilise l'ID déjà parsé et le nom renvoyé par le lookup ownership-aware existant. Le sélecteur et la confirmation de retrait conservaient auparavant seulement le résultat du lookup pour validation ; ils réutilisent maintenant ce même nom au lieu de lancer une seconde requête.

## Clés `ExtraData` supprimées

Les clés parentales devenues redondantes ont été supprimées :

- `QCM` et `QCMID` dans les formulaires QCM edit/delete ;
- `QCMName` dans la composition ;
- `QCMID` dans le sélecteur ;
- `QCMID` dans la confirmation de retrait.

Les collections, filtres, états vides, contenus de question et URLs spécifiques restent volontairement dans `ExtraData`.

## Helper URL

Le helper pur suivant a été ajouté dans `internal/templates/data/qcm.go` :

```go
func QCMURL(base string, qcmID int64) string
```

Il centralise uniquement les URLs simples possédant le paramètre `qcm_id`, avec `strconv.FormatInt` et `url.QueryEscape`, selon la convention de `QuestionURL` et `VariantURL`. Les URLs à plusieurs identifiants, comme le retrait d'une relation, restent construites localement.

## Templates migrés

Cinq templates utilisent désormais `.QCMContext.ID` ou `.QCMContext.Name`, sans changement visuel volontaire :

- `internal/templates/qcm/edit_form_qcm.html` ;
- `internal/templates/qcm/delete_form_qcm.html` ;
- `internal/templates/qcmquestions/table_qcmquestion.html` ;
- `internal/templates/qcmquestions/add_form_qcm_question.html` ;
- `internal/templates/qcmquestions/delete_form_qcm_question.html`.

Les titres, tableaux, classes Bootstrap, boutons, libellés, états vides et ordre des actions sont inchangés.

## Tests

Un test handler tabulaire couvre trois scénarios :

- composition : contexte `{ID: 1, Name: "owned"}` ;
- sélecteur : contexte `{ID: 3, Name: "owned empty"}` ;
- confirmation de retrait : contexte du QCM auquel appartient réellement la relation.

Les fonctions de rendu sont remplacées localement dans ce test afin d'inspecter les PageData sans snapshot HTML ni assertion Bootstrap. Les tests existants continuent de vérifier les 404 pour QCM absent ou étranger et le mauvais parent de relation.

Un test unitaire couvre également `QCMURL`.

## Invariants inchangés

Aucune règle métier, requête SQL, migration, fichier sqlc, transaction, question family, filtre, preview, suppression QCM, ordre de questions, mélange, worker ou génération Typst n'a changé.

## Validations

- `go test ./...` : réussi ;
- `go vet ./...` : réussi ;
- `git diff --check` : réussi.

## Fichiers modifiés pour ce jalon

- `internal/templates/data/qcm.go` ;
- `internal/templates/data/qcmQuestions.go` ;
- `internal/templates/data/family_test.go` ;
- `internal/handlers/qcm/handlers.go` ;
- `internal/handlers/qcmQuestions/handlers.go` ;
- `internal/handlers/qcmQuestions/handlers_test.go` ;
- `internal/templates/qcm/edit_form_qcm.html` ;
- `internal/templates/qcm/delete_form_qcm.html` ;
- `internal/templates/qcmquestions/table_qcmquestion.html` ;
- `internal/templates/qcmquestions/add_form_qcm_question.html` ;
- `internal/templates/qcmquestions/delete_form_qcm_question.html` ;
- `docs/audits/qcm-context-refactor.md`.

Aucun commit n'a été créé.
