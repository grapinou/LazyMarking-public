# Détection des cercles de réponses à 300 ppp — 5 octobre 2026

## 1. Symptôme

Une génération de classe échoue avec `CircleDetectionAnswerreturn not ok or no answers detected between questions`, malgré un aperçu QCM correct. L'erreur est déclenchée dans `BuildQcmStudentCtx` lorsqu'une bande située entre deux repères de questions ne contient aucune réponse détectée.

État initial : dépôt propre, HEAD `7a0f83c` (`Document pre-test finalization`). Aucune modification préexistante n'a été supprimée.

## 2. Cause et limites du diagnostic

La plage Hough de réponses **18–23 px est trop étroite pour certains rendus des glyphes de réponse**. Les réponses utilisent `U+25CB` à `2.5em`, avec une taille de texte de 11 pt ; leur géométrie dépend de la police. Les repères de questions sont des cercles Typst géométriques de rayon 8 pt, soit environ 33,3 px à 300 ppp.

Le changement de layout ne suffit pas, à lui seul, à reproduire l'incident ici : les tests existants et le nouveau cas avec Liberation Sans passent aussi avec 18–23 px. En revanche, le même layout et les mêmes tailles de texte reproduisent l'échec lorsque la famille demandée est indisponible et que Typst utilise sa police de repli embarquée. Le rendu avec Libertinus Serif produit également cet échec.

La reproduction contrôlée confirme le défaut de tolérance et une explication possible de la différence entre environnements. **L'absence ou la substitution de police sur le serveur réel n'est pas établie** : ni le PNG précis de la génération en échec ni ses versions/polices effectives n'ont été fournis. Le rapport ne prétend donc pas prouver que l'évolution du layout a agrandi les cercles sur tous les environnements.

Les contrôles sur la reproduction excluent les autres causes proposées :

- Typst compile réellement et produit trois pages A4 de **2480 × 3508 px**, avec l'exporteur de production à **300 ppp**.
- Les 12 repères de questions sont détectés, avec des rayons entiers de 32–33 px.
- Les limites verticales sont celles de la génération : bas du repère courant, haut du suivant, ou 3390 pour le dernier bloc.
- Chaque bande contient quatre réponses avec le correctif, entièrement à l'intérieur de ses limites ; aucune réponse n'apparaît au-dessus de la première question des pages suivantes.
- Inclure volontairement le repère noir dans la ROI de réponses ne change pas les quatre cercles retournés.
- Seule la borne maximale Hough change entre l'échec et la réussite : ni pagination, ni résolution, ni ROI, ni contenu des réponses ne changent.

## 3. Pourquoi 23 px ne suffit plus

Avec la police de repli, la première bande (`page=1 question=1 top=677 bottom=1216`) retourne `answers=[] ok=true` avec 18–23 px, puis quatre cercles de rayon entier 23 px avec 18–28 px. Il ne faut pas interpréter ce rayon entier comme une mesure exacte du contour : `CircleDetectionAnswer` convertit en `int` le rayon flottant de Hough, et la plage de recherche influence son ajustement.

Sur les 12 blocs du rendu de repli, les rayons Hough retournés après conversion vont de **23 à 27 px**. Avec Liberation Sans, ils vont de **19 à 25 px** avec la nouvelle plage, contre **19–20 px** avec l'ancienne. Les centres et ajustements peuvent varier de quelques pixels selon la plage Hough et la position du glyphe sur la grille de rasterisation.

## 4. Paramètres OpenCV avant/après

| Paramètre | Avant | Après |
| --- | --- | --- |
| Méthode | HoughGradient | inchangé |
| dp | 1 | 1 |
| minDist | 20 | 20 |
| param1 | 100 | 100 |
| param2 | 30 | 30 |
| minRadius réponses | 18 | 18 |
| maxRadius réponses | 23 | **28** |
| Rayon des repères de questions | 30–35 | inchangé |
| Filtre du centre des réponses | moyenne grise > 200 | inchangé |
| Flou médian | 5 | inchangé |
| Export PNG | 300 ppp | inchangé |

## 5. Justification du choix

La borne 28 couvre les ajustements observés jusqu'à 27 px, avec une petite marge, tout en restant inférieure au minimum 30 du détecteur des repères noirs. Le minimum 18 conserve les petits glyphes historiquement acceptés. Le filtre de centre blanc continue d'exclure les repères noirs. Aucun autre seuil n'est assoupli et aucune refonte Hough n'est introduite.

## 6. Fichiers modifiés et diagnostic

- `internal/handlers/tools/circleDetectionAnswer.go` : borne maximale 28 et commentaire expliquant les deux tailles de cercles.
- `internal/handlers/tools/buildQcmStudentCtx.go` : erreurs contextualisées sur chaque zone analysée. Les numéros de questions correspondent à l'ordre global de la copie, à partir de 1 ; les pages commencent à 1. Les limites sont exprimées en pixels du PNG natif.
- `internal/handlers/tools/generatedLayout_test.go` : test réel du layout courant, avec police configurée et repli de police.
- Le présent rapport.

Exemple du format de l'erreur entre questions :

```text
 -> CircleDetectionAnswer return not ok or no answers detected between questions: page=1 questions=1->2 top=677 bottom=1216 answers=0 ok=true
```

Les erreurs des autres zones indiquent `questions=none`, `questions=qr->N` ou `questions=N->bottom`. Aucun nom d'élève, identifiant personnel ou chemin contenant un nom d'utilisateur n'est ajouté. Les conditions qui déclenchent les erreurs restent identiques.

## 7. Test de non-régression

`TestGeneratedLayoutAnswerCircles` réutilise `layoutWorkspace`, `layoutExample`, `TypstWriter`, `ExportTypstToPNGs`, `CircleDetection` et `CircleDetectionAnswer`. Il rend 12 questions de quatre réponses, réparties ici sur trois pages, dans deux cas :

1. `configured_font` : source générée actuelle inchangée.
2. `missing_font_fallback` : uniquement dans la source temporaire du test, remplacement de la famille demandée par un nom volontairement inexistant. Typst exerce ainsi son vrai repli de police ; les tailles, le reste du layout et l'exporteur restent identiques. Une assertion empêche une disparition silencieuse de cette substitution si le template évolue.

Le test contrôle les dimensions à 300 ppp, le total de questions/réponses, quatre cercles dans chaque bande, l'absence de réponses orphelines en haut de page, les rayons distincts et l'égalité des réponses lorsque la ROI inclut aussi le gros repère noir. Les coordonnées détectées sont journalisées avec `-v`.

La vérification avant/après a été effectuée avec le test final : remettre temporairement `maxRadius=23` fait échouer `missing_font_fallback` dès la première bande (`answers=[]`), puis restaurer 28 le fait réussir sur les 12 blocs. Le cas configuré passe avec les deux seuils. Les artefacts temporaires du test sont nettoyés ; aucun PDF/PNG n'est ajouté à Git.

## 8. Vérifications

Environnement : Go 1.26.5, GoCV 0.42.0, OpenCV 4.12.0, Typst 0.15.1 ; Poppler et Goose disponibles. Les tests de pipeline migrent exclusivement une base SQLite temporaire créée par le test. Aucune migration destructive ni modification de la base réelle `2026-2027` n'a été effectuée.

| Vérification | Résultat |
| --- | --- |
| Nouveau test avec 18–23 px, police de repli | FAIL attendu : aucune réponse dans le premier bloc |
| Nouveau test avec 18–28 px, deux polices | PASS : 12 questions / 48 réponses / 3 pages par cas |
| Tests ciblés layout, pipeline, ancien snapshot, homographie historique | PASS |
| `go test ./internal/handlers/tools/...` (sortie JSON) | PASS : 403 résultats de tests/sous-tests PASS, 12 SKIP attendus, 0 FAIL |
| `go test ./internal/handlers/generateExams/...` | PASS |
| `go test ./...` (sortie JSON) | PASS : 42 paquets, 1374 résultats de tests/sous-tests PASS, les mêmes 12 SKIP attendus, 0 FAIL ; 15 paquets sans tests |
| `git diff --check` | PASS |

Les 12 SKIP de `tools` concernent uniquement le corpus privé non configuré : cinq tests de reconnaissance des réponses, deux calibrations d'ambiguïté, le rapprochement QR/base, le pipeline QR historique et trois tests de correction historique. Les variables `LAZYMARKING_TEST_*` et `LAZYMARKING_CALIBRATION_DIR` nécessaires sont absentes. Les nouveaux tests de layout et le pipeline réel sur base temporaire ont effectivement tourné, sans SKIP.

## 9. Risques résiduels et compatibilité

Une plage plus large peut modifier les ajustements Hough des **nouvelles** copies ; les mesures ci-dessus le montrent. Les snapshots et coordonnées déjà persistés, les références PNG historiques et la logique de correction ne sont pas réécrits. Le pipeline existant valide la reproduction identique des PNG de référence et la correction après modification de la banque ; le test d'homographie conserve les détections et scores historiques.

Les glyphes d'autres polices ou des illustrations contenant des anneaux peuvent encore poser problème. Le test contrôle le nombre exact de réponses et l'exclusion des repères noirs sur ses documents, sans prétendre couvrir toutes les illustrations. Les questions très longues avec réponses réparties entre pages restent hors de ce correctif ; leur prise en charge historique est conservée.

## 10. Recommandations ultérieures

Comparer les polices et versions effectives du serveur au rendu local, et garder une fixture anonymisée d'un PNG réellement en échec si elle devient disponible. Pour stabiliser les nouvelles générations, distribuer explicitement la police attendue et vérifier sa disponibilité. À plus long terme seulement, envisager des cercles géométriques de rayon explicite pour les réponses, avec une version de layout distincte et des tests de compatibilité ; cela n'est pas inclus dans ce correctif.

Aucun commit ni push effectué.
