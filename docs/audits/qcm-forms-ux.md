# Harmonisation UX des formulaires QCM

## Création

Le formulaire « Créer un QCM » est présenté dans une carte Bootstrap de largeur raisonnable. Il conserve l’unique champ métier `qcm`, désormais libellé « Nom du QCM », avec l’exemple « Forces et interactions ».

Les actions sont « Créer le QCM » et « Annuler ». Annuler retourne vers Mes QCM via `Routes.QcmURL`.

## Modification

Le formulaire « Modifier le QCM » affiche le nom courant dans son contexte d’en-tête et préremplit le champ `new_qcm`. Le `qcm_id` caché, l’action et la méthode POST restent inchangés.

Les actions sont « Enregistrer » et « Annuler ». Le retour vers Mes QCM est cohérent avec le point d’entrée Modifier de la liste.

## Suppression

La confirmation utilise le wording professionnel suivant :

- titre : « Supprimer le QCM » ;
- confirmation : « Voulez-vous supprimer ce QCM ? » ;
- précision : « Les questions de votre banque ne seront pas supprimées. »

Le nom du QCM est clairement affiché. Le bouton « Supprimer » conserve le style Bootstrap danger et Annuler retourne vers Mes QCM. Le formulaire POST et son `qcm_id` restent inchangés.

## Comportement Annuler

Les trois formulaires utilisent la même destination typée `Routes.QcmURL`, soit la liste Mes QCM. Aucune URL parallèle dans `ExtraData` n’a été ajoutée.

## Ancien filtrage des guillemets

Les templates de création et modification contenaient une fonction JavaScript `removeForbiddenCharacters` appelée par `oninput`, qui supprimait les guillemets doubles. Cette restriction n’avait aucun équivalent dans le contrat backend et a été supprimée intégralement.

Aucun JavaScript personnalisé ne subsiste dans les trois formulaires.

## QCMContext et données de vue

`QCMContext` reste la donnée parentale typée des formulaires Modifier et Supprimer. Le handler de création utilise les routes typées existantes. Aucun `ExtraData` redondant n’était nécessaire et aucun nouveau modèle générique n’a été créé.

De petites fonctions de rendu injectables ont été ajoutées uniquement pour capturer les PageData dans les tests, sans changer le rendu de production.

## Guillemets et apostrophes

Le backend continue à appliquer uniquement `strings.TrimSpace` avant les requêtes paramétrées. Les tests démontrent que des noms tels que `Chapitre "Forces" et l'action mécanique` sont créés puis stockés sans altération, et qu’un nom contenant simultanément apostrophe et guillemets peut être enregistré lors d’une modification.

La règle d’unicité `(name, user_id)`, le rejet des noms vides, l’ownership et les lignes affectées ne changent pas.

## Responsive et accessibilité

- largeur responsive centrée ;
- cartes Bootstrap ;
- labels associés par `for`/`id` ;
- actions textuelles avec `flex-wrap` ;
- bouton destructif explicite ;
- contenu long autorisé à revenir à la ligne.

## Règles métier

Aucune règle métier, migration, requête SQL, sémantique de suppression, Preview, génération ou donnée Exams n’a été modifiée. Le refus d’un QCM utilisé par une évaluation reste couvert par les tests existants.

## Tests

Trois fonctions de test, représentant six scénarios ciblés, ont été ajoutées :

- PageData des formulaires créer/modifier/supprimer : bonne URL Annuler et bons `QCMContext` ;
- création avec nom normal ;
- création avec guillemets et apostrophe ;
- modification avec guillemets et apostrophe.

Les tests ownership et suppression protégée existants continuent à passer. Aucune classe Bootstrap ni structure détaillée de HTML n’est testée.

## Validations

- `go test ./...` : succès.
- `go vet ./...` : succès.
- `git diff --check` : succès.

## Fichiers modifiés par ce jalon

- `internal/handlers/qcm/handlers.go`
- `internal/handlers/qcm/handlers_test.go`
- `internal/templates/qcm/add_form_qcm.html`
- `internal/templates/qcm/edit_form_qcm.html`
- `internal/templates/qcm/delete_form_qcm.html`
- `docs/audits/qcm-forms-ux.md`

Les autres entrées visibles dans le worktree (`README.md`, `app`, les autres fichiers de `docs/` et `reset.sh`) préexistaient à ce jalon et n’ont pas été modifiées pour cette harmonisation.
