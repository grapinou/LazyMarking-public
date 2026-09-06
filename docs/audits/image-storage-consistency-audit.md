# Audit de décision — Cohérence DB ↔ filesystem des images

## 1. Synthèse et décision

État audité : code présent dans le worktree au 29 août 2026, après les corrections de parenté des images de variantes et les tests récemment ajoutés. L’ancien audit `images-audit.md` est donc une source historique, pas la description de référence de l’état actuel.

**Recommandation : conserver l’ordre DB → filesystem, rendre chaque échec de suppression observable, puis ajouter une réconciliation ciblée et prudente, d’abord en lecture seule et ensuite avec une purge explicite assortie d’un délai de grâce.**

Pour LazyMarking (mono-instance, SQLite, stockage local), cet ordre privilégie l’intégrité visible : après une suppression acceptée, aucune ligne DB ne continue à annoncer une image qui n’existe plus. Son mode d’échec est un fichier orphelin non servi, donc essentiellement une fuite bornée d’espace disque. C’est préférable à filesystem → DB, qui peut laisser une ligne active vers un fichier perdu et casser l’UI, les aperçus et les examens.

Il n’existe pas de risque P0 identifié dans l’état actuel : l’orphelin filesystem n’est pas accessible par la route applicative sans référence DB appartenant à l’utilisateur. Il existe en revanche des lacunes P1 de récupération/observabilité et des lacunes P2 de couverture et de maintenance.

## 2. Invariants et composants actuels

- Stockage physique : `assets/images` (`config.ImageSavePath`).
- Service HTTP : `GET /static/images/{filename}` (`config.PublicImageBaseURL`).
- Tables : `images` et `alt_images`, avec une image au plus par parent (`question_id UNIQUE`, `alt_question_id UNIQUE`).
- Les FK vers `questions` et `alt_questions` utilisent `ON DELETE CASCADE`.
- Les requêtes de mutation et de lecture sont bornées par `user_id` et vérifient le parent.
- `SaveUploadedFile` crée exclusivement (`O_EXCL`), n’écrase pas, supprime une écriture partielle après erreur de copie/fermeture et refuse les chemins non sûrs.
- `RemoveStoredImageFile` refuse les composants non sûrs, ne suit pas un répertoire d’images symbolique et considère `os.ErrNotExist` comme un succès idempotent.
- `UserOwnsImage` n’autorise le service HTTP que si le nom existe dans `images` ou `alt_images` pour l’utilisateur connecté.

## 3. Cycles de vie exhaustifs

### 3.1 Ajout d’une image principale

État initial normal : question possédée présente en DB, aucune ligne `images` pour cette question, aucun fichier portant le nom calculé.

Ordre réel dans `AddImageHandler` :

1. `CheckImageFile` borne et parse le multipart, lit le champ `image`, contrôle taille, extension, type réel et dimensions ;
2. parse `question_id` et `width`, puis valide les dimensions effectives ;
3. `SanitizeFilename` calcule le nom ;
4. `SaveUploadedFile` écrit le fichier permanent ;
5. `ImageCircleCheck` effectue le contrôle OpenCV ;
6. `CreateImage` tente l’INSERT ownership-aware ;
7. si INSERT en erreur ou zéro/mauvais nombre de lignes, tentative de suppression du fichier ;
8. succès : ligne DB et fichier présents, redirection 303 vers la gestion de l’image.

| Échec | DB finale | Filesystem final | Réponse/observation |
|---|---|---|---|
| Validation multipart/image | inchangée | aucun fichier permanent | 400, log `CheckImageFile` ; fichiers multipart temporaires nettoyés |
| ID/largeur invalide | inchangée | aucun fichier permanent | 400 ou redirection 303 vers erreur |
| Nom invalide | inchangée | aucun fichier permanent | redirection 303 vers erreur |
| Sauvegarde échoue | inchangée | aucun nouveau fichier complet ; une copie partielle est supprimée ; un fichier préexistant est préservé | 500, log `SaveUploadedFile` |
| OpenCV rejette | inchangée | fichier supprimé normalement | redirection 303 vers erreur |
| Nettoyage après rejet OpenCV échoue | inchangée | **orphelin possible** | redirection 303 maintenue, échec seulement journalisé |
| INSERT échoue (unicité/DB) | inchangée | fichier supprimé normalement | redirection 303 vers erreur |
| Parent absent/étranger | `CreateImage` affecte 0 ligne | fichier supprimé normalement | 404 via `HandleOwnedMutationRows` |
| Nettoyage après erreur/0 ligne échoue | aucune ligne créée | **orphelin possible** | réponse d’erreur initiale maintenue, échec de nettoyage journalisé |

Nuance unicité : avec exactement le même nom calculé, `O_EXCL` peut refuser l’écriture avant l’INSERT et produire un 500. Avec un autre basename, le fichier est écrit, la contrainte `question_id UNIQUE` refuse l’INSERT, puis le nettoyage est tenté.

### 3.2 Ajout d’une image de variante

Le flux de `AddAltImageHandler` est identique, avec `question_id`, `alt_question_id`, un nom basé sur la variante et `CreateAltImage`.

État initial normal : question et variante cohérentes et possédées, aucune ligne `alt_images`, aucun fichier cible. État final normal : une ligne `alt_images` et un fichier.

Les mêmes points d’échec et états s’appliquent. `CreateAltImage` exige que la variante corresponde simultanément à `alt_question_id`, `question_id` et `user_id`. Un mauvais parent, un parent absent ou étranger produit zéro ligne, puis le fichier est normalement supprimé et le handler répond 404. Si ce nettoyage échoue, un orphelin subsiste.

### 3.3 Modification de taille principale

État avant : ligne `images` et fichier attendus présents. Ordre de `EditImageHandler` : lecture de l’image ownership-aware, `ReadImageConfig` sur le fichier, validation de `width`, contrôle OpenCV, puis `UpdateSizeImage`.

- Succès : seul `resize_percentage` change ; fichier inchangé.
- Ligne absente/étrangère : 404, aucun changement.
- Fichier absent/invalide : 500 après échec `ReadImageConfig`, DB inchangée.
- Taille ou contrôle OpenCV refusé : redirection 303 vers erreur, DB et fichier inchangés.
- UPDATE en erreur : 500, fichier inchangé ; SQLite conserve normalement l’ancienne valeur.
- UPDATE à zéro/mauvais nombre de lignes : réponse de `HandleOwnedMutationRows`, fichier inchangé.

### 3.4 Modification de taille de variante

Même séquence et mêmes états dans `EditAltImageHandler`, avec lecture et UPDATE bornés par `alt_question_id + question_id + user_id`. La correction actuelle transmet bien `QuestionID` à `GetAltImageByAltQuestionID`. Aucun fichier n’est réécrit.

### 3.5 Suppression directe principale

État avant normal : ligne DB et fichier présents.

Ordre de `DeleteImageHandler` :

1. lecture du nom par `GetImageByQuestionID(question_id, user_id)` ;
2. `DeleteImage` ;
3. contrôle strict : 0 ligne → 404, plus d’une → 500 ;
4. `RemoveStoredImageFile(image.ImageName)` ;
5. redirection 303 vers la page Image, même si la suppression physique échoue.

| Échec | État laissé | Réponse/log |
|---|---|---|
| Lecture préalable échoue | DB et fichier inchangés | absent/étranger : 404 via `HandleOwnedLookupError` |
| DELETE DB échoue | DB et fichier inchangés | 500, log `DeleteImage DB error` |
| DELETE affecte 0/>1 | fichier non touché ; DB selon résultat anormal | 404/500, log explicite |
| Fichier déjà absent | ligne supprimée, aucun fichier | considéré comme succès, 303, aucun log d’erreur |
| Suppression physique échoue | ligne supprimée, **fichier orphelin** | log `RemoveStoredImageFile`, puis 303 |

Le risque réel est donc une fuite d’espace disque et éventuellement de données résiduelles accessibles à un administrateur du serveur, pas une exposition via LazyMarking.

### 3.6 Suppression directe de variante

`DeleteAltImageHandler` suit exactement DB → filesystem, avec lecture et DELETE par `alt_question_id + question_id + user_id`, contrôle 0/>1, puis suppression physique et redirection 303.

- Fichier absent : succès idempotent et 303.
- Échec filesystem : ligne supprimée, fichier orphelin, log `From DeleteAltImageHandler -> RemoveStoredImageFile`, 303.
- Particularité actuelle : une erreur de lecture préalable, y compris `sql.ErrNoRows` pour une variante absente/étrangère/mal appariée, est traitée comme « DB error » et répond 500, contrairement au handler principal. Cela ne crée pas d’incohérence, mais constitue une asymétrie fonctionnelle hors du mécanisme filesystem.

### 3.7 Suppression d’une variante parente

`DeleteAltQuestionHandler` :

1. vérifie la variante par `GetAltQuestionByParentID` ;
2. lit avant cascade l’éventuelle image et mémorise son nom ; `sql.ErrNoRows` signifie « pas d’image » ;
3. supprime la variante ; la FK cascade supprime `alt_images` ;
4. contrôle 0/>1 ;
5. supprime le fichier mémorisé ;
6. redirige 303, même si le filesystem échoue.

Tout échec avant le DELETE laisse DB et fichier inchangés. Un échec DB laisse également le fichier intact. Après succès DB, un échec filesystem laisse un orphelin non référencé.

### 3.8 Suppression d’une question principale et de toutes ses variantes

`DeleteQuestionHandler` collecte **avant la cascade** :

1. l’existence de la question possédée ;
2. tous les IDs de variantes illustrées via `GetAltQuestionIDsWithImage` ;
3. chaque nom d’image de variante via `GetAltImageByAltQuestionID` avec le bon `QuestionID` ;
4. l’éventuelle image principale via `GetImageByQuestionID` ;
5. seulement ensuite `DeleteQuestion`, qui cascade vers image principale, variantes, images de variantes et réponses ;
6. après contrôle strict du nombre de lignes, boucle sur tous les noms et appelle `RemoveStoredImageFile` sans interrompre la boucle en cas d’échec ;
7. redirection 303.

Tout échec de collecte avant le DELETE laisse le graphe DB et tous les fichiers intacts. Un échec du DELETE fait de même. Un crash après le DELETE mais avant ou pendant la boucle peut laisser tout ou partie des fichiers orphelins.

Exemple demandé : `main.png`, `a.png`, `b.png` sont collectés (dans l’ordre des variantes retournées, puis l’image principale). Après succès DB, si `main.png` est supprimé, `a.png` échoue et `b.png` est supprimé, l’état final est : question, variantes et trois lignes image absentes de DB ; `main.png` et `b.png` absents ; `a.png` présent mais sans référence. L’échec de `a.png` est journalisé et ne bloque pas `b.png` ni la redirection 303.

## 4. Ligne DB présente, fichier absent

| Surface | Comportement actuel | Classification |
|---|---|---|
| Page de gestion | la ligne est trouvée et un `<img src="/static/images/...">` est rendu ; la requête image répond 404, donc aperçu cassé, page HTML 200 | bug fonctionnel + incohérence |
| Route statique | ownership DB vrai, puis `validateRegularFile`/`os.Open` échoue ; réponse 404 | comportement sûr ; pas de fuite |
| Formulaire GET de taille/suppression | HTML 200 et `<img>` cassée ; la valeur DB reste affichable | bug fonctionnel mineur |
| POST taille | `ReadImageConfig` échoue avant UPDATE ; réponse 500, valeur DB inchangée | bug fonctionnel + incohérence, pas sécurité |
| Aperçu principal/variante | la ligne est transformée en chemin Typst ; l’écriture `.typ` réussit, puis `typst compile` échoue sur l’image manquante ; réponse 500 | bug fonctionnel |
| Génération d’examen | construction conserve le nom DB ; Typst échoue dans le traitement étudiant ; la génération est marquée en échec et aucun résultat final valide n’est produit | bug fonctionnel important |

Ce cas est plus nuisible à l’utilisateur qu’un orphelin filesystem. Il peut provenir d’une suppression manuelle, d’une perte/corruption du stockage, ou de la stratégie filesystem → DB ; le flux normal DB → filesystem ne le crée pas.

## 5. Fichier présent, ligne DB absente

Pour `assets/images/orphan.png` sans ligne dans `images` ou `alt_images` :

- `/static/images/orphan.png` exige authentification puis appelle `UserOwnsImage` ;
- aucun utilisateur n’obtient `owned=true` ;
- la réponse est 404 avant toute ouverture du fichier ;
- connaître ou deviner le nom ne contourne pas ce contrôle ;
- les chemins et séparateurs sont en plus rejetés par `safePathComponent`.

Conclusion : dans le modèle de menace applicatif, l’orphelin est une fuite d’espace disque et une rétention de données sur le serveur, pas un fichier servi ni une élévation d’accès. Une personne ayant déjà accès au filesystem du serveur peut évidemment le lire selon les permissions OS ; ce n’est pas un accès via LazyMarking.

## 6. Comparaison des stratégies

Notation qualitative : excellent / bon / moyen / faible.

| Stratégie | Intégrité visible | Risque de perte | Crash | Simplicité/test | Avis |
|---|---|---|---|---|---|
| A. DB puis filesystem | bon : référence retirée d’abord | faible : peut laisser un orphelin, pas perdre une image encore référencée | crash → orphelins | excellent | **à conserver** |
| B. filesystem puis DB | faible : DB peut annoncer un fichier supprimé | élevé si DELETE DB échoue | crash → ligne cassée | excellent | pire pour l’utilisateur |
| C. transaction DB seule | n’englobe pas `os.Remove` | dépend de l’ordre autour de la transaction | fenêtre de crash toujours présente | bon | utile seulement pour grouper des mutations DB |
| D. compensation DB | incertain, restauration peut échouer | risque de recréer des références vers des fichiers absents | deux opérations DB + FS, davantage de fenêtres | faible | déconseillée |
| E. quarantaine/rename | très bon si même filesystem et protocole complet | faible | éléments en quarantaine à réconcilier | moyen/faible | robuste mais disproportionnée maintenant |
| F. DB d’abord + réconciliation | bon | faible | récupère les orphelins après crash | bon | **meilleur compromis** |

### A — conserver DB → filesystem

Avantages : cohérence immédiatement visible, suppression DB ownership-aware, cascade simple, fichier absent traité idempotemment, échec limité à un objet non servi. Inconvénients : crash ou permission refusée laisse un orphelin ; le simple log n’offre pas de retry ni d’inventaire durable.

### B — filesystem → DB

Si `os.Remove` réussit puis le DELETE SQLite échoue, la ligne reste active vers un fichier définitivement absent. L’utilisateur voit une image cassée, ne peut plus régler sa taille, et aperçu/génération échouent. Cet état est nettement pire qu’un orphelin non servi. Une copie préalable serait nécessaire pour compenser, ce qui rejoint la quarantaine.

### C — transaction SQLite seule

Une transaction SQLite ne peut ni enrôler ni annuler `os.Remove` : SQLite et le filesystem n’ont pas de coordinateur transactionnel commun. Une transaction peut stabiliser les lectures de noms et la suppression du graphe parental, ou garantir plusieurs mutations DB entre elles, mais elle ne supprime pas la fenêtre de crash entre commit et suppression physique.

### D — compensation

Après DELETE DB puis échec filesystem, restaurer la ligne nécessite de conserver ID, parent, nom, taille et user. Pour une suppression parente, il faudrait restaurer question, variantes, réponses, images et relations dans un ordre compatible avec toutes les FK, contraintes d’unicité et éventuelles références concurrentes. La compensation peut elle-même échouer ou recréer une référence vers un fichier devenu incertain. Pour les suppressions directes elle est techniquement possible mais ajoute un mode d’échec plus dangereux que l’orphelin initial. Déconseillée.

### E — renommage/quarantaine

Un `rename` vers une quarantaine située sur le **même filesystem** est généralement atomique : déplacer, supprimer DB, puis effacer définitivement ; si DB échoue, renommer en retour. Mais :

- l’atomicité n’est pas garantie entre filesystems ;
- un crash après mise en quarantaine mais avant DELETE laisse une ligne cassée jusqu’à récupération ;
- un crash après commit laisse un élément de quarantaine à purger ;
- les suppressions parentes multiples exigent manifeste, rollback de plusieurs fichiers et gestion des collisions ;
- Windows et les fichiers ouverts ajoutent des différences de comportement.

C’est une option si les exigences de confidentialité imposent une disparition physique immédiate et vérifiable, mais elle est trop complexe pour le risque actuel.

### F — DB d’abord + nettoyage différé

Cette stratégie conserve le bon mode d’échec et ajoute la récupération manquante. Pour LazyMarking, un scanner déterministe et une commande de maintenance explicite suffisent ; une file distribuée ou un service externe seraient disproportionnés.

## 7. Réconciliation proposée (conceptuelle)

Construire deux ensembles exacts, après validation des noms :

- `FS =` entrées régulières directement sous `assets/images` ;
- `DB =` union des `image_name` de `images` et `alt_images`.

Résultats :

- orphelins : `FS - DB` ;
- lignes cassées : `DB - FS` ;
- cohérents : intersection.

Précautions :

1. première version **lecture seule**, avec rapport du nom, type d’écart et éventuellement âge/mtime ;
2. ne pas suivre de symlink et signaler séparément toute entrée non régulière ;
3. ne jamais supprimer sur un scan incomplet : si lecture DB ou listing FS échoue, arrêter sans mutation ;
4. purge des orphelins uniquement sur commande explicite, avec délai de grâce (par exemple 24 h) pour ne pas supprimer un upload en cours entre écriture et INSERT ;
5. ne jamais supprimer automatiquement une ligne DB cassée : la signaler, car elle représente une perte potentielle et nécessite une décision/restauration ;
6. ignorer ou signaler séparément les fichiers connus comme actifs temporaires si le répertoire en accueille un jour ; actuellement les uploads permanents sont écrits directement dans ce répertoire.

Moment recommandé : commande d’administration/maintenance à la demande, utilisable manuellement et dans une sauvegarde opérée. Au démarrage, exécuter au plus un **audit non destructif** journalisé ; éviter une purge automatique qui rallongerait le startup ou agirait pendant un état transitoire. Une périodicité peut être ajoutée plus tard seulement si les métriques montrent une accumulation réelle.

## 8. Upload avant ownership

Les deux POST appellent `CheckImageFile` avant même de parser les IDs, puis écrivent le fichier permanent et lancent OpenCV avant que `CreateImage`/`CreateAltImage` ne valide définitivement le parent. Le contrôle SQL empêche toute création non autorisée et le nettoyage est tenté : il n’y a pas de violation d’ownership DB.

Il est néanmoins préférable de déplacer une vérification explicite avant le travail lourd :

- principale : après parsing multipart/IDs au minimum, `GetQuestionByID` avant écriture permanente et OpenCV ;
- variante : `GetAltQuestionByParentID(question_id, alt_question_id, user_id)` avant écriture permanente et OpenCV.

Idéalement, les champs texte seraient obtenus avant décodage complet, mais le contrat multipart actuel parse et valide le fichier en premier ; une amélioration simple peut déjà vérifier le parent immédiatement après `CheckImageFile`, avant `SaveUploadedFile` et OpenCV.

Classement : **robustesse et performance**, avec un aspect de durcissement contre la consommation abusive de CPU/disque par un utilisateur authentifié. Ce n’est pas une faille d’accès, car l’INSERT reste ownership-aware. Cela réduit aussi les occasions de créer un orphelin sur parent rejeté.

## 9. Username dans le nom physique

`SanitizeFilename` construit `{userID}_{username}_{type}_{parentID}_{basename}` puis `SaveUploadedFile` applique `safePathComponent`.

`safePathComponent` rendrait impossible un upload si le username contenait `/`, `\` ou NUL (ou si le nom final devenait un chemin non basename). Cependant, l’inscription actuelle impose `^[[:alnum:]_.-]{3,64}$` : **aucun username créé par le flux actuel ne contient ces caractères**, et `.`/`..` seuls ne posent pas problème une fois préfixés. Le problème n’est donc pas réel pour les comptes actuels validés ; il peut seulement concerner des données legacy/importées qui auraient contourné cette validation.

À terme, le nom physique gagnerait à ne pas incorporer le username mutable et visible. `userID + type + parentID + UUID` (ou basename normalisé si sa conservation est utile) est plus stable, évite les caractères hérités et réduit les collisions. C’est une amélioration P2, pas un correctif urgent. Une migration des noms existants n’est pas justifiée pour ce seul motif.

## 10. Inventaire des tests actuels

| Scénario | État | Preuve / manque |
|---|---|---|
| Ajout principal succès | non couvert | aucun test handler multipart `AddImageHandler` |
| Ajout variante succès | non couvert | aucun test handler multipart `AddAltImageHandler` |
| Parent étranger | partiellement couvert | SQL `CreateImage` à 0 ligne couvert ; formulaires GET ownership couverts ailleurs ; pas le POST upload + nettoyage |
| Mauvais parent variante | partiellement couvert | GET table/suppression et UPDATE SQL couverts ; pas `AddAltImageHandler` multipart |
| Unicité / seconde image | partiellement couvert | contrainte/schema présente et DB testée indirectement ; aucun test handler vérifiant le nettoyage après refus |
| Nettoyage fichier après INSERT échoué | non couvert | aucune injection d’échec INSERT/cleanup dans les handlers d’ajout |
| Suppression principale DB + fichier | non couvert au niveau handler | `DeleteImage` rows et helper fichier sont testés séparément, pas le flux complet |
| Suppression variante DB + fichier | couvert | `TestDeleteAltImageHandlerRemovesOwnedVariantImageAndFile` |
| Suppression parent avec images | partiellement couvert | variante parente + fichier couvert ; question parente avec une image de variante couverte ; pas image principale + plusieurs variantes ensemble |
| Fichier déjà absent | partiellement couvert | helper `RemoveStoredImageFile` couvert ; pas les quatre handlers destructifs |
| Échec réel de suppression filesystem | partiellement couvert | helper rejette répertoire symbolique ; aucun handler ne vérifie DB supprimée + poursuite/réponse/log |
| DB présente / fichier absent | partiellement couvert | helper de conversion manque fichier et edit image absent en DB couverts ; pas UI/static/Typst avec ligne DB cassée de bout en bout |
| Fichier présent / DB absente | non couvert | aucune preuve handler statique 404 sur orphelin |
| Route static owner | non couvert | pas de test `ServeUserImageHandler` trouvé |
| Route static autre utilisateur | non couvert | pas de test `ServeUserImageHandler` trouvé |

Tests connexes solides : validation de contenu image, dimensions et multipart ; sécurité de `SaveUploadedFile`; suppression idempotente et sécurité symlink de `RemoveStoredImageFile`; compte strict des lignes critiques ; cascades DB ; correction de lecture des images de variantes ; sécurité du chemin Typst.

## 11. Priorités

### P0 — 0

Aucune corruption, perte active ou exposition applicative exigeant un blocage avant utilisation réelle n’a été identifiée. Le résidu après échec actuel est inaccessible via la route statique sans ligne DB.

### P1 — 3

1. **Réconciliation et observabilité** : scanner exact en lecture seule, rapporter orphelins et lignes cassées, puis offrir une purge explicite des seuls orphelins avec délai de grâce. Conserver les logs des handlers avec le nom concerné.
2. **Prévalidation des parents à l’ajout** : vérifier question/variante possédée avant écriture permanente et OpenCV afin de limiter CPU/disque inutiles et chemins de nettoyage.
3. **Couverture des modes d’échec critiques** : ajouts avec INSERT refusé + nettoyage, suppression directe principale, suppressions parentes multiples, échec filesystem après succès DB, et route statique owner/foreign/orphelin.

### P2 — 2

1. **Nommage physique indépendant du username** pour les nouveaux uploads, sans migration précipitée des fichiers existants.
2. **Audit non destructif optionnel au startup ou périodique**, seulement après validation de la commande à la demande et si le besoin opérationnel est démontré.

## 12. Jalons d’implémentation recommandés

1. **Jalon 1 — scanner de cohérence en lecture seule** : fonction pure autant que possible, listing sécurisé, lecture des deux tables, résultat structuré (`orphans`, `missing`, `unsafe`), tests avec plusieurs fichiers/lignes et échec de scan ; aucune suppression.
2. **Jalon 2 — tests des handlers destructifs et injection contrôlée d’échec filesystem** : vérifier explicitement DB supprimée, orphelin laissé, autres fichiers encore traités et réponse 303.
3. **Jalon 3 — commande de purge explicite** : supprimer uniquement les orphelins réguliers plus anciens qu’un délai de grâce ; dry-run par défaut ; jamais de mutation des lignes cassées.
4. **Jalon 4 — prévalidation ownership des uploads** avant écriture permanente/OpenCV, avec tests parent absent/étranger/mal apparié et absence de fichier créé.
5. **Jalon 5 — simplification du nommage des nouveaux fichiers**, séparée de toute migration éventuelle.

La quarantaine ne devrait être reconsidérée que si une exigence future impose la suppression physique immédiate ou si la réconciliation révèle une fréquence d’échec significative. Dans l’état actuel, elle augmenterait sensiblement le code, les fenêtres de crash et le coût de test sans améliorer la disponibilité visible autant que la solution F.
