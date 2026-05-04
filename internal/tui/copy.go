package tui

// copy.go — locked Spanish copy strings from design §10.
// ALL UI copy for the agent-monitor screens lives here.
// View functions MUST import from here — NO inline string literals.

// --- Monitor screen ----------------------------------------------------------

// CopyMonitorEmpty is displayed when the active session set is zero.
const CopyMonitorEmpty = "El atelier duerme."

// CopyMonitorLoading is displayed while Scanner.ListActive is in flight.
const CopyMonitorLoading = "El atelier despierta…"

// CopyMonitorUnmatched is the group label for sessions whose cwd matches no registered project.
const CopyMonitorUnmatched = "Sin tomo registrado"

// --- Tile fields -------------------------------------------------------------

// CopyCostLine is the format string for the per-tile cost display.
// Use fmt.Sprintf(CopyCostLine, totalUSD).
const CopyCostLine = "Pergaminos gastados: USD %0.4f"

// CopyLastEventHeader is the header for the last-event preview inside a tile.
const CopyLastEventHeader = "Último trazo:"

// --- Replay screen -----------------------------------------------------------

// CopyReplayHeader is the title line for the replay screen.
const CopyReplayHeader = "Crónica del taller"

// --- Flash messages ----------------------------------------------------------

// CopyPricingWarning is the flash message for an unknown model pricing.
// Use fmt.Sprintf(CopyPricingWarning, modelID).
const CopyPricingWarning = "El pergamino de precios trae un símbolo desconocido: %s"

// CopyWatcherError is the flash message for a watcher failure.
// Use fmt.Sprintf(CopyWatcherError, err.Error()).
const CopyWatcherError = "Los oídos del atelier perdieron el rastro: %s"

// --- Footer hints (verbatim from design §10 — LOCKED) -------------------------

// CopyFooterMonitor is the key-hint line for ScreenAgentMonitor.
const CopyFooterMonitor = "  1-9: ir al tile  ·  j/k: navegar  ·  enter: zoom  ·  o/c: abrir/cerrar  ·  esc: volver"

// CopyFooterZoom is the key-hint line for ScreenAgentZoom.
const CopyFooterZoom = "  r: revivir  ·  esc: volver"

// CopyFooterReplay is the key-hint line for ScreenAgentReplay.
const CopyFooterReplay = "  </>: paso  ·  espacio: pausar  ·  +/-: velocidad  ·  esc: volver"
