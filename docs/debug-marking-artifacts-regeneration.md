# Diagnostic de la régénération des artefacts de correction

## Symptôme observé

Après la correction initiale du paquet de 6eB et la validation manuelle des 16 réponses ambiguës, le PDF corrigé visible dans l'interface ne change pas. Pour le job réel `4`, la base contient `review_revision = 16` et `artifacts_revision = 0`. Les fichiers canoniques `corrected.pdf` et `mark-table.pdf` ont conservé leur date de génération initiale du 3 septembre 2026 à 19:04, antérieure aux dernières écritures de review.

Le backend n'a donc pas publié de nouveaux artefacts. Ce symptôme ne vient pas de l'affichage d'un PDF nouvellement généré qui serait resté en cache.

## Cause racine

Le serveur utilisé pour le test réel n'était pas le binaire recompilé contenant le correctif de pagination.

Deux exécutables distincts existent :

- `/home/sighto/LazyMarking/app`, construit le 3 septembre à 19:01, module `3c0531c3901e+dirty`, ne contient plus le message `page answer snapshot mismatch` ;
- `/home/sighto/Documents/lazymarking-6e-test/lazymarking-server`, construit le 2 septembre à 17:44, module `b6aa8486b4c2+dirty`, contient encore ce message et donc l'ancienne implémentation.

La recompilation a bien créé le premier fichier, mais elle n'a ni remplacé le second ni redémarré le test avec le nouvel exécutable. Le chemin réel de données (`runtime/assets/tmp/...`) et la base du test sont ceux de l'environnement `lazymarking-6e-test`, où l'ancien exécutable était encore installé.

La preuve complémentaire est une reproduction sur une copie isolée du job 4, de sa base et de tous ses artefacts : avec le code courant, la régénération termine sans erreur, remplace les deux PDF et fait passer `artifacts_revision` de 0 à 16. La taille et la date des deux fichiers changent. Il ne subsiste donc pas d'erreur applicative reproductible sur ces données après le correctif de pagination.

## Chemin d'exécution

1. La page de résultat affiche un formulaire `POST /dashboard/marking/artifacts/regenerate` avec `job_id` et le jeton CSRF.
2. `RegenerateMarkingArtifactsHandler` authentifie la requête, lit l'utilisateur et son nom de session, parse le `job_id`, puis appelle `tools.RegenerateMarkingArtifacts`.
3. `RegenerateMarkingArtifacts` charge par `(marking_job_id, user_id)` les révisions, `ambiguity_delta` et les chemins canoniques. Il ne fait rien uniquement si `artifacts_revision == review_revision`.
4. Le workspace est `assets/tmp/<username>/marking-<job_id>`. Les chemins stockés en base doivent désigner exactement `corrected.pdf` et `mark-table.pdf` dans ce workspace.
5. Un staging `.review-artifacts-*` est créé dans ce workspace.
6. Le générateur charge toutes les copies du job. Chaque copie corrigée est identifiée par son `copy_result_id`, son `student_exam_id`, son nombre de pages et son snapshot QCM.
7. `ListEffectiveMarkingAnswersForArtifacts` joint détections et reviews et reconstruit l'état avec `COALESCE(reviewed_state, detected_state)`. Les valeurs manuelles priment donc sur la détection automatique.
8. `regenerateCorrectedCopy` recalcule chaque résultat de question avec ces états effectifs et vérifie sa cohérence avec les résultats persistés. Pour le job 4, 16 réponses ont été revues et 9 diffèrent de la détection automatique.
9. Pour chaque page, le snapshot de positions et l'image alignée historique sont chargés. Les questions et les réponses sont consommées avec leurs offsets indépendants, puis `DrawMarking` écrit l'annotation sur la copie de l'image dans le staging.
10. Chaque PNG annoté est converti en PDF. Les pages d'une copie sont fusionnées en `student-exam-<id>.pdf`.
11. Les PDF des copies sont fusionnés dans le `corrected.pdf` du staging ; le tableau des notes est produit dans le même staging.
12. Les deux PDF générés sont validés, les fichiers canoniques existants sont renommés en backups, puis les nouveaux fichiers sont renommés vers les chemins canoniques. Toute erreur déclenche un rollback de la paire.
13. La base avance `artifacts_revision` uniquement après publication réussie et seulement si les deux révisions attendues n'ont pas changé concurremment.
14. Le handler répond par une redirection HTTP 303 vers `/dashboard/marking/success?job_id=<id>`. En cas d'erreur, il ajoute `notice=artifacts_failed`.
15. La page de résultat reconstruit l'URL `/dashboard/marking/pdf?operation=marking-<id>&file=corrected.pdf`. `ServePdfNamed` ouvre exactement le fichier canonique de ce workspace et utilise `http.ServeContent` avec sa date de modification.

Le dernier formulaire de review emprunte le même chemin automatiquement : après `ApplyMarkingAnswerReview`, s'il ne reste aucun candidat, `ApplyMarkingReviewHandler` appelle directement la régénération avant sa redirection vers la page de résultat.

## Investigations réalisées

- Appel HTTP : la route, la méthode, l'authentification, le CSRF, le `job_id`, les redirections de succès et d'échec ont été vérifiés.
- État réel : les jobs 2, 3 et 4 ont chacun `review_revision = 16` et `artifacts_revision = 0`.
- Reviews : le job 4 contient bien 16 reviews ; 9 états manuels diffèrent de la détection automatique.
- Sélection des états : la requête de régénération utilise bien `COALESCE(reviewed_state, detected_state)` et trie par question puis réponse.
- Intégrité : pour les copies corrigées des jobs concernés, les nombres de questions, réponses, détections, pages attendues, snapshots de page et images alignées concordent.
- Génération isolée : le job réel copié régénère correctement avec le code courant et atteint la révision 16.
- Publication : sur la copie, les timestamps et tailles des PDF canoniques changent, et la révision n'avance qu'après les renommages.
- Fichiers réels : leurs timestamps n'ont pas changé après les reviews ; aucun nouveau PDF n'a donc atteint le chemin servi par l'interface.
- Interface et cache : l'URL est stable et `ServeContent` fournit la date de modification, mais ce point n'explique pas le cas observé puisque le fichier réel et la révision de base sont restés anciens.
- Déploiement : le binaire recompilé et le binaire installé pour le test ont des hashes, dates, révisions de module et chaînes embarquées différents. Seul le binaire installé contient encore l'ancienne erreur.
- Diagnostic temporaire : aucun log permanent n'a été ajouté, car la reproduction isolée et l'inspection des exécutables ont fourni une cause déterministe. Le petit exécutable de diagnostic a été retiré du dépôt.

## Fichiers modifiés

- `internal/handlers/tools/markingArtifactsGenerator.go` : correctif local précédent, qui consomme indépendamment les questions et réponses page par page.
- `internal/handlers/tools/markingArtifactsGenerator_test.go` : tests de pagination et de snapshots incohérents.
- `docs/debug-marking-artifacts-regeneration.md` : présent rapport de diagnostic.

## Correctif appliqué

Le correctif applicatif nécessaire est déjà celui de `regenerateCorrectedCopy` : liste globale des réponses et deux offsets indépendants. Aucun second changement de logique n'est justifié par les données réelles, puisque ce code régénère correctement le job 4 complet.

La correction du symptôme réel est opérationnelle : installer le binaire recompilé à l'emplacement effectivement lancé pour le test, arrêter l'ancienne instance si elle tourne encore, puis démarrer cette nouvelle version avec le répertoire de travail `runtime`. Une nouvelle demande d'actualisation pourra alors publier les PDF et avancer `artifacts_revision` à 16.

Le dépôt ne modifie volontairement ni scoring, ni détection, ni review, ni `DrawMarking`, ni SQL, ni migrations, ni templates, ni politique de cache.

## Tests ajoutés ou modifiés

Les tests de régression ciblés dans `markingArtifactsGenerator_test.go` couvrent :

- pagination classique ;
- question dont les réponses traversent une page ;
- page avec réponses et sans question ;
- dépassement des questions ou réponses ;
- questions ou réponses non entièrement consommées.

Le scénario réel a en plus été exécuté sur une copie isolée de la base et des artefacts 6eB : résultat `Regenerated: true`, révisions `16/16`.

## Validation

- Tests ciblés : `go test ./internal/handlers/tools` — réussi.
- Suite complète : `go test ./...` — réussi.
- Vérification du diff : `git diff --check` — réussi.
- Reproduction isolée du job 4 : réussie ; les deux PDF sont remplacés et la base avance à la révision 16.

## Points restant à surveiller

- Le processus de déploiement local permet actuellement de compiler un nouveau fichier sans remplacer automatiquement l'exécutable installé. Le chemin du binaire lancé doit être contrôlé explicitement après chaque recompilation.
- L'URL du PDF reste identique entre deux révisions. Ce n'est pas la cause du présent incident, mais un onglet déjà ouvert peut nécessiter un rechargement après une publication réussie selon le comportement du navigateur.
- Les jobs 2, 3 et 4 de l'environnement réel restent volontairement inchangés par ce diagnostic (`artifacts_revision = 0`) ; aucune donnée réelle n'a été publiée ou modifiée.
