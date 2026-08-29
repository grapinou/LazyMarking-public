package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	appdb "github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/imagestorage"
)

var maintenanceNow = time.Now

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "images" {
		fmt.Fprintln(stderr, "Usage: maintenance images [--execute] [--grace duration]")
		return 1
	}

	conn, err := appdb.InitDB(config.DatabasePath)
	if err != nil {
		fmt.Fprintf(stderr, "Erreur d’ouverture de la base : %v\n", err)
		return 1
	}
	defer conn.Close()

	return runImages(ctx, args[1:], stdout, stderr, appdb.New(conn))
}

func runImages(ctx context.Context, args []string, stdout, stderr io.Writer, queries *appdb.Queries) int {
	flags := flag.NewFlagSet("maintenance images", flag.ContinueOnError)
	flags.SetOutput(stderr)
	execute := flags.Bool("execute", false, "supprimer les vieux fichiers orphelins")
	grace := flags.Duration("grace", imagestorage.DefaultOrphanGracePeriod, "délai de grâce des orphelins")
	if err := flags.Parse(args); err != nil {
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "Argument inattendu : %s\n", flags.Arg(0))
		return 1
	}
	if *grace < 0 {
		fmt.Fprintln(stderr, "Le délai de grâce ne peut pas être négatif.")
		return 1
	}

	consistency, err := imagestorage.Scan(ctx, queries)
	if err != nil {
		fmt.Fprintf(stderr, "Audit du stockage impossible : %v\n", err)
		return 1
	}
	printConsistency(stdout, consistency)

	options := imagestorage.PurgeOptions{
		Execute:     *execute,
		GracePeriod: *grace,
		Now:         maintenanceNow(),
	}
	if *execute {
		fmt.Fprintf(stdout, "\nMode : EXÉCUTION (délai de grâce %s)\n", grace.String())
	} else {
		fmt.Fprintf(stdout, "\nMode : DRY RUN (délai de grâce %s)\n", grace.String())
	}

	result, err := imagestorage.PurgeOrphans(ctx, queries, options)
	if err != nil {
		fmt.Fprintf(stderr, "Maintenance des orphelins impossible : %v\n", err)
		return 1
	}
	printPurgeResult(stdout, result)
	if len(result.Failed) != 0 {
		return 1
	}
	return 0
}

func printConsistency(output io.Writer, consistency imagestorage.Consistency) {
	fmt.Fprintf(output, "Résumé : %d orphelin(s), %d référence(s) DB manquante(s), %d entrée(s) unsafe.\n",
		len(consistency.Orphans), len(consistency.Missing), len(consistency.Unsafe))
	printNames(output, "Orphans", consistency.Orphans)

	fmt.Fprintln(output, "Missing (signalement uniquement, aucune ligne DB modifiée) :")
	if len(consistency.Missing) == 0 {
		fmt.Fprintln(output, "  aucun")
	}
	for _, missing := range consistency.Missing {
		fmt.Fprintf(output, "  - %s (%s)\n", missing.Name, missing.ReferenceType)
	}

	fmt.Fprintln(output, "Unsafe (signalement uniquement) :")
	if len(consistency.Unsafe) == 0 {
		fmt.Fprintln(output, "  aucun")
	}
	for _, unsafe := range consistency.Unsafe {
		fmt.Fprintf(output, "  - %s (source=%s, nature=%s)\n", unsafe.Name, unsafe.Source, unsafe.Kind)
	}
}

func printPurgeResult(output io.Writer, result imagestorage.PurgeResult) {
	printNames(output, "Candidats orphelins", result.Candidates)
	printNames(output, "Supprimés", result.Deleted)
	printNames(output, "Ignorés car récents", result.SkippedRecent)
	printNames(output, "Ignorés car redevenus référencés", result.SkippedReferenced)
	fmt.Fprintln(output, "Échecs :")
	if len(result.Failed) == 0 {
		fmt.Fprintln(output, "  aucun")
	}
	for _, failure := range result.Failed {
		fmt.Fprintf(output, "  - %s : %v\n", failure.Name, failure.Err)
	}
}

func printNames(output io.Writer, title string, names []string) {
	fmt.Fprintf(output, "%s :\n", title)
	if len(names) == 0 {
		fmt.Fprintln(output, "  aucun")
		return
	}
	for _, name := range names {
		fmt.Fprintf(output, "  - %s\n", name)
	}
}
