# Protection CSRF globale

## État trouvé à la reprise

Le jalon était partiellement réalisé et non documenté. `go.mod`/`go.sum` contenaient déjà `github.com/gorilla/csrf v1.7.3`, `cmd/server/main.go` installait le middleware global, `internal/httpsecurity/csrf.go` et deux fichiers de tests existaient sans être suivis par Git, 73 formulaires unsafe possédaient déjà `{{ csrfField }}`, et les fonctions de rendu avaient été adaptées. Le rapport présent n'existait pas. Le README avait des modifications utilisateur préexistantes et n'a pas été touché.

## Contrat implémenté

- Bibliothèque : `github.com/gorilla/csrf v1.7.3` (avec mise à niveau indirecte de `github.com/gorilla/securecookie` de 1.1.1 à 1.1.2).
- Secret : `CSRF_AUTH_KEY`, exactement 32 octets, obligatoire au démarrage, sans valeur par défaut, et refusé s'il est identique à `SESSION_KEY` lorsqu'elle est définie. Les tests utilisent une clé synthétique locale.
- Configuration transport : `SESSION_SECURE` doit être explicitement un booléen et configure aussi `Secure` sur le cookie CSRF, conformément au mode HTTP/HTTPS existant.
- Cookie : `_lazymarking_csrf`, `Path=/`, `HttpOnly=true`, `Secure` configurable, `SameSite=Strict`, durée 7200 secondes.
- Middleware : `SecurityHeaders(CSRF(mux))`. Le CSRF enveloppe donc le routeur, l'authentification et les handlers métier ; il fournit le contexte/token lors des GET et refuse les méthodes unsafe avant l'authentification et le métier. Toutes les routes POST sont couvertes, sans validation recopiée dans les handlers.
- Méthodes : Gorilla valide toute méthode non sûre, donc POST, PUT, PATCH et DELETE. GET, HEAD, OPTIONS et TRACE sont sûres. Le routeur déclare actuellement 71 POST et aucune route PUT/PATCH/DELETE.
- Corps multipart : Gorilla lit le formulaire avant le handler. Une limite globale de 100 Mio est donc posée avant Gorilla sur les corps unsafe afin que l'analyse CSRF ne contourne pas la limite d'admission Marking ; les contrôles métier existants (`MaxBytesReader`, `ParseMultipartForm`, PDF, pages et dimensions) restent en place.
- Erreur : token absent, invalide ou incompatible avec le cookie donne 403 avec le seul message français « Cette requête de sécurité n'est plus valide. Rechargez la page puis réessayez. ».

`SameSite=Strict` reste une défense navigateur en profondeur. Le token CSRF apporte séparément une preuve explicite que la soumission provient d'un formulaire émis par l'application.

## Couverture navigateur

L'inventaire trouve 73 formulaires HTML unsafe dans 71 fichiers ; tous utilisent la fonction centrale `csrfField`. Cela couvre login, inscription, réinitialisation de mot de passe, logout, CRUD questions/QCM/examens/référentiels/élèves, CSV, images, génération d'examens et Marking.

Le login GET émet le cookie et le champ ; son POST sans token est refusé avant toute création de session. Logout est une route POST uniquement : GET retourne 405, POST sans token retourne 403 et POST valide invalide la session puis redirige. Les trois POST Marking sont globalement protégés : `/dashboard/marking/processing`, `/dashboard/marking/review/apply`, `/dashboard/marking/artifacts/regenerate`. Il n'existe aucune exemption POST et aucune route navigateur non couverte identifiée.

## GET mutatifs restant hors périmètre

Inventaire exact : **4** GET produisent des fichiers temporaires de prévisualisation puis redirigent vers leur lecture : prévisualisation question, question alternative, QCM portrait et QCM paysage. Ils ne modifient ni la base ni la session métier, mais créent/purgent un espace de travail utilisateur. Leur conversion en POST est une dette HTTP séparée, non nécessaire à la protection globale des méthodes unsafe. Logout n'en fait plus partie.

## Tests

Les tests middleware démontrent GET autorisé et champ émis, cookie et attributs, POST absent/invalide/valide, incompatibilité token-cookie, méthodes unsafe refusées avant le handler, message 403 neutre, configuration obligatoire et secret distinct, inventaire exhaustif des formulaires, et multipart valide.

Les tests de routes couvrent login et logout, les trois POST Marking et un CRUD Subject. Ils vérifient le refus sans token, l'acceptation d'un token valide à la frontière CSRF, et le passage multipart jusqu'à la frontière d'authentification existante. Le refus avec des dépendances métier nulles prouve que le handler métier n'est pas atteint ; les tests purs existants continuent de tester les comportements métier avec middleware absent.

## Inventaire et migration

- POST protégés : **71**.
- Exemptions POST : **0**.
- GET mutatifs restant : **4**, tous liés aux artefacts temporaires de prévisualisation.
- Migration DB/sqlc : aucune.

Fichiers créés pour ce jalon : `internal/httpsecurity/csrf.go`, `internal/httpsecurity/csrf_test.go`, `internal/httpsecurity/routes_integration_test.go`, ce rapport.

Fichiers modifiés pour ce jalon : `cmd/server/main.go`, `go.mod`, `go.sum`, `internal/handlers/dashboard/view.go`, `internal/handlers/login/view.go`, `internal/handlers/register/view.go`, `internal/handlers/resetpassword/view.go`, `internal/handlers/tools/renderMergeTemplate.go`, et les 71 templates contenant les 73 formulaires unsafe sous `internal/templates/` (altanswers, altimages, altquestions, answers, classcodes, dashboard, difficulties, exams, home, images, marking, periods, points, qcm, qcmquestions, questions, skills, studentClassCodes, students, subjects, themes, yearlevels et years).

Le README modifié avant ce jalon, les autres rapports non suivis, le binaire `app`, `reset.sh` et les tests Marking préexistants ne font pas partie de cette modification.

## Validation finale

- `./scripts/check.sh` : succès (modules, replay Goose 0001→0040, formatage, vet, tests, build et diff-check).
- `go test -race ./...` : succès.
- `git diff --check` : succès après finalisation du rapport.
