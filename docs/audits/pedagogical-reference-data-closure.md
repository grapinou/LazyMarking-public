# Audit de clôture — Référentiels pédagogiques

## Résumé exécutif

Le chantier est terminé. Les six référentiels disposent d'une UX cohérente, de données de vue typées, d'un ownership effectif, de suppressions sûres et correctement classifiées, ainsi que d'une couverture ciblée. Le contrat numérique Points est aligné entre DB, serveur et interface.

Sur les neuf constats prioritaires de l'audit initial, huit sont résolus. La seule réserve restante concerne d'éventuelles relations historiques antérieures à la migration 0030. Aucun état incohérent réel n'est démontré ; ce point est une vérification ponctuelle de dette technique, pas un correctif requis avant le chantier suivant.

## Statut des constats initiaux

| Priorité initiale | Constat | Statut actuel | Vérification |
|---|---|---|---|
| P1 | Edit Points non prérempli | RÉSOLU | `PointFormData.CurrentValue` alimente directement l'input ; 5 et 101 sont testés ; le POST inchangé conserve 5. |
| P1 | Points non contraint serveur/DB | RÉSOLU | migration 0034, `CHECK(point_value >= 1)`, validation Add/Edit, aucune limite haute. |
| P2 | Toute erreur DELETE présentée comme FK | RÉSOLU | helper SQLite structuré ; FK vers message métier, autre erreur vers 500. |
| P2 | View-data fragile et slices parallèles | RÉSOLU | six `*ListItem`, six contextes ciblés, IDs `int64`, URLs associées dans chaque item. |
| P2 | JavaScript retirant les guillemets | RÉSOLU | aucun script restant ; apostrophes/guillemets testés dans les cinq domaines textuels. |
| P2 | Couverture suppression/handlers inégale | RÉSOLU | matrices DB et handlers sur les six domaines, plus tests view-data propres. |
| P2 | Cohérence historique pré-0030 non vérifiée rétroactivement | TOUJOURS PRÉSENT, reclassé P3 | aucune donnée réelle incohérente démontrée ; mutations et lectures actuelles sont protégées. |
| P3 | Listes CRUD anciennes | RÉSOLU | cartes responsive, actions textuelles, états vides et retour Banque. |
| P3 | Formulaires/terminologie anciennes | RÉSOLU | Add/Edit/Delete, Annuler, labels, confirmations et vocabulaire français harmonisés. |

## État des six UX

Pour Matières, Thèmes, Niveaux, Compétences, Difficultés et Points, le code actuel confirme :

- liste responsive sans table technique ;
- état vide pédagogique ;
- formulaires Ajouter, Modifier et Supprimer ;
- action Annuler vers la liste du domaine ;
- retour à la banque de questions ;
- actions textuelles ;
- item de liste typé ;
- contexte Edit/Delete typé ;
- aucun `ExtraData` fonctionnel ;
- aucun JavaScript local filtrant les guillemets.

Les structures finales sont :

- `SubjectListItem` / `SubjectContext` ;
- `ThemeListItem` / `ThemeContext` ;
- `YearLevelListItem` / `YearLevelContext` ;
- `SkillListItem` / `SkillContext` ;
- `DifficultyListItem` / `DifficultyContext` ;
- `PointListItem` / `PointContext`, complétés par `PointFormData`.

Il n'existe plus de slice `Action` parallèle, de booléen `NoX`, ni d'ID parental transporté en chaîne dans ces PageData. La ressemblance assumée entre les types ne justifie pas une abstraction CRUD générique.

## Terminologie

La recherche dans les handlers, PageData et templates des six domaines ne trouve plus les reliquats visibles suivants :

- « Classe/Classes » à la place de Niveau/Niveaux ;
- `difficultée` ou `difficil` ;
- « Back to question » ;
- « Edit/Sup » ;
- « C'est mon dernier mot » ;
- « Es-tu sur ».

Points affiche l'accord `1 point` ou `n points`. Les titres, champs, actions et confirmations utilisent le vocabulaire métier français attendu.

## Suppressions

Le contrat est identique et correct dans les six domaines :

- valeur possédée libre : suppression puis HTTP 303 vers la liste ;
- valeur utilisée : FK `ON DELETE RESTRICT`, question et référentiel conservés, HTTP 303 vers le message métier ;
- valeur absente ou étrangère : zéro ligne puis HTTP 404 ;
- autre erreur DB : HTTP 500.

`IsSQLiteForeignKeyConstraint` utilise les codes structurés SQLite sans analyser le texte. Il couvre le code FK et le code réellement émis par un `ON DELETE RESTRICT`. Une erreur SQL non-FK est testée séparément.

QCM utilise désormais ce même helper. Son contrat reste inchangé : un QCM protégé par une évaluation produit le message métier, les autres erreurs restent serveur, et les tests QCM passent.

## Points

Le contrat final est confirmé :

- `point_value INTEGER NOT NULL CHECK(point_value >= 1)` ;
- validation serveur Add/Edit `>= 1` ;
- aucune limite haute DB ou serveur ;
- `<input type="number" min="1" step="1">` sans `max` ;
- valeur 101 directement préremplie et éditable ;
- soumission sans changement conservant la valeur ;
- information visible sur l'effet partagé d'une modification ;
- `questions.point_id` toujours obligatoire et `ON DELETE RESTRICT`.

La migration 0034 vérifie les anciennes données avant reconstruction, préserve les IDs et les références Question, et refuse explicitement une ancienne valeur non positive sans correction silencieuse.

## Ownership

Le CRUD des six domaines reste ownership-aware : session pour la création, listes/lookups filtrés par `user_id`, mutations `id + user_id`, contrôle des lignes affectées et 404 pour absent/étranger.

Les associations Question sont protégées à trois niveaux : mutations SQL ownership-aware, triggers `questions_owner_insert/update` de 0030 et lectures filtrant la cohérence des six parents. Un utilisateur ne peut pas associer à sa question un référentiel étranger par les parcours actuels.

### Réserve pré-0030

La migration 0030 n'a pas scanné rétroactivement les lignes créées avant ses triggers. Une incohérence historique éventuelle serait cachée par les lectures modernes, mais aucun cas réel n'est établi. Recommandation : conserver un diagnostic read-only ponctuel dans la dette technique et l'exécuter avant une future reconstruction de `questions` ou lors d'une maintenance de base réelle. Aucun chantier ni blocage n'est nécessaire maintenant.

## Intégration Questions et QCM

Aucune régression n'est trouvée :

- création et modification Question chargent les six listes et envoient les six IDs aux mutations ownership-aware ;
- la banque construit toujours les familles depuis les questions principales ;
- `GetFilteredQuestions` applique toujours indépendamment les six filtres ;
- le sélecteur QCM transmet toujours matière, thème, niveau, compétence, difficulté et points à cette requête ;
- les variantes restent des enfants sans référentiels indépendants.

Les refactors réalisés concernent les PageData et templates propres aux référentiels ; ils n'ont modifié aucun contrat Question/QCM.

## Couverture de tests

La couverture utile est suffisante :

- ownership et rows affected table-driven pour les six référentiels ;
- associations Question étrangères refusées et lectures incohérentes masquées ;
- listes typées, URLs, états vides, contextes et CancelURL ;
- Add/Edit avec apostrophes et guillemets pour les cinq domaines textuels ;
- suppression libre/utilisée, conservation de la question, absent/étranger et vraie erreur DB sur les six domaines ;
- Points accepte 1, 100 et 101, refuse 0 et -1 ;
- Edit Points conserve la valeur courante ;
- migration 0034 Up/Down, données historiques invalides et références Question.

Le seul manque à valeur réelle correspond à la réserve historique : aucun test/outil n'inspecte une base réelle pré-0030. Il ne manque pas de test métier courant justifiant un correctif avant la suite.

## Checklist smoke manuel

1. Créer une matière, un thème, un niveau, une compétence, une difficulté et une valeur de points.
2. Modifier chaque valeur, dont un nom avec apostrophe/guillemets.
3. Supprimer une valeur libre dans chaque domaine.
4. Associer les six métadonnées à une question, puis tenter de supprimer chacune : vérifier le refus métier.
5. Vérifier que la question et ses six références existent toujours après les refus.
6. Créer puis modifier une question avec les nouvelles valeurs.
7. Filtrer la banque sur chacun des six critères.
8. Filtrer le sélecteur QCM sur chacun des six critères.
9. Créer puis modifier une valeur Points supérieure à 100.
10. Vérifier listes, cartes et actions sur une largeur mobile.

## Classification finale

### P0 — 0

Aucun problème de sécurité, perte ou corruption active identifié.

### P1 — 0

Aucun bug métier ou défaut d'intégrité important restant.

### P2 — 0

Aucun correctif requis avant le prochain chantier.

### P3 — 1

**Diagnostic historique pré-0030.** Impact potentiel limité à une ancienne base ayant déjà contenu une relation Question/référentiel inter-utilisateur avant les triggers. Fichiers de référence : migrations 0029/0030, requêtes Questions et `questionGraphIntegrity_test.go`. Recommandation : diagnostic read-only ponctuel avant une future reconstruction de la table Questions. Taille : petite.

## Verdict

**A — référentiels pédagogiques terminés, aucun correctif nécessaire avant le prochain chantier.**

Le prochain chantier naturel dans le workflow produit est l'audit du bloc Évaluations/Exams, qui consomme les QCM désormais stabilisés.

## Périmètre de cet audit

Seul ce rapport a été créé. Aucun fichier Go, SQL, migration, template, test ou configuration n'a été modifié.
