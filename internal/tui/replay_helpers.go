package tui

import "time"

// replaySpeeds is the locked speed cycle: 0.5x → 1x → 2x → 4x (R5.4).
var replaySpeeds = []float64{0.5, 1.0, 2.0, 4.0}

// replaySpeedUp cycles to the next faster speed (wraps from 4x → 0.5x).
func replaySpeedUp(current float64) float64 {
	for i, s := range replaySpeeds {
		if s == current {
			return replaySpeeds[(i+1)%len(replaySpeeds)]
		}
	}
	return 1.0 // default if current is not in cycle
}

// replaySpeedDown cycles to the next slower speed (wraps from 0.5x → 4x).
func replaySpeedDown(current float64) float64 {
	for i, s := range replaySpeeds {
		if s == current {
			idx := (i - 1 + len(replaySpeeds)) % len(replaySpeeds)
			return replaySpeeds[idx]
		}
	}
	return 1.0 // default if current is not in cycle
}

// replayInterval returns the tick duration for the given replay speed.
// 1x = 1 event/second; 4x = 4 events/second; 0.5x = 1 event/2 seconds.
func replayInterval(speed float64) time.Duration {
	if speed <= 0 {
		return time.Second
	}
	return time.Duration(float64(time.Second) / speed)
}
