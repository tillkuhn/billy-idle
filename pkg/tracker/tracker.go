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
				msg := fmt.Sprintf("%s Enter idle mode after %v Busy time", ist.Icon(), ist.TimeSinceLastSwitch())
				ist.SwitchState()
				_ = t.completeRecord(ctx, ist.id, msg)
			case ist.ExceedsCheckTolerance(t.opts.IdleTolerance):
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter idle mode after sleep mode was detected at %s (%v ago)",
					ist.Icon(), ist.lastCheck.Format(time.RFC3339), ist.TimeSinceLastCheck())
				// We have to date back the end of the Busy period to the last known active check
				// Oh, you have to love Go's time and duration handling: https://stackoverflow.com/a/26285835/4292075
				_ = t.completeRecordWithTime(ctx, ist.id, msg, time.Now().Add(ist.TimeSinceLastCheck()*-1))
			case ist.ApplicableForBusy(idleMillis, t.opts.IdleTolerance):
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter busy mode after %v idle time", ist.Icon(), ist.TimeSinceLastSwitch())
				ist.id, _ = t.newRecord(ctx, msg)
			}
			t.checkpoint(ist, idleMillis)
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
	err = statement.QueryRowContext(ctx, msg, t.opts.ClientID, randomTask(), time.Now().Round(time.Second)).Scan(&id)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("%s rec=#%d", msg, id)
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
		log.Println(err.Error())
	}
	affected, _ := res.RowsAffected()
	log.Printf("%s rec=#%d rowsUpdated=%d", msg, id, affected)
	return err
}

// Report experimental report for time tracking apps
func (t *Tracker) Report(ctx context.Context, w io.Writer) error {
	var records []TrackRecord
	// query := `SELECT * FROM track WHERE project_id=$1 AND branch IN ('main','master') ORDER BY id DESC LIMIT 1`
	query := `SELECT * FROM track WHERE busy_end IS NOT NULL and busy_start >= DATE('now', '-7 days') ORDER BY busy_start LIMIT 100`
	// We could use get since we expect a single result, but this would return an error if nothing is found
	// which is a likely use case
	if err := t.db.SelectContext(ctx, &records, query /*, args*/); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "\n%s DAILY BILLY IDLE REPORT %s\n", strings.Repeat("-", 30), strings.Repeat("-", 30))
	var spent time.Duration
	for _, r := range records {
		_, _ = fmt.Fprintln(w, r)
		spent += r.Duration()
	}
	kitKat := 30 * time.Minute
	_, _ = fmt.Fprintln(w, strings.Repeat("-", 84))
	_, _ = fmt.Fprintf(w, "Total time spent: %v (net) / %v (with %v kitKat break)\n",
		spent.Round(time.Minute), (spent + kitKat).Round(time.Minute), kitKat.Round(time.Minute))
	_, _ = fmt.Fprintln(w, strings.Repeat("-", 84))
	return nil
}

// Input for select func
// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track

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
