package disk

import "fmt"

const (
	kb = 1024
	mb = 1024 * kb
	gb = 1024 * mb
)

// HumanReadable formats bytes as a human-readable string.
// Below 1 KB: "N B" (integer, no decimals).
// 1 KB and above: "N.NN KB" / "N.NN MB" / "N.NN GB" (2 decimal places).
// Design R5.5: B / KB / MB / GB units only.
func HumanReadable(bytes int64) string {
	switch {
	case bytes < kb:
		return fmt.Sprintf("%d B", bytes)
	case bytes < mb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/kb)
	case bytes < gb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/mb)
	default:
		return fmt.Sprintf("%.2f GB", float64(bytes)/gb)
	}
}
