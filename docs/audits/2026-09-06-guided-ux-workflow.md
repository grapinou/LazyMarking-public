# Parcours guidé et accompagnement UX

## 1. Objectif

Guider le professeur de la préparation pédagogique à la correction, en conservant Bootstrap, les templates Go et la direction graphique actuelle. Aucun wizard, aucune architecture supplémentaire.

## 2. État initial

- `git status --short` : seulement `?? reset.sh`.
- `git diff --check` : OK.
- Trois derniers commits : `847480f`, `568204d`, `f993831`.
- `git ls-remote public refs/heads/main` confirme `847480f8f4dcd3f176fe6674110a8ef75dce8080`, identique au HEAD local.
- Aucun AGENTS.md applicable dans le dépôt.
- `reset.sh` n’a été ni modifié, ni ajouté, ni supprimé. Aucun commit ni push.

Audit préalable : handlers et templates about/register/dashboard/questions, listes des six paramètres, Élèves, évaluations et producteurs Typst. La convention réutilisée est l’encadré Bootstrap `alert alert-info` turquoise de la page Élèves, avec explication et action directe. Les cartes, boutons, grilles responsive et liens existants sont conservés.

## 3. `/about`

Quatre cartes numérotées et illustrées : Préparer, Composer, Évaluer, Corriger. Chaque carte présente une courte chaîne visuelle et son utilité. Modularité, réutilisation des questions dans plusieurs QCM et des QCM dans plusieurs évaluations/classes sont explicites.

Les affirmations ont été confrontées à `GetQCMQuestionsAnswersCtx`, `BuildQuestionCtx`, `GetQuestionAnswerCtx` et `GetAltQuestionAltAnswerCtx` : mélange des questions/réponses, choix d’une formulation avec réponses dans une famille. Aucune promesse de copies toutes différentes ni de sélection arbitraire d’un sous-ensemble de questions.

## 4. Aide connectée

Nouvelle route authentifiée `/dashboard/help`, entrée « Aide » dans la navigation commune. Même représentation que la page publique, avec actions vers la banque, les QCM, les évaluations et la correction. Les résultats sont présentés comme la suite de la correction, sans inventer une route de résultats indépendante.

## 5. Inscription et erreurs

Formulaire complet avec labels, champs obligatoires, e-mail typé et explications de l’identifiant et du mot de passe. Suppression du JavaScript qui retirait silencieusement les guillemets saisis.

Les champs manquants, validations rejetées, erreurs de hachage et échecs de création réaffichent le formulaire HTML avec une alerte et le statut HTTP existant. Identifiant et e-mail sont conservés et échappés par html/template. Le mot de passe n’est jamais réinjecté ; un message demande de le saisir à nouveau.

La règle reste **12 à 72 octets**, sans conversion en nombre de caractères Unicode. La formulation destinée au professeur donne la plage de 12 à 72 caractères pour lettres non accentuées, chiffres et signes courants, en précisant que les accents et émojis comptent davantage. Ce message partagé bénéficie aussi au formulaire utilisant le même validateur ; aucune règle n’est modifiée. Les erreurs de protocole/sécurité, notamment CSRF, restent hors du traitement des validations utilisateur.

## 6. Parcours de création d’une question

Le handler charge les six listes appartenant à l’utilisateur. La banque affiche leur disponibilité et une action de création pour chaque liste vide. Les boutons de création conduisent au premier paramètre manquant ; l’accès direct au formulaire vérifie également cette condition.

Lorsque les six listes sont renseignées, le bouton ouvre normalement le formulaire. Aucun changement des contrôles de création ni des relations SQL.

## 7. Navigation des paramètres

Ordre proposé : Matières → Thèmes → Niveaux → Compétences → Difficultés → Points → Créer une question. Ces listes sont indépendantes : cet ordre est pédagogique, pas une dépendance relationnelle. L’encadré l’explique explicitement.

Chaque liste conserve ses actions habituelles et offre Retour aux questions, Précédent et Suivant selon sa position. Après une création, le retour existant à la liste donne accès à l’étape suivante. Pas de parcours imposé, de session de wizard ni de nouvelle persistance.

## 8. Cartes de prérequis

Convention turquoise réutilisée pour la banque, les paramètres et le formulaire d’évaluation. Les états ne dépendent pas d’exemples ou de données globales : la banque et le formulaire évaluent les listes de l’utilisateur.

Le sélecteur de questions d’un QCM propose également de préparer une question dans la banque lorsqu’aucune question n’est disponible. La page Élèves reste la référence visuelle et conserve son fonctionnement.

## 9. Évaluations

La route réelle est **`/dashboard/exams/add`**, et non `/exams/add`. Aucun alias n’a été introduit. Le chemin abrégé non enregistré aboutit actuellement à l’accueil via le routage existant ; les vérifications finales ciblent bien le formulaire réel.

Carte avec disponibilité du QCM, de la classe, de l’année et de la période, et liens directs vers leurs formulaires de création. Le bouton de soumission reste absent si l’une des quatre collections manque.

Les élèves et les questions avec réponses sont expliqués comme nécessaires à la **génération des copies**, sans les transformer en nouvelles conditions de création d’une évaluation. Un lien conduit aux élèves pour les ajouter/rattacher à une classe. Aucun changement du traitement de génération ou de reconnaissance.

## 10. Données de démonstration

Les identités « John Doe », « dit la fritte du nord » et la classe « 666 » étaient réellement utilisées dans les prévisualisations question, variante et QCM. Remplacées par « Prénom », « Nom » et « Classe exemple ». Le modèle mémo utilise désormais « Élève exemple ».

Recherche des occurrences dans le code et les modèles actifs : aucune ancienne valeur ciblée restante. Les documents historiques et données de tests/runtime ne sont pas nettoyés arbitrairement. Aucune donnée réelle de DB supprimée ou modifiée.

## 11. Consigne PDF

Texte appliqué au modèle portrait et au producteur paysage, ainsi qu’aux assertions des tests :

> Répondez au stylo bleu ou noir. Coloriez complètement le/les cercle(s) correspondant(s) à votre/vos réponse(s).

Recherche des occurrences de l’ancienne consigne dans le code actif : aucune restante. La visibilité de la consigne selon le type de document reste inchangée, notamment son masquage dans la prévisualisation administrative d’une question. Aucun seuil, cercle ou algorithme modifié.

## 12. Tests

- `git diff --check` : OK.
- `go test ./...` : OK.
- `./scripts/check.sh` : OK (modules, migrations Goose, gofmt, vet, tests, build, diff).
- Tests ajoutés : rendu about/aide et liens, navigation des six listes vides, progression des prérequis jusqu’à l’état prêt, inscription avec champ manquant/e-mail invalide/mot de passe trop court, conservation des valeurs non sensibles et absence de mot de passe/message en octets.
- Tests adaptés : quatre prérequis d’évaluation absents et consigne Typst portrait/paysage. Les premiers échecs sur les anciens textes des alertes ont été corrigés en adaptant les attentes au nouveau rendu.

Validation réelle Chromium/Playwright avec Bootstrap chargé : about, register, erreur de validation, création d’un compte temporaire, connexion, dashboard, aide, banque vide, paramètres, formulaire d’évaluation sans prérequis. Parcours effectué jusqu’au formulaire question après création successive des six paramètres. Erreur de compte dupliqué également contrôlée : formulaire conservé, mot de passe vide.

Largeurs 1365 px et 390 px ; captures inspectées visuellement, absence de débordement horizontal, menu mobile Aide vérifié. Les liens du formulaire d’évaluation ont été ouverts et leurs destinations vérifiées.

Environnement : base neuve migrée dans `/tmp/lm-guided-runtime`, serveur compilé depuis le dépôt. `scripts/run-2026-2027.sh` a été inspecté mais pas exécuté pour éviter ses traitements de récupération/purge sur les données de l’utilisateur. Aucune écriture dans la base 2026-2027. Captures et scripts exploratoires dans `/tmp/lm-guided-browser*` (artefacts temporaires, pas une nouvelle suite E2E du dépôt).

Limite : vérification de la consigne dans la génération de sources Typst par les tests existants ; pas de nouveau cycle physique impression/scan ni de validation optique après impression.

## 13. Fichiers modifiés

- `docs/audits/2026-09-06-guided-ux-workflow.md`
- `internal/config/ref_memo_base_typst.txt`
- `internal/config/ref_qcm.txt`
- `internal/handlers/altPreview/handlers.go`
- `internal/handlers/dashboard/handler.go`
- `internal/handlers/dashboard/help_test.go`
- `internal/handlers/dashboard/route.go`
- `internal/handlers/exams/handlers_test.go`
- `internal/handlers/preview/handlers.go`
- `internal/handlers/qcmPreview/handlers.go`
- `internal/handlers/questions/handlers.go`
- `internal/handlers/questions/prerequisites_test.go`
- `internal/handlers/register/form_test.go`
- `internal/handlers/register/handler.go`
- `internal/handlers/register/view.go`
- `internal/handlers/tools/passwordValidation.go`
- `internal/handlers/tools/typstLandscapeContent.go`
- `internal/handlers/tools/typstProducerEscape_test.go`
- `internal/templates/dashboard/dashboard.html`
- `internal/templates/dashboard/help.html`
- `internal/templates/data/home.go`
- `internal/templates/data/question.go`
- `internal/templates/difficulties/table_difficulties.html`
- `internal/templates/exams/add_form_exam.html`
- `internal/templates/home/about.html`
- `internal/templates/home/register.html`
- `internal/templates/points/table_points.html`
- `internal/templates/qcmquestions/add_form_qcm_question.html`
- `internal/templates/questions/table_questions.html`
- `internal/templates/skills/table_skills.html`
- `internal/templates/subjects/table_subjects.html`
- `internal/templates/themes/table_themes.html`
- `internal/templates/yearlevels/table_yearlevels.html`

## 14. Points volontairement hors périmètre

Gestion de compte, page Mon compte, suppression de compte, modification du profil ou de l’e-mail : non réalisées. Consolidation des corrections, rattrapages et règles métier de correction : inchangés. Aucun modèle SQL, migration, permission ou architecture frontend modifié. Aucun refactoring général. Les modifications métier envisagées, notamment une longueur de mot de passe en caractères Unicode, ne sont pas réalisées.

## 15. Verdict

**Jalon terminé avec limites documentées**.

Parcours, accompagnement et contrôles navigateur réalisés. Limites : règle historique de longueur des mots de passe conservée, validation sur base temporaire plutôt que sur les données utilisateur, pas de nouveau cycle physique PDF/scan. Aucun commit ni push.
