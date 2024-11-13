package tracker

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
)

func (t *Tracker) UpsertBusyRecord(ctx context.Context, busyDuration int) error {
	uQuery := `UPDATE busy
			   SET busy_secs=$2,client=$3
               WHERE day=$1`
	day := time.Now().Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	uRes, err := t.db.ExecContext(ctx, uQuery, day, busyDuration, t.opts.ClientID)
	if err != nil {
		return errors.Wrap(err, "unable to update busy table")
	}
	if updated, _ := uRes.RowsAffected(); updated > 0 {
		log.Printf("🥫 updated existing busy record for day %s duraction %ds", day, busyDuration)
		return nil // record was already present, insert not required
	}

	// No update - let's insert a new row
	iQuery := `INSERT INTO busy (day,busy_secs,client) VALUES ($1,$2,$3) RETURNING id`
	var id int
	if err := t.db.QueryRowContext(ctx, iQuery, day, busyDuration, t.opts.ClientID).Scan(&id); err != nil {
		return errors.Wrap(err, "unable to insert new record in busy table")
	}
	log.Printf("🥫new busy record for day %s duraction %ds created with id=%d", day, busyDuration, id)
	return nil
}
