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

// ============================================================================
// Daily driver pack copy — locked Spanish strings from design §10
// ============================================================================

// --- ScreenMemoryBrowser -----------------------------------------------------

// CopyMemoryEmpty is displayed when a tomo has no engram observations.
const CopyMemoryEmpty = "Este tomo aún no tiene memorias."

// CopyMemoryLoading is displayed while loadMemoryCmd is in flight.
const CopyMemoryLoading = "Consultando los pergaminos…"

// CopyMemoryError is the error flash format for memory read failures.
// Use fmt.Sprintf(CopyMemoryError, err.Error()).
const CopyMemoryError = "Los pergaminos están sellados: %s"

// --- ScreenProjectHistory ----------------------------------------------------

// CopyHistoryEmpty is displayed when a tomo has no git commits and no SDD archives.
const CopyHistoryEmpty = "Este tomo no tiene crónicas registradas."

// CopyHistoryLoading is displayed while loadHistoryCmd is in flight.
const CopyHistoryLoading = "Reuniendo la crónica…"

// CopyHistoryError is the error flash format for history load failures.
const CopyHistoryError = "El cronista no responde: %s"

// --- ScreenDiskUsage ---------------------------------------------------------

// CopyDiskLoading is displayed while loadDiskUsageCmd is in flight.
const CopyDiskLoading = "Pesando los pergaminos…"

// CopyDiskZeroPerTomo is shown for a tomo with no matching ~/.claude/projects/<key>/ entry.
const CopyDiskZeroPerTomo = "Sin crónica"

// --- Action flash messages ---------------------------------------------------

// CopyVSCodeMissing is shown when no VS Code executable is found.
const CopyVSCodeMissing = "VS Code no encontrado. Instalalo o agregálo al PATH."

// CopyGitMissing is shown when git is not on PATH.
const CopyGitMissing = "git no instalado."

// --- History entry markers (NOT translated — terminal-friendly) ---------------

// CopyHistoryGitMarker is the source prefix for git commit entries.
const CopyHistoryGitMarker = "[git]"

// CopyHistorySDDMarker is the source prefix for SDD archive entries.
const CopyHistorySDDMarker = "[sdd]"

// --- Footer hints (verbatim from design §10 — LOCKED) -------------------------

// CopyFooterMemoryList is the key-hint line for ScreenMemoryBrowser in list mode.
const CopyFooterMemoryList = "  /: buscar  ·  enter: abrir  ·  esc: volver"

// CopyFooterMemoryDetail is the key-hint line for ScreenMemoryBrowser in detail mode.
const CopyFooterMemoryDetail = "  esc: volver"

// CopyFooterHistoryList is the key-hint line for ScreenProjectHistory in list mode.
const CopyFooterHistoryList = "  enter: abrir  ·  esc: volver"

// CopyFooterHistoryDetail is the key-hint line for ScreenProjectHistory in detail mode.
const CopyFooterHistoryDetail = "  esc: volver"

// CopyFooterDiskUsage is the key-hint line for ScreenDiskUsage.
const CopyFooterDiskUsage = "  j/k: navegar  ·  enter: abrir en explorer  ·  esc: volver"

// CopyFooterProjectsExt REPLACES the current ScreenProjects footer hint.
const CopyFooterProjectsExt = "  n: inscribir  ·  /: buscar  ·  d: borrar  ·  m: monitor  ·  r: refrescar  ·  esc: volver"

// --- Date formatting helper --------------------------------------------------

// FormatHistoryDate renders ISO 2026-05-04 (locked answer #4, no relative dates).
func FormatHistoryDate(t interface{ Format(string) string }) string {
	return t.Format("2006-01-02")
}
