# Smoke test fonctionnel — installation moderne fraîche

## Installation persistante et migrations

Test effectué le 2 septembre 2026 exclusivement via les routes HTTP publiques pour toutes les données métier. DB persistante : `/home/sighto/Documents/lazymarking-smoke/app.db`. Elle a été créée vide puis migrée par Goose de 0001 à 0040 ; `goose status` confirme toutes les migrations appliquées. `db/data/app.db` n'a pas été touchée.

Le chemin DB étant relatif (`./db/data/app.db`), aucun changement de configuration n'a été nécessaire : le serveur est lancé depuis `/home/sighto/Documents/lazymarking-smoke/runtime`, où `db/data/app.db` est un lien vers la DB smoke, `internal` référence les templates/configurations du dépôt et `assets/` demeure un stockage privé propre au smoke test. Le binaire persistant est `/home/sighto/Documents/lazymarking-smoke/lazymarking-server`.

Configuration de développement : `SESSION_SECURE=false`, clés synthétiques distinctes de 32 octets, aucun secret committé.

## Parcours de session et CSRF

Compte créé par GET/POST `/register`, puis login, accès dashboard avec session, logout POST, constat de redirection vers login, puis nouveau login réussi. Chaque POST a utilisé le token obtenu depuis le formulaire GET correspondant et le même cookie jar. Les formulaires urlencoded et multipart ont fonctionné sous la protection CSRF.

- utilisateur : `<IDENTIFIANT_TEST_PRIVE>` ;
- email : `<EMAIL_TEST_PRIVE>` ;
- mot de passe : `<MOT_DE_PASSE_TEST>`.

## Référentiels et CRUD

Référentiels finaux créés via formulaires : matière `Géographie`, thème `Capitales et drapeaux du monde`, niveau `Smoke Test`, compétence `Identifier`, difficulté `Standard`, points `1`, année `2026-2027`, période `Smoke`.

Suppressions réellement exercées via l'application :

| Type | Création | Modification | Suppression | Résultat |
|---|---:|---:|---:|---|
| matière temporaire | oui | oui | oui | disparue |
| thème temporaire | oui | oui | oui | disparu |
| difficulté temporaire | oui | oui | oui | disparue |
| réponse temporaire | oui | oui | oui | disparue |
| image alternative temporaire | oui | oui (largeur) | oui | DB et fichier supprimés |
| question alternative temporaire | oui | non | oui | disparue |
| image principale temporaire | oui | oui (largeur) | oui | DB et fichier supprimés |
| question principale temporaire | oui | oui | oui | disparue |
| QCM temporaire composé | oui | non | oui | QCM et composition disparus, aucun orphan |

Refus attendu testé : suppression de la matière finale déjà référencée par les questions. L'application a rencontré la contrainte référentielle, redirigé et conservé la matière. Aucun correctif SQL n'a été utilisé.

## Banque finale

État final contrôlé dans l'interface puis confirmé par SELECT read-only :

- 5 questions principales, couvrant Europe, Afrique, Asie, Amériques et Océanie ;
- 20 réponses principales, exactement une correcte par question ;
- 5 alternatives, une par famille ;
- 20 réponses alternatives, exactement une correcte par alternative ;
- 5 images alternatives, une par famille, et aucun enregistrement image temporaire restant ;
- images privées sources dans `/home/sighto/Documents/lazymarking-smoke/assets/`, fichiers réellement uploadés dans le runtime privé `assets/images/`.

Le texte de la question Italie a été créé sous une formulation provisoire puis corrigé via le formulaire d'édition. Une réponse temporaire et plusieurs titres de référentiels temporaires ont aussi été modifiés puis vérifiés avant suppression.

Les bitmaps de drapeaux synthétiques ont été produits avec imagegen pour ce smoke test et restent entièrement hors dépôt.

## QCM, composition et previews

`QCM temporaire à supprimer` a été créé, composé avec une famille puis supprimé via l'UI ; la suppression cascade de `qcm_questions` est propre.

QCM final : `Capitales et drapeaux du monde`, 5 familles et donc 5 questions par copie, jamais 10. La page « Gérer les questions » affiche les cinq familles et positions 1–5 sans erreur `no such column qcm_questions.position`. Un mouvement vers le haut puis vers le bas a été soumis via les formulaires réels et la composition demeure cohérente.

Preview portrait : succès, PDF A4 de 2 pages. Preview paysage : succès, PDF A4 paysage d'une page. Les deux sont lisibles et montrent questions, réponses et images. Défaut cosmétique relevé : la preview portrait conserve l'identité factice historique « John Doe… » et la classe `666` ; cela n'affecte ni l'examen final ni les données.

## Classe, élèves, examen et génération

Créés par formulaires : classe `Smoke Test`, élèves `Testeur 01` et `Testeur 02`, année/période, puis évaluation `Évaluation smoke — Capitales` liée au QCM final.

La première génération a révélé un bug bloquant reproductible : `TypstWriter` crée un basename aléatoire sans underscore final, `ExportTypstToPNGs` concatène `page-{0p}-of-{t}`, mais `ExtractPageNumber` exigeait `_page-…`. Symptôme : les deux workers échouaient avec `ExtractPageNumber return not ok`, puis le mécanisme de cleanup supprimait correctement la génération échouée.

Correctif minimal : le parseur accepte les deux noms produits, avec ou sans underscore avant `page`. Fichiers modifiés : `internal/handlers/tools/extractPageNumber.go`; test ajouté : `internal/handlers/tools/extractPageNumber_test.go`. Après reconstruction du binaire, la génération relancée via POST UI a réussi : statut `success`, `processed_students=2`, `total_students=2`.

Résultat final : 2 `student_exam`, 4 pages/snapshots et 4 références PNG modernes pré-QR à 300 DPI. PDF téléchargé via la route authentifiée et conservé sous `/home/sighto/Documents/lazymarking-smoke/Évaluation-smoke-Capitales.pdf` : 4 pages A4, 1 092 792 octets. Inspection visuelle : deux identités synthétiques correctes, 5 familles par copie avec sélection principale/alternative, réponses mélangées, images visibles, pagination 1–2, aucune page vide/cassée. Décodage privé avec le lecteur réel : 4/4 QR valides, deux copies, pages logiques `{1,2}` pour chacune.

Marking, scan, scoring et review n'ont pas été lancés.

## Anomalies

- Bloquant rencontré et corrigé : 1, incompatibilité du nom de PNG Typst avec `ExtractPageNumber`.
- Important : 0.
- Cosmétique : 1, identité/classe factices peu présentables dans la preview portrait.

## Validation technique

- `./scripts/check.sh` : succès, replay Goose 0001→0040, gofmt, vet, tests, build et diff-check inclus ;
- `go test -race ./...` : succès ;
- test QR privé par overlay : succès, sans fichier de test privé conservé dans le dépôt.

## Reprise humaine

Commande exacte :

```sh
cd /home/sighto/Documents/lazymarking-smoke/runtime && \
SESSION_SECURE=false \
SESSION_KEY='<SESSION_KEY_PRIVEE>' \
CSRF_AUTH_KEY='<CSRF_KEY_PRIVEE>' \
/home/sighto/Documents/lazymarking-smoke/lazymarking-server
```

Puis ouvrir `http://127.0.0.1:8080/login`, se connecter avec `<IDENTIFIANT_TEST_PRIVE>` / `<MOT_DE_PASSE_TEST>`, ouvrir « Évaluations » et le PDF de `Évaluation smoke — Capitales`. Le téléchargement persistant est également disponible au chemin indiqué ci-dessus.

## État du dépôt

Aucune DB, image ou PDF smoke n'est dans le dépôt. Le README modifié et tous les fichiers utilisateur préexistants ont été préservés. Seuls le correctif/test ciblé et ce rapport local non committé ont été ajoutés au worktree pendant ce jalon.
