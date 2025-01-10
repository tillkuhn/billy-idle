package tracker

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"

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

	// Outer Loop: key days (20xx-10-04)
	for dayIdx, day := range dailyRecs {
		lastDay := dayIdx == len(dailyRecs)-1

		// inner loop: show all track records per day
		recs := recMap[day]
		first := recs[0]
		last := recs[len(recs)-1]
		var spentBusy, spentTotal time.Duration
		var skippedTooShort int

		// headline per day
		table := tablewriter.NewWriter(t.opts.Out)
		bold := tablewriter.Colors{tablewriter.Bold}
		table.SetHeader([]string{"🕰", "Time Range", "🐝 Busy Record"})
		table.SetHeaderColor(bold, bold, bold)
		table.SetBorder(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAutoWrapText(false) // or long lines will be split
		// table.SetAutoMergeCells(true) // don't merge

		// Headline 📅 Friday, January 10, 20xx (YYYY-MM-DD) Daily Report
		color.Set(color.FgCyan, color.Bold)
		_, _ = fmt.Fprintf(w, "%s (%s) Daily Report\n%s\n", first.BusyStart.Format("Monday January 02, 2006"), day, strings.Repeat("-", sepLineLen))
		color.Unset()

		// busy / idle entries for each record per day
		for _, rec := range recs {
			if rec.Duration() >= t.opts.MinBusy {
				to := rec.BusyEnd.Time.Format("15:04:05")
				// _, _ = fmt.Fprintln(w, rec) // print details
				table.Append([]string{
					DayTimeIcon(rec.BusyStart),
					fmt.Sprintf("%s → %s", rec.BusyStart.Format("15:04:05"), to),
					fmt.Sprintf("Spent %s %s #%d", FDur(rec.Duration()), rec.Task, rec.ID), // rec.Id
				})
				spentBusy += rec.Duration()
			} else {
				skippedTooShort++
			}
		}

		if last.BusyEnd.Valid {
			spentTotal = last.BusyEnd.Time.Sub(first.BusyStart) // last record is complete
		} else {
			// end date for last record is missing: show anyway, and use either start instead of end time
			// or if this is the last record of the last day, calculate the relative time to now()
			// since this is likely the record that is currently still active
			table.Append([]string{
				DayTimeIcon(last.BusyStart),
				fmt.Sprintf("%s → now", last.BusyStart.Format("15:04:05")),
				fmt.Sprintf("Still busy with %s #%d", last.Task, last.ID), // rec.Id
			})
			if lastDay {
				spentTotal = time.Since(first.BusyStart)
				spentBusy += time.Since(last.BusyStart)
			} else {
				spentTotal = last.BusyStart.Sub(first.BusyStart)
				// ignore incomplete record for busy calc
			}
		}

		// footer with summaries and assessment
		// _, _ = fmt.Fprintln(w, strings.Repeat("-", sepLineLen))
		kitKat := mandatoryBreak(spentBusy)
		spentBusy = spentBusy.Round(time.Minute)
		spentTotal = spentTotal.Round(time.Minute)
		table.SetFooter([]string{"🧮",
			fmt.Sprintf("%s +break: %s", FDur(spentBusy), FDur((spentBusy + kitKat).Round(time.Minute))),
			fmt.Sprintf("Busy+Idle: %s  Skip(<%v): %d  >Reg(%v): %v  >Max(%s): %v",
				// first.BusyStart.Format("2006-01-02 Mon"),
				// FDur(spentBusy), // busy time (total - idle)
				// busy time including break
				FDur(spentTotal),                      // total time both busy + idle
				FDur(t.opts.MinBusy), skippedTooShort, // number of skipped records
				FDur(t.opts.RegBusy) /* reg busy time e.g. (7:48) */, FDur(spentBusy-t.opts.RegBusy), // overtime
				FDur(t.opts.MaxBusy), FDur(spentBusy-t.opts.MaxBusy), // over max
			),
			// fmt.Sprintf("%v\n>%v", FDur(overtime), FDur(plannedBusyTotal)),
		}) // Add Footer
		table.SetFooterColor(tablewriter.Colors{}, bold, tablewriter.Colors{})
		table.SetFooterAlignment(tablewriter.ALIGN_LEFT)
		// tablewriter.Colors{tablewriter.FgHiGreenColor}, tablewriter.Colors{}, tablewriter.Colors{})

		table.Render()

		// todo: raise warning if totalBusy  is > 10h (or busyPlus > 10:45), since more than 10h are not allowed
		sugStart, _ := time.Parse("15:04", "09:00")

		// Suggestion for time entry:
		overDur := spentBusy - t.opts.RegBusy
		var overInfo string
		if overDur > 0 {
			color.Set(color.FgGreen)
			overInfo = fmt.Sprintf("you've been busy %s longer than expected", FDur(overDur))
		} else {
			color.Set(color.FgHiMagenta)
			overInfo = fmt.Sprintf("you're expected to be busy for another %s", FDur(overDur*-1))
		}
		_, _ = fmt.Fprintf(w, "Suggestestion: %v → %v (inc. %vm break), %s!\n\n",
			// first.BusyStart.Format("Monday"),
			sugStart.Format("15:04"), // Simplified start
			sugStart.Add((spentBusy + kitKat).Round(time.Minute)).Format("15:04"), // Simplified end
			kitKat.Round(time.Minute).Minutes(),                                   // break duration depending on total busy time
			overInfo,
		)
		color.Unset()
		// _, _ = fmt.Fprintln(w, strings.Repeat("=", sepLineLen))
	}
	return nil
}
