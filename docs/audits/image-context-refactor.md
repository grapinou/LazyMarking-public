# Refactor des contextes de vue Images

## Objectif

Ce jalon migre exclusivement le contexte parental des huit pages Images vers les structures typées existantes :

```go
data.QuestionContext
data.VariantContext
```

Il ne modifie ni l’apparence, ni les règles métier, ni le stockage, ni Typst, ni le contrat parental sqlc corrigé précédemment.

## 1. PageData modifiées

### `ImagePageData`

Fichier : `internal/templates/data/image.go`.

Champ ajouté :

```go
QuestionContext QuestionContext
```

Cette PageData représente une page située directement sous une question principale. Elle ne reçoit pas `VariantContext`.

### `AltImagePageData`

Fichier : `internal/templates/data/altImage.go`.

Champs ajoutés :

```go
QuestionContext QuestionContext
VariantContext  VariantContext
```

Cette PageData représente la hiérarchie complète Question principale → Variante → Image.

Aucune PageData générique, `FamilyContext`, navigation typée ou abstraction supplémentaire n’a été créée.

## 2. Handlers GET migrés

Huit handlers GET ont été migrés.

### Images principales

1. `TableImageHandler` conserve son appel existant à `GetQuestionByID` et construit `QuestionContext` avec `question.ID` et `question.Content`.
2. `AddFormImageHandler` ne jette plus le résultat de `GetQuestionByID` et construit le même contexte.
3. `EditFormImageHandler` conserve `GetImageByQuestionID`, puis appelle explicitement `GetQuestionByID` pour le contexte de vue. La nouvelle lecture utilise `HandleOwnedLookupError`.
4. `DeleteFormImageHandler` conserve le contrôle existant de l’image, puis appelle explicitement `GetQuestionByID` pour le contexte de vue.

Les lectures supplémentaires ne remplacent aucun contrôle d’image existant.

### Images de variantes

Les quatre handlers suivants sont migrés :

1. `TableAltImageHandler` ;
2. `AddFormAltImageHandler` ;
3. `EditFormAltImageHandler` ;
4. `DeleteFormAltImageHandler`.

Chacun montre désormais explicitement :

```go
queries.GetAltQuestionByParentID(...)
queries.GetQuestionByID(...)
```

Le premier appel vérifie le triplet variante, question et utilisateur. Le second fournit le contenu de la question principale à la vue. Les résultats construisent :

```go
data.QuestionContext{ID: question.ID, Content: question.Content}
data.VariantContext{ID: altQuestion.ID, Content: altQuestion.Content}
```

Les erreurs passent par `HandleOwnedLookupError`. Aucun loader ou service ne masque les validations parentales.

## 3. Anciennes clés `ExtraData` supprimées

Onze entrées parentales ont été supprimées des maps `ExtraData`.

### Branche principale — 4 entrées

- `QuestionContent` dans la page de gestion ;
- `QuestionID` dans le formulaire d’ajout ;
- `QuestionID` dans le formulaire de modification de taille ;
- `QuestionID` dans le formulaire de suppression.

### Branche variante — 7 entrées

- `AltQuestionContent` dans la page de gestion ;
- `QuestionID` et `AltQuestionID` dans le formulaire d’ajout ;
- `QuestionID` et `AltQuestionID` dans le formulaire de modification de taille ;
- `QuestionID` et `AltQuestionID` dans le formulaire de suppression.

Les onze accès template correspondants ont également été remplacés. Il n’existe plus de double source de vérité parentale dans ces handlers ou templates.

Les données spécifiques restent dans `ExtraData`, notamment :

```text
Image
AltImage
ImageSize
NoImage
NoAltImage
AddURL
EditURL
DeleteURL
PublicImageBaseURL
```

## 4. Anciennes clés conservées

La recherche finale dans les handlers et templates `images` et `altImages` ne trouve plus aucune clé ou accès `ExtraData` nommé :

```text
QuestionContent
QuestionID
AltQuestionContent
AltQuestionID
```

Les identifiants portant ces noms dans les paramètres sqlc, les variables Go et les structures DB ne sont pas d’anciennes clés de vue : ils restent nécessaires au parsing, à l’ownership et aux mutations métier.

## 5. Usages de `QuestionContext`

`QuestionContext` est construit dans les huit handlers GET.

Dans les templates principaux :

- `table_image.html` utilise `.QuestionContext.Content` ;
- les trois formulaires utilisent `.QuestionContext.ID` dans le champ caché.

Dans les templates de variante :

- les trois formulaires utilisent `.QuestionContext.ID` ;
- la page de gestion dispose également du contexte, sans nouvel affichage puisque ce jalon ne modifie pas le rendu.

Les IDs restent des `int64`. Aucune copie string n’est placée dans les données de vue.

## 6. Usages de `VariantContext`

`VariantContext` est construit dans les quatre handlers GET de `altImages`.

Dans les templates :

- `table_alt_image.html` utilise `.VariantContext.Content` ;
- les formulaires ajout, modification et suppression utilisent `.VariantContext.ID` dans leurs champs cachés.

Le contenu de la question principale est désormais disponible sur toutes ces pages pour un jalon UX futur, mais il n’a pas été ajouté visuellement dans ce jalon.

## 7. Usages de `QuestionURL` et `VariantURL`

Les constructions simples ont été remplacées lorsque leur forme correspondait exactement aux helpers existants.

`QuestionURL` est utilisé pour les URL d’ajout, modification et suppression d’image principale, leurs redirections après mutation et le retour de la branche variante vers la liste Variantes.

`VariantURL` est utilisé pour les URL d’ajout, modification et suppression d’image de variante ainsi que leurs redirections après mutation.

Aucun nouveau helper URL n’a été créé.

## 8. Tests ajoutés ou adaptés

Aucun nouveau test HTML ou Bootstrap n’a été ajouté.

La fixture existante de `internal/handlers/altImages/handlers_test.go` a été adaptée pour fournir les métadonnées relationnelles attendues par `GetQuestionByID`. Cela permet au test positif de la page de gestion de traverser le nouveau chargement explicite de la question.

Les tests existants continuent à couvrir :

- image de variante existante ;
- mauvais parent question/variante ;
- variante étrangère ;
- formulaire de suppression avec mauvais parent ;
- suppression DB et filesystem ;
- contrat parental `GetAltImageByAltQuestionID` ;
- chemins Typst root-relative.

Les nouveaux appels `GetQuestionByID` utilisent la même gestion d’erreur 404 que les autres branches. Aucun test DB existant n’a été dupliqué.

## 9. Absence de changement métier

Ce jalon ne modifie pas l’upload, les formats, les limites, OpenCV, la validation de taille, `resize_percentage`, l’unicité, les mutations, l’ordre DB/filesystem, l’ownership, les routes publiques, SQL, les migrations ou sqlc.

Les templates ont uniquement changé leur source de données. Titres, textes, boutons, tableaux, formulaires et classes Bootstrap sont inchangés.

## 10. Correctifs techniques précédents

### Contrat parental de l’image de variante

Tous les appels existants à `GetAltImageByAltQuestionID` conservent explicitement le véritable `QuestionID`. Aucun SQL, paramètre sqlc ou contrôle parental n’a été modifié.

### Chemins d’images Typst

`typstImagePath`, `ImageSavePath` et les références root-relative `/assets/images/...` n’ont pas été modifiés. Tous leurs tests continuent à passer, y compris la compilation Typst réelle lorsqu’elle est disponible.

## 11. Résultat de `go test ./...`

Réussi. Tous les packages passent.

## 12. Résultat de `go vet ./...`

Réussi, sans diagnostic.

## 13. Résultat de `git diff --check`

Réussi, sans erreur.

## 14. Liste exacte des fichiers du jalon

### Données de templates

- `internal/templates/data/image.go`
- `internal/templates/data/altImage.go`

### Handlers

- `internal/handlers/images/handlers.go`
- `internal/handlers/altImages/handlers.go`

### Templates image principale

- `internal/templates/images/table_image.html`
- `internal/templates/images/add_form_image.html`
- `internal/templates/images/edit_form_image.html`
- `internal/templates/images/delete_form_image.html`

### Templates image de variante

- `internal/templates/altimages/table_alt_image.html`
- `internal/templates/altimages/add_form_alt_image.html`
- `internal/templates/altimages/edit_form_alt_image.html`
- `internal/templates/altimages/delete_form_alt_image.html`

### Test adapté

- `internal/handlers/altImages/handlers_test.go`

### Rapport

- `docs/audits/image-context-refactor.md`

Le working tree contenait déjà des changements des deux jalons techniques précédents ainsi que `README.md`, `app`, `reset.sh` et les audits antérieurs. Ils ont été préservés.
