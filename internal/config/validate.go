package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/UnitVectorY-Labs/datacur8/internal/selector"
)

var (
	semverRe      = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	typeNameRe    = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	pathSelectorRe = regexp.MustCompile(`^path\.(file|parent|ext|[a-zA-Z_][a-zA-Z0-9_]*)$`)
)

// Validate checks cfg for structural and semantic errors.
// cliVersion is the running binary version (e.g. "1.0.0"); pass "dev" or ""
// to skip version comparison.
func Validate(cfg *Config, cliVersion string) (warnings []string, errs []error) {
	// 1. Version – must be valid semver
	cfgParts := semverRe.FindStringSubmatch(cfg.Version)
	if cfgParts == nil {
		errs = append(errs, fmt.Errorf("version %q is not valid semver (expected major.minor.patch)", cfg.Version))
	}

	// 2-3. Version comparison
	if cfgParts != nil {
		if cliVersion == "" || cliVersion == "dev" {
			warnings = append(warnings, "CLI version is dev/empty; skipping config version compatibility check")
		} else {
			cliParts := semverRe.FindStringSubmatch(cliVersion)
			if cliParts == nil {
				warnings = append(warnings, fmt.Sprintf("CLI version %q is not semver; skipping version comparison", cliVersion))
			} else {
				if cliParts[1] != cfgParts[1] {
					errs = append(errs, fmt.Errorf("major version mismatch: config requires %s.x.x but CLI is %s", cfgParts[1], cliVersion))
				} else if compareSemver(cliParts, cfgParts) < 0 {
					errs = append(errs, fmt.Errorf("CLI version %s is older than config version %s", cliVersion, cfg.Version))
				}
			}
		}
	}

	// 4. strict_mode
	switch cfg.StrictMode {
	case "", "DISABLED", "ENABLED", "FORCE":
	default:
		errs = append(errs, fmt.Errorf("strict_mode %q is invalid; must be DISABLED, ENABLED, or FORCE", cfg.StrictMode))
	}

	// 5. reporting.mode
	if cfg.Reporting != nil {
		switch cfg.Reporting.Mode {
		case "", "text", "json", "yaml":
		default:
			errs = append(errs, fmt.Errorf("reporting.mode %q is invalid; must be text, json, or yaml", cfg.Reporting.Mode))
		}
	}

	// 6. types
	typeNames := make(map[string]bool, len(cfg.Types))
	outputPaths := make(map[string]string) // path -> type name

	for i, t := range cfg.Types {
		prefix := fmt.Sprintf("types[%d](%s)", i, t.Name)

		// unique name
		if typeNames[t.Name] {
			errs = append(errs, fmt.Errorf("%s: duplicate type name %q", prefix, t.Name))
		}
		typeNames[t.Name] = true

		// name format
		if !typeNameRe.MatchString(t.Name) {
			errs = append(errs, fmt.Errorf("%s: type name must match %s", prefix, typeNameRe.String()))
		}

		// input format
		switch t.Input {
		case "json", "yaml", "csv":
		default:
			errs = append(errs, fmt.Errorf("%s: input %q must be json, yaml, or csv", prefix, t.Input))
		}

		// match.include
		if len(t.Match.Include) == 0 {
			errs = append(errs, fmt.Errorf("%s: match.include must have at least 1 pattern", prefix))
		}
		for j, pat := range t.Match.Include {
			if _, err := regexp.Compile(pat); err != nil {
				errs = append(errs, fmt.Errorf("%s: match.include[%d] invalid regex: %v", prefix, j, err))
			}
		}
		for j, pat := range t.Match.Exclude {
			if _, err := regexp.Compile(pat); err != nil {
				errs = append(errs, fmt.Errorf("%s: match.exclude[%d] invalid regex: %v", prefix, j, err))
			}
		}

		// schema
		if t.Schema == nil {
			errs = append(errs, fmt.Errorf("%s: schema is required", prefix))
		} else if st, ok := t.Schema["type"]; !ok || st != "object" {
			errs = append(errs, fmt.Errorf("%s: schema.type must be \"object\"", prefix))
		}

		// csv
		if t.Input == "csv" && t.CSV == nil {
			errs = append(errs, fmt.Errorf("%s: csv config is required when input is csv", prefix))
		}
		if t.CSV != nil && len([]rune(t.CSV.Delimiter)) != 1 {
			errs = append(errs, fmt.Errorf("%s: csv.delimiter must be exactly 1 character", prefix))
		}

		// output
		if t.Output != nil {
			switch t.Output.Format {
			case "json", "yaml", "jsonl":
			default:
				errs = append(errs, fmt.Errorf("%s: output.format %q must be json, yaml, or jsonl", prefix, t.Output.Format))
			}
			if prev, exists := outputPaths[t.Output.Path]; exists {
				errs = append(errs, fmt.Errorf("%s: output.path %q conflicts with type %q", prefix, t.Output.Path, prev))
			}
			outputPaths[t.Output.Path] = t.Name
		}

		// constraints
		for ci, con := range t.Constraints {
			cprefix := fmt.Sprintf("%s.constraints[%d]", prefix, ci)
			switch con.Type {
			case "unique":
				errs = append(errs, validateSelector(cprefix, "key", con.Key)...)
				switch con.Scope {
				case "", "item", "type":
				default:
					errs = append(errs, fmt.Errorf("%s: scope %q must be item or type", cprefix, con.Scope))
				}

			case "foreign_key":
				errs = append(errs, validateSelector(cprefix, "key", con.Key)...)
				if con.References == nil {
					errs = append(errs, fmt.Errorf("%s: references is required for foreign_key", cprefix))
				} else {
					if con.References.Type == "" {
						errs = append(errs, fmt.Errorf("%s: references.type is required", cprefix))
					} else if !typeNames[con.References.Type] {
						// referenced type might be defined later; collect for deferred check
					}
					errs = append(errs, validateSelector(cprefix, "references.key", con.References.Key)...)
				}

			case "path_equals_attr":
				if !pathSelectorRe.MatchString(con.PathSelector) {
					errs = append(errs, fmt.Errorf("%s: path_selector %q is invalid", cprefix, con.PathSelector))
				}
				if con.References == nil {
					errs = append(errs, fmt.Errorf("%s: references is required for path_equals_attr", cprefix))
				} else {
					errs = append(errs, validateSelector(cprefix, "references.key", con.References.Key)...)
				}

				// capture group validation
				captureName := extractCaptureName(con.PathSelector)
				if captureName != "" {
					groupName := captureName
					for pi, pat := range t.Match.Include {
						re, err := regexp.Compile(pat)
						if err != nil {
							continue // already reported
						}
						if !hasNamedGroup(re, groupName) {
							errs = append(errs, fmt.Errorf(
								"%s: path_selector uses capture %q but match.include[%d] does not define named group (?P<%s>...)",
								cprefix, captureName, pi, groupName))
						}
					}
				}

			default:
				errs = append(errs, fmt.Errorf("%s: unknown constraint type %q", cprefix, con.Type))
			}
		}
	}

	// deferred check: foreign_key references must point to known type names
	for i, t := range cfg.Types {
		prefix := fmt.Sprintf("types[%d](%s)", i, t.Name)
		for ci, con := range t.Constraints {
			if con.Type == "foreign_key" && con.References != nil && con.References.Type != "" {
				if !typeNames[con.References.Type] {
					errs = append(errs, fmt.Errorf("%s.constraints[%d]: references.type %q does not match any defined type", prefix, ci, con.References.Type))
				}
			}
		}
	}

	return warnings, errs
}

func validateSelector(prefix, field, value string) []error {
	if value == "" {
		return []error{fmt.Errorf("%s: %s is required", prefix, field)}
	}
	if _, err := selector.Parse(value); err != nil {
		return []error{fmt.Errorf("%s: %s %q is not a valid selector: %v", prefix, field, value, err)}
	}
	return nil
}

// compareSemver compares two parsed semver match groups [full, major, minor, patch].
// Returns -1, 0, or 1.
func compareSemver(a, b []string) int {
	for i := 1; i <= 3; i++ {
		ai, _ := strconv.Atoi(a[i])
		bi, _ := strconv.Atoi(b[i])
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

// extractCaptureName returns the capture name from a path_selector like "path.<name>"
// where name is not one of the built-in segments (file, parent, ext).
func extractCaptureName(ps string) string {
	if !strings.HasPrefix(ps, "path.") {
		return ""
	}
	name := ps[5:]
	switch name {
	case "file", "parent", "ext":
		return ""
	}
	return name
}

// hasNamedGroup returns true if the compiled regex defines a named capture group
// with the given name.
func hasNamedGroup(re *regexp.Regexp, name string) bool {
	for _, n := range re.SubexpNames() {
		if n == name {
			return true
		}
	}
	return false
}
