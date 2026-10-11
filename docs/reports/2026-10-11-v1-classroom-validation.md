# LazyMarking V1 — validation en classe et synchronisation du correctif

Date : 11 octobre 2026. **Correctif récupéré, testé et publié ; déploiement du
commit Git sur le Raspberry Pi et tag `v1.0.0` en attente.**

## Parcours réellement validé

Le responsable du projet confirme la validation en conditions réelles, le
11 octobre 2026, du parcours questions → QCM → examen → impression → correction
→ publication → consultation des copies par les élèves. Cette validation
concerne la version fonctionnelle du Pi, avec les modifications locales du
7 octobre. La mission n'a pas rejoué ce parcours sur la production.

Le futur tag représentera cette validation en classe, après confirmation du
déploiement et du fonctionnement du commit correspondant. Il ne constitue pas
une certification générale de toutes les configurations. Aucun chantier UX de
la feuille de route n'a été entrepris.

## Diagnostic initial et provenance

### Raspberry Pi : source du correctif

- `remi@dock` : échec SSH, code 255, `Temporary failure in name resolution`.
- `remi@dock.local` : connexion SSH réussie avec `BatchMode=yes`, sans écriture.
- Dépôt réel : `/home/remi/LazyMarking` ; branche `main`, suivant `origin/main`.
- HEAD : `ce7bc366233adc981a842a53bc9b2728c5e0f678` —
  `Fix answer circle detection tolerance`.
- Remote du Pi : `origin` vers
  `https://github.com/grapinou/LazyMarking-public.git` pour fetch et push.
- Index : aucune modification (`git diff --cached` vide).
- Fichiers suivis non indexés : les trois sources ci-dessous.
- Commits locaux non poussés : aucun selon
  `git log --branches --not --remotes` ; branche suivie sans avance/retard.
- Fichier non suivi : `internal/httpsecurity/csrf.go.bak`, inventorié seulement,
  sans lecture de contenu ni récupération sur le PC.

```text
 M internal/handlers/marking/studentAccess.go
 M internal/handlers/tools/securityHeaders.go
 M internal/httpsecurity/csrf.go
?? internal/httpsecurity/csrf.go.bak
```

Les dates de modification des sources sur le Pi sont le 7 octobre 2026 à
00:58:09, 00:26:48 et 00:19:41 CEST, respectivement. Ces métadonnées corroborent
le contexte fourni, sans prouver à elles seules la date ou l'auteur du patch.
Le correctif réside dans l'arbre de travail, pas dans un commit à cherry-picker.

Récupération limitée à `git diff HEAD --` sur ces trois chemins, relu puis
appliqué avec `git apply --check` sur le PC. Empreinte SHA-256 du patch original :

```text
aabb9438a7d8155d63e7fa4a185f72161dd981402c61f4ee31b49556f1c65097
```

Le patch original et les sorties de tests synthétiques sont conservés localement
dans `/tmp/lazymarking-v1-evidence.CQKR3R`, hors Git ; ce dossier temporaire
n'est pas une sauvegarde durable. Aucune base, aucun PDF d'élève, aucun contenu
de `runtime/`, `testdata/`, `.env`, sauvegarde ou journal réel n'a été transféré
depuis le Pi.

### Service et binaire en fonctionnement

`lazymarking.service` est `loaded`, `active`, `running`, depuis le
9 octobre 2026 à 01:47:42 CEST ; `ExecMainStatus=0`. Le PID principal 807 est
un wrapper Bash ; le serveur est son enfant PID 1010, exécutable
`/home/remi/LazyMarking/bin/lazymarking-server`.

Empreinte SHA-256 de l'exécutable réellement ouvert par ce processus :

```text
4cb6cb89a58c7cf56a1b236105eeb8f24d8474b058b18cd4e95bab842d6be55f
```

`go version -m /proc/1010/exe` indique Go 1.25.14, arm64,
`vcs.revision=ce7bc366233adc981a842a53bc9b2728c5e0f678` et
`vcs.modified=true`. **Le SHA de base du binaire est confirmé ; le détail des
modifications compilées et leur égalité avec les sources actuelles ne le sont
pas.** Un binaire `+dirty` n'identifie pas un commit Git déployé proprement.

Le service appelle `/home/remi/LazyMarking/scripts/run-2026-2027.sh`, identique
au fichier suivi à HEAD (blob `84d4b6dbc8b98f080e3d44c89fbc508fd67f7b7b`).
Ce script compile à chaque démarrage, conserve la clé de session et crée une
nouvelle clé CSRF ; le serveur appelle ensuite `OpenMigratedDB`. Un redémarrage
peut donc déclencher des migrations si le schéma est ancien. Aucun contenu de
`.env`, aucune ligne de journal et aucune donnée SQLite n'a été consulté.

Le contrôle final en lecture seule retrouve le même HEAD, le même statut, la
même empreinte du patch, les mêmes PID, la même date de démarrage et la même
empreinte binaire. **Pi non modifié par cette mission, aucun déploiement ni
redémarrage effectué.**

### PC et destinations Git

- Dépôt : `/home/sighto/LazyMarking` ; état initial propre, branche `main`.
- Branche de départ : `origin/main` à
  `ce7bc366233adc981a842a53bc9b2728c5e0f678`, après fetch réussi.
- Branche de travail : `fix/v1-publication-2026-10-11`.
- Branche de destination : `main`, confirmée comme branche par défaut par
  `git ls-remote --symref` et l'API GitHub pour les deux dépôts.
- `origin` : `https://github.com/grapinou/LazyMarking.git`, dépôt privé de
  référence ; fetch et push vers ce même dépôt.
- `public` : `git@github.com:grapinou/LazyMarking-public`, miroir public.
- `git fetch origin` et `git fetch public` réussis. Au début de la mission,
  les deux têtes distantes réelles sont encore exactement `ce7bc36` ; aucune
  divergence (`origin/main...public/main` : `0 0`).
- Les autres branches locales `refactor/project-review` et `smoke-public`
  ont été inventoriées et laissées intactes. Tags initiaux :
  `historical-regression-2026-08-26`, `reliability-baseline-2026-08-27`.
- Les métadonnées GitHub indiquent `main` non protégée et des droits de push.
  L'API détaillée de protection/règles du dépôt privé retourne HTTP 403 :
  inspection administrative complète indisponible, sans tentative de
  contournement. Les pushes ordinaires ont été acceptés.

Aucun script ou workflow automatique de synchronisation du miroir n'a été
trouvé dans les scripts, README et workflows suivis. Le miroir partage déjà
les mêmes objets et le même historique de `main` ; la livraison utilise donc
une intégration locale en fast-forward, un push explicite vers `origin`, puis
le même commit vers `public`, sans réécriture ni push forcé. Le remote `origin`
du Pi désigne le miroir, pas le dépôt privé : les noms des remotes ne sont pas
interchangeables entre les machines.

## Modifications intégrées

| Fichier | Changement |
| --- | --- |
| `internal/handlers/marking/studentAccess.go` | Politique `origin` sur le rendu HTML, l'entrée du handler de consultation et le PDF individuel, exactement comme sur le Pi. |
| `internal/handlers/tools/securityHeaders.go` | Politique générale `strict-origin-when-cross-origin`, exactement comme sur le Pi. |
| `internal/httpsecurity/csrf.go` | Reprise du paramètre variadique et de `csrf.TrustedOrigins`, alimenté par l'hôte et le port de `APP_BASE_URL`. Suppression du diagnostic temporaire du Pi. Validation renforcée de l'URL et des origines HTTPS mutantes. |
| `internal/handlers/marking/studentAccessSecurity_test.go` | Nouveau parcours coupon et publication à travers les middlewares réels, avec fixture synthétique. |
| `internal/handlers/marking/studentAccess_test.go` | Assertion de politique Referer sur un vrai PDF individuel synthétique. |
| `internal/handlers/tools/securityHeaders_test.go` | Assertion de la valeur exacte de la politique générale. |
| `internal/httpsecurity/csrf_test.go` | Configuration, HTTPS derrière proxy, rejets, cookies et absence de diagnostic sensible. |
| `docs/reports/2026-10-11-v1-classroom-validation.md` | Ce rapport et les étapes de déploiement différées. |

Le diagnostic supprimé journalisait méthode, chemin, raison CSRF, `Origin`,
`Host`, en-têtes de forwarding, `Referer` et `Sec-Fetch-Site`. Un chemin ou
Referer pouvait contenir un jeton de copie ; il n'est pas conservé dans le
correctif livré. Le fichier `.bak` du Pi reste à sa place, intact.

Deux durcissements ciblés accompagnent la reprise du Pi : `APP_BASE_URL` doit
être une URL HTTP(S) avec hôte, sans credentials, query ni fragment ; en mode
HTTPS, une requête mutante présentant `Origin` doit porter une origine HTTPS
bien formée. Gorilla 1.7.3 compare les origines de confiance par hôte : ce
second contrôle empêche l'ajout de l'hôte public d'accepter une origine HTTP.
Les en-têtes `X-Forwarded-*` n'élargissent jamais la liste de confiance.

La clé CSRF distincte, le cookie CSRF `Secure`/`HttpOnly`/`SameSite=Strict`,
la limite de corps et les vérifications de jeton restent obligatoires. La
session élève conserve sa signature liée à une copie et reste distincte de la
session professeur ; aucune modification de leurs règles ou des données.

### Confidentialité du Referer

La politique `origin` conserve seulement schéma, hôte et port ; les chemins
`/copies/<jeton>` et leurs paramètres ne partent pas vers des tiers. La politique
générale conserve l'URL complète seulement vers la même origine ; vers un tiers
HTTPS, elle transmet seulement l'origine et supprime le Referer lors d'une
descente vers HTTP. Référence :
[W3C, politiques origin et strict-origin-when-cross-origin](https://www.w3.org/TR/referrer-policy/).

Les handlers de copies imposent `origin` après les en-têtes généraux, y compris
sur leurs réponses 404. `origin` peut révéler l'origine du site à un tiers HTTP,
mais aucun chemin privé ni jeton. Le HTML élève ne charge pas de ressource
tierce ; la politique du Pi est conservée. Les réponses CSRF rejetées gardent
la politique générale, qui ne transmet pas leur chemin à une autre origine.
Les tests vérifient les en-têtes et simulent le Referer réduit à l'origine ;
aucune mesure de trafic d'un navigateur de production n'est revendiquée.

## Vérifications observées sur le PC

Environnement : Go 1.26.5 linux/amd64, OpenCV 4.12.0, Goose 3.27.3,
Typst 0.15.1, outils Poppler disponibles. Aucun outil de validation n'a été
installé ou exécuté sur le Pi.

| Commande | Résultat observé |
| --- | --- |
| `./scripts/check.sh` | Code 0 sur les sources finales : modules vérifiés, migrations 0001 à 0049 rejouées sur une base temporaire du PC, gofmt, vet, tests de tous les packages, build et diff check réussis. Certains packages inchangés utilisent le cache Go. |
| `git diff --check` et `git diff --cached --check` | Code 0, aucune erreur d'espacement. |
| `go test -count=1 -v -run 'Test(CSRF\|AllUnsafeHTML\|StudentCopy\|StudentCoupons\|SecurityHeaders)' ./internal/httpsecurity ./internal/handlers/marking ./internal/handlers/tools` | Code 0, sans cache ; tous les tests sélectionnés passent. `marking` : 2,166 s ; `tools` : 0,319 s. Les PDF de coupons et individuel synthétiques sont effectivement générés. |
| `go test -count=1 -v ./internal/httpsecurity` | Code 0 : configuration, jetons, cookies, formulaires, proxy et rejets passent. |
| `go test -race -count=1 ./internal/httpsecurity ./internal/handlers/marking ./internal/handlers/tools` | Code 0, aucune course détectée : 1,217 s / 32,961 s / 95,808 s. |
| `go test -race -count=1 -v ./internal/httpsecurity` | Code 0 sur l'ultime ajustement des méthodes sûres : 1,137 s, aucune course détectée. |

Les cas couvrent jeton absent/invalide, cookie absent ou étranger, origine ou
Referer étrangers, Referer absent en HTTPS, Referer HTTP, origine HTTP pourtant
sur l'hôte de confiance, origine mal formée, mauvais port, sous-domaine et
suffixe malveillant. Les requêtes rejetées ne mutent pas la publication ni
l'accès élève ; une session élève ne donne pas les droits professeur. Le
parcours teste accès avant/après publication et refus des PDF anonymes ou
d'une autre copie ; les tests existants couvrent révocation et expiration.

Le premier lancement ciblé et le premier `check.sh` ont retourné code 1 : le
nouveau test recopiait le jeton HTML sans décoder les entités, conduisant à un
403. Le test utilise maintenant `html.UnescapeString`, comme la lecture d'un
champ par le navigateur. Les relances finales ci-dessus sont vertes. Le build
du premier check n'avait pas été atteint ; celui du check final l'a été.

Aucune variable `LAZYMARKING_TEST_*` ou `LAZYMARKING_CALIBRATION_*` n'est
configurée. Les tests de scans historiques privés conditionnés par ces
variables sont ignorés, pas certifiés. La CI distante n'est pas annoncée
réussie par ce rapport ; les résultats ci-dessus sont locaux.

## Livraison Git

Commit correctif : **`58d4e01212f35c584c58ee68c5d7db66fc3e7523`**,
`Fix student copy publication CSRF headers`, parent `ce7bc36`.
Les sept fichiers de code et tests ont été ajoutés explicitement, sans
`git add -A`, sans secret ni donnée réelle.

Ce commit est déjà effectivement publié et vérifié par `git ls-remote` :

- dépôt privé `grapinou/LazyMarking` :
  `fix/v1-publication-2026-10-11` et `main` ;
- miroir `grapinou/LazyMarking-public` : `main` seulement.

Le rapport est versionné dans un commit documentaire suivant, sur la même
branche de travail, puis intégré en fast-forward à `main` et synchronisé aux
mêmes destinations. Le SHA final de livraison est fourni dans le résumé de
mission et peut être retrouvé avec :

```sh
git log -1 --format='%H %s' -- docs/reports/2026-10-11-v1-classroom-validation.md
git rev-parse main origin/main public/main
git status --short
```

Un fichier versionné ne peut contenir le SHA de son propre commit sans changer
ce SHA. Le commit documentaire ne modifie pas le code testé ; son SHA doit
être épinglé explicitement pour le déploiement. L'arbre PC est propre après le
commit correctif ; le contrôle final de statut et d'égalité des refs après
publication du rapport figure dans le résumé de livraison.

## Déploiement sûr préparé — aucune commande mutante exécutée sur le Pi

**Prérequis non satisfaits à ce stade : sauvegarde valide non vérifiée,
autorisation de déploiement/redémarrage non donnée et test de la version Git
sur le Pi non effectué.** Les étapes suivantes restent conditionnelles.

1. Épingler le SHA final livré ; le comparer au `main` du miroir public. Refaire
   le diagnostic en lecture seule du Pi. Tout changement supplémentaire,
   commit local ou divergence entraîne un arrêt et une nouvelle revue.
2. Faire vérifier par l'opérateur une sauvegarde privée et cohérente de
   l'évaluation complète : SQLite avec son état WAL, images, références PNG,
   scans alignés et originaux, `.env` et clés stables. Le README détaille les
   données à conserver ; une copie brute de SQLite active ne suffit pas.
   Vérifier l'intégrité sur une **copie de sauvegarde**, sa restauration isolée
   et le schéma 49 attendu. Aucun fichier de cette sauvegarde ne va au PC ou à
   GitHub. Ne pas lancer de migration. Si le schéma de production n'est pas
   déjà à jour, suspendre ce déploiement : le démarrage automatique ne doit pas
   devenir une migration implicite non revue.
3. Après autorisation, préserver sur le Pi dans un emplacement privé protégé
   les états Git, les différences indexées et non indexées, les sources locales
   ciblées, `csrf.go.bak` et le binaire fonctionnel identifié ci-dessus. Conserver
   aussi une référence Git de retour à `ce7bc36`. Vérifier les empreintes et la
   possibilité de restaurer les sources et le binaire avant toute modification.
4. Récupérer uniquement le code Git du miroir par `git fetch origin`, puis
   vérifier l'ascendance et le contenu du SHA épinglé. Ne pas faire `git pull`
   sur l'arbre sale. Pour l'état actuellement inventorié, préserver d'abord le
   patch original sur le Pi, vérifier son empreinte ci-dessus et
   `git apply --reverse --check` sur les trois sources seulement. Son inversion
   ciblée, après préservation vérifiée, retire les modifications désormais
   intégrées dans Git sans perdre leur provenance. Préserver le `.bak` en le
   déplaçant dans l'emplacement privé, sans le supprimer ni le publier. En cas
   d'écart, ne pas inverser le patch : utiliser un worktree séparé ou arrêter.
5. Sur un arbre suivi propre, avancer `main` avec `git merge --ff-only` vers
   le SHA épinglé. Vérifier HEAD, statut et absence de différences dans les
   migrations, scripts, configuration et templates par rapport à la baseline.
   Ni `reset`, ni `clean`, ni écrasement aveugle de sources ou de données.
6. Compiler d'abord un candidat séparé, en laissant le processus actif intact.
   Contrôler son `go version -m` : SHA épinglé et `vcs.modified=false`.
   Le service existant reconstruira ensuite depuis le même arbre à son
   redémarrage ; sa sélection de runtime, sa base et `.env` sont conservées.
7. Dans la fenêtre autorisée, redémarrer proprement `lazymarking.service`.
   Contrôler état, nouveau PID enfant, `/proc/<PID>/exe`, SHA-256 et métadonnées
   Go ; confirmer le SHA exact exécuté. Les formulaires ouverts devront être
   rechargés après la rotation de la clé CSRF. Ne pas transmettre de journaux
   contenant des URL de copies ou des données réelles.
8. Refaire via l'accès HTTPS utilisé en classe une consultation par coupon,
   la publication explicite d'une copie prévue pour cet essai, puis la
   consultation et le téléchargement individuels. Vérifier le refus sans code,
   avec une session d'une autre copie et sans jeton CSRF. Choisir avec
   l'opérateur l'évaluation et la copie de test avant toute mutation métier.
   Documenter heure, résultat et SHA exécuté, sans jeton ni identité d'élève.
9. En cas d'échec, conserver les états pour diagnostic et restaurer le code
   et le binaire précédemment préservés selon le plan de retour validé, sans
   restaurer aveuglément une base qui pourrait avoir reçu des corrections
   entre-temps. Aucun tag tant que les essais ne sont pas confirmés.

## Tag v1.0.0 : non créé et non publié

Le tag n'existait ni localement, ni sur `origin`, ni sur `public` lors des
contrôles de cette mission. Il n'est **pas créé**, car le commit Git livré n'a
pas encore été déployé et testé sur le Pi. Aucune manipulation d'un tag
existant n'a eu lieu.

Après sauvegarde vérifiée, déploiement autorisé et essais confirmés, recontrôler
l'absence du tag sur les deux dépôts. Sur le PC, utiliser exclusivement le SHA
exact dont l'exécution a été confirmée, pas implicitement la tête courante :

```sh
# À exécuter seulement après confirmation du déploiement et des essais.
validated_sha='<SHA exact exécuté et testé sur le Pi>'
git show --no-patch "$validated_sha"
git tag -a v1.0.0 "$validated_sha" -m 'LazyMarking V1.0.0 — classroom-validated end-to-end workflow, 2026-10-11.'
git push origin refs/tags/v1.0.0
git push public refs/tags/v1.0.0
```

Si un tag apparaît entre-temps, inspecter son objet annoté et sa cible ; ne
jamais le déplacer ni employer `--force`. La sauvegarde, le déploiement et les
essais restent les étapes manuelles requises avant le jalon.
