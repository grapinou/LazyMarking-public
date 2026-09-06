# Audit final de modernisation UX

## 1. Objectif

Finaliser l’UX existante de LazyMarking, exclusivement sur la présentation. Aucun ajout fonctionnel, changement de règles, de permissions, de routes, de format de données ou de schéma. Aucun git add, commit ou push.

## 2. État initial du jalon

Avant modification : 12 fichiers suivis modifiés et 100 fichiers non suivis préexistants. `git diff --check` réussissait. Une copie des fichiers, leurs empreintes et le diff initial ont permis de distinguer les modifications du jalon de celles déjà présentes. Les fichiers SQL, migrations, code généré et tests de correction déjà modifiés appartiennent au travail préexistant.

## 3. État au moment de la reprise

Le rapport existait, avec un inventaire fichier par fichier mais des conclusions encore provisoires. 94 fichiers existants avaient déjà été modifiés ou supprimés dans le jalon. Les principales corrections étaient présentes : accueil/authentification, années/périodes, import CSV/PDF, progression, erreurs et actions secondaires. Le second passage de `go test ./...` avait réussi. Le serveur isolé et Chromium avaient été préparés ; le parcours réel des pages et `scripts/check.sh` restaient à finaliser.

La reprise a conservé ces changements, sauvegardé l’état de reprise et terminé les contrôles. Elle n’a pas recréé une autre direction visuelle.

## 4. Références UX

Dashboard et banque de questions déjà modernisés ; formulaires de suppression des questions/réponses, référentiels, QCM, évaluations et revue. Bootstrap 5.3.3 et Bootstrap Icons existants ; titres h1/h2, cartes shadow-sm, texte secondaire, actions principales bleues, secondaires en contour, suppressions rouges explicitement nommées. Navigation conservée : Banque de questions, Élèves, QCM, Évaluations, Correction.

## 5. Inventaire complet

Les 99 templates initiaux sont répertoriés ci-dessous, dont les deux layouts partagés et un template orphelin supprimé. Pas de système distinct de partiels, de modales ou de pagination trouvé. L’inventaire couvre aussi les formulaires secondaires liés depuis les tables, les confirmations et les branches vides. « OK » désigne le constat du code ; le parcours navigateur porte sur les exemples détaillés en section 11, pas sur toutes les combinaisons de données.

| Zone | Surface | État initial | État final | Action |
| ---- | ------- | ------------ | ---------- | ------ |
| altanswers | `internal/templates/altanswers/add_form_alt_answer.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altanswers | `internal/templates/altanswers/delete_form_alt_answer.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| altanswers | `internal/templates/altanswers/edit_form_alt_answer.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altanswers | `internal/templates/altanswers/table_alt_answers.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| altimages | `internal/templates/altimages/add_form_alt_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altimages | `internal/templates/altimages/delete_form_alt_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altimages | `internal/templates/altimages/edit_form_alt_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altimages | `internal/templates/altimages/table_alt_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altquestions | `internal/templates/altquestions/add_form_alt_question.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altquestions | `internal/templates/altquestions/delete_form_alt_question.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| altquestions | `internal/templates/altquestions/edit_form_alt_question.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| altquestions | `internal/templates/altquestions/table_alt_questions.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| answers | `internal/templates/answers/add_form_answer.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| answers | `internal/templates/answers/delete_form_answer.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| answers | `internal/templates/answers/edit_form_answer.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| answers | `internal/templates/answers/table_answers.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| classcodes | `internal/templates/classcodes/add_form_class_code.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| classcodes | `internal/templates/classcodes/delete_form_class_code.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| classcodes | `internal/templates/classcodes/edit_form_class_code.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| classcodes | `internal/templates/classcodes/table_class_codes.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| dashboard | `internal/templates/dashboard/dash_home.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| dashboard | `internal/templates/dashboard/dashboard.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| difficulties | `internal/templates/difficulties/add_form_difficulty.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| difficulties | `internal/templates/difficulties/delete_form_difficulty.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| difficulties | `internal/templates/difficulties/edit_form_difficulty.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| difficulties | `internal/templates/difficulties/table_difficulties.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| errors | `internal/templates/errors/question_feature_error.html` | P1 — ancienne UX visible | OK | Carte d’erreur lisible, retour et tableau de bord. |
| exams | `internal/templates/exams/add_form_exam.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| exams | `internal/templates/exams/delete_form_exam.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| exams | `internal/templates/exams/edit_form_exam.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| exams | `internal/templates/exams/table_exams.html` | P3 — détail cosmétique | OK | Icône décorative masquée aux technologies d’assistance. |
| generateExam | `internal/templates/generateExam/processing_students.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| generateExam | `internal/templates/generateExam/success_processing.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| generateExam | `internal/templates/generateExam/unavailable_pdf.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| home | `internal/templates/home/about.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/home.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/layout.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/login.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/register.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/resetpassword.html` | P3 — détail cosmétique | Supprimé (orphelin) | Template invalide sans consommateur ; le formulaire de reset réellement routé est conservé. |
| home | `internal/templates/home/showrequestresetpasswordform.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/showresetpasswordform.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| home | `internal/templates/home/success.html` | P1 — ancienne UX visible | OK | Cartes et formulaires alignés, français, labels, navigation et retours accessibles. |
| images | `internal/templates/images/add_form_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| images | `internal/templates/images/delete_form_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| images | `internal/templates/images/edit_form_image.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| images | `internal/templates/images/table_image.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| marking | `internal/templates/marking/progress_marking.html` | P1 — ancienne UX visible | OK | Dépôt PDF accessible ou progression en carte ; mêmes traitements. |
| marking | `internal/templates/marking/review.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| marking | `internal/templates/marking/success_marking_processing.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| marking | `internal/templates/marking/table_marking.html` | P1 — ancienne UX visible | OK | Dépôt PDF accessible ou progression en carte ; mêmes traitements. |
| periods | `internal/templates/periods/add_form_period.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| periods | `internal/templates/periods/delete_form_period.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| periods | `internal/templates/periods/edit_form_period.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| periods | `internal/templates/periods/table_periods.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| points | `internal/templates/points/add_form_point.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| points | `internal/templates/points/delete_form_point.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| points | `internal/templates/points/edit_form_point.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| points | `internal/templates/points/table_points.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcm | `internal/templates/qcm/add_form_qcm.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcm | `internal/templates/qcm/delete_form_qcm.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcm | `internal/templates/qcm/edit_form_qcm.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcm | `internal/templates/qcm/table_qcm.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcmquestions | `internal/templates/qcmquestions/add_form_qcm_question.html` | P2 — incohérence mineure | OK | Action principale bleue ; badges pouvant revenir à la ligne. |
| qcmquestions | `internal/templates/qcmquestions/delete_form_qcm_question.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| qcmquestions | `internal/templates/qcmquestions/table_qcmquestion.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| questions | `internal/templates/questions/add_form_question.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| questions | `internal/templates/questions/delete_form_question.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| questions | `internal/templates/questions/edit_form_question.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| questions | `internal/templates/questions/table_questions.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| skills | `internal/templates/skills/add_form_skill.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| skills | `internal/templates/skills/delete_form_skill.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| skills | `internal/templates/skills/edit_form_skill.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| skills | `internal/templates/skills/table_skills.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| studentClassCodes | `internal/templates/studentClassCodes/add_form_student_class_code.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| studentClassCodes | `internal/templates/studentClassCodes/delete_form_student_class_code.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| studentClassCodes | `internal/templates/studentClassCodes/table_student_class_codes.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| students | `internal/templates/students/add_csv_form_student.html` | P2 — incohérence mineure | OK | Bouton natif et champ fichier visible ; suppression du style historique sans focus. |
| students | `internal/templates/students/add_form_student.html` | P2 — incohérence mineure | OK | Hiérarchie des actions, sémantique ou libellés harmonisés. |
| students | `internal/templates/students/delete_form_all_students.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| students | `internal/templates/students/delete_form_student.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| students | `internal/templates/students/edit_form_student.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| students | `internal/templates/students/table_students.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| subjects | `internal/templates/subjects/add_form_subject.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| subjects | `internal/templates/subjects/delete_form_subject.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| subjects | `internal/templates/subjects/edit_form_subject.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| subjects | `internal/templates/subjects/table_subjects.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| themes | `internal/templates/themes/add_form_theme.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| themes | `internal/templates/themes/delete_form_theme.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| themes | `internal/templates/themes/edit_form_theme.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| themes | `internal/templates/themes/table_themes.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| yearlevels | `internal/templates/yearlevels/add_form_yearlevel.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| yearlevels | `internal/templates/yearlevels/delete_form_yearlevel.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| yearlevels | `internal/templates/yearlevels/edit_form_yearlevel.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| yearlevels | `internal/templates/yearlevels/table_yearlevels.html` | OK | OK | Conserver les conventions existantes ; actions et branches inspectées. |
| years | `internal/templates/years/add_form_year.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| years | `internal/templates/years/delete_form_year.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| years | `internal/templates/years/edit_form_year.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| years | `internal/templates/years/table_years.html` | P1 — ancienne UX visible | OK | Remplacement des listes/formulaires/confirmations hérités ; état vide et actions cohérentes. |
| Transversal | Titres et erreurs des handlers | P2 — incohérence mineure | OK pour les textes applicatifs traités | Traduction des chaînes visibles ; statuts et conditions inchangés. |
| Authentification | E-mail de réinitialisation et validation inscription | P2 — incohérence mineure | OK | Français ; même politique et même envoi. Aucun e-mail envoyé pendant l’audit. |
| Transport HTTP | Réponses standard 404/CSRF et erreurs techniques dynamiques | P2 — incohérence mineure | Limite documentée | Format texte/framework conservé ; pas de nouveau gestionnaire d’erreurs. |
| Documents | PDF d’examen, copies corrigées, notes et prévisualisations | OK | Conservé | Liens et présentation des pages inspectés ; mise en page et calculs des documents hors modification. |

## 6. Écarts trouvés avant la reprise

- `home/*.html` : P1, accueil et authentification visuellement hérités, navigation repliable incorrectement reliée, champs sans associations explicites. Cartes, liens de retour, labels et navigation réparés.
- `years/*.html`, `periods/*.html` : P1, anciens tableaux/formulaires et confirmations ; cartes, tables responsive, états vides et hiérarchie des actions.
- `marking/table_marking.html`, `students/add_csv_form_student.html` : P1/P2, zone de dépôt personnalisée et champ fichier caché ; bouton natif, champ visible, focus Bootstrap et annonce des erreurs. Flux et validation existants conservés.
- `marking/progress_marking.html`, `errors/question_feature_error.html` : P1, progression et erreur héritées ; cartes, textes français et retours.
- Formulaires `questions`, `answers`, `altquestions`, `altanswers`, `images`, `altimages` : P2, ordre visuel/clavier et boutons secondaires ; action principale suivie d’Annuler, groupes flex-wrap. Les suppressions de questions/réponses étaient déjà modernes et ont été explicitement conservées.
- `qcmquestions/add_form_qcm_question.html` : P2, bouton vert et badges risquant de déborder ; bouton principal et texte repliable.
- `dashboard/dashboard.html`, `home/layout.html` : P2, accès direct au contenu manquant ; lien d’évitement et cible focalisable.
- Handlers et mailer : P2, titres et erreurs anglais ; traduction de présentation seulement.
- `home/resetpassword.html` : P3, template invalide non référencé ; suppression après recherche des consommateurs. Aucun endpoint supprimé.

## 7. Travail déjà réalisé avant la reprise

Les corrections ci-dessus étaient présentes, ainsi que les pluriels de la revue, l’accessibilité de quelques icônes/liens et la confirmation de retrait d’une classe. Deux attentes de tests de rendu avaient été adaptées : bouton natif CSV à la place du tabindex manuel ; inventaire CSRF passant de 73 à 72 formulaires après suppression du template orphelin. La vérification CSRF de chaque formulaire actif est conservée.

## 8. Travail réalisé pendant cette reprise

- Relecture du diff et comparaison aux copies initiales, conservation des changements corrects.
- Traduction des deux erreurs de `internal/handlers/register/validation.go` encore en anglais : nom d’utilisateur et adresse e-mail. Expressions de validation inchangées.
- Parcours Chromium des pages réelles, contrôles DOM/mobile, captures représentatives et clavier des imports.
- Contrôle des états vides avec un compte temporaire, résultat et revue sur copie de données, exécution complète des vérifications projet.
- Finalisation du présent rapport avec les catégories et limites effectivement observées.

## 9. Cohérence finale

Navigation et titres en français ; action principale distincte des retours/annulations ; accès à la suppression en contour rouge et confirmation rouge explicitement nommée. Les tables conservent leurs conteneurs responsive et des états vides contextualisés. Les formulaires emploient labels, espacements Bootstrap et groupes d’actions repliables. Les messages applicatifs traduits restent utiles sans exposer inutilement les détails des logs.

Les champs de fichier restent accessibles directement ; les zones de dépôt sont des boutons activables au clavier. L’ordre des actions dans le DOM correspond à leur présentation. Les couleurs ne constituent pas leur seul libellé. Les limites de largeur des cartes et les dimensions dynamiques de progression restent en place lorsqu’elles servent le rendu : pas de nettoyage CSS général.

## 10. Éléments volontairement non modifiés

Métier, correction, scores, règles de validation, workflow de génération/revue/régénération, requêtes, migrations, permissions, routes et formats. Les modifications SQL et de persistance visibles dans le diff global sont préexistantes, pas produites par ce jalon. La validation du mot de passe reste en octets conformément à l’implémentation ; aucune nouvelle contrainte HTML n’a été inventée.

Les réponses HTTP standard et certains messages techniques dynamiques restent au format existant. Les remplacer par des pages HTML communes demanderait de revoir les contrats d’erreur ; ce travail n’est pas improvisé dans ce jalon. Pas de redesign des PDF, de nouveau composant, de dépendance frontend ni de refactoring partagé.

## 11. Tests

| Vérification | Résultat |
| --- | --- |
| `git diff --check` initial, reprise et final | Succès, aucune erreur de whitespace. |
| `go test ./...` final | Succès, code 0. |
| `./scripts/check.sh` final | Succès, code 0 : modules, replay migrations sur DB temporaire, gofmt, vet, tests, build et diff-check. |
| Comparaison lexicale Go aux copies initiales | 329 fichiers de production identiques hors chaînes littérales et commentaires ; aucun changement d’instruction/condition. |
| Comparaison des fichiers `db/` et `internal/db/` au début du jalon | Identiques ; changements du diff Git exclusivement préexistants. |
| Chromium, 75 pages internes, 1280 px et contrôle DOM à 390 px | Un h1 par page, aucun champ inspecté sans label, aucun bouton vide sans nom accessible, aucun débordement global à 390 px. |
| Accueil, à propos, login, inscription, demande/reset, succès | 7 pages publiques rendues ; pas de débordement ni champ sans label à 390 px. |
| États vides | Questions, élèves, QCM, évaluations, correction, années, périodes : explications et actions présentes. Inscription et connexion du compte temporaire réussies. |
| Résultat, revue, génération, progression | Résultats de correction et génération terminée affichés ; revue en attente avec image chargée ; progression de correction affichée. |
| Clavier et navigation | Menu mobile ouvert (aria-expanded=true) ; Entrée et Espace activent chacun les boutons d’import PDF et CSV. |

Environnement de rendu : serveur compilé lancé depuis `runtime/diagnostics/ux-modernization-20260905`, avec DB copiée de `testdata/real/app.db` et assets copiés de `runtime/real/assets`. Aucune utilisation des liens runtime pointant vers les originaux. Les tests de reprise ont créé un compte jetable ; une revue a été retirée et un statut PDF rendu transitoire **uniquement dans cette copie**, pour exposer les branches de présentation sans nouvelle correction. Aucun e-mail externe envoyé.

Captures examinées : dashboard large, formulaire d’évaluation étroit, suppression de question large et revue étroite. D’autres captures ont servi aux contrôles de génération, QCM, upload, résultat et progression. Les captures privées et scripts de contrôle ne sont pas ajoutés au dépôt.

Les premiers échecs intermédiaires concernaient trois attentes de rendu : titre pluralisé, bouton CSV et nombre de formulaires. Le titre Go a été conservé et le singulier traité dans le template ; les deux adaptations de tests sont strictement liées au HTML voulu. Le passage final complet réussit.

## 12. Fichiers modifiés

### Modifications du jalon présentes avant la reprise

- `internal/handlers/about/handler.go`
- `internal/handlers/about/view.go`
- `internal/handlers/altAnswers/handlers.go`
- `internal/handlers/altImages/handlers.go`
- `internal/handlers/altPreview/handlers.go`
- `internal/handlers/altQuestions/handlers.go`
- `internal/handlers/answers/handlers.go`
- `internal/handlers/classCodes/handlers.go`
- `internal/handlers/classCodes/viewData.go`
- `internal/handlers/dashboard/handler.go`
- `internal/handlers/difficulties/handlers.go`
- `internal/handlers/errorsmessages/handlers.go`
- `internal/handlers/exams/handlers.go`
- `internal/handlers/generateExams/cleanupFailedExamGeneration.go`
- `internal/handlers/generateExams/handlers.go`
- `internal/handlers/home/handler.go`
- `internal/handlers/home/view.go`
- `internal/handlers/images/handlers.go`
- `internal/handlers/login/handler.go`
- `internal/handlers/login/middlewares.go`
- `internal/handlers/login/view.go`
- `internal/handlers/logout/handler.go`
- `internal/handlers/marking/handlers.go`
- `internal/handlers/periods/handlers.go`
- `internal/handlers/points/handlers.go`
- `internal/handlers/preview/handlers.go`
- `internal/handlers/qcm/handlers.go`
- `internal/handlers/qcmPreview/handlers.go`
- `internal/handlers/qcmQuestions/handlers.go`
- `internal/handlers/questions/handlers.go`
- `internal/handlers/register/handler.go`
- `internal/handlers/register/view.go`
- `internal/handlers/resetpassword/handler.go`
- `internal/handlers/resetpassword/view.go`
- `internal/handlers/skills/handlers.go`
- `internal/handlers/studentClassCode/handlers.go`
- `internal/handlers/studentClassCode/viewData.go`
- `internal/handlers/students/handlers.go`
- `internal/handlers/students/viewData.go`
- `internal/handlers/students/viewData_test.go`
- `internal/handlers/subjects/handlers.go`
- `internal/handlers/themes/handlers.go`
- `internal/handlers/tools/checkRequest.go`
- `internal/handlers/tools/ownedMutation.go`
- `internal/handlers/tools/passwordValidation.go`
- `internal/handlers/tools/servePdfNamed.go`
- `internal/handlers/tools/serveUserImage.go`
- `internal/handlers/yearlevels/handlers.go`
- `internal/handlers/years/handlers.go`
- `internal/httpsecurity/csrf_test.go`
- `internal/mailer/mailer.go`
- `internal/templates/altanswers/add_form_alt_answer.html`
- `internal/templates/altanswers/edit_form_alt_answer.html`
- `internal/templates/altimages/add_form_alt_image.html`
- `internal/templates/altimages/delete_form_alt_image.html`
- `internal/templates/altimages/edit_form_alt_image.html`
- `internal/templates/altimages/table_alt_image.html`
- `internal/templates/altquestions/add_form_alt_question.html`
- `internal/templates/altquestions/edit_form_alt_question.html`
- `internal/templates/answers/add_form_answer.html`
- `internal/templates/answers/edit_form_answer.html`
- `internal/templates/dashboard/dashboard.html`
- `internal/templates/errors/question_feature_error.html`
- `internal/templates/exams/table_exams.html`
- `internal/templates/home/about.html`
- `internal/templates/home/home.html`
- `internal/templates/home/layout.html`
- `internal/templates/home/login.html`
- `internal/templates/home/register.html`
- `internal/templates/home/resetpassword.html` (supprimé)
- `internal/templates/home/showrequestresetpasswordform.html`
- `internal/templates/home/showresetpasswordform.html`
- `internal/templates/home/success.html`
- `internal/templates/images/add_form_image.html`
- `internal/templates/images/delete_form_image.html`
- `internal/templates/images/edit_form_image.html`
- `internal/templates/marking/progress_marking.html`
- `internal/templates/marking/review.html`
- `internal/templates/marking/success_marking_processing.html`
- `internal/templates/marking/table_marking.html`
- `internal/templates/periods/add_form_period.html`
- `internal/templates/periods/delete_form_period.html`
- `internal/templates/periods/edit_form_period.html`
- `internal/templates/periods/table_periods.html`
- `internal/templates/qcmquestions/add_form_qcm_question.html`
- `internal/templates/questions/add_form_question.html`
- `internal/templates/questions/edit_form_question.html`
- `internal/templates/studentClassCodes/delete_form_student_class_code.html`
- `internal/templates/students/add_csv_form_student.html`
- `internal/templates/students/add_form_student.html`
- `internal/templates/years/add_form_year.html`
- `internal/templates/years/delete_form_year.html`
- `internal/templates/years/edit_form_year.html`
- `internal/templates/years/table_years.html`

### Modifications ajoutées pendant la reprise

- `internal/handlers/register/validation.go`
- `docs/audits/2026-09-05-ux-modernization-audit.md` : créé avant reprise, complété pendant reprise.

### État préexistant exact, conservé comme référence

Cette liste précède le jalon UX. Les deux fichiers de production `internal/handlers/marking/handlers.go` et `internal/handlers/tools/servePdfNamed.go` reçoivent aussi des traductions UX ; leurs changements fonctionnels préexistants sont conservés. Les autres fichiers préexistants restent inchangés par ce jalon.

```text
 M db/query/markingJobs.sql
 M db/query/markingReviews.sql
 M internal/db/markingJobs.sql.go
 M internal/db/markingReviews.sql.go
 M internal/db/markingReviews_test.go
 M internal/db/models.go
 M internal/handlers/marking/handlers.go
 M internal/handlers/marking/handlers_test.go
 M internal/handlers/marking/hybridLifecycle_test.go
 M internal/handlers/marking/review_test.go
 M internal/handlers/tools/servePdfNamed.go
 M internal/handlers/tools/servePdfNamed_test.go
?? db/migrations/0044_add_marking_source_filename.sql
?? docs/audits/6e-class-test-preparation-final.md
?? docs/audits/6e-class-test-preparation.md
?? docs/audits/alt-image-parent-contract-fix.md
?? docs/audits/ci-implementation.md
?? docs/audits/exam-delete-history-protection.md
?? docs/audits/exam-failed-generation-cleanup.md
?? docs/audits/exam-failed-generation-lifecycle.md
?? docs/audits/exam-forms-ux.md
?? docs/audits/exam-generated-immutability.md
?? docs/audits/exam-generation-preflight.md
?? docs/audits/exam-generation-ux.md
?? docs/audits/exam-generation-view-data-refactor.md
?? docs/audits/exam-list-ux.md
?? docs/audits/exam-name-contract.md
?? docs/audits/exam-parent-delete-error-handling.md
?? docs/audits/exam-pdf-stable-access.md
?? docs/audits/exam-view-data-refactor.md
?? docs/audits/exams-audit.md
?? docs/audits/exams-closure-audit.md
?? docs/audits/exams-final-closure.md
?? docs/audits/global-csrf-protection.md
?? docs/audits/goose-full-replay-fix.md
?? docs/audits/image-context-refactor.md
?? docs/audits/image-delete-failure-tests.md
?? docs/audits/image-maintenance-cli.md
?? docs/audits/image-orphan-purge.md
?? docs/audits/image-storage-consistency-audit.md
?? docs/audits/image-storage-consistency-scanner.md
?? docs/audits/image-upload-parent-prevalidation.md
?? docs/audits/images-audit.md
?? docs/audits/legacy-real-corpus-mapping-audit.md
?? docs/audits/marking-admission-capacity.md
?? docs/audits/marking-aligned-pages-production.md
?? docs/audits/marking-ambiguity-calibration-expanded.md
?? docs/audits/marking-ambiguity-calibration-final.md
?? docs/audits/marking-ambiguity-calibration.md
?? docs/audits/marking-ambiguity-crop-review-preparation.md
?? docs/audits/marking-ambiguity-delta5-activation.md
?? docs/audits/marking-ambiguity-review-design.md
?? docs/audits/marking-artifacts-regeneration.md
?? docs/audits/marking-audit.md
?? docs/audits/marking-corrected-not-modern.md
?? docs/audits/marking-final-closure-audit.md
?? docs/audits/marking-historical-reference-design.md
?? docs/audits/marking-job-generation-binding.md
?? docs/audits/marking-recovery-startup.md
?? docs/audits/marking-reference-consumption.md
?? docs/audits/marking-reference-contract.md
?? docs/audits/marking-reference-production.md
?? docs/audits/marking-result-persistence-design.md
?? docs/audits/marking-result-production-persistence.md
?? docs/audits/marking-result-schema.md
?? docs/audits/marking-review-crop-endpoint.md
?? docs/audits/marking-review-finalization.md
?? docs/audits/marking-review-post-prg.md
?? docs/audits/marking-review-read-page.md
?? docs/audits/marking-review-result-page.md
?? docs/audits/marking-review-schema.md
?? docs/audits/marking-review-transactional-recalculation.md
?? docs/audits/marking-review-ux-design.md
?? docs/audits/marking-runtime-result.md
?? docs/audits/marking-success-retention.md
?? docs/audits/modern-fresh-install-smoke.md
?? docs/audits/modern-paper-marking-smoke-result.md
?? docs/audits/pedagogical-reference-data-audit.md
?? docs/audits/pedagogical-reference-data-closure.md
?? docs/audits/points-contract-fix.md
?? docs/audits/points-ux.md
?? docs/audits/qcm-closure-audit.md
?? docs/audits/qcm-composition-ux.md
?? docs/audits/qcm-concurrent-order-implementation.md
?? docs/audits/qcm-context-refactor.md
?? docs/audits/qcm-delete-semantics.md
?? docs/audits/qcm-final-fixes.md
?? docs/audits/qcm-forms-ux.md
?? docs/audits/qcm-list-ux.md
?? docs/audits/qcm-position-db-implementation.md
?? docs/audits/qcm-preview-reference-order.md
?? docs/audits/qcm-question-move-implementation.md
?? docs/audits/qcm-question-order-audit.md
?? docs/audits/qcm-question-selector-ux.md
?? docs/audits/qcm-ux-audit.md
?? docs/audits/question-content-uniqueness-fix.md
?? docs/audits/reference-delete-error-handling.md
?? docs/audits/students-closure-audit.md
?? docs/audits/subjects-ux-pilot.md
?? docs/audits/text-reference-ux-rollout.md
?? docs/audits/typst-image-path-fix.md
?? docs/debug-marking-artifacts-regeneration.md
?? docs/debug-marking-review-scoring.md
?? docs/improve-corrected-pdf-visual-semantics.md
?? docs/reports/2026-09-04-marking-artifact-filenames.md
?? internal/db/markingReviewReadModel_test.go
?? internal/handlers/tools/drawMarking_test.go
?? internal/handlers/tools/markingAmbiguityCalibrationBatch_integration_test.go
?? internal/handlers/tools/markingArtifactFilename.go
?? internal/handlers/tools/markingArtifactFilename_test.go
?? internal/handlers/tools/markingArtifactsGenerator_test.go
?? reset.sh
```

## 13. Limites restantes

- Réponses HTTP standard 404/CSRF et erreurs techniques dynamiques : certaines restent brutes ou anglaises ; pas de modification de leur contrat dans ce jalon.
- Contrôle visuel représentatif sous Chromium, sans certification WCAG complète, lecteur d’écran, autres navigateurs ou test exhaustif de chaque longueur de contenu.
- Les branches transitoires de génération, toutes les erreurs réseau et la régénération réelle des artefacts ne sont pas toutes déclenchées dans le navigateur ; leurs templates et les tests existants ont été inspectés/exécutés. Le test n’affirme pas une exécution exhaustive des workflows métier.
- Les PDF eux-mêmes n’ont pas été redessinés ou comparés pixel par pixel.

Vérification finale : les empreintes SHA-256 des 1 132 fichiers historiques/corpus sont identiques (0 modification). Les 137 fichiers de `db/` et `internal/db/` comparés au début du jalon sont identiques. Les modifications et fichiers non suivis préexistants sont conservés.

Le serveur et le navigateur de test sont arrêtés. Suppression effectuée des seuls éléments temporaires du jalon : `runtime/diagnostics/ux-modernization-20260905/` (DB/assets copiés, profils navigateur), `/tmp/lazymarking-ux-0m2np7r1/` (copies initiales, scripts, binaire, captures et journaux) et `/tmp/lazymarking-ux-location`. Aucune donnée privée de contrôle n’est conservée dans Git.

## 14. Verdict

**UX modernisée à l'exception de quelques points documentés**.

Les templates actifs inspectés ne présentent plus les surfaces manifestement héritées identifiées au départ. L’exception concerne les réponses standard/techniques hors pages applicatives et les limites de validation ci-dessus. Le travail préexistant a été conservé ; aucune amélioration métier n’a été introduite.
