package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
)

// DiscoveredFile represents a file matched to a single type definition.
type DiscoveredFile struct {
	Path         string            // Repo-relative path using forward slashes
	TypeName     string            // Name of the matched type
	TypeDef      *config.TypeDef   // Pointer to the type definition
	PathCaptures map[string]string // Named captures from the include regex
}

// hiddenOrIgnored returns true for directories that should be skipped during walk.
var ignoreDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
}

// Discover walks the rootDir and matches files against the configured types.
// Returns discovered files and any errors (multi-type match, subdirectory .datacur8, etc.)
func Discover(rootDir string, types []config.TypeDef) ([]DiscoveredFile, []error) {
	var errs []error

	// Pre-compile include and exclude regexes per type.
	type compiledType struct {
		def      *config.TypeDef
		includes []*regexp.Regexp
		excludes []*regexp.Regexp
	}

	compiled := make([]compiledType, len(types))
	for i := range types {
		ct := compiledType{def: &types[i]}
		for _, pat := range types[i].Match.Include {
			re, err := regexp.Compile(pat)
			if err != nil {
				errs = append(errs, fmt.Errorf("type %q: invalid include pattern %q: %w", types[i].Name, pat, err))
				continue
			}
			ct.includes = append(ct.includes, re)
		}
		for _, pat := range types[i].Match.Exclude {
			re, err := regexp.Compile(pat)
			if err != nil {
				errs = append(errs, fmt.Errorf("type %q: invalid exclude pattern %q: %w", types[i].Name, pat, err))
				continue
			}
			ct.excludes = append(ct.excludes, re)
		}
		compiled[i] = ct
	}

	if len(errs) > 0 {
		return nil, errs
	}

	// Collect output paths so we can skip them during matching.
	outputPaths := make(map[string]bool)
	for i := range types {
		if types[i].Output != nil && types[i].Output.Path != "" {
			normalized := filepath.ToSlash(types[i].Output.Path)
			outputPaths[normalized] = true
		}
	}

	var discovered []DiscoveredFile

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := info.Name()

		if info.IsDir() {
			// Skip hidden directories and common ignore dirs.
			if name != "." && (strings.HasPrefix(name, ".") || ignoreDirs[name]) {
				return filepath.SkipDir
			}
			return nil
		}

		// Compute repo-relative path with forward slashes.
		relPath, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return relErr
		}
		relPath = filepath.ToSlash(relPath)

		// Check for .datacur8 files in subdirectories.
		if name == ".datacur8" {
			dir := filepath.ToSlash(filepath.Dir(relPath))
			if dir != "." {
				errs = append(errs, fmt.Errorf("found .datacur8 in subdirectory %q; only root .datacur8 is allowed", dir))
			}
			return nil
		}

		// Skip output files.
		if outputPaths[relPath] {
			return nil
		}

		// Match against each type.
		type matchInfo struct {
			typeName string
			typeDef  *config.TypeDef
			captures map[string]string
		}

		var matches []matchInfo

		for _, ct := range compiled {
			captures, matched := matchType(relPath, ct.includes, ct.excludes)
			if matched {
				// Add built-in path captures.
				captures["path.file"] = fileNameWithoutExt(name)
				captures["path.ext"] = normalizeExt(filepath.Ext(name))
				captures["path.parent"] = parentFolder(relPath)

				matches = append(matches, matchInfo{
					typeName: ct.def.Name,
					typeDef:  ct.def,
					captures: captures,
				})
			}
		}

		if len(matches) > 1 {
			names := make([]string, len(matches))
			for i, m := range matches {
				names[i] = m.typeName
			}
			errs = append(errs, fmt.Errorf("file %q matches multiple types: %s", relPath, strings.Join(names, ", ")))
			return nil
		}

		if len(matches) == 1 {
			m := matches[0]
			discovered = append(discovered, DiscoveredFile{
				Path:         relPath,
				TypeName:     m.typeName,
				TypeDef:      m.typeDef,
				PathCaptures: m.captures,
			})
		}

		return nil
	})

	if err != nil {
		errs = append(errs, fmt.Errorf("walking directory: %w", err))
	}

	// Sort by path for deterministic ordering.
	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].Path < discovered[j].Path
	})

	if len(errs) > 0 {
		return discovered, errs
	}

	return discovered, nil
}

// matchType checks if relPath matches any include pattern and no exclude pattern.
// Returns named captures from the first matching include pattern.
func matchType(relPath string, includes, excludes []*regexp.Regexp) (map[string]string, bool) {
	// Check excludes first.
	for _, ex := range excludes {
		if ex.MatchString(relPath) {
			return nil, false
		}
	}

	// Check includes.
	for _, inc := range includes {
		match := inc.FindStringSubmatch(relPath)
		if match != nil {
			captures := make(map[string]string)
			for i, name := range inc.SubexpNames() {
				if name != "" && i < len(match) {
					captures["path."+name] = match[i]
				}
			}
			return captures, true
		}
	}

	return nil, false
}

// fileNameWithoutExt returns the file name with its extension removed.
func fileNameWithoutExt(name string) string {
	ext := filepath.Ext(name)
	return strings.TrimSuffix(name, ext)
}

// normalizeExt returns the lowercase extension without the leading dot.
// Normalizes "yml" to "yaml".
func normalizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if ext == "yml" {
		return "yaml"
	}
	return ext
}

// parentFolder returns the name of the parent directory from a forward-slash relative path.
func parentFolder(relPath string) string {
	dir := filepath.Dir(relPath)
	dir = filepath.ToSlash(dir)
	if dir == "." {
		return ""
	}
	return filepath.Base(dir)
}
