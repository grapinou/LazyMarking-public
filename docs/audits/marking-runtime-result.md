# Résultat runtime détaillé Marking

Date : 2026-08-31

## Périmètre

Ce jalon conserve les mesures et résultats détaillés en mémoire. Il n'ajoute aucune migration, table, requête SQLC ou écriture DB de production. Le seuil, la règle de score, OpenCV, les PDF, statistiques et lifecycle restent inchangés.

## `AnswerDetection`

`config.AnswerDetection` contient :

- `State int`, exactement la valeur historique 0/1;
- `MeanGray float64`, exactement la moyenne du canal gris calculée sur la ROI par GoCV.

`GetAnswerDetections` lit chaque ROI une seule fois et construit les deux valeurs depuis le même `avg`. `GetAnswersState` reste un adaptateur de compatibilité : il appelle cette primitive puis extrait les états. Il n'existe donc aucun second passage image ni recalcul de moyenne.

Les valeurs NaN, infinies ou hors `[0,255]` produisent une erreur; aucun clamp n'est effectué. Les états autres que 0/1 sont également rejetés lors de la construction du résultat détaillé.

## Seuil inchangé

Le nombre magique local est devenu la constante interne `markingDetectionThreshold = 150.0`, sans changement de valeur ni de comparaison :

- `avg < 150` : cochée, état 1;
- `avg == 150` : non cochée, état 0;
- `avg > 150` : non cochée, état 0.

Aucune version moteur ou metadata DB n'est ajoutée.

## Résultat par question

`config.MarkingQuestionResult` conserve :

- `QuestionIndex int`, zéro-based;
- le `config.QuestionMark` produit par `CountingPoints`, sans recalcul;
- les `AnswerDetections` ordonnées de cette question.

Il ne duplique ni formulation, tags, réponse attendue ou coordonnées du snapshot.

`BuildMarkingCopyResult` exige autant de `QuestionMark` que de questions historiques et exactement autant de détections que de réponses historiques. Il découpe la séquence dans l'ordre du snapshot et copie les détections dans chaque question. Un mismatch reste une erreur de correction, jamais un résultat corrected incomplet.

## Résultat détaillé de copie

`config.MarkingCopyResult` contient :

- `StudentExamID`;
- `ExpectedPages`;
- `DetectedPages`;
- les questions détaillées;
- `Score float64`;
- `Total int`.

Le score et le total sont produits par le même `CountingTotalPoint(questionMarks)` que le runtime historique. Le résultat est attaché à `MarkExam.DetailedResult` après succès. Les anciens champs `MarkExam` sont toujours remplis avec les mêmes variables `mark`, `tot`, `skill` et `themeSkill` qu'avant.

## Compatibilité `MarkExam`, PDF et statistiques

`CountingPoints` n'a subi aucune modification métier. `MarkingStudentExam` appelle désormais la lecture structurée, extrait ses états 0/1, puis continue avec `validateMarkingVectors`, `CountingPoints`, `CountingTotalPoint`, `GetThemeSkill` et `DrawMarking` dans le même ordre.

Le résultat détaillé est construit depuis les `QuestionMark` déjà calculés; il n'existe pas de second calcul parallèle.

- `DrawMarking` et son interface sont inchangés;
- les réponses attendues dessinées, scores, marques et ordre sont inchangés;
- `corrected.pdf`, `mark-table.pdf`, `corrected_NOT.pdf` et `LeftPages` sont inchangés;
- moyenne, médiane, écart-type, compétences et thèmes continuent d'utiliser les champs historiques de `MarkExam`;
- aucun calcul statistique DB n'est introduit.

## Conversion `score_half_units`

`ScoreHalfUnits` est un adaptateur pur qui multiplie le score `float64` par deux et exige une distance au nombre entier le plus proche inférieure ou égale à `1e-9`. Il refuse :

- les valeurs non multiples de 0,5, dont 0,25 et 1,25;
- les valeurs négatives;
- NaN et les infinis;
- les valeurs dépassant la capacité `int64`.

Les conversions attendues sont préservées : 0→0, 0,5→1, 1→2, 1,5→3.

## Mapping des états

`MarkingQuestionStateName` centralise le mapping futur DB :

- `config.Incorrect` → `incorrect`;
- `config.Partial` → `partial`;
- `config.Correct` → `correct`.

Tout état inconnu est refusé. Aucun switch string équivalent n'est dispersé ailleurs.

## Adaptateur repository

`MarkingCopyResultToPersistedInput` transforme purement le résultat runtime en `db.PersistedMarkingCopyInput`. Il reçoit séparément `userID` et `markingJobID`, puis conserve :

- student exam et nombres de pages;
- score copie en demi-points et total;
- question_index zéro-based;
- état, score en demi-points et total de chaque question;
- answer_index zéro-based;
- état détecté et `MeanGray` float64 inchangé.

L'adaptateur vérifie la cohérence état/score, la concordance état/moyenne, les positions d'indices et les sommes copie/questions. Une mesure `143.25` reste exactement `143.25` dans l'entrée repository.

Il n'appelle pas `PersistCorrectedMarkingCopy`. `ProcessMarking` et `MarkingStudentExam` n'appellent aucune écriture de résultat DB. La recherche finale montre les appels Create/Persist uniquement dans le repository 0037 et ses tests.

## Indices et ordre

Les pages continuent d'être triées avant traitement. Les détections sont concaténées page par page dans l'ordre des cercles historiques. `BuildMarkingCopyResult` les découpe ensuite selon `qcm.Questions` et `question.Answers`.

Les indices sont strictement zéro-based. Le test vérifie notamment `(question 0, answer 0)`, `(question 0, answer 1)` et `(question 1, answer 0)`, sans conversion d'affichage.

## Lifecycle et artefacts

Lifecycle modifié : non. `CompleteMarkingJob`, `FailMarkingJob`, `DeleteMarkingJob`, `ListExpiredMarkingJobs` et `RecoverRunningMarkingJobs` sont inchangés.

Artefacts et rétention modifiés : non. Aucun appel, nom, contenu ou emplacement PDF n'est changé.

## Tests

Tests synthétiques ajoutés/adaptés :

- classification sous, exactement sur et au-dessus de 150;
- conservation exacte de plusieurs MeanGray;
- rejet des moyennes négatives, supérieures à 255, NaN et infinies;
- résultat multi-question avec full et deux partials;
- équivalence des `QuestionMark`, score/total et agrégats compétence/thème;
- indices question/réponse zéro-based dans l'adaptateur;
- demi-point 0,5→1 et 1,5→3;
- rejet des scores 0,25 et 1,25;
- mapping des trois états et rejet d'un état inconnu;
- rejet des détections manquantes ou surnuméraires;
- cas historiques full, partial, incorrect, mauvaise réponse, aucune réponse et sur-sélection;
- compilation et suites PDF/statistiques existantes inchangées.

Aucune vraie copie, base privée ou variable `LAZYMARKING_TEST_*` n'est utilisée.

## Validation

- `./scripts/check.sh` : succès;
- `go test -race ./...` : succès;
- `git diff --check` : succès.

## Fichiers modifiés

- `internal/config/config.go`;
- `internal/handlers/tools/getAnswersState.go`;
- `internal/handlers/tools/markingStudentExam.go`;
- `internal/handlers/tools/markingRuntimeResult.go`;
- `internal/handlers/tools/markingRuntimeResult_test.go`;
- `internal/handlers/tools/countingPoints_test.go`;
- `docs/audits/marking-runtime-result.md`.

Aucun SQL, fichier généré, template, migration ou commit n'a été ajouté dans ce jalon.
