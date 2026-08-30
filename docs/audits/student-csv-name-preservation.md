# Conservation intégrale des noms importés par CSV

## 1. Comportement avant correction

`ValidateCSVStructure`, dans `internal/handlers/tools/checkCSVStructure.go`, validait chaque ligne puis transformait les deux champs prénom/nom de la façon suivante :

1. validation UTF-8 ;
2. suppression des guillemets et espaces périphériques selon le traitement existant ;
3. rejet d’un champ vide ;
4. conversion en `[]rune` ;
5. coupe silencieuse aux 25 premières runes.

La troncature concernait donc le prénom comme le nom et intervenait avant leur insertion en base. La valeur complète fournie par l’utilisateur était définitivement perdue sans avertissement.

## 2. Origine de la limite

La limite était une constante locale `maxNameLength = 25` dans le validateur CSV. Aucun commentaire, migration, contrainte de schéma, texte de template ou autre règle métier du dépôt ne documentait son origine.

La recherche des usages confirme que `ValidateCSVStructure` est utilisé par l’import CSV Élèves. L’ajout manuel ne possédait pas cette limite.

## 3. Décision produit

LazyMarking conserve désormais l’identité complète saisie ou importée. Les contraintes de mise en page devront être traitées dans le rendu concerné et non par une mutation de la donnée source.

La constante, la conversion en runes et le découpage ont été supprimés. Aucun remplacement par une autre longueur maximale n’a été introduit.

## 4. Nouveau contrat CSV

Après validation de la structure, de l’UTF-8 et du contenu non vide, prénom et nom sont retournés intégralement par `ValidateCSVStructure`.

Par exemple :

```text
"Jean-Christophe-Alexandre";"Dupond-Dupont-Très-Long"
```

produit exactement :

- `Jean-Christophe-Alexandre` ;
- `Dupond-Dupont-Très-Long`.

La longueur n’est plus un motif de refus ni de transformation.

## 5. Cohérence avec l’ajout manuel

L’ajout manuel continue d’appliquer son `strings.TrimSpace` existant. Le parseur CSV conserve son traitement existant des guillemets et espaces périphériques ; aucune nouvelle normalisation n’a été ajoutée dans ce jalon.

Un test d’intégration crée deux élèves aux identités longues comparables, l’un par `AddStudentHandler`, l’autre par le multipart CSV `AddCSVStudentHandler`. Les valeurs longues normalisées sont relues en base et comparées exactement aux chaînes complètes attendues.

Les contraintes communes de non-vide et d’unicité restent celles de la base. Aucun comportement spécifique à la longueur ne subsiste dans l’import.

## 6. Traitement Unicode

La validation `utf8.ValidString` reste exécutée avant toute normalisation. Les chaînes valides ne sont plus converties en runes pour être découpées et restent intactes.

Les tests utilisent des valeurs de plus de 25 runes comportant notamment :

- accents (`É`, `é`) ;
- apostrophe typographique (`’`) ;
- caractères composés visuellement (`Coëffé`, `Ångström`) ;
- caractères turcs et CJK (`Çağdaş`, `李小龍`, `非常に長い名前`).

La valeur analysée et la valeur stockée sont comparées exactement, ce qui garantit l’absence d’altération en bytes comme en runes.

## 7. Validations CSV conservées

Le changement ne concerne que la coupe à 25 runes. Restent inchangés :

- séparateur `;` ;
- exactement deux colonnes par ligne ;
- rejet d’un CSV vide ;
- rejet d’un champ vide après le traitement périphérique existant ;
- rejet d’un encodage UTF-8 invalide ;
- rejet d’une structure CSV invalide ;
- limite de 10 000 enregistrements ;
- limite multipart de 2 Mio via `CheckCSVFile` ;
- ownership de la classe ;
- transaction globale d’import ;
- rollback complet en cas d’échec ;
- contraintes DB de non-vide et d’unicité.

Le template d’import n’affichait aucune limite de longueur : il n’a pas été modifié.

## 8. Fichiers modifiés

- `internal/handlers/tools/checkCSVStructure.go`
- `internal/handlers/tools/checkCSVStructure_test.go`
- `internal/handlers/students/handlers_test.go`
- `docs/audits/student-csv-name-preservation.md`

Les autres modifications préexistantes du working tree n’ont pas été touchées.

## 9. Tests et résultats

Les tests couvrent désormais :

- conservation exacte d’un prénom et d’un nom longs ASCII/latin ;
- conservation exacte de deux valeurs Unicode dépassant l’ancienne limite ;
- normalisation périphérique préexistante sans raccourcissement ;
- stockage intégral via un véritable import multipart CSV ;
- cohérence avec l’ajout manuel pour des identités longues comparables ;
- validations préexistantes : CSV vide, colonne manquante, colonne supplémentaire et UTF-8 invalide.

Résultats :

- `go test ./internal/handlers/tools ./internal/handlers/students` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go modifiés : succès.

## 10. Invariants préservés

Aucun changement n’a été apporté au schéma, aux migrations, au SQL, à SQLC, aux contraintes d’unicité, à l’ownership, à la transaction d’import, aux routes, aux paramètres HTTP, aux suppressions, aux templates, à la génération PDF ou au rendu Typst.

La taille des fichiers, le nombre de lignes, la validation UTF-8, la structure du CSV, les doublons et le rollback restent protégés comme avant.

## 11. Statut du P2

**Résolu.** L’import CSV conserve désormais intégralement les prénoms et noms, y compris les longues valeurs Unicode, sans limite silencieuse à 25 runes.
