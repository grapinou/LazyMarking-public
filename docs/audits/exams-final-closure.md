# Mini audit de clôture final Exams

Date : 30 août 2026

## P1 PDF clos

**Oui.**

Le chemin `success` actuel suit cet ordre :

1. `GetExamStatus(generation_id, user_id)` prouve l'ownership de la génération ;
2. seul `status = success` entre dans le chemin de succès ;
3. `GetExamNameAndClassCodeName` revérifie génération, Exam, classe et parents avec le même `user_id` ;
4. seulement ensuite `ResolveExamGenerationPDFName(username, generation_id)` inspecte le workspace utilisateur `exam-<generation_id>` ;
5. `CopiesURL` est construite avec le nom réellement trouvé.

La résolution technique ne reçoit ni `ExamName` ni `ClassName`. Ces valeurs restent uniquement des métadonnées d'affichage. Le test fonctionnel `TestSuccessKeepsHistoricalPDFAccessAfterClassRename` place un PDF portant l'ancien nom, charge success, renomme la classe, recharge success, puis vérifie que la même URL historique sert exactement le même contenu et qu'aucun second PDF n'a été créé.

Le resolver :

- reste confiné sous `assets/tmp/<username>/exam-<generation_id>` ;
- valide l'arbre du workspace ;
- refuse/ignore les symlinks et les entrées non régulières ;
- ne considère que les extensions PDF ;
- refuse explicitement plusieurs PDF candidats ;
- retourne une erreur si aucun PDF n'existe.

Le handler transforme l'absence ou l'ambiguïté en 404. Le handler de service conserve ensuite ses validations de composants, son refus des symlinks, son contrôle de fichier régulier et `os.SameFile`.

Conclusion : le précédent P1 est clos.

## P0

**0.** Aucun défaut critique restant identifié dans le bloc Exams.

## P1

**0.** Aucun défaut majeur restant identifié après stabilisation de l'accès au PDF historique.

## P2

### Durabilité conjointe DB et workspaces success

Le PDF success est un artefact durable sur filesystem et non un blob SQLite. Ce choix n'est pas en lui-même un défaut applicatif empêchant la clôture : le contrat cohérent est que la base SQLite **et** les workspaces `success` constituent ensemble l'état durable de LazyMarking.

Politique minimale recommandée :

- sauvegarder la DB SQLite et les workspaces `assets/tmp/<username>/exam-<generation_id>` success dans la même procédure cohérente ;
- les restaurer ensemble ;
- conserver leurs propriétaires et permissions ;
- contrôler après restauration que chaque génération success possède exactement un PDF final régulier ;
- documenter la rétention et tester périodiquement une restauration.

Une ligne success dont le fichier a disparu retourne volontairement 404 ; aucune URL fictive ou régénération silencieuse n'est tentée. Cette dette P2 est une exigence d'exploitation/documentation acceptable pour un verdict A.

## P3

### Workspace orphelin après cleanup

Le cleanup failed/running supprime d'abord la ligne DB, puis le workspace. Si la suppression filesystem échoue après le DELETE réussi :

- l'Exam est débloqué ;
- les descendants failed/running ont été supprimés ;
- aucun historique success n'est touché ;
- seul un dossier orphelin consomme de l'espace disque.

La recovery ne peut plus retrouver ce dossier via une ligne DB absente. Une réconciliation filesystem prudente reste une amélioration P3, non bloquante.

### `DeleteExamGenerated` générique

La requête `DeleteExamGenerated(id, user_id)` sans filtre de statut existe toujours. La recherche de tous ses appels ne trouve que :

- la méthode sqlc générée ;
- quatre usages dans `internal/db/pipelineMutationRows_test.go`.

Aucun handler, worker, cleanup ou recovery de production ne l'appelle. Les workflows réels utilisent `DeleteRunningExamGenerated` ou `DeleteFailedExamGenerated`, chacune contrainte par statut. Cette primitive reste un cleanup technique P3 : sa présence augmente le risque d'un mauvais usage futur, mais n'empêche pas la clôture actuelle.

### Reproductibilité byte-for-byte

Les copies existantes restent corrigeables grâce à `student_exam`, `student_exam_content` et `student_exam_page_content`, qui conservent les choix de contenu, ordre, formulations, réponses et géométries/pages. Le PDF historique success est désormais retrouvé indépendamment des libellés vivants.

Aucune seed aléatoire n'est persistée ; régénérer byte-for-byte les mêmes copies depuis les sources vivantes n'est donc pas garanti. Sans besoin métier explicite de régénération exacte, cette limitation P3 est acceptable et ne bloque pas Exams.

## Frontière Marking

La frontière observable est correcte. `AddPdfFormMarkingHandler` appelle `GetExamsGeneratedSuccess(user_id)`. La requête impose :

- `exams_generated.status = 'success'` ;
- ownership de la génération ;
- ownership de l'Exam ;
- ownership de la classe.

Correction ne propose donc ni running ni failed. L'intérieur de Marking n'a pas été audité ici.

## Tests et CI

Le dernier correctif est couvert par :

- renommage de classe après génération success ;
- fichier historique portant l'ancien schéma de nom ;
- vérification du même contenu et de l'absence de second artefact ;
- ambiguïté de deux PDF refusée ;
- workspace étranger non résolu ;
- username invalide refusé ;
- symlink, répertoire `.pdf` et non-PDF ignorés/refusés ;
- PDF/workspace absent ;
- protections existantes du handler de service : traversal, symlinks de parents, fichier régulier et contrôle `os.SameFile`.

Validations exécutées sur l'état actuel :

- `./scripts/check.sh` : succès ;
- `go test -race ./...` : succès.

## Verdict

**A — Exams clos.**

Aucun P0 ou P1 ne subsiste. Les dettes conservées sont exactement :

- P2 exploitation : sauvegarde/restauration conjointe DB + workspaces success ;
- P3 : possible workspace orphelin après échec filesystem post-DELETE ;
- P3 : primitive générique `DeleteExamGenerated` inutilisée en production ;
- P3 : absence de reproductibilité byte-for-byte sans besoin métier confirmé.

Ces points ne remettent pas en cause la protection de l'historique, l'immutabilité, le lifecycle failed, l'ownership, l'accès stable au PDF ou la capacité de Correction à sélectionner uniquement des générations success.

## Prochain bloc

Passage recommandé à l'audit **Correction / Marking**.

## Modification effectuée par cet audit

Création de `docs/audits/exams-final-closure.md` uniquement. Aucun fichier de code, SQL, migration ou template n'a été modifié et aucun commit n'a été créé.
