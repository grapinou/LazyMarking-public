# Audit Exams / Évaluations

## Résumé exécutif

Une évaluation est actuellement une configuration possédée composée d'un nom, d'un QCM, d'une classe, d'une période et d'une année. Tous ces parents sont obligatoires et vivants. La création d'une génération fige ensuite, pour chaque élève traité, le QCM individualisé complet dans `student_exam_content`; elle ne fige toutefois pas les libellés de l'évaluation, de la classe ni le périmètre de classe au niveau de `exams`.

L'ownership courant est solide (SQL ownership-aware et triggers depuis 0030), la liste ne comporte pas de N+1 et les trois niveaux d'individualisation sont préservés. Deux points métier doivent précéder la refonte UX : la suppression d'une évaluation cascade aujourd'hui sur tout son historique généré, et une évaluation déjà générée reste entièrement modifiable alors que son contenu sérialisé et ses métadonnées vivantes peuvent diverger. Verdict : **B — quelques correctifs métier/robustesse avant UX**.

## Cartographie

### Tables et migrations

| Concept | Table | Migration d'origine | Rôle |
|---|---|---|---|
| Évaluation | `exams` | 0022 | configuration nom + QCM + classe + période + année |
| Classe | `class_codes` | 0015, reconstruite par 0029 | groupe pédagogique nommé |
| Élève | `students` | 0016, reconstruite par 0029 | identité possédée |
| Appartenance classe | `student_class_codes` | 0017 | relation vivante élève-classe |
| Année | `years` | 0020, reconstruite par 0029 | référentiel possédé |
| Période | `periods` | 0021, reconstruite par 0029 | référentiel possédé |
| Génération | `exams_generated` | 0023 | une exécution, progression et statut |
| Copie élève | `student_exam` | 0024 | élève inclus dans une génération |
| Snapshot logique de copie | `student_exam_content` | 0025 | QCM individualisé sérialisé + nombre de pages |
| Snapshot géométrique par page | `student_exam_page_content` | 0026 | cercles/questions/réponses sérialisés par page |
| Job de correction | `marking_jobs` | 0027, 0028, 0031 | progression de correction, sans FK directe vers `exams` |

Les structs sqlc correspondants sont `db.Exam`, `ExamsGenerated`, `StudentExam`, `StudentExamContent`, `StudentExamPageContent`, `ClassCode`, `Student`, `StudentClassCode`, `Year` et `Period` dans `internal/db/models.go`.

### Code applicatif

- CRUD : `internal/handlers/exams`, routes `/dashboard/exams`, `/add`, `/edit`, `/delete`.
- Génération complète et mini : `internal/handlers/generateExams`.
- Construction d'une copie : `tools.BuildQcmStudentCtx`, `GetQCMQuestionsAnswersCtx`, `BuildQuestionCtx` et helpers Typst/QR/OpenCV.
- Récupération au démarrage : `tools.RecoverRunningExamGenerations`, appelée par `cmd/server/main.go`.
- Correction : `internal/handlers/marking` et `tools.MarkingStudentExam`, consommateurs des snapshots de copies.
- Templates : `internal/templates/exams/*` et `internal/templates/generateExam/*`.
- Données de vue : `ExamPageData`, `GenerateExamPageData`, toutes deux encore centrées sur `ExtraData`.
- Tests principaux : `handlers/exams/handlers_test.go`, `db/examRelationshipIntegrity_test.go`, tests `generateExams`, `recoverExamGenerations`, concurrence, QR et marking.

## Modèle relationnel final

`exams` contient six champs métier obligatoires : `name`, `qcm_id`, `class_code_id`, `period_id`, `year_id`, `user_id`. Le nom est non vide après `trim` au niveau CHECK. L'unicité est `(name, qcm_id, class_code_id, user_id)` : année et période ne participent pas à la clé.

Toutes les FK parentes sont `NOT NULL` et `ON DELETE RESTRICT` : utilisateur, QCM, classe, année et période. Aucun `ON UPDATE` particulier ni index explicite dédié n'est défini. Un QCM peut donc être réutilisé par plusieurs évaluations.

La descendance est au contraire destructive :

```text
exams
  └─CASCADE─ exams_generated (UNIQUE exam_id,user_id)
      └─CASCADE─ student_exam
          ├─CASCADE─ student_exam_content
          └─CASCADE─ student_exam_page_content
```

`student_exam.student_id` n'a pas de règle de suppression explicite (NO ACTION). `marking_jobs` n'est pas relié à cette chaîne par FK.

## Ownership

Les lectures et mutations CRUD sont filtrées par `user_id`. `CreateExam` et `UpdateExam` vérifient en SQL l'ownership des quatre parents; `GetExamByID` et la liste revérifient également ces parents. `HandleOwnedLookupError` et `HandleOwnedMutationRows` traduisent absence, cible étrangère et zéro ligne en 404.

Depuis 0030, les triggers `exams_owner_insert/update`, `generated_exams_owner_*`, `student_exams_owner_*`, `student_exam_content_owner_*` et `student_exam_pages_owner_*` imposent aussi la cohérence relationnelle. Les lectures de copies et de génération sont filtrées par utilisateur. La récupération de classe exige l'ownership de l'élève, de la relation et de la classe.

Conclusion : ownership correct dans l'état final. Comme ailleurs, une base ayant connu des données incohérentes avant 0030 pourrait mériter un diagnostic ponctuel, mais les lectures modernes ne les exposent pas.

## Création

Le GET charge, en quatre requêtes globales ownership-aware, les QCM, classes, années et périodes. Le POST parse les quatre IDs puis exécute `CreateExam`, un `INSERT SELECT` qui refuse tout parent absent ou étranger. Zéro ligne devient 404; unicité ou nom vide redirige vers le message d'erreur générique.

Le handler ne fait pas `TrimSpace` : le CHECK refuse un nom entièrement blanc, mais les espaces périphériques sont conservés. Aucun champ n'est optionnel. Un QCM vide et une classe vide sont acceptés à la création. La classe vide est refusée proprement seulement au lancement de la génération; le QCM vide échoue plus tard lors de la construction. Aucun doublon n'est autorisé pour le même nom/QCM/classe/utilisateur, même si année ou période diffèrent.

## Modification

Tous les champs sont modifiables : nom, QCM, classe, période et année. L'ownership de la cible et des nouveaux parents est protégé. Il n'existe pas de verrou après génération ni d'état fonctionnel sur `exams`.

Cela peut faire diverger une génération existante : son `student_exam_content` conserve les copies déjà produites, mais le nom, la classe et les autres informations affichées restent lus depuis l'évaluation vivante. Changer le QCM ou la classe ne réécrit pas les copies existantes. Le nom du PDF final est aussi reconstruit depuis les noms vivants sur la page de succès, alors que le fichier a été créé avec les noms au moment de la génération.

## Suppression

`DeleteExam` est ownership-aware et traite correctement zéro/une ligne. Mais supprimer une évaluation, même déjà générée ou corrigée, supprime par cascade `exams_generated`, toutes les `student_exam`, leurs contenus sérialisés et leurs géométries. Les jobs de correction peuvent survivre sans relation FK. Le template actuel ne prévient pas de cette perte historique.

Le contrat est donc techniquement atomique mais métierment risqué : il convient à une configuration inutilisée, pas à un historique réel. C'est le premier correctif recommandé : protéger une évaluation ayant une génération (RESTRICT ou suppression interdite applicativement avec garantie DB), tout en conservant une suppression simple pour une évaluation jamais générée.

## QCM vivant, snapshot et reproductibilité

Avant génération, l'évaluation référence le QCM vivant. Toute nouvelle génération consommerait sa composition et les questions/réponses/variantes courantes. `exams_generated` impose toutefois une seule génération par évaluation.

Lors de la génération, chaque copie obtient un snapshot JSON complet dans `student_exam_content`, incluant l'ordre choisi, la formulation, les réponses et les coordonnées détectées; la correction relit ce snapshot. Les pages conservent aussi leur géométrie. Ainsi une copie déjà produite reste corrigeable malgré des changements ultérieurs de la banque, tant que sa chaîne DB n'est pas supprimée.

La reproductibilité est **partielle** : le contenu exact d'une copie existante est figé, mais l'environnement pédagogique et la cohorte ne le sont pas intégralement, le PDF final est un artefact de workspace et aucune seed ne permet de régénérer à l'identique. Une nouvelle génération à partir du QCM vivant ne reproduirait pas nécessairement l'ancienne.

## Individualisation

`BuildQcmStudentCtx` appelle `GetQCMQuestionsAnswersCtx`. Cette fonction lit les IDs par position, applique `ShuffleSlice`, puis les workers indexés préservent cette permutation. `BuildQuestionCtx` choisit aléatoirement la question principale ou une variante via `GetRandomQuestionByQuestionID`. `GetQuestionAnswerCtx` ou `GetAltQuestionAltAnswerCtx` mélange les réponses. Les trois niveaux sont donc préservés : ordre des familles, formulation, réponses.

Les données propres à Exam intervenant dans une copie sont le nom, le QCM et la classe; la classe fournit la cohorte et son nom. Année et période ne participent pas au contenu généré.

## Classes et élèves

L'évaluation cible une **classe vivante**, pas une liste figée. Au lancement, `GetAllStudentsByClassCodeID` relit les appartenances actuelles. Une classe vide est autorisée dans l'évaluation mais bloque la génération avec un message métier. Ajouter ou retirer un élève avant génération change la cohorte; après génération, les `student_exam` figent les élèves effectivement traités. La suppression d'un élève déjà présent dans `student_exam` est bloquée par la FK sans cascade.

La suppression d'une classe référencée par `exams` est RESTRICT. Le handler redirige actuellement toute erreur DB comme « champ utilisé », sans distinguer FK et véritable panne DB; le message mentionne improprement « utilisé par une classe ».

## Années et périodes

`years` et `periods` sont deux référentiels indépendants, obligatoires, possédés et uniques par `(name,user_id)`. Ils ne sont pas reliés entre eux et ne pilotent aucune logique de génération observée; ils qualifient seulement l'évaluation et sa liste.

Leur suppression est RESTRICT lorsqu'un examen les utilise. Leurs handlers, comme celui des classes, classent aujourd'hui toute erreur DELETE comme « utilisée », au lieu de distinguer une contrainte FK d'une erreur DB. Leur UX est indépendante mais conserve l'ancien motif CRUD.

## Génération PDF, jobs et reprise

La génération complète est lancée par POST. Après validation ownership de l'évaluation et lecture de la classe, elle crée `exams_generated`, un workspace `exam-<id>`, puis une goroutine de fond suivie par le `WaitGroup` serveur. Au plus cinq élèves sont traités simultanément, chacun avec un timeout de 60 secondes. Chaque worker crée la copie, le JSON de correction, les pages, QR codes et PDF. Les PDF sont fusionnés, les intermédiaires nettoyés et le statut passe de `running` à `success`; toute erreur place la génération en `failed` et nettoie le workspace.

La page de progression interroge le statut toutes les deux secondes. Sur échec, elle supprime le workspace et la ligne `exams_generated` (ce qui permet de réessayer). Au redémarrage, les générations restées `running` sont supprimées avec leur descendance et leurs workspaces. Les générations `failed` ne sont purgées que lorsque leur page de progression est consultée. Il n'y a ni reprise au milieu d'une génération ni queue persistante; le job appartient bien au domaine de génération Exam, tandis que `marking_jobs` est une infrastructure séparée de correction.

Le PDF final reste dans le workspace utilisateur et est servi par un handler à chemins contrôlés. Il n'est pas stocké dans la DB.

## UX actuelle

La liste est une table dense non responsive avec tableaux imbriqués, slices parallèles `Exams`/`Action`, actions icon-only, colonne « Edit/Sup », états et wording techniques. Elle affiche néanmoins nom, QCM, classe, période et année, ainsi que génération complète et mini. L'état vide est minimal. Années et périodes sont exposées depuis cette page.

Les formulaires utilisent une table pour quatre selects, `ExtraData`, IDs convertis en chaînes, aucun Annuler et des titres incohérents (« Modifier la question »). Add/Edit contiennent un JavaScript qui retire les guillemets. Le POST backend ne porte pas cette restriction. La confirmation Delete contient « Es-tu sur » et « C'est mon dernier mot », sans expliquer la cascade historique. Les listes vides de parents ne sont pas expliquées.

Le workflow fonctionne depuis Dashboard → Examens → création → génération, mais le passage QCM → Évaluation n'est pas matérialisé : aucune action « Utiliser ce QCM » avec présélection. Après génération, la progression et l'ouverture du PDF sont compréhensibles mais dépendent d'une popup. Le chemin vers scan/correction se fait via le bloc Correction, sans continuité contextuelle explicite avec l'évaluation.

## Données de vue et SQL/N+1

`ExamPageData.ExtraData` contient des collections, flags, lignes, actions parallèles et contextes de formulaire. Une migration vers `ExamContext`, `ExamListItem` et un petit modèle de formulaire aurait une valeur réelle. `GenerateExamPageData.ExtraData` contient statut/progression/URL et peut également être typé localement.

La liste ne présente **aucun N+1** : `GetExamsAllInfos` joint en une requête examens, QCM, classe, période et année, avec ownership sur chaque table. Les quatre listes du formulaire sont quatre requêtes fixes, indépendantes du nombre d'évaluations.

## Suppressions des parents

| Parent | Contrat DB | Handler actuel |
|---|---|---|
| QCM | RESTRICT | message métier structuré déjà corrigé |
| Classe | RESTRICT | 303 « utilisée » pour toute erreur DB, wording imprécis |
| Année | RESTRICT | 303 « utilisée » pour toute erreur DB |
| Période | RESTRICT | 303 « utilisée » pour toute erreur DB |

Les trois derniers devraient réutiliser la classification SQLite structurée déjà disponible afin de conserver 303 uniquement pour FK et 500 pour les autres erreurs.

## Tests actuels et manques utiles

Sont bien couverts : CRUD réussi, 404 cible absente/étrangère, parents étrangers au create/update, reads isolés, rows affected, cascade Exam→génération, triggers ownership dans la suite générale, transitions running/success/failed, récupération des générations interrompues, nettoyage/workspaces, concurrence et paniques, snapshots de correction, QR et individualisation/outils.

Manques à valeur réelle :

- suppression d'une vraie chaîne complète Exam→génération→copies→contenus, avec décision métier future de protection;
- blocage ou politique de modification après génération et test de cohérence des métadonnées;
- handlers de génération : QCM vide, classe vide, ownership et échec/retry en test HTTP intégré;
- classification FK/non-FK des suppressions classe/année/période;
- tests de PageData/liste/formulaires lors du futur refactor UX;
- test explicite d'un nom Exam avec guillemets après retrait du filtre frontend.

## Checklist pour la base réelle

- lister les examens et vérifier l'existence/ownership de QCM, classe, période et année;
- détecter les lignes inter-utilisateurs héritées d'avant 0030;
- comparer chaque `exams_generated` à son Exam et ses compteurs;
- vérifier `processed_students <= total_students` et cohérence du statut;
- trouver les `student_exam` sans contenu ou avec pages incomplètes;
- comparer les élèves générés à la cohorte historique connue, sans supposer que la classe actuelle est identique;
- repérer les générations `running`/`failed` anciennes et leurs workspaces;
- vérifier que chaque contenu JSON se désérialise et que chaque copie reste corrigeable;
- inventorier les jobs de correction sans historique Exam correspondant;
- vérifier l'existence des PDF/workspaces succès encore attendus.

## Priorités

### P0 — 0

Aucune fuite ownership ou corruption active démontrée.

### P1 — 2

1. **Suppression destructive de l'historique généré** — impact : perte des copies sérialisées et de leur base de correction; fichiers : migrations 0023–0026, `DeleteExamHandler`, confirmation; recommandation : protéger les Exams générés et tester l'atomicité; taille moyenne.
2. **Mutation libre après génération** — impact : métadonnées vivantes et snapshot divergent, nom de PDF attendu potentiellement différent, historique pédagogique ambigu; fichiers : `exams.sql`, handlers Exam/génération; recommandation : décider puis verrouiller les champs immuables dès qu'une génération existe; taille moyenne.

### P2 — 6

1. QCM vide accepté puis échec tardif de génération : prévalidation métier explicite; petite.
2. Nom non normalisé par `TrimSpace` et erreurs create/edit trop génériques; petite.
3. Génération `failed` nettoyée uniquement lorsque la page de progression est revisitée; définir une purge/retry observable; petite à moyenne.
4. Suppressions classe/année/période classent toute erreur DB comme FK; réutiliser le helper SQLite; petite.
5. Couverture HTTP de génération et des cas limites encore faible; ajouter des tests ciblés; moyenne.
6. View-data et UX Exams encore techniques (`ExtraData`, slices parallèles, scripts, icon-only, absence Annuler); refactor incrémental après les contrats métier; moyenne.

### P3 — 3

1. Navigation QCM → « Utiliser dans une évaluation » absente; petite.
2. Années/périodes/classes conservent leur ancien CRUD et wording; moyenne.
3. Succès de génération dépend d'une popup et wording/progression restent sommaires; petite.

## Roadmap incrémentale

1. Décider et implémenter la protection de suppression d'un Exam généré, avec migration/tests si nécessaire.
2. Définir l'immutabilité après génération (au minimum QCM/classe/année/période et nom) et verrouiller SQL/handlers/tests.
3. Prévalider le QCM non vide et sécuriser le cycle de vie des générations failed/retry.
4. Harmoniser les erreurs de suppression classe/année/période.
5. Introduire des view-models Exam typés, sans changer l'UX.
6. Refaire liste puis Create/Edit/Delete et progression.
7. Ajouter la transition QCM → Évaluation et réaliser les smoke tests sur base réelle.

## Verdict

**B — quelques correctifs métier/robustesse avant UX.** Le noyau ownership et le snapshot de chaque copie sont solides, mais la suppression et la mutabilité d'une évaluation générée doivent être décidées avant de présenter une UX qui donnerait une fausse impression de sécurité historique.

## Fichiers inspectés et absence de modification

Ont notamment été inspectés : migrations 0015–0017 et 0020–0031, requêtes `exams*`, `studentExam*`, classes/élèves/années/périodes, structs sqlc, handlers/templates Exams et génération, helpers de construction/recovery/marking, routes serveur et tests de relation, handlers, jobs, QR et correction.

Aucun code, SQL, migration ou template n'a été modifié. Seul ce rapport d'audit a été créé.
