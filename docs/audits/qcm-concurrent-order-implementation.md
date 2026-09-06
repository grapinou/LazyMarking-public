# Préservation de l'ordre dans le pipeline concurrent QCM

## 1. Pipeline avant

Les deux fonctions publiques lisaient les IDs, appelaient `ShuffleSlice`, puis envoyaient uniquement les IDs aux workers. Chaque worker envoyait uniquement la question construite dans un channel de résultats. Le consommateur faisait ensuite `append` dans l'ordre d'arrivée.

La permutation choisie avant les workers pouvait donc être modifiée une seconde fois, involontairement, par les différences de temps de construction.

## 2. Pipeline après

Le pipeline est désormais :

```text
IDs lus par position
  → ShuffleSlice inchangé
  → jobs {index, questionID}
  → workers concurrents
  → results {index, question, err}
  → questions[result.index] = result.question
```

Le slice final est préalloué à la longueur des IDs. Les workers choisissent seulement quand une question est construite ; ils ne choisissent plus sa place.

## 3. Types adoptés

La petite primitive interne dédiée au QCM se trouve dans `internal/handlers/tools/qcmQuestionWorkers.go` :

- `qcmQuestionJob` contient `Index` et `QuestionID` ;
- `qcmQuestionResult` contient `Index`, `Question` et `Err` ;
- `buildQCMQuestionsInOrder` exécute cinq workers et reconstruit le résultat indexé.

Il ne s'agit ni d'un worker pool générique, ni d'une interface, ni d'un nouveau package.

## 4. Préservation de l'index

Chaque ID reçoit son index lors de la création des jobs. Un job n'est lu que par un worker et produit exactement un résultat. Le consommateur unique écrit chaque résultat à l'index correspondant dans le slice préalloué.

Les workers n'écrivent pas directement dans le slice partagé. Il n'existe donc aucune écriture concurrente dans celui-ci et chaque index est assigné une seule fois.

## 5. `ShuffleSlice`

Le shuffle reste exécuté au même endroit, immédiatement après `GetQCMQuestionsIDs`, dans les deux fonctions publiques. Un hook local typé permet seulement aux tests de choisir une permutation déterministe.

La génération réelle continue ainsi de choisir une permutation aléatoire par copie. La correction garantit désormais que cette permutation exacte survit aux workers.

## 6. Variantes et réponses

`BuildQuestion`, `BuildQuestionCtx`, `GetRandomQuestionByQuestionID`, le retry, le choix main/`alt_question` et le mélange des réponses n'ont pas été modifiés. `MainQuestionID` et les structures sérialisées restent inchangés.

## 7. Erreurs et contexte

Chaque worker produit un résultat contenant éventuellement l'erreur. Les channels sont dimensionnés au nombre de jobs ; le producteur remplit puis ferme `jobs`, attend tous les workers et ferme une seule fois `results`. Il n'existe ni sender bloqué, ni double fermeture, ni goroutine abandonnée.

Après collecte, la première erreur observée est propagée et aucun slice partiel n'est retourné. Tous les jobs démarrés par cette petite primitive terminent proprement, ce qui rend également vérifiable qu'une question n'est jamais construite deux fois.

La version contextuelle vérifie désormais `ctx.Done()` pendant l'acquisition du sémaphore global et continue de transmettre le contexte à `BuildQuestionCtx`. Une annulation est propagée comme erreur.

## 8. Tests déterministes

Quatre tests ont été ajoutés :

1. entrée `[10,20,30]`, achèvement forcé `[30,20,10]`, sortie `[10,20,30]` ;
2. permutation choisie `[30,10,20]`, autre ordre d'achèvement forcé, sortie strictement identique à la permutation ;
3. propagation d'une erreur avec vérification que chaque ID est construit exactement une fois et qu'aucun résultat partiel n'est rendu ;
4. propagation de `context.Canceled` dans la version contextuelle.

Les tests utilisent des channels de démarrage, de libération et d'achèvement. Aucun `time.Sleep` ni comportement implicite du scheduler ne pilote l'ordre.

De petits hooks locaux remplacent uniquement le builder et le shuffle pendant les tests. Les deux fonctions publiques sont donc réellement exercées, avec une DB SQLite minimale fournissant les IDs par position.

## 9. Race detector

`go test -race ./internal/handlers/tools/...` : réussi.

## 10. Preview

`PreviewQCMHandler` et `PreviewQCMLandscapeHandler` n'ont pas été modifiés. Ils continuent à appeler la fonction publique mélangée. Le passage à l'ordre de référence reste un jalon séparé.

## 11. Génération réelle

`BuildQcmStudentCtx` et `GenerateMiniPDFHandler` sont inchangés. Ils continuent à appeler les versions qui mélangent les IDs. La génération reste individualisée ; seule la permutation déjà choisie est maintenant respectée exactement.

## 12. Validations

- `go test ./...` : réussi ;
- `go vet ./...` : réussi ;
- `git diff --check` : réussi ;
- `go test -race ./internal/handlers/tools/...` : réussi.

## 13. Fichiers modifiés

- `internal/handlers/tools/getQCMQuestionsAnswers.go` ;
- `internal/handlers/tools/getQCMQuestionsAnswersCtx.go` ;
- `internal/handlers/tools/qcmQuestionWorkers.go` ;
- `internal/handlers/tools/getQCMQuestionsAnswers_test.go` ;
- `docs/audits/qcm-concurrent-order-implementation.md`.

Aucun SQL, sqlc, migration, template, preview, caller de génération, choix de variante ou mélange de réponses n'a été modifié. Aucun commit n'a été créé.
