# P4 — Restitution temporaire des copies corrigées (23 septembre 2026)

## 1. Analyse du workflow existant

Une évaluation `exams` est générée en `exams_generated`. Chaque élève reçoit une ligne `student_exam`, reliée à cette génération, ainsi qu'un snapshot JSON immuable `student_exam_content` et des repères de page `student_exam_page_content`. Le PDF des sujets est assemblé avec Typst, des QR techniques contenant `student_exam_id` et `page_exam`, puis `pdfunite`. Ces QR servent uniquement à reconnaître les pages lors de la correction et ne confèrent aucun accès public.

Chaque import est un `marking_jobs` rattaché à une génération. Ses `marking_copy_results`, `marking_question_results`, détections et éventuelles décisions de revue portent les résultats. `ListCurrentExamResultsForGeneration` sélectionne, pour chaque `student_exam`, la correction issue du dernier import réussi en donnant priorité à une copie corrigée ; un import de rattrapage avec une copie absente ne fait donc pas disparaître la correction antérieure. Les revues encore en attente bloquent la finalisation individuelle.

Le PDF corrigé de l'import est assemblé à partir de scans alignés, annotés par `DrawMarking`, puis réunis avec une table des matières. Le bilan cumulé P3 est un autre PDF Typst, recalculé à la demande, avec les résultats et statistiques de la génération. Aucun de ces PDF collectifs n'est exposé sur les routes publiques P4.

## 2. Modèle de données et migration

La migration `0048_student_copy_access.sql` crée `student_copy_access`, avec une ligne maximum par `student_exam` : token public unique, hash du code, état (`unpublished`, `published`, `revoked`), dates de publication et d'expiration, compteur d'échecs et verrou temporaire. La clé étrangère suit la suppression de la copie. Les accès sont créés lors du premier téléchargement des coupons ou de la première publication, seulement pour une génération réussie et appartenant au professeur. La migration ne crée aucun accès historique automatiquement.

## 3. Token, code et décisions de sécurité

Le token d'URL est produit par 32 octets aléatoires cryptographiques et encodé en base64url (43 caractères). Il ne contient aucun identifiant SQL. Le code personnel de 12 caractères, présenté en deux groupes de six, est dérivé du token par HMAC avec `SESSION_KEY`, dans un domaine distinct de celui du cookie de session P4 ; seul son hash bcrypt est enregistré. Cette dérivation permet de réimprimer un coupon sans conserver le code en clair. La vérification accepte le code avec ou sans tiret et verrouille l'accès pendant 15 minutes après cinq échecs. Le message d'échec ne distingue pas un code faux d'un verrouillage.

Après validation, un cookie HTTP-only signé, limité au token, expire en 20 minutes. Le serveur revérifie à chaque téléchargement l'état, la date et la correction finale courante, puis une seconde fois après le rendu PDF afin qu'une révocation ou une nouvelle revue intervenue pendant le rendu prévale. Les réponses publiques et les PDF portent `Cache-Control: no-store, private` et `Referrer-Policy: no-referrer`. Les routes professeur gardent leur authentification, leur contrôle d'appartenance et le CSRF global. Les codes ne sont ni mis dans les URL ni écrits dans les journaux.

`APP_BASE_URL` fournit l'URL imprimée et encodée dans le QR. Une URL HTTPS est exigée hors `localhost` et `127.0.0.1`. En exploitation, elle doit pointer vers l'adresse publique accessible aux élèves. `SESSION_KEY` doit rester stable pour permettre la réimpression de coupons déjà produits ; une rotation ne rend pas les anciens codes invalides (leurs hashes restent vérifiables), mais empêche leur réimpression à l'identique.

## 4. Publication, expiration, révocation et rattrapage

Le scan et la validation du code fonctionnent dès la distribution du coupon. Avant publication, la page indique que la correction n'est pas disponible. Le bouton professeur « Rendre les copies corrigées disponibles » publie uniquement les copies dont le résultat courant est corrigé, noté et sans revue en attente. Une copie techniquement corrigée n'est donc jamais publiée automatiquement.

Chaque publication individuelle ouvre sa propre fenêtre de 14 × 24 heures, à partir de l'instant de publication. Le contrôle d'expiration est effectué côté serveur. Une copie non vue ou non finalisée reste non publiée ; après son rattrapage, le professeur peut répéter l'action groupée et déclencher alors sa fenêtre de 14 jours, sans reprendre la date des autres élèves. La révocation est possible avant ou après publication. La réouverture d'une copie finale révoquée ou expirée démarre une nouvelle fenêtre de 14 jours, sans modifier la correction.

## 5. Coupons et interface professeur

Un PDF de coupons est accessible depuis la confirmation de génération et depuis le bilan de correction. Il contient quatre bandes horizontales pleine largeur par page A4, séparées par une ligne de coupe. L'ordre suit le nom puis le prénom selon la collation française, avec un coupon par copie. Chaque bande comprend nom, évaluation, classe, QR vers l'URL propre à la copie, code personnel, URL de secours complète et durée d'accès. Le PDF est compilé réellement avec Typst et contient des QR PNG générés pour ces URL.

Le bilan de génération affiche le bouton de publication, l'état et la date limite de chaque accès, ainsi que les actions de fermeture et de réouverture. La publication peut être relancée après un rattrapage. Le PDF cumulé professeur reste accessible et conserve son rôle d'archive.

## 6. Page publique et PDF individuel

`/copies/{token}` demande le code, puis affiche un état simple sur smartphone. L'identité et le nom de l'évaluation ne sont affichés qu'après validation du code et lorsque la copie est disponible. `/copies/{token}/pdf` exige en plus la session signée, l'état publié non expiré et une correction finale courante.

Le téléchargement individuel appelle `regenerateCorrectedCopy`, la même fonction que le moteur de régénération des PDF corrigés professeur. Elle reprend le snapshot, les réponses effectives après revue, les points persistés, les scans alignés et `DrawMarking`. Seul le résultat sélectionné pour le token est rendu dans un espace temporaire isolé ; aucun PDF de classe n'est lu ou extrait. Les points et annotations correspondent donc à la correction actuelle. Le rendu temporaire est supprimé après la réponse.

## 7. Compatibilité et limites

Les générations anciennes réussies sont compatibles si leurs snapshots, pages alignées et résultats sont encore présents. Elles nécessitent une activation explicite par téléchargement de nouveaux coupons ou publication ; aucun secret faible n'est inventé rétroactivement. Un ancien résultat sans données suffisantes pour reconstruire la copie individuelle reste indisponible au téléchargement et reçoit une erreur temporaire, sans ouvrir le PDF collectif.

La pile des coupons est triée par nom/prénom. Le PDF de sujets préexistant assemble ses fichiers par nom technique ; son ordre physique n'est pas garanti identique à celui des coupons. Harmoniser cet ordre impliquerait un changement de la chaîne de génération hors P4. La réimpression dépend de la stabilité de `SESSION_KEY`, comme indiqué ci-dessus. Il n'y a ni compte élève, ni espace permanent, ni lien avec les futurs decks P5.

## 8. Tests et validations

Tests ajoutés : accès valide/inconnu, code correct/faux, cinq tentatives et déverrouillage, état avant correction, publication individuelle, expiration exacte, révocation avant/après publication, réouverture, appartenance professeur et actions de l'interface, refus du PDF sans session ou avec le token d'une autre copie, rattrapage tardif et fenêtre indépendante. Les tests PDF réels compilent cinq coupons sur deux pages, vérifient noms accentués, nom long, ordre, codes et URL, puis extraient et décodent les cinq QR effectivement intégrés au PDF. Deux PDF individuels distincts d'une page sont générés avec annotations visibles. L'inventaire CSRF couvre les nouveaux formulaires.

Commandes exécutées et résultats :

| Commande | Résultat |
| --- | --- |
| `go test ./...` | Réussi après mise à jour de l'inventaire CSRF (79 formulaires). |
| `go test -count=1 ./...` | Réussi. |
| `go test -race ./...` | Réussi, aucune course signalée. |
| `git diff --check` | Réussi. |
| `go test ./internal/handlers/tools -run 'TestStudent(Coupons\|Corrected)' -count=1 -v` | Réussi : Typst, PDF et raster réels. |
| `go test ./internal/handlers/marking -run 'TestStudent' -count=1 -v` | Réussi : routes, cycle de vie et rattrapage. |
| `GOCACHE=/tmp/lazymarking-go-cache ./scripts/check.sh` | Réussi : modules, migrations jusqu'à 48, gofmt, `go vet`, tests, build et diff. |

Typst provient de `/snap/bin/typst` dans cet environnement ; les tests PDF ont été exécutés dans le contexte disponible sans sortie supplémentaire du sandbox. Le script de contrôle rejoue les migrations, vérifie format, `go vet`, tests, build et diff.

## 9. Avant P5

P5 peut construire ses decks collectifs à partir de l'analyse P3 et leur attribuer leurs propres liens publics. Les tokens et coupons P4 restent strictement associés à une copie individuelle et ne servent pas d'identité élève réutilisable.
