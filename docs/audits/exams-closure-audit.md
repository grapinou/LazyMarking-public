# Audit de clôture Exams / Évaluations

Date : 30 août 2026

## Périmètre et méthode

Cet audit est un contrôle en lecture seule du code, des requêtes SQL, des migrations finales, des templates, des tests et de la CI. Les rapports antérieurs ont servi d'index, mais les conclusions ci-dessous ont été vérifiées contre l'état courant du dépôt.

Aucun code, SQL, migration ou template n'a été modifié. Le présent rapport est le seul fichier créé.

Validations exécutées :

- `./scripts/check.sh` : succès (`go mod verify`, contrôle `gofmt`, `go vet ./...`, `go test ./...`, `go build ./...`, `git diff --check`) ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès avant création du rapport.

## 1. Modèle métier final

Une évaluation est la ligne `exams` appartenant à un utilisateur et associant exactement :

- un nom ;
- un QCM (`qcm_id`) ;
- une classe (`class_code_id`) ;
- une année (`year_id`) ;
- une période (`period_id`).

Les cinq parents et l'Exam doivent appartenir au même utilisateur. Le schéma et les requêtes applicatives contrôlent cette cohérence ; les triggers d'ownership constituent une défense DB supplémentaire.

Le graphe historique est :

```text
exams
└── exams_generated (au plus une ligne par exam_id/user_id)
    └── student_exam
        ├── student_exam_content
        └── student_exam_page_content
```

Avant génération, `exams` est une configuration vivante : les cinq valeurs de l'Exam sont modifiables et l'Exam est supprimable. La classe et le QCM sont relus lors du lancement ; leurs élèves et contenus ne sont donc pas encore figés.

Pendant `running`, la ligne `exams_generated`, son compteur et le workspace `exam-<generation_id>` sont temporaires. L'Exam est déjà verrouillé afin d'empêcher une divergence entre le travail en cours et sa configuration.

Après `success`, l'Exam et ses identifiants de parents sont figés. `exams_generated`, `student_exam`, les contenus JSON, les géométries/pages et le PDF conservé dans le workspace constituent l'historique généré.

`failed` est un état de transition interne : ses descendants éventuels et sa ligne de génération sont supprimés, puis le workspace est retiré. L'Exam redevient alors un brouillon.

Nuance importante : l'immuabilité porte sur la ligne Exam et ses FK, pas sur tous les attributs des parents. Les noms de classe, année, période et QCM restent modifiables dans leurs domaines respectifs. Le contenu concret de chaque copie est toutefois stocké dans `student_exam_content`.

## 2. Brouillon, running, success et failed

Le contrat est cohérent dans les handlers, le SQL et la liste :

| État | Modification | Suppression | Génération | Liste |
|---|---:|---:|---:|---|
| aucune génération | oui | oui | oui | Générer, Mini test, Modifier, Supprimer |
| `running` | non | non | non | Voir la progression |
| `success` | non | non | non | Voir les copies |
| `failed` résiduel | non tant que la ligne existe | non tant que la ligne existe | non | aucune action dangereuse |
| après cleanup failed | oui | oui | oui | brouillon |

La pré-vérification des handlers améliore le message utilisateur. La protection réelle ne dépend pas seulement d'elle : `UpdateExam` contient un `NOT EXISTS` atomique et la FK `exams_generated.exam_id -> exams.id ON DELETE RESTRICT` arbitre la course de suppression.

Aucune exception de mutation de l'Exam lui-même n'a été trouvée. Les seuls appels applicatifs à `CreateExam`, `UpdateExam` et `DeleteExam` sont dans le CRUD Exams.

## 3. Suppression et conservation de l'historique

La migration finale reconstruit `exams_generated` avec une FK vers `exams` en `ON DELETE RESTRICT`. L'ancien comportement en cascade n'est donc plus le schéma final.

Le POST Delete :

1. charge l'Exam avec `id + user_id` et ownership des parents ;
2. refuse une génération déjà visible avec un 303 métier ;
3. exécute `DeleteExam` avec `id + user_id` ;
4. classe une violation FK due à une génération concurrente en refus métier ;
5. réserve le 500 aux autres erreurs DB.

Les tests couvrent l'Exam libre, l'Exam généré, l'ownership, la course entre pré-vérification et DELETE, et la conservation des descendants. Supprimer un Exam ne peut plus supprimer une génération `success` réelle.

## 4. Immutabilité de l'Exam

`UpdateExam` modifie en une seule requête `name`, `qcm_id`, `class_code_id`, `period_id` et `year_id`. Sa clause WHERE exige :

- l'ID et le propriétaire de l'Exam ;
- l'ownership des quatre nouveaux parents ;
- l'absence atomique d'une génération pour cet Exam et cet utilisateur.

Le handler conserve la pré-vérification métier puis reclasse zéro ligne en immutabilité si une génération est apparue pendant la course. Aucun autre UPDATE de `exams` n'a été trouvé.

L'historique de l'association est donc sûr. En revanche, les objets référencés restent des entités vivantes : renommer une classe ne change pas `class_code_id`, mais change le libellé lu ultérieurement. Cette distinction explique le problème PDF classé ci-dessous.

## 5. Contrat du nom

Create et Edit appliquent `strings.TrimSpace` avant leur mutation. Un résultat vide produit une erreur métier et aucune requête de création/mise à jour. Apostrophes et guillemets sont transmis sans filtrage supplémentaire.

La contrainte `CHECK (length(trim(name)) > 0)` reste une défense DB. `UNIQUE(name, qcm_id, class_code_id, user_id)` est inchangée ; le trim applicatif fait donc naturellement collisionner `"Contrôle"` et `"  Contrôle  "` pour le même triplet QCM/classe/utilisateur. Les erreurs SQLite UNIQUE sont classées structurellement ; une autre panne DB retourne 500.

Aucun JavaScript de filtrage de caractères n'est présent dans les templates Create/Edit Exams. Les noms historiques ne sont pas normalisés silencieusement et le GET Edit affiche la valeur stockée.

## 6. Préflight de génération

La génération complète charge l'Exam possédé, puis :

1. charge les élèves de la classe et refuse une classe vide ;
2. charge les IDs de questions du QCM et refuse un QCM vide ;
3. seulement ensuite crée `exams_generated`, le workspace et le travail asynchrone.

Les tests instrumentent l'absence de ligne, de workspace et de worker pour les deux refus.

La mini génération applique également les prévalidations classe et QCM avant son traitement. Elle ne crée pas de ligne `exams_generated` et utilise son workflow temporaire propre. Aucune règle de génération n'a été déplacée dans le CRUD.

## 7. Individualisation à trois niveaux

Les trois niveaux sont toujours présents dans le chemin réel :

1. `GenerateExamsHandler` appelle `BuildQcmStudentCtx` pour chaque élève ;
2. `BuildQcmStudentCtx` appelle `GetQCMQuestionsAnswersCtx` ; celui-ci lit l'ordre de référence, mélange les IDs de familles/questions avec `shuffleQCMQuestionIDs`/`ShuffleSlice`, puis reconstruit le résultat dans l'ordre de la permutation choisie malgré le travail concurrent ;
3. `BuildQuestionCtx` appelle `GetRandomQuestionByQuestionID`, dont le pool contient la question principale et ses variantes, avec choix `ORDER BY RANDOM() LIMIT 1` ;
4. `GetQuestionAnswerCtx` ou `GetAltQuestionAltAnswerCtx` mélange ensuite les réponses avec `ShuffleSlice`.

Les tests vérifient la conservation de la permutation choisie malgré des fins concurrentes inversées, le mode de référence sans mélange, les variantes et le mélange des réponses. Les refactors de view-data et de templates n'ont pas touché ce chemin.

## 8. Snapshot et reproductibilité

Au moment de la génération sont persistés :

- l'identité de la ligne élève via `student_exam.student_id` ;
- l'élève et les libellés utiles inclus dans le JSON QCM ;
- les questions/familles effectivement choisies et leur ordre ;
- la formulation principale ou variante effectivement choisie ;
- les réponses et leur ordre/état ;
- les paramètres de copie, pagination et nombre de pages ;
- les géométries de questions/réponses et les contenus page par page ;
- le PDF final comme artefact du workspace.

Restent vivants : les tables sources QCM/questions/réponses, les attributs des parents (notamment le nom de classe), les données élève en table, les assets externes éventuels et le filesystem contenant le PDF. L'Exam empêche le changement des associations mais ne snapshotte ni ne interdit l'édition de toutes les entités parentes.

Verdict : reproductibilité **partielle**. L'historique contient suffisamment de contenu et de géométrie pour relire/corriger les copies déjà produites, mais il n'existe ni graine aléatoire permettant de régénérer à l'identique ni stockage durable DB du PDF. Cette limite n'impose pas à elle seule une nouvelle architecture tant que le besoin est de conserver et corriger les artefacts produits plutôt que de les régénérer.

## 9. Lifecycle failed et recovery

Le worker possède un cleanup différé en cas de non-succès. Le chemin sûr est :

```text
running -> FailExamGeneration (failed) -> DeleteFailedExamGenerated -> cascade descendants -> suppression workspace
```

Les mutations `FailExamGeneration`, `CompleteExamGeneration`, l'incrément de progression et les suppressions de recovery sont contraintes par propriétaire et état attendu.

Le polling GET ne supprime ni DB ni fichiers. Une ligne `failed` redirige vers le message métier ; si le cleanup a déjà supprimé la ligne, les paramètres `exam_id` et `generation_started` permettent de reconnaître ce cas après vérification de l'Exam possédé. Fermer la page ne stoppe pas le job serveur. Après cleanup, l'unicité est libérée et Edit/Delete/retry redeviennent possibles.

Au démarrage :

- `running` est nettoyé avec une suppression limitée à `running` ;
- `failed` passe par le cleanup limité à `failed` ;
- `success` n'est ni listé ni supprimé.

Les descendants partiels sont supprimés par les cascades sous `exams_generated`. Le workspace est supprimé ensuite.

## 10. Protection de success et primitives de suppression

Les workflows automatiques appellent uniquement :

- `DeleteRunningExamGenerated`, limité à `status = 'running'` ;
- `DeleteFailedExamGenerated`, limité à `status = 'failed'`.

Le helper de cleanup refuse explicitement une ligne qui n'est pas `failed`. La recovery ne sélectionne que `running` et `failed`. Une ligne `success` est donc protégée contre les workflows automatiques actuels.

Il reste une requête générée `DeleteExamGenerated(id, user_id)` sans condition de statut. Aucun appel de production n'a été trouvé ; seuls des tests génériques de nombre de lignes l'utilisent. Elle n'est donc pas une vulnérabilité active, mais constitue une primitive dangereuse latente à supprimer ou rendre privée lors d'un petit durcissement ultérieur.

## 11. Classes, années et périodes

Les FK de `exams` vers `class_codes`, `years` et `periods` sont `ON DELETE RESTRICT`. Leurs handlers Delete utilisent ID + propriétaire, distinguent :

- succès : 303 vers la liste ;
- zéro ligne/parent étranger : 404 ;
- FK connue : 303 avec message métier ;
- autre erreur DB : 500.

Les tests dédiés vérifient ces classifications et la conservation de l'Exam après le refus. L'édition des libellés de ces référentiels reste autorisée ; ce n'est pas un contournement de l'UPDATE Exam, mais cela signifie que leurs noms ne sont pas des métadonnées historiques figées.

## 12. Ownership de bout en bout

- **Create** : INSERT conditionné par l'ownership des quatre parents.
- **List** : Exam et quatre parents vérifiés ; LEFT JOIN génération sur `exam_id` **et** `user_id`.
- **Edit/Delete** : chargement et mutation par `id + user_id`, ownership des parents, contraintes atomiques.
- **Generate/mini** : Exam, classe, élèves et QCM sont lus dans le périmètre utilisateur ; le trigger DB protège aussi la génération.
- **Progression** : statut et compteurs utilisent `generation_id + user_id`.
- **Success** : le contexte joint génération, Exam et classe avec le même `user_id`, plus existence possédée de QCM/période/année.
- **Failed nettoyé** : le fallback utilise `exam_id + user_id`.
- **Cleanup/recovery** : les listes globales sont internes au démarrage, récupèrent le propriétaire enregistré puis exécutent des mutations `id + user_id + status` et un workspace sous son username.
- **Marking boundary** : `GetExamsGeneratedSuccess` filtre explicitement `status = 'success'` et impose l'ownership de génération, Exam et classe.

Exception de conception : le handler de téléchargement PDF ne recharge pas la génération en DB. Il authentifie l'utilisateur puis confine le fichier au répertoire de son `username`, avec composants de chemin validés. Il ne permet donc pas un accès inter-utilisateur, mais ne prouve pas que `operation=exam-N` correspond encore à une génération `success`. Les URLs normales sont, elles, produites après le contrôle DB de success.

Conclusion ownership : correct sur les frontières utilisateurs ; aucune exposition par `generation_id` seul n'a été trouvée.

## 13. Liste Exams

`GetExamsAllInfos` est une requête jointe unique : Exam, QCM, classe, période, année, puis LEFT JOIN de la génération avec `exam_id` et `user_id`. L'unicité `(exam_id, user_id)` garantit au plus une ligne de liste par Exam. Aucun lookup par item n'est effectué.

Le view-model regroupe données et URLs dans `ExamListItem`. Les actions sont cohérentes :

- draft : Générer les copies, Mini test, Modifier, Supprimer ;
- running : Voir la progression ;
- success : Voir les copies ;
- failed résiduel : état affiché sans action dangereuse.

## 14. Formulaires CRUD

Create/Edit/Delete utilisent `ExamPageData`, `ExamListItem`, `ExamContext` et `ExamFormData` sans `ExtraData`. Les IDs sont des `int64`. Les collections QCM/classes/années/périodes sont chargées par quatre requêtes fixes ownership-aware ; Edit reporte exactement les sélections courantes. `CancelURL` ramène à la liste Exams.

Les templates sont en français, présentent les cinq dimensions métier sans ID technique, traitent les collections vides et ne contiennent plus l'ancien JavaScript supprimant guillemets/apostrophes. L'Exam généré reste bloqué avant rendu Edit et au POST Delete.

## 15. UX progression et succès

La page running consomme `Context` et `Progress`, affiche `processed/total`, calcule un pourcentage borné avec protection de `total <= 0`, et conserve le meta-refresh de deux secondes vers `ProgressURL`. Elle ne propose que le retour à la liste, aucune mutation impossible.

La page success consomme `Context` et `Success`, affiche `ExamName` et `ClassName`, utilise directement `CopiesURL` et `ExamsURL`, et ne montre aucun ID. Il n'existe plus de popup automatique ni de JavaScript sur ces deux pages ; le polling repose uniquement sur le meta-refresh.

## 16. ExtraData

`ExamPageData` et `GenerateExamPageData` ne possèdent plus de champ `ExtraData`. Aucun usage fonctionnel d'`ExtraData` n'a été trouvé dans les handlers/templates Exams ou generateExam. Les usages encore présents appartiennent à d'autres domaines, notamment Marking, et sont hors périmètre.

## 17. N+1 et performances

Aucun N+1 problématique n'a été trouvé dans Exams :

- liste : une jointure ;
- formulaires : quatre collections fixes, indépendantes du nombre d'Exams ;
- progression : statut puis compteurs, nombre constant par poll ;
- succès : statut puis contexte, nombre constant ;
- génération : lectures par élève/question/réponse nécessaires à la production individualisée et déjà orchestrées concurremment ; elles ne remplacent pas une liste qui pourrait être jointe simplement.

La progression pourrait un jour être ramenée à une requête, mais le gain serait mineur et ce n'est pas un N+1.

## 18. Cycle des workspaces

- création : après création de la ligne `exams_generated` et après préflight ;
- success : nettoyage des intermédiaires, conservation du PDF final dans `exam-<generation_id>` ;
- failed : suppression DB limitée à failed, cascade descendants, puis suppression du workspace ;
- restart : running et failed sont repris, success est conservé ;
- Exam draft supprimé : aucune génération/workspace durable ne doit exister.

Dette connue : si le DELETE DB du failed/running réussit puis que la suppression filesystem échoue, le dossier devient orphelin. Comme la ligne n'existe plus, les listes de recovery ne le retrouveront pas automatiquement. L'effet est une fuite disque, pas une perte d'historique `success` ni un blocage de l'Exam. Priorité **P3**, acceptable pour clôturer le cœur métier ; une réconciliation bornée des seuls répertoires `exam-<id>` sans ligne DB serait un correctif ultérieur raisonnable.

Un échec de suppression pendant recovery interrompt aussi la passe et peut retarder le nettoyage des éléments suivants jusqu'au prochain démarrage. Là encore, success reste protégé.

## 19. PDF success et chemin

La sécurité du chemin est solide :

- nom final passé par `safeExamFilenamePart` (séparateurs et contrôles neutralisés, `.`/`..` refusés comme composants spéciaux) ;
- opération dérivée de l'ID entier : `exam-<generation_id>` ;
- serveur PDF limité au répertoire de l'utilisateur authentifié ;
- validation des composants, extension `.pdf`, arbre sans symlink, fichier régulier et contrôle `os.SameFile` après ouverture ;
- fichier absent : 404, sans fuite de chemin.

En revanche, l'accès historique n'est pas stable : le fichier est créé avec `examGenerationPDFName(username, exam.Name, classCodeName)`, puis l'URL success reconstruit ce nom à partir de `GetExamNameAndClassCodeName`. Le nom Exam est immuable après génération, mais `UpdateClassCode` permet encore de renommer la classe référencée. Après un renommage, l'URL vise un nouveau nom inexistant et renvoie 404 alors que l'ancien PDF est toujours présent.

Ce point est classé **P1** : il casse une action centrale sur un historique success par une opération métier autorisée. Le correctif recommandé est ciblé : rendre l'identité de l'artefact indépendante des libellés vivants (par exemple un nom stable basé sur la génération, ou persister le nom final déjà produit), puis ajouter un test success après renommage de classe. Il n'est pas nécessaire d'introduire un snapshot général du domaine.

Le stockage du PDF reste par ailleurs local au filesystem. Une perte/déplacement manuel du workspace laisse la ligne `success` mais produit un 404. C'est une contrainte d'exploitation à inclure dans sauvegarde/restauration ; priorité **P2** tant qu'aucune politique de durabilité des artefacts n'est formalisée.

## 20. Frontière vers Marking

Sans auditer l'intérieur de Marking, la frontière observable est correcte : la liste d'entrée `GetExamsGeneratedSuccess` sélectionne uniquement les générations `success` appartenant à l'utilisateur, jointes à son Exam et sa classe. Une ligne running/failed n'est donc pas proposée à Correction.

Le contenu historique et les géométries sont rattachés aux `student_exam` descendants de la génération. Le cleanup failed les supprime en cascade ; ils ne doivent plus être consommables après cleanup. Aucune incohérence Exams -> Marking bloquante supplémentaire n'a été observée.

## 21. Couverture de tests à forte valeur

La suite couvre notamment :

- ownership DB et handlers ;
- intégrité des parents Create/Update ;
- CRUD, TrimSpace, blanc, caractères autorisés, UNIQUE et vraies erreurs DB ;
- suppression historique, FK RESTRICT et courses ;
- immutabilité et course génération/UPDATE ;
- préflight QCM/classe complet et mini, sans ligne/workspace/worker ;
- lifecycle failed, cleanup idempotent, descendants, workspace et polling nettoyé ;
- recovery running/failed et conservation success ;
- ordre individualisé, variantes et réponses ;
- liste multi-états, URLs et liste vide ;
- view-data des formulaires ;
- progression/success et URLs ;
- noms de PDF, path traversal, symlinks, fichiers absents et TOCTOU.

Trou principal à ajouter avant clôture définitive : un test fonctionnel qui génère/représente un success, renomme la classe, puis vérifie que le lien vers l'artefact reste valide. Ce test échouerait avec l'implémentation actuelle et matérialise le P1.

Tests utiles mais non bloquants : réconciliation d'un workspace orphelin après suppression DB réussie et suppression filesystem échouée, si cette fonctionnalité P3 est ajoutée ; test de restauration d'un success avec PDF manquant selon la future politique d'exploitation.

## 22. CI

Le dépôt possède `scripts/check.sh` et `.github/workflows/ci.yml`. Le job principal exécute le contrat local complet avec la version de Go tirée de `go.mod`, cache Go, `CGO_ENABLED=1`, permissions repository en lecture seule et timeout. Un job séparé exécute `go test -race ./...`.

Les packages Exams, generateExams, outils de lifecycle, DB et templates/data sont inclus par `./...`. Les validations locales de cet audit, y compris race, sont passantes.

## 23. Checklist pour l'ancienne base réelle

À exécuter ultérieurement sur une copie isolée et anonymisée, jamais dans la CI publique :

- vérifier la version de migration et `PRAGMA foreign_key_check` ;
- inventorier les Exams, y compris noms avec espaces historiques et collisions potentielles après trim ;
- vérifier que chaque Exam pointe vers un QCM, une classe, une année, une période et un utilisateur existants/cohérents ;
- inventorier les générations par statut et contrôler l'unicité `(exam_id, user_id)` ;
- confirmer qu'aucun `failed` durable ni `running` ancien ne subsiste après recovery contrôlée ;
- compter `student_exam` par génération et comparer à `total_students`/`processed_students` pour les success ;
- valider la décodabilité des JSON `student_exam_content` et `student_exam_page_content` sans exposer leur contenu ;
- vérifier les FK et le chaînage des contenus/pages vers le bon utilisateur ;
- faire correspondre chaque success à son workspace `exam-<generation_id>` et à un PDF régulier non symlink ;
- tester l'ouverture d'un petit échantillon synthétiquement identifié, sans copier ni committer de données élèves ;
- rechercher les workspaces `exam-*` sans ligne DB correspondante ;
- sauvegarder DB et workspaces ensemble avant toute opération réelle ;
- tester explicitement le cas d'une classe renommée depuis une génération success et constater/résoudre le lien PDF avant déploiement.

## 24. Dette restante classée

### P0

Aucune.

### P1

1. **Lien PDF success dépendant du nom vivant de la classe.**
   - Impact : un renommage autorisé rend les copies historiques inaccessibles depuis l'interface (404).
   - Preuve : génération et téléchargement appellent séparément `examGenerationPDFName`; le second utilise `GetExamNameAndClassCodeName`, tandis que `UpdateClassCode` reste autorisé.
   - Recommandation : identité stable de l'artefact, indépendante des libellés vivants, et test de renommage.
   - Taille : petite à moyenne selon choix de compatibilité avec les fichiers historiques.

### P2

1. **Durabilité du PDF uniquement filesystem non contractualisée.**
   - Impact : une ligne success peut survivre à la disparition de son PDF et l'UI renvoie alors 404.
   - Preuve : le PDF est servi depuis `assets/tmp/<username>/exam-<id>` et n'est pas stocké en DB ; aucun mécanisme de restauration n'est visible.
   - Recommandation : formaliser sauvegarde/rétention/restauration des workspaces success, sans forcément changer le modèle DB.
   - Taille : moyenne, principalement exploitation.

### P3

1. **Workspace orphelin après DELETE DB réussi puis erreur filesystem.**
   - Impact : fuite disque bornée par incident, sans perte de success ni verrou métier.
   - Preuve : suppression DB précède `RemoveOperationTempDir`; la recovery ne liste plus une génération absente.
   - Recommandation : réconciliation périodique prudente des workspaces sans génération.
   - Taille : petite à moyenne.

2. **Primitive générique `DeleteExamGenerated` sans filtre de statut.**
   - Impact : risque futur si elle était appelée par erreur sur success ; aucun appel de production actuel.
   - Preuve : requête sqlc présente, usages limités aux tests génériques.
   - Recommandation : supprimer la requête ou la remplacer partout par les primitives terminales typées.
   - Taille : petite.

3. **Reproductibilité exacte non garantie.**
   - Impact : impossible de régénérer byte-for-byte les mêmes copies depuis les seules sources vivantes ; l'historique déjà produit reste exploitable via JSON/PDF.
   - Preuve : choix aléatoires sans graine persistée, parents/assets vivants.
   - Recommandation : ne traiter que si un besoin métier de régénération exacte est confirmé.
   - Taille : grande si exigée, sinon aucune action.

## 25. Verdict

Verdict **B — quelques correctifs encore nécessaires avant clôture**.

Les contrats de sécurité métier centraux sont cohérents : historique non supprimable, Exam atomiquement immuable, failed nettoyé sans toucher success, ownership systématique, préflight sûr, individualisation intacte et Correction alimentée uniquement par success. La suite générale et le race detector passent.

La clôture ne doit toutefois pas déclarer l'accès aux copies totalement stabilisé tant qu'un simple renommage de classe peut casser le lien vers le PDF d'une génération success. Ce correctif ciblé et son test sont recommandés avant de passer pleinement à l'audit Correction/Marking. La durabilité filesystem doit au minimum être explicitée comme exigence d'exploitation.

## 26. Fichiers modifiés par cet audit

- `docs/audits/exams-closure-audit.md` uniquement.

Aucun fichier de code, SQL, migration, template ou CI n'a été modifié et aucun commit n'a été créé.
