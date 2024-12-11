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
func (t *Tracker) PunchReport(ctx context.Context) error {
	recs, err := t.PunchRecords(ctx)
	if err != nil {
		return err
	}
	var spentBusy time.Duration
	table := tablewriter.NewWriter(t.opts.Out)
	bold := tablewriter.Colors{tablewriter.Bold}
	table.SetHeader([]string{"🕰 Date", "CW", "Weekday", "🐝 Busy Time"})
	table.SetHeaderColor(bold, bold, bold, bold)
	table.SetBorder(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)

	for _, r := range recs {
		spentDay := time.Duration(r.BusySecs) * time.Second
		_, week := r.Day.ISOWeek()
		table.Append([]string{
			r.Day.Format(" 2006-01-02"),
			strconv.Itoa(week),
			r.Day.Format("Monday"),
			FDur(spentDay),
		})
		spentBusy += spentDay
	}

	spentBusy = spentBusy.Round(time.Minute)
	pDays := len(recs)
	expected := time.Duration(pDays) * t.opts.RegBusy
	overtime := spentBusy - expected

	// Table Footer with totals
	table.SetFooter([]string{"", "", "Total\nOvertime",
		fmt.Sprintf("%s (%ddays)\n%v (>%v)", FDur(spentBusy), pDays, FDur(overtime), FDur(expected)),
	}) // Add Footer
	table.SetFooterColor(tablewriter.Colors{}, tablewriter.Colors{}, bold,
		tablewriter.Colors{tablewriter.FgHiGreenColor})
	table.Render()

	color.Set(color.FgGreen)
	// fmt.Printf("AVG/DAY: %v  REGULAR (%dd*%v): %v\n", tracker.FDur(spentBusy/time.Duration(pDays)),  pDays, tracker.FDur(punchOpts.RegBusy) )
	color.Unset()

	return nil
}

// UpsertPunchRecord Updates or inserts a punch record into the database based on whether it already exists.
func (t *Tracker) UpsertPunchRecord(ctx context.Context, busyDuration time.Duration, day time.Time) error {
	uQuery := `UPDATE ` + tablePunch + `
			   SET busy_secs=$2,client=$3
               WHERE day=$1`
	day = TruncateDay(day) // https://stackoverflow.com/a/38516536/4292075
	uRes, err := t.db.ExecContext(ctx, uQuery, day, busyDuration.Seconds(), t.opts.ClientID)
	if err != nil {
		return errors.Wrap(err, "unable to update busy table")
	}
	if updated, _ := uRes.RowsAffected(); updated > 0 {
		log.Printf("🥫 Updated existing busy record for day %v duraction %v", day, busyDuration)
		return nil // record was already present, insert not required
	}

	// No update - let's insert a new row
	iQuery := `INSERT INTO ` + tablePunch + ` (day,busy_secs,client) VALUES ($1,$2,$3) RETURNING id`
	var id int
	if err := t.db.QueryRowContext(ctx, iQuery, day, busyDuration.Seconds(), t.opts.ClientID).Scan(&id); err != nil {
		return errors.Wrap(err, "unable to insert new record in busy table")
	}
	log.Printf("🥫 New busy record for day %v duraction %v created with id=%d", day, busyDuration, id)
	return nil
}

// PunchRecords retried existing punch records for the current month
func (t *Tracker) PunchRecords(ctx context.Context) ([]PunchRecord, error) {
	// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track
	// current month: select * from punch where substr(day, 6, 2) = strftime('%m', 'now')
	query := `SELECT day,busy_secs FROM ` + tablePunch + ` WHERE substr(day, 6, 2) = strftime('%m', 'now') ` +
		`ORDER BY DAY` // WHERE busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 500`
	// We could use get since we expect a single result, but this would return an error if nothing is found
	// which is a likely use case
	var records []PunchRecord
	if err := t.db.SelectContext(ctx, &records, query /*, args*/); err != nil {
		return nil, err
	}
	return records, nil
}
