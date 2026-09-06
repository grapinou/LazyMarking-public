# Prévalidation de génération d'une évaluation

## 1. QCM vide avant / après

Avant, une évaluation liée à un QCM vide passait la validation du handler, créait `exams_generated`, son workspace et ses workers, puis échouait tardivement pendant la construction.

Après ce jalon, elle reste un brouillon Exam valide mais tout lancement de génération complète ou mini est refusé immédiatement avec un 303 vers `ErrorMessageURL` :

> Ajoutez au moins une question au QCM avant de générer l'évaluation.

## 2. Point exact de validation

Pour la génération complète, l'ordre est désormais :

1. authentification et parsing de `exam_id`;
2. `GetExamByID` ownership-aware;
3. lecture des élèves actuels de la classe et refus de la classe vide;
4. prévalidation ownership-aware du QCM et de sa non-vacuité;
5. validation existante des noms d'élèves;
6. seulement ensuite `CreateExamGenerated`, workspace et goroutine.

La génération mini suit les quatre premières étapes avant de démarrer ses workers ou de créer son workspace.

## 3. Requête/helper utilisé

Le helper local `validateExamQCMHasQuestions` réutilise `GetQCMQuestionsIDs(qcm_id,user_id)`. Cette requête :

- ne charge que les IDs nécessaires;
- est ordonnée par position;
- filtre `qcm_questions.user_id`;
- vérifie explicitement que le QCM appartient à l'utilisateur.

Aucune nouvelle requête SQL et aucun `sqlc generate` n'ont été nécessaires.

## 4. Ownership

`GetExamByID` refuse déjà un Exam absent, étranger ou dont un parent QCM est incohérent. La lecture directe des IDs revérifie malgré tout l'ownership du QCM. Une fixture d'ancienne ligne Exam appartenant à l'utilisateur mais liée à un QCM étranger retourne 404 et n'est jamais présentée comme « QCM vide ».

## 5. Artefacts créés

Pour un QCM vide :

- `exams_generated` créée : non;
- `student_exam` créée : non;
- workspace créé : non;
- goroutine/worker lancé : non.

Les tests exécutent les handlers dans un répertoire temporaire et vérifient que l'arbre `assets` n'est même pas créé.

## 6. Classe vide

Le comportement existant était déjà correctement placé avant `CreateExamGenerated`. Il est conservé : 303 métier indiquant que la classe ne contient aucun élève, sans génération ni worker. La prévalidation QCM vient ensuite et ne change pas ce contrat.

## 7. Génération mini

La génération mini consomme le même Exam, sa classe et le même QCM individualisé, même si elle ne produit pas d'historique `exams_generated`. Elle recevait donc le même échec tardif. Elle applique maintenant la même prévalidation et retourne le même message sans worker ni workspace.

Son ordre aléatoire, le choix main/variante et le mélange des réponses sont inchangés.

## 8. Limite de concurrence du QCM vivant

La validation traite le cas normal où le QCM est déjà vide au clic. Elle ne verrouille pas le QCM : celui-ci peut théoriquement être vidé après le précheck. Les workers relisent les IDs pour chaque copie via `GetQCMQuestionsAnswers*`; ils ne consomment donc pas un snapshot global pris par le preflight.

Dans cette course rare, le pipeline existant peut encore rencontrer un QCM vide pendant la construction et échouer tardivement, voire produire des comportements différents selon le moment où chaque worker relit le QCM. Résoudre totalement cette fenêtre demanderait un contrat de snapshot/verrouillage transactionnel hors périmètre. Aucun verrou QCM n'a été ajouté.

## 9. Périmètre inchangé

- Create/Edit Exam modifiés : non;
- schéma/migration/SQL : non;
- QCM : non;
- Marking : non;
- individualisation et workers : non;
- unicité, retry et cleanup `exams_generated` : non.

## 10. Tests

Scénarios ajoutés :

- génération complète avec QCM vide : 303, aucune génération/copie/workspace/goroutine;
- génération mini avec QCM vide : même protection;
- classe vide : 303 et aucune génération;
- Exam absent : 404;
- Exam étranger : 404;
- ancien Exam incohérent lié à un QCM étranger : 404;
- QCM non vide : le handler franchit le preflight et atteint le garde-fou de génération déjà existante.

Les fixtures représentent un QCM réellement non vide lorsque le scénario doit franchir la nouvelle précondition.

## 11. Validations

- `sqlc generate` : non nécessaire, SQL inchangé;
- `go test ./...` : réussi;
- `go vet ./...` : réussi;
- `git diff --check` : réussi.

## 12. Fichiers modifiés

- `internal/handlers/generateExams/handlers.go`;
- `internal/handlers/generateExams/handlers_preflight_test.go`;
- `docs/audits/exam-generation-preflight.md`.
