package tracker

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

const minPerHour = 60
const sepLineLen = 100

// Report experimental report for time tracking apps
func (t *Tracker) Report(ctx context.Context, w io.Writer) error {
	recMap, err := t.getRecords(ctx)
	if err != nil {
		return err
	}

	// go maps are unsorted, so we have to https://yourbasic.org/golang/sort-map-keys-values/
	dailyRecs := make([]string, 0, len(recMap))
	for k := range recMap {
		dailyRecs = append(dailyRecs, k)
	}
	sort.Strings(dailyRecs)

	_, _ = fmt.Fprintf(w, "\n%s DAILY BILLY IDLE REPORT %s\n", strings.Repeat("-", 30), strings.Repeat("-", 30))
	// Outer Loop: key days (2024-10-04)
	for dayIdx, day := range dailyRecs {
		lastDay := dayIdx == len(dailyRecs)-1

		// inner loop: track records per day
		recs := recMap[day]
		first := recs[0]
		last := recs[len(recs)-1]
		var spentBusy, spentTotal time.Duration
		var skippedTooShort int

		// headline per day
		color.Set(color.FgCyan, color.Bold)
		_, _ = fmt.Fprintf(w, "🕰  %s (%s) Daily Report\n%s\n", first.BusyStart.Format("Monday January 02, 2006"), day, strings.Repeat("-", sepLineLen))
		color.Unset()

		for _, rec := range recs {
			if rec.Duration() >= t.opts.MinBusy {
				_, _ = fmt.Fprintln(w, rec) // print details
				spentBusy += rec.Duration()
			} else {
				skippedTooShort++
			}
		}

		if last.BusyEnd.Valid {
			spentTotal = last.BusyEnd.Time.Sub(first.BusyStart) // last record is complete
		} else {
			// last record not complete, show anyway and use either start instead of end time
			// or if this is the last record of the last day, calculate the relative time to now()
			// since this is likely the record that is still active
			_, _ = fmt.Fprintln(w, last)
			if lastDay {
				spentTotal = time.Since(first.BusyStart)
				spentBusy += time.Since(last.BusyStart)
			} else {
				spentTotal = last.BusyStart.Sub(first.BusyStart)
				// ignore incomplete record for busy calc
			}
		}
		_, _ = fmt.Fprintln(w, strings.Repeat("-", sepLineLen))

		kitKat := mandatoryBreak(spentBusy)
		spentBusy = spentBusy.Round(time.Minute)
		spentTotal = spentTotal.Round(time.Minute)
		color.Set(color.FgGreen)
		// todo: raise warning if totalBusy  is > 10h (or busyPlus > 10:45), since more than 10h are not allowed
		_, _ = fmt.Fprintf(w, "Busy: %s  WithBreak(%vm): %s  Skipped(<%v): %d  >Max(%s): %v  Range: %s\n",
			// first.BusyStart.Format("2006-01-02 Mon"),
			fDur(spentBusy),
			kitKat.Round(time.Minute).Minutes(),
			fDur((spentBusy + kitKat).Round(time.Minute)),
			fDur(t.opts.MinBusy), skippedTooShort,
			fDur(t.opts.MaxBusy), spentBusy > t.opts.MaxBusy,
			fDur(spentTotal),
		)
		sugStart, _ := time.Parse("15:04", "09:00")

		_, _ = fmt.Fprintf(w, "Simple Entry for %s: %v → %v (inc. break)  Overtime(>%v): %v\n",
			first.BusyStart.Format("Monday"),
			sugStart.Format("15:04"),
			sugStart.Add((spentBusy + kitKat).Round(time.Minute)).Format("15:04"),
			fDur(t.opts.RegBusy), fDur(spentBusy-t.opts.RegBusy),
		)
		color.Unset()
		_, _ = fmt.Fprintln(w, strings.Repeat("=", sepLineLen))
		_, _ = fmt.Fprintln(w, "")
	}
	return nil
}

func fDur(d time.Duration) string {
	switch {
	case d.Hours() > 0:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%minPerHour)
	case d.Hours() < 0:

		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(math.Abs(d.Minutes()))%minPerHour)
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}
