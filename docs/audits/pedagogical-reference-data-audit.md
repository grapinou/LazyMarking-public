# Audit des référentiels pédagogiques

## 1. Résumé exécutif

Les six référentiels pédagogiques ont été audités de bout en bout : schéma historique et final, requêtes sqlc, handlers, routes, templates, données de vue, intégration aux questions et tests.

Le modèle relationnel est simple et cohérent. Toute question possède obligatoirement une matière, un thème, un niveau, une compétence, une difficulté et une valeur de points. Les six FKs sont `NOT NULL` et `ON DELETE RESTRICT`. Aucun référentiel n'est directement rattaché à un autre : un thème, une compétence ou un niveau est global au compte enseignant, pas enfant d'une matière.

La sémantique de suppression DB est déjà la bonne dans les six cas : une valeur libre peut être supprimée ; une valeur utilisée est protégée ; aucune question n'est supprimée ou rendue orpheline. **Aucune migration de suppression n'est nécessaire.** Les handlers traduisent aujourd'hui l'échec FK en redirection métier, mais ils traduisent aussi toute autre erreur DB comme si la valeur était utilisée. Cette classification doit être rendue précise avant la refonte des confirmations.

L'ownership courant est correctement défendu aux niveaux HTTP, SQL et DB : listes et lookups filtrés par `user_id`, mutations ownership-aware avec contrôle des lignes affectées, associations Question/référentiels contrôlées par `INSERT SELECT`/`UPDATE` et triggers. Une réserve historique subsiste : la migration 0030 interdit les nouvelles relations inter-utilisateurs mais n'a pas audité rétroactivement les lignes préexistantes ; les lectures modernes masquent explicitement une éventuelle ligne incohérente.

Deux problèmes P1 concernent spécifiquement Points : le formulaire d'édition ne préselectionne pas la valeur stockée, donc une soumission sans changement peut transformer la valeur en `1` pour toutes les questions liées ; et le backend/DB acceptent tout entier, y compris zéro ou une valeur négative, alors que l'UI ne propose que `1..100`.

Le motif UX est extrêmement commun pour les cinq référentiels textuels. Points doit partager la grammaire visuelle mais conserver son contrôle numérique et son contrat métier propre.

## 2. Cartographie des six référentiels

| Domaine métier | Table / modèle sqlc | Migration d'origine | Requête | Handler / routes / templates | Vocabulaire affiché actuel |
|---|---|---|---|---|---|
| Matière | `subjects` / `db.Subject` | `0003_create_subjects_table.sql` | `db/query/subjects.sql` | `internal/handlers/subjects`; liste `/dashboard/questions/subjects`; CRUD `/subjects/{add,edit,delete}`; `internal/templates/subjects` | « Matière » ; certains `PageTitle` restent anglais (`subjects`, `add subject`, etc.). |
| Thème | `themes` / `db.Theme` | `0004_create_themes_table.sql` | `db/query/themes.sql` | `internal/handlers/themes`; liste `/dashboard/questions/themes`; CRUD `/themes/{add,edit,delete}`; `internal/templates/themes` | « Thème » ; titres internes anglais. |
| Niveau | `year_levels` / `db.YearLevel` | `0005_create_year_levels_table.sql` | `db/query/year_levels.sql` | `internal/handlers/yearlevels`; liste `/dashboard/questions/year-levels`; CRUD `/yearlevels/{add,edit,delete}`; `internal/templates/yearlevels` | La banque et les filtres disent « Niveau », mais les pages dédiées disent encore « Classe ». Cette collision avec les vraies classes (`class_codes`) est à corriger. |
| Compétence | `skills` / `db.Skill` | `0006_create_skills_table.sql` | `db/query/skills.sql` | `internal/handlers/skills`; liste `/dashboard/questions/skills`; CRUD `/skills/{add,edit,delete}`; `internal/templates/skills` | « Compétence » ; titres internes anglais. |
| Difficulté | `difficulties` / `db.Difficulty` | `0007_create_difficulties_table.sql` | `db/query/difficulties.sql` | `internal/handlers/difficulties`; liste `/dashboard/questions/difficulties`; CRUD `/difficulties/{add,edit,delete}`; `internal/templates/difficulties` | « Difficulté », souvent mal orthographié « difficultée » et exemples « difficil ». |
| Points | `points` / `db.Point` | `0008_create_points_table.sql` | `db/query/points.sql` | `internal/handlers/points`; liste `/dashboard/questions/points`; CRUD `/points/{add,edit,delete}`; `internal/templates/points` | « Point(s) », avec accords incohérents (« nombre de point », « Ajouter point »). |

Chaque package expose le même cycle : liste GET via une route de `QuestionRoutes`, formulaire GET et mutation POST sur les routes propres au domaine. Les six utilisent un `*PageData`, un `*Routes`, un `*ActionURLs` et quatre templates add/edit/delete/table.

## 3. Schéma relationnel final

### Référentiels textuels

Après la migration corrective 0029, `subjects`, `themes`, `year_levels`, `skills` et `difficulties` ont le même contrat :

- `id INTEGER PRIMARY KEY AUTOINCREMENT` ;
- `name TEXT NOT NULL CHECK(length(trim(name)) > 0)` ;
- `user_id INTEGER NOT NULL REFERENCES users(id)` ;
- `UNIQUE(name, user_id)`.

Deux enseignants peuvent donc employer le même libellé ; un enseignant ne peut pas créer deux fois exactement le même libellé.

### Points

Après 0029, `points` contient :

- `id INTEGER PRIMARY KEY AUTOINCREMENT` ;
- `point_value INTEGER NOT NULL DEFAULT 1` ;
- `user_id INTEGER NOT NULL REFERENCES users(id)` ;
- `UNIQUE(point_value, user_id)`.

Points est bien une table de référentiel normalisée, pas une valeur portée directement par chaque question. Modifier une ligne Points modifie donc la valeur pédagogique de toutes les questions qui la référencent.

### Questions

Le schéma final de `questions`, reconstruit en 0029, contient six colonnes obligatoires :

| Colonne Question | Cible | Nullabilité | Suppression | Update FK |
|---|---|---|---|---|
| `subject_id` | `subjects(id)` | `NOT NULL` | `ON DELETE RESTRICT` | aucun `ON UPDATE` explicite, donc `NO ACTION` |
| `theme_id` | `themes(id)` | `NOT NULL` | `ON DELETE RESTRICT` | `NO ACTION` |
| `year_level_id` | `year_levels(id)` | `NOT NULL` | `ON DELETE RESTRICT` | `NO ACTION` |
| `skill_id` | `skills(id)` | `NOT NULL` | `ON DELETE RESTRICT` | `NO ACTION` |
| `difficulty_id` | `difficulties(id)` | `NOT NULL` | `ON DELETE RESTRICT` | `NO ACTION` |
| `point_id` | `points(id)` | `NOT NULL` | `ON DELETE RESTRICT` | `NO ACTION` |

Une question ne peut donc exister sans l'une de ces six caractéristiques. `SET NULL` n'est compatible ni avec le schéma ni avec le modèle actuel.

### Relations entre référentiels

Il n'existe aucune FK ni table de jointure entre les six référentiels :

- un thème n'appartient pas à une matière ;
- une compétence n'appartient pas à une matière ;
- un niveau n'appartient pas à une matière ;
- difficulté et points sont également globaux au compte.

Les formulaires Question combinent librement une valeur possédée de chaque liste. Les filtres appliquent indépendamment les six IDs aux métadonnées de la question principale. Ce modèle est cohérent avec le code actuel ; aucune hiérarchie implicite ne doit être inventée pendant la refonte UX.

## 4. Ownership

### CRUD des référentiels

Pour les six domaines :

- création : `user_id` provient de la session authentifiée ;
- liste : `GetAll* WHERE user_id = :user_id` ;
- formulaire edit/delete : lookup `id + user_id`, puis `HandleOwnedLookupError`, donc absent/étranger donne 404 ;
- mutation edit/delete : `WHERE id = :id AND user_id = :user_id`, requête `:execrows`, puis `HandleOwnedMutationRows` ; zéro ligne donne 404 ;
- une ligne étrangère ne peut donc être lue, modifiée ou supprimée par ces parcours.

### Association à une question

`CreateQuestion` exige par six `EXISTS` que chaque ID appartienne au même `user_id`. `UpdateQuestion` exige la même chose pour la question cible et les six nouvelles valeurs. Les triggers `questions_owner_insert` et `questions_owner_update` de 0030 répètent cette garantie directement dans SQLite.

Les lectures (`GetAllQuestions`, `GetQuestionByID`, `GetTagsByQuestionID`, `GetFilteredQuestions`, `GetRandomQuestionByQuestionID`) vérifient aussi la cohérence d'ownership des référentiels. Une éventuelle ligne historique incohérente est masquée plutôt qu'exposée.

### Conclusion ownership

L'ownership courant est correct pour les six référentiels. Aucune faille P0/P1 actuelle n'a été trouvée. Réserve historique : 0030 ajoute des triggers mais ne valide pas les lignes déjà présentes. `TestQuestionReadsHideLegacyInconsistentRows` prouve que le projet anticipe ce cas en lecture, sans démontrer qu'une base réelle ancienne en est exempte. Une vérification d'intégrité ponctuelle est recommandée avant toute future migration de ces tables.

## 5. Suppressions actuelles

Le comportement normal est identique dans les six domaines.

| Référentiel | Valeur non utilisée | Valeur utilisée par une question | Réponse handler actuelle | Question conservée ? |
|---|---|---|---|---|
| Matière | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | 303 vers `ErrorMessageURL` : « Ce champ est utilisé… » | Oui |
| Thème | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | même redirection | Oui |
| Niveau | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | même redirection | Oui |
| Compétence | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | même redirection | Oui |
| Difficulté | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | même redirection | Oui |
| Points | DELETE réussi, 303 vers liste | FK `RESTRICT`, DELETE échoue | même redirection | Oui |

Pour une ressource absente ou étrangère, le DELETE affecte zéro ligne et `HandleOwnedMutationRows` produit 404. Les contraintes sont effectives parce que `InitDB` active les foreign keys SQLite sur les connexions ; ce contrat possède un test dédié.

Limite commune : chaque handler considère **toute** erreur retournée par `Delete*` comme « valeur utilisée ». Une panne DB réelle ou une autre contrainte serait donc présentée en 303 avec un message métier faux au lieu d'un 500. La suppression est sûre, mais la classification d'erreur n'est pas assez précise.

## 6. Recommandations de suppression

| Référentiel | Contrat actuel | Contrat recommandé | Changement DB ? | Changement handler/UX ? |
|---|---|---|---|---|
| Matière | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite des questions avant suppression | Non | Oui |
| Thème | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite avant suppression | Non | Oui |
| Niveau | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite avant suppression | Non | Oui |
| Compétence | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite avant suppression | Non | Oui |
| Difficulté | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite avant suppression | Non | Oui |
| Points | obligatoire + RESTRICT | conserver RESTRICT ; réaffectation explicite avant suppression | Non | Oui |

Il ne faut introduire ni CASCADE vers `questions`, qui détruirait la banque, ni `SET NULL`, qui contredirait le modèle obligatoire et rendrait filtres/génération incomplets.

Le handler devrait distinguer la violation FK avec le code étendu SQLite, comme le fait déjà le bloc QCM, ou effectuer une prévalidation d'usage tout en conservant la FK comme autorité finale. Une violation FK produit le message pédagogique ; une vraie erreur DB reste 500. La confirmation devrait expliquer : « Cette valeur ne pourra pas être supprimée tant qu'elle est utilisée par une question. » Une future action de réaffectation peut être étudiée séparément ; elle n'est pas nécessaire pour refaire les listes.

Synthèse centrale : **les six sémantiques DB de suppression sont déjà correctes ; les six handlers/confirmations doivent être harmonisés et leur classification d'erreur durcie.**

## 7. Cas particulier des Points

Le modèle de référentiel reste pertinent : plusieurs questions peuvent partager la même valeur, l'unicité est par utilisateur et les filtres utilisent un ID stable. Il n'y a pas de raison de remplacer systématiquement `point_id` par un entier dans `questions`.

Deux défauts demandent une décision avant l'UX :

1. `EditFormPointHandler` lit la valeur courante uniquement pour vérifier l'ownership, puis la jette. Le template reçoit `PointID` et `Seq`, mais aucune valeur sélectionnée. Le premier `<option>` (`1`) est donc affiché par défaut. Soumettre sans changement transforme une valeur 5, 10, etc. en 1 et affecte toutes les questions liées.
2. Add/Edit parsèment tout entier signé. La DB n'a aucun `CHECK`; un POST forgé peut stocker `0`, `-1` ou `1000`, alors que l'UI propose seulement `1..100`. C'est un problème d'intégrité métier du compte, pas une fuite inter-utilisateur.

Recommandation : confirmer le contrat simple `point_value >= 1`, décider explicitement si 100 est seulement une commodité UI ou une limite métier, auditer les valeurs existantes, puis aligner validation serveur, contrainte DB et options. Le formulaire edit doit toujours préselectionner la valeur stockée. Aucun redesign plus profond du modèle Points n'est justifié.

## 8. Intégration aux Questions

### Création et modification

`GetAllFeaturesQuestion` effectue six lectures globales ownership-aware. Add/Edit refusent d'afficher le formulaire si l'une des listes est vide, ce qui correspond aux six FKs obligatoires. Les POST parsèment les six IDs et les mutations SQL revérifient leur ownership. Il n'existe pas de confiance dans les seules options HTML.

Le helper retourne toutefois un `map[string]any`; c'est une dette locale au bloc Questions, pas une raison de refondre les six référentiels dans cet audit.

### Banque et filtres

Les métadonnées sont attachées à la question principale. `GetFilteredQuestions` joint les six tables, filtre leurs `user_id` et applique les filtres indépendamment. Les variantes n'ont pas leurs propres référentiels. Le sélecteur QCM réutilise exactement ces familles et ces métadonnées principales.

### Performance et cohérence

- aucun N+1 par question/famille dans le sélecteur : nombre fixe de requêtes de référentiels, une requête filtrée et une lecture globale des variantes ;
- six requêtes de listes dans les formulaires Question, mais coût constant ;
- add/edit/filter utilisent les mêmes six domaines ;
- les FKs RESTRICT empêchent les valeurs devenues orphelines ;
- les lectures modernes cachent une éventuelle relation historique inter-utilisateur.

Aucune incohérence entre création, modification et filtrage n'a été trouvée, hormis le contrat numérique Points.

## 9. UX actuelle des listes

Les six listes sont presque des copies exactes :

- titre formulé comme « Ajouter… » plutôt que comme nom du référentiel ;
- exemple libre sous le titre ;
- bouton anglais « Back to question » ;
- action Ajouter ;
- table Bootstrap non enveloppée dans un conteneur responsive ;
- colonne technique « Edit/Sup » ;
- boutons Modifier/Supprimer uniquement iconographiques ;
- état vide « Pas de … pour l'instant… » sans explication ni appel d'action dédié ;
- tableaux parallèles `rows` / `Action` indexés dans le template.

Différences de vocabulaire : YearLevel est présenté comme « Classe » plutôt que « Niveau » ; Difficulté comporte plusieurs fautes ; Points a des accords incohérents. La navigation revient vers la Banque de questions, ce qui est conceptuellement correct, mais son libellé ne l'est pas.

Direction future : page « Matières », « Thèmes », etc., compteur éventuel, liste/cartes responsive, action Ajouter visible, actions textuelles Modifier/Supprimer, état vide pédagogique et « Retour à la banque de questions ».

## 10. UX actuelle des formulaires

### Commun aux cinq référentiels textuels

- add/edit ont un unique champ texte, sans `id` ni association `for` sur le label ;
- aucun formulaire n'offre Annuler ;
- titres « Editer » et `PageTitle` anglais/techniques ;
- add/edit embarquent un JavaScript local qui retire les guillemets, sans contrainte backend correspondante ;
- delete emploie « Es-tu sur » et « C'est mon dernier mot » ;
- delete n'explique pas la protection lorsqu'une question utilise la valeur ;
- IDs et valeurs sont transportés en chaînes dans `ExtraData`.

Le script de retrait des guillemets doit disparaître : le vrai contrat est trim + non-vide + unicité par utilisateur. Apostrophes et guillemets n'ont aucune raison métier d'être interdits.

### Points

Points n'a pas le script de guillemets et utilise un `<select>` 1..100, mais partage l'absence d'Annuler, les confirmations informelles, les labels non associés et les `ExtraData`. Son edit ne préselectionne pas la valeur courante, ce qui constitue le bug P1 décrit plus haut.

Les erreurs Create/Update de chaque domaine sont également agrégées sous un même message « doublon ou vide », y compris une vraie erreur DB. Cette dette peut être traitée avec la même convention d'erreurs que la suppression, sans déplacer les contraintes vers HTML.

## 11. Motif UX commun futur

Un langage visuel commun est pertinent :

- en-tête au pluriel et courte explication métier ;
- liste responsive avec compteur, état vide et Ajouter ;
- item montrant valeur + actions textuelles Modifier/Supprimer ;
- formulaire add/edit court, label explicite, aide optionnelle, action principale + Annuler ;
- confirmation destructive montrant la valeur, précisant la protection des questions, Supprimer + Annuler ;
- retour cohérent vers la liste du référentiel et vers la Banque.

Il ne faut pas en déduire un service CRUD ou un repository générique. Les cinq domaines textuels ont une forme identique mais des types, routes et libellés métier clairs peuvent rester explicites.

Points ne doit pas être forcé dans un champ texte : valeur numérique, préselection, validation et éventuelle plage sont spécifiques. YearLevel doit conserver le vocabulaire « Niveau » et ne pas être confondu avec les classes d'élèves. Aucun domaine ne doit acquérir artificiellement un parent Subject.

## 12. View-data et ExtraData

Les six `PageData` exposent `Routes`, routes du domaine, `PageTitle` et `ExtraData map[string]any`.

| Usage | Évaluation | Recommandation |
|---|---|---|
| `NoSubject`, `NoTheme`, etc. | inutile : dérivable de la longueur de la liste | supprimer lors du typage de liste |
| collections `Subjects`, `Themes`, etc. | légitimes, mais map fragile | remplacer par une slice typée d'items de vue |
| `Action` parallèle aux collections | fragile | intégrer EditURL/DeleteURL dans chaque item |
| valeurs edit/delete (`Subject`, etc.) | cohérentes mais faiblement typées | petit contexte dédié avec `ID int64`, valeur et CancelURL |
| IDs edit/delete sous forme de string | fragile | conserver `int64` dans Go |
| `Seq` Points | donnée de vue légitime | champ typé `[]int` ou bornes, accompagné de la valeur courante |

Une petite forme conceptuelle `ReferenceListItem{ID, Label, EditURL, DeleteURL}` pourrait réduire les tableaux parallèles, mais il n'est pas nécessaire de créer un modèle générique transversal avant d'avoir refait un premier domaine. Le principe important est l'item cohérent, pas l'abstraction.

## 13. Tests actuels et manques utiles

### Couverture existante

- `internal/db/referenceMutationRows_test.go` couvre table-driven les updates/deletes possédés, absents et étrangers pour les six domaines, ainsi que rows affected ;
- ce fichier vérifie sur Subject qu'une contrainte unique/FK retourne une erreur SQL et non zéro ligne ;
- `internal/handlers/subjects/handlers_test.go` couvre 404 edit/delete GET et POST pour Subject absent/étranger ;
- `internal/db/questionGraphIntegrity_test.go` couvre Create/Update Question avec chacun des six parents étrangers, les lectures incohérentes masquées et les six filtres ;
- `internal/db/ownership_test.go` applique réellement 0030 et vérifie qu'une question inter-utilisateur est refusée ;
- `internal/db/init_test.go` verrouille l'activation des FKs SQLite sur les connexions.

### Manques ayant une vraie valeur

1. Test du schéma final réel pour les six suppressions : valeur libre supprimée, valeur utilisée refusée, question conservée.
2. Tests handlers table-driven sur les six domaines : succès, absent/étranger 404, FK utilisée redirigée, vraie erreur DB 500 après durcissement.
3. Tests Create/Edit et données de vue au-delà de Subject, sans snapshots Bootstrap.
4. Tests Points : valeur courante préselectionnée, zéro/négatif rejeté, valeur positive acceptée, unicité par utilisateur.
5. Test de migration 0029 sur un schéma historique, prouvant données/IDs/FKs préservés et unicité devenue locale à l'utilisateur.
6. Audit/test ciblé des éventuelles relations historiques antérieures à 0030.

Il n'est pas utile de dupliquer six infrastructures de test ; les invariants réellement identiques se prêtent à des tables de cas.

## 14. Migrations historiques et bases réelles

Les migrations d'origine 0003 à 0008 déclarent à la fois `UNIQUE` sur la valeur seule et `UNIQUE(value, user_id)`. Elles imposaient donc en pratique une unicité globale entre enseignants.

La migration 0029, `NO TRANSACTION`, désactive temporairement les foreign keys, reconstruit les six référentiels puis `questions`, préserve toutes les colonnes/IDs et retire les contraintes globales au profit des contraintes par utilisateur. Son Down est volontairement impossible pour éviter une perte de données. Une base créée de zéro aujourd'hui traverse bien cette correction ; une base historique migrée obtient le même schéma final.

La reconstruction de `questions` en 0029 réaffirme les six `NOT NULL` et `ON DELETE RESTRICT`. La migration 0030, appliquée ensuite, crée les triggers ownership. Aucune migration ultérieure ne reconstruit `questions` ou les six référentiels ; les migrations QCM 0032/0033 ne touchent que `qcm_questions`.

Deux réserves de validation : il n'existe pas de test de migration réel dédié à 0029, et 0030 ne contrôle pas rétroactivement les lignes antérieures. Ces points ne démontrent pas une corruption, mais justifient une vérification ciblée avant une nouvelle migration Points.

## 15. Classification finale

### P0 — 0

Aucune fuite inter-utilisateur, cascade destructive ou corruption immédiate n'a été trouvée.

### P1 — 2

1. **Points — édition non préremplie.**
   - Impact : soumettre le formulaire sans intention de changement peut remplacer la valeur par 1 pour toutes les questions liées.
   - Fichiers : `internal/handlers/points/handlers.go`, `internal/templates/points/edit_form_point.html`, données de vue et tests Points.
   - Recommandation : transmettre la valeur courante et marquer l'option correspondante `selected`.
   - Taille : petite.

2. **Points — domaine numérique non contraint côté serveur/DB.**
   - Impact : zéro/négatif ou valeur hors plage UI possible, avec effet sur notation et filtres.
   - Fichiers : migration future `points`, handlers Points, tests migration/handler.
   - Recommandation : contrat positif explicite, audit des valeurs existantes, validation serveur et `CHECK`; décider séparément d'une limite haute.
   - Taille : moyenne.

### P2 — 5

1. **Six suppressions — toute erreur DB est présentée comme FK utilisée.**
   - Impact : panne réelle masquée par un message métier faux.
   - Fichiers : les six `handlers.go`; éventuellement un minuscule helper neutre de classification SQLite.
   - Recommandation : distinguer FK de vraie erreur et conserver rows/404.
   - Taille : petite.

2. **Données de vue fragiles et tableaux parallèles.**
   - Impact : risque d'URL associée au mauvais item pendant la refonte, IDs convertis en chaînes.
   - Fichiers : six fichiers `internal/templates/data/*.go`, handlers et templates de listes/formulaires.
   - Recommandation : items et contextes ciblés, sans framework CRUD.
   - Taille : moyenne mais mécanique.

3. **Scripts frontend supprimant les guillemets sur cinq domaines.**
   - Impact : restriction utilisateur non soutenue par le backend, contrat incohérent.
   - Fichiers : add/edit Subjects, Themes, YearLevels, Skills, Difficulties.
   - Recommandation : retirer les scripts, conserver trim/non-vide/unicité serveur.
   - Taille : petite.

4. **Couverture handler/suppression inégale.**
   - Impact : seule Matière verrouille actuellement les 404 GET/POST ; aucun tableau complet ne prouve used/unused et réponse HTTP des six.
   - Fichiers : tests des six handlers et `internal/db`.
   - Recommandation : infrastructure table-driven minimale centrée sur données et statuts.
   - Taille : moyenne.

5. **Cohérence historique non vérifiée rétroactivement.**
   - Impact : une base ancienne pourrait contenir une question inter-utilisateur cachée par les lectures modernes.
   - Fichiers : test/audit de migration ou commande de diagnostic ponctuelle ; migrations 0029/0030 comme référence.
   - Recommandation : scan read-only avant toute reconstruction future, signaler plutôt que réparer silencieusement.
   - Taille : petite à moyenne.

### P3 — 2

1. **Listes CRUD anciennes.**
   - Impact : tables peu responsive, boutons icon-only, états vides faibles, anglais « Back to question ».
   - Fichiers : six templates `table_*`.
   - Recommandation : motif responsive commun et actions textuelles.
   - Taille : moyenne.

2. **Formulaires et terminologie anciennes.**
   - Impact : pas d'Annuler, labels non associés, confirmations informelles, « Classe »/« difficultée »/accords Points.
   - Fichiers : dix-huit templates add/edit/delete et PageTitle handlers.
   - Recommandation : motif court commun, wording professionnel, navigation vers la liste.
   - Taille : moyenne.

## 16. Plan incrémental recommandé

1. **Contrat Points** : décider `>=1` et la limite haute éventuelle, auditer les valeurs existantes, corriger la valeur courante d'edit, ajouter validation/migration/tests.
2. **Suppression commune** : conserver les six RESTRICT, distinguer précisément FK/autres erreurs et ajouter la matrice used/unused/ownership/questions conservées.
3. **View-data pilote** : typer items et contexte sur un référentiel textuel, sans abstraction générale ; valider le motif avant réplication.
4. **Listes** : appliquer le motif commun aux cinq textes, puis à Points avec son libellé numérique.
5. **Formulaires add/edit** : actions Annuler, labels accessibles, suppression du JS de guillemets ; Points conserve sa sélection spécifique.
6. **Confirmations delete** : wording professionnel, valeur concernée, avertissement RESTRICT, Annuler ; aucune promesse de suppression si utilisé.
7. **Clôture** : tests ciblés, smoke réel des six domaines et vérification que création/edit/filtres Question restent intacts.

La mutualisation doit porter sur le langage visuel et les invariants testés, pas sur une nouvelle architecture CRUD générique.

## 17. Verdict

Les suppressions ne nécessitent aucun chantier DB : **6/6 sont sémantiquement sûres et doivent conserver RESTRICT**. Avant la refonte UX générale, il faut toutefois traiter le contrat Points (deux P1) puis la classification des erreurs de suppression et les tests associés.

Le bloc est suffisamment homogène pour un chantier commun, avec deux variantes : cinq référentiels textuels strictement similaires et Points comme référentiel numérique spécialisé. Aucun lien Theme→Subject, Skill→Subject ou YearLevel→Subject ne doit être ajouté sans décision métier séparée.

## 18. Fichiers inspectés et absence de modification

Ont été inspectés : migrations 0003–0009, 0029 et 0030 ; les six requêtes sqlc et fichiers générés ; les six packages handlers/routes/views ; les vingt-quatre templates ; les six PageData ; les chemins Question, filtres et sélecteur QCM directement nécessaires ; les tests ownership, mutation rows, graph Question et Subjects.

Aucun fichier Go, SQL, migration, template, test ou configuration n'a été modifié. Le seul fichier créé est `docs/audits/pedagogical-reference-data-audit.md`. Aucun commit n'a été créé.
