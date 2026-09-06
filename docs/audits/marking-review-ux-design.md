# Design UX de la revue humaine Marking

## Périmètre et conclusion

Ce jalon est un audit/design en lecture seule. Aucun handler, route, template,
view model, service, schéma ou comportement production n'est modifié.

Le workflow recommandé est une revue séquentielle server-rendered, accessible
depuis la page de résultat du job. Le professeur valide chaque case proposée,
est redirigé automatiquement vers la suivante, puis une unique régénération des
artefacts est tentée après la dernière décision. Si elle échoue, les décisions
restent enregistrées et un bouton permet de réessayer sans refaire la revue.

## UX Marking actuelle

### Routes et handlers

Toutes les routes Marking sont sous `/dashboard/marking`, enregistrées dans
`internal/handlers/marking/routes.go` et protégées par `login.CheckAuth` :

| Méthode | Route | Handler | Rôle actuel |
|---|---|---|---|
| GET | `/dashboard/marking` | `AddPdfFormMarkingHandler` | choix d'une génération et dépôt du paquet PDF |
| POST | `/dashboard/marking/processing` | `ProcessingMarkingHandler` | création du job et lancement asynchrone |
| GET | `/dashboard/marking/progress?job_id=…` | `ProgressMarkingHandler` | polling de progression |
| GET | `/dashboard/marking/success?job_id=…` | `SuccessMarkingProcessingHandler` | résultat terminal et préparation des URLs PDF |
| GET | `/dashboard/marking/servePDF?operation=…&file=…` | `ServeFullMarkingPdfHandler` | lecture confinée d'un PDF du workspace |

Les conventions présentes utilisent des routes fixes et des IDs en query string
pour les GET, puis des champs de formulaire pour les POST. Le router est le
`http.ServeMux` standard avec patterns méthode + chemin.

### Lifecycle et page de statut

Après le POST initial, un PRG redirige vers la progression. Le template
`progress_marking.html` recharge la page toutes les deux secondes et montre
séparément pages, copies et `status_pdf`, mais affiche directement les valeurs
techniques `status`/`status_pdf`.

- `running` reste sur la page de progression ;
- `failed` redirige vers une page d'erreur générique, avec un message long et
  familier ;
- `success` + `status_pdf=success` redirige vers la page de succès ;
- les outcomes `incomplete`, `error` et `not_seen` ne sont pas présentés comme
  états structurés sur cette page : ils se matérialisent surtout dans
  `corrected_NOT.pdf` et les informations du tableau de notes.

Le lifecycle technique du job et le lifecycle de review ne sont actuellement
pas représentés ensemble. La nouvelle UX doit les garder orthogonaux.

### Page de succès et téléchargements

`success_marking_processing.html` annonce une correction réussie puis ouvre
automatiquement `corrected.pdf`, `mark-table.pdf` et éventuellement
`corrected_NOT.pdf` dans des popups au chargement. Il n'y a pas de boutons de
téléchargement visibles ni de notion d'artefact stale. Le template contient en
outre un `console.log` de diagnostic.

`ServeFullMarkingPdfHandler` vérifie l'authentification, puis délègue à un helper
qui confine `username`, `operation` et `filename`, refuse traversals, symlinks et
fichiers non réguliers, et sert le PDF inline. En revanche, le handler construit
son accès depuis les paramètres filesystem plutôt que depuis un `job_id`
ownership-aware ; la sécurité repose donc sur le username de session et le
confinement du workspace. Les futurs liens finaux devraient être dérivés d'une
lecture du job possédé et ne jamais être affichés comme actuels lorsque les
révisions divergent.

### Templates et view models

Les trois pages utilisent `data.MarkingPageData` avec `ExtraData map[string]any`.
Les templates connaissent des clés libres (`Status`, `StatusPdf`, `JobID`, URLs
PDF, compteurs). Cette forme masque les contrats, favorise les erreurs de clé et
mélange routage, présentation et données métier.

Le socle visuel commun est Bootstrap 5.3, Bootstrap Icons, une largeur
`container-xl`, une navigation responsive et du rendu Go server-side. La page
d'upload utilise du JavaScript pour le drag-and-drop, mais le futur workflow de
review n'en a pas besoin.

## Point d'entrée et page résultat recommandés

Le point d'entrée est la page actuelle
`/dashboard/marking/success?job_id=…`, à faire évoluer en véritable page résultat
du job. Elle connaît déjà le job possédé, sépare naturellement la fin du
traitement et centralise les artefacts.

Pour un job technique `success`, elle doit afficher exactement un état de review :

| État review | Présentation et action principale |
|---|---|
| `no_review_needed` | « Aucune réponse à vérifier » ; PDF finaux disponibles si current |
| `pending` | « N réponses à vérifier » et bouton primaire « Vérifier les réponses » |
| `completed`, artefacts current | « Toutes les réponses ont été vérifiées » ; PDF finaux disponibles |
| `completed`, artefacts stale | « Réponses enregistrées — PDF à actualiser » ; bouton « Actualiser les PDF » |
| `legacy_unavailable` | « Revue assistée non disponible pour cette ancienne correction » ; PDF historiques conservés |

Un job `running` ou `failed` ne doit pas acquérir artificiellement un état de
review exploitable. Les copies `incomplete`, `error` et `not_seen` restent dans
un bloc séparé « Copies non corrigées / pages à contrôler » ; elles ne sont
jamais comptées parmi les réponses ambiguës.

## Workflow professeur

1. La page résultat annonce « N réponses à vérifier ».
2. Le bouton « Vérifier les réponses » ouvre le premier candidat pending.
3. Une page affiche une seule case, la progression et le contexte pédagogique
   strictement utile.
4. Le professeur choisit « Non cochée » ou « Cochée », puis « Valider et
   suivante ».
5. Le POST applique la décision et redirige vers le prochain candidat pending.
6. Après la dernière décision, la page de fin annonce que toutes les réponses
   sont vérifiées et tente une seule régénération des deux PDF.
7. En cas de succès, la page résultat présente les PDF finaux. En cas d'échec,
   elle conserve la réussite de la review et propose de réessayer la génération.

Confirmation et override ont exactement le même vocabulaire : le professeur
indique seulement ce qu'il voit. Leur différence reste interne au backend.

## Écran séquentiel de review

Contenu recommandé :

- titre « Vérification des réponses » ;
- progression visible, par exemple « Réponse 1 sur 3 » ;
- identité de la copie limitée au nom/prénom utile au professeur, provenant du
  snapshot historique, sans exposer d'identifiant technique ;
- « Question 4 — réponse B » (index humain 1-based et lettre quand raisonnable) ;
- crop large et responsive ;
- texte « Détection automatique : non cochée » ou « cochée » ;
- deux contrôles radio ou boutons de choix explicitement libellés « Non cochée »
  et « Cochée » ;
- bouton primaire « Valider et suivante ».

La couleur peut renforcer l'état, jamais le porter seule. L'image reçoit un alt
neutre tel que « Extrait de la case à vérifier », sans identité. L'ordre DOM est
celui du workflow, les labels sont associés aux champs et le focus après PRG
revient sur le titre ou le choix. Sur tablette et ordinateur, le crop peut
occuper la colonne principale ; sur mobile, il passe au-dessus de deux grands
boutons empilés ou occupant chacun une demi-largeur confortable.

`MeanGray`, seuil et delta sont absents de la vue normale. Un `<details>` discret
« Informations de diagnostic » peut ultérieurement afficher ces valeurs aux
utilisateurs qui en ont besoin, sans les faire entrer dans la décision métier.

## Crop sécurisé à la demande

La génération à la demande est recommandée. Elle évite un second stockage, les
problèmes de purge et les artefacts identifiants. La source est exclusivement la
page PNG alignée non annotée de `marking_aligned_pages`, résolue avec le helper
sécurisé existant. Aucun path filesystem n'apparaît dans l'URL ou la réponse.

Une primitive applicative partagée, conceptuellement
`ResolveMarkingReviewCandidateROI`, doit réaliser une seule fois le mapping :

```text
job possédé + answer_detection
  -> copy_result + student_exam + question_index + answer_index
  -> student_exam_page_content ordonné
  -> page_exam + index local de réponse
  -> centre + rayon du snapshot
  -> marking_aligned_pages résolue
```

Elle retourne un DTO technique (page résolue, centre, rayon), jamais une row sqlc
brute. La règle de crop doit partager la même définition de ROI centrale que la
mesure MeanGray, avec une marge d'inspection définie une seule fois. Le handler
ne refait aucun offset.

Le GET crop doit vérifier : session, ownership du job, `status=success`, lien de
la détection au job, outcome `corrected`, unicité du mapping page/ROI, intégrité
de la page alignée et dimensions. Il décode et découpe en mémoire, encode en PNG,
retourne `Content-Type: image/png`, `X-Content-Type-Options: nosniff`, et hérite
du `Cache-Control: no-store` de l'auth middleware. Un
`Content-Disposition: inline` avec un nom générique est possible. Les erreurs
cross-user et IDs inconnus doivent être indistinguables (404 selon la convention
actuelle), sans log de contenu pédagogique.

## POST de décision et concurrence

Le POST parse uniquement :

- `job_id` et `answer_detection_id` ;
- `reviewed_state` dans `{0,1}` ;
- la révision attendue de la review de réponse, nullable si elle n'existe pas ;
- `expected_review_revision` du job.

Après contrôle de forme, il appelle `ApplyMarkingAnswerReview`. Aucun scoring,
aucune formule d'ambiguïté et aucun accès filesystem ne résident dans le
handler. Une réussite applique PRG vers le prochain pending.

En cas de conflit optimistic locking, le choix proposé n'est pas réappliqué
silencieusement. Un message flash conceptuel indique : « Cette correction a été
modifiée dans un autre onglet. La page a été actualisée. » Le handler redirige
vers la review du job ; le GET recharge les révisions et ouvre le premier pending
courant. Si la détection a déjà été revue, il passe à la suivante. Cette
stratégie évite le last-write-wins et une page d'erreur technique.

Le dépôt n'a pas de middleware/token CSRF explicite. Il possède une session
signée `HttpOnly`, `Secure` configurable, `SameSite=Strict`, des routes POST
séparées, l'auth middleware, le contrôle de méthode et des contrôles ownership en
DB. Le futur POST doit conserver toutes ces conventions. La mise en place d'une
protection CSRF globale est une dette de sécurité transverse ; elle ne devrait
pas être inventée uniquement et différemment dans Marking.

## Régénération et artefacts stale

### Stratégie retenue

Ne pas régénérer après chaque review. Le coût PDF, Typst et filesystem serait
payé N fois, augmenterait les surfaces d'échec et ralentirait le parcours sans
bénéfice utilisateur.

La stratégie recommandée est : une tentative automatique unique juste après la
dernière candidate pending, puis redirection vers la page résultat. Elle combine
la simplicité de l'option « pending atteint zéro » avec un bouton explicite de
retry/finalisation uniquement lorsque nécessaire.

Une modification humaine ultérieure rendra de nouveau les artefacts stale ; la
page résultat présentera alors le même bouton « Actualiser les PDF ». Cette
politique s'étend naturellement à un futur éditeur de réponses non ambiguës.

### Échec

Une erreur de régénération ne remet pas en cause les décisions ni les scores DB.
Afficher : « Les réponses sont enregistrées, mais les PDF n'ont pas pu être
actualisés. » Puis proposer le POST idempotent « Réessayer la génération ». Le
job reste techniquement `success`; aucun professeur ne refait la revue.

### Téléchargements stale

Quand `artifacts_revision < review_revision`, masquer les liens finaux vers
`corrected.pdf` et `mark-table.pdf` et afficher à leur place « Mise à jour
nécessaire ». Un bouton désactivé serait frustrant et conserverait l'ambiguïté
sur le contenu. Les anciens fichiers ne doivent jamais être présentés comme
finaux. `corrected_NOT.pdf`, indépendant des answer reviews, peut rester proposé
dans un bloc distinct si présent.

## View models dédiés

Proposition conceptuelle, sans dépendance Bootstrap et sans row sqlc exposée :

```go
type MarkingResultPageData struct {
    Routes             MarkingRoutes
    PageTitle          string
    JobID              int64
    TechnicalStatus    string
    ReviewStatus       MarkingReviewStatusView
    TotalCandidates    int64
    ReviewedCandidates int64
    PendingCandidates  int64
    ArtifactsCurrent   bool
    FinalPDFs          MarkingArtifactLinksView
    NonCorrected       MarkingNonCorrectedSummaryView
    Flash              NoticeView
}

type MarkingReviewPageData struct {
    Routes       MarkingRoutes
    PageTitle    string
    JobID        int64
    Position     int
    Total        int
    Candidate    MarkingReviewCandidateView
    JobRevision  int64
    Flash        NoticeView
}

type MarkingReviewCandidateView struct {
    DetectionID           int64
    StudentDisplayName    string
    QuestionNumber        int
    AnswerLabel           string
    DetectedStateLabel    string
    CurrentReviewedState  *int
    AnswerReviewRevision  *int64
    CropURL                string
}
```

Les statuts métier peuvent rester typés en interne ; les labels français et les
classes Bootstrap sont déterminés par la couche de présentation. `ExtraData` et
les rows sqlc disparaissent des nouvelles vues.

## Routes minimales proposées

Pour rester cohérent avec les routes fixes et IDs query/form du dépôt :

| Méthode | Route | Paramètres | Rôle |
|---|---|---|---|
| GET | `/dashboard/marking/review?job_id=…` | query `job_id` | premier candidat pending ou fin de review |
| GET | `/dashboard/marking/review/crop?job_id=…&answer_detection_id=…` | query | crop PNG ownership-aware |
| POST | `/dashboard/marking/review/apply` | formulaire | décision puis PRG |
| POST | `/dashboard/marking/artifacts/regenerate` | formulaire `job_id` + révision attendue si utile | retry/finalisation des PDF |

La page résultat reste `/dashboard/marking/success?job_id=…` durant cette
première implémentation afin de ne pas casser les redirections existantes. Un
renommage ultérieur vers `/result` serait cosmétique.

## Review hors ambiguïté et évolution

La V1 ne liste que les candidats automatiques et ne crée pas un éditeur complet.
Elle ne doit toutefois pas enfouir l'identité du job/copy/detection dans la vue :
la primitive ROI et le POST acceptent déjà toute détection possédée, conformément
au backend. Une évolution naturelle pourra ajouter depuis le détail d'une copie
« Corriger une autre réponse », réutilisant exactement la même page de décision
et le même POST, sans modifier la file automatique ni ses compteurs.

## Tests à prévoir

- résultat `no_review_needed`, `pending`, `completed/current`,
  `completed/stale`, `legacy_unavailable` ;
- séparation des copies `incomplete`, `error`, `not_seen` ;
- crop owner, cross-user, job/détection incohérents, page absente/corrompue ;
- mapping exact question/réponse vers page et ROI, y compris frontières de page ;
- PNG, headers `no-store`, type et disposition ;
- POST confirmation et override ;
- `reviewed_state` absent, non numérique ou hors `{0,1}` ;
- conflit de révision avec message et rechargement ;
- prochain candidat et dernière candidate ;
- régénération unique après dernière candidate ;
- succès, échec et retry de régénération ;
- PDF stale non présenté comme final ;
- running/failed refusés ;
- legacy sans file artificielle et PDF historiques disponibles ;
- navigation clavier, labels et rendu responsive de base.

## Dettes UX adjacentes

### À traiter avec la review UX

- remplacer les popups automatiques par des boutons explicites ;
- retirer le `console.log` de la page succès ;
- remplacer `ExtraData` par les view models dédiés ;
- masquer le vocabulaire `status_pdf`, les statuts anglais et les IDs bruts ;
- distinguer résultat technique, review et copies non corrigées ;
- empêcher la présentation finale des PDF stale ;
- harmoniser erreurs et conflits avec des messages français non techniques ;
- rendre la page succès réellement responsive et accessible.

### Peut attendre la clôture P3

- transformer toutes les anciennes routes query-string en routes path-based ;
- uniformiser l'ensemble des pages Marking avec les autres listes du dashboard ;
- remplacer le meta-refresh par du polling progressif ;
- refondre le drag-and-drop d'upload et son JavaScript ;
- généraliser les notices flash à toute l'application ;
- déployer une protection CSRF transverse ;
- ajouter l'éditeur complet des réponses non ambiguës ;
- revoir le téléchargement générique `operation`/`file` au profit d'un endpoint
  strictement job/artifact ownership-aware.

## Roadmap d'implémentation

1. Introduire les view models dédiés et transformer la page résultat en lecture
   ownership-aware du lifecycle review et de la fraîcheur des artefacts.
2. Ajouter la primitive partagée de mapping ROI et le GET crop sécurisé.
3. Ajouter la page séquentielle et le POST PRG vers
   `ApplyMarkingAnswerReview`, avec gestion du conflit.
4. Brancher la progression automatique et l'écran de fin.
5. Déclencher une régénération unique après la dernière candidate et ajouter le
   POST de retry.
6. Verrouiller la politique des liens stale et les cas legacy/non-corrected.
7. Finaliser le responsive, l'accessibilité, les messages et supprimer les
   popups/termes techniques.

La première étape d'implémentation doit être le read model et la nouvelle page
résultat : elle rend visibles les cinq états sans mutation, fournit le point
d'entrée stable de la review et permet de tester le contrat UX avant d'ajouter
le crop ou les POST.
