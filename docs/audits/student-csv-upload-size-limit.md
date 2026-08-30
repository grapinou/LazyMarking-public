# Limite de taille de l’import CSV

## 1. Défaut initial

`AddCSVStudentHandler` lisait `class_code_id` avec `r.FormValue` avant d’appeler `tools.CheckCSVFile`.

Pour une requête `multipart/form-data`, `FormValue` peut déclencher implicitement `ParseMultipartForm`. Le helper installait seulement ensuite `http.MaxBytesReader` et rappelait `ParseMultipartForm`. Comme le formulaire avait déjà été parsé, ce second appel ne relisait pas le corps à travers la limite : une requête supérieure aux 2 Mio annoncés pouvait donc avoir été intégralement consommée auparavant.

Le défaut exposait le serveur à une consommation excessive de mémoire ou d’espace temporaire par un utilisateur authentifié.

## 2. Comportement de `FormValue` et du parsing multipart

La bibliothèque standard Go autorise `FormValue` à appeler le parsing du formulaire lorsque celui-ci n’a pas encore été effectué, et cette API ignore l’erreur de parsing. `ParseMultipartForm` ne recommence pas réellement le traitement lorsque `r.MultipartForm` est déjà initialisé.

L’ordre précédent était donc :

1. lecture de `class_code_id` avec `FormValue` ;
2. parsing multipart implicite possible ;
3. installation tardive de `MaxBytesReader` ;
4. second appel sans effet protecteur sur les octets déjà lus.

## 3. Pourquoi la limite arrivait trop tard

La protection portait uniquement sur les lectures effectuées après le remplacement de `r.Body`. Elle ne pouvait ni annuler ni limiter le parsing déclenché auparavant par `FormValue`.

Le problème n’était donc pas la valeur de la limite, mais l’ordre des opérations.

## 4. Nouvel ordre d’exécution

`AddCSVStudentHandler` suit désormais cet ordre :

1. authentification et validation de la méthode HTTP ;
2. appel immédiat à `CheckCSVFile`, sans aucune lecture préalable du formulaire ;
3. installation de `MaxBytesReader` ;
4. parsing multipart explicite et traitement immédiat de son erreur ;
5. extraction du fichier CSV ;
6. lecture directe de `class_code_id` depuis `r.MultipartForm.Value` ;
7. validation CSV puis démarrage de la transaction métier.

Aucun `FormValue`, `FormFile` ou parsing implicite ne précède désormais l’installation de la limite.

## 5. Emplacement de `MaxBytesReader`

La constante ciblée `tools.MaxCSVRequestBytes` conserve la valeur existante :

```go
2 << 20
```

soit 2 MiB (2 × 1024 × 1024 octets).

`CheckCSVFile` reçoit le `http.ResponseWriter`, remplace immédiatement `r.Body` par `http.MaxBytesReader`, puis appelle `ParseMultipartForm`. L’erreur originale est encapsulée avec `%w`, ce qui conserve la détection structurée de `*http.MaxBytesError` via `errors.As`.

## 6. Contrat réel de la limite multipart

La limite s’applique au corps HTTP multipart complet, et pas uniquement au contenu du fichier CSV. Elle inclut donc :

- les séparateurs et en-têtes multipart ;
- le champ `class_code_id` ;
- les métadonnées de la partie fichier ;
- les octets du CSV.

L’interface continue d’exprimer une limite de 2 Mio sans promettre qu’un fichier de 2 Mio exact, auquel s’ajouterait l’enveloppe multipart, sera accepté.

## 7. Gestion du dépassement

Une erreur `http.MaxBytesError` produit une redirection vers le mécanisme métier existant avec le message :

> La requête d’import CSV dépasse la taille maximale autorisée de 2 Mio.

Une erreur multipart différente produit un message distinct indiquant que le formulaire est invalide ou incomplet. Le dépassement n’est donc présenté ni comme un CSV structurellement invalide, ni comme un doublon, ni comme une panne DB.

Le parsing échoue avant la lecture de `class_code_id`, avant la validation du CSV et avant l’ouverture de la transaction d’import.

## 8. Absence d’écriture partielle

Le test d’intégration construit une véritable enveloppe multipart dont `ContentLength` dépasse `MaxCSVRequestBytes`. Il vérifie :

- le refus déterministe avec le message de dépassement ;
- l’absence de nouvel élève ;
- l’absence de relation élève/classe ;
- l’absence de transaction métier partiellement appliquée.

Le chemin valide sous la limite conserve le comportement existant et importe normalement les données.

## 9. Nettoyage des ressources multipart

Sur erreur de parsing ou d’extraction du fichier, `CheckCSVFile` appelle `MultipartForm.RemoveAll` lorsqu’un formulaire partiel existe.

Après un parsing réussi, le handler diffère :

1. la fermeture du fichier multipart ;
2. la suppression des éventuels fichiers temporaires avec `r.MultipartForm.RemoveAll`.

L’ordre LIFO des `defer` ferme le fichier avant de retirer les ressources multipart. Aucun nouveau fichier temporaire persistant n’est introduit.

## 10. Fichiers modifiés

- `internal/handlers/students/handlers.go` ;
- `internal/handlers/students/handlers_test.go` ;
- `internal/handlers/tools/checkCSVFile.go` ;
- `internal/handlers/tools/checkCSVFile_test.go` ;
- `docs/audits/student-csv-upload-size-limit.md`.

## 11. Tests et validations

Cas ajoutés :

- multipart valide sous la limite ;
- corps multipart réel supérieur à 2 MiB et reconnaissance de `http.MaxBytesError` ;
- multipart malformé ;
- refus du handler sans création d’élève ni de relation.

Résultats :

| Commande | Résultat |
| --- | --- |
| `go test ./internal/handlers/tools` | succès |
| `go test ./internal/handlers/students` | succès |
| `go test ./...` | succès |
| `git diff --check` | succès |
| `gofmt` sur les fichiers Go modifiés | appliqué |

## 12. Invariants préservés

Le jalon ne modifie pas :

- la valeur de la limite existante ;
- le nom du champ `class_code_id` ;
- le nom du champ fichier `csvfile` ;
- les routes ou méthodes HTTP ;
- l’ownership de la classe ;
- la structure, l’UTF-8, les colonnes, les champs vides ou le nombre maximal de lignes du CSV ;
- la transaction et son rollback ;
- le schéma, les migrations, le SQL ou SQLC ;
- la suppression des guillemets littéraux, volontairement hors périmètre ;
- la classification DB des créations CSV, volontairement hors périmètre.

## 13. Statut du P1

**Résolu.**

La limite de 2 MiB est installée avant toute lecture ou tout parsing multipart, le dépassement est distingué proprement, aucune écriture métier ne commence dans ce cas et les ressources multipart sont nettoyées.
