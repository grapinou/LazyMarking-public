package imagestorage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/grapinou/LazyMarking/internal/config"
	"github.com/grapinou/LazyMarking/internal/db"
	"github.com/grapinou/LazyMarking/internal/filesafety"
)

type ReferenceType string

const (
	MainImageReference      ReferenceType = "main_image"
	VariantImageReference   ReferenceType = "variant_image"
	MainAndVariantReference ReferenceType = "main_and_variant_image"
)

type UnsafeSource string

const (
	DatabaseSource   UnsafeSource = "database"
	FilesystemSource UnsafeSource = "filesystem"
)

type UnsafeKind string

const (
	UnsafeName       UnsafeKind = "unsafe_name"
	SymlinkEntry     UnsafeKind = "symlink"
	DirectoryEntry   UnsafeKind = "directory"
	SpecialFileEntry UnsafeKind = "special_file"
)

type MissingImage struct {
	Name          string
	ReferenceType ReferenceType
}

type UnsafeEntry struct {
	Name   string
	Source UnsafeSource
	Kind   UnsafeKind
}

type Consistency struct {
	Orphans []string
	Missing []MissingImage
	Unsafe  []UnsafeEntry
}

type referenceKinds uint8

const (
	mainReference referenceKinds = 1 << iota
	variantReference
)

func Scan(ctx context.Context, queries *db.Queries) (Consistency, error) {
	mainNames, err := queries.ListAllImageNames(ctx)
	if err != nil {
		return Consistency{}, fmt.Errorf("list main image references: %w", err)
	}
	variantNames, err := queries.ListAllAltImageNames(ctx)
	if err != nil {
		return Consistency{}, fmt.Errorf("list variant image references: %w", err)
	}

	filesystemNames, filesystemUnsafe, err := readImageDirectory(
		config.ImageSavePath,
		len(mainNames)+len(variantNames) > 0,
	)
	if err != nil {
		return Consistency{}, err
	}

	return compare(mainNames, variantNames, filesystemNames, filesystemUnsafe), nil
}

func readImageDirectory(path string, hasDatabaseReferences bool) ([]string, []UnsafeEntry, error) {
	if err := filesafety.ValidateDirectoryTree(path); err != nil {
		if errors.Is(err, os.ErrNotExist) && !hasDatabaseReferences {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("validate image directory: %w", err)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && !hasDatabaseReferences {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read image directory: %w", err)
	}

	var names []string
	var unsafe []UnsafeEntry
	for _, entry := range entries {
		name := entry.Name()
		if !filesafety.IsSafePathComponent(name) {
			unsafe = append(unsafe, UnsafeEntry{Name: name, Source: FilesystemSource, Kind: UnsafeName})
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			unsafe = append(unsafe, UnsafeEntry{Name: name, Source: FilesystemSource, Kind: SymlinkEntry})
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, nil, fmt.Errorf("inspect image directory entry %q: %w", name, err)
		}
		switch {
		case info.Mode().IsRegular():
			names = append(names, name)
		case info.IsDir():
			unsafe = append(unsafe, UnsafeEntry{Name: name, Source: FilesystemSource, Kind: DirectoryEntry})
		default:
			unsafe = append(unsafe, UnsafeEntry{Name: name, Source: FilesystemSource, Kind: SpecialFileEntry})
		}
	}
	return names, unsafe, nil
}

func compare(mainNames, variantNames, filesystemNames []string, filesystemUnsafe []UnsafeEntry) Consistency {
	references := make(map[string]referenceKinds)
	unsafeByKey := make(map[string]UnsafeEntry)

	addReferences := func(names []string, kind referenceKinds) {
		for _, name := range names {
			if !filesafety.IsSafePathComponent(name) {
				entry := UnsafeEntry{Name: name, Source: DatabaseSource, Kind: UnsafeName}
				unsafeByKey[unsafeKey(entry)] = entry
				continue
			}
			references[name] |= kind
		}
	}
	addReferences(mainNames, mainReference)
	addReferences(variantNames, variantReference)

	files := make(map[string]struct{}, len(filesystemNames))
	for _, name := range filesystemNames {
		files[name] = struct{}{}
	}

	result := Consistency{}
	for name := range files {
		if _, referenced := references[name]; !referenced {
			result.Orphans = append(result.Orphans, name)
		}
	}
	for name, kinds := range references {
		if _, present := files[name]; !present {
			result.Missing = append(result.Missing, MissingImage{
				Name:          name,
				ReferenceType: referenceType(kinds),
			})
		}
	}
	for _, entry := range filesystemUnsafe {
		unsafeByKey[unsafeKey(entry)] = entry
	}
	for _, entry := range unsafeByKey {
		result.Unsafe = append(result.Unsafe, entry)
	}

	sort.Strings(result.Orphans)
	sort.Slice(result.Missing, func(i, j int) bool {
		if result.Missing[i].Name == result.Missing[j].Name {
			return result.Missing[i].ReferenceType < result.Missing[j].ReferenceType
		}
		return result.Missing[i].Name < result.Missing[j].Name
	})
	sort.Slice(result.Unsafe, func(i, j int) bool {
		if result.Unsafe[i].Name != result.Unsafe[j].Name {
			return result.Unsafe[i].Name < result.Unsafe[j].Name
		}
		if result.Unsafe[i].Source != result.Unsafe[j].Source {
			return result.Unsafe[i].Source < result.Unsafe[j].Source
		}
		return result.Unsafe[i].Kind < result.Unsafe[j].Kind
	})
	return result
}

func referenceType(kinds referenceKinds) ReferenceType {
	switch kinds {
	case mainReference:
		return MainImageReference
	case variantReference:
		return VariantImageReference
	default:
		return MainAndVariantReference
	}
}

func unsafeKey(entry UnsafeEntry) string {
	return string(entry.Source) + "\x00" + string(entry.Kind) + "\x00" + entry.Name
}
