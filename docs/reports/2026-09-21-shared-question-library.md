# P6.1 — Bibliothèque de familles de questions partageables

Date : 21 septembre 2026. P6.2 (QCM partageables) est identifié comme chantier suivant.

## Architecture initiale

L'audit a précédé toute migration. Le dépôt était sur `6c5c584` (P5.1), avec seulement `README.md` modifié et `reset.sh` non suivi ; ces changements étrangers ont été conservés.

Une famille est constituée de :

- `questions` : énoncé principal, consigne commune `instruction`, six caractéristiques et `user_id` ;
- `answers` : réponses de la principale, avec contenu et état correct/incorrect ;
- `images` : au plus une image principale, son nom de fichier et son pourcentage de redimensionnement ;
- `alt_questions` : variantes rattachées à la principale ;
- `alt_answers` et `alt_images` : réponses et image propres à chaque variante ;
- `subjects`, `year_levels`, `themes`, `skills`, `difficulties`, `points` : caractéristiques personnelles, uniques par libellé/valeur et utilisateur depuis 0029.

Les requêtes existantes contrôlent l'ownership des parents et des enfants. Les triggers de 0030, recréés lors des reconstructions de tables, interdisent les relations entre propriétaires différents. Depuis 0041, un même énoncé peut exister plusieurs fois : aucune contrainte d'unicité de contenu ne bloque une copie.

`questionfamilies.Build` regroupe les variantes pour « Mes questions » et les sélecteurs QCM. P2 filtre en mémoire la banque personnelle par texte (sans distinction de casse/accents), matière, niveau et thème. Les routes de création, édition et suppression sont séparées. Aucun mécanisme de duplication complète de famille ou de QCM n'a été trouvé.

`qcm` contient le nom et le propriétaire ; `qcm_questions` associe uniquement les principales, avec une position unique par QCM et un contrôle de propriétaire commun. Les aperçus QCM suivent cet ordre. La génération peut sélectionner une variante et mélanger les réponses. `student_exam_content` persiste le contenu effectivement généré, sa consigne, les caractéristiques et `layout_version` ; les snapshots par page et références PNG servent à la correction historique. La nouvelle bibliothèque ne participe pas à ces lectures historiques.

Les images sont des fichiers locaux sous `assets/images`. La DB porte un nom de fichier, pas les octets. Les suppressions existantes enlèvent le fichier ; une simple copie de référence aurait donc créé une dépendance dangereuse. `/static/images/{filename}` exige déjà l'ownership et reste inchangé.

## Modèle de partage retenu

Une entrée dans `question_shares(question_id)` signifie « partagée ». Son absence signifie « privée ». Le propriétaire est celui de la question ; il n'est pas dupliqué dans la table de partage.

La publication porte sur la famille entière, dans son état actuel : consigne, principale, variantes, réponses, images et caractéristiques. Seuls les enseignants connectés accèdent à la bibliothèque. Une modification de l'original partagé apparaît dans ses futurs aperçus et copies ; les copies déjà réalisées restent indépendantes.

Le POST de partage vérifie le propriétaire dans la même transaction que l'ajout ou le retrait. Ces actions sont idempotentes. Retirer supprime seulement l'entrée de partage. Supprimer une question enlève cette entrée par cascade, sans lien vers les copies.

## Copie

`internal/sharedlibrary.CopyShared` ouvre une transaction SQLite et vérifie le partage avant toute création. L'auteur est lu depuis la DB, jamais fourni par le formulaire. La lecture de la famille et toutes les écritures appartiennent à cette transaction.

La copie crée :

1. les caractéristiques manquantes du destinataire ;
2. une nouvelle principale, avec son énoncé et sa consigne ;
3. ses réponses, états et image ;
4. toutes les variantes, leurs réponses, états et images ;
5. une provenance informative.

Chaque ligne métier créée porte le `user_id` issu de la session du destinataire. Les identifiants de la famille sont nouveaux. Aucune ligne `question_shares` n'est créée pour la copie : elle est privée. L'ordre relatif des variantes et des réponses est conservé par lecture des IDs croissants. Les variantes héritent de la nouvelle principale et de sa consigne commune.

Une erreur provoque le rollback de toutes les lignes, y compris des classifications nouvellement créées, et la suppression des fichiers nouvellement copiés. Les tests injectent une erreur SQL après création des images et une image de variante manquante après copie de l'image principale. Aucune famille partielle ne subsiste.

Les copies répétées sont permises et produisent des familles distinctes. La V1 n'effectue ni synchronisation, ni déduplication automatique, ni suivi de versions.

## Images

Chaque image est copiée physiquement vers un nom aléatoire `copy_<UUID>.<extension>`. Les octets et le redimensionnement sont conservés. Un remplacement ou une suppression du fichier original ne touche donc pas le fichier du destinataire.

Les noms sont validés comme composants simples. Les répertoires et les fichiers symboliques sont refusés ; l'ouverture utilise `os.OpenRoot`, `Lstat` et la vérification d'identité du fichier ouvert. Les fichiers sont créés exclusivement, écrits puis synchronisés avant le commit SQLite.

L'aperçu utilise une route dédiée : `/dashboard/library/image?question_id=...&variant_id=...` (0 pour la principale). Elle vérifie à chaque requête le partage puis l'appartenance de l'image à cette famille. Elle n'accepte jamais un chemin ou nom de fichier fourni par le navigateur. Les accès utilisent `Cache-Control: private, no-store`, notamment pour éviter une réutilisation en cache après retrait.

La route d'images personnelles ne donne aucun droit supplémentaire sur les fichiers de l'auteur. Après copie, les références DB du destinataire permettent leur affichage et leur gestion via les routes existantes.

Limite du stockage existant : SQLite et le système de fichiers ne forment pas une transaction commune. Un arrêt brutal du processus avant commit peut laisser un fichier sans référence, mais pas une famille partiellement commitée. Le scanner et la purge d'images orphelines existants restent applicables ; une erreur de nettoyage normale est remontée et journalisée. Aucun fichier source n'est nettoyé par l'opération de copie.

## Classification

Les caractéristiques textuelles sont recherchées chez le destinataire par équivalence typographique (P6.1.1, détaillée ci-dessous), puis réutilisées ou créées. Les points gardent l'équivalence numérique. Les libellés affichés ne sont pas normalisés ni réécrits.

Aucun ID de caractéristique de l'auteur n'est réutilisé chez un autre propriétaire. Une caractéristique réutilisée chez Bob peut naturellement servir à plusieurs questions de Bob, comme dans le modèle personnel existant. Les doublons historiques ne sont jamais fusionnés.

## Provenance

`question_copy_origins` conserve l'identifiant source et le nom de l'auteur au moment de la copie. Seule sa liaison vers la nouvelle question a une clé étrangère ; aucune clé étrangère ne lie la copie à la source ou à son auteur.

« Copiée depuis une question de Alice » apparaît dans Mes questions. Cette mention survit à la suppression de l'original et ne fournit pas un lien permettant de consulter une question devenue privée. C'est la provenance immédiate de la copie, pas une chaîne complète d'auteurs.

## UX

- La navigation distingue « Mes questions » et « Bibliothèque ».
- Mes questions conserve les actions existantes, avec un badge Privée/Partagée et un bouton Partager la famille/Retirer de la bibliothèque. Une phrase explique la portée du partage.
- Bibliothèque affiche uniquement les familles partagées : énoncé, consigne éventuelle, matière, niveau, thème, compétence, auteur, nombre de variantes et points.
- Les filtres Go de P2 sont réutilisés dans le même package. Le formulaire est extrait dans `library_filters.html`, commun aux deux pages ; la recherche inclut aussi la consigne.
- L'état sans partage est « Aucune question partagée pour le moment. » ; une recherche sans résultat possède son propre message.
- L'aperçu HTML montre la consigne, la principale et toutes les variantes, leurs réponses correctes/incorrectes, images, caractéristiques, difficulté et barème. Il est en lecture seule.
- « Copier dans Mes questions » crée la copie privée et redirige vers la banque personnelle avec un message de succès.
- Pour l'auteur, « Votre question » remplace l'action de copie dans l'aperçu, avec un retour vers Mes questions. La liste porte aussi cette indication.

Le rendu utilise les templates HTML échappés et Bootstrap existants, sans CSS global ni modification Typst. Il s'agit d'un aperçu de contenu administratif, sans génération PDF supplémentaire.

## Sécurité

Toutes les nouvelles routes passent par `login.CheckAuth`. Toutes les mutations sont des POST protégés par le middleware CSRF global et ses champs de formulaire. Le destinataire est toujours extrait de la session, même si un formulaire tente de fournir un autre `user_id`.

Les IDs inconnus et les questions non partagées donnent 404 dans l'aperçu, les images et la copie. Un utilisateur ne peut partager ou retirer que ses propres questions. Un ID de variante d'une autre famille est refusé, même si la principale demandée est partagée.

Le partage ne modifie aucune autorisation des handlers d'édition existants. Le test HTTP vérifie les refus de GET d'édition, POST d'édition et POST de suppression pour la principale, réponses, variantes, réponses alternatives et images d'Alice, même quand la famille est partagée.

## QCM : audit et découpage P6.2

Le partage de QCM est faisable, mais ajoute une autorisation et une transaction portant sur plusieurs familles. Il faut décider si partager un QCM rend son contenu consultable uniquement dans ce QCM ou publie aussi ses familles séparément. Publier implicitement ces familles serait incompatible avec l'opt-in attendu ici.

P6.1 livre donc les familles partageables. P6.2 devra :

1. ajouter une publication explicite du QCM, sans publier implicitement ses questions dans la liste des familles ;
2. autoriser la lecture des familles via l'appartenance à ce QCM partagé, indépendamment de `question_shares` ;
3. copier chaque famille distincte dans une seule transaction globale ;
4. créer un QCM personnel, résoudre son éventuel conflit de nom, puis recréer `qcm_questions` avec les nouveaux IDs et les positions d'origine ;
5. nettoyer tous les fichiers copiés si une famille ou une liaison échoue ;
6. tester le retrait, les modifications concurrentes de composition et l'absence de copie partielle du QCM.

`cloneFamily` est déjà séparée de l'ouverture/validation/commit de transaction : elle pourra servir à cette opération globale. Aucun examen, élève, snapshot ou résultat de correction ne doit être cloné avec un QCM. Le test P6.1 vérifie déjà qu'une famille copiée peut être insérée dans un QCM du destinataire et lue par le builder de génération habituel.

## Migration et compatibilité

0046 ajoute seulement les deux tables de partage et provenance. Elle ne remplit aucune publication, ne reconstruit pas de table historique et ne modifie aucune migration précédente. Les tables métier et les snapshots restent inchangés. Les fichiers SQL/sqlc sont régénérés.

La garde P5.1 embarque automatiquement 0046 et l'applique avant les handlers. Les scripts des trois runtimes restent inchangés. La validation s'est faite exclusivement dans des copies temporaires des bases sous `testdata` :

| Base source | Migration de la copie | Questions privées après migration | Intégrité / clés étrangères | Empreinte source conservée |
|---|---|---|---|---|
| smoke | 40 → 46 | oui | OK | oui |
| real | 41 → 46 | oui | OK | oui |
| 2026-2027 | 45 → 46 | oui | OK | oui |

Les contenus des questions et des snapshots avant/après migration sont comparés. Le scénario synthétique P5.1 version 44 est conservé et monte maintenant jusqu'à 46. Le test sur copie 2026-2027 part directement de sa version source 45, sans down de préparation.

## Tests

Nouveaux tests :

- `TestSharedLibraryAliceBobJourney` : état privé, partage auteur, affichage, filtres, aperçu, consigne, réponses, variantes, images, copie depuis une autre session, refus d'édition/suppression étrangère, CSRF, verbes HTTP, propriétaire forgé, édition personnelle, builder de génération, composition QCM, retrait, suppression de l'original et conservation de la copie ;
- `TestCopyCompleteFamilyAndIndependence` : ownership de toutes les tables, classification réutilisée/créée, barème, réponses et états, images et tailles, provenance, indépendance et conservation après suppression ;
- `TestCopyFailureRollsBackRowsAndFiles` : image manquante, erreur DB après copie physique, source privée/inconnue, lien symbolique et absence de lignes/fichiers partiels ;
- `TestOpenImageRejectsUnsafeFiles` : traversée, chemins absolus et fichiers non réguliers ;
- `TestCopyFamilyWithoutInstructionImagesOrVariants` : famille simple sans contenu optionnel.

Tests adaptés : fixtures de Mes questions, version cible des tests de garde de migration et vérification des trois copies, inventaire CSRF passé de 72 à 74 formulaires. Le premier passage complet a trouvé uniquement le compteur CSRF ancien ; il a été corrigé après vérification des deux nouveaux formulaires.

Validation finale :

- `go test ./...` : réussi ;
- `go test -count=1 ./internal/handlers/questions ./internal/handlers/qcm ./internal/handlers/qcmQuestions ./internal/handlers/qcmPreview ./internal/db ./internal/sharedlibrary ./internal/httpsecurity ./internal/questionfamilies ./internal/templates/data` : réussi ;
- `go test -v -count=1 ./internal/db -run TestOpenMigratedDBRuntimeCopies` : trois sous-tests exécutés, réussis ;
- `sqlc generate -f db/sqlc.yaml` : réussi, empreintes de tous les fichiers Go DB identiques avant/après régénération ;
- `git diff --check` : réussi.

Les templates sont réellement rendus par les tests HTTP avec CSRF ; aucune validation visuelle manuelle dans le navigateur n'est revendiquée. Les suites existantes de génération, snapshots et correction passent avec l'ensemble du dépôt.

## P6.1.1 — Cohérence des classifications

### Règle commune et limites

`internal/classification.Key` produit une clé de comparaison : Unicode case folding, décomposition canonique NFD, suppression des marques combinatoires de catégorie Mn, suppression des espaces de début/fin et regroupement des espaces Unicode (y compris tabulations et espaces insécables), puis recomposition NFC. La fonction est idempotente.

Exemples équivalents : `Seconde` / `seconde` / ` SECONDE ` ; `Électricité` / `electricite` / `ÉLECTRICITÉ` / forme Unicode décomposée ; `Physique  Chimie` / `Physique Chimie`.

Les tirets et les autres signes de ponctuation restent significatifs. Aucune abréviation ni équivalence sémantique n'est déduite : `2nde` ≠ `Seconde`, `PC` ≠ `Physique-Chimie`, `6ème` ≠ `Sixième`, `Physique-Chimie` ≠ `Physique Chimie`.

La recherche textuelle P2/P6 appelle la même fonction pour éviter deux traitements Unicode divergents. Les listes de filtres et leurs sélections exactes restent fondées sur les libellés affichés ; elles ne fusionnent pas les classifications.

### Copie et création manuelle

`db.GetOrCreateClassification` centralise la comparaison et la création pour matières, niveaux, thèmes, compétences et difficultés textuelles. La liste des tables autorisées est fixe et la recherche est toujours limitée au destinataire. En présence de doublons historiques normalisés, le plus petit ID est choisi de manière déterministe, sans modifier aucune ligne.

La copie appelle ce helper dans sa transaction existante. Les créations manuelles ouvrent une transaction pour la recherche et l'insertion. Les libellés sont conservés tels quels ; le trim déjà effectué par les formulaires à la création reste inchangé. Le traitement numérique des points ne change pas.

Si une création retrouve un équivalent, aucune ligne n'est ajoutée. La réponse redirige vers la liste avec l'identifiant existant, qui est relu dans la liste appartenant à l'utilisateur. Une notification discrète indique par exemple : « Seconde » existe déjà dans vos niveaux. La valeur existante a été conservée. Un identifiant étranger ne permet pas d'afficher le libellé d'un autre utilisateur.

Les éditions explicites de libellés conservent leur fonctionnement historique ; ce complément protège les chemins de création et de copie demandés. Il ne consolide pas les classifications déjà existantes et n'ajoute pas de contrainte globale à la base.

### Aide non bloquante

Les écrans de liste, création et édition des cinq classifications textuelles affichent une aide commune sous leur titre :

> Conseil : gardez des libellés clairs, complets et cohérents (par exemple « Seconde »), et évitez les abréviations inutiles. Vos appellations restent libres.

Le texte est centralisé dans `internal/templates/data/classification.go`, servi par le helper de template `classificationAdvice`. Aucun modal, nomenclature imposée ou blocage du partage n'est ajouté.

### Sa propre famille partagée

La liste identifie la ressource de l'utilisateur par « Votre question ». L'aperçu montre le même état et un retour vers Mes questions, sans formulaire de copie. L'auteur peut toujours prévisualiser et retirer depuis sa banque personnelle.

Le service `CopyShared` vérifie également le propriétaire dans sa transaction et retourne `ErrOwnFamily` avant toute création. Le handler traduit ce cas en HTTP 409 avec « Cette question est déjà dans Mes questions. Vous pouvez la modifier depuis votre banque. » Le POST forgé ne crée ni ligne ni fichier. Les autres utilisateurs conservent leur parcours de copie normal.

### Données et tests

Aucune migration P6.1.1 ni colonne normalisée n'est ajoutée. SQL/sqlc n'a pas changé dans ce complément et n'a pas été régénéré. 0046 reste la seule migration P6.1 en attente de commit.

Lors de cette reprise, la base source `2026-2027` était déjà à 46 avec deux partages. Le test runtime P6.1 supposait encore une base jamais partagée ; il a été adapté pour comparer les IDs partagés avant/après sur la copie temporaire, plutôt que d'exiger zéro. Pour les bases antérieures à 46, l'attendu reste aucune publication implicite. Les sources ne sont pas modifiées.

Tests ajoutés ou complétés :

- `TestKey` : casse, accents, formes Unicode canoniquement équivalentes, espaces, ponctuation, idempotence et non-équivalences sémantiques ;
- `TestClassificationReuseAndHistoricalDuplicates` : réutilisation, isolation utilisateur, libellés conservés, choix déterministe, doublons historiques toujours présents après redémarrage, refus des tables non autorisées et libellés vides ;
- `TestCopyReusesNormalizedClassifications` : réutilisation des cinq classifications et du barème, absence de doublons, libellé Seconde conservé, création séparée pour 2nde ;
- `TestManualClassificationCreationReusesLabelsAndShowsAdvice` : les cinq POST de création, notifications réellement rendues, aide en liste/création/édition, absence de fuite via un identifiant étranger et création d'une autre valeur ;
- `TestCopyOwnFamilyRefusedWithoutWrites` : refus métier sans aucune écriture DB/fichier ;
- parcours HTTP Alice/Bob complété : badge auteur, absence de bouton/formulaire Copier, POST auteur refusé en 409, copie de Bob toujours fonctionnelle.

Validation finale P6.1.1 :

- `go test ./...` : réussi ;
- `go test -count=1 ./internal/classification ./internal/handlers/questions ./internal/sharedlibrary ./internal/questionfamilies ./internal/db ./internal/templates/data ./internal/handlers/subjects ./internal/handlers/themes ./internal/handlers/yearlevels ./internal/handlers/skills ./internal/handlers/difficulties` : réussi ;
- `git diff --check` : réussi ;
- `gofmt -l` sur les fichiers Go concernés : aucune sortie ;
- pas de migration supplémentaire, pas de régénération sqlc, pas de commit ni push.

## Points ouverts

- P6.2 : partage, aperçu et copie transactionnelle de QCM, suivant le découpage ci-dessus.
- Si le volume partagé devient important : pagination et filtres SQL, comme pour l'évolution envisagée de P2. La V1 réutilise volontairement le filtrage en mémoire.

Aucun commit ni push réalisé pour P6.1. `README.md` et `reset.sh` restent hors périmètre.
