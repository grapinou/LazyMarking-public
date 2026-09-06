# Audit de clôture du bloc QCM

## 1. Résumé exécutif

Le code actuel confirme que les fondations métier, d'intégrité et de navigation du bloc QCM sont désormais saines. Un QCM est une composition possédée de familles de questions principales, dotée d'un ordre pédagogique explicite. Le Preview suit cet ordre de référence, tandis que les copies réelles conservent les trois niveaux d'individualisation : ordre des familles, choix de la formulation principale ou variante, et ordre des réponses.

La suppression possède maintenant le contrat attendu : la composition est dépendante du QCM, les questions de banque sont conservées, et un examen protège le QCM qu'il utilise. Les pages principales ont quitté la présentation CRUD technique au profit d'un parcours enseignant cohérent.

Sur douze constats concrets retenus de l'audit initial, neuf sont résolus. Deux petits correctifs P2 restent utiles avant de fermer complètement le chantier : valider explicitement le parent lors d'un POST d'ajout sans sélection, et harmoniser la confirmation de retrait d'une question. L'absence d'action contextuelle vers Évaluations reste un P3 non bloquant, naturellement rattachable au prochain chantier Exams.

Verdict : **B — bloc QCM terminé après deux petits correctifs**. Aucun P0 ni P1 ne subsiste.

## 2. Constats de l'audit initial

Le dénominateur ci-dessous reprend douze problèmes ou dettes concrètes de `qcm-ux-audit.md`. Les constats qui étaient déjà positifs dans l'audit initial, comme la séparation QCM/Évaluation et le modèle family, ne sont pas artificiellement comptés comme des problèmes résolus.

| # | Constat initial | Statut actuel | Vérification dans le code actuel |
|---:|---|---|---|
| 1 | Parent QCM implicite, IDs convertis en chaînes, données parentales dans `ExtraData` | **RÉSOLU** | `QCMContext{ID int64, Name string}` est fourni à la composition, au sélecteur, au retrait et aux formulaires edit/delete. |
| 2 | Liste « Mes QCM » trop CRUD, table, actions icon-only, aucun compteur ni état vide pédagogique | **RÉSOLU** | `QCMListItem`, comptage agrégé, cartes responsive, action principale « Gérer les questions », actions textuelles et état vide. |
| 3 | Formulaires QCM isolés, sans Annuler, vocabulaire informel et filtrage JavaScript des guillemets | **RÉSOLU** | Les trois formulaires sont harmonisés, Annuler revient à « Mes QCM », le JS a disparu et les noms avec apostrophes/guillemets restent acceptés côté serveur. |
| 4 | Composition technique intitulée comme un ajout, tableau sans ordre ni Preview | **RÉSOLU** | Page « Questions du QCM », `QCMQuestionItem`, position DB visible, cartes, Monter/Descendre, Retirer, états vides et aperçus. |
| 5 | Sélecteur sans nom du parent, table dense, filtres et variantes peu lisibles | **RÉSOLU** | `QCMContext`, `QCMQuestionSelectorData`, cartes de familles, filtres GET, reset, variantes enfants informatives et états vides distincts. |
| 6 | POST d'ajout sans `question_ids` : transaction vide et parent non validé | **TOUJOURS PRÉSENT** | `AddQCMQuestionHandler` trie une slice vide, ouvre/commit une transaction vide et redirige sans lookup ownership-aware du QCM. |
| 7 | Aucun ordre pédagogique stocké ; lectures non contractuelles ; workers réordonnant par achèvement | **RÉSOLU** | `position`, lectures ordonnées, mutations compactes et worker pool indexé préservant l'ordre d'entrée. |
| 8 | Preview absent/étranger confondu avec QCM vide ; ordre aléatoire ; portrait/paysage peu explicites | **RÉSOLU** | Lookup parent explicite, 404 ownership-aware, QCM vide redirigé vers une erreur métier, ordre de référence et actions portrait/paysage textuelles. |
| 9 | Suppression QCM composée/protégée aboutissant à une erreur générique | **RÉSOLU** | CASCADE limité à `qcm_questions`, RESTRICT côté Exams, prévalidation métier et traduction robuste d'une FK concurrente. |
| 10 | Couverture insuffisante des succès, relations, migrations, concurrence et Preview | **RÉSOLU** | Les suites actuelles couvrent CRUD, ownership, ordre, migrations Up/Down, ajout/retrait/move, rollback, workers, Preview, liste, composition et sélecteur. |
| 11 | Confirmation de retrait informelle, sans Annuler et encore alimentée par un petit `ExtraData` | **TOUJOURS PRÉSENT** | `delete_form_qcm_question.html` contient encore « Es-tu sur » et « C'est mon dernier mot », sans retour vers la composition. |
| 12 | Aucun passage contextuel QCM vers Évaluations | **TOUJOURS PRÉSENT** | La navigation globale expose Évaluations et ses formulaires chargent les QCM possédés, mais liste/composition n'offrent pas d'action contextuelle. |

Bilan : **9/12 constats initiaux résolus**, 3 encore présents, dont un seul est facultatif pour la clôture fonctionnelle.

## 3. Ordre pédagogique

Le contrat final est complet :

- `qcm_questions.position` est `INTEGER NOT NULL CHECK(position >= 1)` ;
- `UNIQUE(qcm_id, position)` interdit les doublons de position dans un QCM ;
- `UNIQUE(qcm_id, question_id)` interdit la même famille deux fois ;
- la migration `0032_add_qcm_question_position.sql` effectue un backfill déterministe par `ROW_NUMBER() OVER (PARTITION BY qcm_id ORDER BY id)` ;
- `GetAllQuestionsByQCMID` et `GetQCMQuestionsIDs` trient par `position ASC` ;
- `CreateQCMQuestion` ajoute en fin avec `COALESCE(MAX(position), 0) + 1` dans l'`INSERT SELECT` ownership-aware ;
- l'ajout multiple trie les IDs numériques avant les insertions transactionnelles ;
- le retrait charge la position, supprime, puis compacte dans la même transaction via une plage temporaire compatible avec l'unicité ;
- Monter/Descendre sont des POST ownership-aware et échangent deux positions dans une transaction via `MAX(position)+1` ;
- les bornes sont des no-op normaux, et l'UI les matérialise par des boutons désactivés.

La DB garantit `position >= 1` et l'unicité ; la continuité exacte `1..n` est un invariant maintenu par les mutations applicatives transactionnelles et leurs tests. Aucun chemin métier QCM actuel ne crée volontairement un trou.

Le Preview appelle `GetQCMQuestionsAnswersInReferenceOrder`, sans mélange des IDs. La génération réelle appelle les variantes historiques mélangées. `buildQCMQuestionsInOrder` associe chaque job et chaque résultat à son index, préalloue la sortie et écrit chaque question à l'index choisi : l'ordre d'achèvement des goroutines ne peut plus modifier l'ordre demandé.

## 4. Individualisation des copies

Les trois niveaux d'individualisation sont préservés.

### Ordre des familles/questions

`BuildQcmStudentCtx` appelle `GetQCMQuestionsAnswersCtx`. La génération mini appelle également `GetQCMQuestionsAnswers`. Ces fonctions lisent les IDs par position, puis appellent `shuffleQCMQuestionIDs`, alias de `ShuffleSlice[int64]`, avant la construction indexée. Chaque copie réelle reçoit donc toujours une permutation de familles. Les workers préservent ensuite exactement cette permutation.

### Formulation principale ou variante

`BuildQuestion` et `BuildQuestionCtx` appellent `GetRandomQuestionByQuestionID`. La requête construit l'union de la question principale et de ses `alt_questions` possédées, puis utilise `ORDER BY RANDOM() LIMIT 1`. Le retry jusqu'à une formulation ayant au moins une réponse reste en place. `Tags.MainQuestionID` conserve l'ID de la famille principale, quelle que soit la formulation retenue.

### Ordre des réponses

Pour une formulation principale, `GetQuestionAnswer` / `GetQuestionAnswerCtx` appellent `ShuffleSlice` sur les réponses. Pour une variante, `GetAltQuestionAltAnswer` / `GetAltQuestionAltAnswerCtx` font de même sur les réponses alternatives.

Aucun refactor QCM récent n'a uniformisé ou supprimé l'un de ces mécanismes. Le Preview ne mélange plus les familles, mais continue intentionnellement à choisir une formulation et à mélanger ses réponses.

## 5. Families et variantes

Le contrat family reste appliqué sur tout le parcours :

- **Banque** : `handlers/questions` charge les questions principales et les variantes possédées, puis `questionfamilies.Build` les groupe ; la page présente « Question principale » et « Formulations alternatives » ;
- **Sélecteur QCM** : `GetFilteredQuestions` ne retourne que des questions principales ; `.Main.Selectable` porte la seule checkbox `question_ids` ; les variantes sont une liste informative sans checkbox ;
- **Composition** : `qcm_questions.question_id` référence uniquement `questions(id)` et `QCMQuestionItem` représente la relation de la famille principale ;
- **Génération** : elle part toujours de l'ID principal et choisit ensuite main ou alt via `GetRandomQuestionByQuestionID`.

Une `alt_question` ne peut donc pas être ajoutée comme relation indépendante dans `qcm_questions`. Les filtres portent sur les métadonnées principales, et les variantes associées restent des enfants informatifs.

## 6. Preview

Portrait et paysage partagent `loadPreviewQCMQuestions` et le même contrat :

- QCM absent : **404** via `GetQCMNameByID` et `HandleOwnedLookupError` ;
- QCM étranger : **404**, sans lancer la construction ;
- QCM possédé mais vide : **303 vers `ErrorMessageURL`** avec un message explicite demandant d'ajouter une question ;
- famille sans aucune formulation répondable : `ErrQuestionWithNoAnswer`, traduit en **303 vers `ErrorMessageURL`** ;
- ordre des familles : ordre pédagogique de référence ;
- formulation : sélection main/variante toujours aléatoire ;
- réponses : toujours mélangées.

La seule différence entre portrait et paysage reste le writer Typst. Le cycle des workspaces temporaires et le service PDF ownership-aware n'ont pas été altérés.

## 7. Suppression d'un QCM

La migration `0033_cascade_qcm_composition_delete.sql` reconstruit `qcm_questions` avec :

- `qcm_id REFERENCES qcm(id) ON DELETE CASCADE` ;
- `question_id REFERENCES questions(id) ON DELETE RESTRICT` ;
- les contraintes de position et d'unicité de `0032` ;
- les triggers `qcm_questions_owner_insert` et `qcm_questions_owner_update` recréés sans simplification.

`exams.qcm_id` reste `ON DELETE RESTRICT`. Une suppression de QCM sans Exam supprime donc sa composition, jamais ses questions de banque. Les questions partagées et les compositions des autres QCM restent intactes.

Pour un QCM utilisé, `QCMHasExams` effectue une prévalidation ownership-aware. La FK reste l'autorité finale : si un examen apparaît entre cette lecture et le DELETE, `isSQLiteForeignKeyConstraint` reconnaît le code étendu `ErrConstraintForeignKey` sans parser le texte de l'erreur. Le handler redirige alors en 303 avec le message métier « Ce QCM est utilisé par une évaluation et ne peut pas être supprimé. »

Le test de migration verrouille l'atomicité du statement SQLite : lorsque le RESTRICT Exams refuse le DELETE, le QCM, sa composition et l'examen restent tous présents. Les migrations Up et Down vérifient aussi données, IDs, positions, FKs et triggers ownership.

## 8. UX actuelle

### Mes QCM

La table CRUD a été remplacée par des cartes responsive. Chaque carte montre nom et nombre de questions. « Gérer les questions » est l'action principale ; aperçus, modifier et supprimer sont secondaires, textuels, et les aperçus d'un QCM vide sont désactivés. L'état vide guide vers « Créer un QCM ».

### Créer, modifier et supprimer un QCM

Les pages ont des titres métier, labels associés, largeur raisonnable, actions textuelles qui peuvent s'enrouler et Annuler vers « Mes QCM ». La suppression explique que la composition est supprimée mais pas les questions de banque. Aucun ancien script JavaScript ne filtre les guillemets ; le serveur conserve `TrimSpace`, les contraintes DB et l'unicité par utilisateur.

### Questions du QCM

La page montre le nom parent, le nombre de questions, la position stockée et le contenu principal dans des cartes. Monter/Descendre sont des formulaires POST textuels, Retirer est secondaire/destructif, les aperçus sont accessibles depuis la composition, et l'état vide masque les actions d'aperçu inutiles.

### Ajouter des questions

Le QCM parent est explicite. Les six filtres GET, le reset, la sélection multiple POST, les métadonnées principales, les variantes enfants et les deux états vides sont présentés sans table ni JavaScript personnalisé. Annuler retourne à la composition.

### Reliquat significatif

La confirmation de retrait d'une question n'a pas suivi cette harmonisation. Elle conserve un titre et un texte informels, un bouton « C'est mon dernier mot », aucun Annuler, et une mise en page ancienne. C'est le seul reliquat CRUD/terminologique manifeste dans les templates QCM inspectés.

Aucune URL technique n'est affichée, aucun bouton important n'est icon-only, et aucun ancien script de filtrage n'est présent dans les autres pages du bloc.

## 9. Données de vue

| Structure | État | Conclusion |
|---|---|---|
| `QCMContext` | `ID int64`, `Name string`, utilisé pour les vrais parents de page | Légitime et correctement délimité. |
| `QCMListItem` | ID, nom, compteur et URLs propres à chaque QCM | Légitime ; supprime les tableaux d'URLs parallèles. |
| `QCMQuestionItem` | relation, position, contenu, bornes et actions | Légitime ; ordre et relation ne dépendent plus d'indices parallèles externes. |
| `QCMQuestionSelectorData` | URLs, référentiels, familles, sélections et état des filtres | Légitime ; données spécifiques au sélecteur, sans prétention générique. |

`QCMPageData.ExtraData` existe encore dans le type mais n'est plus alimenté ni lu par les handlers/templates QCM actuels. Sa suppression serait un nettoyage mineur sans valeur fonctionnelle ; elle ne justifie pas un jalon autonome.

`QCMQuestionPageData.ExtraData` ne sert plus qu'à la confirmation de retrait, pour `QCMQuestionID` sous forme de chaîne et `QuestionContent`. C'est une dette intéressante à retirer lors de l'harmonisation de cette confirmation, en introduisant au plus deux champs typés dédiés. Les collections, URLs et filtres du sélecteur ne sont plus dans `ExtraData`.

Il n'est ni nécessaire ni souhaitable de typer davantage uniquement pour atteindre zéro `ExtraData`.

## 10. SQL, ownership et intégrité

Les requêtes pertinentes appliquent les contrats attendus :

- `GetAllQCM(user_id)` utilise un unique `LEFT JOIN` sur `qcm_id` **et** `user_id`, agrège avec `COUNT(qcm_questions.id)` et ne crée aucun N+1 ni mélange de compte entre utilisateurs ;
- les lookups QCM filtrent `id + user_id` ; les mutations QCM utilisent le même couple et contrôlent les lignes affectées ;
- les lectures de composition filtrent question, relation et parent possédé ;
- la création de relation est un `INSERT SELECT` qui exige QCM et question possédés, complété par les triggers DB ;
- retrait et déplacement exigent `user_id + qcm_id + qcm_question_id`, vérifient les rows affected et s'exécutent en transaction ;
- `UNIQUE(qcm_id, question_id)`, `UNIQUE(qcm_id, position)` et `CHECK(position >= 1)` restent actifs après `0032` puis `0033` ;
- la migration `0033` conserve les triggers ownership et ne transforme pas le FK question en cascade ;
- l'ajout multiple est atomique : une question étrangère ou un doublon annule le lot ;
- les mouvements et le compactage vérifient le nombre de lignes attendu avant commit.

Aucune régression n'a été trouvée dans le comptage `LEFT JOIN + COUNT`, la position ou la cascade de composition.

La prévalidation Exams puis DELETE n'est pas dans une transaction unique, mais la FK RESTRICT ferme correctement la fenêtre TOCTOU et son erreur est traduite de façon robuste. Il n'en résulte aucun état partiel.

## 11. POST sans sélection

Le comportement actuel est exactement le suivant :

1. le handler authentifie l'utilisateur et parse `qcm_id` ;
2. `r.Form["question_ids"]` est vide ;
3. la slice vide est triée ;
4. une transaction est ouverte ;
5. aucun `CreateQCMQuestion` n'est exécuté ;
6. la transaction vide est commit ;
7. le handler redirige en 303 vers la composition demandée.

Il n'y a ni mutation, ni fuite de données, ni contournement d'un contrôle sur une question. En revanche, un QCM absent ou étranger n'est jamais validé sur ce chemin et reçoit un succès apparent avant que la page cible ne réponde éventuellement 404. C'est une asymétrie ownership-aware et un travail DB inutile.

Recommandation : **B — valider le parent possédé, puis traiter l'absence de sélection comme un no-op avant d'ouvrir la transaction**. La validation doit employer `GetQCMNameByID(qcmID, userID)` ou le lookup existant équivalent et `HandleOwnedLookupError`. Ce point n'est pas un P0/P1 ; c'est un petit correctif P2 de cohérence fonctionnelle.

## 12. Couverture de tests actuelle

| Domaine | Couverture actuelle | Manque utile |
|---|---|---|
| CRUD QCM | formulaires/contextes, noms avec apostrophes/guillemets, ownership edit/delete, suppression protégée | Aucun manque bloquant. |
| Liste | items typés, compteurs, URLs isolées par QCM, état vide, comptage ownership sans N+1 | Aucun. |
| Position/migration | backfill réel, contraintes, ownership triggers, Up/Down, lectures par position, ajout indépendant | Aucun. |
| Ajout multiple | tri déterministe, rollback d'une sélection mixte, contraintes DB | Ajouter le cas zéro sélection lors du correctif recommandé. |
| Retrait | succès, compactage, ownership/mauvais parent, rollback du compactage | La donnée de confirmation est couverte, mais pas son UX — ce qui ne justifie pas un snapshot HTML. |
| Monter/Descendre | milieu, bornes, deux éléments, identité, positions, rollback, handlers/ownership | Aucun. |
| Concurrence | achèvement inversé, permutation préservée, erreurs, annulation contextuelle, absence de construction double | Aucun. |
| Preview | absent/étranger, portrait/paysage communs, ordre de référence, QCM vide, question sans réponse | Aucun manque bloquant. |
| Suppression QCM | migration réelle, cascade, questions partagées, isolation, Exam RESTRICT/atomicité, handler et race FK | Aucun. |
| Families/sélecteur | ownership, exclusion des présentes, six filtres sur main, variantes enfants, états vides | Aucun hors POST vide. |

Le seul test automatisé à forte valeur encore manquant est donc le POST vide pour un QCM possédé, absent et étranger, à écrire avec le petit correctif correspondant. Pour la randomisation réelle, un test statistique serait fragile et n'apporterait pas plus que les tests déterministes des hooks de shuffle et la vérification manuelle de copies réelles.

## 13. Checklist de smoke tests manuels

1. Depuis « Mes QCM », créer un QCM et vérifier son apparition avec `0 question`.
2. Ouvrir « Gérer les questions », puis « Ajouter des questions ».
3. Vérifier que plusieurs familles peuvent être cochées, que les variantes sont visibles comme enfants et qu'aucune variante n'a sa propre checkbox.
4. Ajouter plusieurs familles et vérifier leur ordre déterministe initial ainsi que les positions `1..n`.
5. Utiliser Monter/Descendre, y compris aux bornes, puis recharger et vérifier que l'ordre reste stable.
6. Retirer une question au milieu et vérifier que les positions sont compactées.
7. Ouvrir les aperçus portrait et paysage plusieurs fois : l'ordre des familles doit rester celui de la composition.
8. Générer plusieurs copies réelles de la même évaluation et constater les trois individualisations : ordre des questions potentiellement différent, formulation principale/variante potentiellement différente, réponses mélangées.
9. Supprimer un QCM non utilisé : il disparaît, tandis que ses questions restent dans la banque.
10. Créer une évaluation utilisant un QCM, tenter de supprimer ce QCM et vérifier la redirection avec le message métier, puis confirmer que QCM, composition et évaluation existent toujours.
11. Vérifier sur mobile ou fenêtre étroite les cartes de liste, composition et sélecteur, ainsi que l'enroulement des actions.

## 14. Workflow enseignant

Le parcours principal est désormais compréhensible et cohérent :

```text
Mes QCM
  -> Gérer les questions
  -> Ajouter des questions depuis la banque
  -> Monter / Descendre / Retirer
  -> Aperçu portrait ou paysage
  -> Évaluations via la navigation globale
```

Les retours sont cohérents : les formulaires QCM reviennent à « Mes QCM », le sélecteur revient à la composition, et la composition revient à la liste. La seule rupture interne est la confirmation de retrait, dépourvue d'Annuler. Le passage vers Évaluations existe dans la navigation globale, mais pas comme continuation contextuelle depuis le QCM ; cela ne bloque pas le workflow et peut être traité au début du chantier Exams.

## 15. Priorités restantes

### P0 — 0

Aucun problème restant de sécurité, corruption ou perte de données n'a été identifié.

### P1 — 0

Aucun bug métier ou d'intégrité important ne reste dans le bloc QCM.

### P2 — 2

1. **Prévalider le QCM sur un POST sans sélection.**
   - Impact : succès apparent pour un parent absent/étranger et transaction vide inutile ; aucune mutation ni fuite.
   - Fichiers : `internal/handlers/qcmQuestions/handlers.go`, `internal/handlers/qcmQuestions/handlers_test.go`.
   - Recommandation : lookup ownership-aware, puis no-op 303 avant transaction si `question_ids` est vide.
   - Taille : **petite**.

2. **Harmoniser la confirmation de retrait d'une question.**
   - Impact : dernier écran QCM au vocabulaire informel, sans Annuler ; rupture visible du workflow enseignant.
   - Fichiers : `internal/templates/qcmquestions/delete_form_qcm_question.html`, `internal/templates/data/qcmQuestions.go`, `internal/handlers/qcmQuestions/handlers.go` et test ciblé de données de vue.
   - Recommandation : contexte QCM visible, question concernée, wording « Retirer la question du QCM », Annuler vers la composition, bouton danger textuel ; remplacer le petit `ExtraData` par des champs typés si cela simplifie le diff.
   - Taille : **petite**.

### P3 — 1

1. **Proposer une continuation contextuelle vers Évaluations.**
   - Impact : la navigation globale fonctionne, mais le professeur doit comprendre seul l'étape suivante après le Preview.
   - Fichiers : à décider dans le futur chantier Exams ; potentiellement la composition ou la liste QCM et le formulaire de création d'évaluation.
   - Recommandation : étudier « Utiliser dans une évaluation » avec le QCM préselectionné, sans fusionner QCM et Exam.
   - Taille : **petite à moyenne**, selon le contrat de préselection Exams.

Le champ `QCMPageData.ExtraData` inutilisé est un nettoyage facultatif à faire opportunistiquement, pas une priorité autonome.

## 16. Verdict final

**B — le bloc QCM est terminé après deux petits correctifs.**

Le prochain travail recommandé est un unique petit jalon de clôture combinant :

1. prévalidation ownership-aware du POST sans sélection avant toute transaction ;
2. harmonisation de la confirmation de retrait avec retour vers la composition.

Après ce jalon, le chantier peut passer à Exams. Le lien contextuel « Utiliser dans une évaluation » doit être considéré comme un sujet du workflow Exams, pas comme une raison de rouvrir l'architecture QCM.

## 17. Fichiers vérifiés et absence de modification de code

Le code actuel a été vérifié dans les handlers, templates et données de vue QCM/QCM Questions/QCM Preview, les helpers de construction et de génération directement concernés, les requêtes `qcm.sql`, `qcmquestion.sql` et `questions.sql`, les migrations `0032`/`0033`, la FK Exams, les routes et les tests QCM pertinents. Les rapports intermédiaires ont été utilisés comme historique, jamais comme autorité à la place du code.

Aucun fichier Go, SQL, migration, template, test ou configuration n'a été modifié. Le seul fichier créé par cet audit est `docs/audits/qcm-closure-audit.md`. Aucun commit n'a été créé.
