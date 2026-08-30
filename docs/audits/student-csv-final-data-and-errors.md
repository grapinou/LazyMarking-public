# Contrat final des données et erreurs de l’import CSV

## 1. Suppression silencieuse des guillemets avant correction

`ValidateCSVStructure` appliquait après décodage :

```go
strings.Trim(value, "\" ")
```

Cette opération retirait non seulement les espaces ASCII périphériques, mais aussi tout guillemet double placé au début ou à la fin de la valeur logique. Une identité correctement encodée pouvait donc être modifiée silencieusement avant son stockage.

## 2. Comportement de `encoding/csv`

Le parseur standard `encoding/csv` est responsable de la syntaxe du format : il interprète les guillemets syntaxiques et restitue la valeur logique décodée. Un guillemet qui demeure dans cette valeur fait partie de la donnée utilisateur et ne doit pas être supprimé par une seconde transformation.

Les nouveaux tests utilisent `csv.Writer` avec le séparateur `;` afin de produire de vrais enregistrements correctement échappés. Ils ne simulent pas une valeur déjà décodée.

## 3. Nouveau contrat de normalisation

Chaque champ suit désormais le même contrat que l’ajout manuel :

1. validation UTF-8 ;
2. `strings.TrimSpace` ;
3. refus d’une valeur vide ;
4. conservation de tous les autres caractères ;
5. application des CHECK et UNIQUE DB existants.

Les guillemets doubles, apostrophes et autres signes de ponctuation appartenant à l’identité ne sont plus retirés. Aucune nouvelle transformation ou limite de longueur n’a été introduite.

## 4. Tests de conservation des guillemets

La couverture du parseur vérifie avec des flux CSV réellement encodés :

- des guillemets à l’intérieur du prénom ;
- un guillemet littéral initial ;
- un guillemet littéral final ;
- des guillemets littéraux aux deux extrémités ;
- la conservation de la ponctuation lors du retrait des espaces périphériques.

Un test d’intégration multipart importe également une identité contenant des guillemets littéraux puis relit la ligne en base pour comparer exactement le prénom et le nom stockés.

Les tests existants de noms longs et Unicode continuent à passer.

## 5. Classification DB avant correction

Dans la boucle de `AddCSVStudentHandler`, toute erreur de `CreateStudentAndReturnID` était journalisée puis présentée comme un doublon. Une contrainte CHECK ou une panne SQLite inattendue pouvait donc recevoir un message métier erroné.

Le rollback transactionnel fonctionnait déjà, mais le diagnostic et la nature de la réponse HTTP étaient incorrects.

## 6. UNIQUE, CHECK et erreur inattendue

La branche réutilise désormais les helpers structurés existants :

- `tools.IsSQLiteUniqueConstraint` → redirection métier « Cet élève existe déjà. » ;
- `tools.IsSQLiteCheckConstraint` → redirection métier indiquant que prénom et nom doivent être renseignés ;
- toute autre erreur → log contextualisé `AddCSVStudentHandler -> CreateStudentAndReturnID DB error` puis HTTP 500.

La classification repose sur les codes étendus du driver SQLite et non sur le texte complet d’une erreur.

Dans le flux HTTP normal, `ValidateCSVStructure` refuse les champs vides avant la transaction : une violation CHECK de non-vide ne devrait donc pas être atteinte. Sa classification reste néanmoins présente en défense en profondeur et cohérente avec l’ajout manuel.

## 7. Comportement transactionnel

L’import conserve une transaction unique pour l’ensemble du fichier :

- ouverture après validation multipart, paramètre de classe et structure CSV ;
- utilisation des queries liées à la transaction ;
- rollback différé ;
- sortie immédiate à la première erreur ;
- commit uniquement après toutes les lignes et relations réussies.

Une erreur métier connue et une erreur DB inattendue annulent toutes deux le batch complet. Aucun import partiel n’a été introduit.

La branche `CreateStudentWithClassCode` était déjà correcte pour le périmètre demandé : une erreur inattendue produit HTTP 500 et le rollback supprime l’élève tout juste créé. Un test dédié confirme l’absence d’identité orpheline.

## 8. Rollback du batch

Les tests démontrent :

- première ligne valide puis deuxième ligne en doublon : message métier et rollback de la première ligne avec sa relation ;
- première ligne valide puis trigger SQLite inattendu sur la deuxième insertion : HTTP 500, aucune redirection de doublon et rollback complet ;
- erreur forcée lors de la création de la première relation : HTTP 500 et aucun élève orphelin.

Les élèves et relations de référence préexistants restent inchangés.

## 9. Absence de régression du P1 multipart

Le correctif de limite multipart n’a pas été réorganisé :

- `CheckCSVFile` reste le premier accès au formulaire après l’authentification ;
- `MaxBytesReader` est installé avant `ParseMultipartForm` ;
- la limite reste `MaxCSVRequestBytes` ;
- `class_code_id` est lu depuis `r.MultipartForm.Value` après parsing ;
- `http.MaxBytesError` conserve son message distinct ;
- le fichier est fermé et `MultipartForm.RemoveAll` reste appelé ;
- le test d’une enveloppe multipart réellement supérieure à 2 MiB continue à passer sans écriture DB.

## 10. Fichiers modifiés

- `internal/handlers/tools/checkCSVStructure.go` ;
- `internal/handlers/tools/checkCSVStructure_test.go` ;
- `internal/handlers/students/handlers.go` ;
- `internal/handlers/students/handlers_test.go` ;
- `docs/audits/student-csv-final-data-and-errors.md`.

Aucun SQL, fichier SQLC, schéma, migration, route ou template n’a été modifié.

## 11. Résultats des tests

| Commande | Résultat |
| --- | --- |
| `go test ./internal/handlers/tools` | succès |
| `go test ./internal/handlers/students` | succès |
| `go test ./...` | succès |
| `git diff --check` | succès |
| `gofmt` sur les fichiers Go modifiés | appliqué |

Les validations CSV antérieures restent couvertes : fichier vide, nombre de colonnes, UTF-8, champs vides, limite de lignes, noms longs et Unicode. La transaction, l’ownership et les contraintes DB sont inchangés.

## 12. Statut des deux P2

- **P2 guillemets : résolu.** La normalisation retire uniquement les espaces Unicode périphériques et conserve exactement la ponctuation logique décodée par `encoding/csv`.
- **P2 erreurs DB : résolu.** UNIQUE et CHECK produisent des erreurs métier ciblées ; une panne DB inattendue produit HTTP 500 et annule intégralement le batch.
