# Correction des chemins d’images dans les documents Typst

## 1. Cause exacte du bug

Les producteurs Typst construisaient chaque référence d’image à partir de la constante :

```go
ImagePathTypst = "../../images/"
```

Une prévisualisation est désormais écrite dans :

```text
assets/tmp/<username>/preview-<uuid>/<document>.typ
```

Typst résout un chemin relatif depuis le dossier du document source. Depuis ce dossier, `../../images/<nom>` désigne :

```text
assets/tmp/images/<nom>
```

alors que le fichier réel se trouve dans :

```text
assets/images/<nom>
```

Le chargement DB de l’image était correct après le jalon précédent. L’échec intervenait exclusivement lors de la résolution filesystem effectuée par Typst.

## 2. Ancien modèle de répertoires supposé

Le préfixe fixe `../../images/` supposait implicitement que chaque document Typst se trouvait exactement deux niveaux sous un répertoire dont `images` était un enfant direct.

Cette hypothèse n’était représentée ni par le chemin réel du document ni par le `--root` fourni au compilateur. Elle était donc fragile dès que la profondeur d’un workspace changeait.

Remplacer simplement ce préfixe par `../../../images/` aurait corrigé la profondeur actuelle, mais aurait recréé la même dépendance à un nombre fixe de parents.

## 3. Modèle actuel des workspaces

Les workspaces d’opération sont créés par `CreateOperationTempDir` sous :

```text
assets/tmp/<username>/<operation>/
```

Les opérations observées sont notamment :

- `preview-<uuid>` pour les aperçus question, variante et QCM ;
- `exam-<id>` pour la génération normale ;
- `mini-<uuid>` pour les mini-QCM ;
- `marking-<id>` pour la reconstruction utilisée pendant la correction.

Les fichiers Typst sont aujourd’hui directement dans ces dossiers, mais la correction doit également fonctionner dans une profondeur future telle que :

```text
assets/tmp/<username>/<operation>/subdir/document.typ
```

Le stockage des images reste :

```text
assets/images/<filename>
```

## 4. Utilisations de `ImagePathTypst`

Trois utilisations ont été trouvées :

1. `internal/handlers/tools/typstWriter.go`, dans `TypstWriter` ;
2. `internal/handlers/tools/typstWriterLandscape.go`, dans `TypstWriterLandscape` ;
3. `internal/handlers/tools/typstLandscapeContent.go`, dans `TypstLandscapeContent`.

Des assertions de tests dans `typstProducerEscape_test.go` encodaient également l’ancienne chaîne `../../images/...`.

Après migration, une recherche globale ne trouve plus aucune occurrence de `ImagePathTypst`. La constante a donc été supprimée de `internal/config/config.go` afin d’éviter deux sources de vérité.

## 5. Producteurs Typst concernés

### `TypstWriter`

Producteur portrait commun à :

- aperçu d’une question principale ;
- aperçu d’une variante ;
- aperçu QCM portrait ;
- génération normale d’un examen ;
- reconstruction d’un examen pendant le marquage.

Pour `ExamQCM`, le fichier est créé avec un nom temporaire :

```text
assets/tmp/<username>/<operation>/student-exam-<suffixe>.typ
```

Pour les aperçus, il est nommé à partir du username et du type de document.

### `TypstWriterLandscape`

Producteur de l’aperçu QCM paysage :

```text
assets/tmp/<username>/preview-<uuid>/<username>_qcm_landscape.typ
```

### `TypstLandscapeContent`

Produit les fragments de questions illustrées destinés au mini-QCM. Ces fragments sont ensuite assemblés par `TypstWriterLandscapeAllContent` dans :

```text
assets/tmp/<username>/mini-<uuid>/<username>_miniqcm_landscape.typ
```

Ce producteur ne connaît pas le chemin final du document. La solution root-relative lui permet d’utiliser exactement la même règle que les deux producteurs de fichiers, sans lui transmettre artificiellement un chemin de destination.

## 6. Parcours concernés

| Parcours | Producteur | Emplacement actuel du `.typ` | Image réelle | Référence Typst produite |
|---|---|---|---|---|
| Aperçu question principale | `TypstWriter` | `assets/tmp/<user>/preview-<uuid>/<user>_question_preview.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Aperçu variante | `TypstWriter` | `assets/tmp/<user>/preview-<uuid>/<user>_question_preview.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Aperçu QCM portrait | `TypstWriter` | `assets/tmp/<user>/preview-<uuid>/<user>_qcm_preview.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Aperçu QCM paysage | `TypstWriterLandscape` | `assets/tmp/<user>/preview-<uuid>/<user>_qcm_landscape.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Examen normal | `TypstWriter` | `assets/tmp/<user>/exam-<id>/student-exam-<suffixe>.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Reconstruction de marquage | `TypstWriter` | `assets/tmp/<user>/marking-<id>/student-exam-<suffixe>.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |
| Mini-QCM | `TypstLandscapeContent` puis `TypstWriterLandscapeAllContent` | `assets/tmp/<user>/mini-<uuid>/<user>_miniqcm_landscape.typ` | `assets/images/<nom>` | `/assets/images/<nom>` |

Dans tous les cas, `CompileTypst` exécute :

```text
typst compile --root <projectRoot> <document.typ> <document.pdf>
```

Avec `<projectRoot>` égal à la racine LazyMarking, `/assets/images/<nom>` désigne donc toujours le fichier réel `<projectRoot>/assets/images/<nom>`.

## 7. Solution choisie

Une petite fonction pure a été ajoutée :

```go
func typstImagePath(imageName string) (string, error)
```

Elle :

1. valide que `imageName` est un composant de chemin unique avec la règle filesystem existante `safePathComponent` ;
2. vérifie que `config.ImageSavePath` est un chemin relatif situé dans la racine projet ;
3. joint `config.ImageSavePath` et le nom ;
4. normalise les séparateurs avec `filepath.ToSlash` ;
5. préfixe le résultat par `/` pour produire un chemin absolu au sens du projet Typst.

Résultat :

```text
/assets/images/<filename>
```

Typst 0.15.1 documente `--root` comme configurant « the project root (for absolute paths) ». Un test réel confirme ce comportement dans l’environnement.

Les trois producteurs appellent désormais exclusivement cette fonction.

## 8. Robustesse à la profondeur du workspace

Le chemin commence par `/` dans le document Typst. Il n’est donc pas résolu relativement à `filepath.Dir(document.typ)` mais relativement au projet configuré par `--root`.

Les quatre emplacements suivants produisent exactement la même référence :

```text
assets/tmp/Sighto/preview-123/document.typ
assets/tmp/Sighto/exam-42/student-exam.typ
assets/tmp/Sighto/mini-123/Sighto_mini.typ
assets/tmp/Sighto/operation/subdir/document.typ
```

Ajouter ou retirer des niveaux au workspace ne modifie plus la résolution.

Le nom d’image ne peut pas injecter `..`, `/`, `\` ou un chemin absolu. Cette validation complète les garanties existantes de création et de stockage des noms.

## 9. Fichiers modifiés

Fichiers applicatifs modifiés ou créés par ce jalon :

- `internal/config/config.go`
- `internal/handlers/generateExams/handlers.go`
- `internal/handlers/tools/typstWriter.go`
- `internal/handlers/tools/typstWriterLandscape.go`
- `internal/handlers/tools/typstLandscapeContent.go`
- `internal/handlers/tools/typstImagePath.go` — créé

Tests modifiés ou créés :

- `internal/handlers/tools/typstProducerEscape_test.go`
- `internal/handlers/tools/typstImagePath_test.go` — créé

Documentation créée :

- `docs/audits/typst-image-path-fix.md`

Les fichiers déjà modifiés par le jalon précédent sur le contrat parental des images de variante restent présents dans le working tree mais ne font pas partie de cette correction de chemin Typst. `README.md`, `app` et `reset.sh` étaient également déjà modifiés ou non suivis.

## 10. Tests unitaires ajoutés

### Stabilité selon la profondeur

`TestTypstImagePathIsStableAcrossWorkspaceDepths` couvre :

- workspace aperçu ;
- workspace examen ;
- workspace mini ;
- niveau supplémentaire `operation/subdir`.

Pour chaque cas, le résultat est :

```text
/assets/images/test.png
```

et sa résolution depuis la racine Typst correspond à :

```text
assets/images/test.png
```

### Sécurité du nom

`TestTypstImagePathRejectsPathComponents` refuse :

- `../escape.png` ;
- `subdir/image.png` ;
- `subdir\image.png` ;
- le nom vide.

### Producteurs

Les tests existants adaptés vérifient qu’une question illustrée produit `/assets/images/image-name.png` dans :

- `TypstWriter` ;
- `TypstWriterLandscape` ;
- `TypstLandscapeContent`.

La signature de `TypstLandscapeContent` retourne désormais une erreur. `GenerateMiniPDFHandler` la propage dans son canal d’erreurs au lieu d’ignorer un nom non résolvable.

## 11. Test Typst réel

`TestTypstCompilesRootRelativeImageFromNestedWorkspace` :

1. détecte `typst` avec `exec.LookPath` et se désactive proprement s’il est absent ;
2. crée une image PNG contrôlée et unique sous `assets/images` ;
3. crée un document sous un workspace temporaire imbriqué `assets/tmp/.../operation/subdir` ;
4. utilise exactement `typstImagePath` pour construire la référence ;
5. lance réellement Typst avec `--root` sur la racine du dépôt ;
6. vérifie que le PDF existe et n’est pas vide ;
7. nettoie automatiquement l’image et le workspace temporaires.

Résultat avec `typst 0.15.1 (9dfd3a08)` : réussi.

Le test a aussi confirmé qu’une racine de projet arbitraire entièrement placée dans le répertoire temporaire système ne reproduisait pas correctement les conditions de lancement de l’exécutable installé. Le test final utilise donc exactement la racine et le modèle de workspace de production.

## 12. Résultat aperçu question illustrée

Le parcours charge l’image dans `config.Question.Image`, appelle `TypstWriter`, puis `CompileTypst`. `TypstWriter` émet maintenant `/assets/images/<nom>` et le test Typst réel confirme que cette forme compile depuis un workspace plus profond que celui d’un aperçu réel.

Le contrat de compilation d’une question illustrée est donc rétabli. Aucun handler d’aperçu ni contrat DB n’a été modifié.

## 13. Résultat aperçu variante illustrée

L’aperçu de variante utilise le même `TypstWriter` après construction de la variante par `GetAltQuestionAltAnswer`. Le jalon précédent rétablit le chargement de son image ; ce jalon rétablit maintenant la résolution du fichier par Typst.

La référence produite est identique à celle d’une question principale : `/assets/images/<nom>`. Aucun SQL ou paramètre image n’a été modifié.

## 14. Impact QCM portrait et paysage

- Portrait : `PreviewQCMHandler` passe par `TypstWriter`, migré.
- Paysage : `PreviewQCMLandscapeHandler` passe par `TypstWriterLandscape`, migré.

Les tests producteurs vérifient une question illustrée dans les deux formes. Le chemin ne dépend plus du dossier `preview-<uuid>`.

## 15. Impact génération examen et mini

- Examen normal et reconstruction de marquage : `TypstWriter` crée ses `student-exam-*.typ` sous les workspaces `exam-*` ou `marking-*`. La référence root-relative est identique.
- Mini-QCM : `TypstLandscapeContent` produit désormais la même référence stable avant que `TypstWriterLandscapeAllContent` n’assemble le document final sous `mini-*`.

Le test de profondeur couvre explicitement les formes examen et mini, et le test Typst réel utilise une profondeur encore supérieure.

## 16. Résultat de `go test ./...`

Réussi. Tous les packages passent, y compris le test de compilation avec Typst 0.15.1.

## 17. Résultat de `go vet ./...`

Réussi, sans diagnostic.

## 18. Résultat de `git diff --check`

Réussi, sans erreur d’espace ou de format de patch.
