package transcripts

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PriceTable computes the USD cost for a single assistant event.
// The Cost method returns (cost, known). known==false means the model is not
// in the table; callers should treat cost as 0 and flash a warning.
type PriceTable interface {
	Cost(modelID string, in, out, cacheCreate, cacheRead int) (float64, bool)
}

// modelPricing holds per-million-token prices for one model.
type modelPricing struct {
	InputPerM       float64
	OutputPerM      float64
	CacheCreatePerM float64
	CacheReadPerM   float64
}

// defaultPriceTable is the hardcoded price table (§8 of design).
// All prices are USD per million tokens.
type defaultPriceTable struct {
	entries map[string]modelPricing
}

// DefaultPriceTable returns a PriceTable with hardcoded prices for the four
// models listed in the design (§8). It is the source of truth unless overridden
// by ~/.atelier/model-prices.yaml at runtime.
func DefaultPriceTable() PriceTable {
	return &defaultPriceTable{
		entries: map[string]modelPricing{
			"claude-opus-4-7": {
				InputPerM:       15.00,
				OutputPerM:      75.00,
				CacheCreatePerM: 18.75,
				CacheReadPerM:   1.50,
			},
			"claude-sonnet-4-6": {
				InputPerM:       3.00,
				OutputPerM:      15.00,
				CacheCreatePerM: 3.75,
				CacheReadPerM:   0.30,
			},
			"claude-haiku-4-5": {
				InputPerM:       1.00,
				OutputPerM:      5.00,
				CacheCreatePerM: 1.25,
				CacheReadPerM:   0.10,
			},
			"claude-haiku-4-5-20251001": {
				InputPerM:       1.00,
				OutputPerM:      5.00,
				CacheCreatePerM: 1.25,
				CacheReadPerM:   0.10,
			},
		},
	}
}

// Cost computes:
//
//	cost = (in/1e6)*inPrice + (out/1e6)*outPrice
//	     + (cacheCreate/1e6)*createPrice + (cacheRead/1e6)*readPrice
//
// Returns (0, false) when modelID is not in the table.
func (t *defaultPriceTable) Cost(modelID string, in, out, cacheCreate, cacheRead int) (float64, bool) {
	p, ok := t.entries[modelID]
	if !ok {
		return 0, false
	}
	cost := (float64(in)/1e6)*p.InputPerM +
		(float64(out)/1e6)*p.OutputPerM +
		(float64(cacheCreate)/1e6)*p.CacheCreatePerM +
		(float64(cacheRead)/1e6)*p.CacheReadPerM
	return cost, true
}

// ---------------------------------------------------------------------------
// Override price table
// ---------------------------------------------------------------------------

// overridePriceTable merges an override set with a base (default) table.
// Override entries take precedence; models not in the override fall through to
// the base.
type overridePriceTable struct {
	base    PriceTable
	entries map[string]modelPricing
}

func (t *overridePriceTable) Cost(modelID string, in, out, cacheCreate, cacheRead int) (float64, bool) {
	if p, ok := t.entries[modelID]; ok {
		cost := (float64(in)/1e6)*p.InputPerM +
			(float64(out)/1e6)*p.OutputPerM +
			(float64(cacheCreate)/1e6)*p.CacheCreatePerM +
			(float64(cacheRead)/1e6)*p.CacheReadPerM
		return cost, true
	}
	return t.base.Cost(modelID, in, out, cacheCreate, cacheRead)
}

// ---------------------------------------------------------------------------
// YAML schema for the override file
// ---------------------------------------------------------------------------

// yamlModelEntry matches one entry under "models:" in model-prices.yaml.
type yamlModelEntry struct {
	Input       float64 `yaml:"input"`
	Output      float64 `yaml:"output"`
	CacheCreate float64 `yaml:"cache_create"`
	CacheRead   float64 `yaml:"cache_read"`
}

// yamlPriceFile is the top-level structure for ~/.atelier/model-prices.yaml.
type yamlPriceFile struct {
	Models map[string]yamlModelEntry `yaml:"models"`
}

// LoadOverridePriceTable reads path (normally ~/.atelier/model-prices.yaml),
// parses it, and returns a PriceTable that merges the overrides with the
// default hardcoded table.
//
// Returns (nil, error) when:
//   - the file does not exist
//   - the YAML is malformed
//
// The caller is responsible for falling back to DefaultPriceTable() on error.
func LoadOverridePriceTable(path string) (PriceTable, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("transcripts: price override: read %q: %w", path, err)
	}

	var pf yamlPriceFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("transcripts: price override: parse %q: %w", path, err)
	}

	entries := make(map[string]modelPricing, len(pf.Models))
	for id, e := range pf.Models {
		entries[id] = modelPricing{
			InputPerM:       e.Input,
			OutputPerM:      e.Output,
			CacheCreatePerM: e.CacheCreate,
			CacheReadPerM:   e.CacheRead,
		}
	}

	return &overridePriceTable{
		base:    DefaultPriceTable(),
		entries: entries,
	}, nil
}
