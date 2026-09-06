# Tests des échecs filesystem après suppression DB

## 1. Simulation de l’échec filesystem

Chaque package concerné possède désormais un alias de fonction non exporté :

```go
var removeStoredImageFile = tools.RemoveStoredImageFile
```

Le code de production appelle cet alias uniquement dans le flux de suppression testé. Les tests le remplacent temporairement par une fonction déterministe qui retourne une erreur, puis le restaurent avec `t.Cleanup`.

Cette injection locale évite les erreurs dépendantes des permissions OS et ne crée ni interface, ni service, ni framework de mocks.

## 2. Modifications de production nécessaires

Les quatre packages utilisent l’alias local pour l’étape filesystem de leur suppression :

- `images` ;
- `altImages` ;
- `questions` ;
- `altQuestions`.

Les logs des suppressions principale, variante et variante parente incluent maintenant explicitement le nom du fichier, comme le faisait déjà la suppression de question principale multi-images. Aucun autre comportement de production n’a changé.

## 3. Suppression directe principale

`TestDeleteImageHandlerKeepsDatabaseDeletionWhenFilesystemRemovalFails` prépare une question, une ligne `images` et `main.png`, puis injecte une erreur de suppression physique.

Le test vérifie :

- réponse HTTP 303 inchangée ;
- ligne `images` supprimée ;
- fichier `main.png` toujours présent ;
- absence de restauration DB.

## 4. Suppression directe variante

`TestDeleteAltImageHandlerKeepsDatabaseDeletionWhenFilesystemRemovalFails` utilise les vrais paramètres `question_id=42`, `alt_question_id=7` et l’utilisateur authentifié 1.

Le test vérifie :

- réponse HTTP 303 ;
- ligne `alt_images` supprimée ;
- fichier `variant-7.png` toujours présent ;
- contrat parental `AltQuestionID + QuestionID + UserID` inchangé.

## 5. Suppression de la variante parente

`TestDeleteAltQuestionHandlerKeepsCascadeWhenFilesystemRemovalFails` utilise une DB SQLite avec FK activées et `ON DELETE CASCADE`.

Après erreur filesystem injectée :

- réponse HTTP 303 ;
- variante absente ;
- ligne `alt_images` absente par cascade ;
- fichier toujours présent ;
- aucune compensation ni recréation DB.

## 6. Suppression d’une question avec plusieurs images

`TestDeleteQuestionHandlerContinuesRemovingFilesAfterOneFailure` construit :

```text
question 42
  ├── main.png
  ├── variante 7 → a.png
  └── variante 8 → b.png
```

L’injection échoue uniquement pour `a.png` et délègue `main.png` et `b.png` au véritable `tools.RemoveStoredImageFile`.

Après le handler :

- question, variantes, `images` et `alt_images` sont vides ;
- `main.png` et `b.png` sont supprimés ;
- `a.png` reste orphelin ;
- chaque nom a été tenté exactement une fois.

La suppression réussie de `b.png` et `main.png` prouve qu’une erreur sur `a.png` n’interrompt pas la boucle.

## 7. Fichier déjà absent

`TestDeleteImageHandlerAcceptsAlreadyAbsentFile` supprime une ligne `images` sans créer le fichier correspondant. Le vrai helper considère `os.ErrNotExist` comme un succès : la ligne disparaît et le handler répond normalement 303.

Ce scénario est couvert une fois au niveau intégration, sans duplication dans les quatre handlers.

## 8. DELETE DB échoué

`TestDeleteImageHandlerDoesNotRemoveFileWhenDatabaseDeleteFails` installe un trigger SQLite qui force l’échec du DELETE après que la lecture préalable a réussi.

Le test vérifie :

- réponse HTTP 500 ;
- ligne DB toujours présente ;
- fichier toujours présent ;
- alias filesystem jamais appelé.

Le sens inverse reste donc protégé : aucune suppression physique n’est tentée avant un DELETE DB réussi.

## 9. Observabilité

Chaque échec filesystem des quatre handlers journalise désormais :

- l’opération/handler concerné ;
- le nom exact du fichier ;
- l’erreur retournée.

Les tests exercent les chemins d’erreur sans dépendre du texte exact des logs.

## 10. Ordre DB → filesystem

L’ordre reste strictement inchangé : collecte/lecture du nom, DELETE DB et contrôle du nombre de lignes, puis suppression physique. Les réponses HTTP de succès après erreur filesystem restent des redirections 303.

Aucune purge, compensation, transaction, quarantaine, restauration ou utilisation du scanner n’a été ajoutée.

## 11. Tests ajoutés

Six scénarios de suppression sont ajoutés :

1. suppression directe principale avec échec filesystem ;
2. fichier principal déjà absent ;
3. DELETE DB principal échoué sans appel filesystem ;
4. suppression directe variante avec échec filesystem ;
5. suppression de variante parente avec échec filesystem ;
6. suppression de question parente multi-images avec échec partiel et poursuite de boucle.

## 12. Validations

- `go test ./...` : réussi.
- `go vet ./...` : réussi.
- `git diff --check` : réussi.

## 13. Fichiers modifiés pour ce jalon

1. `internal/handlers/images/handlers.go`
2. `internal/handlers/images/handlers_test.go`
3. `internal/handlers/altImages/handlers.go`
4. `internal/handlers/altImages/handlers_test.go`
5. `internal/handlers/questions/handlers.go`
6. `internal/handlers/questions/handlers_test.go`
7. `internal/handlers/altQuestions/handlers.go`
8. `internal/handlers/altQuestions/handlers_test.go`
9. `docs/audits/image-delete-failure-tests.md`
