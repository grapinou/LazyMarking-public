# Audit de l'ordre des questions d'un QCM

## Synthèse

LazyMarking doit stocker un ordre pédagogique de référence indépendant de l'individualisation. La DB peut fournir une séquence stable, le preview peut la montrer telle quelle, tandis que la génération réelle continue à mélanger une copie de cette séquence avant construction.

Aujourd'hui, deux mécanismes modifient l'ordre : `ShuffleSlice` le mélange volontairement, puis les workers le réordonnent involontairement selon leur temps d'achèvement. Le choix main/variante est une randomisation distincte, effectuée plus tard dans `BuildQuestion*`.

## 1. Pipeline actuel complet

### Composition affichée

```text
qcm_questions
  → GetAllQuestionsByQCMID
  → JOIN questions
  → aucun ORDER BY
  → slice sqlc dans l'ordre livré par SQLite
  → table de composition
```

La table n'a ni `position` ni ordre contractuel. L'ordre observé peut sembler stable sur une base donnée, mais SQL ne le garantit pas.

### Preview et génération

```text
qcm_questions
  → GetQCMQuestionsIDs
  → ORDER BY question_id
  → []question_id déterministe croissant
  → ShuffleSlice en place
  → jobs envoyés dans cet ordre
  → 5 workers construisent les questions
  → results channel dans l'ordre d'achèvement
  → append dans qcmQuestions
  → Typst itère le slice final
```

Les étapes exactes sont :

1. `GetQCMQuestionsIDs` sélectionne seulement les relations dont QCM, relation et utilisateur sont cohérents ;
2. SQL trie par `question_id`, pas par insertion ;
3. `GetQCMQuestionsAnswers` et sa version `Ctx` appellent toujours `ShuffleSlice` ;
4. les IDs mélangés sont placés dans un channel `jobs` tamponné ;
5. cinq goroutines lisent ce channel ;
6. chaque worker appelle `BuildQuestion` ou `BuildQuestionCtx` ;
7. le worker envoie seulement `config.Question`, sans index, dans `results` ;
8. après attente de tous les workers, le consommateur fait `append` dans l'ordre du channel ;
9. `TypstWriter`, `TypstWriterLandscape` et `TypstLandscapeContent` parcourent ce slice dans cet ordre.

L'ordre SQL initial est donc déterministe, mais il n'est déjà plus visible après le shuffle. Même l'ordre mélangé des jobs n'est pas préservé par les résultats concurrents.

## 2. Endroits où l'ordre change

| Notion | État actuel |
|---|---|
| ordre de sélection DB pour la composition | non défini (`GetAllQuestionsByQCMID` sans `ORDER BY`) |
| ordre de sélection DB pour construction | `question_id` croissant |
| ordre de référence | inexistant |
| ordre de génération demandé | mélange explicite par `ShuffleSlice` |
| ordre final réel | ordre d'achèvement des workers après mélange |
| ordre Typst | ordre du slice final |

L'ordre d'insertion n'est utilisé explicitement nulle part. `qcm_questions.id` existe néanmoins et constitue un proxy historique stable pour un futur backfill.

## 3. Effet de la concurrence

La concurrence affecte actuellement l'ordre. Les résultats ne transportent pas l'index du job. Une question lente peut donc apparaître après une question envoyée plus tard, indépendamment de la permutation créée par `ShuffleSlice`.

Pour conserver demain un ordre demandé sans renoncer aux workers :

```text
job{index, questionID}
  → worker
  → result{index, question, err}
  → questions[index] = question
```

Le slice de résultats est préalloué à la taille des IDs. Le mode « référence » fournit les IDs triés par position ; le mode « copie mélangée » mélange d'abord une copie des IDs. Dans les deux cas, les workers conservent exactement l'ordre choisi en amont. Cette correction sépare déterminisme et parallélisme ; elle ne supprime aucune individualisation.

## 4. Ordre aléatoire et variante aléatoire

Les deux randomisations sont déjà situées à des niveaux distincts :

- **ordre des familles** : `ShuffleSlice(questionsIDs)` dans `GetQCMQuestionsAnswers*` ;
- **formulation d'une famille** : `GetRandomQuestionByQuestionID`, appelé dans `BuildQuestion*`, choisit avec `ORDER BY RANDOM()` la question principale ou une `alt_question` possédée.

Le retry de `BuildQuestion*` cherche jusqu'à 100 fois une formulation qui possède au moins une réponse. Ensuite les réponses elles-mêmes sont également mélangées par `GetQuestionAnswer*` ou `GetAltQuestionAltAnswer*`. Il existe donc trois axes indépendants : ordre des familles, choix de formulation, ordre des réponses.

L'architecture permet de conserver les deux derniers tout en affichant les familles selon leur position de référence.

## 5. Preview actuel et recommandation

Les previews portrait et paysage appellent actuellement le même `GetQCMQuestionsAnswers` que la génération/mini-génération. Ils mélangent donc les IDs, choisissent une formulation aléatoire et récupèrent les questions selon l'achèvement des workers.

Recommandation métier : **un aperçu QCM doit montrer l'ordre pédagogique de référence**. Cela correspond à la composition que l'enseignant vient d'organiser. Le choix main/variante peut faire l'objet d'une décision UX ultérieure ; le point de ce chantier est que l'ordre des familles ne doit pas être mélangé dans cet aperçu.

Un futur « aperçu d'une copie individualisée » pourrait explicitement réutiliser le mode mélangé. Les deux orientations portrait/paysage devraient partager le même ordre de référence.

## 6. Impact sur la génération réelle

Deux parcours consomment la construction aléatoire :

- `BuildQcmStudentCtx`, utilisé pour les copies réelles d'une évaluation, appelle `GetQCMQuestionsAnswersCtx` pour chaque élève ;
- `GenerateMiniPDFHandler` appelle `GetQCMQuestionsAnswers` pour chaque élève de la mini-génération.

Chaque élève obtient donc aujourd'hui sa permutation, ses formulations et ses réponses. La recommandation est de conserver ce mélange réel après introduction de `position`.

La génération sérialise ensuite le `config.QCM` final dans `student_exam_content`. Avant sérialisation, les cercles détectés sur le PDF sont associés aux questions et réponses dans le même ordre que celui utilisé par Typst. La correction relit ce JSON propre à la copie, valide les nombres, calcule points/compétences et aligne les cercles dans cet ordre.

Conséquence importante : modifier l'ordre de référence après génération ne réinterprète pas une copie existante. En revanche, il ne faut jamais retrier le slice d'une copie entre rendu Typst, association des cercles et sérialisation.

## 7. Schéma `position` recommandé

Contrat futur recommandé :

```text
position INTEGER NOT NULL CHECK (position >= 1)
UNIQUE (qcm_id, position)
```

- positions à partir de **1**, naturelles pour l'enseignant ;
- `user_id` est inutile dans la contrainte d'unicité : `qcm_id` identifie déjà un QCM globalement et les triggers garantissent l'ownership ;
- conserver `UNIQUE(qcm_id, question_id)` pour empêcher les doublons pédagogiques ;
- utiliser uniquement des entiers, sans fractional ordering ;
- maintenir des positions compactes `1..n` par les mutations applicatives transactionnelles.

SQLite ne peut pas exprimer directement l'absence de trous par une simple contrainte. Cet invariant doit être maintenu et testé par les requêtes/transactions, tandis que `NOT NULL`, `CHECK` et `UNIQUE` protègent les erreurs les plus dangereuses.

## 8. Backfill recommandé

Attribuer les positions par QCM selon :

```text
ORDER BY qcm_questions.id
```

`id` représente le meilleur proxy disponible pour la chronologie d'ajout. `question_id` est seulement l'ordre de lecture actuel de la génération et peut différer de l'ordre d'insertion. Aucun backfill ne peut reconstruire un ordre pédagogique historique qui n'a jamais été stocké.

Le backfill doit produire `ROW_NUMBER() OVER (PARTITION BY qcm_id ORDER BY id)` ou un équivalent déterministe compatible avec la version SQLite retenue, puis poser les contraintes dans une reconstruction de table testée. En cas d'anciens IDs atypiques, le résultat reste stable et explicable.

## 9. Stratégie d'ajout

### Une question

Dans la même transaction, attribuer `COALESCE(MAX(position), 0) + 1` pour le QCM possédé. La première question reçoit 1.

### Plusieurs questions

Le navigateur envoie normalement les checkboxes dans l'ordre du document, et le sélecteur actuel affiche les principales par `q.id`. Ce comportement HTML ne doit toutefois pas devenir le seul contrat métier implicite.

Tant qu'aucune UI de classement n'accompagne l'ajout multiple, trier explicitement les IDs sélectionnés par `question_id` croissant, puis attribuer les positions successives à partir de `max(position)+1`. Cette règle est simple, déterministe et correspond à l'ordre visible actuel du sélecteur. Un réordonnancement explicite peut suivre immédiatement après.

L'ensemble de l'ajout multiple doit rester transactionnel. Deux ajouts concurrents seront arbitrés par SQLite et la contrainte unique ; une erreur ne doit pas produire un lot partiel.

## 10. Stratégie de suppression

Recommandation : **compactage**.

Après retrait de la position `p`, les positions supérieures descendent d'une unité dans la même transaction. L'enseignant voit toujours une séquence 1, 2, 3 et les actions Monter/Descendre restent faciles à raisonner.

Avec `UNIQUE(qcm_id, position)`, un `UPDATE position = position - 1` peut rencontrer une collision temporaire selon l'ordre interne de SQLite. Une stratégie sûre en deux phases est nécessaire : déplacer d'abord les positions concernées dans une plage temporaire libre au-dessus du maximum, puis les ramener compactées. Toute erreur rollbacke DELETE et compactage.

Laisser des trous simplifierait le DELETE mais reporterait la normalisation sur l'affichage, les mouvements et les futurs inserts. Pour une petite application SQLite, une transaction de compactage est plus claire à maintenir.

## 11. Réordonnancement futur

Pour « Monter » / « Descendre », un échange adjacent transactionnel suffit :

1. vérifier QCM + relation + user et trouver la ligne voisine ;
2. déplacer la ligne courante vers une position temporaire libre, par exemple `MAX(position)+1` ;
3. déplacer la voisine vers l'ancienne position ;
4. déplacer la ligne temporaire vers la position de la voisine ;
5. commit.

Cette séquence ne viole jamais `UNIQUE(qcm_id, position)` et fonctionne avec `CHECK(position >= 1)`. En haut, « Monter » affecte zéro ligne fonctionnelle ; en bas, « Descendre » également. Le handler devra traduire parent/relation étrangère en 404 et une absence de voisin en action impossible ou no-op explicite.

Pour un déplacement arbitraire futur (drag-and-drop), la même idée de plage temporaire puis compactage peut être étendue, sans float ni lexorank.

## 12. Requêtes SQL existantes concernées

Quatre requêtes existantes doivent évoluer :

| Requête | Actuel | Futur souhaité |
|---|---|---|
| `GetAllQuestionsByQCMID` | aucun `ORDER BY` | retourner `position`, trier `ORDER BY qcm_questions.position` |
| `GetQCMQuestionsIDs` | `ORDER BY question_id` | trier `ORDER BY position` pour fournir l'ordre de référence |
| `CreateQCMQuestion` | insert sans position | insérer la prochaine position sous ownership, dans une transaction |
| `DeleteQCMQuestion` | DELETE isolé | fournir/obtenir la position supprimée et compacter transactionnellement |

Des requêtes minimales nouvelles seront probablement nécessaires pour `MAX(position)`, la lecture de la relation/position et les échanges Monter/Descendre. Il est préférable de ne pas surcharger artificiellement une seule requête sqlc. `GetQuestionContentByQCMQuestionID` n'a pas besoin de changer pour la sémantique d'ordre.

Le sélecteur utilise `GetQCMQuestionsIDs` uniquement comme ensemble d'appartenance ; son résultat ordonné par position reste compatible.

## 13. Risques de régression

- **Preview/Typst** : enlever le shuffle ne suffit pas ; sans résultats indexés, les workers détruisent encore l'ordre de référence.
- **Génération réelle** : elle doit continuer à mélanger avant les workers, puis préserver cette permutation précise.
- **Correction** : le JSON par copie et l'ordre des cercles protègent actuellement l'alignement. Tout retri après rendu casserait questions, réponses, points et annotations.
- **Numérotation** : Typst itère le slice ; la position visuelle dépend donc directement du pipeline.
- **Variantes** : `MainQuestionID` conserve l'identité de famille même lorsqu'une variante est choisie ; `position` appartient à la relation avec la principale, jamais à la variante.
- **Réponses** : leur shuffle est indépendant mais leur ordre final doit rester stable entre rendu, cercles et JSON.
- **Tableaux de résultats/compétences** : ils parcourent le QCM sérialisé ; l'ordre n'altère pas les totaux, mais une désynchronisation des slices les altérerait.
- **Concurrence d'ajout/réordonnancement** : toutes les mutations de position doivent être transactionnelles et protégées par l'unicité.
- **Migrations historiques** : une reconstruction doit conserver IDs de relation, ownership et les deux contraintes uniques/FKs/triggers effectifs.

Aucun test actuel ne verrouille un ordre de sortie de `GetQCMQuestionsAnswers*`. Les tests d'intégrité QCM vérifient ownership et restrictions, pas les positions.

## 14. Tests minimaux du futur jalon

### Migration/DB

1. backfill par `qcm_questions.id`, partitionné par QCM ;
2. positions 1..n déterministes et indépendantes entre deux QCM ;
3. première insertion à 1 ;
4. insertion après existantes à max+1 ;
5. ajout multiple déterministe et atomique ;
6. rejet de deux positions identiques dans un QCM ;
7. même position autorisée dans deux QCM ;
8. question/QCM/user étrangers toujours refusés ;
9. suppression puis compactage atomique ;
10. Monter/Descendre, bornes haut/bas et mauvais parent.

### Lecture/pipeline

11. composition triée par position ;
12. lecture IDs triée par position ;
13. construction concurrente en mode référence conservant l'index malgré des durées inversées ;
14. construction en mode mélangé conservant exactement la permutation fournie ;
15. preview portrait et paysage en ordre de référence ;
16. génération réelle toujours individualisée/mélangée ;
17. choix main/variante toujours indépendant ;
18. sérialisation/correction conservant l'association question-réponses-cercles.

Les tests de concurrence doivent injecter une petite fonction de construction contrôlée ou utiliser des barrières/channels, sans `sleep` fragile.

## 15. Décision finale

| Décision | Recommandation |
|---|---|
| A — stocker un ordre pédagogique | **oui** |
| B — type | `position INTEGER NOT NULL CHECK(position >= 1)` |
| C — unicité | `UNIQUE(qcm_id, position)` en plus de `UNIQUE(qcm_id, question_id)` |
| D — backfill | position 1..n par QCM, `ORDER BY qcm_questions.id` |
| E — suppression | compactage transactionnel |
| F — preview | ordre de référence |
| G — génération réelle | conserver le mélange actuel des familles, mais préserver ensuite la permutation choisie malgré les workers |

## 16. Plan incrémental

1. **Migration + SQL + tests DB** : colonne/contraintes, backfill, lectures ordonnées, ajout en fin et suppression compacte ; aucun changement UX.
2. **Primitive de construction ordonnée** : jobs/résultats indexés et choix explicite interne « préserver » ou « mélanger », avec tests concurrents ; génération réelle reste en mode mélangé.
3. **Composition ordonnée** : afficher la séquence de référence et sa position, sans action de déplacement.
4. **Actions Monter/Descendre** : requêtes transactionnelles ownership-aware et tests de bornes.
5. **Preview de référence** : portrait/paysage utilisent le mode sans mélange des familles ; variantes/réponses restent une décision séparée.
6. **UX complète** : libellés, actions responsive et éventuel drag-and-drop dans un chantier ultérieur.

Le premier jalon recommandé est **migration + requêtes SQL + tests DB**, séparé du pipeline concurrent et de l'interface.

## 17. Fichiers inspectés et absence de modification

Fichiers directement inspectés :

- `db/migrations/0019_create_qcm_questions_table.sql` ;
- `db/query/qcmquestion.sql` et `internal/db/qcmquestion.sql.go` ;
- `internal/handlers/qcmQuestions/*` ;
- `internal/handlers/qcmPreview/*` ;
- `internal/handlers/tools/getQCMQuestionsAnswers.go` et `getQCMQuestionsAnswersCtx.go` ;
- `internal/handlers/tools/buildQuestion.go` et `buildQuestionCtx.go` ;
- helpers de construction principale/variante/réponses et `shuffleSlice.go` ;
- `internal/handlers/tools/buildQcmStudentCtx.go` ;
- consommation ciblée dans `internal/handlers/generateExams/handlers.go` ;
- sérialisation `student_exam_content`, validation et consommation ciblée par la correction ;
- tests QCM et tests connexes portant sur ces invariants.

Aucun fichier Go, SQL, sqlc, migration, template ou test n'a été modifié. Le seul fichier créé est ce rapport. Aucun commit n'a été réalisé.
