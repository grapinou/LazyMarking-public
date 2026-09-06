# Nommage des artefacts de correction depuis le PDF source

Date : 2026-09-04

## Conclusion

Le nom original du PDF est désormais conservé pour les nouveaux jobs et sert uniquement à construire le nom proposé au téléchargement. Les chemins internes restent inchangés et indépendants de toute donnée utilisateur : `corrected.pdf` et `mark-table.pdf` dans le workspace du job.

Un upload `6eB.pdf` donne donc `6eB_corrected.pdf` et `6eB_marks.pdf`. Les jobs antérieurs, dépourvus de provenance fiable, restent accessibles avec les noms de repli `corrected.pdf` et `marks.pdf`.

Ni la reconnaissance, ni la politique de revue, ni le scoring n'ont été modifiés.

## Comportement actuel audité

Le flux précédent était le suivant :

1. `ProcessingMarkingHandler` recevait le multipart `pdffile` et validait son contenu PDF ; le `multipart.FileHeader.Filename` n'était pas conservé.
2. Le flux copiait le contenu dans un fichier temporaire `lazymarking-upload-*.pdf`, puis transmettait seulement un lecteur à `ProcessMarking`.
3. `ProcessMarking` publiait le corrigé sous `<workspace>/corrected.pdf`. La compilation Typst du tableau publiait `<workspace>/mark-table.pdf`.
4. `marking_jobs.exam_name` et `marking_jobs.mark_table_name` conservaient les chemins physiques de ces deux artefacts. Ces champs ne contenaient pas le nom source.
5. Les URLs de résultat transmettaient le basename physique au handler de téléchargement.
6. `ServeFullMarkingPdfHandler` contrôlait l'ownership, les revues en attente et la synchronisation des révisions, puis `ServePdfNamed` servait le fichier avec son nom physique dans `Content-Disposition`.
7. Après la dernière revue, la régénération remplaçait atomiquement les mêmes fichiers canoniques et avançait `artifacts_revision`. Aucun nom source n'était alors disponible.

La provenance demandée n'était donc pas durablement disponible. La déduire du fichier temporaire, du workspace ou des artefacts aurait été incorrect.

## Architecture retenue

La séparation est explicite :

- stockage : chemins canoniques existants, inchangés ;
- provenance : nouveau champ nullable `marking_jobs.source_pdf_filename` ;
- présentation : nom calculé au moment de la réponse HTTP ;
- sécurité : seul le basename sans caractères de contrôle est persisté, et ce nom n'est jamais utilisé dans un chemin de stockage.

Le handler d'upload extrait le nom du `FileHeader` après validation multipart, le réduit à un basename et l'enregistre lors de la création du job. Le handler de téléchargement charge cette valeur avec la cible de régénération et choisit le suffixe selon l'artefact physique demandé.

## Persistance et migration

La migration additive `0044_add_marking_source_filename.sql` ajoute la colonne nullable `source_pdf_filename`. Elle ne réalise aucun backfill. Un trigger interdit sa modification après création afin que la provenance et les noms restent stables pendant les revues, les régénérations et les redémarrages.

La nullabilité préserve les jobs historiques. La migration `Down` supprime le trigger puis la colonne.

Les requêtes SQL modifiées sont :

- `CreateHybridMarkingJob`, qui persiste le basename source ;
- `GetMarkingArtifactsRegenerationTarget`, qui restitue cette provenance au résultat et au téléchargement.

`sqlc generate -f db/sqlc.yaml` a été exécuté. Les bindings générés ont été mis à jour.

## Fonction de nommage

`MarkingArtifactFilename(originalFilename, kind)` centralise le contrat :

- retrait de la dernière extension seulement ;
- suffixe `_corrected` ou `_marks` ;
- extension de sortie `.pdf`, qui correspond aux deux formats actuels ;
- conservation des points précédents, espaces, accents et Unicode raisonnable ;
- support d'un nom sans extension ;
- fallback `corrected.pdf` ou `marks.pdf` si le nom est inexploitable.

Exemples vérifiés :

- `6eB.pdf` → `6eB_corrected.pdf` / `6eB_marks.pdf` ;
- `classe.6eB.pdf` → `classe.6eB_corrected.pdf` ;
- `TEST.PDF` → `TEST_corrected.pdf` ;
- `fichier sans extension` → `fichier sans extension_marks.pdf`.

## Sécurité et Content-Disposition

Les séparateurs Unix et Windows sont normalisés avant l'appel à `path.Base`. Les retours chariot, sauts de ligne et octets NUL sont supprimés. Une entrée `../../6eB.pdf` produit donc seulement `6eB_corrected.pdf`.

Le chemin réellement ouvert continue d'être validé par les contrôles existants : composants sûrs, extension PDF, workspace attendu, refus des liens symboliques et vérification que le fichier ouvert est bien celui contrôlé.

Une nouvelle enveloppe `ServePdfDownloadNamed` reçoit séparément :

- le nom physique validé ;
- le nom de téléchargement.

`mime.FormatMediaType` construit `Content-Disposition: attachment` et encode correctement les espaces et caractères Unicode. Le nom utilisateur ne participe jamais à `filepath.Join`.

Les autres usages de `ServePdfNamed`, notamment les previews, gardent leur comportement `inline` historique.

## Régénération et cycle de revue

La régénération continue à produire et publier `corrected.pdf` et `mark-table.pdf`. Elle ne touche pas à `source_pdf_filename`, rendu immuable par la base. Le nom visible est donc identique avant et après régénération, ainsi qu'après redémarrage.

Les protections antérieures sont conservées : un artefact n'est servi que pour le propriétaire du job, sans revue hybride en attente et lorsque `artifacts_revision == review_revision`. Aucun nouveau chemin ne contourne ces contrôles.

## Compatibilité historique

Une ligne sans `source_pdf_filename` reste lisible et téléchargeable :

- corrigé : `corrected.pdf` ;
- notes : `marks.pdf`.

Aucun backfill n'est tenté depuis `exam_name`, `mark_table_name`, un UUID ou un chemin runtime. Ces valeurs ne prouvent pas le nom soumis par l'utilisateur.

## Tests ajoutés et adaptés

Les tests couvrent :

- les six noms fonctionnels demandés ;
- le retrait de la dernière extension ;
- les traversées Unix et Windows ;
- les caractères de contrôle et les caractères spéciaux ;
- la capture persistante du nom lors de l'upload ;
- un `Content-Disposition` Unicode correctement parsable ;
- l'identité des octets servis ;
- l'absence de fichier physique portant le nom de téléchargement ;
- le corrigé et le tableau d'un job final ;
- le fallback d'un ancien job ;
- les protections existantes contre une revue en attente et une révision d'artefacts obsolète, toujours couvertes par la suite existante.

Les schémas SQLite minimaux de quelques tests ont reçu la nouvelle colonne nullable.

## Contrôle privé 6eB

Le job réel 6eB `marking_job_id=10` a été trouvé dans la base privée de validation avec les chemins physiques attendus : `corrected.pdf` et `mark-table.pdf`. Créé avant la migration, il ne contient pas le nom source `6eB.pdf`. Conformément au contrat de compatibilité, il présenterait donc les fallbacks `corrected.pdf` et `marks.pdf`.

La base privée n'a pas été modifiée pour fabriquer cette provenance. Le comportement `6eB_corrected.pdf` / `6eB_marks.pdf` est vérifié par le test HTTP avec une ligne portant réellement `source_pdf_filename='6eB.scans.pdf'`. Il sera effectif sur tout nouveau job uploadé sous le nom `6eB.pdf`.

Une copie privée a été créée sous `runtime/diagnostics/marking-artifact-filenames/migration-check.db`. La séquence migration 44 `up`, `down-to 43`, puis `up` a réussi. L'original n'a pas été modifié.

## Fichiers modifiés

- `db/migrations/0044_add_marking_source_filename.sql` ;
- `db/query/markingJobs.sql` ;
- `db/query/markingReviews.sql` ;
- bindings `sqlc` sous `internal/db/` ;
- `internal/handlers/marking/handlers.go` ;
- `internal/handlers/tools/markingArtifactFilename.go` ;
- `internal/handlers/tools/servePdfNamed.go` ;
- tests ciblés associés et schémas de fixtures minimaux.

## Validations

- `sqlc generate -f db/sqlc.yaml` : succès ;
- `go test ./...` : succès ;
- `./scripts/check.sh` : succès, y compris replay intégral des migrations jusqu'à 44, `go vet`, tests et build ;
- `git diff --check` : succès ;
- migration sur copie privée, `up → down → up` : succès.

## Points d'attention et prochain jalon

Les anciens jobs ne peuvent pas recevoir rétroactivement un nom métier fiable. Ce choix est volontaire et évite une provenance inventée. Le format des deux artefacts est actuellement PDF ; si le tableau change un jour de format, le kind `marks` devra conserver l'extension réelle comme le prévoit le contrat UX.

Le prochain jalon recommandé est un test opérationnel avec un nouveau job de correction nommé explicitement `6eB.pdf`, afin de vérifier les deux noms dans un navigateur réel. La prévention du réupload d'un PDF corrigé reste volontairement hors périmètre.
