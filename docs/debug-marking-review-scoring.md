# Diagnostic du recalcul des notes après review

## Symptôme

Sur le job réel `4` (6eB), le PDF régénéré montre les réponses attendues, mais certaines questions revues semblent ne pas gagner de point et la note globale paraît ne pas évoluer autant qu'attendu.

L'analyse ci-dessous n'utilise que des identifiants techniques. Aucune donnée d'identité d'élève n'est incluse.

## Règle de scoring actuelle

`markingscoring.ScoreQuestion(expected, actual, totalPoints)` compare le vecteur attendu au vecteur effectif complet :

- égalité exacte : score complet, état `correct` ;
- sélection non vide composée uniquement de bonnes réponses, sans dépasser le nombre de bonnes réponses : demi-score, état `partial` ;
- sinon : zéro, état `incorrect`.

Toutes les questions observées ici valent un point et ont exactement une réponse attendue. Il n'existe donc pas de cas partiel dans cet échantillon : le vecteur effectif doit être exactement égal au vecteur attendu pour obtenir le point.

Les cercles et croix dessinés sur les réponses représentent le vecteur **attendu**. Le score de la question utilise le vecteur **effectif de l'élève**, soit `COALESCE(reviewed_state, detected_state)` réponse par réponse. Voir la bonne réponse entourée dans le PDF ne signifie donc pas que l'élève a cette réponse dans son vecteur effectif.

## Données du job 4

- 16 réponses ont une review manuelle.
- 9 reviews changent l'état par rapport à la détection automatique.
- Ces reviews concernent 16 questions et 8 copies.
- 6 des 9 changements corrigent le vecteur complet et font passer la question de `incorrect` à `correct`.
- 3 des 9 changements sélectionnent une mauvaise réponse : le vecteur change, mais le score reste `incorrect`, zéro point.
- Les 7 autres reviews confirment la valeur déjà détectée et ne changent ni le vecteur effectif ni le score.
- Les 16 résultats persistés de question correspondent exactement au recalcul par `ScoreQuestion`.

Au moment de cette lecture, la base examinée conserve `review_revision = 16` et `artifacts_revision = 0` pour le job 4. Cela indique seulement que cette copie de la base considère encore les artefacts comme obsolètes ; cela ne crée aucune divergence de scoring.

## Reviews ayant modifié l'état effectif

Reviews effectives : **9**.

Questions concernées : **9**.

- passage `incorrect -> correct` : **6** (`861`, `891`, `902`, `941`, `978`, `1009`) ;
- passage `incorrect -> partial` : **0** ;
- passage `partial -> correct` : **0** ;
- score inchangé : **3** (`901`, `904`, `950`).

Explication des trois scores inchangés :

- question `901` : attendu `[1, 0, 0, 0]`, review sur la deuxième réponse, effectif `[0, 1, 0, 0]` ;
- question `904` : attendu `[0, 1, 0, 0]`, review sur la troisième réponse, effectif `[0, 0, 1, 0]` ;
- question `950` : attendu `[0, 0, 1, 0]`, review sur la quatrième réponse, effectif `[0, 0, 0, 1]`.

Dans ces trois cas, la review est bien enregistrée et utilisée, mais elle valide une option différente de la réponse attendue du snapshot. Le maintien à zéro est conforme à la règle.

## Analyse question par question

`NULL` signifie que la réponse correspondante n'a pas été revue ; sa détection automatique est alors conservée.

### Question résultat 861

```text
copy_result_id:     97
student_exam_id:    37
question_result_id: 861
question_index:     0
total_points:       1
expected:           [0, 0, 1, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, 1, NULL]
effective:          [0, 0, 1, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 891

```text
copy_result_id:     100
student_exam_id:    40
question_result_id: 891
question_index:     0
total_points:       1
expected:           [0, 0, 0, 1]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, NULL, 1]
effective:          [0, 0, 0, 1]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 901

```text
copy_result_id:     101
student_exam_id:    32
question_result_id: 901
question_index:     0
total_points:       1
expected:           [1, 0, 0, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, 1, NULL, NULL]
effective:          [0, 1, 0, 0]
ScoreQuestion:      incorrect, 0 / 1
persisted:          state=incorrect, score_half_units=0
diagnostic:         cohérent — score inchangé, mauvaise option validée
```

### Question résultat 902

```text
copy_result_id:     101
student_exam_id:    32
question_result_id: 902
question_index:     1
total_points:       1
expected:           [1, 0, 0, 0]
detected:           [0, 0, 0, 0]
reviewed:           [1, NULL, NULL, NULL]
effective:          [1, 0, 0, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 904

```text
copy_result_id:     101
student_exam_id:    32
question_result_id: 904
question_index:     3
total_points:       1
expected:           [0, 1, 0, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, 1, NULL]
effective:          [0, 0, 1, 0]
ScoreQuestion:      incorrect, 0 / 1
persisted:          state=incorrect, score_half_units=0
diagnostic:         cohérent — score inchangé, mauvaise option validée
```

### Question résultat 941

```text
copy_result_id:     105
student_exam_id:    44
question_result_id: 941
question_index:     0
total_points:       1
expected:           [0, 1, 0, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, 1, NULL, NULL]
effective:          [0, 1, 0, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 950

```text
copy_result_id:     105
student_exam_id:    44
question_result_id: 950
question_index:     9
total_points:       1
expected:           [0, 0, 1, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, NULL, 1]
effective:          [0, 0, 0, 1]
ScoreQuestion:      incorrect, 0 / 1
persisted:          state=incorrect, score_half_units=0
diagnostic:         cohérent — score inchangé, mauvaise option validée
```

### Question résultat 968

```text
copy_result_id:     107
student_exam_id:    51
question_result_id: 968
question_index:     7
total_points:       1
expected:           [1, 0, 0, 0]
detected:           [1, 0, 0, 0]
reviewed:           [1, NULL, NULL, NULL]
effective:          [1, 0, 0, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — confirmation, score inchangé
```

### Question résultat 969

```text
copy_result_id:     107
student_exam_id:    51
question_result_id: 969
question_index:     8
total_points:       1
expected:           [0, 0, 1, 0]
detected:           [0, 0, 1, 0]
reviewed:           [NULL, NULL, 1, NULL]
effective:          [0, 0, 1, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — confirmation, score inchangé
```

### Question résultat 970

```text
copy_result_id:     107
student_exam_id:    51
question_result_id: 970
question_index:     9
total_points:       1
expected:           [0, 0, 0, 1]
detected:           [0, 1, 0, 0]
reviewed:           [NULL, 1, NULL, NULL]
effective:          [0, 1, 0, 0]
ScoreQuestion:      incorrect, 0 / 1
persisted:          state=incorrect, score_half_units=0
diagnostic:         cohérent — confirmation d'une mauvaise option
```

### Question résultat 971

```text
copy_result_id:     108
student_exam_id:    48
question_result_id: 971
question_index:     0
total_points:       1
expected:           [0, 0, 0, 1]
detected:           [0, 1, 0, 0]
reviewed:           [NULL, 1, NULL, NULL]
effective:          [0, 1, 0, 0]
ScoreQuestion:      incorrect, 0 / 1
persisted:          state=incorrect, score_half_units=0
diagnostic:         cohérent — confirmation d'une mauvaise option
```

### Question résultat 978

```text
copy_result_id:     108
student_exam_id:    48
question_result_id: 978
question_index:     7
total_points:       1
expected:           [0, 0, 0, 1]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, NULL, 1]
effective:          [0, 0, 0, 1]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 979

```text
copy_result_id:     108
student_exam_id:    48
question_result_id: 979
question_index:     8
total_points:       1
expected:           [0, 0, 0, 1]
detected:           [0, 0, 0, 1]
reviewed:           [NULL, NULL, NULL, 1]
effective:          [0, 0, 0, 1]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — confirmation, score inchangé
```

### Question résultat 1009

```text
copy_result_id:     111
student_exam_id:    35
question_result_id: 1009
question_index:     8
total_points:       1
expected:           [0, 0, 1, 0]
detected:           [0, 0, 0, 0]
reviewed:           [NULL, NULL, 1, NULL]
effective:          [0, 0, 1, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — gain de 1 point
```

### Question résultat 1033

```text
copy_result_id:     114
student_exam_id:    58
question_result_id: 1033
question_index:     2
total_points:       1
expected:           [0, 1, 0, 0]
detected:           [0, 1, 0, 0]
reviewed:           [NULL, 1, NULL, NULL]
effective:          [0, 1, 0, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — confirmation, score inchangé
```

### Question résultat 1039

```text
copy_result_id:     114
student_exam_id:    58
question_result_id: 1039
question_index:     8
total_points:       1
expected:           [1, 0, 0, 0]
detected:           [1, 0, 0, 0]
reviewed:           [1, NULL, NULL, NULL]
effective:          [1, 0, 0, 0]
ScoreQuestion:      correct, 1 / 1
persisted:          state=correct, score_half_units=2
diagnostic:         cohérent — confirmation, score inchangé
```

## Vérification des scores de copie

La somme porte sur toutes les questions de chaque copie, pas seulement celles ayant une review.

| copy_result_id | student_exam_id | score avant reviews effectives | somme persistée des questions | score de copie persisté | total | diagnostic |
|---:|---:|---:|---:|---:|---:|---|
| 97 | 37 | 0 | 2 demi-unités | 2 demi-unités | 10 | cohérent, soit 1/10 |
| 100 | 40 | 2 demi-unités | 4 demi-unités | 4 demi-unités | 10 | cohérent, soit 2/10 |
| 101 | 32 | 2 demi-unités | 4 demi-unités | 4 demi-unités | 10 | cohérent, soit 2/10 |
| 105 | 44 | 0 | 2 demi-unités | 2 demi-unités | 10 | cohérent, soit 1/10 |
| 107 | 51 | 8 demi-unités | 8 demi-unités | 8 demi-unités | 10 | cohérent, soit 4/10 |
| 108 | 48 | 8 demi-unités | 10 demi-unités | 10 demi-unités | 10 | cohérent, soit 5/10 |
| 111 | 35 | 2 demi-unités | 4 demi-unités | 4 demi-unités | 10 | cohérent, soit 2/10 |
| 114 | 58 | 14 demi-unités | 14 demi-unités | 14 demi-unités | 10 | cohérent, soit 7/10 |

Les six questions corrigées ajoutent au total six points répartis sur six copies. Les copies `107` et `114` ne changent pas, car leurs reviews ne font que confirmer la détection. Pour les copies `101` et `105`, plusieurs réponses sont modifiées mais une seule question par copie gagne effectivement un point ; les validations d'options incorrectes ne produisent pas de point.

`marking_copy_results.score_half_units` égale, pour chacune des huit copies, la somme des `marking_question_results.score_half_units`. Le total de points égale également la somme des totaux des questions.

## Vérification de la régénération des artefacts

Le chemin de calcul est identique dans la review et la régénération :

1. `ApplyMarkingAnswerReview` écrit la review dans une transaction.
2. `ListEffectiveQuestionAnswersForReview` produit le vecteur effectif complet.
3. `ScoreQuestion` recalcule la question.
4. `UpdateMarkingQuestionResultFromReview` persiste son état et ses demi-unités.
5. `RecalculateMarkingCopyScoreFromQuestions` resomme les résultats de questions.
6. Lors de la régénération, `ListEffectiveMarkingAnswersForArtifacts` relit les mêmes reviews via `COALESCE(reviewed_state, detected_state)`.
7. `regenerateCorrectedCopy` reconstruit `effectiveAnswers`, appelle le même `ScoreQuestion` et refuse de continuer si ce résultat diffère de celui persisté.
8. Les 16 questions passent cette vérification. `questionMarks` contient donc les résultats recalculés ci-dessus.
9. `CountingTotalPoint(questionMarks)` retrouve les scores persistés des copies.
10. Le `MarkExam.Score` retourné contient ce score et alimente le tableau des notes.
11. `DrawMarking` reçoit, pour chaque cercle de question, le `QuestionMark` recalculé. Il reçoit séparément les réponses attendues pour dessiner la correction.

Aucune ancienne détection n'écrase une review : les 9 différences observées apparaissent toutes dans les vecteurs effectifs et dans les résultats recalculés.

## Cause racine

Aucune divergence logicielle ou de persistance n'est démontrée. Le comportement observé vient de l'interprétation visuelle de la correction : les marques placées sur les réponses montrent les réponses attendues, alors que le point affiché près de la question dépend du vecteur effectif complet de l'élève.

Pour les questions `901`, `904` et `950`, la review a bien changé `0` en `1`, mais sur une mauvaise option. Elles doivent rester à zéro. Les sept reviews de confirmation ne peuvent pas faire évoluer la note. Seules six reviews augmentent effectivement une note.

## Explication du comportement observé

Une validation manuelle signifie « cette case élève est cochée/non cochée », et non « attribuer le point » ni « choisir la réponse correcte ». L'interface de review enregistre l'état de la case ambiguë. Le moteur reconstitue ensuite toutes les cases de la question et applique la règle normale.

Ainsi, un cercle vert visible autour de la bonne réponse dans le PDF décrit le corrigé attendu. Il ne constitue pas une visualisation de la décision de review. Une question peut légitimement afficher la bonne réponse attendue tout en restant marquée incorrecte parce que le vecteur effectif de l'élève sélectionne une autre option.

## Tests et validation

Aucun correctif fonctionnel et aucun nouveau test de régression ne sont nécessaires, puisqu'aucune divergence n'a été trouvée. Les tests existants couvrent déjà le partage de `ScoreQuestion`, le recalcul transactionnel review/question/copie et la reconstruction des artefacts.

- Diagnostic réel : lecture seule de la base du job 4 et recalcul des 16 vecteurs avec `markingscoring.ScoreQuestion`.
- Cohérence questions : 16/16.
- Cohérence copies : 8/8.
- Tests ciblés : `go test ./internal/markingscoring ./internal/db ./internal/handlers/tools` — réussi.
- Suite complète : `go test ./...` — réussi.
- Vérification du diff : `git diff --check` — réussi.

## Conclusion

Les reviews sont enregistrées et consommées correctement, les résultats de questions et de copies sont cohérents, et la régénération utilise ces mêmes résultats. Sur les 9 changements effectifs, 6 ajoutent un point et 3 restent sans point parce que l'option validée ne correspond pas à l'option attendue. Le comportement du job 4 est conforme à la règle de scoring actuelle ; aucune modification de code n'est justifiée.
