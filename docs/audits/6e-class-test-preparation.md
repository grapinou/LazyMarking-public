# Préparation du test réel 6e — arrêt sur bug bloquant

## Verdict

Préparation arrêtée le 2 septembre 2026 conformément à la consigne « bug bloquant : reproduire, identifier, STOP ». L'application interdit de créer, pour un même professeur, deux questions alternatives ayant exactement le même énoncé. Or les familles 2 et 3 demandent toutes deux l'énoncé exact `Quel instrument est représenté ?`. Aucun code, aucune migration et aucune donnée n'ont été contournés ou corrigés.

## Installation et configuration

- installation persistante : `/home/sighto/Documents/lazymarking-6e-test/` ;
- DB : `/home/sighto/Documents/lazymarking-6e-test/app.db` ;
- runtime : `/home/sighto/Documents/lazymarking-6e-test/runtime` ;
- binaire construit depuis le code courant : `/home/sighto/Documents/lazymarking-6e-test/lazymarking-server` ;
- DB SQLite créée neuve ; migrations Goose 0001 à 0040 appliquées avec succès ; `goose status` confirme les 40 migrations ;
- `runtime/db/data/app.db` est un lien vers la DB persistante ; `runtime/internal` référence les templates/configurations du dépôt ; stockage image privé sous `runtime/assets/images` ;
- configuration HTTP locale : `SESSION_SECURE=false` ; clés synthétiques distinctes de 32 octets ; aucun secret dans le dépôt.

Commande exacte de reprise :

```sh
cd /home/sighto/Documents/lazymarking-6e-test/runtime && \
SESSION_SECURE=false \
SESSION_KEY='<SESSION_KEY_PRIVEE>' \
CSRF_AUTH_KEY='<CSRF_KEY_PRIVEE>' \
/home/sighto/Documents/lazymarking-6e-test/lazymarking-server
```

URL : `http://127.0.0.1:8080/login`. Identifiant : `<IDENTIFIANT_TEST_PRIVE>`. Mot de passe : `<MOT_DE_PASSE_TEST>`.

## Parcours applicatif effectué

Toutes les écritures métier effectuées l'ont été par les routes HTTP, formulaires réels, cookies de session et jetons CSRF. Inscription, premier login, dashboard, logout, redirection vers login et second login ont réussi. Le compte reste utilisable.

Référentiels créés via l'application : matière `Sciences`, thème `Culture scientifique — 6e`, niveau `6e`, difficulté `Standard`, points `1`, compétence `Mobiliser ses connaissances scientifiques`, année `2026-2027`, période `Test septembre`.

État partiel au STOP, confirmé par SELECT read-only : 3 principales, 12 réponses principales, 2 alternatives, 8 réponses alternatives, exactement une réponse correcte dans chacune des cinq questions complètes. La famille 1 est complète. La famille 2 est complète et son image de balance a été uploadée via le formulaire. La principale de la famille 3 et ses quatre réponses ont été créées ; son alternative a été refusée. Aucun QCM, classe, élève, examen ou génération n'a ensuite été créé.

## Images

Cinq PNG synthétiques noir sur blanc ont été créés hors dépôt dans `/home/sighto/Documents/lazymarking-6e-test/assets/` : balance, éprouvette graduée, thermomètres 18 °C et 25 °C, circuit fermé. Une image (balance) a été réellement uploadée et stockée sous `runtime/assets/images/1_<IDENTIFIANT_TEST_PRIVE>_altQuestion_2_balance.png`, largeur applicative 32 %. Les autres n'ont pas été uploadées après le STOP.

## Bug bloquant

- symptôme reproductible : le POST authentifié et protégé CSRF de la seconde alternative `Quel instrument est représenté ?` redirige vers la page d'erreur ; log serveur : `UNIQUE constraint failed: alt_questions.content, alt_questions.user_id` ;
- cause identifiée : la migration `0029_scope_unique_constraints_by_user.sql`, lignes 105–116, reconstruit `alt_questions` avec `UNIQUE (content, user_id)`. Cette contrainte interdit le même énoncé dans deux familles distinctes du même compte. Le handler `internal/handlers/altQuestions/handlers.go`, lignes 160–169, transmet l'insertion puis convertit toute erreur DB en message générique ;
- priorité : P1 pour ce jalon, car le cahier des charges impose les textes exacts et interdit de corriger ou contourner le modèle ;
- fichiers concernés probables : `db/migrations/0029_scope_unique_constraints_by_user.sql`, schéma/queries SQL associés à `alt_questions`, et validation/gestion d'erreur dans `internal/handlers/altQuestions/handlers.go` ;
- code modifié : non ; DB modifiée directement : non.

## Étapes non exécutées après le STOP

QCM et composition, previews portrait/paysage, classes 6eA/6eB, 60 élèves, évaluations, générations, références modernes, contrôle d'individualisation, inspection/QR et PDFs : non exécutés. Marking, scans, OpenCV et review : non lancés.

## UX constatée

Un défaut UX : le message retourné pour cette contrainte (`Il ne peut pas exister deux fois la même réponse ou la question ne peut être vide.`) parle de « réponse » alors qu'il s'agit d'une question alternative et fusionne deux causes différentes. Aucun travail UX effectué.

## État Git

L'état initial contenait déjà `README.md` modifié, `app`, `reset.sh`, de nombreux rapports sous `docs/audits/` et deux tests non suivis. Tout a été préservé. L'état final ajoute uniquement ce rapport non suivi. Aucune DB, image, PDF, clé ou script de cette installation n'apparaît dans le dépôt.
