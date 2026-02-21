package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/jsonschema-go/jsonschema"
	"gopkg.in/yaml.v3"
)

//go:embed config.schema.json
var configSchemaJSON []byte

var (
	configSchemaOnce sync.Once
	configSchema     *jsonschema.Resolved
	configSchemaErr  error
)

func validateAgainstEmbeddedSchema(rawYAML []byte) error {
	cfgData, err := parseYAMLToJSONShape(rawYAML)
	if err != nil {
		return err
	}

	resolved, err := getConfigSchema()
	if err != nil {
		return err
	}

	if err := resolved.Validate(cfgData); err != nil {
		return fmt.Errorf("configuration does not match schema: %w", err)
	}
	return nil
}

func getConfigSchema() (*jsonschema.Resolved, error) {
	configSchemaOnce.Do(func() {
		var s jsonschema.Schema
		if err := json.Unmarshal(configSchemaJSON, &s); err != nil {
			configSchemaErr = fmt.Errorf("decoding embedded config schema: %w", err)
			return
		}
		configSchema, configSchemaErr = s.Resolve(nil)
		if configSchemaErr != nil {
			configSchemaErr = fmt.Errorf("resolving embedded config schema: %w", configSchemaErr)
		}
	})
	if configSchemaErr != nil {
		return nil, configSchemaErr
	}
	return configSchema, nil
}

func parseYAMLToJSONShape(raw []byte) (any, error) {
	var v any
	if err := yaml.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	return normalizeYAMLValue(v)
}

func normalizeYAMLValue(v any) (any, error) {
	switch t := v.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, float32, float64:
		return t, nil
	case []any:
		out := make([]any, len(t))
		for i := range t {
			nv, err := normalizeYAMLValue(t[i])
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			nv, err := normalizeYAMLValue(vv)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case map[any]any:
		out := make(map[string]any, len(t))
		for k, vv := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("parsing config file: non-string map key %T found in YAML object", k)
			}
			nv, err := normalizeYAMLValue(vv)
			if err != nil {
				return nil, err
			}
			out[ks] = nv
		}
		return out, nil
	default:
		return nil, fmt.Errorf("parsing config file: unsupported YAML value type %T", v)
	}
}
