# Audit final avant commit

## 1. Objectif

Auditer le diff courant et les fichiers non suivis avant sélection Git. Lecture du rapport UX, examen des modifications et recherche de données sensibles ; aucun changement de source, UX, test, migration ou règle métier. Aucun `git add`, commit ou push.

## 2. État Git initial

105 fichiers suivis affectés : 104 modifiés, 1 supprimé ; 101 fichiers non suivis (93 Markdown, 6 Go, 1 migration SQL, 1 script shell). Index vide : aucun changement déjà stagé. `git diff --check` initial réussi. Diff suivi : 1 343 insertions, 1 300 suppressions.

Le rapport `2026-09-05-ux-modernization-audit.md` distingue les 12 modifications suivies et 100 fichiers non suivis préexistants du jalon UX. Le rapport de nommage `docs/reports/2026-09-04-marking-artifact-filenames.md` documente le travail de persistance/téléchargement antérieur. Les sauvegardes temporaires du jalon UX ont été supprimées ; l’attribution historique s’appuie sur ces rapports et la continuité de session, corroborées par le diff courant, pas sur une nouvelle comparaison à ces sauvegardes disparues.

Le diff complet a été traité par fichier : examen détaillé des changements structurels/HTML et revue des substitutions Go (avec regroupement des substitutions répétées), complétés par comparaison lexicale. Les fichiers non suivis ont été inventoriés et examinés par catégorie ; les 93 documents ont été parcourus par recherche de contenu sensible et les passages signalés inspectés. Ce contrôle ne certifie pas à nouveau toutes les conclusions historiques de chaque rapport.

## 3. Modifications suivies

Chaque fichier du diff suivi figure ci-dessous. Les origines multiples sont indiquées explicitement.

| Fichier / groupe | Origine | Verdict | Remarque |
| ---------------- | ------- | ------- | -------- |
| `db/query/markingJobs.sql` | Préexistant | PREEXISTANT — attendu | Ajout de source_pdf_filename au contrat SQL de nommage PDF. |
| `db/query/markingReviews.sql` | Préexistant | PREEXISTANT — attendu | Ajout de source_pdf_filename au contrat SQL de nommage PDF. |
| `internal/db/markingJobs.sql.go` | Préexistant / sqlc | GÉNÉRÉ — cohérent | Colonne nullable, arguments et Scan alignés sur le SQL et la migration 0044. |
| `internal/db/markingReviews.sql.go` | Préexistant / sqlc | GÉNÉRÉ — cohérent | Colonne nullable, arguments et Scan alignés sur le SQL et la migration 0044. |
| `internal/db/markingReviews_test.go` | Préexistant | TEST — attendu | Fixtures et assertions du nommage PDF ; protections antérieures conservées. |
| `internal/db/models.go` | Préexistant / sqlc | GÉNÉRÉ — cohérent | Colonne nullable, arguments et Scan alignés sur le SQL et la migration 0044. |
| `internal/handlers/about/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/about/view.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/altAnswers/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/altImages/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/altPreview/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/altQuestions/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/answers/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/classCodes/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/classCodes/viewData.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/dashboard/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/difficulties/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/errorsmessages/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/exams/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/generateExams/cleanupFailedExamGeneration.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/generateExams/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/home/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/home/view.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/images/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/login/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/login/middlewares.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/login/view.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/logout/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/marking/handlers.go` | Préexistant + UX | PREEXISTANT — attendu / UX — attendu | Nommage de téléchargement documenté, puis traduction ; contrôles de propriétaire, révisions et chemins conservés. |
| `internal/handlers/marking/handlers_test.go` | Préexistant | TEST — attendu | Fixtures et assertions du nommage PDF ; protections antérieures conservées. |
| `internal/handlers/marking/hybridLifecycle_test.go` | Préexistant | TEST — attendu | Fixtures et assertions du nommage PDF ; protections antérieures conservées. |
| `internal/handlers/marking/review_test.go` | Préexistant | TEST — attendu | Fixtures et assertions du nommage PDF ; protections antérieures conservées. |
| `internal/handlers/periods/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/points/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/preview/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/qcm/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/qcmPreview/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/qcmQuestions/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/questions/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/register/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/register/validation.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/register/view.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/resetpassword/handler.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/resetpassword/view.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/skills/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/studentClassCode/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/studentClassCode/viewData.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/students/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/students/viewData.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/students/viewData_test.go` | UX | TEST — attendu | Attentes de rendu uniquement. |
| `internal/handlers/subjects/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/themes/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/tools/checkRequest.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/tools/ownedMutation.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/tools/passwordValidation.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/tools/servePdfNamed.go` | Préexistant + UX | PREEXISTANT — attendu / UX — attendu | Nommage de téléchargement documenté, puis traduction ; contrôles de propriétaire, révisions et chemins conservés. |
| `internal/handlers/tools/servePdfNamed_test.go` | Préexistant | TEST — attendu | Fixtures et assertions du nommage PDF ; protections antérieures conservées. |
| `internal/handlers/tools/serveUserImage.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/yearlevels/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/handlers/years/handlers.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/httpsecurity/csrf_test.go` | UX | TEST — attendu | Attentes de rendu uniquement. |
| `internal/mailer/mailer.go` | UX | UX — attendu | Chaînes visibles uniquement ; instructions et conditions inchangées. |
| `internal/templates/altanswers/add_form_alt_answer.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altanswers/edit_form_alt_answer.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altimages/add_form_alt_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altimages/delete_form_alt_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altimages/edit_form_alt_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altimages/table_alt_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altquestions/add_form_alt_question.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/altquestions/edit_form_alt_question.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/answers/add_form_answer.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/answers/edit_form_answer.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/dashboard/dashboard.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/errors/question_feature_error.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/exams/table_exams.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/about.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/home.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/layout.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/login.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/register.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/resetpassword.html` | UX | UX — attendu | Suppression documentée du template invalide orphelin ; formulaire de reset routé conservé. |
| `internal/templates/home/showrequestresetpasswordform.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/showresetpasswordform.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/home/success.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/images/add_form_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/images/delete_form_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/images/edit_form_image.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/marking/progress_marking.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/marking/review.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/marking/success_marking_processing.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/marking/table_marking.html` | UX | UX — attendu | Dépôt accessible, état vide, garde JS sans formulaire ; validation PDF métier inchangée. |
| `internal/templates/periods/add_form_period.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/periods/delete_form_period.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/periods/edit_form_period.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/periods/table_periods.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/qcmquestions/add_form_qcm_question.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/questions/add_form_question.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/questions/edit_form_question.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/studentClassCodes/delete_form_student_class_code.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/students/add_csv_form_student.html` | UX | UX — attendu | Bouton natif remplaçant le keydown manuel ; champ fichier visible. |
| `internal/templates/students/add_form_student.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/years/add_form_year.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/years/delete_form_year.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/years/edit_form_year.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |
| `internal/templates/years/table_years.html` | UX | UX — attendu | Présentation, sémantique et actions ; contrat des formulaires conservé. |

## 4. Fichiers non suivis

### Code à conserver

- `internal/handlers/tools/markingArtifactFilename.go` : nécessaire aux handlers modifiés ; nom visible dérivé du basename, sans utilisation comme chemin de stockage. **PREEXISTANT — attendu**.

### Tests à conserver

- `internal/db/markingReviewReadModel_test.go` : états dérivés de revue et incohérences de compteurs.
- `internal/handlers/tools/drawMarking_test.go` : sémantique du rendu des marques.
- `internal/handlers/tools/markingArtifactFilename_test.go` : suffixes, Unicode, traversées et CR/LF.
- `internal/handlers/tools/markingArtifactsGenerator_test.go` : pagination des snapshots et cohérence des réponses.
- `internal/handlers/tools/markingAmbiguityCalibrationBatch_integration_test.go` : outil de test opt-in, pas un résultat de diagnostic. Base copiée, statistiques anonymisées, export des crops imposé hors dépôt ; ne contient pas de corpus. Les variables LAZYMARKING nécessaires sont absentes dans cet audit : le parcours réel n’est pas lancé. Son export remplace ses fichiers contrôlés dans le répertoire explicitement configuré ; ce comportement est destiné au diagnostic manuel, pas au test standard.

Ces cinq fichiers sont **TEST — attendu**, préexistants au jalon UX.

### Migration à conserver avec le code associé

`db/migrations/0044_add_marking_source_filename.sql` : **PREEXISTANT — attendu**. Elle est indispensable aux nouvelles requêtes et ne doit pas être omise d’un commit embarquant les handlers/bindings de nommage.

### Documentation à conserver, sauf exclusions ci-dessous

90 des 93 Markdown non suivis initiaux : rapports sous `docs/audits/`, trois documents techniques à la racine de `docs/`, et rapport de nommage sous `docs/reports/`. **DOCUMENTATION — attendu**. Ils décrivent différents jalons historiques : les verdicts datés ne constituent pas tous des déclarations sur l’état actuel. Les identités repérées sont synthétiques/pseudonymisées, les résultats sont agrégés ou référencés par IDs ; les chemins locaux de diagnostic sont des références textuelles, pas des artefacts embarqués.

Le présent rapport est le seul fichier créé par cet audit. La liste exacte initiale des fichiers non suivis est également disponible dans le rapport UX (avec son propre ajout).

### Fichiers à exclure précisément, sans les supprimer

| Fichier | Motif | Traitement nécessaire avant toute inclusion future |
| --- | --- | --- |
| `docs/audits/6e-class-test-preparation.md` | Lignes 22–23 et 27 : clés de session/CSRF et identifiants de connexion d’une installation persistante de test. | Retirer les valeurs et remplacer les instructions par des références à une configuration privée. |
| `docs/audits/6e-class-test-preparation-final.md` | Lignes 62–63 et 67 : mêmes catégories de clés et mot de passe de test. | Même assainissement ; ne pas recopier les valeurs dans un autre document public. |
| `docs/audits/modern-fresh-install-smoke.md` | Lignes 17, 93–94 et 98 : mot de passe et clés de l’installation smoke. | Assainir les identifiants et commandes de reprise avant inclusion. |
| `reset.sh` | Lignes 6–10 : suppression directe de `db/data/app.db` et nettoyage d’un workspace utilisateur codé en dur ; script local non nécessaire au jalon. | Le laisser local. Son éventuelle transformation en outil partagé exige une décision distincte et des protections explicites ; rien n’a été exécuté ou corrigé ici. |

Exclusions de catégories déjà effectives : `testdata/`, `runtime/`, `assets/`, `.env`, SQLite sous `db/data/`, logs et binaires temporaires. Ne pas forcer leur ajout. Aucun de ces artefacts n’apparaît dans les fichiers candidats du status initial.

## 5. Vérification métier

**Aucune modification métier involontaire identifiée dans le diff examiné.**

- 48 fichiers Go de production modifiés ont une séquence de tokens identique à HEAD en ignorant les chaînes littérales et commentaires. Les substitutions ont été vérifiées : titres, erreurs HTTP, validation textuelle et e-mail. Les `if`, boucles, transactions, appels DB, statuts et autorisations n’y changent pas.
- Les cinq autres fichiers Go de production avec changements structurels sont les trois bindings/modèles SQL et les deux fichiers mixtes cités ci-dessus. Leurs ajouts correspondent au nommage préexistant documenté.
- Dans `marking/handlers.go`, lecture du basename après validation de l’upload, enregistrement du champ nullable et sélection du nom de téléchargement. Les contrôles d’appartenance, génération prête, revue en attente et synchronisation des révisions précèdent toujours le service du PDF.
- Dans `servePdfNamed.go`, le wrapper historique reste `inline`; le nouveau wrapper est `attachment`. Le chemin ouvert dépend toujours du nom physique validé. Composants sûrs, refus de symlinks, fichier régulier et comparaison du fichier ouvert sont conservés. La valeur utilisateur est utilisée seulement dans le nom de présentation encodé par `mime.FormatMediaType`.
- Les 42 templates modifiés encore présents conservent les mêmes actions/méthodes/enctype et champs (noms, types, valeurs, required, bornes, accept, multiple et disabled), vérifiés par extraction des attributs et comparaison à HEAD. Les nouvelles branches JS sont liées au rendu vide/accessible ; les traitements de sélection restent les mêmes.
- L’unique suppression est le template de reset orphelin. La recherche de consommateurs ne trouve que le commentaire du test CSRF ; les templates de reset réellement utilisés restent présents.
- CSRF : aucun middleware ou contrôle métier retiré. L’inventaire de formulaires passe de 73 à 72 à cause de cette suppression. L’autre adaptation de test remplace l’attente tabindex manuel par celle du bouton natif CSV ; le parcours clavier était validé au jalon UX.
- Pas de modification de scoring, consolidation, statut de revue, workflow de génération ou règles de suppression dans le jalon UX.

Le dépôt contient donc bien un changement fonctionnel **préexistant** (noms de téléchargement/persistance du nom source). Il ne faut pas présenter l’ensemble du futur commit comme exclusivement UX.

## 6. Vérification SQL / sqlc

Migration 0044 : colonne TEXT nullable, sans backfill ; trigger d’immutabilité utilisant `IS NOT`, y compris transitions NULL ; Down retire le trigger avant la colonne.

`CreateHybridMarkingJob` : treizième paramètre source_pdf_filename dans le SQL, `?13` dans le binding, `sql.NullString` dans les paramètres et argument transmis au même rang. Le filtre de génération réussie et propriétaire n’est pas modifié.

`GetMarkingArtifactsRegenerationTarget` : même colonne au même rang dans SELECT, structure retournée et Scan, avant PendingCandidates. Le modèle MarkingJob porte la même nullabilité. Les schémas minimaux de tests ont été adaptés. Aucun décalage ou fichier généré manquant trouvé ; `sqlc generate` n’a pas été relancé, puisqu’aucune incohérence ne le justifie.

Le replay des migrations jusqu’à 44 réussit sur la base temporaire créée par `scripts/check-migrations.sh`. Aucune migration n’est appliquée à une base historique pendant cet audit.

## 7. Vérification sécurité et données privées

La recherche couvre toutes les additions/suppressions du diff et tous les fichiers non suivis non ignorés : extensions et contenu texte, mots de passe/clés, tokens/cookies, e-mails, clés privées usuelles, images encodées, identités et payloads QR. Les correspondances ont été contextualisées ; les secrets constatés dans les trois rapports sont volontairement absents de ce rapport.

Les clés sont décrites comme synthétiques, mais les documents les donnent comme configuration de reprise d’installations persistantes ; leur caractère réutilisable justifie l’exclusion. Leur validité actuelle n’a pas été testée. Si ces installations doivent rester utilisées ou ont été exposées, renouveler leurs valeurs dans leur configuration privée avant réutilisation ; aucune rotation n’a été réalisée ici.

Aucun nom/prénom réel d’élève, e-mail réel, cookie de session, QR réel complet, base, scan ou PDF réel identifié dans le périmètre proposé hors exclusions. Les adresses d’exemple repérées utilisent un domaine de test. Les recherches ne constituent pas une preuve absolue d’absence de toute donnée personnelle non reconnaissable automatiquement.

`git ls-files testdata runtime assets` est vide. `git check-ignore -v` confirme `/testdata/` et `/runtime/` dans `.gitignore` (lignes 36–37), ainsi que `.env`. Ces protections préexistantes sont inchangées. Aucun contenu de base historique ou corpus original n’a été ouvert ou modifié par cet audit.

## 8. Tests

Exécutés pendant cet audit, dans l’état courant :

| Commande | Résultat |
| --- | --- |
| `git diff --check` | Succès, code 0 ; aucune sortie d’erreur. |
| `go test ./...` | Succès, code 0 ; cache Go utilisé lorsque applicable. |
| `./scripts/check.sh` | Succès, code 0 : modules, replay Goose sur DB temporaire, gofmt, vet, tests, build, diff-check. |

Les tests d’intégration nécessitant un corpus configuré restent opt-in et ne sont pas rejoués. Aucun serveur applicatif lancé, aucun envoi d’e-mail, aucune exécution de reset.sh. Les contrôles de rendu navigateur du jalon UX ne sont pas inutilement rejoués.

## 9. Anomalies éventuelles

- **Importante — exclusion impérative** : trois rapports contiennent des identifiants et clés de test réutilisables. Ne pas les committer dans leur état actuel. Assainissement exact décrit en section 4 ; aucun changement silencieux effectué.
- **Importante — exclusion impérative** : `reset.sh` est un utilitaire local destructif ciblant la DB applicative, sans isolation. Il reste intact et non exécuté.
- **Mineure — documentation** : le rapport UX regroupe les erreurs standard 404/CSRF parmi les réponses brutes ou potentiellement anglaises. Le CSRF applicatif possède déjà un message français ; le caractère brut est confirmé, pas un défaut de traduction de ce message précis. Aucun correctif produit nécessaire.
- **Aucune anomalie bloquante du code retenu identifiée**. Les limites UX déjà documentées ne deviennent pas des régressions nouvelles dans cet audit. Un ajout global indiscriminé de tous les fichiers reste inapproprié à cause des quatre exclusions.

## 10. Verdict final

**PRÊT À COMMITTER AVEC EXCLUSIONS**.

Exclure exactement :

1. `reset.sh`
2. `docs/audits/6e-class-test-preparation.md`
3. `docs/audits/6e-class-test-preparation-final.md`
4. `docs/audits/modern-fresh-install-smoke.md`

Conserver ensemble le code de nommage, la migration 0044, ses bindings et tests. Les autres modifications examinées sont documentées et cohérentes avec leur origine. Aucun staging, commit ou push effectué. Le seul changement produit par cet audit est le présent rapport ; les fichiers utilisateur, même ceux à exclure, restent intacts.

Contrôle de clôture : les empreintes de tous les fichiers initiaux et le diff suivi sont inchangés. Le répertoire temporaire propre à cet audit (`/tmp/lazymarking-precommit-jay22fxk/`, comparaison et journaux) a été supprimé ; aucun fichier utilisateur n’a été supprimé.
