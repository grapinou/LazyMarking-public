# Audit de clôture du workflow Élèves / Classes

## 1. Résumé exécutif

Le workflow Élèves / Classes est nettement consolidé : ses trois modules utilisent des données de vue typées, les parcours principaux sont cohérents, les frontières d’ownership sont présentes dans les handlers et les requêtes, et les mutations à plusieurs étapes sont transactionnelles. La dernière classe d’un élève est protégée par une condition atomique SQL, indépendamment de l’interface.

L’audit relève toutefois un défaut bloquant : le retrait d’une relation élève/classe est encore une mutation HTTP GET. Un lien de la liste déclenche directement le `DELETE`, sans POST ni confirmation. La session `SameSite=Strict` réduit les attaques intersites classiques, mais ne rend pas une mutation GET sûre face à un suivi, un préchargement ou une activation involontaire dans la session.

Conclusion : **Workflow Élèves / Classes non encore clôturable**. Le retrait de relation doit devenir une mutation POST, idéalement après confirmation. Les autres constats sont importants ou mineurs, mais aucun autre défaut d’intégrité immédiat n’a été identifié.

Décompte :

- bloquants : 1 ;
- importants : 4 ;
- mineurs : 2.

## 2. Périmètre

L’audit couvre :

- `internal/handlers/students`, ses templates et ses données de vue ;
- `internal/handlers/classCodes`, ses templates et ses données de vue ;
- `internal/handlers/studentClassCode`, ses templates et ses données de vue ;
- les requêtes `students.sql`, `classcodes.sql` et `student_class_codes.sql` ;
- les migrations définissant élèves, classes, relations, examens et triggers d’ownership ;
- les tests de handlers, builders, templates et intégrité DB associés.

La base réelle historique n’a pas été ouverte ni modifiée. Les observations historiques sont limitées aux possibilités établies par le schéma et le code du dépôt.

## 3. Architecture actuelle

### Élèves

`StudentPageData` expose des unités typées : `StudentListData`, `StudentListItem`, `StudentFormData`, `StudentContext` et `StudentClassDeleteData`. Les builders regroupent les lignes du `LEFT JOIN` par élève, placent les classes dans chaque item et calculent les URLs Edit/Delete/Classes côté Go.

### Classes

`ClassCodePageData` expose `ClassCodeListData`, `ClassCodeListItem`, `ClassCodeContext` et `CancelURL`. Chaque item porte ses URLs Edit/Delete. Les formulaires utilisent un contexte `int64`/nom et reviennent à la liste Classes.

### Relations élève/classe

`StudentClassCodePageData` expose `StudentClassListData`, `StudentClassListItem`, `StudentClassContext` et `StudentClassFormData`. Chaque relation porte sa `DeleteURL`, le formulaire porte une `ReturnURL`, et `AllowedDelete` reflète l’aide UX dérivée du nombre de classes.

Les trois PageData ne contiennent ni `ExtraData`, ni `map[string]any`, ni `any`. Aucun type DB, SQLC ou `config.ClassCode` n’est transmis aux templates. Les types DB/config restent uniquement des entrées de builders côté Go. Les IDs des modèles de vue sont des `int64` ; seules les URLs sont des chaînes.

## 4. Parcours utilisateur

### Liste Élèves

La page « Élèves » fournit les actions « Ajouter un élève », « Importer un CSV » et « Gérer les classes ». Le filtre GET conserve `class_filter`. Chaque élève porte les actions « Modifier », « Classes » et « Supprimer ». Les classes multiples sont regroupées en badges et une donnée historique sans classe apparaît comme « Aucune classe ».

### Ajout, édition et suppression d’un élève

Add et CSV redirigent vers un message métier lorsqu’aucune classe n’existe. Les formulaires Annuler reviennent à la liste Élèves. Edit préremplit l’identité. Delete passe par une confirmation POST et affiche le nom complet.

### Classes générales

« Gérer les classes » ouvre « Classes ». La page permet de revenir aux élèves ou d’ajouter une classe. Add/Edit/Delete reviennent à la liste Classes avec `CancelURL`. Les actions ont des libellés explicites et les routes sont fournies par les PageData.

### Classes d’un élève

L’action « Classes » ouvre « Classes de l’élève » avec son contexte. L’ajout utilise `List.AddURL`, puis Annuler/Retour utilise `Form.ReturnURL` vers le même élève. Quand toutes les classes sont déjà associées, aucun formulaire impossible n’est présenté. Le retrait utilise la `DeleteURL` de la relation ; c’est précisément le point problématique, car cette URL cible actuellement une mutation GET directe.

### Suppression des élèves d’une classe

La liste Élèves place cette opération dans « Actions avancées ». Un GET avec `class_code_id` ouvre une confirmation, puis le POST exécute la transaction. Le retour final va vers Élèves.

Hormis la mutation GET de relation, aucune reconstruction d’URL n’a été trouvée dans les templates audités et aucun lien mort manifeste n’a été identifié.

## 5. Données de vue

Constat satisfaisant :

- aucun `ExtraData` dans `StudentPageData`, `ClassCodePageData` ou `StudentClassCodePageData` ;
- aucun `map[string]any` ou champ `any` ;
- aucune slice parallèle de données/actions ;
- chaque élève, classe ou relation porte directement ses URLs ;
- les contextes Add/Edit/Delete sont typés ;
- les collections de formulaires utilisent des options de vue, pas des lignes SQLC ;
- les templates ne font plus d’association par index.

Les tests contiennent des garde-fous statiques et par réflexion contre la réintroduction d’`ExtraData` et des anciennes slices d’actions.

## 6. Invariants métier

### Identité d’un élève

Le schéma impose `NOT NULL`, `length(trim(...)) > 0` pour prénom et nom, ainsi que `UNIQUE(user_id, first_name, last_name)`. Add/Edit appliquent `strings.TrimSpace`. Les formulaires HTML marquent les champs requis, mais le backend et la DB restent l’autorité.

### Création avec première classe

La création manuelle ouvre une transaction, insère l’élève, tente la relation ownership-aware, puis commit. Une classe étrangère ou absente produit zéro relation, une réponse 404 et un rollback : aucun élève orphelin n’est conservé.

L’import CSV suit la même structure transactionnelle pour toutes les lignes. Une erreur ou une relation refusée annule l’import complet.

### Classe

Add/Edit appliquent `TrimSpace`. Le schéma impose un nom non vide et `UNIQUE(name, user_id)`. Les lectures et mutations filtrent par `user_id`. `student_class_codes.class_code_id` et `exams.class_code_id` utilisent `ON DELETE RESTRICT`; une classe contenant encore des élèves ou utilisée par une évaluation ne peut pas être supprimée. Le handler classe cette FK en erreur métier.

### Relation élève/classe

`CreateStudentWithClassCode` vérifie l’existence de l’élève et de la classe avec le même `user_id`. Le trigger `student_classes_owner_insert/update` fournit une défense supplémentaire. La contrainte unique interdit les doublons.

Plusieurs classes sont autorisées. `DeleteStudentClassCodeByStudentID` exige une autre relation au moment du `DELETE`. Le handler prévalide élève, classe et relation, puis reclasse un zéro ligne concurrent. L’interface masque le retrait de la dernière classe, mais la protection réelle est serveur/SQL.

### Suppression des élèves d’une classe

Le wording actuel correspond aux requêtes :

- un élève appartenant uniquement à la classe cible est supprimé de `students` ; ses relations disparaissent par cascade ;
- un élève multi-classe reste dans `students` et seule sa relation à la classe cible est supprimée.

Les deux opérations sont exécutées dans une seule transaction. Une erreur de la deuxième mutation annule la première.

## 7. Atomicité et intégrité

### Satisfaisant

- création élève + première classe : transaction ;
- import CSV complet : transaction ;
- ajout d’une relation : unique statement conditionnel et ownership-aware ;
- retrait d’une relation : `DELETE` conditionnel atomique exigeant une autre classe ;
- suppression des élèves d’une classe : deux mutations dans une transaction ;
- suppression individuelle d’un élève : statement ownership-aware, avec cascade des relations ;
- suppression d’une classe : statement ownership-aware et FK restrictives ;
- `rows == 0` : classé comme 404 sur les mutations unitaires pertinentes ;
- course entre deux retraits : une seule suppression peut aboutir lorsque l’élève part de deux classes.

### Limite historique assumée

Le schéma ne peut pas imposer simplement qu’un parent `students` ait toujours au moins un enfant `student_class_codes`. L’invariant est garanti par les chemins applicatifs de production audités, pas par une contrainte globale sur `students`. Les états historiques sans classe restent visibles et réparables via le module d’ajout.

### Élèves présents dans un historique généré

`student_exam.student_id` référence `students(id)` sans cascade. Cette FK protège l’historique, mais `DeleteStudentHandler` traite actuellement son refus comme une erreur générique 500. La suppression de tous les élèves d’une classe peut également rollback entièrement si un élève mono-classe est référencé. L’intégrité est préservée, mais la classification et l’UX ne sont pas métier.

## 8. Ownership

Les frontières suivantes sont satisfaisantes :

- listes Élèves et Classes filtrées par `user_id` ;
- Get/Edit/Delete d’un élève filtrés par `(id,user_id)` ;
- Get/Edit/Delete d’une classe filtrés par `(id,user_id)` ;
- ajout de relation conditionné à un élève et une classe du même utilisateur ;
- trigger DB empêchant les relations étrangères ;
- liste des classes non associées conditionnée à l’élève possédé ;
- retrait : prévalidation de l’élève, de la classe et de la relation, puis `DELETE` filtré par `user_id` ;
- suppression massive : classe prévalidée, transaction et requêtes filtrées par utilisateur ;
- IDs forgés absents/étrangers classés 404 dans les chemins couverts.

Les tests confirment notamment l’impossibilité de créer un élève avec la classe étrangère sans rollback, et le refus des élèves/classes/relations étrangers lors d’un retrait.

## 9. Cas limites

| Cas | Comportement actuel | Verdict |
|---|---|---|
| Aucun élève | État vide avec Add/CSV, ou prérequis Classes | Satisfaisant |
| Aucune classe | Filtre et suppression massive masqués ; Add/CSV redirigent vers le prérequis | Satisfaisant |
| Élève historique sans classe | Visible comme « Aucune classe » ; page de relations propose Add | Défensif et réparable |
| Élève avec une classe | Retrait masqué ; handler et SQL refusent | Satisfaisant |
| Élève avec plusieurs classes | Retrait disponible et atomique | Satisfaisant hors méthode GET |
| Toutes les classes associées | Formulaire remplacé par un message et retour | Satisfaisant |
| Classe vide | Suppression autorisée si aucune évaluation | Satisfaisant |
| Classe avec élèves | FK restrictive, message métier | Satisfaisant |
| Classe utilisée par Exam | FK restrictive, message métier | Satisfaisant |
| Relation inexistante | 404, aucune mutation | Satisfaisant |
| ID absent/non numérique | 400 ou redirection métier selon écran | Cohérent avec l’existant |
| Ressource étrangère | 404/zero-row, aucune mutation | Satisfaisant |
| CSV vide | `ValidateCSVStructure` retourne une erreur | Satisfaisant |
| CSV mauvais nombre de colonnes / UTF-8 invalide | Refus avant transaction | Satisfaisant |
| CSV supérieur à 2 Mio | Refus par `MaxBytesReader` | Satisfaisant |
| Plus de 10 000 lignes | Refus | Satisfaisant |
| Nom CSV supérieur à 25 runes | Tronqué silencieusement avant stockage | Important |
| Élève référencé par `student_exam` | Suppression refusée par FK mais présentée en 500 | Important |

## 10. Compatibilité avec les anciennes données

Les migrations actuelles conservent les élèves, classes et relations historiques et renforcent l’ownership via des triggers. L’audit ne dispose d’aucune preuve qu’une ancienne donnée réelle précise soit incohérente.

Scénarios théoriques :

- un élève sans classe peut avoir été créé avant que tous les chemins applicatifs soient transactionnels ; il reste visible et peut être rattaché à une classe ;
- les élèves multi-classes sont pris en charge par la liste, le retrait atomique et la suppression par classe ;
- des noms historiques contenant des espaces ou guillemets restent affichables ;
- une relation dont les parents appartiendraient à des utilisateurs différents serait contraire aux triggers actuels, mais aucune affirmation n’est faite sur sa présence dans la base réelle sans inspection.

La base réelle ne doit être validée qu’avec des requêtes de diagnostic anonymisées : élèves sans relation, relations dont les `user_id` divergent, doublons logiques, références historiques `student_exam`, puis `PRAGMA foreign_key_check`. Aucun de ces diagnostics n’a été exécuté ici.

## 11. Cohérence UX

Les trois modules partagent désormais :

- titres simples et sous-titres courts ;
- actions principales en bouton primaire ;
- Annuler/Retour en outline secondaire ;
- confirmations explicites pour les suppressions de ressources ;
- terme « Retirer » pour une relation ;
- états vides pédagogiques ;
- tables responsives, labels liés et boutons textuels ;
- contexte élève visible dans les relations.

La distinction est correcte : « Supprimer l’élève », « Supprimer la classe », « Supprimer les élèves d’une classe » et « Retirer » une relation.

Petites limites : le retrait ne possède pas encore de confirmation dédiée et les formulaires Classes conservent un JavaScript historique incompatible avec le contrat backend.

## 12. JavaScript résiduel

`add_form_class_code.html` et `edit_form_class_code.html` définissent chacun `removeForbiddenCharacters` et l’appellent via `oninput`. La fonction applique :

```javascript
input.value = input.value.replace(/[\"]/g, "");
```

Elle retire silencieusement les guillemets doubles. Elle ne retire pas réellement les guillemets simples malgré les anciens commentaires historiques rencontrés ailleurs.

Le backend ne possède aucune règle équivalente : Add/Edit utilisent `TrimSpace`, et le CHECK DB ne vérifie que le non-vide après trim. Un nom de classe contenant `"` est donc valide côté serveur mais altéré par le navigateur.

Classification : **mineur — incohérence fonctionnelle à corriger**. Le nettoyage recommandé est de supprimer les deux appels et fonctions, avec des tests backend/template confirmant la conservation exacte des guillemets. Ce point est comparable au reliquat déjà supprimé des formulaires Élèves.

## 13. Couverture de tests

### Couverture forte existante

- builders de liste/formulaire/contexte pour les trois modules ;
- regroupement multi-classe et lignes intercalées ;
- élève sans classe et slices vides non nil ;
- URLs portées par les bons items ;
- rendus des templates et garde-fous `ExtraData` ;
- états vides et `AllowedDelete` ;
- création manuelle rollbackée sur classe étrangère ;
- ownership absent/étranger sur Edit/Delete Élève ;
- rollback de la suppression massive en cas d’échec de la seconde mutation ;
- ajout/retrait ownership-aware au niveau DB ;
- dernière classe refusée et multi-classe autorisée ;
- relation/élève/classe absent ou étranger ;
- contraintes de suppression Classe utilisées par les tests génériques ;
- validation structurelle CSV et troncature Unicode.

### Lacunes utiles

- aucun test ne vérifie qu’une panne de `GetStudentsWithClasses` produit un 500 ; le handler ignore actuellement cette erreur ;
- aucune couverture handler complète du CSV : classe étrangère avec rollback, doublon au milieu du fichier, fichier vide/invalide au niveau HTTP ;
- aucun test métier du refus FK lors de la suppression d’un élève présent dans `student_exam`, individuellement ou en masse ;
- aucune course réellement concurrente sur deux retraits, même si le contrat atomique est testé au niveau rows ;
- aucun test de conservation des guillemets pour les noms de classe ;
- aucune couverture de classification des pannes DB non-contrainte sur Add/Edit Élève ou Classe ;
- la méthode GET mutante est testée telle quelle mais aucun garde-fou n’impose une sémantique POST.

## 14. Constats classés par gravité

### Bloquant 1 — retrait d’une relation via GET

- Fichiers : `internal/handlers/studentClassCode/routes.go`, `handlers.go`, `internal/templates/studentClassCodes/table_student_class_codes.html`.
- Observation : un lien GET appelle directement `DeleteStudentClassCodeHandler` et exécute le `DELETE`.
- Impact : mutation déclenchable sans formulaire POST/confirmation ; risque d’activation involontaire ou par mécanisme same-site. `SameSite=Strict` limite le CSRF intersite mais ne corrige pas la sémantique unsafe.
- Recommandation : route POST dédiée pour la mutation, formulaire POST ou page GET de confirmation ; conserver les mêmes contrôles ownership et la garde atomique.
- Priorité : P1, nécessaire avant clôture.

### Important 1 — erreur de liste Élèves ignorée

- Fichier : `internal/handlers/students/handlers.go`.
- Observation : l’erreur de `GetStudentsWithClasses` est capturée par un bloc vide `if err != nil { /* ... */ }`, puis le rendu continue.
- Impact : une panne DB peut être présentée comme une liste vide ou partielle au lieu d’un 500 déterministe.
- Recommandation : journaliser et retourner 500 immédiatement ; ajouter un test de panne déterministe.
- Priorité : P2.

### Important 2 — suppression d’élèves historiques classée en 500

- Fichiers : `students/handlers.go`, migration `0024_create_student_exam.sql`.
- Observation : la FK `student_exam.student_id` protège un élève déjà généré, mais Delete individuel et suppression de classe ne classent pas ce refus comme erreur métier.
- Impact : historique protégé, mais workflow de suppression opaque ; en masse, toute la transaction rollback si un élève mono-classe est référencé.
- Recommandation : classifier la FK connue, expliquer que l’élève/historique empêche la suppression et tester les deux chemins sans affaiblir la FK.
- Priorité : P2.

### Important 3 — erreurs DB Add/Edit trop génériquement classées métier

- Fichiers : handlers `students`, `classCodes`, `studentClassCode`.
- Observation : plusieurs handlers transforment toute erreur DB en message doublon/nom vide ou relation dupliquée.
- Impact : une panne DB réelle est présentée comme une erreur de saisie, ce qui complique diagnostic et support.
- Recommandation : classification structurée des contraintes UNIQUE/CHECK connues ; autres erreurs en 500.
- Priorité : P2.

### Important 4 — import CSV tronque silencieusement les noms

- Fichier : `internal/handlers/tools/checkCSVStructure.go`.
- Observation : chaque champ de plus de 25 runes est tronqué avant stockage ; l’interface n’annonce pas cette transformation et l’ajout manuel ne l’applique pas.
- Impact : perte silencieuse d’information et incohérence entre deux chemins de création.
- Recommandation : définir explicitement le contrat produit (refus contrôlé ou conservation), l’exposer à l’utilisateur et tester le chemin HTTP. Ne pas changer sans décision métier.
- Priorité : P2.

### Mineur 1 — filtrage des guillemets dans Classes

- Fichiers : `classcodes/add_form_class_code.html`, `edit_form_class_code.html`.
- Observation : les guillemets doubles valides backend sont supprimés au fil de la saisie.
- Impact : modification silencieuse d’un nom dans un cas peu fréquent.
- Recommandation : supprimer le JS local et tester les guillemets.
- Priorité : P3.

### Mineur 2 — lectures N+1 des noms de classes d’un élève

- Fichier : `studentClassCode/handlers.go`.
- Observation : une requête récupère les IDs, puis `GetClassCodeNameByID` est appelée pour chaque classe.
- Impact : coût proportionnel au nombre de classes ; généralement faible, sans défaut fonctionnel.
- Recommandation : à l’occasion d’un jalon performance, retourner ID+nom dans une seule requête ownership-aware.
- Priorité : P3.

### Aucun correctif — points satisfaisants

- PageData entièrement typées ;
- ownership handler + SQL + triggers ;
- transactions Add/CSV/suppression par classe ;
- garde atomique de dernière classe ;
- FK Classe/Exam/Relations restrictives ;
- distinction UX Supprimer/Retirer ;
- états historiques défensifs ;
- validations automatiques passantes.

## 15. Correctifs recommandés

Avant clôture :

1. remplacer la mutation GET de retrait par un POST, avec confirmation ou formulaire explicite ;
2. conserver sans modification la condition SQL atomique et les contrôles ownership ;
3. adapter les tests pour interdire définitivement une mutation sur GET.

Jalons P2 recommandés ensuite :

1. traiter l’erreur ignorée de la liste Élèves ;
2. classifier la FK `student_exam` lors des suppressions ;
3. structurer la classification UNIQUE/CHECK/autres erreurs DB ;
4. décider le contrat des noms CSV longs.

Nettoyages P3 : supprimer le JS des guillemets Classes et, si utile, éliminer le N+1 des classes d’un élève.

## 16. Décision de clôture

**Workflow Élèves / Classes non encore clôturable.**

Le modèle de données de vue, l’ownership et les invariants de relation sont solides. Toutefois, une suppression de relation reste exposée en GET et constitue un risque d’intégrité suffisant pour empêcher raisonnablement la clôture. Le correctif est petit et ciblé : POST explicite, sans changement SQL ni remise en cause de la protection atomique.

## 17. Suite recommandée après la section Élèves

Après correction et mini-audit final du retrait POST, traiter les dettes P2 dans des jalons courts, puis clôturer Élèves / Classes. Le prochain grand bloc fonctionnel pourra ensuite être choisi sans rouvrir l’architecture de vue de cette section.

## Validations exécutées

- `go test ./internal/handlers/students ./internal/handlers/classCodes ./internal/handlers/studentClassCode ./internal/db` : réussi ;
- `go test ./...` : réussi ;
- `git diff --check` : réussi.

Aucun code, SQL, template, test ou migration n’a été modifié par cet audit. Seul ce rapport a été créé.
