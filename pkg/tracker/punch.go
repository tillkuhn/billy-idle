package tracker

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
)

const (
	tablePunch  = "punch"
	hoursPerDay = 24
)

func (t *Tracker) UpsertPunchRecord(ctx context.Context, busyDuration time.Duration, day time.Time) error {
	uQuery := `UPDATE ` + tablePunch + `
			   SET busy_secs=$2,client=$3
               WHERE day=$1`
	day = truncateDay(day) // https://stackoverflow.com/a/38516536/4292075
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

// PunchRecords retried existing punch records for a specific time period
func (t *Tracker) PunchRecords(ctx context.Context) ([]PunchRecord, error) {
	// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track
	query := `SELECT day,busy_secs FROM ` + tablePunch + ` ORDER BY DAY` // WHERE busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 500`
	// We could use get since we expect a single result, but this would return an error if nothing is found
	// which is a likely use case
	var records []PunchRecord
	if err := t.db.SelectContext(ctx, &records, query /*, args*/); err != nil {
		return nil, err
	}
	return records, nil
}

func truncateDay(t time.Time) time.Time {
	return t.Truncate(hoursPerDay * time.Hour).UTC()
}
