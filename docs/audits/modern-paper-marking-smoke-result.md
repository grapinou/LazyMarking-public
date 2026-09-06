# Audit post-smoke du premier marquage papier moderne

Date de l'audit : 2026-09-02
Nature : audit en lecture seule, sans rejeu du pipeline, sans modification de la DB, des reviews ou des artefacts.

## Verdict

**B — pipeline fonctionnel avec une anomalie non bloquante.**

Le premier parcours papier moderne est techniquement validé : les quatre pages physiquement mélangées ont été reconnues et remises dans le bon ordre logique, les deux copies ont été corrigées, l'ambiguïté a été soumise à une review, les scores sont cohérents avec les états effectifs, et les références modernes ont été utilisées pour les quatre pages. Le seul écart observé est une arborescence de staging vide laissée dans le workspace ; elle est classée P3 et n'empêche ni l'exploitation du résultat ni un essai en classe.

## Sources examinées

- DB smoke : `/home/sighto/Documents/lazymarking-smoke/app.db`
- scan papier, consulté uniquement en lecture : `/home/sighto/Documents/lazymarking-smoke/smoke test result.pdf`
- PDF original généré : `/home/sighto/Documents/lazymarking-smoke/Évaluation-smoke-Capitales.pdf`
- workspace du job : `/home/sighto/Documents/lazymarking-smoke/runtime/assets/tmp/smoke-prof/marking-1/`

Aucun PDF, scan, QR brut, crop ou résultat nominatif n'a été copié dans le dépôt. Les deux copies sont désignées ci-dessous par `copy-A` et `copy-B`.

## Intégrité des sources

Empreintes SHA-256 au début de l'audit :

| Source | SHA-256 |
|---|---|
| DB smoke | `d45d6faeef854fd4232244b791b1d26a634f6f4ced78a97270097313b8f10fde` |
| scan papier | `8ea065e83b141a4ea4311c0a46b75f5538db004558a96ed15d1099597c63fc25` |
| PDF original | `4f8417233085e36f7039c2d5b987dfff9e04422389cdc3ee27e57a6714619b08` |

Les mêmes empreintes ont été retrouvées à la fin de l'audit. `PRAGMA integrity_check`, exécuté via une connexion SQLite en lecture seule avec `query_only`, retourne `ok`.

## Job identifié

| Propriété | Valeur |
|---|---:|
| job id | 1 |
| generation id | 2 |
| statut métier | `success` |
| statut PDF | `success` |
| pages totales / traitées | 4 / 4 |
| copies attendues / traitées | 2 / 2 |
| fin du traitement | 2026-09-02 11:49:44 (horodatage DB) |
| version de schéma résultat | 1 |
| version d'algorithme | 1 |
| seuil de détection | 150 |
| delta d'ambiguïté | 5 |
| review_revision | 1 |
| artifacts_revision | 1 |

Le job est lié à l'évaluation attendue. Les deux `student_exam` reconnus appartiennent tous deux à la génération 2. Aucun `failure_code`, `failure_detail` ou résultat en erreur n'est persisté.

## Mapping QR et pages mélangées

Le scan contient quatre pages. Un décodage isolé des QR, sans appel au traitement Marking, a donné l'ordre physique anonymisé suivant :

| Page physique du scan | Copie | Page logique |
|---:|---|---:|
| 1 | copy-B | 2 |
| 2 | copy-A | 1 |
| 3 | copy-B | 1 |
| 4 | copy-A | 2 |

Résultat :

- QR décodés : 4/4 ;
- pages exactement rattachées : 4/4 ;
- copies distinctes : 2 ;
- pages logiques par copie : 2/2 pour chacune ;
- QR d'une autre génération : 0 ;
- page logique impossible : 0 ;
- doublon problématique : 0 ;
- page non résolue : 0.

L'ordre physique est réellement mélangé (`copy-B/2`, `copy-A/1`, `copy-B/1`, `copy-A/2`) et a été correctement reconstruit par l'identité QR et `page_exam`.

## Résultats persistés et review

Les tables modernes contiennent :

- 2 `marking_copy_results`, tous deux avec l'outcome `corrected` ;
- 10 `marking_question_results`, soit 5 par copie ;
- 40 `marking_answer_detections`, soit 4 réponses par question ;
- 1 `marking_answer_reviews` ;
- 4 `marking_aligned_pages`.

Statistiques anonymisées de détection :

| Mesure | Nombre |
|---|---:|
| états automatiquement détectés cochés | 8 |
| états automatiquement détectés non cochés | 32 |
| réponses dans la bande d'ambiguïté `abs(mean_gray - 150) <= 5` | 1 |
| questions concernées par une ambiguïté | 1 |
| réponses effectivement reviewées | 1 |
| décisions humaines changeant l'état effectif | 0 |

Pour l'unique cas ambigu, `detected_state` vaut 1, une décision humaine `reviewed_state=1` est présente à la révision 1, et l'état effectif `COALESCE(reviewed_state, detected_state)` vaut 1. La review confirme donc la détection. La ligne de détection et sa valeur originale sont toujours présentes : la détection automatique n'a pas été écrasée.

Cette absence de changement effectif explique le comportement des révisions : le service avance `review_revision` et conserve immédiatement les artefacts à jour lorsque `effective_changed=0`. Il n'était ni nécessaire ni attendu que les PDF soient réécrits après cette confirmation.

## Cohérence des scores

Un recalcul indépendant en lecture seule a comparé, pour chacune des 10 questions, les quatre états effectifs avec les états attendus du snapshot historique. Les 10 scores de question correspondent au recalcul, puis chaque score de copie correspond exactement à la somme de ses questions.

| Copie | Pages | Questions | Score final | Ambiguïtés | Reviews |
|---|---:|---:|---:|---:|---:|
| copy-A | 2/2 | 5 | 5/5 | 0 | 0 |
| copy-B | 2/2 | 5 | 1/5 | 1 | 1 |

Aucune appréciation pédagogique n'est tirée de ces scores.

## Révisions et artefacts

`review_revision=1` et `artifacts_revision=1`. Les artefacts sont donc frais au sens du contrat applicatif. La review ayant confirmé l'état détecté, leur contenu n'avait pas à changer.

| Artefact | État |
|---|---|
| `corrected.pdf` | fichier régulier, 4 758 765 octets, signature `%PDF-`, PDF 1.7 valide, 5 pages |
| `mark-table.pdf` | fichier régulier, 29 422 octets, signature `%PDF-`, PDF 1.7 valide, 1 page |
| `corrected_NOT.pdf` | absent, conformément à l'absence de page rejetée |

Les deux chemins persistés sont les chemins canoniques du workspace `marking-1`; aucun artefact indispensable ne manque et aucun chemin ne sort du workspace.

## Pages alignées et références modernes

Les quatre pages alignées non annotées sont conservées, une pour chaque couple copie/page logique. Elles sont toutes des PNG réguliers de 2480 × 3508. Pour chaque fichier, le SHA-256 recalculé correspond exactement au hash persisté.

Les quatre pages historiques de la génération possèdent chacune :

- une storage key conforme `references/student-exam-…/page-….png` ;
- un fichier PNG régulier présent ;
- des dimensions 2480 × 3508 ;
- un DPI persisté de 300 ;
- un SHA-256 de 64 caractères, vérifié contre les octets du fichier ;
- un rattachement cohérent à la génération, à la copie et à la page logique.

Références modernes disponibles et valides : **4/4**. Le résolveur moderne donne priorité à ces références historiques natives ; aucun fallback legacy n'a été nécessaire ni utilisé.

## Reproductibilité future

Une future `RegenerateMarkingArtifacts` dispose de toutes ses entrées durables :

- snapshots complets des deux copies ;
- 40 détections et leurs états effectifs, dont la review persistée ;
- 10 résultats de question et 2 résultats de copie cohérents ;
- 4 pages alignées avec dimensions et hashes valides ;
- 4 références historiques modernes avec dimensions, DPI et hashes valides ;
- métadonnées de version, seuil et delta ;
- révisions synchronisées.

**Régénération reproductible plus tard : oui.**

## Résidus du workspace

Aucun `.review-artifacts-*`, `.review-backup`, fichier temporaire de publication ou artefact orphelin n'a été trouvé. Une arborescence `.aligned-staging/` et ses deux sous-répertoires de copie subsistent, tous vides.

- unité d'anomalie comptée : 1 arborescence de staging vide ;
- impact : consommation négligeable, aucune donnée ou incohérence métier ;
- priorité : **P3** ;
- zone probablement concernée : cycle de nettoyage autour de `StageMarkingAlignedPage` / `StoreMarkingAlignedPage` dans `internal/handlers/tools/markingAlignedPage.go`.

Aucune correction ni suppression n'a été effectuée pendant cet audit.

## Conclusion opérationnelle

| Question | Réponse |
|---|---|
| pages mélangées correctement reconstruites | oui |
| QR corrects | oui |
| deux copies corrigées | oui |
| ambiguïté détectée correctement | oui |
| review persistée correctement | oui |
| scoring recalculé correctement | oui |
| artefacts frais | oui |
| références modernes utilisées | oui, 4/4 |
| fallback legacy utilisé | non |
| pipeline prêt pour un essai en classe | oui |

Le résidu P3 ne justifie pas de bloquer le test réel 6eA/6eB. Aucun défaut P1 ou P2 n'a été constaté.

## Garantie de lecture seule

- DB historique smoke : inchangée ;
- scan papier : inchangé ;
- PDF original : inchangé ;
- code applicatif : inchangé ;
- correction, review et génération d'artefacts : non rejouées ;
- seul fichier créé dans le dépôt : ce rapport local non committé.
