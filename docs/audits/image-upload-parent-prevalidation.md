# Prévalidation du parent lors de l’ajout d’images

## 1. Ordre avant modification

Pour l’image principale comme pour l’image de variante :

```text
CheckImageFile
→ parsing des IDs et de width
→ validation de width
→ SanitizeFilename
→ SaveUploadedFile
→ ImageCircleCheck
→ CreateImage / CreateAltImage ownership-aware
```

Le premier contrôle explicite du parent intervenait donc seulement dans l’INSERT SQL, après écriture permanente et OpenCV.

## 2. Ordre après modification

Image principale :

```text
CheckImageFile
→ parsing de question_id
→ GetQuestionByID(questionID, userID)
→ parsing/validation de width
→ SanitizeFilename
→ SaveUploadedFile
→ ImageCircleCheck
→ CreateImage ownership-aware
```

Image de variante :

```text
CheckImageFile
→ parsing de question_id et alt_question_id
→ GetAltQuestionByParentID(altQuestionID, questionID, userID)
→ parsing/validation de width
→ SanitizeFilename
→ SaveUploadedFile
→ ImageCircleCheck
→ CreateAltImage ownership-aware
```

## 3. Limite conservée autour de CheckImageFile

`CheckImageFile` parse le multipart, borne le corps, valide extension/MIME/dimensions et décode la configuration avant que les champs soient exploités par les handlers. Récupérer les IDs avant cette étape demanderait une réorganisation du pipeline multipart, hors périmètre.

Un parent invalide consomme donc encore la validation initiale de l’upload, mais n’atteint plus :

- le nommage du fichier permanent ;
- l’écriture dans `assets/images` ;
- OpenCV ;
- l’INSERT.

## 4. Lookup image principale

`AddImageHandler` appelle désormais `GetQuestionByID` avec `questionID + userID` immédiatement après validation numérique de `question_id`. L’erreur passe par `HandleOwnedLookupError`.

## 5. Lookup image de variante

`AddAltImageHandler` appelle désormais `GetAltQuestionByParentID` avec les trois dimensions intactes :

- `ID: altQuestionID` ;
- `QuestionID: questionID` ;
- `UserID: userID`.

L’erreur passe également par `HandleOwnedLookupError`.

## 6. Parent absent

Une question principale absente produit HTTP 404. Le test vérifie zéro ligne `images`, aucune entrée permanente et zéro appel au contrôle OpenCV.

## 7. Parent étranger

Une question appartenant à un autre utilisateur produit HTTP 404, sans ligne, fichier permanent ni appel OpenCV.

Une variante étrangère produit le même résultat pour `alt_images`.

## 8. Mauvais parent de variante

Une variante existante rattachée à la question A, envoyée avec la question B, est rejetée par `GetAltQuestionByParentID` avec HTTP 404. Les lignes existantes restent intactes, aucun fichier n’est créé et OpenCV n’est pas exécuté.

## 9. Garde-fous INSERT conservés

`CreateImage` et `CreateAltImage` ainsi que leurs requêtes SQL n’ont pas été modifiés. Le lookup handler ne remplace pas le contrôle SQL : le parent peut théoriquement disparaître ou changer entre le SELECT et l’INSERT. Le second niveau ownership-aware protège donc toujours ce TOCTOU.

## 10. Tests de succès ajoutés

- `AddImageHandler` : upload multipart PNG valide, HTTP 303, ligne `images` associée à la question 42, largeur 25, nom physique attendu et fichier présent.
- `AddAltImageHandler` : upload multipart PNG valide, HTTP 303, ligne `alt_images` associée à la variante 7, largeur 30, nom physique attendu et fichier présent.

Le contrôle OpenCV est remplacé dans ces tests par un alias local déterministe ; toutes les autres étapes du handler restent réelles.

## 11. Tests d’échec ajoutés

Sept scénarios de test ont été ajoutés :

1. question principale absente ;
2. question principale étrangère ;
3. ajout principal réussi ;
4. INSERT principal refusé par unicité avec nettoyage ;
5. variante associée à une autre question ;
6. variante étrangère ;
7. ajout variante réussi.

Les quatre rejets de parent vérifient explicitement que l’alias OpenCV n’est jamais appelé.

## 12. INSERT échoué et nettoyage

Le test principal place une image existante sur une question valide, puis envoie un second fichier avec un nom distinct. La prévalidation réussit, le nouveau fichier est écrit, puis la contrainte d’unicité fait échouer l’INSERT. Le handler conserve sa tentative de nettoyage : une seule ligne DB reste et le nouveau fichier a disparu.

## 13. Éléments hors périmètre inchangés

Aucun template, SQL/sqlc, scanner, handler de suppression, format, taille multipart, validation MIME, dimension, nommage, fonction de sauvegarde, stockage ou implémentation OpenCV n’a été modifié.

## 14. Validations

- `go test ./...` : réussi.
- `go vet ./...` : réussi.
- `git diff --check` : réussi.

## 15. Fichiers modifiés pour ce jalon

1. `internal/handlers/images/handlers.go`
2. `internal/handlers/images/handlers_test.go`
3. `internal/handlers/altImages/handlers.go`
4. `internal/handlers/altImages/handlers_test.go`
5. `docs/audits/image-upload-parent-prevalidation.md`
