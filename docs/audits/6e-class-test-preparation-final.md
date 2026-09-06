# Préparation finale du test réel 6e

## Reprise et installation

La préparation a repris le 2 septembre 2026 sur la DB existante `/home/sighto/Documents/lazymarking-6e-test/app.db`, sans recréation ni suppression. Goose est en version 41. Le compte `<IDENTIFIANT_TEST_PRIVE>`, les huit référentiels, les familles 1 et 2 complètes, la principale et l’alternative de famille 3, les réponses existantes et l’image balance étaient présents. Toute écriture métier de ce jalon a utilisé les routes HTTP, formulaires, cookie de session, CSRF et handlers applicatifs. Les SELECT SQLite ont servi uniquement aux vérifications.

## Banque et images

La famille 3 a été complétée avec ses quatre réponses alternatives et l’image éprouvette. Les familles 4 à 10 ont ensuite été créées conformément au cahier des charges.

État final vérifié : 10 principales, 10 alternatives, 40 réponses principales et 40 réponses alternatives ; chaque question possède quatre réponses et exactement une correcte. Il existe cinq images uploadées et stockées : balance, éprouvette, thermomètre 18 °C, thermomètre 25 °C et circuit fermé. Les largeurs ont été ajustées via les formulaires d’édition à 40 % pour balance/éprouvette/circuit et 42 % pour les thermomètres. La banque et les previews les affichent correctement.

## QCM et previews

Le QCM `Culture scientifique — 6e` contient exactement les dix familles, aux positions 1 à 10 ; la page de gestion fonctionne sans erreur DB. Chaque copie contient donc dix questions.

Preview portrait : succès, A4, 3 pages. Preview paysage : succès, A4 paysage, 2 pages. Inspection visuelle : texte, réponses et images lisibles, aucun chevauchement ou contenu coupé, aucune page blanche. La troisième page portrait peut être peu remplie selon le tirage aléatoire, mais contient bien la fin du QCM et reste dans l’objectif de 2–3 pages.

## Classes, élèves et évaluations

Classes créées : `6eA` et `6eB`. Deux imports CSV via le formulaire réel ont créé exactement `Élève A01` à `Élève A30` et `Élève B01` à `Élève B30`. Contrôle final : 30 élèves dans chaque classe, 60 élèves et 60 associations au total, aucun doublon ni rattachement croisé.

Le modèle liant une évaluation à une classe, deux évaluations ont été créées avec le même QCM : `Culture scientifique — 6eA` et `Culture scientifique — 6eB`.

## Générations

Les deux générations ont été lancées par leurs formulaires réels et attendues jusqu’au statut terminal :

- 6eA : `success`, 30/30, 30 `student_exam`, 90 snapshots/pages ;
- 6eB : `success`, 30/30, 30 `student_exam`, 90 snapshots/pages.

Aucun worker failure, génération fantôme ou `student_exam` incomplet n’a été observé. Chaque copie compte exactement 3 pages, soit 90 pages par PDF et 180 pages au total.

## Références modernes et individualisation

Les 180/180 pages possèdent un contrat moderne complet : clé de stockage relative, dimensions 2480×3508, DPI 300 et SHA-256 de 64 caractères. Les 180 fichiers existent dans leurs workspaces canoniques et le hash calculé concorde avec la DB. `PRAGMA foreign_key_check` est vide et `PRAGMA integrity_check` retourne `ok`. Aucun fallback legacy n’a été utilisé.

Les snapshots JSON ont été comparés anonymement : chaque classe contient 30 signatures complètes distinctes et 30 ordres de questions distincts. L’échantillon montre des choix principale/alternative et des ordres de réponses différents ; aucune diversité n’a été forcée.

## PDFs et QR

- `/home/sighto/Documents/lazymarking-6e-test/Culture-scientifique-6eA.pdf` : 90 pages A4, 30 copies de 3 pages ;
- `/home/sighto/Documents/lazymarking-6e-test/Culture-scientifique-6eB.pdf` : 90 pages A4, 30 copies de 3 pages.

Les fichiers ont été téléchargés par la route authentifiée, sans modification du contenu canonique. Inspection visuelle de premières et dernières copies des deux classes : seules les identités synthétiques attendues apparaissent, QR nets, questions et réponses lisibles, images noir et blanc lisibles, pagination 1–3 cohérente, aucun contenu coupé ni page blanche. Les 60 identités attendues sont confirmées par les relations DB et snapshots ; les PDFs finaux sont rasterisés, donc `pdftotext` ne permet pas un second inventaire textuel autonome.

Le lecteur QR Go réel (gozxing directement, sans OpenCV et sans pipeline Marking) a décodé 12/12 pages rendues depuis les deux PDFs : deux copies échantillonnées par classe, pages logiques 1, 2 et 3, avec quatre `student_exam_id` cohérents.

Marking, scan, OpenCV, scoring et review n’ont pas été lancés.

## Anomalies et UX

Aucun nouveau bug bloquant ou important. Défauts sans correction : identité factice historique `John Doe… / 666` dans la preview portrait ; certaines pages 3 sont volontairement peu remplies selon l’individualisation ; `pdfinfo` émet un avertissement bénin `Suspects object is wrong type (boolean)` sur les PDFs, qui restent lisibles et correctement rendus.

Aucun code ni template n’a été modifié pendant cette reprise. Seul ce rapport non suivi a été ajouté au dépôt ; les scripts de préparation, CSV, images, DB et PDFs restent dans l’installation privée.

## Reprise humaine

```sh
cd /home/sighto/Documents/lazymarking-6e-test/runtime && \
SESSION_SECURE=false \
SESSION_KEY='<SESSION_KEY_PRIVEE>' \
CSRF_AUTH_KEY='<CSRF_KEY_PRIVEE>' \
/home/sighto/Documents/lazymarking-6e-test/lazymarking-server
```

URL : `http://127.0.0.1:8080/login`. Identifiant : `<IDENTIFIANT_TEST_PRIVE>`. Mot de passe : `<MOT_DE_PASSE_TEST>`.

## État Git

Les modifications et fichiers non suivis préexistants sont préservés. Ce jalon ajoute uniquement `docs/audits/6e-class-test-preparation-final.md` dans le dépôt. Aucune DB, image, PDF, identité CSV, script privé ou clé de l’installation 6e n’apparaît dans le worktree.
