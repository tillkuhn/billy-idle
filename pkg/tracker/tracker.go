package tracker

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jmoiron/sqlx"
	sqlite3 "modernc.org/sqlite/lib"
)

// Package embed provides access to Files embedded in the running Go program.
//
//go:embed init-db.sql
var initSQL string

// Tracker tracks idle state periodically and persist state changes in DB
type Tracker struct {
	opts *Options
	db   *sqlx.DB
	wg   sync.WaitGroup
}

// New returns a new Tracker configured with the given Options
func New(opts *Options) *Tracker {
	db, err := initDB(opts)
	if err != nil {
		log.Fatal(err)
	}
	return &Tracker{
		opts: opts,
		db:   db,
	}
}

// Track starts the idle/Busy tracker in a loop that runs until the context is cancelled
func (t *Tracker) Track(ctx context.Context) {
	t.wg.Add(1)
	defer t.wg.Done()
	defer func(db *sqlx.DB) {
		log.Println("🥫 Close database in " + t.opts.AppDir)
		_ = db.Close()
	}(t.db) // last defer is executed first (LIFO)

	var ist IdleState
	ist.SwitchState() // start in idle mode (idle = true)
	log.Printf("👀 Tracker started in idle mode with auto-idle>=%v interval=%v", t.opts.IdleTolerance, t.opts.CheckInterval)

	var done bool
	for !done {
		select {
		case <-ctx.Done():
			// make sure latest status is written to db, must use a fresh context
			msg := fmt.Sprintf("🛑 Tracker stopped after %v %s time", ist.TimeSinceLastSwitch(), ist.State())
			_ = t.completeRecord(context.Background(), ist.id, msg)
			done = true
		default:
			idleMillis, err := IdleTime(ctx, t.opts.Cmd)
			switch {
			case err != nil:
				log.Println(err.Error())
			case ist.ExceedsIdleTolerance(idleMillis, t.opts.IdleTolerance):
				busySince := ist.TimeSinceLastSwitch()
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter idle mode after %v busy time", ist.Icon(), busySince)
				_ = t.completeRecord(ctx, ist.id, msg)
			case ist.ExceedsCheckTolerance(t.opts.IdleTolerance):
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter idle mode after sleep mode was detected at %s (%v ago)",
					ist.Icon(), ist.lastCheck.Format(time.RFC3339), ist.TimeSinceLastCheck())
				// We have to date back the end of the Busy period to the last known active check
				// Oh, you have to love Go's time and duration handling: https://stackoverflow.com/a/26285835/4292075
				_ = t.completeRecordWithTime(ctx, ist.id, msg, time.Now().Add(ist.TimeSinceLastCheck()*-1))
			case ist.IsBusy(idleMillis, t.opts.IdleTolerance):
				idleSince := ist.TimeSinceLastSwitch()
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter busy mode after %v idle time", ist.Icon(), idleSince)
				ist.id, _ = t.newRecord(ctx, msg)
			}
			t.checkpoint(ist, idleMillis) // outputs current state details if debug is enabled
			ist.lastCheck = time.Now()

			// time.Sleep doesn't react to context cancellation, but context.WithTimeout does
			sleep, cancel := context.WithTimeout(ctx, t.opts.CheckInterval)
			<-sleep.Done()
			cancel()
		}
	}
}

// WaitClose wait for the tracker loop to finish uncommitted work
func (t *Tracker) WaitClose() {
	t.wg.Wait()
}

// checkpoint print debug info on current state
func (t *Tracker) checkpoint(ist IdleState, idleMillis int64) {
	if t.opts.Debug {
		idleD := (time.Duration(idleMillis) * time.Millisecond).Round(time.Second)
		asInfo := ist.String()
		if ist.Busy() {
			asInfo = fmt.Sprintf("%s idleSwitchIn=%v", asInfo, t.opts.IdleTolerance-idleD)
		}
		log.Printf("%s Checkpoint idleTime=%v %s", ist.Icon(), idleD, asInfo)
	}
}

// initDB initializes SQLite DB in local filesystem
func initDB(opts *Options) (*sqlx.DB, error) {
	dbFile := filepath.Join(opts.AppDir, "db.sqlite3")
	log.Printf("🥫 Open database file=%s sqlite=%s", dbFile, sqlite3.SQLITE_VERSION)
	db, err := sqlx.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open db %s: %w", dbFile, err)
	}

	opts.ClientID, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	// drop table if exists t; insert into t values(42), (314);
	var dropStmt string
	if opts.DropCreate {
		dropStmt = "DROP TABLE IF EXISTS track;\n"
	}
	if _, err = db.Exec(dropStmt + initSQL); err != nil {
		return nil, err
	}

	return db, nil
}

// newRecord inserts a new tracking records
func (t *Tracker) newRecord(ctx context.Context, msg string) (int, error) {
	statement, err := t.db.PrepareContext(ctx, `INSERT INTO track(message,client,task,busy_start) VALUES (?,?,?,?) RETURNING id;`)
	if err != nil {
		return 0, err
	}
	var id int
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	task := randomTask()
	err = statement.QueryRowContext(ctx, msg, t.opts.ClientID, task, time.Now().Round(time.Second)).Scan(&id)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("%s task='%s' id=#%d", msg, task, id)
	return id, err
}

// completeRecord finishes the active record using time.Now() as period end
func (t *Tracker) completeRecord(ctx context.Context, id int, msg string) error {
	return t.completeRecordWithTime(ctx, id, msg, time.Now())
}

// completeRecord finishes the active record using the provided datetime as period end
func (t *Tracker) completeRecordWithTime(ctx context.Context, id int, msg string, busyEnd time.Time) error {
	// don't use sql ( busy_end=datetime(CURRENT_TIMESTAMP, 'localtime') ) but set explicitly
	statement, err := t.db.PrepareContext(ctx, `UPDATE track set busy_end=(?),message = message ||' '|| (?) WHERE id=(?) and busy_end IS NULL`)
	if err != nil {
		return err
	}
	res, err := statement.ExecContext(ctx, busyEnd.Round(time.Second), msg, id)
	if err != nil {
		log.Printf("Cannot complete record %d: %v", id, err)
	}
	affected, _ := res.RowsAffected()
	log.Printf("%s id=#%d rowsUpdated=%d", msg, id, affected)
	return err
}

// getRecords retried existing track records for a specific time period
func (t *Tracker) getRecords(ctx context.Context) (map[string][]TrackRecord, error) {
	// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track
	query := `SELECT * FROM track WHERE busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 100`
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
	recMap, err := t.getRecords(ctx)
	if err != nil {
		return err
	}

	// go maps are unsorted, so we have to https://yourbasic.org/golang/sort-map-keys-values/
	keys := make([]string, 0, len(recMap))
	for k := range recMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	_, _ = fmt.Fprintf(w, "\n%s DAILY BILLY IDLE REPORT %s\n", strings.Repeat("-", 30), strings.Repeat("-", 30))
	// Outer Loop: key days (2024-10-04)
	for _, k := range keys {
		// inner loop: track records per day
		recs := recMap[k]
		first := recs[0]
		last := recs[len(recs)-1]
		var spentBusy, spentTotal time.Duration
		for _, r := range recs {
			_, _ = fmt.Fprintln(w, k, r)
			spentBusy += r.Duration()
		}

		_, _ = fmt.Fprintln(w, strings.Repeat("-", 100))
		if last.BusyEnd.Valid {
			spentTotal = last.BusyEnd.Time.Sub(first.BusyStart)
		} else {
			spentTotal = last.BusyStart.Sub(first.BusyStart) // last record not complete, use start time instead
		}
		kitKat := mandatoryBreak(spentBusy)
		_, _ = fmt.Fprintf(w, "%s Total totalTime=%v busyTime=%v busyTimePlus=%v (plus=%v break)\n",
			first.BusyStart.Format("2006-01-02 Mon"),
			spentTotal.Round(time.Minute),
			spentBusy.Round(time.Minute),
			(spentBusy + kitKat).Round(time.Minute), kitKat.Round(time.Minute))
		_, _ = fmt.Fprintln(w, strings.Repeat("=", 100))
		_, _ = fmt.Fprintln(w, "")
	}
	return nil
}

// randomTask returns a task with random creative content :-)
func randomTask() string {
	// r := rand.IntN(3)
	switch rand.IntN(4) {
	case 0:
		return fmt.Sprintf("Drinking a %s %s", gofakeit.BeerStyle(), gofakeit.BeerName())
	case 1:
		return fmt.Sprintf("Driving a %s %s to %s", gofakeit.CarModel(), gofakeit.CarType(), gofakeit.City())
	case 2:
		return fmt.Sprintf("Eating a %s with %s", gofakeit.Dessert(), gofakeit.Fruit())
	case 3:
		return fmt.Sprintf("Building app %s in %s", gofakeit.AppName(), gofakeit.ProgrammingLanguage())
	case 4:
		return fmt.Sprintf("Feeding a %s named %s", gofakeit.Animal(), gofakeit.PetName())
	default:
		return "Doing boring stuff"
	}
}

// mandatoryBreak returns the mandatory break time depending on the total busy time
func mandatoryBreak(d time.Duration) time.Duration {
	switch {
	case d <= 6*time.Hour:
		return 0
	case d <= 6*time.Hour+30*time.Minute:
		return d - 6*time.Hour
	case d <= 9*time.Hour:
		return 30 * time.Minute
	case d <= 9*time.Hour+30*time.Minute:
		return d - 9*time.Hour + 30*time.Minute
	default:
		return 45 * time.Minute
	}
}
