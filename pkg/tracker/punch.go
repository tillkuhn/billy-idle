package tracker

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"

	"github.com/pkg/errors"
)

const (
	tablePunch  = "punch"
	hoursPerDay = 24
)

// PunchReport displays the current punch report table layout
func (t *Tracker) PunchReport(ctx context.Context, monthOffset int) error {
	recs, err := t.PunchRecords(ctx, monthOffset)
	if err != nil {
		return err
	}
	var spentBusyTotal, plannedBusyTotal time.Duration

	table := tablewriter.NewWriter(t.opts.Out)
	bold := tablewriter.Colors{tablewriter.Bold}
	table.SetHeader([]string{"CW", "📅 Date", "Weekday", "🐝 Busy", "⏲️  Plan", "🕰 Overtime", "🗒️ Notes"})
	table.SetHeaderColor(bold, bold, bold, bold, bold, bold, bold)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAutoMergeCells(true)

	curWeek := 0
	for _, r := range recs {
		spentDay := time.Duration(r.BusySecs) * time.Second
		plannedDay := time.Duration(r.PlannedSecs) * time.Second

		// handle calendar week, if it changes during the report, print an empty line
		_, week := r.Day.ISOWeek() // week ranges from 1 to 53
		if curWeek > 0 && curWeek != week {
			table.Append([]string{"", "", "", "", "", "", ""})
		}
		curWeek = week

		table.Append([]string{
			strconv.Itoa(week),
			r.Day.Format(" 2006-01-02"),
			r.Day.Format("Monday"),
			FDur(spentDay),
			FDur(plannedDay),
			FDur(spentDay - plannedDay),
			r.Note,
		})
		spentBusyTotal += spentDay
		plannedBusyTotal += plannedDay
	}

	spentBusyTotal = spentBusyTotal.Round(time.Minute)
	pDays := len(recs)
	// expected := // time.Duration(pDays) * t.opts.RegBusy
	overtime := spentBusyTotal - plannedBusyTotal

	// Table Footer with Totals for the entire month
	table.SetFooter([]string{"🧮", "Total",
		fmt.Sprintf("%d days", pDays),
		FDur(spentBusyTotal),
		FDur(plannedBusyTotal),
		FDur(overtime),
		"",
	}) // Add Footer
	table.SetFooterColor(tablewriter.Colors{}, bold, bold,
		tablewriter.Colors{tablewriter.FgHiGreenColor}, tablewriter.Colors{}, bold, bold)
	table.Render()

	color.Set(color.FgGreen)
	color.Unset()
	return nil
}

// UpsertPunchRecord same ad Updates or inserts a punch record using the default planned duration (from Options)
func (t *Tracker) UpsertPunchRecord(ctx context.Context, busyDuration time.Duration, day time.Time, note string) error {
	return t.UpsertPunchRecordWithPlannedDuration(ctx, busyDuration, day, t.opts.RegBusy, note)
}

// UpsertPunchRecordWithPlannedDuration Updates or inserts a punch record into the database based on whether it already exists.
func (t *Tracker) UpsertPunchRecordWithPlannedDuration(ctx context.Context, busyDuration time.Duration,
	day time.Time, plannedDuration time.Duration, note string) error {
	uQuery := `UPDATE ` + tablePunch + `
			   SET busy_secs=$2, client=$3, planned_secs=$4, note=$5
               WHERE day=$1`
	day = TruncateDay(day) // https://stackoverflow.com/a/38516536/4292075
	uRes, err := t.db.ExecContext(ctx, uQuery, day, busyDuration.Seconds(), t.opts.ClientID, plannedDuration.Seconds(), note)
	if err != nil {
		return errors.Wrap(err, "unable to update "+tablePunch+" table")
	}
	if updated, _ := uRes.RowsAffected(); updated > 0 {
		log.Printf("🥫 Updated existing busy record for day %v duration %v note %s", day, busyDuration, note)
		return nil // record was already present, insert not required
	}

	// No update - let's insert a new row
	iQuery := `INSERT ` + `INTO ` + tablePunch + ` (day,busy_secs,client,planned_secs,note) VALUES ($1,$2,$3,$4,$5) RETURNING id`
	var id int
	if err := t.db.QueryRowContext(ctx, iQuery, day, busyDuration.Seconds(), t.opts.ClientID, plannedDuration.Seconds(), note).Scan(&id); err != nil {
		return errors.Wrap(err, "unable to insert new record in busy table")
	}
	log.Printf("🥫 New busy record for day %v duration %v created with id=%d", day, busyDuration, id)
	return nil
}

// PunchRecords retried existing punch records for the current month
func (t *Tracker) PunchRecords(ctx context.Context, monthOffset int) ([]PunchRecord, error) {
	// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track
	// current month: select * from punch where substr(day, 6, 2) = strftime('%m', 'now')
	// strftime uses current month minus monthOffset month as reference date
	// COALESCE hack: avoid null notes (cannot be reflected by a string), return empty string instead
	query := `SELECT day,busy_secs,planned_secs,COALESCE(NULLIF(note,''),'') AS note ` +
		`FROM ` + tablePunch + ` WHERE substr(day, 0, 8) = strftime('%Y-%m', 'now', '-` + strconv.Itoa(monthOffset) + ` month') ` +
		`ORDER BY DAY` // WHERE busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 500`
	// We could use get since we expect a single result, but this would return an error if nothing is found
	// which is a likely use case
	var records []PunchRecord
	if err := t.db.SelectContext(ctx, &records, query /*, args*/); err != nil {
		return nil, err
	}
	return records, nil
}
