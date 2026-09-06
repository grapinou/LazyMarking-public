# Nettoyage automatique des générations d'évaluation échouées

## 1. Cycle de vie avant et après

Avant ce jalon, une génération terminée en erreur restait durablement avec le statut `failed`. Son nettoyage DB, celui de sa descendance et celui de son workspace dépendaient d'une visite ultérieure de la page de progression. Tant que cette visite n'avait pas lieu, l'évaluation restait non modifiable, non supprimable et non régénérable.

Après ce jalon, `failed` est un état transitoire : tout chemin d'échec de la génération complète marque d'abord la génération `failed`, puis la supprime immédiatement avec sa descendance et son workspace. Le nettoyage est effectué côté serveur et ne dépend plus du navigateur. Une génération `success` demeure un historique protégé.

## 2. DeleteFailedExamGenerated

La requête sqlc `DeleteFailedExamGenerated` a été ajoutée dans `db/query/examsGenerated.sql`. Elle supprime une ligne uniquement si les trois conditions suivantes correspondent :

- `id` ;
- `user_id` ;
- `status = 'failed'`.

Elle retourne le nombre de lignes affectées. Les tests démontrent qu'elle retourne `1` pour une génération `failed` possédée et `0` pour `running`, `success`, absente ou étrangère. Elle ne peut donc pas supprimer accidentellement un historique réussi.

Une requête `ListFailedExamGenerations` a également été ajoutée pour permettre au recovery de démarrage de reprendre les éventuelles lignes `failed` laissées par un arrêt ancien ou par une erreur de cleanup DB.

## 3. Primitive et idempotence

`tools.CleanupFailedExamGeneration` coordonne la suppression conditionnée de la ligne DB, la cascade des descendants et la suppression du workspace `exam-<id>`.

L'idempotence repose sur les contrats suivants :

- une ligne `failed` présente est supprimée ;
- une ligne déjà absente est acceptée et le nettoyage du workspace est tout de même tenté ;
- un workspace absent est accepté par la primitive filesystem existante ;
- les descendants absents ne demandent aucun traitement particulier ;
- une ligne encore `running` ou déjà `success` est explicitement refusée.

La primitive locale du package `generateExams` conserve la transition logique `running -> failed` avec `FailExamGeneration`, puis appelle la primitive conditionnée. Si le contexte de génération est déjà annulé, elle utilise un court contexte de cleanup indépendant afin que l'annulation ne laisse pas l'évaluation bloquée.

## 4. Ordre DB / filesystem

Le cleanup supprime d'abord la ligne DB, puis le workspace. Cet ordre donne la priorité à la libération fonctionnelle de l'évaluation. La cascade DB supprime atomiquement les `student_exam`, `student_exam_content` et `student_exam_page_content` associés.

Il n'existe pas de transaction distribuée entre SQLite et le filesystem. Si la suppression du workspace échoue après le DELETE, l'erreur est remontée ou journalisée, mais la ligne DB n'est pas recréée : l'évaluation ne reste pas bloquée. Le recovery ou une nouvelle invocation idempotente peut retenter la suppression du workspace.

Si le DELETE DB échoue après la transition vers `failed`, l'erreur n'est pas masquée. La ligne `failed` reste disponible et le recovery au prochain démarrage peut reprendre son nettoyage.

## 5. Chemins synchrones

Les erreurs survenant après la création de `exams_generated` mais avant la redirection vers la progression utilisent désormais le même cleanup :

- impossible de créer le workspace ;
- impossible de lire le nom de la classe.

La génération est marquée `failed` si elle est encore `running`, supprimée conditionnellement, puis son workspace éventuel est nettoyé avant la réponse HTTP. L'évaluation n'est donc plus bloquée après une erreur synchrone.

## 6. Chemins goroutine

La goroutine de génération possède un cleanup d'échec centralisé et protégé contre les doubles appels. Un `defer` l'exécute pour toute sortie qui n'a pas atteint la finalisation `success`, notamment :

- erreur d'un worker ;
- panic récupérée d'un worker ou du pipeline ;
- contexte annulé ou expiré ;
- erreur de listing des PDF ;
- erreur de fusion PDF ;
- erreur de finalisation DB.

Les anciens blocs dispersés de transition `failed` ont été remplacés par ce chemin unique. La fermeture de la page ou l'absence de polling n'influence pas le cleanup.

Si la finalisation a en réalité déjà placé la génération en `success` mais retourne une anomalie, le cleanup relit le statut et refuse de supprimer la ligne réussie.

## 7. Progression et polling

Le GET de progression n'effectue plus aucun DELETE ni nettoyage de workspace lorsqu'il observe `failed`. Il ne fait qu'afficher le message métier :

> La génération a échoué. Vous pouvez corriger l'évaluation puis réessayer.

La redirection initiale vers la progression transporte désormais `exam_id` et un marqueur `generation_started=1`. Si le polling suivant arrive après que le serveur a déjà supprimé la ligne échouée, le handler valide encore l'appartenance de l'évaluation puis affiche le même message. Une génération absente consultée sans ce contexte de workflow conserve le contrat 404 ; une évaluation absente ou étrangère reste également 404.

Une rare ligne `failed` encore visible au moment d'un poll n'est pas supprimée par ce GET : le cleanup serveur ou le recovery en reste responsable.

## 8. Recovery au démarrage

`RecoverRunningExamGenerations` conserve son nom public pour compatibilité mais traite désormais deux ensembles explicites :

- `running` abandonnée : suppression conditionnée existante et nettoyage workspace ;
- `failed` résiduelle : `CleanupFailedExamGeneration` et nettoyage workspace ;
- `success` : ni listée ni modifiée.

Les requêtes conditionnées par statut empêchent le recovery de supprimer une génération réussie.

## 9. Descendance et workspace

Les cascades existantes n'ont pas été modifiées. Le test de cycle complet construit :

```text
exams_generated failed
  -> student_exam
    -> student_exam_content
    -> student_exam_page_content
```

Après cleanup, toute cette descendance est absente, l'Exam parent est conservé et le workspace est supprimé. Le même cleanup réussit lorsque la ligne et le workspace sont déjà absents. Un test injecte aussi une erreur filesystem et confirme que la ligne DB reste supprimée.

## 10. Retry, modification et suppression

Après cleanup, l'absence de `exams_generated` libère immédiatement la contrainte `UNIQUE(exam_id, user_id)`. Un test recrée une génération sans revisiter la progression.

Les handlers Edit et Delete Exam n'ont pas été modifiés. Ils redeviennent naturellement autorisés après disparition de la génération échouée, conformément à leurs contrats existants fondés sur l'existence de `exams_generated`.

## 11. Protection de success

La protection de `success` est testée à trois niveaux :

- `DeleteFailedExamGenerated` affecte zéro ligne ;
- `CleanupFailedExamGeneration` refuse explicitement son nettoyage ;
- le recovery startup ne la liste ni ne la supprime.

Une génération réussie continue donc à rendre l'Exam immuable et non supprimable, et ses snapshots restent conservés.

## 12. Génération mini et Marking

La génération mini ne crée pas `exams_generated` et n'a pas été modifiée. Ses propres cleanups workspace restent inchangés.

Marking, ses handlers et ses tables n'ont pas été modifiés. Le cleanup automatique réduit la durée de présence d'éventuelles copies partielles sans changer le contrat de correction.

L'individualisation des copies — ordre des familles, choix principale/variante et mélange des réponses — n'a pas été modifiée.

## 13. Tests

Sept fonctions de test ont été ajoutées et une adaptée, avec des sous-scénarios ciblés :

- matrice DB `failed` / `running` / `success` / absent / étranger ;
- cascade complète et idempotence ;
- refus explicite de nettoyer `running` ou `success` ;
- erreur filesystem après succès DB ;
- échec synchrone avant redirection et retry immédiat ;
- échec worker sans polling et suppression du workspace ;
- polling après cleanup, polling arbitraire 404 et GET `failed` non destructif ;
- recovery startup enrichi avec génération `failed`, descendants partiels et workspace ;
- conservation explicite de `success` au recovery.

Tous les tests existants sont conservés.

## 14. Validations

- `sqlc generate -f db/sqlc.yaml` : succès ;
- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## 15. Fichiers du jalon

- `db/query/examsGenerated.sql` ;
- `internal/db/examsGenerated.sql.go` (généré par sqlc) ;
- `internal/handlers/generateExams/cleanupFailedExamGeneration.go` ;
- `internal/handlers/generateExams/handlers.go` ;
- `internal/handlers/generateExams/handlers_preflight_test.go` ;
- `internal/handlers/tools/cleanupFailedExamGeneration.go` ;
- `internal/handlers/tools/cleanupFailedExamGeneration_test.go` ;
- `internal/handlers/tools/recoverExamGenerations.go` ;
- `internal/handlers/tools/recoverExamGenerations_test.go` ;
- `docs/audits/exam-failed-generation-cleanup.md`.

Aucune migration, aucun template, aucune logique de génération mini, aucun code Marking et aucune règle d'individualisation n'ont été modifiés.
