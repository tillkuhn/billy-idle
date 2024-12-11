package tracker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

const minPerHour = 60
const sepLineLen = 100

// trackRecords retried existing track records for a specific time period
func (t *Tracker) trackRecords(ctx context.Context) (map[string][]TrackRecord, error) {
	// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || 'secs' AS total from track
	query := `SELECT * FROM track WHERE busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 500`
	// We could use get since we expect a single result, but this would return an error if nothing is found
	// which is a likely use case
	var records []TrackRecord
	if err := t.db.SelectContext(ctx, &records, query /*, args*/); err != nil {
		return nil, err
	}
	recMap := map[string][]TrackRecord{}
	for _, r := range records {
		k := r.BusyStart.Format("2006-01-02") // go ref Mon Jan 2 15:04:05 -0700 MST 2006
		recMap[k] = append(recMap[k], r)
	}
	return recMap, nil
}

// Report experimental report for time tracking apps
func (t *Tracker) Report(ctx context.Context, w io.Writer) error {
	recMap, err := t.trackRecords(ctx)
	if err != nil {
		return err
	}

	// go maps are unsorted, so we have to https://yourbasic.org/golang/sort-map-keys-values/
	dailyRecs := make([]string, 0, len(recMap))
	for k := range recMap {
		dailyRecs = append(dailyRecs, k)
	}
	sort.Strings(dailyRecs)

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
		// todo: raise warning if totalBusy  is > 10h (or busyPlus > 10:45), since more than 10h are not allowed
		_, _ = fmt.Fprintf(w, "BusyTime: %s  +Break: %s  Busy+Idle: %s  Skipped(<%v): %d  >Max(%s): %v\n",
			// first.BusyStart.Format("2006-01-02 Mon"),
			FDur(spentBusy), // busy time (total - idle)
			FDur((spentBusy + kitKat).Round(time.Minute)), // busy time including break
			FDur(spentTotal),                      // total time both busy + idle
			FDur(t.opts.MinBusy), skippedTooShort, // number of skipped records
			FDur(t.opts.MaxBusy) /* max busy time e.g. 10h */, spentBusy > t.opts.MaxBusy, /* over max? */
		)
		sugStart, _ := time.Parse("15:04", "09:00")

		color.Set(color.FgGreen)
		_, _ = fmt.Fprintf(w, "Suggest.: %v → %v (%vm break)  OverReg(>%v): %v  OverMax(>%v): %v\n",
			// first.BusyStart.Format("Monday"),
			sugStart.Format("15:04"), // Simplified start
			sugStart.Add((spentBusy + kitKat).Round(time.Minute)).Format("15:04"),                // Simplified end
			kitKat.Round(time.Minute).Minutes(),                                                  // break time depending on total busy time
			FDur(t.opts.RegBusy) /* reg busy time e.g. (7:48) */, FDur(spentBusy-t.opts.RegBusy), // overtime
			FDur(t.opts.MaxBusy), FDur(spentBusy-t.opts.MaxBusy), // over max
		)
		color.Unset()
		_, _ = fmt.Fprintln(w, strings.Repeat("=", sepLineLen))
		_, _ = fmt.Fprintln(w, "")
	}
	return nil
}
