package tracker

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/jmoiron/sqlx"
)

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
	return NewWithDB(opts, db)
}

// NewWithDB returns a new Tracker configured with the given Options and DB, good for testing
func NewWithDB(opts *Options, db *sqlx.DB) *Tracker {
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
		log.Println("🥫 Close database in " + t.opts.AppDir())
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
			_ = t.completeTrackRecord(context.Background(), ist.id, msg)
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
				_ = t.completeTrackRecord(ctx, ist.id, msg)
			case ist.ExceedsCheckTolerance(t.opts.IdleTolerance):
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter idle mode after sleep mode was detected at %s (%v ago)",
					ist.Icon(), ist.lastCheck.Format(time.RFC3339), ist.TimeSinceLastCheck())
				// We have to date back the end of the Busy period to the last known active check
				// Oh, you have to love Go's time and duration handling: https://stackoverflow.com/a/26285835/4292075
				_ = t.completeTrackRecordWithTime(ctx, ist.id, msg, time.Now().Add(ist.TimeSinceLastCheck()*-1))
			case ist.IsBusy(idleMillis, t.opts.IdleTolerance):
				idleSince := ist.TimeSinceLastSwitch()
				ist.SwitchState()
				msg := fmt.Sprintf("%s Enter busy mode after %v idle time", ist.Icon(), idleSince)
				ist.id, _ = t.newTrackRecord(ctx, msg)
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

// newTrackRecord inserts a new tracking records
func (t *Tracker) newTrackRecord(ctx context.Context, msg string) (int, error) {
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

// completeTrackRecord finishes the active record using time.Now() as period end
func (t *Tracker) completeTrackRecord(ctx context.Context, id int, msg string) error {
	return t.completeTrackRecordWithTime(ctx, id, msg, time.Now())
}

// completeTrackRecord finishes the active record using the provided datetime as period end
func (t *Tracker) completeTrackRecordWithTime(ctx context.Context, id int, msg string, busyEnd time.Time) error {
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

func randomTask() string {
	// r := rand.IntN(3)
	switch rand.IntN(4) {
	case 0:
		return fmt.Sprintf("Drinking a %s %s with %s", gofakeit.BeerStyle(), gofakeit.BeerName(), gofakeit.BeerAlcohol())
	case 1:
		return fmt.Sprintf("Driving a %s %s to %s", gofakeit.CarType(), gofakeit.CarModel(), gofakeit.City())
	case 2:
		return fmt.Sprintf("Eating a %s topped with %s", gofakeit.Dessert(), gofakeit.Fruit())
	case 3:
		return fmt.Sprintf("Building App %s %s in %s", gofakeit.AppName(), gofakeit.AppVersion(), gofakeit.ProgrammingLanguage())
	case 4:
		return fmt.Sprintf("Feeding a %s named %s with %s", gofakeit.Animal(), gofakeit.PetName(), gofakeit.MinecraftFood())
	default:
		return "Doing boring stuff"
	}
}

// mandatoryBreak returns the mandatory break time depending on the total busy time
// AI generated comment: Here's a breakdown of what the function does:
//
// It takes a duration d as input, which represents the total busy time.
// The function uses a series of switch cases to determine the mandatory break time based on the value of d.
// The cases are:
//
// If d is less than or equal to 6 hours (i.e., d <= 6*time.Hour), then the mandatory break time is 0.
// If d is between 6 hours and 6 hours and 30 minutes (inclusive, i.e., d <= 6*time.Hour+30*time.Minute), then the mandatory break time is the difference between d and 6 hours.
// If d is between 6 hours and 30 minutes and 9 hours (inclusive, i.e., d <= 9*time.Hour), then the mandatory break time is 30 minutes.
// If d is between 9 hours and 9 hours and 30 minutes (inclusive, i.e., d <= 9*time.Hour+30*time.Minute), then the mandatory break time is the difference between d and 9 hours minus 30 minutes,
// Otherwise (i.e., if d is greater than 9 hours and 30 minutes), then the mandatory break time is 45 minutes.
func mandatoryBreak(d time.Duration) time.Duration {
	switch {
	case d <= 6*time.Hour:
		return 0
	case d <= 6*time.Hour+30*time.Minute:
		return d - 6*time.Hour
	case d <= 9*time.Hour:
		return 30 * time.Minute
	case d <= 9*time.Hour+15*time.Minute:
		return d - 9*time.Hour + 30*time.Minute
	default:
		return 45 * time.Minute
	}
}
