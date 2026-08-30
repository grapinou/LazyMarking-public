# Conservation des guillemets dans les noms de classes

## 1. Comportement avant correction

Les formulaires d’ajout et d’édition de classe contenaient chacun une fonction JavaScript locale `removeForbiddenCharacters`.

L’attribut `oninput="removeForbiddenCharacters(this)"` remplaçait silencieusement tous les guillemets doubles `"` par une chaîne vide pendant la saisie. La valeur visible et transmise ne correspondait donc plus à celle entrée par l’utilisateur.

La fonction n’était utilisée que par l’input du template qui la déclarait. Aucun autre comportement utile, script ou contrôle de formulaire n’en dépendait, et aucun commentaire connexe n’était présent.

## 2. Incohérence avec le backend

`AddClassCodeHandler` et `EditClassCodeHandler` appliquent uniquement `strings.TrimSpace` au nom reçu.

Le schéma final impose :

- un nom non vide après trim via `CHECK` ;
- l’unicité du nom par utilisateur ;
- aucune interdiction des apostrophes ou guillemets.

Le filtrage frontend créait donc une restriction non documentée et plus forte que le contrat métier et DB.

## 3. Suppression du filtrage JavaScript

Dans les deux templates :

- l’attribut `oninput` a été supprimé ;
- le bloc `<script>` contenant `removeForbiddenCharacters` a été supprimé ;
- aucune fonction de remplacement ni autre transformation frontend n’a été ajoutée.

Les formulaires conservent leur structure, leurs labels, champs requis, actions, routes et boutons existants.

## 4. Nouveau contrat

Une valeur telle que :

```text
6e "A"
```

reste saisissable dans le navigateur, est transmise intacte au handler, subit uniquement le `TrimSpace` backend existant, puis est stockée avec ses guillemets si elle respecte le non-vide et l’unicité.

L’encodage HTML automatique du template d’édition (`&#34;`) protège l’attribut `value` tout en restituant au navigateur la même valeur logique contenant des guillemets.

## 5. Ajout

Un test POST envoie `  6e "A"  ` à `AddClassCodeHandler`. Il vérifie :

- la redirection normale vers la liste Classes ;
- le trim des seuls espaces périphériques ;
- la présence exacte de `6e "A"` en base ;
- la conservation des deux guillemets.

## 6. Édition

Un test POST renomme une classe en `  6e "Alpha"  `. Il vérifie la redirection normale et relit exactement `6e "Alpha"` dans la base.

Un test de rendu utilise également un nom contenant des guillemets et confirme qu’il est présent, correctement échappé, dans la valeur préremplie du formulaire Edit.

## 7. Tests et résultats

Un garde-fou lit directement les deux templates et refuse toute réapparition de :

- `removeForbiddenCharacters` ;
- un attribut `oninput`.

Les tests existants de rendu, annulation, `UNIQUE`, `CHECK`, erreurs DB inattendues et ownership restent passants.

Résultats :

- `go test ./internal/handlers/classCodes` : succès ;
- `go test ./...` : succès ;
- `git diff --check` : succès ;
- `gofmt` appliqué aux fichiers Go de test modifiés : succès.

Fichiers modifiés :

- `internal/templates/classcodes/add_form_class_code.html` ;
- `internal/templates/classcodes/edit_form_class_code.html` ;
- `internal/handlers/classCodes/handlers_test.go` ;
- `internal/handlers/classCodes/viewData_test.go` ;
- `docs/audits/class-code-quote-preservation.md`.

## 8. Invariants préservés

Aucun changement n’a été apporté au backend, à `TrimSpace`, aux contraintes `CHECK`/`UNIQUE`, à la classification des erreurs DB, à l’ownership, aux routes, aux paramètres HTTP, au SQL, à SQLC, aux migrations, au schéma, aux suppressions, aux relations élève/classe, à l’import CSV ou aux autres modules.

## 9. Statut du P3

**Résolu.** Les formulaires Classes ne retirent plus silencieusement les guillemets et reflètent désormais exactement le contrat backend.
