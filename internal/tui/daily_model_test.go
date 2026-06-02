package tui_test

// daily_model_test.go — RED tests for T28/T29: new model fields and screen iotas.
// These tests compile-guard that the fields and constants exist on Model.

import (
	"testing"

	"github.com/gastonz/atelier/internal/tui"
)

// TestDailyScreenIotas_ExistAfterExisting verifies the three new screen constants
// are defined and have values greater than the last existing screen iota.
func TestDailyScreenIotas_ExistAfterExisting(t *testing.T) {
	// ScreenAgentReplay is the last existing iota (value 7).
	// New screens must be strictly after it (values 8, 9, 10).
	if tui.ScreenMemoryBrowser <= tui.ScreenAgentReplay {
		t.Errorf("ScreenMemoryBrowser (%d) must be > ScreenAgentReplay (%d)", tui.ScreenMemoryBrowser, tui.ScreenAgentReplay)
	}
	if tui.ScreenProjectHistory <= tui.ScreenMemoryBrowser {
		t.Errorf("ScreenProjectHistory (%d) must be > ScreenMemoryBrowser (%d)", tui.ScreenProjectHistory, tui.ScreenMemoryBrowser)
	}
	if tui.ScreenDiskUsage <= tui.ScreenProjectHistory {
		t.Errorf("ScreenDiskUsage (%d) must be > ScreenProjectHistory (%d)", tui.ScreenDiskUsage, tui.ScreenProjectHistory)
	}
	// Existing screens must not change values.
	if tui.ScreenWelcome != 0 {
		t.Errorf("ScreenWelcome = %d, want 0 (must not change)", tui.ScreenWelcome)
	}
	if tui.ScreenAgentReplay != 7 {
		t.Errorf("ScreenAgentReplay = %d, want 7 (must not change)", tui.ScreenAgentReplay)
	}
}

// TestDailyModelFields_NewFieldsZeroValueOnNew verifies that new exported/accessible
// fields are initialized to their zero values by New().
// Tests the fields are accessible (compile-guards them).
func TestDailyModelFields_NewFieldsZeroValueOnNew(t *testing.T) {
	m := newTestModel(t)

	// GitStatusCache accessor — zero map on fresh model (nil is ok, as it's lazy-init)
	cache := m.GitStatusCache()
	_ = cache // nil is acceptable for zero value

	// DiskLoaded — false on fresh model
	if m.DiskLoaded() {
		t.Error("DiskLoaded should be false on fresh model")
	}

	// ActionCursor — 0 on fresh model
	if m.ActionCursor != 0 {
		t.Errorf("ActionCursor = %d, want 0 on fresh model", m.ActionCursor)
	}
}
