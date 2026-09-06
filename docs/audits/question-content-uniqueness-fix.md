# Correction du contrat d’unicité du contenu des questions

## Ancien contrat et invalidité métier

Depuis la migration 0029, `questions` et `alt_questions` imposaient `UNIQUE(content, user_id)`. Ce contrat assimilait à tort un texte à une identité métier. Un même professeur doit pouvoir réutiliser un énoncé identique lorsque le stimulus, l’image, le parent ou le contexte pédagogique diffère, y compris deux fois sous le même parent alternatif.

## Nouveau contrat et migration 0041

`db/migrations/0041_remove_question_content_uniqueness.sql` reconstruit les deux tables sans aucune unicité fondée sur `content`. Les clés primaires, colonnes obligatoires, contrôles `length(trim(content)) > 0`, ownership, clés étrangères et actions `ON DELETE` sont conservés à l’identique.

SQLite validant les triggers dépendants pendant la reconstruction, la migration dépose puis recrée à l’identique les triggers d’ownership de `questions`, `alt_questions`, `answers`, `images`, `alt_answers`, `alt_images` et `qcm_questions`. Aucun contrat des tables enfants n’est changé. Les queries `CreateQuestion`, `UpdateQuestion`, `CreateAltQuestion` et `UpdateAltQuestion` restent compatibles et n’ont pas été régénérées. En particulier, `CreateAltQuestion` conserve sa vérification de parent appartenant au même utilisateur.

Le Down est explicitement et sûrement irréversible, selon la stratégie de 0029 : réintroduire les contraintes pourrait rejeter ou détruire les doublons valides créés après 0041.

## Conservation et intégrité

Le test de migration part d’un graphe contenant questions, alternatives, réponses, images, composition QCM et snapshot historique. Il confirme la conservation des IDs et comptes, la présence des triggers, les cascades des enfants, et la restriction de suppression d’une question utilisée par un QCM.

Une copie de `/home/sighto/Documents/lazymarking-6e-test/app.db` en version 0040 a été migrée avant la DB réelle : les comptes sont restés identiques (1 utilisateur, 3 questions, 12 réponses, 2 alternatives, 8 réponses alternatives et 1 image alternative). `PRAGMA foreign_key_check` n’a retourné aucune ligne ; `PRAGMA integrity_check` a retourné `ok`.

Après cette validation, la DB synthétique 6e a été migrée en 0041. Les mêmes données, tous les référentiels et le fichier uploadé `runtime/assets/images/1_prof-6e-test_altQuestion_2_balance.png` sont présents. Après la preuve HTTP, elle contient 3 alternatives, dont les deux doublons attendus.

## Tests du contrat

`internal/db/questionContentUniquenessMigration_test.go` prouve explicitement :

- deux principales identiques pour le même utilisateur : autorisées ;
- deux alternatives identiques sous des parents différents : autorisées ;
- deux alternatives identiques sous le même parent : autorisées ;
- même contenu entre deux utilisateurs : autorisé ;
- contenus principaux et alternatifs vides ou composés d’espaces : refusés ;
- ressources ou parent appartenant à un autre utilisateur : refusés par les triggers ;
- cascades `answers`, `images`, `alt_questions`, `alt_answers`, `alt_images` et restriction `qcm_questions` : conservées ;
- snapshot `student_exam_content` indépendant et conservé ;
- `foreign_key_check` vide et `integrity_check=ok`.

## Replay et preuve applicative 6e

Le replay Goose complet sur DB fraîche, 0001→0041, réussit. Sur la DB 6e migrée, le serveur a été relancé avec sa configuration privée. Après login réel, un GET du formulaire a fourni le jeton CSRF, puis le POST `/dashboard/questions/altquestions/add` a créé sous la principale 3 l’alternative exacte `Quel instrument est représenté ?`. Réponse : HTTP 303 vers `/dashboard/questions/altquestions?question_id=3`.

Le SELECT read-only confirme : alternative ID 2, parent 2, et alternative ID 3, parent 3, toutes deux avec le même contenu et `user_id=1`. Aucune autre donnée pédagogique n’a été créée ; les familles 4–10 n’ont pas été reprises.

## Validation

- `./scripts/check.sh` : succès (modules, replay Goose, gofmt, vet, tests, build, diff-check) ;
- `go test -race ./...` : succès ;
- `git diff --check` : succès ;
- DB 6e finale : `foreign_key_check` vide, `integrity_check=ok`.

## Dette UX inchangée

Le message générique du handler d’ajout d’alternative parle à tort de « même réponse » et mélange doublon et contenu vide. Retirer l’unicité rend la mention du doublon obsolète, mais la branche traite encore les autres erreurs DB et n’est pas dangereuse. Elle n’a donc pas été modifiée dans ce jalon UX-exclu.

## Fichiers modifiés

- ajouté : `db/migrations/0041_remove_question_content_uniqueness.sql` ;
- ajouté : `internal/db/questionContentUniquenessMigration_test.go` ;
- ajouté : `docs/audits/question-content-uniqueness-fix.md` ;
- aucun handler, query sqlc, template ou autre code métier modifié ;
- `db/migrations/0029_scope_unique_constraints_by_user.sql` inchangée.
