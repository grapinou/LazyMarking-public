# Audit et conception — ambiguïtés et revue humaine Marking

## 1. Contrat actuel et données disponibles

Le moteur mesure dans `GetAnswerDetections` la moyenne en niveaux de gris de la ROI carrée centrée sur `CircleValidated.Position`, de demi-largeur `Radius/2`. `answerDetectionFromMean` conserve le contrat historique strict : `mean_gray < 150` donne `detected_state=1`, et `mean_gray >= 150` donne `detected_state=0`. La valeur 150 est aussi snapshotée dans `marking_jobs.detection_threshold` pour les jobs nouveau-format.

Le schéma durable contient :

- `marking_jobs` : utilisateur, génération, versions schéma/algorithme, seuil, lifecycle technique et chemins des PDF ;
- `marking_copy_results` : un outcome terminal par `(job, student_exam)`, pages et score de copie en demi-points ;
- `marking_question_results` : index zéro-based, état, score et total par question ;
- `marking_answer_detections` : `question_result_id`, `answer_index` zéro-based, `detected_state` immuable conceptuellement et `mean_gray` réel.

La jointure answer → question result → copy result retrouve sans ambiguïté le job, l'utilisateur, la génération et le `student_exam`. L'état attendu, les points, les libellés, compétences et thèmes sont dans `student_exam_content`, sans dépendance aux tables vivantes. `student_exam_page_content`, ordonné par `page`, contient les listes de questions/réponses et leurs centres/rayons dans le repère pixel de la page alignée.

Le mapping d'une détection vers `page_exam` et sa ROI est reconstructible sans table vivante : on calcule son offset global depuis `(question_index, answer_index)` dans le QCM snapshot, puis on le localise dans les blocs `PageContent.Answers` parcourus par page, exactement comme le pipeline runtime concatène aujourd'hui les détections. Ce mapping mérite un helper unique testé ; il ne faut pas réinventer des offsets dans le handler UX.

## 2. Ce qui manque pour une revue visuelle

Le workspace success conserve `corrected.pdf`, `mark-table.pdf` et éventuellement `corrected_NOT.pdf`. En revanche, `MarkingStudentExam` supprime :

- les scans PNG issus de l'upload ;
- chaque image alignée par homographie après l'avoir annotée et convertie en PDF ;
- les PDF individuels et les temporaires Typst.

`corrected.pdf` provient d'une page alignée déjà modifiée par `DrawMarking`, puis rasterisée/convertie. Les marques de correction peuvent entourer les cases et la note affichée peut devenir obsolète après revue. Ce PDF n'est donc ni la source exacte de la mesure MeanGray, ni une base honnête de régénération.

Il manque un artefact durable : la page alignée **non annotée**, exacte source de `GetAnswerDetections`. Elle permet de générer à la demande un crop autour de la ROI réellement mesurée et de redessiner ultérieurement une copie corrigée cohérente, sans nouvelle homographie.

## 3. Ambiguïté : modèle recommandé

Pour la V1, MeanGray est suffisant comme **heuristique de présélection**, pas comme estimation complète de confiance. Le modèle recommandé est une distance symétrique au seuil snapshoté du job :

```text
ambiguous = abs(mean_gray - detection_threshold) <= ambiguity_delta
```

La borne est inclusive ; une valeur exactement 150 reste automatiquement classée `unchecked` par le moteur historique et devient candidate à la revue dès que `delta >= 0`. La couche d'ambiguïté ne change jamais `detected_state`.

Cette formulation est équivalente à une bande symétrique fixe une fois le delta choisi, mais évite de recopier 150 et reste historiquement correcte si une future version de moteur possède un autre seuil. Les seuils asymétriques ne sont pas justifiés sans observation réelle. Ils restent une évolution possible si les faux positifs et faux négatifs humains montrent une asymétrie nette.

`ambiguity_delta` doit être snapshoté sur `marking_jobs` pour les nouveaux jobs utilisant la revue. Il est nullable pour les jobs legacy et doit être immuable après création. L'ambiguïté d'une détection reste dérivée de `mean_gray`, `detection_threshold` et `ambiguity_delta` : aucun booléen `ambiguous` redondant n'est persisté. Ainsi, le corpus exact présenté par un job reste explicable même si la valeur par défaut applicative évolue.

Aucune valeur de delta ne doit être fixée avant calibrage. Les candidats ±5, ±10 et ±15 servent uniquement à produire des comptes comparatifs.

## 4. Limites de MeanGray

La moyenne distingue utilement une ROI très sombre d'une ROI très claire et fournit une distance simple au seuil existant. Elle ne distingue pas correctement :

- un petit trait noir sur fond blanc d'un gris plus uniforme de même moyenne ;
- une gomme, une rature et une coche légère de distributions comparables ;
- un biais global de scan sombre/clair d'un signal local de case ;
- la forme, la variance, la proportion de pixels sombres ou la localisation du trait.

Une case noire ou blanche franche sera généralement loin du seuil. Crayon léger, trait partiel, gomme, rature, gris imprimé et bruit local peuvent être proches ou éloignés de 150 selon leur surface : MeanGray seul ne garantit pas de les signaler tous. Pour un premier workflow assisté par crop humain, la distance à 150 est néanmoins une présélection proportionnée. Il ne faut pas lancer un nouveau moteur vision avant d'avoir observé ses erreurs réelles.

## 5. Trois états distincts

Le modèle doit conserver :

- `detected_state` : décision automatique originale, jamais écrasée ;
- `reviewed_state` : décision humaine 0/1, présente seulement après revue ;
- `effective_state` : `COALESCE(reviewed_state, detected_state)`.

Une revue est matérialisée même lorsqu'elle confirme le moteur. L'absence de ligne de revue signifie « pas revue » ; une ligne avec `reviewed_state == detected_state` signifie « revue et confirmée » ; une ligne avec une valeur différente signifie « revue et modifiée ».

Le statut de revue du job est dérivé :

- `no_review_needed` : zéro détection ambiguë selon le delta snapshoté ;
- `pending` : au moins une ambiguïté sans revue ;
- `completed` : toutes les ambiguïtés ont une ligne de revue.

Il ne faut pas ajouter `clear/ambiguous/reviewed` sur chaque détection : `clear/ambiguous` est dérivable, tandis que `reviewed` est prouvé par la ligne de revue.

## 6. Modèle DB recommandé

### Évolution de `marking_jobs`

- `ambiguity_delta REAL NULL`, `CHECK >= 0`, complet seulement pour les jobs activant la revue, immuable avec le seuil/version ;
- `review_revision INTEGER NOT NULL DEFAULT 0`, monotone ;
- `artifacts_revision INTEGER NOT NULL DEFAULT 0`, dernière révision DB représentée simultanément par `corrected.pdf` et `mark-table.pdf`.

Les jobs historiques gardent `ambiguity_delta NULL` et ne reçoivent pas artificiellement une liste d'ambiguïtés.

### `marking_answer_reviews`

- `id INTEGER PRIMARY KEY` ;
- `answer_detection_id INTEGER NOT NULL UNIQUE`, FK vers `marking_answer_detections(id)`, `ON DELETE CASCADE` avec la suppression explicite du job/résultat ;
- `reviewer_user_id INTEGER NOT NULL`, FK `users(id) ON DELETE RESTRICT` ;
- `reviewed_state INTEGER NOT NULL CHECK IN (0,1)` ;
- `reviewed_at TIMESTAMP NOT NULL` ;
- `revision INTEGER NOT NULL CHECK >= 1` pour l'optimistic locking ;
- pas de commentaire obligatoire en V1.

Une table séparée est préférable à des colonnes sur `marking_answer_detections` : la mesure automatique reste immutable, l'existence d'une revue est explicite, l'ownership peut être protégé par trigger à travers answer/question/copy/job, et un journal append-only pourra remplacer/compléter cette table plus tard. Pour la V1, une seule revue courante par détection suffit ; les modifications ultérieures mettent à jour la ligne avec `WHERE revision = ?`, incrémentent la révision et sont donc détectables en concurrence. L'ajout futur d'un audit log n'exige pas de casser l'identité de la détection.

### `marking_aligned_pages`

- `id INTEGER PRIMARY KEY` ;
- `user_id INTEGER NOT NULL` ;
- `copy_result_id INTEGER NOT NULL`, FK `marking_copy_results(id) ON DELETE CASCADE` ;
- `page_exam INTEGER NOT NULL CHECK >= 1` ;
- `storage_key TEXT NOT NULL`, relative et confinée au workspace `marking-<job>` ;
- `width`, `height` strictement positifs ;
- `sha256` hexadécimal ;
- `created_at TIMESTAMP NOT NULL` ;
- `UNIQUE(copy_result_id, page_exam)`.

L'ownership doit être garanti à l'insertion/mise à jour par trigger : `user_id == copy.user_id == job.user_id`, et la page appartient au `student_exam` de la copie. Les fichiers sont des PNG alignés non annotés, publiés atomiquement avant metadata, validés par un resolver sécurisé analogue aux références Exam.

Les enfants question/answer ne répètent pas `user_id` ; leur parent est l'autorité. Aucun lien aux banques vivantes Questions/Answers/Points n'est ajouté.

## 7. Scores originaux et effectifs

Le score automatique original est entièrement reconstructible : `detected_state` original + états attendus/points du snapshot + règle d'algorithme versionnée reproduisent `CountingPoints`. Il n'est donc pas nécessaire de dupliquer `original_score` à chaque niveau.

Les colonnes actuelles `marking_question_results.state/score_half_units` et `marking_copy_results.score_half_units` doivent représenter le **résultat effectif courant**. Avant revue, il est identique à l'automatique ; après revue, il est recalculé depuis tous les `effective_state`. La trace automatique reste intacte dans les detections et est recalculable.

Une validation humaine ne met jamais à jour seulement le total. Dans une transaction courte :

1. vérifier ownership, job success, ambiguity du candidat, et révision attendue ;
2. insérer/mettre à jour la revue conditionnellement ;
3. reconstruire le vecteur effectif complet de la question depuis DB + snapshot ;
4. réappliquer la règle pure 0/moitié/totalité ;
5. mettre à jour état/score de la question ;
6. sommer les questions pour mettre à jour le score de copie ;
7. incrémenter `marking_jobs.review_revision` ;
8. laisser `artifacts_revision < review_revision`, donc artefacts explicitement obsolètes.

Les invariants CHECK existants continuent de défendre state/score. Les agrégats classe ne sont pas persistés : moyenne, médiane, écart-type, compétences et thèmes sont recalculés depuis les scores effectifs individuels et le snapshot.

## 8. Lifecycle technique et lifecycle de revue

Le lifecycle `running/success/failed` ne doit pas changer. `success` signifie que le traitement automatique, les résultats structurés et les artefacts automatiques sont complets. La revue est un lifecycle orthogonal dérivé.

Avec ambiguïtés, le job peut donc être `success` et `review=pending`. L'UI doit clairement afficher « correction automatique terminée — N réponses à vérifier ». La publication/considération « finale » dans LazyMarking doit demander que la revue soit terminée, mais consulter le résultat automatique ne doit pas être bloqué.

Ce choix évite un nouvel état technique, conserve recovery/purge, et permet une revue ultérieure sans garder un job `running`. Les ambiguïtés doivent être fortement incitées, voire requises avant export final, mais pas confondues avec une erreur de traitement.

## 9. UX minimale

Après succès : compteurs copies corrigées/issues et ambiguïtés restantes, puis bouton « Vérifier les réponses ».

L'écran séquentiel montre : identité de copie nécessaire au professeur, numéro de question/réponse, crop agrandi de la page alignée, décision automatique formulée simplement, boutons « Non cochée » / « Cochée », puis « Valider et suivante ». MeanGray et seuil restent cachés par défaut mais disponibles dans un détail diagnostic.

Le crop est généré à la demande depuis `marking_aligned_pages`, avec marge supérieure à la ROI exacte `centre ± radius/2` afin de donner du contexte. Il ne faut ni stocker un crop par ambiguïté ni utiliser le PDF annoté. Le resolver page + mapping snapshot garantit que l'image affichée est celle réellement mesurée.

## 10. PDF, mark-table et protocole filesystem

Une modification humaine rend `corrected.pdf`, `mark-table.pdf` et les statistiques qu'ils contiennent obsolètes. La direction recommandée est de conserver les noms canoniques et de les régénérer depuis : pages alignées non annotées + états/scores effectifs DB + snapshots. Un PDF automatique incohérent ne doit pas continuer à être servi comme final.

Protocole : la transaction DB valide la revue et incrémente `review_revision`; l'UI refuse le téléchargement final tant que `artifacts_revision != review_revision`. Un worker génère les deux PDF dans des temporaires, les vérifie, les renomme atomiquement, puis avance `artifacts_revision` conditionnellement si `review_revision` n'a pas changé. En cas d'échec filesystem, la DB reste autoritative et l'état « artefacts à régénérer/échec » est visible par l'écart de révisions. Il n'existe aucune fausse atomicité SQLite/filesystem.

La régénération doit devenir DB-driven. Aujourd'hui `mark-table.pdf`, moyenne, médiane, écart-type, compétences et thèmes sont produits depuis `[]MarkExam` runtime ; ce chemin doit être extrait en fonctions capables de reconstruire les mêmes view models depuis résultats effectifs + snapshots avant d'ouvrir l'UX de revue en production.

## 11. Concurrence et traçabilité

Pour une première revue, l'INSERT sous `UNIQUE(answer_detection_id)` garantit qu'un seul onglet gagne. Pour modifier une revue, `UPDATE ... WHERE revision = expected_revision` ; zéro ligne affectée demande un rechargement. La transaction vérifie aussi la révision globale du job lorsqu'elle publie ses nouveaux scores.

Le minimum de traçabilité est `reviewed_state`, `reviewed_at`, `reviewer_user_id`, `revision`. Comparé à `detected_state`, cela distingue confirmation et override et conserve les faux positifs/faux négatifs utiles au calibrage futur. Un commentaire libre n'est pas nécessaire en V1. Si un audit complet de toutes les modifications devient requis, une table append-only d'événements pourra être ajoutée ; la table séparée actuelle ne ferme pas cette voie.

## 12. Calibration privée opt-in

Une suite locale, skipped par défaut et utilisant une copie temporaire de la DB/scans, doit exécuter le pipeline détaillé sans noms, formulations, réponses attendues ou images dans les logs. Elle agrège uniquement :

- nombre de détections ;
- min/max ;
- quantiles (p1, p5, p25, médiane, p75, p95, p99) ;
- histogramme de MeanGray par bins définis ;
- distribution de `abs(mean_gray - 150)` ;
- nombres inclus dans ±5, ±10, ±15 ;
- comptes séparés selon `detected_state`, sans identité de copie.

Le calibrage doit ensuite faire relire humainement un échantillon sécurisé autour et hors bande pour estimer cases douteuses manquées, confirmations et overrides. La distribution seule ne donne pas la vérité terrain. Les données de revue ultérieures permettront de mesurer faux positifs/faux négatifs sans machine learning.

Aucune analyse privée n'a été lancée pendant cet audit.

## 13. Roadmap recommandée

1. Migration minimale : `ambiguity_delta`, révisions artefacts, `marking_answer_reviews`, `marking_aligned_pages`, contraintes et ownership ; tests Up/Down.
2. Conservation atomique des pages alignées non annotées et resolver sécurisé, sans UX ni changement de score.
3. Fonctions pures DB/snapshot : effective states, recalcul question/copie, transaction de revue et concurrence.
4. Lecture historique/statistiques depuis DB effective ; preuve d'équivalence avec le runtime actuel.
5. Calibration opt-in sur scans réels et choix documenté du delta par défaut.
6. UX séquentielle de revue avec crops à la demande.
7. Régénération cohérente `corrected.pdf` + `mark-table.pdf`, révisions et tests d'échec filesystem.

Le premier jalon doit rester strictement schéma/intégrité, sans choisir encore un delta produit ni brancher l'UX.

## 14. Tests de forte valeur à prévoir

- clair checked, clair unchecked, ambigu dans la bande et MeanGray exactement 150 ;
- absence de mutation du `detected_state` ;
- revue confirmée sans changement et overrides 0→1 / 1→0 ;
- recalcul complet question puis copie, full/partial/incorrect ;
- original automatique reconstructible après override ;
- statistiques classe/compétences/thèmes basées sur états effectifs ;
- ownership cross-user/cross-job/cross-generation ;
- double INSERT et UPDATE concurrent avec révision obsolète ;
- job technique success avec review `no_review_needed`, `pending`, `completed` ;
- crop issu de la bonne page/ROI, hash/dimensions et confinement ;
- page alignée absente/corrompue ;
- PDF et mark-table refusés lorsqu'obsolètes puis cohérents après régénération ;
- rollback transactionnel sur échec du recalcul ;
- legacy `ambiguity_delta NULL` inchangé.

## 15. Priorité

Après fermeture des P1 d'intégrité historique et de persistance, ambiguïté + revue humaine est un **P2 élevé de qualité produit**. Il améliore fortement la confiance pédagogique, mais ne corrige plus une rupture structurelle de propriété, d'historicité ou de durabilité. Il doit précéder une clôture UX ambitieuse de Marking et tout export présenté comme final validé.

## 16. Périmètre de cet audit

Seul ce rapport a été créé. Aucun code, SQL, migration, template, algorithme, seuil, score, PDF ou donnée privée n'a été modifié ou exécuté.
