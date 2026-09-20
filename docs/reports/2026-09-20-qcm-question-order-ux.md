# P4 — Ordre des questions dans les QCM

## Fonctionnement initial

L'ordre est porté par la liaison famille de question ↔ QCM :
`qcm_questions.position`, entier obligatoire >= 1, unique par QCM.
Les ajouts s'effectuent en fin de séquence, les suppressions compactent les
positions. Les lectures SQL/sqlc trient par position.

Les déplacements échangent deux positions adjacentes dans une transaction,
via une position temporaire au-delà du maximum. Les contraintes d'unicité,
contrôles de propriété et rollback restent ceux existants. La continuité
est maintenue par les opérations applicatives, pas par le seul CHECK SQL.

La composition affichait un numéro de position et deux formulaires POST
Monter/Descendre sur chaque carte, même pour une seule question.
Le premier bouton Monter et le dernier Descendre étaient désactivés.

## Analyse UX

L'ordre manuel reste utile pour organiser la composition et les aperçus.
Sa fréquence d'utilisation n'est pas mesurée ; le retour utilisateur justifie
de le traiter comme une action occasionnelle. Deux boutons permanents par
question occupent beaucoup de place, particulièrement sur mobile et avec
30 questions, et rendent possible un déplacement involontaire.

## Décision retenue

Option B : mode « Réorganiser », accessible par un seul lien secondaire au-dessus
de la liste lorsque le QCM contient au moins deux questions.
La consultation conserve les numéros, les énoncés et le retrait de questions ;
les formulaires de déplacement ne sont rendus que dans le mode dédié.

Ce mode conserve les boutons accessibles au clavier, sans dépendance ni
JavaScript. Un menu par question conserverait trente déclencheurs répétés ;
le drag & drop ajouterait des interactions et tests sans besoin démontré.
Supprimer les commandes ferait perdre l'organisation des aperçus.

Chaque déplacement est immédiatement enregistré. Le retour après POST reste
dans ce mode. « Terminer la réorganisation » quitte le mode, sans annuler les
déplacements. Les limites restent désactivées et possèdent des libellés
accessibles indiquant la position. Aucun contrôle pour zéro ou une question,
même avec le paramètre de mode fourni manuellement.

## Architecture

Routes GET de composition et POST move-up/move-down conservées.
Le GET accepte `reorder=1` et transmet un booléen ainsi que les URLs de mode
dans `QCMQuestionPageData`. Les formulaires du mode transmettent ce paramètre.
Le handler POST conserve alors le paramètre dans sa redirection locale.
Un POST historique sans paramètre garde sa redirection habituelle.

Le template réutilise les formulaires CSRF existants, les identifiants et les
indicateurs IsFirst/IsLast. Les transactions, requêtes SQL/sqlc et contraintes
ne changent pas. Aucune migration ; aucun ordre existant modifié par la lecture
ou l'activation du mode.

## Génération

- Composition et aperçus portrait/paysage : lecture dans l'ordre des positions.
  Les aperçus utilisent `GetQCMQuestionsAnswersInReferenceOrder`, qui ne
  mélange pas les familles.
- Copies individualisées : `BuildQcmStudentCtx` appelle
  `GetQCMQuestionsAnswersCtx`, qui mélange les IDs avant construction.
- Mini-test : `GetQCMQuestionsAnswers` effectue également ce mélange.
- Variantes : BuildQuestion/BuildQuestionCtx tire une formulation dans chaque
  famille. Les réponses sont également mélangées. Ces tirages restent distincts
  du classement des familles ; un aperçu peut donc conserver l'ordre des
  familles tout en choisissant des formulations différentes.
- Les numéros sur les documents suivent l'ordre effectif du document. Les
  positions de composition ne sont pas les numéros garantis des copies élèves.
- La correction utilise le snapshot JSON de chaque copie et ses coordonnées,
  enregistré dans student_exam_content. Réordonner ensuite le QCM ne modifie
  ni les copies déjà générées ni leurs corrections.
- Les statistiques cumulées exploitent les résultats et snapshots, avec
  regroupement par famille/version ; elles ne relisent pas les positions du QCM.

Le template explique cette différence : ordre utilisé dans les aperçus,
questions mélangées sur les copies élèves et le mini-test.
Les algorithmes aléatoires restent inchangés ; aucune identité bit à bit de
deux nouvelles générations aléatoires n'est promise.

## Tests

- Nouveau test HTTP de consultation normale, activation du mode, bornes,
  une seule question, POST haut/bas, conservation du mode et rechargement
  dans l'ordre persisté A–C–B puis A–B–C.
- Rendu de 10, 20 et 30 questions à énoncés longs : toutes présentes,
  formulaires absents en consultation et présents en réorganisation.
- Tests existants conservés : échanges adjacents, bornes sans effet, rollback,
  ownership, compaction des positions et redirections historiques.
- Tests de génération existants : aperçu sans mélange dans l'ordre des
  positions, génération respectant la permutation tirée malgré la concurrence.

La mise en page conserve les cartes fluides, text-break et flex-wrap Bootstrap :
aucune refonte CSS. Le rendu HTML est testé ; aucune validation visuelle
interactive desktop/mobile n'a été réalisée dans ce chantier.

## Points ouverts

Validation technique réussie :

- `go test ./...`.
- `go test ./internal/handlers/qcm ./internal/handlers/qcmQuestions ./internal/handlers/qcmPreview ./internal/handlers/tools -count=1`.
- `git diff --check`.

Les tests utilisent `GOCACHE=/tmp/lazymarking-go-cache`.

Aucun blocage fonctionnel identifié. Déplacer directement une question sur une
grande distance pourrait devenir utile pour des compositions très longues ;
ce besoin n'est pas établi par le retour utilisateur actuel.
