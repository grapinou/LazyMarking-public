# Accès stable au PDF historique d'une génération Exam

Date : 30 août 2026

## 1. Cause du bug

Le PDF final était produit avec `examGenerationPDFName(username, ExamName, ClassName)`. La page `success` reconstruisait ensuite ce même nom à partir de `GetExamNameAndClassCodeName`.

L'Exam et son `class_code_id` sont immuables après génération, mais `class_codes.name` reste modifiable. Après renommage de la classe, la page construisait donc une URL vers un nouveau nom inexistant alors que le PDF historique conservait son ancien nom.

## 2. Stratégie choisie

Un helper ciblé, `tools.ResolveExamGenerationPDFName`, résout désormais le PDF réellement présent à partir de :

- l'utilisateur authentifié ;
- `generation_id` ;
- le workspace stable `exam-<generation_id>`.

Il retourne uniquement le nom du fichier trouvé. Le handler success construit ensuite l'URL existante avec l'opération stable et ce nom réel. `ClassName` et `ExamName` restent disponibles pour l'affichage, mais ne participent plus à la résolution technique.

## 3. Contrat réel du workspace success

Pendant la génération, un PDF est produit par élève. Après fusion :

1. `MergePdf` crée le PDF final ;
2. `cleanupExamGenerationFiles` supprime la liste des PDF élèves utilisée comme entrée ;
3. le même cleanup supprime les fichiers `*.png` et `*.typ` ;
4. seulement ensuite `CompleteExamGeneration` passe la génération à `success`.

Le contrat normal d'un workspace success est donc un unique PDF final utile. Le resolver applique strictement ce contrat au lieu de choisir arbitrairement un fichier.

## 4. Compatibilité avec les PDF historiques

La génération future conserve son schéma de nommage actuel. Aucun fichier historique n'est renommé.

Comme le resolver inspecte le workspace stable et non les libellés DB vivants, un PDF portant l'ancien nom construit avec l'ancien `ExamName`/`ClassName` reste accessible. Un test fonctionnel place explicitement cet ancien artefact, renomme la classe en DB, recharge success et vérifie que le même contenu est servi sans création d'un second PDF.

## 5. Dépendance au ClassName supprimée

Oui pour l'identité et la résolution du fichier. `ClassName` reste une métadonnée vivante affichée dans la page success.

## 6. Dépendance technique au ExamName supprimée

Oui pour l'identité et la résolution du fichier. `ExamName` reste affiché et continue d'être utilisé par la génération future pour produire un nom humain, sans être requis pour retrouver l'artefact ensuite.

## 7. Migration

Aucune migration, aucune modification SQL et aucune régénération sqlc.

## 8. Ownership

Le flux conserve son ordre de preuve :

1. `GetExamStatus(generation_id, user_id)` vérifie que la génération appartient à l'utilisateur ;
2. seul `status = success` entre dans le chemin success ;
3. `GetExamNameAndClassCodeName` revérifie génération, Exam, classe et autres parents avec le même `user_id` ;
4. le resolver reçoit ensuite le `username` authentifié et ne regarde que `assets/tmp/<username>/exam-<generation_id>`.

La résolution filesystem ne remplace aucune vérification DB. Le test de resolver vérifie également qu'un workspace de Bob n'est pas résolu pour Alice.

## 9. Sécurité du chemin

Les protections existantes du téléchargement restent inchangées : authentification, validation des composants, extension PDF, confinement utilisateur/workspace, refus des symlinks, fichier régulier, validation de l'arbre, puis contrôle `os.SameFile` lors du service.

Le resolver utilise les mêmes primitives internes `operationTempDir`, `ensureDirectoryTree` et `safePathComponent`. Il :

- construit l'opération uniquement depuis un `int64` positif ;
- refuse un arbre de répertoires non sûr ;
- ignore les entrées non PDF ;
- ignore les symlinks et les entrées non régulières ;
- retourne seulement un composant de nom sûr.

Le handler PDF existant revalide ensuite le fichier au moment de son ouverture, ce qui conserve la protection TOCTOU existante.

## 10. PDF absent

Si la génération est success mais que le workspace ou son PDF a disparu, le resolver échoue et la page success retourne 404. Aucune URL fictive n'est produite et aucune régénération automatique n'est tentée. La dette P2 de durabilité filesystem reste volontairement hors périmètre.

## 11. Cas multiple ou ambigu

Si plusieurs PDF réguliers sont présents dans le workspace, le resolver retourne explicitement `ErrAmbiguousExamGenerationPDF`. Le handler répond 404 et ne sélectionne jamais le premier fichier arbitrairement.

Un fichier non PDF, un répertoire suffixé `.pdf` ou un symlink `.pdf` ne devient pas candidat.

## 12. Renommage de classe testé

Oui. Le test fonctionnel :

- crée une génération success possédée ;
- place l'unique PDF historique dans son workspace ;
- charge success et vérifie le lien ;
- renomme la classe de `1A` vers `4e Alpha` ;
- recharge success ;
- vérifie le nouveau libellé affiché et l'URL historique inchangée ;
- appelle le handler PDF et compare exactement le contenu servi ;
- confirme que le workspace contient toujours un seul fichier, portant son ancien nom.

## 13. Ancien nom historique testé

Oui. Le fichier du test porte volontairement le nom historique `teacher_exam_workspace failure_1A.pdf`, alors que la classe en DB vaut ensuite `4e Alpha`.

## 14. Génération modifiée

Non. Le nom produit, la fusion, le cleanup et les transitions de génération sont inchangés.

## 15. Individualisation modifiée

Non. Aucun fichier de construction QCM/élève, questions, variantes, réponses, mélange ou worker n'a été modifié.

## 16. Marking modifié

Non.

## 17. Tests

Tests ajoutés/adaptés :

- résolution d'un PDF historique avec des fichiers non PDF ignorés ;
- refus explicite de deux PDF candidats ;
- refus symlink, répertoire `.pdf` et absence de candidat ;
- confinement utilisateur et username invalide ;
- view-data success utilisant le nom résolu fourni ;
- scénario fonctionnel complet de renommage de classe et accès au même ancien PDF ;
- smoke test template success adapté au filename résolu.

Les tests existants de `ServePdfNamed` continuent de couvrir traversal, parents symlinkés, workspace d'un autre utilisateur, fichier absent, fichier non régulier et service du contenu.

## 18. `./scripts/check.sh`

Résultat final : succès. `go mod verify`, contrôle `gofmt`, `go vet ./...`, `go test ./...`, `go build ./...` et `git diff --check` sont passants.

## 19. Race detector

Résultat final : `go test -race ./...` passe sur tous les packages.

## 20. Fichiers modifiés

- `internal/handlers/tools/resolveExamGenerationPDF.go` ;
- `internal/handlers/tools/resolveExamGenerationPDF_test.go` ;
- `internal/handlers/generateExams/handlers.go` ;
- `internal/handlers/generateExams/viewData_test.go` ;
- `docs/audits/exam-pdf-stable-access.md`.

Aucun commit n'a été créé.
