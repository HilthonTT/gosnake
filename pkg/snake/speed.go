package snake

import "time"

const (
	BaseTickInterval = time.Millisecond * 160
	MinTickInterval  = time.Millisecond * 40
	SpeedIncrement   = time.Millisecond * 12
)

// GetTickInterval returns the tick interval for a given level.
// Higher levels yield faster ticks, clamped at MinTickInterval.
func GetTickInterval(level int) time.Duration {
	if level <= 1 {
		return BaseTickInterval
	}

	interval := BaseTickInterval - time.Duration(level-1)*SpeedIncrement
	if interval < MinTickInterval {
		return MinTickInterval
	}
	return interval
}
