# Audit de mapping du corpus réel historique

## Périmètre et confidentialité

Audit effectué en lecture seule le 2 septembre 2026. Aucun nom, prénom, QR brut, image, scan ou crop n'est reproduit ici. Les comparaisons ont utilisé les identifiants techniques uniquement en mémoire et dans un résultat temporaire privé supprimé en fin d'audit. Aucun PDF ni DB n'a été copié dans le dépôt.

Sources privées utilisées :

- répertoire PDF privé : `/home/sighto/Documents/lz_test_anciennes_copies/` ;
- DB historique source : `/home/sighto/Documents/app.db` ;
- les copies `/home/sighto/Documents/db_test_real.db` et `/home/sighto/Documents/lz_pdf_test/test/app.db` ont la même empreinte que la source, mais n'ont pas été nécessaires.

La DB a été ouverte exclusivement avec SQLite `mode=ro`/`-readonly`. Aucune copie expérimentale n'était nécessaire. Empreinte SHA-256 DB avant et après : `ad9e294836fd9044c72d37ddfebf5e1a3588899e1e30aa4b13255577861e1259`, inchangée.

Empreintes contrôlées avant/après, inchangées :

- original 5e1 : `7378e2caa7000ffdbb2e792ef36b783cd26e6830d966bc5342bda4bdc43b185c` ;
- scans 5e1 : `0f77bdf3b0b1e37bc8826208573c31d134ccdc6a2482517a8f11a8abfb4487d3` ;
- original 6e3 : `f9464cac80769a6b100292db5bdefc50962f7fa08091bef623f0b2be1b8169ab` ;
- scans 6e3 : `58671aa59dbb9f6f951c3f788fa9dab98a3350b5c765363dfd1867dbc28999b6`.

## Schéma QR historique vérifié

Le producteur `QrCodeMaker` sérialise en JSON `config.QrCodeInfo`, puis produit un QR avec correction d'erreur maximale. Le contenu historique vérifié est un objet JSON avec :

- `student_exam_id` : entier 64 bits positif ;
- `page_exam` : entier positif donnant l'ordre logique dans la copie ;
- `page_name` : champ de structure présent dans le type, vide lors de la production historique et renseigné localement après décodage dans le pipeline.

Le lecteur réel essayé pendant l'audit est `QrReader` : d'abord gozxing, puis GoCV avec ROI haut-gauche, niveaux de gris, seuil adaptatif, cadrages élargis et agrandissements. Après décodage, le JSON est validé et les deux identifiants doivent être positifs. Le QR n'encode pas directement l'identifiant de génération : celle-ci est prouvée par la relation DB `student_exam.exam_generated_id`.

## Correspondance DB

Tous les `student_exam_id` des deux originaux existent dans la DB. Pour chaque classe, ils convergent vers exactement une génération, un examen/QCM et une classe DB. `student_exam_content.page_tot` correspond au nombre de pages observé dans l'original pour toutes les copies. Les snapshots JSON de `student_exam_page_content` existent pour toutes les pages attendues.

La structure historique utile est : `student_exam` → `exams_generated` → `exams` → `qcm`/`class_codes`, complétée par `student_exam_content` et `student_exam_page_content`. Aucun scan ne pointe vers une autre génération et aucun identifiant hors DB n'a été trouvé.

## Résultats 5e1

- pages original : **81** ; pages scan : **78** ;
- copies originales : **27** ;
- copies scannées complètes : **26** ;
- copies scannées partielles : **0** ;
- copies absentes / non scannées potentielles : **1** ;
- QR originaux décodés : **81/81** ; QR scans décodés : **78/78** ;
- correspondances exactes original ↔ scan : **78** ;
- pages unresolved : **0** ;
- pages inconnues ou `page_exam` impossible : **0** ;
- cross-generation : **0** ;
- doublons problématiques : **0** ;
- une copie complète est physiquement mélangée dans le paquet, mais son ordre logique 1–3 est reconstruit sans ambiguïté par le QR.

L'unique copie totalement absente est classée « absente / non scannée potentielle », pas comme une erreur.

## Résultats 6e3

- pages original : **53** ; pages scan : **43** ;
- copies originales : **26** ;
- copies scannées complètes : **21** ;
- copies scannées partielles : **0** ;
- copies absentes / non scannées potentielles : **5** ;
- QR originaux décodés : **53/53** ; QR scans décodés : **43/43** ;
- correspondances exactes original ↔ scan : **43** ;
- pages unresolved : **0** ;
- pages inconnues ou `page_exam` impossible : **0** ;
- cross-generation : **0** ;
- doublons problématiques : **0**.

Les cinq copies totalement absentes sont classées « absentes / non scannées potentielles ». Le nombre de pages attendu varie entre copies dans cet original ; l'exhaustivité a donc été évaluée copie par copie avec les couples `(student_exam_id, page_exam)`, jamais par une hypothèse globale de pages par copie.

## Vérification géométrique légère

Six couples anonymisés original/scan ont été rasterisés avec les primitives actuelles, trois par classe. `Homography` (SIFT, ratio test, RANSAC, warp) a réussi **6/6**, sans erreur. Les artefacts étaient dans les répertoires temporaires du test et ont été supprimés automatiquement. Ce résultat établit une faisabilité géométrique légère, pas la qualité du scoring : MeanGray, détection des réponses, scoring et review n'ont pas été exécutés.

## Original legacy et référence moderne

Les contrats ne sont pas équivalents :

- référence moderne : PNG natif exact, rasterisé à 300 DPI avant ajout du QR, conservé avec dimensions, empreinte et rattachement génération/copie/page ;
- original legacy récupéré : page du PDF final historique, contenant le QR et ayant subi la chaîne de composition PDF finale.

Aucune colonne moderne n'a été backfillée. Le PDF historique améliore fortement la preuve d'identité et fournit une base géométrique réelle, mais ne doit pas être présenté comme le PNG pré-QR moderne.

## Faisabilité d'une récupération de référence legacy

Verdict technique : **probablement possible**. La rasterisation contrôlée du PDF original puis l'alignement avec les scans réussissent sur 6/6 échantillons. Une future stratégie devrait fixer DPI et moteur de rasterisation, neutraliser ou masquer la zone QR avant la comparaison si elle perturbe les features, vérifier les dimensions et conserver provenance/empreinte. Elle doit rester explicitement marquée `legacy recovered`, distincte d'une référence moderne native.

Fichiers/fonctions probablement concernés par une future étude, sans défaut corrigé ici : `internal/handlers/tools/homography.go`, `QrReader`, la résolution des références de page et la validation de génération. Priorité : P2 avant exécution réelle du pipeline moderne sur cette DB.

## Anomalies et verdict

Aucune incohérence DB, QR, génération, page inconnue, copie partielle ou doublon problématique n'a été détectée. Les six copies absentes sont compatibles avec des absences/non-remises réelles. L'ordre physique mélangé d'une copie 5e1 est correctement résolu par l'autorité QR + `page_exam`.

Verdict corpus : **B — corpus exploitable avec quelques cas legacy connus**. Le mapping DB ↔ original ↔ scan est fiable, mais l'absence du contrat moderne de référence PNG pré-QR empêche de considérer le corpus directement prêt pour une correction moderne sans étape privée contrôlée de migration/copie et de récupération de référence.

- Peut-on lancer une correction réelle directement avec la DB historique source : **non** ; jamais sur la source, et le pipeline moderne exige une préparation sur copie privée ainsi qu'une stratégie de référence legacy validée.
- Peut-on comparer le nouveau résultat à l'ancienne réalité : **oui**, au niveau des pages/copies et des résultats agrégés, avec anonymisation.
- Les PDF originaux retrouvés améliorent-ils nettement la sécurité du test : **oui**.
- Un mécanisme futur de legacy reference recovery mérite-t-il d'être étudié : **oui**.

Prochaine étape : préparer hors dépôt une copie immuable de la DB, définir et tester le contrat `legacy recovered reference` sur un sous-ensemble anonymisé, puis seulement exécuter un end-to-end sans écriture dans les sources historiques.

## Garantie de lecture seule

La DB historique source et les quatre PDF ont des empreintes avant/après identiques. Aucun code, migration ou donnée privée du dépôt n'a été modifié ou créé ; seul le présent rapport local non committé a été ajouté.
