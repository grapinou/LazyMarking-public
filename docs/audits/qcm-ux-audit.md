# Audit UX et fonctionnel du bloc QCM

## Synthèse de décision

Le bloc est sain sur son invariant central : un QCM est une composition pédagogique appartenant à un utilisateur et contenant des références uniques vers des **questions principales**. Les variantes restent internes à leur famille et servent ensuite de formulations alternatives lors de la construction aléatoire des copies. Une évaluation est un objet distinct qui référence un QCM et ajoute le contexte réel (classe, période, année).

Aucun problème P0 de fuite inter-utilisateur ou de corruption immédiate n'a été identifié. Les trois décisions fonctionnelles à traiter avant une refonte profonde sont l'ordre pédagogique, le contrat des aperçus et la suppression d'un QCM référencé.

## 1. Modèle DB QCM

### `qcm`

La migration `0018_create_qcm_table.sql` définit :

| Colonne | Contrat |
|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `name` | `TEXT NOT NULL`, non vide après `trim` |
| `user_id` | `INTEGER NOT NULL`, FK vers `users(id)` |

`UNIQUE(name, user_id)` autorise le même nom chez deux utilisateurs mais pas deux fois chez le même enseignant. Il n'existe ni description, ni statut, ni timestamp. Les lectures et mutations applicatives filtrent par `user_id`.

### `qcm_questions`

La migration `0019_create_qcm_questions_table.sql` définit :

| Colonne | Contrat |
|---|---|
| `id` | `INTEGER PRIMARY KEY AUTOINCREMENT` |
| `qcm_id` | FK obligatoire vers `qcm(id)` |
| `question_id` | FK obligatoire vers `questions(id)`, `ON DELETE RESTRICT` |
| `user_id` | FK obligatoire vers `users(id)` |

`UNIQUE(qcm_id, question_id)` interdit d'ajouter deux fois la même question principale au même QCM. Il n'existe aucune colonne `position`, `order`, date d'ajout ou configuration de variante.

La FK vers `qcm` n'indique pas `ON DELETE` : SQLite applique `NO ACTION`, ce qui bloque la suppression d'un QCM encore composé de questions. La suppression d'une question référencée est explicitement `RESTRICT`.

Les FKs seules ne garantissent pas que les trois lignes appartiennent au même utilisateur. La migration `0030_enforce_user_relation_ownership.sql` ajoute donc des triggers `qcm_questions_owner_insert/update`. Les requêtes `CreateQCMQuestion` et les lectures/mutations relationnelles répètent volontairement ces contrôles en SQL.

### QCM versus Évaluation

La séparation est réelle :

- le QCM porte seulement un nom, un propriétaire et une composition de questions ;
- `exams` référence `qcm_id` et ajoute nom d'évaluation, classe, période et année ;
- les handlers QCM ne créent ni classe, ni date/période, ni évaluation ;
- les formulaires Exams proposent les QCM possédés via `GetAllQCM`.

Le code respecte donc la frontière « modèle/composition » puis « utilisation en situation réelle ».

## 2. Workflow enseignant actuel

| Étape | Route / méthode | Handler | Vue et données | Suite |
|---|---|---|---|---|
| Liste | `GET /dashboard/qcm` | `TableQCMHandler` | `table_qcm.html`; QCM possédés et cinq URLs par ligne | actions par ligne |
| Création | `GET /dashboard/qcm/add` | `AddFormQCMHandler` | `add_form_qcm.html`; routes seulement | pas d'Annuler |
| Enregistrement | `POST /dashboard/qcm/add` | `AddQCMHandler` | nom `qcm` | `303` vers liste |
| Modification | `GET /dashboard/qcm/edit?qcm_id=…` | `EditFormQCMHandler` | nom et ID dans `ExtraData` | pas d'Annuler |
| Enregistrement | `POST /dashboard/qcm/edit` | `EditQCMHandler` | `new_qcm`, `qcm_id` | `303` vers liste |
| Confirmation suppression | `GET /dashboard/qcm/delete?qcm_id=…` | `DeleteFormQCMHandler` | nom et ID | pas d'Annuler |
| Suppression | `POST /dashboard/qcm/delete` | `DeleteQCMHandler` | `qcm_id` | `303` vers liste si non référencé |
| Composition | `GET /dashboard/qcm/qcmquestion?qcm_id=…` | `TableQCMQuestionsHandler` | nom du QCM, questions présentes, URLs de retrait | retour liste / ajout |
| Banque disponible | `GET /dashboard/qcm/qcmquestion/add?qcm_id=…` | `AddFormQCMQuestionHandler` | filtres, familles disponibles, URLs | retour composition |
| Ajout | `POST /dashboard/qcm/qcmquestion/add` | `AddQCMQuestionHandler` | `qcm_id`, plusieurs `question_ids` | transaction puis `303` composition |
| Confirmation retrait | `GET /dashboard/qcm/qcmquestion/delete?...` | `DeleteFormQCMQuestionHandler` | contenu, IDs relation/QCM | pas d'Annuler |
| Retrait | `POST /dashboard/qcm/qcmquestion/delete` | `DeleteQCMQuestionHandler` | `qcm_id`, `qcm_question_id` | `303` composition |
| Aperçu portrait | `GET /dashboard/qcm/previewqcm?qcm_id=…` | `PreviewQCMHandler` | aucune page HTML ; génération puis redirection PDF | PDF dans nouvel onglet |
| Aperçu paysage | `GET /dashboard/qcm/previewqcmlandscape?qcm_id=…` | `PreviewQCMLandscapeHandler` | idem | PDF dans nouvel onglet |
| Évaluation | `GET /dashboard/exams` puis formulaire Exams | handlers Exams | sélection d'un QCM possédé et du contexte scolaire | workflow séparé |

Le chemin métier existe de bout en bout, mais le passage QCM → Évaluations n'est proposé par aucune action contextuelle sur la liste ou la composition. Plusieurs pages n'ont pas d'action Annuler et certains libellés anglais/techniques (`Back to qcm question`) cassent la continuité.

## 3. Conformité au modèle question family

Le modèle famille est correctement appliqué dans le sélecteur :

1. `GetFilteredQuestions` ne lit que les questions principales et filtre sur leurs métadonnées ;
2. `GetAllOwnedAltQuestions` charge les variantes possédées dont le parent est également possédé ;
3. `questionfamilies.Build` groupe les variantes sous leur question principale ;
4. seule `.Main.ID` est portée par une checkbox `question_ids` ;
5. les variantes apparaissent comme information textuelle et n'ont aucun contrôle de sélection ;
6. `qcm_questions.question_id` référence exclusivement `questions`, jamais `alt_questions`.

La génération part ensuite de l'ID principal et `GetRandomQuestionByQuestionID` choisit la formulation principale ou une variante possédée. Aucune ancienne hypothèse permettant d'ajouter directement une `alt_question` n'a été trouvée dans le périmètre QCM.

## 4. Liste des QCM

`table_qcm.html` affiche uniquement le nom de chaque QCM dans un tableau. Chaque ligne expose : modifier, supprimer, « Ajouter Questions » (qui ouvre en réalité toute la composition), aperçu portrait et aperçu paysage.

Constats UX :

- la composition, action métier principale, est placée au même niveau que les opérations CRUD ;
- modifier/supprimer occupent une colonne dédiée et les icônes seules sont peu explicites ;
- les deux aperçus sont deux colonnes distinctes, avec des boutons œil sans texte ;
- aucun nombre de questions ni état « prêt/incomplet » n'est disponible ;
- l'état vide « Pas de qcm… » n'explique pas la prochaine étape ;
- la création existe et est visible, mais le vocabulaire et la hiérarchie sont faibles ;
- aucune action ne mène vers Évaluations.

Direction UX recommandée : conserver Bootstrap, présenter « Mes QCM », faire de « Gérer les questions » l'action principale de chaque QCM, regrouper aperçu/renommage/suppression comme actions secondaires textuelles ou menu simple, et rendre l'état vide pédagogique. Un tableau responsive reste possible ; des cartes compactes sont également cohérentes si le nombre de QCM reste modéré.

## 5. Formulaires CRUD

Le seul vrai champ métier des formulaires création/modification est le nom. `qcm_id` est nécessaire comme champ caché pour modifier/supprimer mais n'est pas affiché visuellement. La valeur courante est préremplie à la modification.

Les trois pages sont isolées, sans carte de contexte ni Annuler. Les formulaires création et modification ajoutent un JavaScript local qui retire seulement les guillemets, alors que le vrai contrat serveur repose sur `TrimSpace`, le `CHECK` DB et l'unicité. Ce script n'est pas une contrainte métier nécessaire.

La suppression emploie un vocabulaire informel (« Es-tu sur », « C'est mon dernier mot »), ne précise pas les dépendances, et ne permet pas de revenir à la liste. Elle échoue en 500 si le QCM contient encore des questions ou est utilisé par une évaluation, car les FKs bloquent le DELETE.

Direction : formulaires courts avec titre/description, label associé, nom courant, Annuler vers la liste et actions responsive. La confirmation doit expliquer que la suppression n'est possible qu'après retrait des questions et qu'un QCM utilisé par une évaluation est protégé, sans inventer de cascade.

## 6. Gestion des questions

### Composition actuelle

`table_qcmquestion.html` affiche le nom du QCM puis un tableau des contenus sélectionnés. Chaque ligne permet seulement le retrait. La page charge correctement et explicitement le QCM possédé avant la relation.

La distinction métier est présente dans les routes, mais le titre « Ajouter des questions » décrit mal cette page : il s'agit d'abord des **Questions du QCM**. L'état vide et le bouton « Ajouter Question » existent, sans contexte pédagogique ni aperçu depuis cette page.

### Banque disponible

`add_form_qcm_question.html` exclut les IDs déjà présents avant de construire les familles. Une question déjà sélectionnée n'apparaît donc plus. Plusieurs familles peuvent être cochées et ajoutées dans une transaction unique. Un échec sur une relation annule tout le lot.

La banque et la composition sont deux pages distinctes, mais le sélecteur ne rappelle pas le nom du QCM : seul l'ID caché conserve le contexte. La hiérarchie conceptuelle recommandée est : contexte `QCM : <nom>` → « Questions sélectionnées » → action « Ajouter des questions » → banque disponible.

Le formulaire de retrait confirme le contenu de la question mais pas le nom du QCM, n'a pas Annuler, et utilise le même vocabulaire informel que les suppressions.

## 7. Filtres et sélecteur

Six filtres GET sont disponibles : sujet, thème, niveau, compétence, difficulté et points. Les valeurs sont lues par `GetFieldFiltered`, transmises sous forme nullable à `GetFilteredQuestions`, puis conservées dans les listes via les champs `Selected…ID`. « Réinitialiser » conserve seulement `qcm_id`.

Le POST distinct envoie `qcm_id` et zéro à plusieurs `question_ids`. La sélection est multiple par checkboxes. Les filtres portent bien sur les métadonnées de la question principale ; toutes les variantes possédées de la famille restent ensuite visibles, même si elles n'ont pas de métadonnées propres.

Il n'y a pas de N+1 par famille : la page effectue des lectures globales (QCM, six référentiels, questions filtrées, IDs déjà présents, variantes possédées). Le coût est un nombre constant de requêtes, même si les six référentiels ne sont pas dépendants les uns des autres.

Quand aucun résultat ne correspond, le tableau affiche un message correct. Une question déjà dans le QCM est exclue. En revanche, envoyer zéro sélection ouvre et valide une transaction vide puis redirige comme un succès. Avec un `qcm_id` étranger et zéro sélection, le POST ne vérifie jamais le parent ; aucune donnée n'est exposée ou mutée, mais le contrat 404 ownership-aware est contourné. C'est une asymétrie fonctionnelle, pas une fuite de sécurité.

Classification :

- métier correct : familles, exclusion des doublons visibles, transaction multi-sélection ;
- données de vue : absence du nom/contexte QCM typé dans le sélecteur ;
- UX : grand tableau peu mobile, aucune action quand rien n'est coché, filtres denses ;
- asymétrie handler : POST vide accepté et parent non prévalidé.

## 8. Ordre des questions

Il n'existe aucun ordre explicite en DB. `GetAllQuestionsByQCMID` n'a même pas de `ORDER BY`, donc l'ordre de la page de composition n'est pas contractuel.

Pour la construction d'un QCM, `GetQCMQuestionsIDs` trie d'abord par `question_id`, puis `GetQCMQuestionsAnswers*` mélange volontairement les IDs avec `ShuffleSlice`. Les workers concurrents ajoutent ensuite les résultats dans leur ordre d'achèvement. L'aperçu et la génération ne préservent donc ni l'ordre d'insertion ni un ordre pédagogique ; leur ordre est volontairement variable et peut différer entre deux exécutions.

Avant toute migration, il faut décider le besoin :

- si l'individualisation doit aléatoirement réordonner les questions, l'absence d'ordre de sortie est cohérente, mais l'enseignant ne maîtrise pas de structure de référence ;
- si l'enseignant doit définir une progression, une colonne `position` devient un vrai besoin métier, avec une décision séparée sur le mélange lors de la génération.

Ce point doit être tranché avant une UX de réordonnancement.

## 9. Aperçus QCM

Deux orientations existent seulement : portrait (`TypstWriter`) et paysage (`TypstWriterLandscape`). Elles sont déclenchées depuis la liste, dans un nouvel onglet. « Aperçu » signifie implicitement portrait ; seule la seconde orientation est nommée.

Le `qcm_id` est parsé, mais le handler ne charge pas explicitement le QCM. `GetQCMQuestionsIDs` protège les lectures par `user_id` et parent possédé, donc un QCM étranger n'expose aucune question. Cependant un QCM absent/étranger est indistinguable d'un QCM possédé vide : une liste vide est construite et un PDF vide peut être généré au lieu d'un 404. C'est une asymétrie fonctionnelle importante, non une fuite.

Un QCM possédé vide est actuellement accepté. Une famille dont aucune formulation ne possède de réponse conduit à `ErrQuestionWithNoAnswer`, puis à une redirection vers la page d'erreur. Les autres erreurs de construction et les erreurs Typst donnent 500.

Chaque aperçu :

1. purge au mieux les workspaces de prévisualisation expirés de l'utilisateur ;
2. crée un workspace `preview-<UUID>` isolé ;
3. écrit et compile Typst ;
4. supprime le workspace par `defer` sur erreur ;
5. conserve le workspace réussi pour le service PDF, puis le nettoyage différé (rétention une heure).

Le service PDF valide username/opération/nom, refuse les liens symboliques et sert seulement un fichier régulier du workspace. Aucun template ni `QCMPreviewPageData` n'existe : `qcmPreview.go` définit seulement les deux routes de service PDF.

Direction UX : actions textuelles « Aperçu portrait » / « Aperçu paysage », accessibles depuis la composition et éventuellement regroupées sur la liste ; message explicite si le QCM est vide ou incomplet ; 404 pour parent absent/étranger.

## 10. PageData et `ExtraData`

### Structures actuelles

- `QCMPageData` : `Routes`, `QCMRoutes`, `PageTitle`, `ExtraData` ;
- `QCMQuestionPageData` : `Routes`, `QCMQuestionRoutes`, `PageTitle`, `ExtraData` ;
- aperçu : aucune PageData, seulement `PreviewQCMRoutes`.

Toutes les données métier restent dans `map[string]any` : listes QCM, booléens vides, tableaux parallèles d'URLs, nom QCM, ID QCM tantôt `string` tantôt `int64`, familles, référentiels et sélections. Les templates dépendent de l'alignement par index entre lignes et URLs. Plusieurs URLs sont construites dans les handlers, mais le lien Réinitialiser est reconstruit dans le template.

Un petit contexte naturel est justifié :

```go
type QCMContext struct {
    ID   int64
    Name string
}
```

Il pourrait être partagé par composition, sélection, retrait et aperçu. Il réduirait les IDs convertis en chaînes, rendrait le parent explicite et éviterait d'inventer un `FamilyContext` générique. Les collections et états de page gagneraient également à devenir typés progressivement, sans refonte globale en un jalon.

## 11. Ownership et intégrité

| Chemin | État | Conclusion |
|---|---|---|
| Liste QCM | `GetAllQCM(user_id)` | correct |
| GET modifier/supprimer | lookup ID + user, `HandleOwnedLookupError` | correct, 404 absent/étranger |
| POST modifier/supprimer | `WHERE id AND user_id`, contrôle rows | correct, 404 sans mutation |
| GET composition/sélecteur/retrait | lookup parent possédé préalable | correct |
| Ajout relation | QCM + question + user dans `INSERT SELECT`, trigger DB, transaction | correct |
| Sélection mixte possédée/étrangère | rows=0 puis rollback intégral | correct |
| Ajout doublon | contrainte unique, rollback intégral, redirection erreur | correct mais message générique |
| POST ajout sans sélection | aucune mutation, parent non vérifié | asymétrie fonctionnelle |
| GET retrait | relation liée au bon QCM et au bon user | correct |
| POST retrait forgé/mauvais parent | triple contrainte relation/QCM/user + rows | correct, 404 |
| Suppression QCM référencé | FKs empêchent la suppression | intégrité correcte, UX/HTTP 500 insuffisants |
| Preview absent/étranger | lecture vide plutôt que lookup 404 | bug fonctionnel potentiel, pas de fuite |
| Ownership DB transversal | triggers `0030` | correct, défense contre lignes inter-utilisateurs |

Le code moderne `HandleOwnedLookupError` / `HandleOwnedMutationRows` est bien utilisé sur les parcours CRUD et relationnels principaux.

## 12. Couverture de tests

| Scénario | Couverture actuelle |
|---|---|
| création QCM | non couvert au niveau handler |
| QCM étranger | couvert pour formulaires/relations ; liste implicitement SQL |
| modification étrangère | couvert (handler + rows SQL) |
| suppression étrangère | couvert (handler + rows SQL) |
| ajout question succès | non couvert au niveau handler |
| question étrangère | couvert, sélection mixte rollback + SQL |
| mauvais parent QCM/relation | couvert pour retrait et lectures |
| doublon | couvert au niveau requête/contrainte, pas handler |
| suppression relation succès | couvert au niveau requête, pas handler |
| famille de question | couvert : famille possédée, variantes, famille vide, exclusion déjà sélectionnée |
| sélection filtre | couvert pour les six dimensions et métadonnées principales |
| QCM vide | non couvert |
| preview portrait | non couvert au niveau handler |
| preview paysage | non couvert au niveau handler |
| QCM absent/étranger en preview | non couvert |
| suppression QCM référencé | couvert au niveau intégrité DB |

Tests directement inventoriés : un test handler QCM tabulaire ; cinq tests handler/familles dans `qcmQuestions`; quatre tests d'intégrité dans `qcmRelationshipIntegrity_test.go`; le test connexe `TestFilteredQuestionReadsUseOnlyMainQuestionMetadata`. Il n'existe aucun test dans `qcmPreview`.

Il manque surtout des tests fonctionnels de succès et de preview. Aucun test HTML Bootstrap n'est nécessaire.

## 13. Relation avec Exams

`exams.qcm_id` est obligatoire et possède une FK `ON DELETE RESTRICT`. Les créations/mises à jour Exams vérifient en SQL que QCM, classe, période et année appartiennent au même utilisateur ; les triggers de migration `0030` répètent cette garantie. Un QCM utilisé ne peut donc pas être supprimé.

La DB empêche correctement un examen de référencer un QCM inexistant ou étranger. En revanche, l'interface QCM ignore si le modèle est utilisé : la tentative de suppression finit en erreur 500. La liste n'a pas nécessairement besoin d'un badge d'usage dès le premier jalon, mais la confirmation et le traitement de suppression doivent connaître cette restriction ou traduire proprement l'échec.

Le passage vers Évaluations est uniquement assuré par la navigation globale. Une action « Utiliser dans une évaluation » peut être étudiée plus tard, sans fusionner les deux modèles.

## 14. Priorités

### P0 — 0

Aucun bug bloquant d'intégrité ou de sécurité n'a été identifié dans le périmètre. Les ownership checks applicatifs, SQL et triggers se complètent correctement.

### P1 — 3

1. **Décider et maîtriser l'ordre des questions.** Aucun ordre pédagogique n'est stocké ; composition non contractuelle et génération mélangée/concurrente.
2. **Rendre le preview ownership-aware et explicite pour un QCM vide/incomplet.** Absent et étranger produisent actuellement le même ensemble vide qu'un QCM possédé vide.
3. **Traiter fonctionnellement la suppression protégée.** Les questions et examens empêchent correctement le DELETE, mais le handler présente une erreur 500 générique et l'UI n'explique aucune dépendance.

### P2 — 7

1. Introduire progressivement un `QCMContext` typé et réduire `ExtraData`/les tableaux parallèles d'URLs.
2. Rehiérarchiser la liste autour de « Gérer les questions », avec actions textuelles et état vide pédagogique.
3. Harmoniser les formulaires nom/modification/suppression et leurs actions Annuler.
4. Recentrer la page de composition sur « Questions du QCM » et rappeler le contexte parent partout.
5. Améliorer le sélecteur responsive et refuser clairement une soumission vide ; prévalider le parent au POST.
6. Clarifier portrait/paysage et rapprocher l'aperçu de la composition.
7. Compléter les tests de succès CRUD/relation et les tests preview, sans tests de classes Bootstrap.

## 15. Plan de refonte incrémental proposé

1. **Contrat de contexte QCM** : introduire `QCMContext`, le fournir à composition/sélection/retrait, sans changement métier.
2. **Liste « Mes QCM »** : hiérarchie des actions, état vide, accès composition et aperçus.
3. **Formulaires QCM** : création/renommage/confirmation, Annuler et terminologie professionnelle.
4. **Composition « Questions du QCM »** : contexte, sélection actuelle, action Ajouter, état vide.
5. **Sélecteur/filtres** : même modèle famille, contexte conservé, responsive, soumission vide et ownership POST.
6. **Retrait du QCM** : confirmation contextualisée et Annuler vers la composition.
7. **Preview** : lookup QCM explicite, cas vide/incomplet, libellés portrait/paysage et tests ciblés.
8. **Décision ordre** : atelier métier court, puis migration/commandes de réordonnancement seulement si l'ordre de référence est requis.
9. **Suppression et usage Exams** : traduire les restrictions proprement, puis étudier le passage « Utiliser dans une évaluation ».

Le premier jalon recommandé est le contexte typé, car il donne aux pages suivantes un parent stable sans modifier SQL ni comportement.

## 16. Fichiers inspectés

Périmètre direct :

- `internal/handlers/qcm/{handlers.go,handlers_test.go,routes.go,views.go}` ;
- `internal/handlers/qcmQuestions/{handlers.go,handlers_test.go,routes.go,views.go}` ;
- `internal/handlers/qcmPreview/{handlers.go,routes.go}` ;
- tous les templates `internal/templates/qcm/*.html` et `internal/templates/qcmquestions/*.html` ;
- `internal/templates/data/{qcm.go,qcmQuestions.go,qcmPreview.go,dashboard.go}` ;
- `db/query/{qcm.sql,qcmquestion.sql}` ;
- `db/migrations/{0018_create_qcm_table.sql,0019_create_qcm_questions_table.sql}` ;
- `internal/db/qcmRelationshipIntegrity_test.go` et fichiers sqlc générés correspondants.

Dépendances directement utiles :

- `internal/questionfamilies/families.go` ;
- `db/query/{questions.sql,altQuestions.sql}` et `internal/db/questionGraphIntegrity_test.go` ;
- `internal/handlers/tools/{getQCMQuestionsAnswers.go,getQCMQuestionsAnswersCtx.go,buildQuestion.go,buildQuestionCtx.go,typstWriter.go,typstWriterLandscape.go,ephemeralWorkspaces.go,createUserTmpFile.go,servePdf.go,servePdfNamed.go}` ;
- routes d'enregistrement dans `cmd/server/main.go` ;
- frontière Exams : `db/query/exams.sql`, migrations `0022` et `0030`, handlers/tests/formulaires Exams nécessaires, navigation dashboard.

## 17. État du dépôt et validation

Aucun fichier Go, template, SQL, migration, fichier sqlc, test ou configuration n'a été modifié. Le seul fichier créé pour ce jalon est ce rapport d'audit. Aucun commit n'a été créé.

`git diff --check` : réussi.
