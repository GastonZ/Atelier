package nowplaying

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

// NewProvider returns the best Provider for the current OS.
//
// On Windows it returns an SMTC-backed provider that shells out to PowerShell.
// On every other platform it returns a no-op provider so the rest of the app
// (and cross-platform CI builds) compile and run unchanged — the now-playing
// card simply never appears.
func NewProvider() Provider {
	if runtime.GOOS == "windows" {
		return &execProvider{runPS: defaultRunPS}
	}
	return noopProvider{}
}

// execProvider reads the current track by running a PowerShell helper.
//
// runPS is injectable: production uses defaultRunPS; tests swap in a stub that
// returns canned helper output, so Current() is fully unit-testable without
// spawning a real shell or depending on what happens to be playing.
type execProvider struct {
	runPS func() (string, error)
}

// Current runs the helper and parses its output into a Track.
func (p *execProvider) Current() (Track, error) {
	raw, err := p.runPS()
	if err != nil {
		return Track{}, &HelperError{Message: err.Error()}
	}
	return parseTrack(raw)
}

// noopProvider is used on non-Windows platforms.
type noopProvider struct{}

func (noopProvider) Current() (Track, error) { return Track{Present: false}, nil }

// smtcScript queries the Global System Media Transport Controls session.
//
// It awaits the two WinRT async calls (the session manager and the media
// properties), then prints a compact JSON object — or the NO_SESSION sentinel
// when nothing owns a session, or "ERROR: <msg>" on any failure. The AsTask
// selector matches IAsyncOperation* by name to avoid the backtick in the raw
// generic type name (which would otherwise clash with Go's raw string quoting).
const smtcScript = `
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -like 'IAsyncOperation*' })[0]
Function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}
try {
  [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType = WindowsRuntime] | Out-Null
  $mgr = Await ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])
  $session = $mgr.GetCurrentSession()
  if ($null -eq $session) {
    Write-Output 'NO_SESSION'
  } else {
    $info = Await ($session.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])
    $playback = $session.GetPlaybackInfo()
    [PSCustomObject]@{
      Title  = $info.Title
      Artist = $info.Artist
      App    = $session.SourceAppUserModelId
      Status = $playback.PlaybackStatus.ToString()
    } | ConvertTo-Json -Compress
  }
} catch {
  Write-Output ('ERROR: ' + $_.Exception.Message)
}
`

// defaultRunPS executes the SMTC helper with a short timeout and returns its
// stdout. A timeout guards against a wedged WinRT call hanging the poller.
func defaultRunPS() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell",
		"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-Command", smtcScript)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
