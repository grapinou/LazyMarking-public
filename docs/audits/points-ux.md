# Refonte UX — Points

## Résumé

Points reprend désormais la grammaire visuelle des cinq référentiels textuels tout en conservant sa nature numérique et son contrat métier : entier supérieur ou égal à 1, sans limite haute. Aucun comportement métier, SQL, FK ou migration n'a été modifié.

## Liste

La liste « Points » utilise des cartes responsive et présente chaque valeur avec l'accord correct : `1 point`, `2 points`, `5 points`. Elle fournit les actions textuelles Modifier/Supprimer, l'action principale « Ajouter une valeur de points » et le retour « Retour à la banque de questions ».

L'état vide est une carte pédagogique proposant directement l'ajout. L'ancienne table, la colonne technique Edit/Sup et les slices parallèles ont été supprimées.

## View-data

`PointListItem` contient l'ID `int64`, la valeur `int64`, l'URL Edit et l'URL Delete. `PointContext` porte l'ID et la valeur pour la confirmation de suppression. `PointURL` construit les URLs propres au domaine.

`PointFormData` existant est conservé et simplifié : il contient uniquement l'ID et la valeur courante. `Options` a été retiré. `PointPageData` expose les items, le contexte, le formulaire et `CancelURL`.

Il ne reste aucun `ExtraData`, `Seq` ou `Options` dans Points.

## Contrôle numérique

Les formulaires Add et Edit utilisent :

```html
<input type="number" min="1" step="1">
```

Aucun attribut `max` n'est défini. Une valeur stockée comme 101 est donc directement préremplie et éditable. La validation HTML complète, sans remplacer, la validation serveur et le `CHECK(point_value >= 1)` existants.

## Ajouter

Le formulaire « Ajouter une valeur de points » utilise le label « Nombre de points », un exemple 5, puis les actions Ajouter et Annuler. Annuler revient à la liste Points. Le handler continue d'accepter 1, 100, 101 et toute autre valeur entière positive.

## Modifier

Le formulaire « Modifier la valeur de points » préremplit exactement `PointFormData.CurrentValue`. Une ouverture puis une soumission sans changement conserve la valeur, y compris au-delà de 100.

Une information explicite précise : « Modifier cette valeur affectera toutes les questions qui l'utilisent. » Aucun comptage ni aucune nouvelle requête n'ont été ajoutés.

Les actions sont Enregistrer et Annuler, ce dernier revenant à Points.

## Supprimer

La confirmation « Supprimer la valeur de points » affiche la valeur avec le bon accord, demande confirmation et précise qu'une valeur utilisée par une question ne peut pas être supprimée. Les actions sont Supprimer et Annuler.

Le POST, `point_id`, l'ownership, la FK RESTRICT et la classification des erreurs restent inchangés.

## Tests ajoutés ou adaptés

- plusieurs valeurs produisent les bons `PointListItem` et les bonnes URLs ;
- les valeurs 5 et 101 sont exposées sans plafond artificiel ;
- une liste vide produit une slice vide propre ;
- Add fournit la bonne `CancelURL` ;
- Delete fournit le bon `PointContext` et la bonne `CancelURL` ;
- Edit fournit 5 ou 101 comme valeur courante et la bonne `CancelURL` ;
- une soumission sans changement conserve 5 ;
- les tests existants confirment toujours l'acceptation de 1, 100 et 101, ainsi que le rejet de 0 et -1.

Les tests de migration 0034, ownership, FK utilisée et erreur DB non-FK restent inchangés et passants.

## Règles métier

Aucune règle métier n'a été modifiée. Le contrat `>= 1`, l'absence de limite haute, le `CHECK`, les FK Questions, l'effet partagé, l'ownership et les suppressions sécurisées sont inchangés.

## Validations

- `go test ./...` : succès ;
- `go vet ./...` : succès ;
- `git diff --check` : succès.

## Fichiers modifiés

- `internal/handlers/points/handlers.go` ;
- `internal/handlers/points/handlers_test.go` ;
- `internal/templates/data/point.go` ;
- `internal/templates/points/table_points.html` ;
- `internal/templates/points/add_form_point.html` ;
- `internal/templates/points/edit_form_point.html` ;
- `internal/templates/points/delete_form_point.html` ;
- `docs/audits/points-ux.md`.
