# P3 — États Examens et Correction

## États initiaux

Une ligne `exams` ne possède pas de colonne d'état. L'interface Examens
fabriquait la valeur de vue `draft` lorsque la jointure vers `exams_generated`
était absente. Ce cas signifie précisément que les copies individualisées n'ont
jamais été générées. `draft` n'est donc ni écrit ni relu en base.

`exams_generated.status` est contraint à `running`, `success` ou `failed`. Ces
valeurs décrivent la production des copies, avec les libellés existants
« Génération en cours », « Générée » et « Échec de génération ».

`marking_jobs.status` et `status_pdf` utilisent également `running`, `success`
et `failed`. Ils décrivent un import et ses artefacts, pas l'avancement
pédagogique global. L'historique les présente séparément comme traitement en
cours, vérification/résultat disponible ou traitement interrompu.

Pour chaque élève attendu, un job de correction réussi persiste exactement un
`marking_copy_results.outcome` parmi `corrected`, `incomplete`, `not_seen` et
`error`. Les résultats courants P1 retiennent, par élève, une correction acquise
en priorité, sinon le dernier résultat, et uniquement parmi les jobs dont
`status` et `status_pdf` valent `success`. Un résultat `corrected` dont une
détection attend encore une décision humaine est compté à vérifier, sans note
finalisée.

## Mapping UX

| Données observables | Libellé utilisateur |
| --- | --- |
| Exam sans ligne `exams_generated` | Copies non générées |
| Génération `running` | Génération en cours |
| Génération `success` | Générée |
| Génération `failed` | Échec de génération |
| Bilan courant avec zéro copie finalisée | Non corrigé |
| Bilan courant avec au moins une copie finalisée, mais pas toute la cohorte | Correction partielle |
| Toutes les copies du bilan courant sont finalisées | Corrigé |
| Job `running` | Traitement en cours, dans l'historique des imports |
| Job `failed` | Traitement interrompu, dans l'historique des imports |

Les valeurs persistées et leurs contraintes ne changent pas. Les trois états de
correction sont des valeurs de présentation calculées par `buildMarkingProgress`
à partir de `MarkingExamSummaryView`, le bilan courant déjà partagé par l'écran
de résultats et le PDF.

## Examens

Le pseudo-état `draft` conserve son rôle interne dans la vue typée et dans le
choix des actions autorisées : générer, mini test, modifier et supprimer. Seul
son libellé devient « Copies non générées ». Ce mapping est exact car l'absence
de génération est la seule condition qui produit `draft`. Les libellés des trois
états persistés de génération restent inchangés.

## Correction

La liste `/dashboard/marking` charge les générations réussies, puis leurs
résultats courants via `ListCurrentExamResultsForGeneration`. Elle appelle le
même `buildMarkingExamSummary` que le bilan cumulé ; elle ne recompte pas les
issues ou les revues dans une logique parallèle. Le statut interprété est déjà
présent dans le modèle de vue reçu par le template. Le même statut apparaît sur
la page du bilan d'une génération.

« Non corrigé » signifie qu'aucune copie ne possède de correction courante
finalisée. Cela couvre l'absence d'import réussi, un import seulement en cours
ou échoué, et un lot qui ne produit que des copies non détectées, incomplètes,
en erreur ou à vérifier. Le détail précise les catégories disponibles ou
« Aucune copie finalisée ».

« Correction partielle » signifie qu'au moins une copie possède un résultat
`corrected` sans revue en attente, mais que le nombre de copies finalisées est
inférieur au total courant. Le détail distingue les copies corrigées, à
vérifier, à contrôler et sans correction finale.

« Corrigé » signifie que le bilan contient au moins une copie et que chaque
copie attendue possède un résultat `corrected`, une note exploitable et aucune
revue pending. Autrement dit, `Corrected == Total` ; il ne reste alors ni
`not_seen`, ni `incomplete`, ni `error`, ni revue pending.

Les cartes gardent les actions P1 « Voir les résultats » et « Ajouter les
copies manquantes ». Le sélecteur d'import inclut aussi le statut pédagogique.
La table reste responsive et les noms longs peuvent revenir à la ligne.

## Cas particuliers

### Absents et copies non déposées

Le schéma ne possède aucun état « absent » ou « copie volontairement non
déposée ». Ces situations sont actuellement indiscernables d'un `not_seen`,
qui signifie seulement qu'aucune copie exploitable n'a été détectée pour cet
élève dans les imports réussis. Elles restent donc « sans correction finale »
et empêchent le statut « Corrigé ». Les assimiler automatiquement à une
évaluation terminée masquerait de vraies copies manquantes.

### Rattrapages

Le classement courant conserve une ancienne correction lorsqu'un import plus
récent contient `not_seen`, puis ajoute les nouvelles corrections. Le statut et
les compteurs progressent ainsi de « Non corrigé » à « Correction partielle »,
puis à « Corrigé », sans perdre les lots précédents.

### Revues pending

Une copie techniquement `corrected` avec au moins une décision humaine en attente
n'est pas finalisée. Elle est comptée « à vérifier », sa note reste masquée et
elle empêche « Corrigé ». Si aucune autre copie n'est finalisée, l'état global
reste « Non corrigé » ; sinon il est « Correction partielle ».

### Jobs failed/running

Les résultats des jobs en cours ou échoués sont exclus par la requête du bilan
courant et ne peuvent donc améliorer ni dégrader le statut pédagogique. Leur
état technique et leurs actions restent visibles dans l'historique des imports.

### Copies incomplètes et erreurs

`incomplete` et `error` sont regroupés comme copies « à contrôler ». Elles ne
sont jamais des corrections finalisées et empêchent donc « Corrigé ».

## Tests

- Examens : absence de génération affichée comme « Copies non générées », sans
  « Brouillon » ; génération `running` et génération disponible conservées.
- Helper central : aucun résultat, résultats uniquement non finalisés, premier
  lot incomplet, revue pending et cohorte entièrement corrigée.
- Intégration cumulative : trois corrections sur cinq donnent « Correction
  partielle » ; le rattrapage à cinq sur cinq donne « Corrigé » ; une nouvelle
  revue pending redonne « Correction partielle » ; sa résolution rétablit
  « Corrigé ».
- La même intégration ajoute ensuite des résultats artificiels issus de jobs
  `failed` et `running` et vérifie que le statut reste « Corrigé ».
- La page Correction vérifie le détail `3 corrigées · 2 sans correction finale`,
  ainsi qu'une autre génération sans résultat affichée « Non corrigé ».

Validation technique :

- `go test ./...` : succès ;
- `go test ./internal/handlers/exams ./internal/handlers/marking ./internal/templates/data -count=1` : succès ;
- `git diff --check` : succès.

## Points ouverts

La seule ambiguïté métier restante est l'absence volontaire. Pour permettre à
une évaluation d'être déclarée terminée malgré un élève absent, il faudra un état
explicite validé par l'enseignant, distinct de `not_seen`. Ce besoin implique une
évolution métier et de persistance ; il n'est pas simulé par ce correctif UX.
