# Roadmap P3 — analyse pédagogique

## 1. Analyse initiale de la chaîne de données

Le chemin audité est le suivant :

```text
questions / alt_questions
  → qcm_questions (famille logique et position dans le QCM)
  → exams
  → exams_generated
  → student_exam
  → student_exam_content (snapshot individualisé immuable)
  → marking_jobs (import lié à la génération)
  → marking_copy_results
  → marking_question_results (index dans le snapshot)
  → marking_answer_detections / marking_answer_reviews
  → lecture cumulative de l'état courant de la génération
```

`qcm_questions` référence la question principale, qui constitue la famille
logique. Lors de la génération, `BuildQuestionCtx` choisit aléatoirement la
principale ou une ligne `alt_questions`, puis mélange les questions et les
réponses. Le snapshot `student_exam_content` contient exactement ce qui a été
présenté : énoncé, image, réponses et états attendus, classifications, barème,
identité de la famille, ordre et coordonnées propres à la copie.

La correction ne relit pas la banque courante. Un QR identifie le
`student_exam`; le job est déjà lié à son `exams_generated`. Chaque résultat de
question conserve son `question_index`, qui est résolu exclusivement dans le
snapshot de cette copie. Les décisions humaines recalculent transactionnellement
le résultat de question et la note de la copie.

La lecture `ListCurrentExamResultsForGeneration` fournit la base fiable des
statistiques :

- elle est limitée à une génération et à son propriétaire ;
- elle ignore les jobs dont `status` ou `status_pdf` n'est pas `success` ;
- elle retient, par `student_exam`, la correction réussie la plus récente ;
- un `not_seen`, `incomplete` ou `error` ultérieur ne détruit pas une correction
  déjà acquise ;
- une recorréction réussie plus récente remplace la précédente ;
- un rattrapage apporte naturellement les nouvelles copies corrigées ;
- une copie avec revue non résolue reste visible comme « à vérifier », mais sa
  note et ses détails sont exclus des statistiques définitives.

Les notes, snapshots, détails par question et compteurs de revue viennent de la
même lecture. Le calcul pédagogique ne sélectionne donc pas séparément un job
ou une ancienne version du résultat.

## 2. Limites découvertes

Le snapshot historique portait déjà `tags.main_question_id`, suffisant pour
identifier la famille logique, mais pas la ligne principale/alternative tirée.
L'empreinte de contenu existante distinguait la plupart des variantes, sans
pouvoir :

- distinguer deux variantes différentes mais strictement identiques ;
- rattacher durablement le résultat à la ligne de banque source ;
- suivre la même variante si son contenu avait évolué entre deux snapshots.

Les snapshots très anciens peuvent également avoir un détail absent ou
incohérent. Leur note finale reste exploitable pour le taux global, mais ils ne
sont pas intégrés silencieusement aux agrégations par question.

## 3. Décision sur les snapshots et variantes

Les nouveaux snapshots ajoutent deux champs optionnels dans `Tags` :

- `variant_type` : `mainQuestion` ou `altQuestion` ;
- `variant_id` : identifiant de la ligne effectivement sélectionnée.

Le couple est nécessaire parce que les IDs des tables `questions` et
`alt_questions` vivent dans deux espaces distincts. `main_question_id` reste
l'identité de la famille logique.

Cette évolution est additive. Aucun snapshot historique n'est réécrit et
aucune identité n'est inventée à sa lecture. Pour un ancien snapshot :

- la famille utilise `main_question_id` lorsqu'il est présent ;
- la variante utilise une empreinte SHA-256 conservatrice de l'énoncé, image,
  réponses attendues triées, barème et classifications ;
- ordre des réponses, symboles de cases et géométrie sont exclus de l'empreinte ;
- sans ID de famille, seuls des contenus strictement équivalents sont regroupés.

Les statistiques principales regroupent toutes les variantes d'une question
logique. Le détail par variante reste disponible sous cette question.

## 4. Migrations

Aucune migration SQL n'est nécessaire. Le snapshot est déjà un document JSON
versionné de façon additive et immuable ; ajouter des colonnes parallèles aurait
dupliqué la source de vérité et imposé un backfill impossible à garantir pour
les anciennes variantes.

Le replay complet des migrations 0001 à 0047 reste vert. Un test de compatibilité
vérifie qu'un ancien JSON se lit avec des valeurs nulles pour la variante et
qu'une sérialisation ne lui invente pas ces champs.

## 5. Statistiques implémentées

Pour l'état courant d'une génération :

- nombre de copies corrigées prises en compte ;
- nombre de résultats courants exclus car non finalisés ;
- couverture des détails historiques utilisables ;
- taux de réussite global = somme des points obtenus / somme des points
  possibles ;
- moyenne, médiane et écart-type, séparés si plusieurs barèmes coexistent ;
- taux par question logique, crédit partiel inclus ;
- nombre exact de copies utilisées et nombre de réussites complètes par
  question ;
- détail par variante effectivement présentée ;
- agrégation par thème, compétence et couple thème-compétence lorsqu'ils sont
  renseignés.

Le modèle conserve à la fois le pourcentage numérique (`SuccessPercent`) pour
les futurs tris/requêtes P5 et sa représentation française pour l'affichage.
Une question non classée reste présente dans les statistiques par question.

Les niveaux sont centralisés et testés aux bornes :

- rouge si le résultat est strictement inférieur à 40 % ;
- jaune de 40 % à 60 % inclus ;
- vert strictement au-dessus de 60 %.

## 6. Interface ajoutée

La page professeur existante du bilan d'une génération affiche maintenant une
carte « Analyse pédagogique » avant l'historique des imports :

- taux global, niveau et effectif ;
- explication des exclusions et couverture historique ;
- moyenne/médiane/écart-type ;
- tableau lisible des questions avec pourcentage et copies utilisées ;
- sous-lignes de variantes lorsque plusieurs versions ont été présentées ;
- listes par thème et compétence.

La couleur est toujours accompagnée de l'emoji, du libellé et du pourcentage.
Le PDF cumulé reprend le taux global, les questions logiques, les variantes et
les thèmes, sans changer la sélection des résultats.

## 7. Tests ajoutés ou étendus

La couverture P3 comprend :

- une question et plusieurs questions ;
- principales et variantes, réponses mélangées et ordre de questions différent ;
- deux variantes de même contenu qui restent distinctes grâce à leur ID ;
- même variante dont le libellé historique diffère, correctement regroupée ;
- ancien snapshot sans identité de variante, avec fallback conservateur ;
- correction complète, correction partielle et détails historiques incomplets ;
- recorréction réussie ;
- rattrapage ajouté après le premier lot ;
- jobs failed/running et états obsolètes ;
- revue pending, confirmation puis modification humaine ;
- ownership entre utilisateurs et générations ;
- bornes exactes 40 % et 60 % ;
- absence de thème et de compétence ;
- rendu HTML de la carte pédagogique ;
- génération et extraction du PDF réel ;
- sérialisation additive et lecture des anciens snapshots ;
- pipeline réel de génération confirmant l'identité principale/alternative
  stockée dans le snapshot.

## 8. Commandes de validation exécutées

```bash
go test ./...
go test -count=1 ./...
go test -race ./...
git diff --check
GOCACHE=/tmp/lazymarking-go-cache ./scripts/check.sh
```

Le cache Go a été placé dans `/tmp` dans cet environnement. Les campagnes qui
compilent des PDF ont été exécutées hors du bac à sable afin que Typst installé
via Snap puisse accéder à son répertoire `/run/user/1000`.

## 9. Résultats des tests

- `go test ./...` : succès ;
- `go test -count=1 ./...` : succès ;
- `go test -race ./...` : succès ;
- `scripts/check.sh` : succès, y compris `go mod verify`, replay Goose 0001→0047,
  `gofmt`, `go vet`, tests, build et `git diff --check` ;
- tests PDF Typst/pdftotext : exécutés et réussis ;
- `git diff --check` final : succès.

Les tests privés dépendant de variables `LAZYMARKING_TEST_*` gardent leur
comportement opt-in et ne constituent pas une réexécution de scans terrain.

## 10. Limites restantes

- Les anciens snapshots sans `variant_type`/`variant_id` ne peuvent pas être
  reliés avec certitude à une variante de banque ; leur famille et leur contenu
  historique restent toutefois exploitables.
- Une variante supprimée après génération conserve son identité historique,
  mais P5 devra décider s'il réutilise la famille principale, ignore la variante
  supprimée ou copie son contenu figé dans un futur deck.
- Le bilan travaille au niveau de la génération d'examen, ce qui correspond au
  parcours actuel. Une analyse longitudinale entre examens n'est pas P3.
- Les copies pending sont exclues en totalité jusqu'à validation humaine ; il
  n'existe pas de mélange entre questions provisoires et définitives d'une même
  copie.

## 11. Préparation de P4 et P5

La donnée pédagogique est prête pour P4 : une copie courante possède déjà son
élève, son snapshot, sa note, ses résultats de questions et ses décisions
humaines. P4 doit encore décider le contrat d'accès public : token opaque,
non-énumérable et révocable, durée de vie, cache et périmètre des documents.
Cette décision de sécurité doit être prise avant toute restitution sans compte.

La donnée est également prête pour P5 : les questions logiques et variantes ont
des taux numériques, des effectifs et, pour les nouveaux snapshots, une
référence de banque stable. Une future sélection « les moins réussies » peut
trier directement les familles ou variantes sans analyser des libellés ni
recalculer les réponses. P5 devra seulement fixer la politique pour les anciens
snapshots sans ID de variante et pour les éléments de banque supprimés.

Aucun compte élève, lien public, QR de restitution, deck, entraînement ou
fonctionnalité LMS n'a été ajouté dans P3.
