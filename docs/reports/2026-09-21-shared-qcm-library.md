# P6.2 — Bibliothèque de QCM partageables

Date : 21 septembre 2026.

## Architecture initiale

Audit effectué avant toute migration, à partir de P6.1/P6.1.1, commit
`bb4981d`. Le working tree contenait seulement `README.md` modifié et
`reset.sh` non suivi. Ces changements étrangers ont été conservés sans modification.

- `qcm` contient un nom et `user_id`, sans description. Le nom est unique
  **par utilisateur** (`UNIQUE(name,user_id)`), selon l'égalité SQLite existante.
- `qcm_questions` associe des principales à leur QCM et porte `position`.
  Les contraintes interdisent deux positions identiques et deux références à
  la même principale dans un QCM. Les triggers imposent un propriétaire commun
  au QCM, à la liaison et à la question.
- Les actions personnelles, P4 et les aperçus filtrent déjà par propriétaire.
  L'aperçu utilise l'ordre de composition ; la génération conserve son
  comportement existant de sélection des variantes et de mélange.
- P6.1 fournit `LoadShared`, une primitive interne `cloneFamily` sans commit,
  la duplication physique des images et la provenance des familles.
- P6.1.1 fournit `classification.Key` et `GetOrCreateClassification`.
- Les examens, générations, élèves, snapshots et corrections sont des données
  distinctes. Ils ne font pas partie d'une ressource QCM à copier.

## Modèle de partage

La nouvelle table `qcm_shares(qcm_id)` représente une publication explicite.
Absence de ligne = privé. L'auteur reste exclusivement `qcm.user_id`.
Seuls les utilisateurs connectés peuvent consulter la bibliothèque.

Partager ou retirer vérifie le propriétaire dans une transaction ; les deux
actions sont idempotentes. Aucune écriture dans `question_shares` n'est faite.
Un QCM partagé peut donc contenir simultanément des familles privées et des
familles publiées individuellement.

La bibliothèque reflète l'état courant du QCM : nom, composition, ordre,
consignes et contenu des familles. Une modification n'affecte jamais les
copies déjà réalisées.

## Lecture des familles privées

Les nouvelles requêtes `GetSharedQCMComposition` et `GetSharedQCMFamily`
exigent un QCM publié, l'appartenance de la famille à sa composition et un
propriétaire commun. `LoadSharedQCMFamily` utilise ce contrôle avant le
chargement des enfants, toujours filtrés par le propriétaire de la source.

Le chargement des versions de famille a été extrait dans un helper interne
`loadFamily`. Il est appelé soit après le contrôle de publication individuelle
P6.1, soit après le contrôle d'appartenance au QCM publié. Il n'existe pas de
route générale donnant accès à une famille privée.

Les images utilisent une route dédiée :

```text
GET /dashboard/library/qcm/image?qcm_id=…&question_id=…&variant_id=…
```

Cette route revérifie le partage et l'appartenance à chaque requête. Une variante
d'une autre famille, une famille hors composition ou un QCM privé donnent 404.
Les routes personnelles, `/static/images/…`, la liste de familles et les
routes P6.1 restent protégées selon leurs règles initiales.

Liste, aperçu et images portent `Cache-Control: private, no-store`.
Après retrait, toute nouvelle requête via le contexte QCM est refusée.
Comme pour toute consultation, le retrait ne peut effacer des informations déjà
reçues par un utilisateur ; une requête ayant déjà commencé sa transaction peut
terminer sa lecture cohérente.

## Copie : une transaction globale

`CopySharedQCM` ouvre une seule transaction SQLite :

1. contrôle de publication et lecture du QCM, de sa composition et des familles ;
2. refus de la copie par l'auteur lui-même ;
3. choix du nom et création du QCM personnel ;
4. appel de la primitive P6.1 `cloneFamily` pour chaque famille distincte ;
5. recréation des liaisons avec les nouveaux IDs et les positions originales ;
6. provenance puis commit unique.

Toutes les nouvelles lignes métier appartiennent au destinataire issu de la
session. La copie du QCM et ses familles sont privées. Il n'existe aucune
liaison fonctionnelle vers les ressources d'Alice, même si Bob avait déjà copié
une de ses familles auparavant : le QCM reçoit ses propres copies de familles.

La contrainte actuelle interdit une principale répétée dans un QCM. Une table
de correspondance source → copie empêche néanmoins un second clonage de la même
famille au niveau de la primitive. La contrainte de composition reste inchangée.
Les positions sont recopiées telles quelles.

Une erreur de lecture, de famille, d'image, de liaison ou de commit annule toute
l'opération. Aucun QCM, famille, caractéristique ou lien partiel n'est conservé.
Le QCM vide est également partageable et copiable ; son aperçu indique qu'il
ne contient aucune question. Les règles de génération d'un QCM vide restent
celles de l'application existante.

### Cohérence et concurrence

Les lectures et écritures de copie utilisent la même transaction, sans commit
intermédiaire. Le snapshot de lecture SQLite fige la composition et le contenu
lus pour l'opération. En mode journal classique, les verrous peuvent bloquer ou
faire échouer une écriture concurrente. En WAL, une lecture devenue obsolète
ne peut pas être promue en écriture : le test vérifie précisément
`SQLITE_BUSY_SNAPSHOT`, sans copie partielle, puis le succès d'une nouvelle
copie sur l'état courant.

Il n'y a pas de retry automatique. Un conflit ou incident renvoie un message
invitant à réessayer et ne valide aucune copie incohérente.

## Images

Réutilisation de `cloneFamily`, `copyImage` et `OpenImage` de P6.1 :

- fichiers physiquement dupliqués avec des noms UUID nouveaux ;
- conservation du pourcentage de redimensionnement ;
- rejet des traversées, symlinks et fichiers non réguliers selon le contrat P6.1 ;
- liste commune des fichiers créés pour toutes les familles ;
- nettoyage de toute cette liste en cas d'échec, même après plusieurs familles
  et même lorsque le commit échoue.

Le helper de nettoyage est désormais commun aux copies de famille et de QCM.
La suppression des fichiers source n'affecte pas les copies. Un incident de
nettoyage est remonté dans l'erreur, jamais masqué.

Les fichiers ne sont pas transactionnels au sens SQLite : comme en P6.1,
un arrêt brutal du processus peut laisser des fichiers orphelins. Aucun mécanisme
de récupération après crash n'est ajouté dans ce chantier. Une suppression
concurrente de fichier source peut faire échouer la copie, qui est alors annulée.

## Classifications

Le clonage réutilise exactement P6.1.1 pour matière, niveau, thème, compétence
et difficulté : case folding Unicode, décomposition canonique, retrait des
marques diacritiques, normalisation des espaces et recomposition Unicode.

Une caractéristique équivalente du destinataire est réutilisée avec son libellé
inchangé ; sinon elle est créée dans la transaction. Les points conservent
l'équivalence numérique. `Seconde` et `seconde` correspondent ; `2nde` et
`Seconde` restent distincts. Aucun ID appartenant à l'auteur n'est utilisé
chez le destinataire et aucun doublon historique n'est fusionné.

## Provenance

`qcm_copy_origins` conserve l'ID du QCM source et le nom de son auteur au moment
de la copie. Seul le nouveau QCM possède une clé étrangère ; il n'y a aucune
clé étrangère vers le QCM source ou son auteur.

Mes QCM affiche « Copié depuis un QCM de Alice ». Les familles conservent
également la provenance P6.1. Ces mentions survivent à la suppression de la
source et ne permettent pas d'accéder à une ressource retirée du partage.
Il s'agit de l'auteur immédiat de la ressource copiée, sans chaîne de versions.

## Conflits de noms

La copie conserve le nom original s'il est libre chez le destinataire.
Sinon elle essaie successivement :

```text
Contrôle énergie — copie
Contrôle énergie — copie 2
Contrôle énergie — copie 3
```

Le choix et l'insertion se font dans la transaction globale, en respectant
l'unicité existante par utilisateur. Aucune normalisation sémantique du nom
de QCM n'est introduite.

## UX

- Une entrée de navigation « Bibliothèque », avec deux onglets : Questions / QCM.
- `/dashboard/library/qcm` : nom, auteur, nombre de questions, matières et
  niveaux de la composition ; marqueur « Votre QCM » pour l'auteur.
- Recherche sur nom, auteur, matières et niveaux avec la clé Unicode commune ;
  filtres simples par matière et niveau. Les listes de filtres gardent les
  libellés affichés, comme P6.1. Aucun champ de description n'a été ajouté.
- États explicites « Aucun QCM partagé pour le moment. » et absence de résultat.
- Aperçu HTML en lecture seule : composition ordonnée, consignes communes,
  principales, variantes, réponses correctes/incorrectes, images, classifications
  et barèmes. Les contrôles d'édition personnels ne sont jamais proposés.
- « Copier dans Mes QCM » crée la copie puis redirige vers Mes QCM avec un
  message de succès. Pour l'auteur : « Votre QCM » et retour à Mes QCM.
- Mes QCM conserve P4 et ses aperçus, ajoute Privé/Partagé, Partager/Retirer de
  la bibliothèque et la provenance.
- Aide visible : « Partager un QCM permet aux autres enseignants de le consulter
  et de le copier. Ses questions ne sont pas publiées individuellement. »

Le retrait enlève uniquement la publication. Il ne supprime ni l'original,
ni ses familles, ni les copies. Retirer le QCM ne retire pas une famille
publiée séparément.

## Sécurité

Toutes les routes sont enregistrées derrière `login.CheckAuth`. Les mutations
sont des POST avec CSRF ; les autres verbes sont refusés. Les IDs non positifs
ou malformés donnent 400, les ressources inconnues/privées/étrangères 404.
Le POST de copie de son propre QCM donne 409 avec un message compréhensible.
Le destinataire et le propriétaire des mutations viennent exclusivement de la
session, jamais d'un `user_id` soumis.

Les requêtes de partage/retrait sont elles-mêmes filtrées par propriétaire.
Les droits d'édition des QCM, relations, principales, réponses, variantes et
images restent inchangés. Les tests HTTP vérifient ces frontières après partage,
pas seulement l'absence de boutons dans l'aperçu.

## Migration et compatibilité

`0047_shared_qcm.sql` ajoute seulement `qcm_shares` et
`qcm_copy_origins`, toutes deux vides. Tous les QCM existants restent privés.
Aucune migration ancienne, aucun snapshot, aucun script de lancement n'est
modifié. La garde embarquée P5.1 applique 0047 avant le démarrage.
SQL/sqlc ont été régénérés.

Validation exclusivement sur copies temporaires :

| Source | Version de la copie | Partages de familles préservés | QCM privés | Intégrité / FK | Empreinte source inchangée |
|---|---|---|---|---|---|
| smoke | 40 → 47 | oui | oui | OK | oui |
| real | 41 → 47 | oui | oui | OK | oui |
| 2026-2027 | 46 → 47 | oui | oui | OK | oui |

Les données des QCM et leurs positions ont été ajoutées au digest de contrôle
avant/après, en plus des questions et snapshots. Le test ne suppose plus que
la base source est sans familles partagées : les publications existantes sont
comparées et conservées. Le cas synthétique 44 migre jusqu'à 47 ; base neuve,
redémarrage à jour, échec de migration et version future restent testés.
Les bases sources et les trois runtimes n'ont pas été modifiés.

## Tests

### Tests ajoutés

- `TestSharedQCMCompleteCopyAndIndependence` : privé par défaut, partage auteur,
  familles privées accessibles uniquement en contexte QCM, trois familles,
  ownership, ordre, consignes, versions, réponses, images physiques, barèmes,
  classifications réutilisées, provenance, indépendance dans les deux sens,
  mise à jour de l'aperçu, retrait et suppression des originaux.
- `TestCopyQCMRollback` : source privée/inconnue, auteur lui-même, erreur dans
  une famille tardive, image tardive manquante, liaison tardive et erreur au
  commit par contrainte différée. Contrôle de toutes les lignes et fichiers.
- `TestCopyQCMNamesAndEmptyComposition` : conflits successifs et QCM vide.
- `TestCopyQCMLeavesExistingExamHistoryUntouched` : examens, génération, élève
  et snapshots existants chez Alice restent inchangés et ne sont jamais copiés.
- `TestCopyQCMConcurrentSourceChangeIsCoherent` : conflit WAL réel, refus de
  promotion d'un ancien snapshot, nouvelle tentative cohérente.
- `TestSharedQCMAliceBobJourney` : véritables routes, sessions, CSRF et templates ;
  filtres, ordre, familles privées, accès image contextualisé, URLs devinées,
  auteur seul, propre QCM, IDs/session forgés, refus d'édition/suppression
  étrangères, retrait et conservation de la copie.
- Dans ce parcours : ouverture personnelle et réorganisation P4 de la copie,
  compilation et service de PDF portrait et paysage ; après suppression des
  sources et de leurs fichiers, génération réelle d'une copie d'examen avec
  trois familles personnelles, variante illustrée, consigne, coordonnées,
  snapshot persisté et QR décodé sur chaque page (une page pour cette fixture).

Les tests existants P6.1/P6.1.1 restent exécutés. Les fixtures Mes QCM sont
adaptées aux tables de partage et l'inventaire CSRF passe de 74 à 76 formulaires.
Le runtime HTTP temporaire est placé sous `assets/tmp` pour être accessible
au binaire Typst installé via Snap ; il est nettoyé à la fin du test.
Aucun contrôle visuel manuel dans le navigateur n'est revendiqué.

### Commandes de validation

- `go test ./...` : réussi.
- `go test -count=1 ./internal/handlers/qcm ./internal/handlers/qcmQuestions ./internal/handlers/qcmPreview ./internal/sharedlibrary ./internal/db ./internal/handlers/questions ./internal/httpsecurity ./internal/questionfamilies ./internal/templates/...` : réussi.
- `go test -v -count=1 ./internal/db -run TestOpenMigratedDBRuntimeCopies` : trois copies effectivement testées, réussite.
- `sqlc generate -f db/sqlc.yaml` : réussi ; deuxième génération stable.
- `git diff --check` : réussi.

## Fichiers principaux

- `db/migrations/0047_shared_qcm.sql`, `db/query/qcmSharing.sql` et code sqlc.
- `internal/sharedlibrary/qcm.go`, réutilisation de `family.go` et `copy.go`.
- `internal/handlers/qcm/sharing.go`, liste personnelle et routes serveur.
- Templates QCM de liste et aperçu partagé, liste personnelle et onglets de
  bibliothèque ; structures de données de navigation et de liste.
- Tests du service, du parcours HTTP, des migrations et de l'inventaire CSRF.

## Points ouverts

Aucun point fonctionnel P6.2 restant. La récupération des fichiers orphelins
après arrêt brutal reste une limite du contrat filesystem P6.1, à traiter
séparément si nécessaire. Pas de synchronisation, versioning, publication
implicite de familles, partage d'examens ni module d'entraînement.

Aucun commit ni push effectué.
