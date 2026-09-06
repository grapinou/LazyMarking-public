# Audit ciblé — Images des questions et des variantes

Point critique découvert : la requête `GetAltImageByAltQuestionID` exige désormais `QuestionID`, mais tous ses appelants l’omettent. Sa valeur reste donc `0`. Cela casse actuellement la relecture des images de variantes, leur édition/suppression, leur aperçu/génération et certains nettoyages lors de suppressions parentes.

## 1. Architecture de l’image principale

La branche est :

```text
utilisateur
  └── question
        └── image (0 ou 1)
```

Routes :

- `GET /dashboard/questions/images` : page de gestion.
- `GET/POST /dashboard/questions/images/add` : formulaire et création.
- `GET/POST /dashboard/questions/images/edit` : formulaire et modification du pourcentage.
- `GET/POST /dashboard/questions/images/delete` : confirmation et suppression.

Elles sont enregistrées dans `internal/handlers/images/routes.go` et définies dans `internal/templates/data/image.go`.

### Page de gestion

`TableImageHandler` :

1. parse `question_id` en `int64` ;
2. appelle explicitement `GetQuestionByID(questionID, userID)` ;
3. appelle `GetImageByQuestionID(questionID, userID)` ;
4. traite `sql.ErrNoRows` comme l’état sans image ;
5. construit les URL ajout/édition/suppression ;
6. fournit l’image et le contenu de la question au template.

Données actuelles :

```text
UserID
QuestionContent
NoImage
Image
AddURL
EditURL
DeleteURL
PublicImageBaseURL
```

Seule la page de gestion reçoit `QuestionContent`. Les formulaires ajout, redimensionnement et suppression ne le reçoivent pas.

### Ajout

L’ajout reçoit un véritable fichier multipart et un pourcentage `width`.

Ordre des opérations :

1. validation et décodage de l’upload ;
2. parsing du `question_id` et du pourcentage ;
3. validation des dimensions redimensionnées ;
4. génération du nom ;
5. sauvegarde exclusive sur disque ;
6. contrôle OpenCV des cercles ;
7. insertion DB conditionnée par l’ownership de la question ;
8. suppression du fichier si l’insertion échoue ou affecte un nombre incorrect de lignes ;
9. redirection vers la page Image.

Le POST ne charge pas explicitement la question avant de sauvegarder. L’ownership final est garanti par `CreateImage`, mais un upload appartenant à un mauvais `question_id` est donc traité et temporairement écrit avant d’être rejeté et nettoyé.

### « Modification »

Il n’existe aucun remplacement du fichier.

La route `edit` modifie uniquement :

```text
resize_percentage
```

Elle relit le fichier existant, valide ses dimensions, relance le contrôle de cercles avec la nouvelle taille, puis exécute `UpdateSizeImage`.

Le futur vocabulaire devrait donc être « Modifier la taille » ou « Ajuster l’affichage », et non « Remplacer l’image ».

### Suppression

Le handler :

1. relit la ligne image avec `question_id + user_id` ;
2. supprime la ligne DB ;
3. vérifie que exactement une ligne a été supprimée ;
4. tente ensuite de supprimer le fichier ;
5. journalise l’échec filesystem mais redirige quand même.

La suppression DB précède donc la suppression physique.

## 2. Architecture de l’image de variante

La relation attendue est :

```text
utilisateur
  └── question
        └── variante
              └── image (0 ou 1)
```

Routes :

- `GET /dashboard/questions/altquestions/altimages` : page de gestion, via `DefaultAltQuestionRoutes.AltImageURL`.
- `GET/POST /dashboard/altquestions/altimages/add`
- `GET/POST /dashboard/altquestions/altimages/edit`
- `GET/POST /dashboard/altquestions/altimages/delete`

Les chemins add/edit/delete sont légèrement incohérents avec le chemin de la page de gestion : ils omettent `/questions`.

### Validation des parents

Les GET liste/ajout/édition/suppression parsèrent tous :

```text
question_id
alt_question_id
```

et appellent `GetAltQuestionByParentID`, ce qui vérifie simultanément :

- l’ID de la variante ;
- son `question_id` ;
- son `user_id` ;
- l’existence d’une question parente appartenant au même utilisateur.

En revanche, aucun handler `altImages` n’appelle `GetQuestionByID`. Ils connaissent donc le lien parental, mais pas le contenu de la question principale.

Les POST :

- création : repose sur `CreateAltImage`, qui exige variante, question et utilisateur cohérents ;
- redimensionnement : repose sur `UpdateSizeAltImage` ;
- suppression : repose sur `DeleteAltImage`.

### Données actuelles

La page de gestion reçoit :

```text
UserID
AltQuestionContent
NoAltImage
AltImage
AltQuestionURL
AddURL
EditURL
DeleteURL
PublicImageBaseURL
```

Elle connaît le contenu de la variante, mais pas celui de la question principale.

Les formulaires ajout, redimensionnement et suppression ne connaissent que :

```text
QuestionID
AltQuestionID
```

Le formulaire de redimensionnement reçoit aussi `ImageSize`.

Ainsi :

- page liste : variante visible, question principale absente ;
- ajout : aucun contenu parental ;
- redimensionnement : aucun contenu parental ;
- suppression : aucun contenu parental.

Le retour vers les variantes existe uniquement sur la page de gestion via `AltQuestionURL`. Les trois formulaires n’ont aucun bouton Annuler/Retour.

### Défaut actuel de relecture

`db/query/altImages.sql` impose désormais à `GetAltImageByAltQuestionID` :

```sql
alt_question_id
user_id
question_id
```

La structure générée confirme ces trois champs. Pourtant tous les appels transmettent seulement :

```go
AltQuestionID: altQuestionID,
UserID:        userID,
```

`QuestionID` vaut donc zéro.

Conséquences :

- la liste considère normalement qu’aucune image n’existe ;
- le formulaire de redimensionnement retourne une erreur ;
- le POST de redimensionnement ne retrouve pas l’image ;
- la confirmation et le POST de suppression ne la retrouvent pas correctement ;
- l’aperçu et la génération de variante perdent l’image ;
- la suppression d’une variante peut laisser son fichier orphelin ;
- la suppression d’une question ayant une variante illustrée peut échouer avant suppression.

Ce défaut préexiste à l’audit et doit être traité avant ou avec le prochain jalon.

## 3. Relations DB et ownership

Les tables définissent :

```text
images.question_id         UNIQUE NOT NULL
alt_images.alt_question_id UNIQUE NOT NULL
```

Il y a donc structurellement zéro ou une image par question/variante.

Les clés étrangères utilisent `ON DELETE CASCADE`.

Les requêtes ajoutent des contrôles applicatifs :

- `CreateImage`, `GetImageByQuestionID`, `UpdateSizeImage`, `DeleteImage` vérifient question + utilisateur ;
- les quatre équivalents variante vérifient variante + question + utilisateur ;
- `GetAltQuestionByParentID` vérifie directement le bon parent.

La migration `0030_enforce_user_relation_ownership.sql` ajoute également des triggers empêchant les insertions ou changements de parent entre utilisateurs.

La protection existe donc à trois niveaux :

```text
handler/session
  ↓
requêtes SQL avec user_id et parent
  ↓
FK + triggers d’ownership
```

## 4. Fonctionnement filesystem

Configuration :

- stockage : `assets/images`
- URL d’affichage : `/static/images/{filename}`
- chemin Typst relatif : `../../images/{filename}`

### Nommage

`SanitizeFilename` construit :

```text
{userID}_{username}_{mainQuestion|altQuestion}_{ID}_{basenameOriginal}
```

L’extension est normalisée et limitée à `.jpg`, `.jpeg`, `.png`.

Le basename retire les chemins éventuellement fournis dans le nom original. Le nom final est ensuite vérifié comme composant de chemin unique par `SaveUploadedFile`.

### Écriture

`SaveUploadedFile` :

- refuse les noms comportant séparateur, `.`/`..` ou NUL ;
- vérifie chaque répertoire avec `Lstat` ;
- refuse les répertoires parents symboliques ;
- utilise `O_EXCL` : aucun fichier existant n’est écrasé ;
- mode fichier `0640`, répertoire `0750` ;
- supprime tout fichier partiellement écrit en cas d’erreur.

### Lecture

`ReadImageConfig`, `ImageCircleCheck` et le serveur d’image :

- refusent les chemins non sûrs ;
- refusent les symlinks finaux ;
- exigent un fichier régulier ;
- utilisent `SameFile` après ouverture pour réduire les risques de substitution entre validation et ouverture.

### Suppression

`RemoveStoredImageFile` :

- refuse la traversée ;
- refuse un répertoire d’images symbolique ;
- ne suit pas un symlink final : `os.Remove` retire le lien, pas sa cible ;
- considère un fichier absent comme déjà supprimé avec succès.

## 5. Données disponibles par écran

| Écran | Question principale | Variante | Image |
|---|---|---|---|
| Gestion image principale | contenu + ID parsé | — | ligne complète ou état vide |
| Ajout principale | ID seulement | — | nouvel upload |
| Taille principale | ID seulement | — | pourcentage uniquement |
| Suppression principale | ID seulement | — | existence vérifiée, non affichée |
| Gestion image variante | absente visuellement | contenu + ID | ligne attendue ou état vide |
| Ajout variante | ID seulement | ID seulement | nouvel upload |
| Taille variante | ID seulement | ID seulement | pourcentage uniquement |
| Suppression variante | ID seulement | ID seulement | existence vérifiée, non affichée |

## 6. Problèmes UX actuels

Les deux branches utilisent encore l’ancienne interface :

- tableaux pour une ressource unique ;
- actions uniquement iconographiques sans libellé accessible explicite ;
- « Back to question » et « Back to alt question » en anglais ;
- « alt image », « question alt » et « image alternative » comme termes techniques ;
- titres « Ajouter une image » même lorsqu’une image existe ;
- bouton Ajouter toujours visible, bien que la DB n’autorise qu’une image ;
- « Editer taille de l’image » et fautes telles que « redimmensionnent » / « saisisser » ;
- confirmation « C’est mon dernier mot » ;
- tutoiement et formulation « Es-tu sur » ;
- aucun bouton Annuler dans les formulaires ;
- aucune image affichée dans les écrans de redimensionnement ou suppression ;
- aucun contexte parental sur les formulaires ;
- question principale absente de toute la branche image de variante ;
- attribut HTML `accept=".jpg, .png, .svg"` incohérent avec le backend, qui refuse SVG ;
- aucune indication claire que le pourcentage concerne la taille imprimée/générée.

## 7. État vide, image existante et remplacement

Le métier réel est :

- question principale : zéro ou une image ;
- variante : zéro ou une image ;
- ajout quand aucune image n’existe ;
- modification du pourcentage quand elle existe ;
- suppression du fichier et de la relation ;
- aucun remplacement direct.

Quand une image existe déjà :

- la page montre toujours le bouton Ajouter ;
- un nouvel ajout ne remplace rien ;
- avec le même nom généré, `O_EXCL` rejette l’écriture ;
- avec un autre nom, le fichier est écrit puis l’unicité DB rejette l’insertion, après quoi le handler tente de nettoyer le nouveau fichier.

Le vocabulaire « Ajouter ou remplacer » ne correspond donc pas au métier actuel. Sans changement métier, la cible correcte est :

- état vide : « Ajouter une image » ;
- image existante : « Modifier la taille » et « Supprimer l’image » ;
- pas de bouton Remplacer.

## 8. Intégration de `QuestionContext`

`ImagePageData` devrait recevoir :

```go
QuestionContext data.QuestionContext
```

Pour obtenir le contenu sur tous les écrans sans changer le métier :

- liste : réutiliser le `GetQuestionByID` existant ;
- ajout : conserver le résultat actuellement ignoré ;
- redimensionnement et suppression : ajouter une lecture explicite de la question si l’UX doit afficher son contenu.

Les clés supprimables seraient :

```text
QuestionID
QuestionContent
```

`Image`, `ImageSize`, `NoImage`, les URL et `PublicImageBaseURL` peuvent rester spécifiques à la page.

`QuestionURL` couvre naturellement :

- page de gestion ;
- ajout ;
- redimensionnement ;
- suppression ;
- retour après mutation.

## 9. Intégration de `VariantContext`

`AltImagePageData` devrait recevoir :

```go
QuestionContext data.QuestionContext
VariantContext  data.VariantContext
```

`GetAltQuestionByParentID` fournit déjà ID et contenu de la variante. Pour obtenir `QuestionContext`, les handlers devront conserver un appel explicite à `GetQuestionByID`.

Les clés supprimables seraient :

```text
QuestionID
AltQuestionID
AltQuestionContent
```

`VariantURL` couvre naturellement :

- page de gestion de l’image ;
- ajout ;
- redimensionnement ;
- suppression ;
- redirections après mutation.

Le retour vers la liste Variantes reste une URL à un seul ID et relève donc de `QuestionURL`.

Aucune abstraction supplémentaire n’est nécessaire.

## 10. Protections de sécurité actuelles

### Upload

- requête limitée à 24 Mio ;
- fichier limité à 20 Mio ;
- multipart en mémoire limité à 1 Mio avant stockage temporaire ;
- suppression des fichiers multipart temporaires ;
- fichier vide refusé ;
- dimensions maximales : 8000 × 8000 ;
- maximum 25 millions de pixels ;
- extension `.png`, `.jpg` ou `.jpeg` seulement ;
- détection MIME par contenu ;
- décodage réel via les bibliothèques image ;
- cohérence extension/MIME/format décodé ;
- contrôle des dimensions après redimensionnement ;
- rejet de NaN, infini, zéro et tailles excessives ;
- contrôle OpenCV des cercles incompatibles.

### Filesystem

- basename du fichier original ;
- validation stricte du composant de chemin ;
- prévention de la traversée ;
- refus des parents symboliques ;
- écriture exclusive sans écrasement ;
- suppression ciblée ;
- lecture limitée aux fichiers réguliers ;
- accès HTTP authentifié ;
- vérification DB `UserOwnsImage` avant livraison du fichier.

### DB

- unicité par parent ;
- clés étrangères ;
- triggers inter-utilisateurs ;
- ownership répété dans les requêtes ;
- mutations `execrows` ;
- contrôles `rows == 0` et `rows > 1`.

## 11. Trous réels

Prioritaires :

1. `QuestionID` absent de tous les appels `GetAltImageByAltQuestionID`.
2. L’aperçu et la génération de variante sont affectés par le même défaut.
3. Les suppressions parentes utilisant cette requête peuvent échouer ou laisser un fichier orphelin.
4. Le HTML annonce SVG alors que le backend le refuse.
5. Aucune opération de remplacement n’existe malgré une UX qui laisse réessayer Ajouter.
6. DB supprimée puis suppression physique échouée : fichier orphelin, erreur seulement journalisée.
7. Une ligne DB dont le fichier manque produit une image cassée sur la liste et une erreur 500 lors du redimensionnement.
8. Un fichier sans ligne DB reste sur disque. Il n’est pas servi par la route authentifiée, mais aucun mécanisme ne le réconcilie ou ne le purge.
9. Aucun test handler ne couvre la branche `altImages`.
10. Les formulaires GET image principale d’édition/suppression reposent sur la requête image pour l’ownership, mais ne chargent pas la question : suffisant pour la sécurité, insuffisant pour le futur contexte visuel.
11. Le nom intègre `username`. Un caractère interdit n’autorise pas une traversée, car la sauvegarde le rejette, mais il peut rendre l’upload impossible pour certains usernames.
12. L’upload est validé avant le contrôle du parent dans les POST : un parent invalide consomme tout le traitement image avant rejet SQL.

## 12. Tests existants

### Handlers

- `images/handlers_test.go` vérifie seulement qu’un redimensionnement d’image absente retourne 404.
- Aucun test handler pour `altImages`.

### DB

- création image principale pour parent possédé ;
- rejet de création sur question étrangère ;
- rejet du redimensionnement principal étranger ;
- suppression principale absente → zéro ligne ;
- rejet du redimensionnement variante avec mauvais parent ;
- comptage 1 puis 0 pour `DeleteImage` et `DeleteAltImage` ;
- cascades DB question/variante ;
- triggers empêchant les relations inter-utilisateurs.

### Upload et filesystem

- PNG/JPEG valides ;
- faux contenu et incohérence extension/format ;
- extension interdite ;
- limite fichier et limite requête ;
- dimensions et redimensionnement ;
- nettoyage multipart ;
- refus des symlinks ;
- refus de traversée ;
- non-écrasement ;
- nettoyage d’une écriture partielle ;
- suppression ciblée ;
- fichier absent accepté ;
- protection d’un répertoire symbolique.

## 13. Tests manquants utiles

Priorité élevée :

- test handler `altImages` avec bon et mauvais `question_id` ;
- test qui aurait détecté l’absence de `QuestionID` dans `GetAltImageByAltQuestionID` ;
- liste variante avec image existante ;
- redimensionnement variante réussi ;
- suppression variante réussie et mauvais parent ;
- création variante : parent correct, étranger et mauvais parent ;
- création principale avec question étrangère et vérification du nettoyage du fichier ;
- suppression principale/variante vérifiant DB et filesystem ensemble ;
- ligne DB avec fichier absent ;
- échec de suppression filesystem après suppression DB ;
- ajout lorsqu’une image existe déjà, avec vérification qu’aucun fichier supplémentaire ne reste ;
- route `/static/images/{filename}` : propriétaire, autre utilisateur, fichier absent et ligne DB absente.

Moins prioritaire :

- test direct de `SanitizeFilename` avec basename, extensions et username problématique ;
- test explicite du contrôle OpenCV, potentiellement coûteux et dépendant de fixtures maîtrisées.

Aucun test Bootstrap/HTML n’est nécessaire.

## 14. Proposition UX : image principale

```text
Image de la question

[Retour à la banque de questions]

Question principale
┌────────────────────────────────────┐
│ Énoncé complet                     │
└────────────────────────────────────┘

Image
┌────────────────────────────────────┐
│ État vide :                        │
│ Aucune image associée.             │
│ [Ajouter une image]                │
│                                    │
│ ou image existante :               │
│ [Aperçu de l’image]                │
│ Taille utilisée : 50 %             │
│ [Modifier la taille] [Supprimer]   │
└────────────────────────────────────┘
```

Formulaire d’ajout :

- formats réellement acceptés : PNG, JPG, JPEG ;
- champ fichier requis ;
- « Taille de l’image dans le document (%) » ;
- boutons Annuler / Ajouter l’image.

Formulaire de taille :

- contexte question ;
- aperçu actuel ;
- valeur actuelle ;
- Annuler / Enregistrer.

Suppression :

- contexte question et aperçu ;
- confirmation professionnelle ;
- Annuler / Supprimer l’image.

## 15. Proposition UX : image de variante

```text
Image de la variante

[Retour aux variantes]

Question principale
┌────────────────────────────────────┐
│ Énoncé principal                   │
└────────────────────────────────────┘
                 ↓
Variante
┌────────────────────────────────────┐
│ Formulation alternative            │
└────────────────────────────────────┘
                 ↓
Image
┌────────────────────────────────────┐
│ État vide ou aperçu actuel         │
│ Taille utilisée                    │
│ Actions adaptées à l’état          │
└────────────────────────────────────┘
```

Les actions et formulaires seraient identiques à la branche principale. Seule la double hiérarchie parentale change.

## 16. Estimation des modifications

Pour une harmonisation sans changement métier :

- 2 structures PageData ;
- 2 packages de handlers ;
- 8 templates actuels ;
- tests handlers des deux branches ;
- remplacement local des constructions d’URL par `QuestionURL` / `VariantURL` ;
- correction de tous les appelants de `GetAltImageByAltQuestionID`, y compris aperçu, génération et suppressions parentes.

Aucun SQL, service, repository, loader ou partial n’est nécessaire.

La partie visuelle est petite à moyenne. La correction et la couverture du contrat `GetAltImageByAltQuestionID` sont plus sensibles, car elles traversent gestion, suppression, aperçu et génération.

## 17. Découpage proposé

1. **Rétablir le contrat de lecture des images de variante**
   Fournir `QuestionID` à tous les appels, ajouter les tests de mauvais parent, aperçu/génération et suppressions parentes.

2. **Introduire les contextes dans les PageData Image**
   `QuestionContext` pour `ImagePageData`, puis `QuestionContext + VariantContext` pour `AltImagePageData`. Conserver les chargements DB explicites.

3. **Harmoniser les pages de gestion**
   Remplacer les tableaux par une carte de ressource unique, sans modifier les opérations.

4. **Harmoniser ajout et redimensionnement**
   Contextes parentaux, libellés français, formats exacts, boutons Annuler. Conserver la distinction ajout/taille.

5. **Harmoniser les confirmations de suppression**
   Aperçu et contexte, formulation professionnelle, comportement DB/filesystem inchangé.

6. **Renforcer uniquement les tests d’intégration utiles**
   Ownership, mauvais parent, rows affected, nettoyage filesystem, fichier manquant et non-remplacement.

7. **Traiter séparément la divergence DB/filesystem si souhaité**
   Une éventuelle stratégie transactionnelle ou de réconciliation est un changement de robustesse métier distinct ; elle ne devrait pas être dissimulée dans le refactor UX.
