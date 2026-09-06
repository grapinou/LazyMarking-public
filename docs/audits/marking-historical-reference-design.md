# Conception d’une référence visuelle historique pour Marking

## Verdict

Le besoin de Marking n’est pas le source Typst historique ni les fichiers image
historiques pris séparément. L’algorithme consomme finalement une **image raster
de référence de chaque page**, dans le même repère que les coordonnées
persistées dans `student_exam_page_content`. La plus petite architecture fiable
est donc de conserver, dans le workspace durable de la génération Exam, le PNG
300 PPI déjà produit pendant la génération, avant ajout du QR, pour chaque
couple `(student_exam_id, page_exam)`.

Ce raster est une donnée source durable, pas un cache : sans lui, une
reconstruction ultérieure dépend du template, des images, de Typst et des
polices vivants. Le PDF global success reste le document imprimable durable,
mais il ne possède actuellement aucun mapping physique persistant suffisamment
fiable pour être l’unique primitive de lecture des pages nouvelles.

## 1. Traçage exact du pipeline actuel

### Lecture et reconstruction pendant Marking

Pour chaque groupe QR, `GroupQrCodes` dans
`internal/handlers/tools/groupQrCodes.go` construit un `config.Exam` identifié
par `StudentExamID`, avec ses pages triées par `page_exam`.

`MarkingStudentExam` dans
`internal/handlers/tools/markingStudentExam.go` exécute ensuite :

1. `GetStudentContentExam` lit `student_exam_content` avec ownership ; son JSON
   est désérialisé en `config.QCM`.
2. `TypstWriter` (`internal/handlers/tools/typstWriter.go`) relit
   `internal/config/ref_qcm.txt`, injecte examen, élève, classe, questions,
   réponses et chemins d’images.
3. `typstImagePath` (`internal/handlers/tools/typstImagePath.go`) transforme le
   seul nom snapshoté en `/assets/images/<name>` ; les octets et leur hash ne
   sont pas dans le snapshot.
4. `ExportTypstToPNGs` (`internal/handlers/tools/exportTypstPNG.go`) appelle
   `typst compile --format png --ppi 300`, avec la racine du dépôt.
5. Pour chaque page, `GetPageContent` lit dans
   `student_exam_page_content` les centres/rayons historiques.
6. `Homography` (`internal/handlers/tools/homography.go`) charge le scan et le
   PNG reconstruit, convertit les deux en gris, applique le seuillage adaptatif,
   SIFT, BF/KNN, ratio test, RANSAC, puis projette le scan vers une image ayant
   exactement `img2.Cols() × img2.Rows()`, c’est-à-dire les dimensions du PNG
   de référence.
7. `GetAnswerDetections` (`internal/handlers/tools/getAnswersState.go`) mesure
   les ROI aux coordonnées historiques sur le scan aligné.

L’homographie consomme donc réellement deux matrices de pixels OpenCV : le scan
et le raster historique attendu. Typst, le template et les images ne sont que
des moyens actuels de recréer tardivement ce second raster.

### Création de la référence et de la géométrie pendant GenerateExam

`GenerateExamsHandler` dans
`internal/handlers/generateExams/handlers.go` lance en parallèle un
`BuildQcmStudentCtx` par élève. Cette fonction, dans
`internal/handlers/tools/buildQcmStudentCtx.go` :

1. crée `student_exam` ;
2. construit le QCM depuis la banque vivante ;
3. appelle le même `TypstWriter`, puis le même `ExportTypstToPNGs` à 300 PPI ;
4. détecte questions et réponses sur chaque PNG **avant ajout du QR** avec
   `CircleDetection` et `CircleDetectionAnswer` ;
5. persiste ces coordonnées/rayons par page dans
   `student_exam_page_content` ;
6. colle le QR sur une copie de même canevas avec `PasteQrCodeOnPage` ;
7. transforme ce PNG QR en PDF A4 par `ConvertPngTopdf` ;
8. fusionne les pages d’une copie si elle en a plusieurs ;
9. persiste le QCM complet dans `student_exam_content`.

Après tous les élèves, le handler récupère tous les PDF intermédiaires par
`GetAllFiles("*.pdf")` (`filepath.Glob`), les fusionne par `MergePdf`/`pdfunite`,
puis `cleanupExamGenerationFiles` supprime tous les PDF intermédiaires, PNG et
Typst. La génération passe à success seulement après cette fusion et ce
cleanup. Le workspace success conserve le seul PDF global final.

## 2. Artefact réellement nécessaire

Le besoin minimal est **C : le rendu visuel final historique de la page dans le
repère 300 PPI utilisé pour détecter la géométrie**. Le meilleur artefact pour
préserver strictement le comportement actuel est le PNG pré-QR produit par
`ExportTypstToPNGs` :

- il est exactement l’actuel `pngBase` de `Homography` ;
- les coordonnées/rayons ont été détectés directement dessus ;
- il évite toute nouvelle compilation et toute conversion PDF ;
- il évite de modifier indirectement la sélection des correspondances SIFT par
  l’ajout du QR dans l’image de référence.

Le PNG avec QR est le raster effectivement incorporé dans le PDF imprimable et
conserve les mêmes dimensions, mais son emploi comme nouvelle référence peut
changer les points SIFT. Il mérite une comparaison sur scans réels, pas une
substitution silencieuse. Conserver le PNG pré-QR n’empêche pas le PDF global
d’être l’autorité documentaire de ce qui a été imprimé ; le PNG devient
l’autorité technique de correction.

## 3. Géométrie et DPI

`student_exam_page_content.content` sérialise des `config.PageContent` dont les
positions X/Y et rayons sont des pixels détectés sur le PNG Typst 300 PPI. Pour
une page A4, le raster attendu est approximativement 2480 × 3508 pixels. Les
bornes codées en génération (`qrPosition = 415`, `bottomPosition = 3390`)
confirment ce repère pixel.

`Homography` projette le scan vers les dimensions du raster de référence, puis
les ROI utilisent directement les coordonnées sauvegardées. Une autre taille
ou un autre DPI casserait ce contrat ou demanderait une transformation explicite
des coordonnées. La référence recommandée doit donc rester un PNG 300 PPI avec
dimensions vérifiées et persistées.

Extraire le PDF global à 300 PPI devrait retrouver le même canevas A4 : le PDF
est lui-même construit à partir du PNG 300 PPI, sans redimensionnement
intentionnel, par `ConvertPngTopdf`. Il faut toutefois valider les dimensions et
le rendu du rasteriseur ; `ConvertPdfToPng` actuel n’impose pas 300 DPI et ne
convient donc pas tel quel au contrat futur.

## 4. PDF global et mapping physique

Le PDF success contient toutes les copies concaténées. L’ordre physique n’est
pas stocké en DB : ni `student_exam`, ni `student_exam_content`, ni
`student_exam_page_content` ne porte d’ordinal global.

Le mapping implicite n’est pas un contrat robuste :

- `GetAllFiles` utilise l’ordre lexical de `filepath.Glob` ;
- une copie multi-page est renommée `student-exam-<id>.pdf` ;
- une copie d’une seule page n’entre pas dans cette branche de renommage et
  garde un nom issu du fichier Typst temporaire aléatoire ;
- les workers sont concurrents ;
- l’ordre lexical d’identifiants numériques n’est pas leur ordre numérique.

Il ne faut donc pas déduire une page physique depuis l’ID ou l’ordre SQL.
Cependant chaque page PDF contient un QR avec `{student_exam_id, page_exam}`.
Pour les générations legacy, on peut rasteriser toutes les pages du PDF durable,
décoder leur QR, vérifier ownership/génération/unicité/complétude, puis établir
un mapping opportuniste fiable. Un échec de décodage ou un ensemble incomplet
doit rendre cette génération legacy non migrable automatiquement.

## 5. Images et template vivants

Dans `config.Question.Image`, le snapshot conserve seulement `Name` et `Width`.
Il ne contient ni octets, ni hash, ni copie du fichier. Le rendu tardif relit
`assets/images/<Name>` :

- fichier supprimé : compilation Typst impossible ;
- fichier remplacé sous le même nom : référence visuelle différente ;
- contenu ou dimensions modifiés : pagination, positions et traits SIFT peuvent
  diverger même si le nom snapshoté est identique.

`ref_qcm.txt` fixe notamment police `Liberation Sans`, taille 11 pt, marges,
header/footer, pagination et marqueurs. `TypstWriter` fixe aussi tableaux,
cercles, espacements et rendu des réponses. Une modification de police,
spacing, marges, tailles de cercle ou structure peut changer pagination,
coordonnées et descripteurs. La reconstruction dépend en plus :

- de la version du binaire Typst ;
- de la disponibilité/version de la police système `Liberation Sans` ;
- du rasteriseur Typst ;
- des assets questions ;
- du code Go qui assemble le source Typst.

Le QR et sa bibliothèque n’interviennent pas dans la reconstruction Marking
actuelle, car la référence reconstruite est pré-QR. Avec le raster durable,
aucune de ces dépendances n’est requise pour l’homographie future.

## 6. Options comparées

| Option | Intégrité historique | Complexité | Stockage | Performance | Legacy | Impact génération | Impact Marking |
|---|---|---:|---:|---|---|---|---|
| A. Snapshot template + images | Incomplète sans Typst/polices/version ; re-rendu possible mais pas garanti identique | Élevée | Faible à moyen, images dupliquées/dédupliquées à gérer | Compilation coûteuse conservée | Mauvaise : assets historiques inconnus | Copier/versionner plusieurs dépendances | Pipeline actuel presque inchangé mais fragile |
| B. PNG pré-QR historique par page | Maximale pour l’actuel repère et l’actuel algorithme | Faible | Moyen | Meilleure : simple lecture | Nouveau seulement ; legacy via PDF/QR si validé | Déplacer/copier le PNG avant cleanup, vérifier puis success | Supprime Typst/images/compile du chemin de référence |
| C. PDF global + mapping page | Bonne pour le document imprimé ; rasterisation à spécifier et QR présent | Moyenne | Aucun doublon supplémentaire | CPU de rasterisation à chaque correction ou cache | Bonne si QR décodable | Persister l’ordinal pour les nouveaux jobs | Resolver PDF, extraction, validation dimensions |
| D. PDF individuel durable par copie | Bonne, mapping simple par nom | Moyenne | Duplication avec PDF global | Rasterisation encore nécessaire | Les intermédiaires legacy sont déjà supprimés | Ne plus supprimer les PDF individuels | Extraction par copie/page |

L’option B est recommandée. Elle préserve exactement l’entrée de référence
actuelle, supprime le plus de dépendances vivantes et ne duplique pas cette
référence à chaque marking_job.

## 7. Stockage, sécurité et intégrité

La référence appartient à la génération Exam, car elle doit exister avant le
premier Marking et être réutilisée par plusieurs jobs. Emplacement recommandé :

`assets/tmp/<username>/exam-<generation_id>/references/student-exam-<student_exam_id>/page-<page_exam>.png`

Le resolver doit partir d’un utilisateur et d’une génération possédée, vérifier
le lien du `student_exam`, rejeter symlinks/traversées et ne jamais exposer ce
répertoire comme fichiers statiques partagés. Le workspace success Exam est déjà
durable et isolé selon ce scope ; le workspace Marking serait trop tardif et
dupliquerait les références entre tentatives.

Une metadata DB minimale est recommandée sur la ligne historique de page
existante, plutôt qu’une grande nouvelle hiérarchie :

- `reference_storage_key` (nullable pour legacy) ;
- `reference_width`, `reference_height` ;
- `reference_dpi` ;
- `reference_sha256`.

`student_exam_id`, `page` et `user_id` existent déjà dans
`student_exam_page_content`. Une future contrainte doit rendre les cinq valeurs
de référence toutes NULL (legacy) ou toutes présentes, et rendre le binding
immuable après success. Une table dédiée reste possible, mais n’apporte pas ici
de relation supplémentaire. Le storage key explicite évite de transformer une
convention de nom en unique autorité ; les contrôles ownership doivent toujours
traverser `student_exam -> exams_generated`.

Le SHA-256 a une valeur réelle : le PNG devient la seule preuve visuelle
technique permettant la correction historique. Le hash détecte corruption,
restauration partielle ou substitution de fichier ; il ne remplace ni backup ni
contrôle d’accès.

Ces références contiennent noms, évaluation et images. Le backup doit traiter
ensemble SQLite et le workspace `exam-<generation_id>` (PDF final + références),
avec restauration cohérente et vérification des hashes. Copier seulement la DB
ou seulement `assets/tmp` ne restaure pas l’état durable.

## 8. Volume et performance

Un A4 300 PPI représente environ 8,7 millions de pixels. En PNG, une page de
texte noir sur fond blanc compresse généralement bien, mais des images de cours
peuvent augmenter fortement le volume. Pour 30 élèves × 1 à 3 pages, prévoir un
ordre de grandeur de quelques dizaines à quelques centaines de Mo par
génération (souvent nettement moins pour du texte pur), à mesurer sur le corpus
réel. Le brut non compressé serait beaucoup plus gros et n’est pas recommandé.

JPEG introduirait des artefacts autour des textes et cases et n’apporte pas un
gain justifiant le risque. PNG est déjà le format natif du pipeline et est sans
perte. Le PDF global reste plus compact/commode pour impression et backup, mais
impose extraction et mapping.

La performance doit s’améliorer : aujourd’hui chaque copie relit le template et
les images puis relance Typst pour toutes ses pages. Demain Marking chargera
directement N PNG. Le coût OpenCV reste identique ; compilation et rasterisation
disparaissent du chemin chaud.

## 9. Lifecycle et atomicité de génération

Le meilleur moment est juste après production du PNG par
`ExportTypstToPNGs`, après validation de page/géométrie, et avant le cleanup.
Chaque fichier doit être copié/renommé atomiquement vers son emplacement final,
hashé et associé à la bonne page. La génération ne doit devenir success qu’après :

1. toutes les copies et snapshots écrits ;
2. toutes les références attendues présentes et validées ;
3. PDF global fusionné ;
4. cleanup des seuls intermédiaires ;
5. transition DB conditionnelle vers success.

Une génération running interrompue ou failed peut laisser des références
partielles dans son workspace, mais le cleanup/recovery existant supprime la
génération et le workspace entier. Une génération success est protégée contre
suppression ; ses références doivent suivre le même contrat. Aucun artefact
durable ne doit survivre seul à une génération failed.

## 10. Impact Marking attendu

Avant :

`snapshot -> TypstWriter -> ref_qcm.txt/images -> Typst compile 300 PPI -> PNG -> Homography`

Après :

`resolver ownership-aware -> PNG historique vérifié -> Homography`

Dans `MarkingStudentExam`, les appels de reconstruction `TypstWriter` et
`ExportTypstToPNGs`, ainsi que le cleanup du `.typ` et des PNG de référence
temporaires, deviendraient inutiles. Les lectures de `student_exam_content` et
`student_exam_page_content` restent nécessaires pour réponses attendues,
identité, statistiques et ROI. `Homography`, SIFT, RANSAC, détection, seuil et
score ne changent pas.

Cette solution garantit A, la capacité à corriger historiquement. Elle ne vise
pas B, une régénération byte-for-byte de l’Exam : le source Typst, sa version et
les dépendances ne sont pas archivés. Le PDF final demeure la preuve imprimable.

## 11. Compatibilité legacy

Les anciennes générations ont des metadata de référence NULL et aucun PNG
garanti. Aucune reconstruction exacte depuis template/images vivants ne doit
être présentée comme historique.

Stratégie recommandée :

1. ouvrir ownership-aware le PDF global durable ;
2. rasteriser ses pages à 300 PPI ;
3. décoder le QR de chaque page pour retrouver `(student_exam_id, page_exam)` ;
4. vérifier que chaque ID appartient à la génération, qu’il n’existe aucun
   doublon et que chaque page attendue est présente ;
5. vérifier dimensions et comparer la compatibilité de géométrie ;
6. seulement si tout est valide, enregistrer opportunistement les références
   legacy (ou les utiliser comme fallback explicitement marqué).

Ces pages comportent le QR alors que la référence historique courante ne
l’avait pas. Une validation synthétique puis réelle est donc obligatoire avant
promotion. Si le PDF manque, si un QR échoue ou si le set est incomplet, la
génération doit rester `legacy_reference_unavailable` conceptuellement. Un
fallback vers le template vivant peut être conservé temporairement pour
compatibilité, mais doit être identifié comme reconstruction non garantie et ne
doit jamais créer silencieusement une prétendue référence historique.

## 12. Tests futurs

Tests synthétiques de forte valeur :

- suppression de l’image vivante après génération : Marking réussit ;
- modification de `ref_qcm.txt` après génération : résultat inchangé ;
- remplacement d’une image sous le même nom : résultat inchangé ;
- deux marking_jobs résolvent le même artefact sans duplication ;
- référence absente, hash invalide, PNG corrompu ou dimensions/DPI inattendus :
  échec générique, jamais fallback silencieux ;
- mauvais utilisateur, génération, student_exam ou page : refus sans fuite ;
- génération success impossible avec une référence manquante ;
- failed/recovery supprime les références partielles ; success les conserve ;
- même scan et même référence : homographie, MeanGray, detected_state et score
  identiques au pipeline actuel ;
- extraction legacy du PDF : mapping QR complet, rejet des doublons/inconnus.

Validation locale opt-in avec `db_test_real.db` et les trois paquets privés :

1. conserver les sorties runtime de la reconstruction actuelle ;
2. produire/résoudre la référence historique candidate (PNG natif pour une
   nouvelle génération, extraction PDF pour legacy) ;
3. comparer dimensions, matrice/qualité d’homographie si exposable, MeanGray par
   réponse, états détectés, QuestionMark et score ;
4. exiger des résultats identiques ou expliquer et valider toute amélioration ;
5. ne jamais copier scans, DB ou résultats nominatifs dans Git/CI.

## 13. Jalons d’implémentation

1. **Contrat page/référence** : metadata nullable, ownership, immutabilité,
   hash/dimensions/DPI et resolver sécurisé, sans modifier Marking.
2. **Production de référence** : conserver atomiquement le PNG pré-QR pendant
   `BuildQcmStudentCtx`; rendre le success Exam conditionnel à la couverture.
3. **Lecture Marking** : remplacer uniquement TypstWriter/export par le resolver
   de PNG, avec tests d’équivalence synthétiques.
4. **Lifecycle** : tests failure/recovery/backup et corruption.
5. **Legacy PDF** : extraction 300 PPI + mapping QR strict, explicitement
   opportuniste.
6. **Validation réelle opt-in** : les trois corpus privés.

Le premier jalon doit être le contrat persistant page/référence et son resolver,
car il permet de sécuriser l’identité et le stockage avant de changer les deux
pipelines.

## 14. Priorité

Ce chantier reste P1. Tant que Marking reconstruit sa référence depuis des
dépendances vivantes, un résultat historique peut devenir impossible ou
différent sans modification du scan. Il est bloquant avant la clôture Marking et
doit précéder les améliorations d’ambiguïté et d’UX qui supposent une mesure
historiquement fiable.
