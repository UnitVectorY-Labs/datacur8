package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/UnitVectorY-Labs/datacur8/internal/config"
	"github.com/UnitVectorY-Labs/datacur8/internal/constraints"
	"github.com/UnitVectorY-Labs/datacur8/internal/discovery"
	"github.com/UnitVectorY-Labs/datacur8/internal/export"
	"github.com/UnitVectorY-Labs/datacur8/internal/schema"
	"github.com/UnitVectorY-Labs/datacur8/internal/tidy"
	"gopkg.in/yaml.v3"
)

// Exit codes
const (
	ExitOK             = 0
	ExitConfigInvalid  = 1
	ExitDataInvalid    = 2
	ExitExportFailure  = 3
	ExitTidyFailure    = 4
)

// reportEntry is a structured error/warning for JSON/YAML output.
type reportEntry struct {
	Level    string `json:"level" yaml:"level"`
	Type     string `json:"type,omitempty" yaml:"type,omitempty"`
	File     string `json:"file,omitempty" yaml:"file,omitempty"`
	Row      *int   `json:"row,omitempty" yaml:"row,omitempty"`
	Message  string `json:"message" yaml:"message"`
}

// RunValidate runs the validate command.
// configOnly: if true, only validate config, not data.
// format: output format (text, json, yaml) - from --format flag, overrides config.
// version: CLI version string.
// Returns exit code.
func RunValidate(configOnly bool, format string, version string) int {
	cfg, resolvedFormat, code := loadAndValidateConfig(format, version)
	if code != ExitOK {
		return code
	}

	if configOnly {
		return ExitOK
	}

	if len(cfg.Types) == 0 {
		fmt.Fprintln(os.Stderr, "no types configured")
		return ExitOK
	}

	rootDir, _ := os.Getwd()
	files, discoverErrs := discovery.Discover(rootDir, cfg.Types)
	if len(discoverErrs) > 0 {
		reportErrors(resolvedFormat, toReportEntries("error", "discovery", discoverErrs))
		return ExitConfigInvalid
	}

	items, parseEntries, schemaEntries := parseAndValidateFiles(files, cfg)

	constraintErrs := constraints.Evaluate(items, cfg.Types)
	constraintEntries := constraintErrorsToEntries(constraintErrs)

	allEntries := append(parseEntries, schemaEntries...)
	allEntries = append(allEntries, constraintEntries...)

	if len(allEntries) > 0 {
		reportErrors(resolvedFormat, allEntries)
		return ExitDataInvalid
	}

	return ExitOK
}

// RunExport runs the export command.
// version: CLI version string.
// Returns exit code.
func RunExport(version string) int {
	cfg, resolvedFormat, code := loadAndValidateConfig("", version)
	if code != ExitOK {
		return code
	}

	if len(cfg.Types) == 0 {
		fmt.Fprintln(os.Stderr, "no types configured")
		return ExitOK
	}

	rootDir, _ := os.Getwd()
	files, discoverErrs := discovery.Discover(rootDir, cfg.Types)
	if len(discoverErrs) > 0 {
		reportErrors(resolvedFormat, toReportEntries("error", "discovery", discoverErrs))
		return ExitConfigInvalid
	}

	items, parseEntries, schemaEntries := parseAndValidateFiles(files, cfg)

	constraintErrs := constraints.Evaluate(items, cfg.Types)
	constraintEntries := constraintErrorsToEntries(constraintErrs)

	allEntries := append(parseEntries, schemaEntries...)
	allEntries = append(allEntries, constraintEntries...)

	if len(allEntries) > 0 {
		reportErrors(resolvedFormat, allEntries)
		return ExitDataInvalid
	}

	// Check if any types define output
	hasOutput := false
	for _, td := range cfg.Types {
		if td.Output != nil {
			hasOutput = true
			break
		}
	}
	if !hasOutput {
		fmt.Fprintln(os.Stderr, "no types define output")
		return ExitOK
	}

	// Collect export data
	exportData := make(map[string][]any)
	for typeName, typeItems := range items {
		for _, item := range typeItems {
			exportData[typeName] = append(exportData[typeName], item.Data)
		}
	}

	results, exportErrs := export.Export(exportData, cfg.Types, rootDir)
	if len(exportErrs) > 0 {
		reportErrors(resolvedFormat, toReportEntries("error", "export", exportErrs))
		return ExitExportFailure
	}

	for _, r := range results {
		fmt.Fprintf(os.Stderr, "exported %d items to %s (%s)\n", r.Count, r.Path, r.Format)
	}

	return ExitOK
}

// RunTidy runs the tidy command.
// dryRun: if true, don't write changes.
// version: CLI version string.
// Returns exit code.
func RunTidy(dryRun bool, version string) int {
	cfg, resolvedFormat, code := loadAndValidateConfig("", version)
	if code != ExitOK {
		return code
	}

	if !cfg.Tidy.IsEnabled() {
		fmt.Fprintln(os.Stderr, "tidy is disabled")
		return ExitOK
	}

	if len(cfg.Types) == 0 {
		fmt.Fprintln(os.Stderr, "no types configured")
		return ExitOK
	}

	rootDir, _ := os.Getwd()
	files, discoverErrs := discovery.Discover(rootDir, cfg.Types)
	if len(discoverErrs) > 0 {
		reportErrors(resolvedFormat, toReportEntries("error", "discovery", discoverErrs))
		return ExitConfigInvalid
	}

	var tidyErrors []reportEntry
	var changed []string

	for _, f := range files {
		var sortBy []string
		if f.TypeDef.Tidy != nil {
			sortBy = f.TypeDef.Tidy.SortArraysBy
		}

		csvDelimiter := ","
		if f.TypeDef.CSV != nil && f.TypeDef.CSV.Delimiter != "" {
			csvDelimiter = f.TypeDef.CSV.Delimiter
		}

		absPath := filepath.Join(rootDir, f.Path)
		result, err := tidy.TidyFile(absPath, f.TypeDef.Input, sortBy, csvDelimiter, dryRun)
		if err != nil {
			tidyErrors = append(tidyErrors, reportEntry{
				Level:   "error",
				Type:    f.TypeName,
				File:    f.Path,
				Message: err.Error(),
			})
			continue
		}

		if result.Changed {
			changed = append(changed, f.Path)
		}
	}

	if len(tidyErrors) > 0 {
		reportErrors(resolvedFormat, tidyErrors)
		return ExitTidyFailure
	}

	if dryRun {
		for _, p := range changed {
			fmt.Fprintf(os.Stderr, "would tidy: %s\n", p)
		}
	} else {
		for _, p := range changed {
			fmt.Fprintf(os.Stderr, "tidied: %s\n", p)
		}
	}

	return ExitOK
}

// loadAndValidateConfig loads the .datacur8 config, applies defaults, validates it,
// and resolves the output format. Returns the config, resolved format, and exit code.
func loadAndValidateConfig(formatOverride string, version string) (*config.Config, string, int) {
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, "text", ExitConfigInvalid
	}

	configPath := filepath.Join(rootDir, ".datacur8")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "error: .datacur8 not found in current directory")
		return nil, "text", ExitConfigInvalid
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, "text", ExitConfigInvalid
	}

	resolvedFormat := cfg.Reporting.Mode
	if formatOverride != "" {
		resolvedFormat = formatOverride
	}

	warnings, errs := config.Validate(cfg, version)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if len(errs) > 0 {
		reportErrors(resolvedFormat, toReportEntries("error", "config", errs))
		return nil, resolvedFormat, ExitConfigInvalid
	}

	return cfg, resolvedFormat, ExitOK
}

// parseAndValidateFiles parses each discovered file and validates against schema.
// Returns the constraint items map, parse errors, and schema errors.
func parseAndValidateFiles(files []discovery.DiscoveredFile, cfg *config.Config) (
	map[string][]constraints.Item, []reportEntry, []reportEntry,
) {
	items := make(map[string][]constraints.Item)
	var parseEntries []reportEntry
	var schemaEntries []reportEntry

	for _, f := range files {
		rootDir, _ := os.Getwd()
		absPath := filepath.Join(rootDir, f.Path)

		rawData, err := os.ReadFile(absPath)
		if err != nil {
			parseEntries = append(parseEntries, reportEntry{
				Level:   "error",
				Type:    f.TypeName,
				File:    f.Path,
				Message: fmt.Sprintf("reading file: %v", err),
			})
			continue
		}

		parsed, perrs := parseDataFile(rawData, f.TypeDef.Input, f.TypeDef, f.Path)
		parseEntries = append(parseEntries, perrs...)

		if len(perrs) > 0 {
			continue
		}

		for i, data := range parsed {
			rowIndex := -1
			if f.TypeDef.Input == "csv" {
				rowIndex = i
			}

			schemaErrs := schema.ValidateItem(f.TypeDef.Schema, data, cfg.StrictMode)
			for _, se := range schemaErrs {
				entry := reportEntry{
					Level:   "error",
					Type:    f.TypeName,
					File:    f.Path,
					Message: se.Error(),
				}
				if rowIndex >= 0 {
					entry.Row = intPtr(rowIndex)
				}
				schemaEntries = append(schemaEntries, entry)
			}

			items[f.TypeName] = append(items[f.TypeName], constraints.Item{
				TypeName:     f.TypeName,
				FilePath:     f.Path,
				Data:         data,
				PathCaptures: f.PathCaptures,
				RowIndex:     rowIndex,
			})
		}
	}

	return items, parseEntries, schemaEntries
}

// parseDataFile parses raw file bytes into a slice of data items.
// JSON and YAML produce a single-element slice; CSV produces one per row.
func parseDataFile(raw []byte, inputFormat string, td *config.TypeDef, filePath string) ([]map[string]any, []reportEntry) {
	switch inputFormat {
	case "json":
		return parseJSON(raw, filePath)
	case "yaml":
		return parseYAML(raw, filePath)
	case "csv":
		return parseCSV(raw, td, filePath)
	default:
		return nil, []reportEntry{{
			Level:   "error",
			File:    filePath,
			Message: fmt.Sprintf("unsupported input format: %s", inputFormat),
		}}
	}
}

func parseJSON(raw []byte, filePath string) ([]map[string]any, []reportEntry) {
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, []reportEntry{{
			Level:   "error",
			File:    filePath,
			Message: fmt.Sprintf("parsing JSON: %v", err),
		}}
	}
	return []map[string]any{data}, nil
}

func parseYAML(raw []byte, filePath string) ([]map[string]any, []reportEntry) {
	var data map[string]any
	if err := yaml.Unmarshal(raw, &data); err != nil {
		return nil, []reportEntry{{
			Level:   "error",
			File:    filePath,
			Message: fmt.Sprintf("parsing YAML: %v", err),
		}}
	}
	return []map[string]any{data}, nil
}

func parseCSV(raw []byte, td *config.TypeDef, filePath string) ([]map[string]any, []reportEntry) {
	delim := ','
	if td.CSV != nil && td.CSV.Delimiter != "" {
		delim = rune(td.CSV.Delimiter[0])
	}

	reader := csv.NewReader(bytes.NewReader(raw))
	reader.Comma = delim
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []reportEntry{{
			Level:   "error",
			File:    filePath,
			Message: fmt.Sprintf("parsing CSV: %v", err),
		}}
	}

	if len(records) == 0 {
		return nil, []reportEntry{{
			Level:   "error",
			File:    filePath,
			Message: "CSV file is empty (no header row)",
		}}
	}

	headers := records[0]

	// Extract schema property types for type conversion
	propTypes := schemaPropertyTypes(td.Schema)

	// Extract required properties
	requiredProps := schemaRequiredProperties(td.Schema)

	// Validate headers: unknown headers are invalid
	var headerErrors []reportEntry
	for _, h := range headers {
		if _, ok := propTypes[h]; !ok {
			headerErrors = append(headerErrors, reportEntry{
				Level:   "error",
				File:    filePath,
				Message: fmt.Sprintf("CSV header %q not found in schema properties", h),
			})
		}
	}

	// Validate all required properties are in headers
	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}
	for _, req := range requiredProps {
		if !headerSet[req] {
			headerErrors = append(headerErrors, reportEntry{
				Level:   "error",
				File:    filePath,
				Message: fmt.Sprintf("required property %q missing from CSV headers", req),
			})
		}
	}

	if len(headerErrors) > 0 {
		return nil, headerErrors
	}

	var items []map[string]any
	var parseErrors []reportEntry

	for i, row := range records[1:] {
		item := make(map[string]any, len(headers))
		rowHasError := false

		for j, h := range headers {
			val := ""
			if j < len(row) {
				val = row[j]
			}

			propType := propTypes[h]
			converted, err := convertCSVValue(val, propType)
			if err != nil {
				parseErrors = append(parseErrors, reportEntry{
					Level:   "error",
					File:    filePath,
					Row:     intPtr(i),
					Message: fmt.Sprintf("row %d, column %q: %v", i, h, err),
				})
				rowHasError = true
				continue
			}
			item[h] = converted
		}

		if !rowHasError {
			items = append(items, item)
		}
	}

	if len(parseErrors) > 0 {
		return nil, parseErrors
	}

	return items, nil
}

// schemaPropertyTypes extracts property name -> type from a JSON Schema map.
func schemaPropertyTypes(schemaMap map[string]any) map[string]string {
	types := make(map[string]string)
	props, ok := schemaMap["properties"].(map[string]any)
	if !ok {
		return types
	}
	for name, v := range props {
		propSchema, ok := v.(map[string]any)
		if !ok {
			continue
		}
		if t, ok := propSchema["type"].(string); ok {
			types[name] = t
		}
	}
	return types
}

// schemaRequiredProperties extracts the "required" array from a JSON Schema map.
func schemaRequiredProperties(schemaMap map[string]any) []string {
	req, ok := schemaMap["required"].([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, v := range req {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// convertCSVValue converts a CSV string value to the appropriate Go type based on schema type.
func convertCSVValue(val string, schemaType string) (any, error) {
	switch schemaType {
	case "boolean":
		if val == "" {
			return nil, fmt.Errorf("empty value for boolean type")
		}
		lower := strings.ToLower(val)
		switch lower {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean value: %q", val)
		}

	case "number":
		if val == "" {
			return nil, fmt.Errorf("empty value for number type")
		}
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number value: %q", val)
		}
		return f, nil

	case "integer":
		if val == "" {
			return nil, fmt.Errorf("empty value for integer type")
		}
		i, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid integer value: %q", val)
		}
		return float64(i), nil

	default:
		// "string" or unknown types: return as-is
		return val, nil
	}
}

// reportErrors outputs errors in the given format.
func reportErrors(format string, entries []reportEntry) {
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(entries)
	case "yaml":
		_ = yaml.NewEncoder(os.Stdout).Encode(entries)
	default:
		for _, e := range entries {
			parts := []string{"error:"}
			if e.Type != "" {
				parts = append(parts, fmt.Sprintf("[%s]", e.Type))
			}
			if e.File != "" {
				parts = append(parts, e.File)
			}
			if e.Row != nil {
				parts = append(parts, fmt.Sprintf("(row %d)", *e.Row))
			}
			parts = append(parts, e.Message)
			fmt.Fprintln(os.Stderr, strings.Join(parts, " "))
		}
	}
}

// toReportEntries converts a slice of errors into reportEntry values.
func toReportEntries(level, category string, errs []error) []reportEntry {
	entries := make([]reportEntry, len(errs))
	for i, e := range errs {
		entries[i] = reportEntry{
			Level:   level,
			Type:    category,
			Message: e.Error(),
		}
	}
	return entries
}

// constraintErrorsToEntries converts constraint errors to report entries.
func constraintErrorsToEntries(errs []constraints.Error) []reportEntry {
	entries := make([]reportEntry, len(errs))
	for i, e := range errs {
		entries[i] = reportEntry{
			Level:   "error",
			Type:    e.TypeName,
			File:    e.FilePath,
			Message: fmt.Sprintf("[%s] %s", e.ConstraintType, e.Message),
		}
		if e.RowIndex >= 0 {
			entries[i].Row = intPtr(e.RowIndex)
		}
	}
	return entries
}

func intPtr(i int) *int { return &i }
