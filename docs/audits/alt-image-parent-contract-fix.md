# Correction du contrat parental des images de variante

## Objectif du jalon

Ce jalon corrige exclusivement les appelants de `GetAltImageByAltQuestionID`. La requête attend les trois identifiants suivants :

```go
AltQuestionID int64
UserID        int64
QuestionID    int64
```

Les appelants ne renseignaient pas `QuestionID`, qui conservait donc sa valeur zéro. La requête SQL renforcée n’a pas été modifiée : chaque appelant fournit désormais le véritable parent de la variante.

Aucun changement UX, template, SQL, migration, schéma, contexte typé ou stratégie filesystem n’a été réalisé.

## 1. Inventaire exhaustif des appelants

Neuf appelants applicatifs ont été trouvés. Ils étaient tous incorrects avant ce jalon.

| Fichier | Fonction | IDs déjà disponibles | Source correcte de `QuestionID` | Rôle |
|---|---|---|---|---|
| `internal/handlers/altImages/handlers.go` | `TableAltImageHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé et validé par `GetAltQuestionByParentID` | Gestion / affichage |
| `internal/handlers/altImages/handlers.go` | `EditFormAltImageHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé et validé par `GetAltQuestionByParentID` | Formulaire de redimensionnement |
| `internal/handlers/altImages/handlers.go` | `EditAltImageHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé depuis le POST | Redimensionnement |
| `internal/handlers/altImages/handlers.go` | `DeleteFormAltImageHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé et validé par `GetAltQuestionByParentID` | Formulaire de suppression |
| `internal/handlers/altImages/handlers.go` | `DeleteAltImageHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé depuis le POST | Suppression / nettoyage |
| `internal/handlers/altQuestions/handlers.go` | `DeleteAltQuestionHandler` | `userID`, `questionID`, `altQuestionID` | `questionID` parsé et validé par `GetAltQuestionByParentID` | Suppression parente / nettoyage |
| `internal/handlers/questions/handlers.go` | `DeleteQuestionHandler` | `userID`, `questionID`, liste des `altQuestionID` | `questionID` de la question en cours de suppression | Suppression parente / nettoyage |
| `internal/handlers/tools/getAltQuestionAltAnswer.go` | `GetAltQuestionAltAnswer` | `userID`, `altQuestionID`, ligne `altQuestionDB` | `altQuestionDB.QuestionID` | Aperçu et génération synchrone |
| `internal/handlers/tools/getAltQuestionAltAnswerCtx.go` | `GetAltQuestionAltAnswerCtx` | `userID`, `altQuestionID`, ligne `altQuestionDB` | `altQuestionDB.QuestionID` | Génération avec contexte |

La méthode sqlc générée dans `internal/db/altImages.sql.go` n’est pas comptée comme appelant applicatif.

## 2. Appelants incorrects

Les neuf littéraux `db.GetAltImageByAltQuestionIDParams` ne contenaient auparavant que :

```go
AltQuestionID: altQuestionID,
UserID:        userID,
```

Le champ `QuestionID` valait implicitement `0`. Comme le SQL vérifie que la variante appartient à `QuestionID`, une image existante n’était normalement jamais retrouvée.

## 3. Corrections effectuées

Dans les handlers possédant déjà le parent parsé, le champ suivant a été ajouté :

```go
QuestionID: questionID,
```

Dans les deux helpers utilisés par l’aperçu et la génération, la variante vient d’être chargée avec `GetAltQuestionByID`. Son parent réel est donc utilisé localement :

```go
QuestionID: altQuestionDB.QuestionID,
```

Aucune valeur artificielle, aucun nouveau chargement générique et aucun affaiblissement du SQL n’ont été introduits.

Une recherche finale de tous les littéraux `GetAltImageByAltQuestionIDParams` confirme que les neuf appelants fournissent explicitement `QuestionID`.

## 4. Gestion, redimensionnement et suppression

### Page de gestion

`TableAltImageHandler` transmet maintenant le couple validé `questionID + altQuestionID`. Une image existante de variante n’est plus interprétée comme absente à cause d’un parent zéro.

### Redimensionnement

Les deux lectures préalables sont corrigées :

- `EditFormAltImageHandler` retrouve la ligne et sa taille actuelle ;
- `EditAltImageHandler` retrouve le fichier avant `ReadImageConfig`, `ValidateImageResize` et `ImageCircleCheck`.

`UpdateSizeAltImage` transmettait déjà `QuestionID` et n’a pas été modifié.

### Suppression

Les deux chemins sont corrigés :

- `DeleteFormAltImageHandler` vérifie l’image avec les trois IDs ;
- `DeleteAltImageHandler` récupère correctement `ImageName` avant la suppression DB puis le nettoyage filesystem existant.

`DeleteAltImage` transmettait déjà `QuestionID` et n’a pas été modifié.

## 5. Suppressions parentes

### Suppression d’une variante illustrée

`DeleteAltQuestionHandler` utilisait le bon `questionID` pour valider la variante, mais ne le transmettait pas lors de la recherche de son image. Il le transmet maintenant, ce qui permet :

1. de récupérer `ImageName` ;
2. de supprimer la variante ;
3. de laisser la cascade DB supprimer la ligne image lorsque les FK sont actives ;
4. d’appeler le nettoyage filesystem prévu avec le bon nom.

### Suppression d’une question contenant une variante illustrée

`DeleteQuestionHandler` possède la question parente et parcourt ses IDs de variantes illustrées. Chaque lecture d’image utilise maintenant ce même `questionID`, ce qui permet de constituer la liste des fichiers avant la suppression de la question et ses cascades.

La stratégie existante DB puis filesystem reste inchangée.

## 6. Aperçu et génération

Deux helpers construisent une `config.Question` à partir d’une variante :

- `GetAltQuestionAltAnswer`, utilisé notamment par l’aperçu et la génération synchrone ;
- `GetAltQuestionAltAnswerCtx`, utilisé par les chemins de génération avec `context.Context`.

Ils chargent déjà `altQuestionDB`. `altQuestionDB.QuestionID` est désormais transmis à `GetAltImageByAltQuestionID`. Une variante illustrée restitue à nouveau :

```text
question.Image.Name
question.Image.Width
```

Le contrat remis en état est celui que les producteurs Typst consomment déjà. Aucun producteur Typst n’a été modifié.

## 7. Tests ajoutés

### Handlers `altImages`

Nouveau fichier `internal/handlers/altImages/handlers_test.go` :

- `TestTableAltImageHandlerUsesQuestionParentToReadExistingImage`
  - utilisateur 1 ;
  - question 42 ;
  - variante 7 ;
  - image `variant-7.png` ;
  - vérifie que le nom apparaît dans le rendu ;
  - vérifie que l’état « Pas d’image » n’est pas affiché.
- `TestTableAltImageHandlerRejectsMismatchedAndForeignVariants`
  - variante d’une autre question possédée ;
  - variante d’un autre utilisateur ;
  - réponse 404 dans les deux cas.
- `TestDeleteFormAltImageHandlerRejectsMismatchedParent`
  - mauvais couple question/variante ;
  - réponse 404.
- `TestDeleteAltImageHandlerRemovesOwnedVariantImageAndFile`
  - retrouve l’image avec les trois IDs ;
  - supprime la ligne DB ;
  - supprime le fichier prévu ;
  - redirige après succès.

### Aperçu et génération

Nouveau fichier `internal/handlers/tools/getAltQuestionAltAnswer_test.go` :

- `TestAltQuestionBuildersIncludeOwnedVariantImage`
  - scénario utilisateur → question 42 → variante 7 → image ;
  - sous-test du helper requête utilisé par l’aperçu ;
  - sous-test du helper contexte utilisé par la génération ;
  - vérifie `Image.Name == "variant-7.png"` ;
  - vérifie `Image.Width == "65"`.

### Suppressions parentes

Ajouts dans les suites existantes :

- `TestDeleteAltQuestionHandlerRemovesIllustratedVariantAndFile`
  - vérifie la suppression de la variante illustrée ;
  - vérifie le nettoyage du fichier associé.
- `TestDeleteQuestionHandlerRemovesVariantImageFile`
  - vérifie que la question peut être supprimée malgré une variante illustrée ;
  - vérifie le nettoyage du fichier de variante.

## 8. Cas de mauvais parent couverts

Les tests couvrent explicitement :

- question A + variante appartenant à question B → 404 sur la page de gestion ;
- question A + variante appartenant à question B → 404 sur le formulaire de suppression ;
- variante d’un utilisateur étranger → 404 ;
- le scénario positif question 42 + variante 7 + utilisateur propriétaire.

Les protections SQL existantes restent inchangées et continuent à vérifier le même triplet lors des lectures et mutations.

## 9. Chemins difficiles à tester

Le POST de redimensionnement réussi exécute réellement :

- lecture du fichier image ;
- validation des dimensions ;
- traitement OpenCV ;
- détection de cercles.

Un test de succès complet exigerait une fixture image adaptée à OpenCV et serait plus fragile que la correction ciblée. Aucun faux test ou contournement du traitement n’a été ajouté.

Le contrat de lecture préalable employé par ce POST est néanmoins le même littéral corrigé et il est protégé indirectement par :

- la lecture positive de la page de gestion ;
- les chemins de suppression ;
- les deux helpers aperçu/génération ;
- la recherche statique finale des neuf paramètres.

Les producteurs Typst eux-mêmes n’ont pas été relancés dans un nouveau test PDF : les nouveaux tests vérifient leur entrée contractuelle (`config.Question.Image`), tandis que les tests Typst existants continuent à passer.

## 10. Résultat de `go test ./...`

Réussi. Tous les packages testés passent, notamment :

```text
internal/handlers/altImages
internal/handlers/altQuestions
internal/handlers/questions
internal/handlers/tools
internal/db
```

## 11. Résultat de `go vet ./...`

Réussi, sans diagnostic.

## 12. Résultat de `git diff --check`

Réussi, sans erreur d’espace ou de format de patch.

## 13. Liste exacte des fichiers du jalon

### Fichiers applicatifs modifiés

- `internal/handlers/altImages/handlers.go`
- `internal/handlers/altQuestions/handlers.go`
- `internal/handlers/questions/handlers.go`
- `internal/handlers/tools/getAltQuestionAltAnswer.go`
- `internal/handlers/tools/getAltQuestionAltAnswerCtx.go`

### Fichiers de tests modifiés

- `internal/handlers/altQuestions/handlers_test.go`
- `internal/handlers/questions/handlers_test.go`

### Fichiers de tests créés

- `internal/handlers/altImages/handlers_test.go`
- `internal/handlers/tools/getAltQuestionAltAnswer_test.go`

### Documentation créée

- `docs/audits/alt-image-parent-contract-fix.md`

Les modifications préexistantes de `README.md` et les fichiers non suivis `app`, `reset.sh` ainsi que l’audit Images précédent n’appartiennent pas à ce jalon et n’ont pas été modifiés.
