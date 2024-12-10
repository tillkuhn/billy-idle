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

// FDur formats a duration to a human-readable string with hours and minutes
// it takes a duration (time.Duration) as input and returns a string representation of it in the format "XhYm",
// where X is the number of hours and Y is the number of minutes.
// The function handles edge cases such as zero minutes or negative hours.
func FDur(d time.Duration) string {
	if d.Minutes() == 0 {
		return "0m"
	}
	hourStr := ""
	if d.Hours() > 1 || d.Hours() < -1 {
		hourStr = fmt.Sprintf("%dh", int(d.Hours()))
	}
	minStr := ""
	if int(d.Minutes())%minPerHour != 0 {
		if d.Hours() <= -1 {
			minStr = fmt.Sprintf("%dm", int(math.Abs(d.Minutes()))%minPerHour) // no negative minutes in negative hours
		} else {
			minStr = fmt.Sprintf("%dm", int(d.Minutes())%minPerHour)
		}
	}
	return hourStr + minStr
}
