package tracker

import (
	"fmt"
	"math"
	"time"
)

// TruncateDay truncates a time to the beginning of the day
func TruncateDay(t time.Time) time.Time {
	return t.Truncate(hoursPerDay * time.Hour).UTC()
}

// FDur formats a duration to a human-readable string with hours (if > 0) and minutes
func FDur(d time.Duration) string {
	minutes := int(d.Minutes()) % minPerHour
	if d.Hours() < 0 {
		minutes = int(math.Abs(d.Minutes())) % minPerHour // no negative minutes
	}
	if minutes == 0 {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), minutes)
	/*
		switch {
		case d.Hours() > 0:
			return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%minPerHour)
		case d.Hours() < 0:

			return fmt.Sprintf("%dh%dm", int(d.Hours()), int(math.Abs(d.Minutes()))%minPerHour)
		default:
			return fmt.Sprintf("%dm", int(d.Minutes()))
			}
	*/
}
