package transcripts_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gastonz/atelier/internal/transcripts"
)

// ---- DefaultPriceTable tests -------------------------------------------------

func TestDefaultPriceTable_SonnetCost(t *testing.T) {
	// S4.1: known model with non-zero token counts gives correct cost
	// claude-sonnet-4-6: input=3.00, output=15.00, cacheCreate=3.75, cacheRead=0.30 (per million)
	pt := transcripts.DefaultPriceTable()

	// 1,000,000 input tokens = $3.00
	cost, known := pt.Cost("claude-sonnet-4-6", 1_000_000, 0, 0, 0)
	if !known {
		t.Error("claude-sonnet-4-6 should be known")
	}
	if abs(cost-3.00) > 0.0001 {
		t.Errorf("expected $3.00 for 1M input tokens on Sonnet, got %.6f", cost)
	}
}

func TestDefaultPriceTable_OpusCost(t *testing.T) {
	// claude-opus-4-7: input=15.00, output=75.00 per million
	pt := transcripts.DefaultPriceTable()

	// 1,000,000 output tokens = $75.00
	cost, known := pt.Cost("claude-opus-4-7", 0, 1_000_000, 0, 0)
	if !known {
		t.Error("claude-opus-4-7 should be known")
	}
	if abs(cost-75.00) > 0.0001 {
		t.Errorf("expected $75.00 for 1M output tokens on Opus, got %.6f", cost)
	}
}

func TestDefaultPriceTable_HaikuCost(t *testing.T) {
	// claude-haiku-4-5-20251001: input=1.00, output=5.00 per million
	pt := transcripts.DefaultPriceTable()

	cost, known := pt.Cost("claude-haiku-4-5-20251001", 0, 1_000_000, 0, 0)
	if !known {
		t.Error("claude-haiku-4-5-20251001 should be known")
	}
	if abs(cost-5.00) > 0.0001 {
		t.Errorf("expected $5.00 for 1M output tokens on Haiku, got %.6f", cost)
	}
}

func TestDefaultPriceTable_CacheTokenPricing(t *testing.T) {
	// S4.2: cache tokens priced at their respective rates
	// claude-sonnet-4-6: cacheCreate=3.75, cacheRead=0.30 per million
	pt := transcripts.DefaultPriceTable()

	// 1M cache_create tokens
	costCreate, known := pt.Cost("claude-sonnet-4-6", 0, 0, 1_000_000, 0)
	if !known {
		t.Error("claude-sonnet-4-6 should be known")
	}
	if abs(costCreate-3.75) > 0.0001 {
		t.Errorf("expected $3.75 for 1M cache_create tokens, got %.6f", costCreate)
	}

	// 1M cache_read tokens
	costRead, known := pt.Cost("claude-sonnet-4-6", 0, 0, 0, 1_000_000)
	if !known {
		t.Error("claude-sonnet-4-6 should be known")
	}
	if abs(costRead-0.30) > 0.0001 {
		t.Errorf("expected $0.30 for 1M cache_read tokens, got %.6f", costRead)
	}
}

func TestDefaultPriceTable_UnknownModel(t *testing.T) {
	// S4.3 / R4.4: unknown model returns (0, false) — does NOT panic
	pt := transcripts.DefaultPriceTable()

	cost, known := pt.Cost("claude-unknown-99", 1_000_000, 1_000_000, 0, 0)
	if known {
		t.Error("unknown model should return known=false")
	}
	if cost != 0 {
		t.Errorf("unknown model should return cost=0, got %.6f", cost)
	}
}

func TestDefaultPriceTable_CombinedCost(t *testing.T) {
	// Verify the full formula: input+output+cacheCreate+cacheRead
	// claude-opus-4-7: input=15, output=75, cacheCreate=18.75, cacheRead=1.50
	// Use: 100K input, 50K output, 200K cacheCreate, 500K cacheRead
	pt := transcripts.DefaultPriceTable()

	cost, known := pt.Cost("claude-opus-4-7", 100_000, 50_000, 200_000, 500_000)
	if !known {
		t.Error("claude-opus-4-7 should be known")
	}

	expected := (100_000.0/1e6)*15.00 +
		(50_000.0/1e6)*75.00 +
		(200_000.0/1e6)*18.75 +
		(500_000.0/1e6)*1.50

	if abs(cost-expected) > 0.000001 {
		t.Errorf("expected %.6f, got %.6f", expected, cost)
	}
}

// ---- LoadOverridePriceTable tests -------------------------------------------

func TestLoadOverridePriceTable_ValidYAML(t *testing.T) {
	// S4.4: valid override file merges with default; override wins on collision
	dir := t.TempDir()
	yamlContent := `models:
  claude-opus-4-7:
    input: 99.00
    output: 199.00
    cache_create: 24.00
    cache_read: 2.00
  my-custom-model:
    input: 2.50
    output: 12.00
    cache_create: 3.00
    cache_read: 0.25
`
	path := filepath.Join(dir, "model-prices.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("could not write fixture: %v", err)
	}

	pt, err := transcripts.LoadOverridePriceTable(path)
	if err != nil {
		t.Fatalf("LoadOverridePriceTable error: %v", err)
	}

	// Override wins: claude-opus-4-7 input should be $99.00/M, not $15.00/M
	cost, known := pt.Cost("claude-opus-4-7", 1_000_000, 0, 0, 0)
	if !known {
		t.Error("claude-opus-4-7 should be known after override")
	}
	if abs(cost-99.00) > 0.0001 {
		t.Errorf("expected override price $99.00, got %.6f", cost)
	}

	// Custom model should be found
	cost2, known2 := pt.Cost("my-custom-model", 1_000_000, 0, 0, 0)
	if !known2 {
		t.Error("my-custom-model should be known after override")
	}
	if abs(cost2-2.50) > 0.0001 {
		t.Errorf("expected custom price $2.50, got %.6f", cost2)
	}

	// Non-overridden model (claude-sonnet-4-6) still uses default
	costSonnet, knownSonnet := pt.Cost("claude-sonnet-4-6", 1_000_000, 0, 0, 0)
	if !knownSonnet {
		t.Error("claude-sonnet-4-6 should still be known via default fallback")
	}
	if abs(costSonnet-3.00) > 0.0001 {
		t.Errorf("expected default Sonnet price $3.00, got %.6f", costSonnet)
	}
}

func TestLoadOverridePriceTable_MalformedYAML(t *testing.T) {
	// S4.5: malformed YAML returns error; does NOT panic
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("this: is: not: valid: yaml:::::"), 0600); err != nil {
		t.Fatalf("could not write fixture: %v", err)
	}

	pt, err := transcripts.LoadOverridePriceTable(path)
	if err == nil {
		t.Error("expected error for malformed YAML, got nil")
	}
	// Even on error, pt should be nil or non-functional (caller uses default).
	// We just verify no panic occurred (reaching here is proof).
	_ = pt
}

func TestLoadOverridePriceTable_MissingFile(t *testing.T) {
	// R4.4: missing file returns error
	_, err := transcripts.LoadOverridePriceTable("/nonexistent/path/model-prices.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ---- T9: TRIANGULATE edge cases + FakePriceTable ----------------------------

func TestDefaultPriceTable_ZeroTokenCounts_ZeroCost(t *testing.T) {
	// Zero token counts yield $0.00 cost — not negative, not NaN
	pt := transcripts.DefaultPriceTable()

	cost, known := pt.Cost("claude-sonnet-4-6", 0, 0, 0, 0)
	if !known {
		t.Error("claude-sonnet-4-6 should be known")
	}
	if cost != 0.0 {
		t.Errorf("expected $0.00 for zero tokens, got %.6f", cost)
	}
}

func TestLoadOverridePriceTable_PartialOverride_DefaultFallthrough(t *testing.T) {
	// Override file with only some models: non-overridden models use default values.
	dir := t.TempDir()
	yamlContent := `models:
  my-cheap-model:
    input: 0.50
    output: 2.00
    cache_create: 0.75
    cache_read: 0.10
`
	path := filepath.Join(dir, "partial.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0600); err != nil {
		t.Fatalf("could not write fixture: %v", err)
	}

	pt, err := transcripts.LoadOverridePriceTable(path)
	if err != nil {
		t.Fatalf("LoadOverridePriceTable error: %v", err)
	}

	// Non-overridden claude-haiku-4-5-20251001 should still be at default price
	cost, known := pt.Cost("claude-haiku-4-5-20251001", 1_000_000, 0, 0, 0)
	if !known {
		t.Error("claude-haiku-4-5-20251001 should be known via default fallback")
	}
	if abs(cost-1.00) > 0.0001 {
		t.Errorf("expected default Haiku price $1.00/M, got %.6f", cost)
	}

	// Custom model is found
	cost2, known2 := pt.Cost("my-cheap-model", 1_000_000, 0, 0, 0)
	if !known2 {
		t.Error("my-cheap-model should be known")
	}
	if abs(cost2-0.50) > 0.0001 {
		t.Errorf("expected $0.50 for my-cheap-model, got %.6f", cost2)
	}
}

// FakePriceTable is a configurable PriceTable for use in other test files
// within this package. It returns a fixed cost for all known models.
// Unknown models return (0, false).
type FakePriceTable struct {
	FixedCost  float64
	KnownModels map[string]bool
}

// NewFakePriceTable returns a FakePriceTable that treats all modelIDs in
// known as recognized models, returning fixedCost for each.
func NewFakePriceTable(fixedCost float64, known ...string) *FakePriceTable {
	m := make(map[string]bool, len(known))
	for _, k := range known {
		m[k] = true
	}
	return &FakePriceTable{FixedCost: fixedCost, KnownModels: m}
}

// Cost implements transcripts.PriceTable.
func (f *FakePriceTable) Cost(modelID string, _, _, _, _ int) (float64, bool) {
	if f.KnownModels[modelID] {
		return f.FixedCost, true
	}
	return 0, false
}

func (f *FakePriceTable) String() string {
	return "FakePriceTable(fixed=" + fmt.Sprintf("%.4f", f.FixedCost) + ")"
}

func TestFakePriceTable_KnownModel(t *testing.T) {
	fake := NewFakePriceTable(0.0042, "claude-sonnet-4-6")

	cost, known := fake.Cost("claude-sonnet-4-6", 100, 200, 0, 0)
	if !known {
		t.Error("expected known=true for claude-sonnet-4-6")
	}
	if cost != 0.0042 {
		t.Errorf("expected fixed cost 0.0042, got %.6f", cost)
	}
}

func TestFakePriceTable_UnknownModel(t *testing.T) {
	fake := NewFakePriceTable(1.00, "claude-sonnet-4-6")

	cost, known := fake.Cost("mystery-model", 100, 200, 0, 0)
	if known {
		t.Error("expected known=false for mystery-model")
	}
	if cost != 0 {
		t.Errorf("expected cost=0 for unknown model, got %.6f", cost)
	}
}

// ---- abs helper (no math import needed for these simple checks) --------------

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
