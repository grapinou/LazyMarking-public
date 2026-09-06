# Cycle de vie des générations d'évaluation en échec

## Résumé

`exams_generated` possède trois états : `running`, `success`, `failed`. Une erreur de fond passe normalement la ligne à `failed` et supprime immédiatement le workspace, mais conserve la ligne et toutes les copies partielles. Cette ligne n'est supprimée, avec sa descendance, que si l'enseignant revisite la page de progression. Les générations `failed` ne sont pas récupérées au redémarrage.

Depuis les nouveaux contrats Exams, cette dépendance au polling devient bloquante : une ligne `failed` empêche modification, suppression et nouvelle génération. Certains échecs synchrones surviennent même avant que le navigateur reçoive l'URL de progression; l'utilisateur ne dispose alors d'aucun chemin normal de nettoyage. La priorité est désormais **P1**.

## États et acteurs

| État | Création / transition | Lectures | Suppression actuelle |
|---|---|---|---|
| `running` | valeur par défaut de `CreateExamGenerated` | progression, recovery startup, updates de compteur | `DeleteRunningExamGenerated` au startup; `DeleteExamGenerated` lors du polling après passage failed |
| `success` | `CompleteExamGeneration`, uniquement depuis running | page de succès, liste des générations réussies | aucune suppression automatique; historique protégé |
| `failed` | `FailExamGeneration`, uniquement depuis running | page de progression; helpers d'interprétation idempotente | uniquement branche failed de `GetExamProgressPageHandler` |

Le CHECK DB n'autorise aucun autre statut. Les transitions success/failed sont ownership-aware, conditionnées à `status='running'` et contrôlent les lignes affectées. Les helpers interprètent proprement les appels répétés et les contextes annulés.

## Création et déroulement

Le lancement complet suit :

```text
prévalidations Exam/classe/QCM
  -> CreateExamGenerated (running)
  -> CreateOperationTempDir
  -> lecture du nom de classe
  -> goroutine de génération
      -> workers élèves
      -> fusion PDF
      -> nettoyage intermédiaire
      -> CompleteExamGeneration (success)
```

Les erreurs se répartissent ainsi :

- **workspace impossible** : la ligne est passée à failed, aucun workspace n'existe, réponse HTTP 500; aucune URL de progression n'a été donnée;
- **nom de classe illisible après workspace** : failed, workspace explicitement supprimé, HTTP 500 sans URL de progression;
- **worker/panique/annulation** : collecte des erreurs, failed, puis le defer de goroutine supprime le workspace;
- **listing/fusion PDF** : failed, puis defer de cleanup;
- **finalisation success impossible** : tentative failed, puis cleanup du workspace;
- **écriture DB partielle dans un worker** : l'erreur remonte, mais les lignes déjà créées restent sous la génération failed jusqu'à suppression de cette dernière.

Il n'existe pas de transaction globale couvrant tous les élèves. C'est acceptable si le cleanup de la génération est garanti, car les cascades internes suppriment toute la descendance.

## État failed

Après un échec normal de la goroutine :

- workspace : supprimé par le defer (Remove absent est idempotent);
- `exams_generated` : conservée avec `status='failed'`;
- `student_exam` déjà créées : conservées;
- `student_exam_content` et pages déjà écrits : conservés;
- compteurs partiels : conservés;
- possibilité de retry : bloquée par `UNIQUE(exam_id,user_id)`.

Si le processus est tué au mauvais moment, le workspace peut également subsister. Une ligne peut rester `running` si le processus meurt avant la transition failed; ce cas est pris en charge au redémarrage. Une ligne failed déjà écrite avant le crash n'est, elle, pas récupérée.

## Page de progression : consultation destructive

`GetExamProgressPageHandler` lit `GetExamStatus`. Si le statut est failed, un simple GET de polling effectue dans cet ordre :

1. `RemoveOperationTempDir(username, "exam-<id>")`;
2. `DeleteExamGenerated(id,user_id)`;
3. cascade vers `student_exam`, `student_exam_content` et `student_exam_page_content`;
4. redirection 303 vers un message d'erreur.

La consultation HTTP est donc **destructive**. C'est actuellement le seul cleanup des lignes failed. Le workspace est généralement déjà absent, ce que le helper considère comme un succès.

## Retry et nouveaux verrous Exam

Tant que la ligne failed existe :

- Edit Exam : interdit par la prévalidation et l'UPDATE atomique;
- Delete Exam : interdit par la prévalidation et la FK RESTRICT;
- nouvelle génération : refusée par `UNIQUE(exam_id,user_id)`.

Le seul chemin de sortie est de revisiter l'URL de progression exacte. Après ce GET, la ligne et ses descendants disparaissent et l'Exam redevient modifiable, supprimable et générable. Revenir simplement à la liste Exams puis relancer ne nettoie rien; le message « déjà généré » ne donne pas le mécanisme réel de résolution.

Les échecs survenus avant la redirection vers la progression (workspace ou nom de classe) peuvent donc créer un blocage sans chemin UX accessible.

## Redémarrage et recovery

`RecoverRunningExamGenerations`, appelé au démarrage serveur, liste uniquement `status='running'` :

- running : `DeleteRunningExamGenerated`, cascade des copies/contenus, puis suppression du workspace;
- failed : ignoré, ligne et éventuel workspace conservés;
- success : ignoré et conservé, comme attendu pour l'historique réel.

La fonction est idempotente sur une ligne déjà résolue. Son ordre DB puis filesystem peut laisser un workspace orphelin si sa suppression échoue après le DELETE; l'erreur est remontée, mais la DB ne permet plus de retrouver ce workspace via la génération.

## Scénarios de crash réalistes

- **processus tué en running** : ligne, données partielles et workspace possibles; le startup supprime la ligne et sa descendance puis le workspace;
- **failed écrit, processus tué avant le defer** : ligne failed, descendants et workspace possibles; le startup ne touche rien;
- **workspace déjà absent + failed présente** : le polling nettoie correctement, car l'absence du workspace est acceptée;
- **ligne absente + workspace présent** : possible après cleanup DB réussi puis échec filesystem/crash; le recovery DB ne la retrouve pas, donc workspace orphelin;
- **erreur de worker après quelques réussites** : copies complètes et partielles restent jusqu'au polling failed.

## Success et historique

Une génération success constitue un historique réel. Elle reste visible via `GetExamsGeneratedSuccess`, conserve copies et snapshots, bloque correctement Edit/Delete Exam et n'est jamais ciblée par les suppressions conditionnées à running.

La stratégie recommandée pour failed doit utiliser des requêtes strictement conditionnées à `status='failed'` et ne jamais élargir `DeleteExamGenerated` aux success.

## Primitives de cleanup existantes

- `DeleteExamGenerated(id,user_id)` : suppression générale actuellement utilisée par le polling failed;
- `DeleteRunningExamGenerated(id,user_id,status='running')` : recovery startup sûr;
- cascades DB de génération vers copies et contenus;
- `RemoveOperationTempDir`, idempotent si absent;
- `failExamGeneration`, transition idempotente running→failed avec fallback sur contexte indépendant;
- `RecoverRunningExamGenerations`, idempotent pour les running déjà résolues.

Il manque une primitive explicite et ownership-aware `DeleteFailedExamGenerated` (condition `status='failed'`) qui éviterait qu'une évolution appelle accidentellement la suppression générale sur un success.

## Marking

Le workflow normal ne présente jamais le PDF final d'une génération failed; elle n'est donc normalement pas consommable par Marking. Toutefois, les requêtes de correction basées sur `student_exam_id` ne vérifient pas explicitement le statut de la génération parente. Une copie partielle exceptionnellement récupérée hors workflow ne serait pas refusée au niveau DB pour ce seul motif. Ce n'est pas le correctif recommandé ici, mais c'est un garde-fou à considérer dans le chantier Marking.

## UX actuelle après échec

Lorsque l'utilisateur reste sur la page de progression, le meta-refresh finit par observer failed, déclenche le cleanup destructif puis affiche « Erreur lors de la génération du qcm, contacter admin ». Il n'existe aucun bouton Retry/Nettoyer ni retour explicite vers Exams.

Si la page est fermée avant le poll failed, rien n'indique l'état dans la liste Exams et aucun cleanup n'a lieu. Si l'erreur survient avant la redirection initiale, l'utilisateur voit un HTTP 500 et ne possède même pas l'URL de progression.

## Options

### A — cleanup automatique à la finalisation de l'échec

Avantages : libère immédiatement l'Exam et le retry, nettoie les données partielles même si la page est fermée, reste simple avec les cascades existantes. Inconvénient : la ligne failed devient transitoire et le polling peut arriver après sa suppression; il faut traduire proprement cette disparition en message d'échec. Les logs restent la source diagnostique actuelle, car aucun détail d'erreur persistant n'existe.

### B — failed visible jusqu'à Retry/Nettoyer explicite

Avantages : état observable et action utilisateur claire. Inconvénients : nécessite d'exposer statut/actions dans la liste, maintient volontairement l'Exam bloqué et ne résout pas seul les utilisateurs qui abandonnent le workflow. Plus grand jalon.

### C — conservation temporaire puis purge

Ajoute horloge, politique de rétention et tâche de maintenance sans bénéfice décisif pour cette application mono-instance.

## Recommandation

**A — cleanup failed automatique dès que l'erreur est finalisée.** Une tentative de génération échouée n'est pas un historique pédagogique; les snapshots partiels ne doivent pas immobiliser l'Exam. La génération success reste totalement exclue.

Le plus petit jalon recommandé :

1. ajouter `DeleteFailedExamGenerated(id,user_id,status='failed')`;
2. créer une petite primitive idempotente de cleanup failed : transition vers failed, suppression conditionnelle DB avec cascades, puis workspace;
3. l'appeler sur tous les chemins d'échec, y compris avant goroutine;
4. étendre le recovery startup aux failed laissées par un crash;
5. adapter le polling : une génération disparue après un échec connu doit produire un message retry compréhensible plutôt qu'un blocage; ne jamais supprimer success;
6. tester cleanup partiel, retry immédiat, fermeture de page simulée, restart et échecs DB/filesystem.

Pour préserver un diagnostic minimal sans garder la ligne bloquante, les erreurs détaillées restent loguées. Un historique persistant des erreurs serait un chantier distinct, non requis pour débloquer l'Exam.

## Tests existants et manques

Couvert actuellement : transitions running→success/failed, idempotence et conflits terminaux, fallback lorsque le contexte est annulé, ownership/rows affected, recovery running avec cascade de `student_exam`, conservation failed/success au restart, workspaces de recovery et primitives de suppression.

Manques importants :

- test HTTP complet du polling failed et de sa cascade;
- chaîne partielle complète (`student_exam_content` et pages) nettoyée sur failed;
- échec avant redirection vers la progression;
- retry sans revisite de page;
- recovery startup d'une failed abandonnée;
- erreur de suppression DB ou workspace pendant cleanup;
- garantie explicite qu'une success n'est jamais supprimée par le futur cleanup.

## Priorité

**P1.** Le problème était P2 avant les nouveaux contrats. Il devient P1 car une failed abandonnée verrouille durablement modification, suppression et nouvelle génération, parfois sans chemin utilisateur permettant d'atteindre le cleanup.

## Fichiers inspectés et absence de modification

Ont été inspectés : migrations et requêtes `exams_generated`/descendance, handlers de génération et progression, transitions failed/success, recovery startup, workspaces, génération de copies, correction et tests DB/handlers/tools associés.

Aucun code, SQL, migration ou template n'a été modifié. Seul ce rapport a été créé.
