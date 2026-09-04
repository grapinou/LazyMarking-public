# Contrôle final du cycle de vie d'un désaccord hybride non résolu

Date : 2026-09-04

## Conclusion

Le contrôle a identifié puis corrigé une anomalie de cycle de vie : un job avec des revues en attente affichait un avertissement, mais ses artefacts initiaux étaient encore considérés « courants » lorsque `review_revision=artifacts_revision=0`. Les liens vers le PDF corrigé et le tableau de notes restaient donc visibles, l'endpoint de téléchargement ne consultait pas le statut de revue, et l'entrée de régénération n'interdisait pas explicitement une file hybride non résolue.

Après correction, le cycle de vie est **sûr** : pour un job hybride, la présence d'au moins un désaccord non revu masque les artefacts finaux, bloque leur téléchargement direct et interdit leur régénération. La résolution du dernier désaccord recalcule les résultats depuis les états effectifs humains, régénère les deux artefacts et synchronise leurs révisions. Un job hybride sans désaccord suit directement le chemin final normal. Les jobs historiques conservent leur logique de bande et leur comportement antérieur.

Aucun algorithme de détection, seuil V2, règle pédagogique ou calcul de score n'a été modifié.

## Chemin audité

Le chemin complet est désormais :

`image alignée` → détecteurs historique et V2 → politique d'accord → persistance → score initial technique → statut de revue dérivé → page résultat protégée → revue humaine → recalcul → révision → dernière revue → régénération atomique → téléchargement final.

Le champ SQL `marking_jobs.status='success'` signifie que le traitement technique du scan et la persistance sont terminés. Il ne suffit pas à rendre un résultat hybride définitif : l'état utilisateur est dérivé séparément par `GetMarkingReviewSummary` et `DeriveMarkingReviewStatus` (`pending`, `completed` ou `no_review_needed`).

## Consommation des trois états

### `detected_state`

`detected_state` est le champ historique non nullable. Pour un nouveau job hybride, il reçoit l'état du détecteur historique. Il est consommé :

- pendant le scoring initial, avant toute revue ;
- comme repli technique dans `COALESCE(reviewed_state, detected_state)` ;
- pour dessiner les premiers artefacts produits par le pipeline ;
- pour afficher à l'opérateur la proposition provisoire dans la revue.

Il est donc exact que le scoring initial peut techniquement utiliser `detected_state` pour un désaccord. `COALESCE` n'ignore pas ce désaccord. Ce score et les artefacts correspondants sont provisoires tant que la file hybride contient une revue en attente.

### `automatic_state`

`automatic_state` explicite l'autorité de la politique :

- accord `0/0` : valeur 0 ;
- accord `1/1` : valeur 1 ;
- désaccord : `NULL`.

Il sert à l'audit et à empêcher d'interpréter le support provisoire historique comme une décision automatique. Le scoring ne lit pas cette métrique : il continue volontairement à consommer seulement des états binaires effectifs.

### `reviewed_state`

`reviewed_state` est écrit dans `marking_answer_reviews` par l'utilisateur. Dès qu'il existe, les requêtes d'état effectif utilisent `COALESCE(reviewed_state, detected_state)` et donnent donc autorité à la décision humaine. Le service de revue recalcule la question touchée puis la note totale de la copie à partir du vecteur effectif complet.

## Avant la revue

Un désaccord hybride est persisté avec :

- `detected_state` égal à l'état historique, comme support provisoire compatible ;
- `automatic_state=NULL` ;
- `review_reason='detector_disagreement'` ;
- aucune ligne `marking_answer_reviews` avant intervention humaine.

La requête hybride de candidats sélectionne exactement ce motif, indépendamment de `ambiguity_delta`. Le résumé compte les candidats sans revue et produit le statut applicatif `pending`.

Le score initial existe techniquement parce que le schéma historique exige des résultats binaires et que la génération du job doit terminer. Il n'est plus accessible comme résultat final : la page annonce les réponses à vérifier, les liens finaux sont absents et l'endpoint refuse le téléchargement.

## Protection contre une finalisation prématurée

Trois gardes complémentaires sont maintenant en place pour les jobs dont `review_policy_version='detector-agreement-v1'` :

1. **Page résultat** : si le statut est `pending`, aucun lien vers le PDF corrigé ou le tableau de notes n'est construit, même lorsque les deux révisions valent encore zéro.
2. **Téléchargement direct** : le handler extrait strictement le job de l'opération `marking-<id>`, vérifie l'appartenance en base, charge le nombre de candidats en attente et répond `409 Conflict` si un désaccord hybride reste non résolu. Il refuse aussi tout artefact dont la révision ne correspond pas à celle des revues.
3. **Régénération** : le service de régénération reçoit désormais `pending_candidates` dans sa cible et retourne `ErrMarkingArtifactsUnavailable` avant tout appel au générateur lorsqu'un job hybride possède une revue en attente.

Ainsi, connaître ou reconstruire manuellement l'URL ne contourne pas la revue. Les fichiers provisoires peuvent exister dans l'espace privé du job car le pipeline les produit avant de connaître les choix humains, mais aucun chemin HTTP applicatif ne les présente comme définitifs.

## Dernière revue et finalisation

Chaque revue applique transactionnellement les étapes existantes :

1. création ou mise à jour de `reviewed_state` ;
2. chargement de tous les états effectifs de la question ;
3. recalcul pédagogique par le moteur existant ;
4. mise à jour de la question et de la somme de copie ;
5. incrément atomique de `review_revision`, avec contrôle de concurrence optimiste ;
6. conservation d'`artifacts_revision` en retard lorsqu'un état effectif a changé.

Tant qu'un autre candidat subsiste, le handler redirige vers la revue suivante et ne régénère rien. Après la dernière revue, `ListPendingMarkingReviewCandidates` est vide : le handler lance alors la régénération des deux PDF à partir des états effectifs, publie la paire de manière atomique, puis avance `artifacts_revision` jusqu'à `review_revision`. Les liens finaux ne redeviennent visibles qu'une fois les révisions synchronisées.

Si la régénération échoue, les choix humains et les scores recalculés restent enregistrés, les artefacts restent marqués obsolètes et les liens finaux restent masqués ; la page propose une nouvelle tentative.

## Job hybride sans désaccord

Lorsque tous les couples sont `0/0` ou `1/1` :

- chaque détection possède un `automatic_state` ;
- aucune ligne ne porte `review_reason='detector_disagreement'` ;
- le résumé contient zéro candidat ;
- le statut applicatif est `no_review_needed` ;
- `review_revision` et `artifacts_revision` sont égales dès la sortie du pipeline ;
- les deux artefacts finaux sont immédiatement proposés.

Aucune revue artificielle et aucune régénération supplémentaire ne sont déclenchées.

## Compatibilité historique

Les requêtes continuent de reconnaître les anciens jobs par l'absence de `review_policy_version`. Leurs candidats restent déterminés par :

`ABS(mean_gray - detection_threshold) <= ambiguity_delta`.

Les nouveaux gardes sur `pending_candidates` sont conditionnés par la présence de la politique hybride. La présentation historique des artefacts pendant une ancienne file de revue n'a donc pas été changée par ce contrôle. Les résultats anciens sans fonction de revue restent lisibles et téléchargeables selon le chemin existant, sous réserve du contrôle d'appartenance et de cohérence des révisions déjà applicable aux résultats modernes.

## Correction effectuée

Les changements sont minimaux et localisés :

- `db/query/markingReviews.sql` enrichit la cible d'artefacts avec `review_policy_version` et un compte exact des candidats non revus selon la politique du job ;
- les fichiers sqlc correspondants ont été régénérés ;
- `buildMarkingResultPageData` masque les deux artefacts finaux pendant une revue hybride en attente ;
- `ServeFullMarkingPdfHandler` vérifie le job, les candidats hybrides en attente et les révisions avant de servir un PDF ;
- `RegenerateMarkingArtifacts` refuse la régénération hybride avant résolution complète ;
- seuls des tests de cycle de vie ont été ajoutés ou ajustés.

La détection historique, V2, la politique de désaccord et le package de scoring n'ont reçu aucun changement.

## Tests

La couverture vérifie désormais explicitement :

- job hybride avec désaccord non résolu : aucun lien final ;
- téléchargement direct d'un PDF provisoire : `409 Conflict` ;
- tentative de régénération avec candidat hybride en attente : refus avant appel du générateur ;
- résolution du dernier désaccord : état humain enregistré, zéro candidat restant, régénération appelée et révisions synchronisées ;
- résultat final courant après revue : liens disponibles ;
- job hybride sans désaccord : aucun candidat et chemin final direct ;
- job historique : sélection par bande et comportement de présentation antérieur préservés.

Ces tests complètent les tests transactionnels existants qui contrôlent déjà le recalcul de question, la somme de copie, l'idempotence, les conflits de révision, l'échec récupérable de régénération et la publication atomique de la paire de PDF.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès ;
- `go test ./...` : succès ;
- `./scripts/check.sh` : succès, y compris replay Goose jusqu'à la migration 0042, formatage, vet, tests et build ;
- `git diff --check` : succès.

Aucun commit n'a été effectué.
