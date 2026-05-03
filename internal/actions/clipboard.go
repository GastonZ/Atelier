package actions

import "github.com/atotto/clipboard"

// atottoClipboard implements Clipboard using github.com/atotto/clipboard.
// Wrapped behind the Clipboard interface so swapping to a different backend
// is a one-file change.
type atottoClipboard struct{}

// WriteAll writes text to the system clipboard.
func (a *atottoClipboard) WriteAll(text string) error {
	return clipboard.WriteAll(text)
}
