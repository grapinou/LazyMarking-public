# Conception de la persistance des résultats Marking

Date : 2026-08-31
Nature : audit et conception en lecture seule, sans implémentation

## 1. Résultat runtime actuel

### 1.1 `config.MarkExam`

`MarkExam` est construit uniquement à la fin réussie de `MarkingStudentExam`. Avant cela, une valeur partielle avec `Status=false` circule; seuls `FirstName` et `LastName` sont renseignés juste après lecture du snapshot. En cas d'erreur plus précoce, même ces champs restent vides.

| Champ | Origine et signification | Durée de vie | `corrected.pdf` | `mark-table.pdf` | Statistiques compétences/thèmes |
|---|---|---|---|---|---|
| `StudentExamID int64` | QR regroupé, puis recopié depuis `config.Exam`; identité de la copie historique | Mémoire du job; nom du PDF individuel temporaire | Associe `student-exam-<id>.pdf` à la copie et au sommaire | Non affiché | Non |
| `Status bool` | `false` par défaut, `true` seulement après correction et création du PDF individuel | Mémoire du job | Filtre indirectement les copies présentes | Filtre indirectement `markExams`; non affiché | Seules les copies `true` entrent dans les agrégats |
| `ExamName string` | `qcm.Name` du snapshot `student_exam_content` | Mémoire; actuellement inutilisé après construction | Non | Non | Non |
| `FirstName string` | `qcm.Student.FirstName` du snapshot | Mémoire | Nom et entrée de sommaire | Ligne élève; tentative d'affichage des non-corrigés | Non |
| `LastName string` | `qcm.Student.LastName` du snapshot | Mémoire | Nom et entrée de sommaire | Ligne élève | Non |
| `ClassName string` | `qcm.Student.ClassCodes.Name` du snapshot | Mémoire; actuellement inutilisé après construction | Non | Non | Non |
| `Pages int` | Nombre de pages Typst exportées, vérifié contre pages scannées et `page_tot` | Mémoire | Incrémente la pagination du sommaire | Non | Non |
| `Score float64` | Somme des `QuestionMark.Score` | Mémoire | Le score par question est dessiné, pas ce total comme donnée structurée | Note `%.2f/Total`; source de moyenne, médiane et écart-type | Contribue indirectement aux statistiques générales, pas aux maps de tags |
| `Total int` | Somme des points théoriques `QuestionMark.Total` | Mémoire | Les totaux par question sont dessinés | Dénominateur de la note et des statistiques | Non directement |
| `Skill map[int64]CounterTag` | Agrégation par ID de compétence depuis le snapshot et les scores par question | Mémoire | Non | Agrégée au niveau classe puis affichée en pourcentage | Oui, source directe |
| `ThemeSkill map[string]CounterTag` | Agrégation par clé textuelle `themeID-skillID` | Mémoire | Non | Agrégée au niveau classe puis affichée en pourcentage | Oui, source directe |

### 1.2 Structures imbriquées et transitoires

- `QrCodeInfo` porte `student_exam_id`, `page_exam` et un nom de page ajouté après décodage. Il sert au scope génération puis au regroupement.
- `Exam` porte un `StudentExamID` et des `Page{Number, Name}` triées. C'est le lot d'entrée d'une copie, pas un résultat.
- `QCM` est le snapshot historique désérialisé : nom de l'examen, identité/libellés de l'élève et questions ordonnées.
- `Question` contient formulation, image, cercle, réponses ordonnées et `Tags` historiques (IDs et noms de matière, thème, niveau, compétence, difficulté, valeur de points).
- `Answer` contient symbole, libellé, état correct attendu `State int64` et cercle historique.
- `PageContent` fournit les cercles historiques de questions/réponses d'une page pour l'alignement et la lecture.
- `HomoPage` associe le nom de l'image alignée à son `PageContent`; il disparaît après dessin.
- `answersState []int` est la séquence aplatie des réponses détectées, dans l'ordre page puis cercle/réponse historique. Les valeurs produites sont 0 ou 1.
- `QuestionMark` porte réellement le résultat par question : `Score float64`, `Total int64`, et `State` parmi `Incorrect(0)`, `Partial(1)`, `Correct(2)`. Cette structure est dessinée et agrégée, puis perdue.
- `CounterTag{Name, Score, Total}` porte une agrégation transitoire de scores par compétence ou thème-compétence.

## 2. Trace exacte d'une copie corrigée

1. Le QR fournit `student_exam_id` et `page_exam`; le jalon précédent prouve leur appartenance au job et à sa génération.
2. `GetStudentContentExam` charge `page_tot` et le JSON `QCM` de `student_exam_content`.
3. Le JSON donne, dans son ordre historique, les questions, réponses, états corrects, points, libellés, tags et coordonnées globales.
4. `GetPageContent` charge pour chaque page les coordonnées historiques des cercles.
5. Le scan est aligné sur la page Typst reconstruite par homographie.
6. Pour chaque cercle réponse, `GetAnswersState` calcule la moyenne de gris de la ROI; `avg < 150` produit 1, sinon 0. Seul le 0/1 quitte la fonction.
7. Les états détectés sont concaténés dans `answersState`, en ordre de pages triées puis ordre des cercles du snapshot.
8. `validateMarkingVectors` vérifie la cohérence des tailles avant calcul.
9. `CountingPoints` découpe `answersState` selon le nombre de réponses de chaque question. Il compare le vecteur attendu au vecteur détecté.
10. Égalité exacte : totalité des points. Sinon, une sélection non vide, sans mauvaise réponse, contenant au moins une bonne réponse et n'excédant pas le nombre de bonnes réponses vaut la moitié. Tout autre cas vaut zéro.
11. `CountingTotalPoint` somme scores et totaux. `GetThemeSkill` agrège les mêmes scores par tags historiques.
12. `DrawMarking` dessine l'état et le score par question sur la copie alignée. Il dessine aussi les bonnes réponses attendues; cette seconde annotation ne représente pas le vecteur détecté.
13. Le PDF individuel alimente `corrected.pdf`; `MarkExam` alimente son sommaire, `mark-table.pdf` et les statistiques.

Disparaissent aujourd'hui après le job : chaque état détecté, chaque moyenne de gris, la correspondance question/réponse, tous les `QuestionMark`, les raisons structurées d'échec, le bilan des copies attendues/non vues/incomplètes, et les agrégats runtime. Après purge disparaissent aussi les PDF et pages résiduelles.

## 3. Réponses attendues et détectées

La représentation détectée réelle est `[]int`, pas `[]bool`. Le moteur produit actuellement uniquement 0 ou 1.

Pour une question, trois axes doivent rester distincts :

- `question_index` et `answer_index` identifient l'ordre historique dans le JSON `QCM`;
- `Answer.State` dans le snapshot est l'état correct attendu;
- `detected_state` est l'interprétation 0/1 du scan pour cette tentative de correction.

Le résultat futur doit persister `detected_state` par réponse et ses indices. Il ne doit pas recopier l'état attendu : celui-ci reste dans le snapshot immuable et se retrouve par position. Une FK vers la table vivante `answers` serait incorrecte, notamment pour les variantes et l'historique.

## 4. Score par question

`QuestionMark` porte déjà le score numérique, le total théorique et la classe correct/partial/incorrect. Il n'est pas conservé dans `MarkExam`; après dessin et agrégation il disparaît.

Les points de base sont des entiers positifs. Le contrat actuel accorde uniquement 0, la moitié, ou le total. Les scores `float64` sont donc toujours des multiples exacts de 0,5; aucun arrondi métier n'est appliqué. `%.2f` est uniquement un formatage PDF.

La représentation DB la plus fidèle et contrainte est un entier en demi-points (`score_half_units`, soit deux fois le score), accompagné de `total_points` entier et d'un état métier. Elle évite les comparaisons flottantes sans inventer une précision absente. Si le barème gagne plus tard des quarts ou coefficients, une migration explicite sera préférable à une précision implicite.

## 5. Score global

- obtenu : `MarkExam.Score`, `float64`, somme des scores par question;
- possible : `MarkExam.Total`, `int`, somme des points entiers;
- précision effective : demi-point exact;
- affichage : `%.2f/%d` dans les PDF;
- arrondi métier : aucun.

Le score global peut être stocké en `score_half_units` entier et le total en points entiers. Il est techniquement dérivable des lignes question, mais le conserver sur le résultat copie est justifié comme total validé au moment de la finalisation, pour lecture rapide et contrôle d'intégrité. Une contrainte/test doit garantir l'égalité avec la somme des questions.

## 6. Identité durable et plusieurs jobs

La chaîne d'autorité recommandée est :

`user -> marking_job -> exams_generated -> student_exam -> snapshot`, avec en plus `marking_copy_result -> marking_job` et `marking_copy_result -> student_exam`.

Il n'est pas nécessaire de dupliquer `exam_generated_id` dans chaque résultat : il est déterminé par le job et le `student_exam`. Un trigger doit toutefois prouver que les deux aboutissent à la même génération et au même user.

La stratégie recommandée est l'option A : chaque job est une tentative historique indépendante. Un même `student_exam` peut donc avoir un résultat dans les jobs 1, 2 et 3, avec `UNIQUE(marking_job_id, student_exam_id)` mais sans unicité globale sur `student_exam_id`. Aucun résultat ne remplace silencieusement un autre et aucune notion de « courant » n'est introduite tant que le produit ne l'a pas décidée.

## 7. Copies non corrigées et succès partiel

Le vocabulaire minimal proposé pour un résultat attendu par copie est :

- `corrected` : résultat complet, questions/réponses et score persistés;
- `incomplete` : certaines pages de cette copie ont été reconnues mais il en manque ou elles sont dupliquées;
- `not_seen` : aucun QR exploitable n'a rattaché de page à cette copie attendue;
- `error` : toutes les préconditions semblaient réunies mais une étape technique de correction a échoué.

« Non corrigée » est un agrégat d'affichage regroupant `incomplete`, `not_seen` et `error`, pas nécessairement un cinquième état. Un `failure_code` stable et un détail technique optionnel/sanitisé expliquent la raison sans figer dès maintenant une taxonomie exhaustive.

Les pages QR illisibles ne peuvent honnêtement être rattachées à un `student_exam`; elles doivent devenir des `marking_job_issues` au niveau du job, avec au minimum type et numéro de page d'entrée. Le nom temporaire n'est pas une identité durable.

Le statut actuel `success` signifie « pipeline terminé avec au moins une copie corrigée », pas « toutes les copies corrigées ». La persistance par copie suffit à calculer attendues, corrigées et non corrigées, à condition qu'une ligne outcome existe pour chaque `student_exam` attendu de la génération. Aucun agrégat chiffré supplémentaire ni nouveau statut job n'est requis dans le premier jalon. Un futur `completed_with_issues` peut être ajouté séparément si l'UX doit distinguer ce cas immédiatement.

## 8. Ambiguïté future

Conserver seulement 0/1 empêcherait d'expliquer ou de reclasser une mesure proche du seuil sans retraitement d'image. La donnée brute pertinente déjà calculée est `avg`, moyenne `float64` du canal gris dans la ROI intersectée avec l'image, nominalement sur une échelle 0–255. Les coordonnées et rayons restent dans `student_exam_page_content`.

Le modèle doit donc conserver par réponse :

- `detected_state` 0/1;
- `mean_gray` REAL issu du `float64` Go;
- le seuil effectivement appliqué au niveau du job (`150` aujourd'hui);
- une version minimale du moteur de détection/correction.

La distance au seuil et une future classe ambiguë sont dérivables de `mean_gray` et du seuil; elles ne doivent pas être stockées maintenant. La mesure ne remplace toutefois pas une preuve visuelle : une validation humaine complète peut encore nécessiter l'artefact de scan/copie.

## 9. Comparaison des granularités

| Option | Avantages | Coût | Reconstruire la note | Statistiques | Ambiguïtés | Dépendance au scan |
|---|---|---|---|---|---|---|
| A. Copie globale | Très simple, faible volume | Perte du détail | Relit le total mais ne l'explique pas | Seulement notes globales | Impossible | Forte pour toute vérification |
| B. Copie + question | Explique total, demi/plein/zéro, compétences/thèmes recalculables via snapshot | Quelques centaines/milliers de lignes par classe | Oui pour le score | Oui | Ne sait pas quelle case a été lue | Scan requis pour interprétation réponse |
| C. Copie + question + réponse/mesure | Fidélité du moteur, diagnostic et future revue d'ambiguïté | Quelques milliers de petites lignes par classe | Oui et vérifiable de bout en bout | Oui | Oui, sans retraitement complet pour le signal numérique | Scan encore utile pour validation visuelle, mais non pour retrouver la mesure |

Recommandation : C. C'est la première granularité qui remplit les rôles de copie corrigée, note explicable et diagnostic pédagogique sans faire du PDF la base de données.

## 10. Normalisé versus JSON et snapshots existants

Les outcomes, scores par question, détections et issues doivent être relationnels : ils ont des contraintes d'unicité, servent aux comptes/statistiques et doivent être auditables indépendamment.

Il ne faut pas recopier dans les résultats : formulations, symboles/libellés de réponses, états corrects attendus, métadonnées du barème source, noms de compétences/thèmes, coordonnées, identité/libellé historique de l'élève ou classe. Ces données existent déjà dans `student_exam_content` et `student_exam_page_content` et sont rattachées au `student_exam` historique. Les seuls totaux répétés plus bas sont les valeurs numériques effectivement validées par l'exécution, afin de contrôler la cohérence du résultat.

Le JSON existant est approprié comme snapshot de document hétérogène et ordonné. Les résultats utilisent des indices positionnels stables vers ce snapshot. Un éventuel petit JSON de diagnostic technique peut exister pour des détails non requêtés, mais ne doit pas remplacer les colonnes métier.

Le snapshot `QCM` contient déjà les noms/prénoms/classe, le nom d'examen, les formulations, réponses, états attendus, points, compétences et thèmes avec leurs libellés. Une lecture historique ne nécessite donc pas les tables vivantes `students`, `questions`, `answers`, `points` ou `class_codes`. Le calcul et l'affichage futurs doivent explicitement charger le snapshot, pas rejoindre ces tables vivantes.

## 11. Modèle DB recommandé

### `marking_copy_results`

Une ligne par copie attendue et par tentative.

- PK `id`;
- `user_id` NOT NULL, FK user;
- `marking_job_id` NOT NULL, FK job;
- `student_exam_id` NOT NULL, FK snapshot/copie;
- `outcome` NOT NULL (`corrected`, `incomplete`, `not_seen`, `error`, vocabulaire à valider avant migration);
- `expected_pages` NOT NULL;
- `detected_pages` NOT NULL;
- `score_half_units` NULL sauf `corrected`;
- `total_points` NULL sauf `corrected`;
- `failure_code` NULL pour `corrected`, sinon renseignable;
- `failure_detail` NULL, diagnostic sanitisé;
- `completed_at` NULL pendant construction, NOT NULL logiquement à la finalisation;
- UNIQUE `(marking_job_id, student_exam_id)`.

### `marking_question_results`

Une ligne par question d'une copie corrigée.

- PK `id`;
- `copy_result_id` NOT NULL, FK copie-result;
- `question_index` NOT NULL, index zéro ou un à fixer une fois et conserver;
- `state` NOT NULL (`incorrect`, `partial`, `correct`);
- `score_half_units` NOT NULL;
- `total_points` NOT NULL;
- UNIQUE `(copy_result_id, question_index)`.

Le total est volontairement répété depuis le snapshot comme valeur effectivement utilisée/validée par cette exécution. Il permet de prouver le résultat sans relancer l'algorithme; sa cohérence avec le snapshot doit être testée lors de l'écriture.

### `marking_answer_detections`

Une ligne par réponse de chaque question corrigée.

- PK `id`;
- `question_result_id` NOT NULL, FK question-result;
- `answer_index` NOT NULL;
- `detected_state` NOT NULL, contraint à 0/1;
- `mean_gray` NOT NULL, REAL correspondant au `float64` mesuré;
- UNIQUE `(question_result_id, answer_index)`.

L'état attendu reste dans le snapshot à `(question_index, answer_index)`.

### `marking_job_issues`

Incidents non rattachables honnêtement à une copie.

- PK `id`;
- `marking_job_id` NOT NULL, FK job;
- `input_page_number` NULL si inconnu;
- `kind` NOT NULL (par exemple QR illisible, page hors scope déjà fatale dans le contrat actuel);
- `detail` NULL et sanitisé;
- pas d'unicité artificielle tant que plusieurs incidents identiques sont possibles.

### `marking_job_artifacts`

Métadonnées des artefacts, séparées du résultat métier.

- PK `id`;
- `marking_job_id` NOT NULL, FK job;
- `kind` NOT NULL (`corrected`, `mark_table`, `unprocessed`);
- `storage_key` NOT NULL;
- `created_at` NOT NULL;
- `purged_at` NULL;
- UNIQUE `(marking_job_id, kind)` pour les trois artefacts actuels.

### Métadonnées minimales sur `marking_jobs`

Conceptuellement, les nouveaux jobs de résultat doivent porter `result_schema_version`, `marking_algorithm_version` et `detection_threshold`. Ils distinguent les jobs legacy et expliquent le calcul sans construire un système de versioning complexe.

## 12. Intégrité, ownership et suppressions

Les FK seules ne suffisent pas. Les protections doivent imposer :

- résultat copie et job du même `user_id`;
- `student_exam` du même user;
- `student_exam.exams_generated_id = marking_job.exam_generated_id`;
- enfant question rattaché à une copie-result valide;
- enfant réponse rattaché à une question-result valide;
- unicités positionnelles par tentative.

Le pattern adapté au dépôt est une création `INSERT SELECT` ownership-aware complétée par des triggers INSERT/UPDATE pour empêcher un accès SQL direct incohérent. Les tables enfant n'ont pas forcément besoin de répéter `user_id` si toutes les lectures traversent le parent; si la convention de scope direct par user est conservée, il faut le répéter et le contrôler par trigger.

Une génération ou un `student_exam` ayant des résultats historiques ne doit pas être supprimé implicitement : RESTRICT est cohérent. Les enfants question/réponse/issues/artifacts peuvent avoir CASCADE vers leur résultat/job uniquement lors d'une suppression explicite et coordonnée de cet historique. En revanche, la purge automatique ne doit plus supprimer un job success et déclencher cette cascade.

## 13. Artefacts PDF

Les PDF ne doivent plus être la seule source historique : les données DB deviennent l'autorité métier.

Recommandation mixte :

- `corrected.pdf` durable, au moins tant que les scans sources ne sont pas eux-mêmes conservés. Il contient les pages scannées alignées et annotées; il n'est pas honnêtement régénérable depuis les seuls snapshots et résultats DB puisque l'upload original est supprimé;
- `mark-table.pdf` dérivé des résultats persistés et des libellés du snapshot. Il peut être régénérable fonctionnellement, mais pas nécessairement bit-à-bit avec une future version Typst; sa conservation durable exacte est donc une décision de politique/produit, pas une condition d'intégrité des notes;
- `corrected_NOT.pdf` est un artefact de diagnostic des pages non attribuées. Il doit être conservé au moins pendant la résolution/revue, avec une politique explicite; il ne remplace pas les issues structurées.

Le chantier historique Typst/images restant incomplet, il ne faut pas promettre une régénération identique des PDF. La persistance des données peut néanmoins précéder ce chantier si `corrected.pdf` est traité comme artefact durable.

## 14. Rétention sept jours

Actuellement `completed_at` est fixé au passage success/failed. `ListExpiredMarkingJobs` sélectionne les deux états après sept jours; la purge supprime le workspace puis la ligne job. Le handler de progression supprime même immédiatement un job failed lorsqu'il est consulté.

Cette politique est incompatible avec un job success devenu racine d'un historique durable. Évolution conceptuelle recommandée :

- conserver les jobs success et leurs résultats;
- purger séparément les temporaires et artefacts explicitement non durables;
- conserver `corrected.pdf` selon la politique durable retenue;
- conserver les failed assez longtemps pour diagnostic, puis permettre leur suppression si aucun résultat utile n'existe;
- rendre la durée d'artefacts/failed configurable ultérieurement;
- ne plus utiliser `DeleteMarkingJob` comme effet de bord du polling failed.

Les résultats et le job historique doivent avoir une rétention distincte du workspace.

## 15. Statistiques et rôle produit

LazyMarking doit conserver les copies corrigées, les notes, les résultats de classe et le diagnostic pédagogique. Pronote reste hors périmètre et demeure le lieu de publication officielle.

Moyenne, médiane, écart-type population, scores de compétences et scores thème-compétence sont dérivables des résultats individuels et des tags du snapshot. Ils ne doivent pas être persistés dans le premier modèle. Les recalculer évite les incohérences après correction/revue. La formule/algorithm version permet d'expliquer les résultats historiques; si les formules changent, le système pourra présenter « calcul actuel » versus artefact historique sans avoir dupliqué tous les agrégats.

## 16. Historique minimal de l'algorithme

Une chaîne/version courte du moteur et le seuil réellement appliqué suffisent au premier jalon. Elle doit identifier au minimum la sémantique : homographie/lecture moyenne ROI, seuil 150 et règle de score 0/moitié/total. Il n'est pas nécessaire de versionner chaque fonction ou dépendance.

Dans six mois, une détection s'explique par : snapshot et coordonnées historiques, `mean_gray`, seuil du job, `detected_state`, version du moteur, puis vecteur attendu et règle de score. Pour une preuve visuelle, le PDF corrigé ou un futur artefact de scan reste nécessaire.

## 17. Volume

Pour 30 élèves, quelques dizaines de questions et plusieurs réponses par question, le modèle produit environ 30 résultats copie, quelques centaines à quelques milliers de résultats question, et quelques milliers de détections. Ce volume est faible pour SQLite, même avec plusieurs tentatives, sous réserve d'index sur FK et unicités. Les images/PDF dominent très largement le stockage; les lignes numériques et textuelles courtes ne justifient pas un blob JSON opaque.

## 18. Atomicité recommandée

1. Une copie doit être persistée entièrement dans une transaction : outcome, questions et réponses, ou rien. Une copie en erreur reçoit ensuite une ligne outcome cohérente dans une transaction séparée.
2. Le parallélisme peut conserver une transaction courte par élève; il ne faut pas garder une transaction SQLite ouverte pendant OpenCV/Typst.
3. Tous les `student_exam` attendus doivent avoir un outcome terminal avant que le job devienne success. « Terminal » inclut incomplete/not_seen/error, pas seulement corrected.
4. Les agrégats de contrôle (nombre attendu, somme des outcomes, sommes question/copie) sont vérifiés avant finalisation.
5. Les artefacts sont générés après persistance des résultats, puis déplacés vers leur emplacement final. Une transaction finale enregistre leurs métadonnées et terminalise le job. Un échec fichier laisse le job running/failed et des fichiers orphelins récupérables, jamais un job success sans résultats complets.

Une transaction SQL ne peut pas rendre atomique le filesystem; le protocole doit donc utiliser fichiers temporaires, renommage final et cleanup/recovery idempotent.

## 19. Rerun et retry

Un failed relancé crée un nouveau job; l'ancien reste un historique technique jusqu'à sa rétention explicite. Une correction success relancée crée également un nouveau job et une nouvelle tentative indépendante. Les deux résultats sont conservés et présentés avec date/job/version. Aucun « actif », remplacement ou dernière tentative implicite n'est recommandé au premier jalon.

## 20. Compatibilité legacy

- Les jobs pré-0036 avec génération NULL ne peuvent pas être enrichis honnêtement; aucun backfill depuis nom de fichier ou PDF.
- Les jobs existants post-0036 mais antérieurs à la persistance n'ont pas davantage de résultats structurés.
- `result_schema_version NULL` permet conceptuellement de les reconnaître.
- Ils restent soumis à leur lifecycle/recovery/purge historique.
- Seuls les jobs créés après activation du nouveau contrat exigent un ensemble complet d'outcomes avant success.
- Aucun PDF legacy n'est analysé pour reconstruire des notes.

## 21. Roadmap d'implémentation proposée

1. Définir le schéma minimal des résultats et contraintes ownership, sans encore changer les PDF; commencer par `marking_copy_results`, questions et réponses.
2. Faire produire au moteur un résultat structuré non destructif contenant `QuestionMark`, états détectés et moyennes, avec tests d'équivalence au runtime actuel.
3. Persister atomiquement une copie complète et les outcomes non corrigés.
4. Finaliser le job seulement après couverture de toutes les copies attendues et persistance des issues.
5. Ajouter la lecture historique et recalculer les statistiques depuis DB + snapshots.
6. Séparer rétention job/résultats/workspace et définir les artefacts durables.
7. Ajouter ultérieurement workflow d'ambiguïté/revue, sans modifier rétroactivement les tentatives originales.

Le premier jalon recommandé est donc le schéma et ses invariants, accompagné d'un adaptateur de test capable d'écrire un résultat synthétique complet; pas encore l'intégration OpenCV complète.

## 22. Tests de forte valeur à prévoir

- copie complète : indices, états attendus du snapshot, détections, mesures, scores et total;
- question plein points, moitié et zéro, y compris mauvaise réponse cochée;
- plusieurs élèves dans un job, unicité par `(job, student_exam)`;
- ordre mélangé des pages sans modification des indices historiques;
- copie incomplète, non vue, erreur technique et page QR illisible;
- job partiel : tous les outcomes persistés, compte corrigé/non corrigé exact;
- refus de finaliser si une copie attendue manque ou si un enfant est incomplet;
- deux jobs success pour la même génération et le même élève, deux historiques conservés;
- retry d'un failed par nouveau job;
- ownership Alice/Bob et génération différente, via API SQLC et accès SQL direct;
- rollback au milieu des questions/réponses : aucune demi-copie persistée;
- incohérence score copie/somme questions refusée ou détectée;
- purge des temporaires sans suppression du job success/résultats;
- suppression explicite et cascades enfant contrôlées;
- lecture identique après fermeture/réouverture de la base;
- recalcul moyenne/médiane/écart-type/compétences depuis résultats persistés;
- legacy NULL lisible/purgeable sans prétendre avoir des résultats.

## 23. Validation future avec vraies copies

Les scans réels pourront être utilisés uniquement dans une suite d'intégration opt-in locale, comme les tests réels existants :

1. exécuter le pipeline actuel sur une copie privée;
2. capturer une représentation canonique runtime (student_exam, indices, états détectés, mesures, QuestionMark, score total);
3. persister dans une base temporaire;
4. relire et comparer champ par champ au runtime;
5. vérifier que les PDF produits restent identiques ou fonctionnellement équivalents;
6. ne versionner ni scan, ni base réelle, ni résultat nominatif; CI limitée aux fixtures synthétiques.

## 24. Priorité

La persistance reste P1 : aujourd'hui la donnée métier qui explique une note disparaît et la purge supprime l'unique représentation restante. Elle doit être implémentée avant le chantier de référence historique Typst/images. Cette séquence est sûre à condition de ne pas déclarer les PDF régénérables et de conserver durablement `corrected.pdf` tant que les scans/ressources de rendu ne sont pas historiquement complets.
