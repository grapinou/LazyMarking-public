# Audit final de clôture — workflow Élèves / Classes

## 1. Synthèse exécutive

Le socle Élèves / Classes est désormais largement cohérent : les anciens constats P1/P2/P3 ont été corrigés, les mutations HTTP sont séparées des pages de lecture, l'ownership est vérifié côté SQL et handlers, les opérations multi-écritures sont transactionnelles, les données de vue sont typées et les tests généraux passent.

L'audit transversal final a toutefois découvert trois défauts encore significatifs dans l'import CSV :

- la limite annoncée de 2 Mio est appliquée après un premier parsing multipart implicite et n'est donc pas effective ;
- les guillemets littéraux placés aux extrémités d'un prénom ou d'un nom sont encore supprimés silencieusement ;
- une erreur DB inattendue pendant la création d'un élève importé est encore présentée comme un doublon.

Ces constats ne remettent pas en cause les corrections précédentes, mais empêchent de déclarer le workflow clos.

## 2. Verdict

**NON CLÔTURABLE**

Décompte final :

- P1 : 1 ;
- P2 : 2 ;
- P3 : 1.

Le P3 est uniquement un renforcement de test de concurrence. Les trois constats CSV doivent être corrigés avant la clôture.

## 3. Tests et vérifications exécutés

Toutes les commandes demandées ont réussi :

| Commande | Résultat |
| --- | --- |
| `go test ./internal/handlers/students` | succès |
| `go test ./internal/handlers/classCodes` | succès |
| `go test ./internal/handlers/studentClassCode` | succès |
| `go test ./internal/handlers/tools` | succès |
| `go test ./internal/db` | succès |
| `go test ./...` | succès |
| `git diff --check` | succès |
| `sqlc generate -f db/sqlc.yaml` | succès, aucune divergence générée |

Les empreintes des fichiers SQLC générés et le statut Git ont été comparés avant et après la génération : aucun fichier généré n'a changé.

## 4. Statut des anciens constats

### Ancien P1 — mutation de relation élève/classe via GET

**Fermé.**

- la route GET appelle `DeleteFormStudentClassCodeHandler` et ne réalise aucune mutation ;
- elle valide l'élève, la classe, la relation et leur ownership, puis rend une confirmation typée ;
- la route POST appelle seule `DeleteStudentClassCodeHandler` ;
- la liste masque l'action lorsque `AllowedDelete` est faux ;
- un GET direct sur la dernière relation ne rend aucun formulaire destructif ;
- un POST forgé reste refusé par le handler et par la requête SQL atomique.

### Ancien P2 — erreurs de la liste Élèves

**Fermé.**

Dans `TableStudentsHandler`, une erreur de `GetStudentsWithClasses` ou de `ListClassCodesByUser` est journalisée avec son contexte, produit HTTP 500 et interrompt immédiatement le handler. Une panne DB ne peut plus être rendue comme une liste vide.

### Ancien P2 — protection de l'historique `student_exam`

**Fermé.**

- la FK `student_exam.student_id -> students.id` reste non destructive ;
- aucun cascade de suppression n'a été ajouté ;
- une suppression individuelle protégée est classée via le code étendu SQLite `SQLITE_CONSTRAINT_FOREIGNKEY` et produit une erreur métier ;
- la suppression des élèves d'une classe est transactionnelle et effectue un rollback complet si un élève mono-classe est protégé ;
- les erreurs DB inattendues restent des HTTP 500.

### Ancien P2 — classification Add/Edit

**Fermé pour le périmètre initial Add/Edit et ajout de relation.**

- Students Add/Edit : UNIQUE et CHECK sont classées par codes étendus SQLite ; les autres erreurs produisent 500 ; les mutations à zéro ligne conservent leur contrat 404 ;
- Classes Add/Edit : même distinction UNIQUE/CHECK/autre erreur ;
- StudentClassCode Add : UNIQUE produit l'erreur métier de relation déjà présente, zéro ligne conserve les contrôles d'ownership/absence, toute autre erreur produit 500 ;
- aucune de ces classifications ne parse le texte complet d'une erreur SQLite.

Une branche distincte de l'import CSV reste générique ; elle constitue un nouveau P2 détaillé plus bas.

### Ancien P2 — troncature CSV à 25 runes

**Partie identifiée fermée.**

La limite de 25 runes et sa fonction de troncature ont disparu. Les tests couvrent les noms longs et Unicode. Une autre transformation silencieuse, distincte de cette troncature, subsiste cependant et constitue un nouveau P2.

### Ancien P3 — filtrage des guillemets dans Classes

**Fermé.**

Les templates Add/Edit Classes ne contiennent plus `removeForbiddenCharacters` ni d'attribut `oninput` supprimant les guillemets. Le backend applique uniquement `TrimSpace` avant les contraintes de non-vide et d'unicité. L'échappement HTML normal des templates reste actif.

### Ancien P3 — N+1 des classes d'un élève

**Fermé.**

La liste effectue un nombre constant de lectures : contexte élève puis `ListStudentClassCodesWithNames`. Il n'existe plus de boucle appelant `GetClassCodeNameByID`. La requête jointe contrôle l'ownership de l'élève, de la relation et de la classe, et ordonne les résultats par identifiant de relation.

## 5. HTTP et sécurité des mutations

Les mutations auditées sont exposées par POST : création, édition, import CSV, suppressions individuelles ou massives, ajout et retrait de relation. Les GET rendent uniquement listes, formulaires et confirmations.

Le parcours de retrait est désormais : liste → confirmation GET → mutation POST. L'annulation et le succès reviennent au contexte du même élève. Les identifiants non numériques produisent une erreur de requête, et les ressources absentes ou étrangères sont traitées en 404 selon les conventions du module.

Aucune autre mutation par GET n'a été trouvée dans le périmètre.

Le contournement de limite multipart décrit en P1 est un problème de disponibilité sur une mutation POST authentifiée, et non une mutation GET.

## 6. Ownership

Les lectures et mutations importantes combinent les contrôles suivants :

- filtres `user_id` dans les requêtes SQL ;
- contrôles de lignes affectées pour les mutations conditionnelles ;
- vérification séparée de l'élève, de la classe et de la relation avant les parcours sensibles ;
- FK et triggers d'ownership en défense en profondeur.

La requête jointe des classes d'un élève exige simultanément l'ownership de l'élève, des relations et des classes. L'ajout et le retrait d'une relation ne permettent pas de croiser deux utilisateurs. Les opérations de lecture, modification et suppression d'un élève ou d'une classe étrangère ne sont pas accessibles via un simple identifiant forgé.

Aucune exception d'ownership exploitable n'a été trouvée.

## 7. Intégrité Élèves

- prénom et nom sont normalisés par `strings.TrimSpace` dans les écritures applicatives ;
- les CHECK DB refusent les valeurs vides après trim ;
- l'unicité actuelle reste appliquée par la base ;
- l'ajout manuel de l'élève et de sa première relation est atomique dans une transaction ;
- toute erreur avant le commit déclenche le rollback ;
- l'édition conserve l'ownership et les contraintes DB ;
- la suppression respecte la FK historique `student_exam`.

Un élève historique sans classe est rendu défensivement : le LEFT JOIN de la liste produit une collection de classes vide, sans panic, et la page des classes de l'élève propose la réparation par ajout d'une classe.

## 8. Intégrité Classes

- les noms Add/Edit sont normalisés par `TrimSpace` ;
- le CHECK non-vide et l'unicité par utilisateur sont conservés ;
- les guillemets sont acceptés et stockés sans filtrage frontend ;
- l'ownership est appliqué aux lectures et mutations ;
- la suppression d'une classe référencée est refusée par les FK et classée comme erreur métier connue ;
- les autres pannes DB restent des erreurs 500.

Le wording de confirmation indique correctement que la suppression n'est possible que lorsque la classe n'est plus utilisée.

## 9. Invariant dernière classe

La protection est présente aux quatre niveaux attendus :

1. la liste n'affiche pas de lien actif lorsque l'élève ne possède qu'une classe ;
2. la confirmation GET directe affiche l'impossibilité sans formulaire destructif ;
3. le handler POST revérifie le contexte ;
4. `DeleteStudentClassCodeByStudentID` n'efface la relation que si une autre relation existe encore au moment de la mutation.

Deux retraits concurrents visant les deux dernières relations ne peuvent donc pas tous deux valider une suppression laissant zéro classe : la condition est évaluée dans chaque mutation SQLite et la sérialisation des écritures rend visible le premier retrait au second.

La suite teste le contrat atomique et les cas une/deux classes. Elle ne contient pas de test lançant deux requêtes concurrentes réelles ; ce renforcement est classé P3 non bloquant.

## 10. Suppression des élèves d'une classe

Le contrat réel est correctement conservé et décrit par le template :

- un élève uniquement rattaché à la classe est supprimé ;
- un élève multi-classe est seulement détaché de cette classe ;
- un élève mono-classe référencé par `student_exam` bloque l'opération ;
- toutes les mutations utilisent la même transaction ;
- une erreur provoque un rollback complet, sans suppression ou détachement partiel ;
- le commit intervient uniquement après toutes les opérations réussies.

## 11. Import CSV

Les protections suivantes sont présentes : structure CSV à deux colonnes, séparateur attendu, UTF-8 valide, champs non vides, nombre maximal de lignes, ownership de la classe, transaction globale, rollback sur erreur et contraintes d'unicité DB. Les noms longs et Unicode ne sont plus tronqués à 25 runes.

Trois problèmes subsistent :

1. la limite de taille est installée trop tard après un parsing multipart implicite ;
2. `strings.Trim(record[i], "\" ")` retire encore des guillemets littéraux valides aux extrémités des données ;
3. toute erreur de `CreateStudentAndReturnID` est encore présentée comme un doublon, y compris une panne DB inattendue.

Ces constats sont détaillés dans la section 18.

## 12. View-data et templates

Les trois modules utilisent des contrats de vue dédiés et typés : `StudentPageData`, `ClassCodePageData` et `StudentClassCodePageData`, avec leurs items, contextes et données de formulaire.

La recherche ciblée n'a trouvé :

- aucun `ExtraData` ;
- aucun `map[string]any` ou champ `any` dans ces PageData ;
- aucune structure SQLC exposée aux templates ;
- aucune slice parallèle séparant les ressources de leurs actions ;
- aucune reconstruction de route à partir d'index dans les templates.

Les IDs restent des `int64` en Go. Les URLs sont construites côté handler/builder et portées par les items ou contextes appropriés. Les états vides et historiques sont explicitement représentés.

## 13. Navigation et UX

Les parcours demandés sont cohérents :

- Élèves → Ajouter / Import CSV / Modifier / Classes / Supprimer / Actions avancées ;
- Élèves → Gérer les classes → Ajouter / Modifier / Supprimer → retour Élèves ;
- Élèves → Classes de l'élève → Ajouter ou Retirer → confirmation → POST → retour au même élève.

Les liens et formulaires utilisent les URLs typées. Les identifiants de l'élève sont conservés dans les retours contextuels. Le wording distingue correctement la suppression d'une ressource du retrait d'une relation. Lorsqu'aucune classe n'est disponible ou que la dernière relation est protégée, aucun formulaire impossible n'est rendu actif.

Aucun lien mort ou retour vers le mauvais élève n'a été identifié dans le code et les tests examinés.

## 14. Gestion des erreurs

Le tableau attendu est respecté sur les parcours Add/Edit, suppressions et relations récemment corrigés :

| Situation | Comportement observé |
| --- | --- |
| validation métier connue | erreur métier |
| UNIQUE connue | erreur métier structurée |
| CHECK connue | erreur métier structurée |
| FK historique connue | erreur métier structurée |
| ressource absente | 404 selon contrat |
| ownership refusé | 404 selon contrat |
| panne DB inattendue | log contextualisé + 500 |

Exception : dans l'import CSV, l'erreur de création de l'élève est encore génériquement transformée en doublon. Aucun bloc d'erreur vide ou ignoré n'a été trouvé dans les autres chemins ciblés.

## 15. Transactions

Les opérations multi-écritures recensées sont :

- ajout manuel d'un élève et de sa première classe ;
- import CSV de plusieurs élèves et relations ;
- suppression/détachement des élèves d'une classe.

Elles utilisent toutes une transaction, des queries liées à cette transaction, un rollback différé, des sorties immédiates sur erreur et un commit final dont l'erreur est traitée. Aucune écriture partielle évidente n'a été trouvée.

La mauvaise classification CSV ne compromet pas le rollback : elle compromet uniquement la nature de la réponse utilisateur et le diagnostic.

## 16. SQL / SQLC

- les requêtes sensibles sont ownership-aware ;
- les mutations conditionnelles utilisent le nombre de lignes affectées lorsque le contrat l'exige ;
- le retrait de relation est atomiquement conditionné à l'existence d'une autre classe ;
- `ListStudentClassCodesWithNames` supprime le N+1 et applique un ordre déterministe ;
- la régénération SQLC ne produit aucune différence, ce qui confirme la cohérence entre les sources SQL et les fichiers générés.

Aucun SQL, schéma ou fichier généré n'a été modifié pendant cet audit.

## 17. Base historique réelle

Plusieurs fichiers SQLite existent localement sous `db/data/`, mais aucun n'est clairement identifié dans la documentation du dépôt comme une copie historique jetable et sûre pour cet audit. Pour éviter toute lecture ambiguë d'une base potentiellement sensible ou de référence, aucun diagnostic de contenu n'a été lancé.

En conséquence, les cas de données anciennes incohérentes restent évalués à partir du schéma, du code et des fixtures synthétiques. Aucune affirmation n'est faite sur le contenu de la base réelle.

## 18. Nouveaux constats

### P1 — limite de taille multipart CSV non effective

**Fichiers concernés :**

- `internal/handlers/students/handlers.go` ;
- `internal/handlers/tools/checkCSVFile.go`.

**Comportement observé :** `AddCSVStudentHandler` appelle `r.FormValue("class_code_id")` avant `CheckCSVFile`. Dans la bibliothèque standard Go, `FormValue` déclenche implicitement `ParseMultipartForm` et ignore son erreur. `CheckCSVFile` installe ensuite `http.MaxBytesReader` et rappelle `ParseMultipartForm`, mais ce second appel ne reparcourt pas la requête lorsque `MultipartForm` est déjà initialisé.

**Risque :** la limite applicative annoncée de 2 Mio peut être contournée par un utilisateur authentifié. Le parseur multipart peut consommer davantage de mémoire et écrire les grandes parties sur disque temporaire. Il s'agit d'un risque de disponibilité et de saturation disque.

**Correction recommandée :** appliquer `MaxBytesReader` et parser le multipart avant tout appel à `FormValue`, puis lire `class_code_id` depuis le formulaire déjà parsé. Ajouter un test multipart dépassant réellement la limite.

### P2 — suppression silencieuse de guillemets littéraux dans les identités CSV

**Fichier concerné :** `internal/handlers/tools/checkCSVStructure.go`.

**Comportement observé :** après décodage par `encoding/csv`, chaque champ passe encore par `strings.Trim(record[i], "\" ")`. Le décodeur CSV ayant déjà retiré les guillemets syntaxiques, cette opération supprime des guillemets qui font partie de la donnée utilisateur lorsqu'ils sont placés au début ou à la fin du prénom/nom.

**Risque :** altération silencieuse et définitive d'une identité valide, en contradiction avec le contrat de conservation complète retenu pour l'import CSV et avec l'ajout manuel, qui applique `TrimSpace` sans retirer les guillemets.

**Correction recommandée :** appliquer uniquement la normalisation commune réellement voulue, typiquement `strings.TrimSpace`, et ajouter des tests avec des guillemets littéraux encodés correctement en CSV.

### P2 — panne DB d'import CSV présentée comme doublon

**Fichier concerné :** `internal/handlers/students/handlers.go`.

**Comportement observé :** dans la boucle d'import, toute erreur de `CreateStudentAndReturnID` est journalisée puis redirigée avec le message métier indiquant que l'élève existe déjà. Les helpers structurés UNIQUE/CHECK ne sont pas utilisés sur ce chemin.

**Risque :** une panne SQLite sans rapport avec la saisie est masquée comme doublon. Le rollback reste correct, mais le diagnostic et le statut de réponse sont incorrects.

**Correction recommandée :** réutiliser les classifications `IsSQLiteUniqueConstraint` et `IsSQLiteCheckConstraint`, conserver le message métier uniquement pour ces contraintes attendues et retourner HTTP 500 pour les autres erreurs.

### P3 — absence de test concurrent réel de la dernière classe

**Fichiers concernés :** tests DB/handler de `studentClassCode`.

**Comportement observé :** les tests couvrent une et plusieurs classes ainsi que la condition SQL atomique, mais ne lancent pas deux retraits concurrents réels.

**Risque :** faible. La requête conditionnelle et la sérialisation des écritures SQLite garantissent structurellement l'invariant ; il s'agit d'un garde-fou supplémentaire contre une future régression, pas d'un défaut fonctionnel observé.

**Correction recommandée :** ajouter ultérieurement un test de concurrence ciblé si l'infrastructure SQLite de test permet un résultat stable.

## 19. Dette résiduelle

Les anciens constats sont fermés. La dette résiduelle exacte est :

- P1 : rendre effective la limite de taille de l'upload CSV ;
- P2 : préserver les guillemets littéraux des identités CSV ;
- P2 : classifier correctement les erreurs DB de création pendant l'import CSV ;
- P3 : compléter facultativement le test de concurrence de retrait des deux dernières classes.

Aucune autre dette importante n'a été identifiée dans le périmètre audité.

## 20. Conclusion de clôture

Le workflow n'est **pas encore clôturable**. Les protections d'intégrité, l'ownership, les mutations HTTP, les transactions, les view-models et les anciens correctifs sont cohérents et testés. En revanche, le contournement de limite d'upload CSV constitue un risque de disponibilité P1, et les deux écarts CSV P2 contreviennent aux contrats de conservation et de classification des erreurs.

Après correction et tests de ces trois points, un contrôle final ciblé du seul chemin d'import CSV devrait suffire pour prononcer la clôture Élèves / Classes. Le P3 de test concurrent peut rester non bloquant.
