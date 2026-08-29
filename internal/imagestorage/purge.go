package imagestorage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/filesafety"
)

const DefaultOrphanGracePeriod = 24 * time.Hour

type PurgeOptions struct {
	Execute     bool
	GracePeriod time.Duration
	Now         time.Time
}

type PurgeFailure struct {
	Name string
	Err  error
}

type PurgeResult struct {
	Candidates        []string
	Deleted           []string
	SkippedRecent     []string
	SkippedReferenced []string
	Failed            []PurgeFailure
}

var scanForOrphanPurge = Scan
var removeOrphanFile = os.Remove

func DefaultPurgeOptions() PurgeOptions {
	return PurgeOptions{GracePeriod: DefaultOrphanGracePeriod}
}

func PurgeOrphans(ctx context.Context, queries *db.Queries, options PurgeOptions) (PurgeResult, error) {
	if options.GracePeriod < 0 {
		return PurgeResult{}, errors.New("orphan grace period cannot be negative")
	}
	if err := filesafety.ValidateDirectoryTree(config.ImageSavePath); err != nil {
		return PurgeResult{}, fmt.Errorf("validate image storage root before orphan purge: %w", err)
	}

	consistency, err := scanForOrphanPurge(ctx, queries)
	if err != nil {
		return PurgeResult{}, fmt.Errorf("scan image storage before orphan purge: %w", err)
	}

	result := PurgeResult{Candidates: append([]string(nil), consistency.Orphans...)}
	sort.Strings(result.Candidates)
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	cutoff := now.Add(-options.GracePeriod)

	var eligible []string
	for _, name := range result.Candidates {
		if !filesafety.IsSafePathComponent(name) {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: errors.New("unsafe orphan filename")})
			continue
		}
		info, err := inspectRegularOrphan(name)
		if err != nil {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: err})
			continue
		}
		if info.ModTime().After(cutoff) {
			result.SkippedRecent = append(result.SkippedRecent, name)
			continue
		}
		eligible = append(eligible, name)
	}

	var deletable []string
	for _, name := range eligible {
		referenced, err := queries.ImageNameIsReferenced(ctx, name)
		if err != nil {
			return PurgeResult{}, fmt.Errorf("recheck image reference %q: %w", name, err)
		}
		if referenced {
			result.SkippedReferenced = append(result.SkippedReferenced, name)
			continue
		}
		deletable = append(deletable, name)
	}

	if !options.Execute {
		sortPurgeResult(&result)
		return result, nil
	}

	for _, name := range deletable {
		if !filesafety.IsSafePathComponent(name) {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: errors.New("unsafe orphan filename before removal")})
			continue
		}
		info, err := inspectRegularOrphan(name)
		if err != nil {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: err})
			continue
		}
		if info.ModTime().After(cutoff) {
			result.SkippedRecent = append(result.SkippedRecent, name)
			continue
		}
		referenced, err := queries.ImageNameIsReferenced(ctx, name)
		if err != nil {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: fmt.Errorf("recheck image reference before removal: %w", err)})
			continue
		}
		if referenced {
			result.SkippedReferenced = append(result.SkippedReferenced, name)
			continue
		}
		if err := removeOrphanFile(filepath.Join(config.ImageSavePath, name)); err != nil {
			result.Failed = append(result.Failed, PurgeFailure{Name: name, Err: err})
			continue
		}
		result.Deleted = append(result.Deleted, name)
	}

	sortPurgeResult(&result)
	return result, nil
}

func inspectRegularOrphan(name string) (os.FileInfo, error) {
	path := filepath.Join(config.ImageSavePath, name)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect orphan %q: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("orphan %q is not a regular file", name)
	}
	return info, nil
}

func sortPurgeResult(result *PurgeResult) {
	sort.Strings(result.Candidates)
	sort.Strings(result.Deleted)
	sort.Strings(result.SkippedRecent)
	result.SkippedRecent = compactStrings(result.SkippedRecent)
	sort.Strings(result.SkippedReferenced)
	result.SkippedReferenced = compactStrings(result.SkippedReferenced)
	sort.Slice(result.Failed, func(i, j int) bool { return result.Failed[i].Name < result.Failed[j].Name })
}

func compactStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
