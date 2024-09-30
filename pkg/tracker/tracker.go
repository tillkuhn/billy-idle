package tracker

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	sqlite3 "modernc.org/sqlite/lib"
)

// Package embed provides access to Files embedded in the running Go program.
//
//go:embed init-db.sql
var initSQL string

var idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")

type Tracker struct {
	opts *Options
	db   *sql.DB
	wg   sync.WaitGroup
}

type CurrentState struct {
	id         int
	idle       bool
	lastCheck  time.Time
	lastSwitch time.Time
}

func (cs *CurrentState) timeSinceLastSwitch() time.Duration {
	// see https://stackoverflow.com/a/50061223/4292075 discussion how to avoid issues after OS sleep
	// "Try stripping monotonic clock from one of your Time variables using now = now.Round(0) before calling Sub.
	// That should fix it by forcing it to use wall clock."
	return time.Since(cs.lastSwitch.Round(0)).Round(time.Second)
}

func (cs *CurrentState) timeSinceLastCheck() time.Duration {
	if cs.lastCheck.IsZero() {
		return 0
	}
	return time.Since(cs.lastCheck.Round(0)).Round(time.Second)
}

func (cs *CurrentState) switchState() {
	cs.idle = !cs.idle
	cs.lastSwitch = time.Now()
}

func (cs *CurrentState) busy() bool {
	return !cs.idle
}

func (cs *CurrentState) String() string {
	return fmt.Sprintf("idle=%v lastSwitch=%v ago lastCheck=%v ago", cs.idle, cs.timeSinceLastSwitch(), cs.timeSinceLastCheck())
}

func (cs *CurrentState) exceedsIdleTolerance(idleMillis int64, idleTolerance time.Duration) bool {
	return cs.busy() && idleMillis >= idleTolerance.Milliseconds()
}

func (cs *CurrentState) exceedsCheckTolerance(idleTolerance time.Duration) bool {
	return cs.busy() && cs.timeSinceLastCheck() >= idleTolerance
}

func (cs *CurrentState) applicableForBusy(idleMillis int64, idleTolerance time.Duration) bool {
	return cs.idle && idleMillis < idleTolerance.Milliseconds()
}

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

// Track starts the idle/busy tracker in a loop that runs until the context is cancelled
func (t *Tracker) Track(ctx context.Context) {
	t.wg.Add(1)
	defer t.wg.Done()
	defer func(db *sql.DB) {
		log.Println("🥫 Closing database")
		_ = db.Close()
	}(t.db)

	var cs CurrentState
	cs.switchState() // start in idle mode (idle = true)
	log.Printf("👀 Tracker started in idle mode with auto-idle>=%v interval=%v", t.opts.IdleTolerance, t.opts.CheckInterval.Seconds())

	var done bool
	for !done {
		select {
		case <-ctx.Done():
			// make sure latest status is written to db, must use a fresh context
			msg := fmt.Sprintf("🛑 App stopped, after %v busy time", cs.lastSwitch)
			_ = t.completeRecord(context.Background(), cs.id, msg)
			done = true
		default:
			idleMillis, err := currentIdleTime(ctx, t.opts.Cmd)
			switch {
			case err != nil:
				log.Println(err.Error())
			case cs.exceedsIdleTolerance(idleMillis, t.opts.IdleTolerance):
				msg := fmt.Sprintf("💤 Enter idle mode after %v busy time", cs.timeSinceLastSwitch())
				cs.switchState()
				_ = t.completeRecord(ctx, cs.id, msg)
			case cs.exceedsCheckTolerance(t.opts.IdleTolerance):
				msg := fmt.Sprintf("💤 Enter idle mode since sleep mode was detected after lastCheck %v ago", cs.timeSinceLastCheck())
				cs.switchState()
				// Todo: Need to adapt end date, should not use current date
				_ = t.completeRecord(ctx, cs.id, msg)
			case cs.applicableForBusy(idleMillis, t.opts.IdleTolerance):
				msg := fmt.Sprintf("🐝 Enter busy mode after %v idle time", cs.timeSinceLastSwitch())
				cs.switchState()
				cs.id, _ = t.newRecord(ctx, msg)
			}
			t.debugState(cs, idleMillis)
			cs.lastCheck = time.Now()

			// time.Sleep doesn't react to context cancellation, but context.WithTimeout does
			sleep, cancel := context.WithTimeout(ctx, t.opts.CheckInterval)
			<-sleep.Done()
			cancel()
		}
	}
	log.Printf("🛑 Tracker stopped")
}

func (t *Tracker) debugState(as CurrentState, idleMillis int64) {
	if t.opts.Debug {
		idleD := (time.Duration(idleMillis) * time.Millisecond).Round(time.Second)
		asInfo := as.String()
		if as.busy() {
			asInfo = fmt.Sprintf("%s timeToIdleState=%v", asInfo, t.opts.IdleTolerance-idleD)
		}
		log.Printf("🪲  Debug checkpoint idleTime=%v %s",
			idleD, asInfo)
	}
}

// WaitClose wait for the tracker loop to finish uncommitted work
func (t *Tracker) WaitClose() {
	t.wg.Wait()
}

// initDB initializes SQLite DB in local filesystem
func initDB(opts *Options) (*sql.DB, error) {
	dbFile := filepath.Join(opts.DbDirectory, "db_"+opts.Env)
	log.Printf("🥫 Using Database file=%s sqlite=%s", dbFile, sqlite3.SQLITE_VERSION)
	db, err := sql.Open("sqlite", dbFile)
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
	statement, err := t.db.PrepareContext(ctx, `INSERT INTO track(message,client,task) VALUES (?,?,?) RETURNING id;`)
	if err != nil {
		return 0, err
	}
	var id int
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	err = statement.QueryRowContext(ctx, msg, t.opts.ClientID, randomTask()).Scan(&id)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("%s rec=#%d", msg, id)
	return id, err
}

// completeRecord completes an existing tracking record by setting the busy_end date
func (t *Tracker) completeRecord(ctx context.Context, id int, msg string) error {
	statement, err := t.db.PrepareContext(ctx, `UPDATE track set busy_end=datetime(CURRENT_TIMESTAMP, 'localtime'),message = message ||' '|| (?) WHERE id=(?)`)
	if err != nil {
		return err
	}
	_, err = statement.ExecContext(ctx, msg, id)
	if err != nil {
		log.Println(err.Error())
	}
	log.Printf("%s rec=#%d", msg, id)
	return err
}

// Input for select:
// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track

// currentIdleTime gets the current idle time in milliseconds from the external ioreg command
func currentIdleTime(ctx context.Context, cmd string) (int64, error) {
	cmdExec := exec.CommandContext(ctx, cmd, "-c", "IOHIDSystem")
	stdout, err := cmdExec.Output()
	if err != nil {
		return 0, err
	}

	match := idleMatcher.FindStringSubmatch(string(stdout))
	var t int64
	if match != nil {
		if i, err := strconv.Atoi(match[1]); err == nil {
			t = int64(i) / time.Second.Microseconds()
		}
	} else {
		return t, fmt.Errorf("%w can't parse HIDIdleTime from output %s", err, string(stdout))
	}
	return t, nil
}

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
