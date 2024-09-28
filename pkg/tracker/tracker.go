package tracker

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/tillkuhn/billy-idle/internal/version"
)

var idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")

type Tracker struct {
	opts *Options
	db   *sql.DB
	wg   sync.WaitGroup
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

func (t *Tracker) Track(ctx context.Context) {
	t.wg.Add(1)
	defer t.wg.Done()
	defer func(db *sql.DB) { _ = db.Close() }(t.db)

	var done, idle bool
	lastEvent := time.Now()

	info("🎬 %s tracker started version=%s commit=%s", filepath.Base(os.Args[0]), version.Version, version.GitCommit)
	id, _ := t.insertTrack(ctx, fmt.Sprintf("🐝 Start tracking in busy mode, idle time kicks in after %vs", t.opts.IdleAfter.Seconds()))
	for !done {
		select {
		case <-ctx.Done():
			// make sure latest status is written to db, must use a fresh context
			info("🛑 context cancelled, committing pending busy record %d", id)
			if err := t.completeTrack(context.Background(), id); err != nil {
				info(err.Error())
			}
			done = true
		default:
			idleMillis, err := currentIdleTime(ctx, t.opts.Cmd)
			switch {
			case err != nil:
				info(err.Error())
			case !idle && idleMillis >= t.opts.IdleAfter.Milliseconds():
				idle = true
				info("💤 Entering idle mode after %v of busy time, completing record #%d", time.Since(lastEvent).Round(time.Second), id)
				_ = t.completeTrack(ctx, id)
				lastEvent = time.Now()
			case idle && idleMillis < t.opts.IdleAfter.Milliseconds():
				idle = false
				msg := fmt.Sprintf("🐝 Resuming busy mode after %v of idle time, creating new record", time.Since(lastEvent).Round(time.Second))
				id, _ = t.insertTrack(ctx, msg)
				info(msg + " #" + strconv.Itoa(id))
				lastEvent = time.Now()
			}
			sleep, cancel := context.WithTimeout(ctx, t.opts.CheckInterval)
			log.Printf("Sleeping...")
			<-sleep.Done()
			cancel()
			// time.Sleep(t.opts.CheckInterval)
		}
	}
	info("🛑 tracker stopped")
}

// WaitClose wait for the tracker loop to finish uncommitted work
func (t *Tracker) WaitClose() {
	t.wg.Wait()
}

// initDB initializes SQLite DB in local filesystem
func initDB(opts *Options) (*sql.DB, error) {
	dbFile := filepath.Join(opts.DbDirectory, "db_"+opts.Env)
	info("Using Database %s", dbFile)
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
	if _, err = db.Exec(dropStmt + `
CREATE TABLE IF NOT EXISTS track (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"busy_start" DATETIME NOT NULL DEFAULT (datetime(CURRENT_TIMESTAMP, 'localtime')), 
		"busy_end" DATETIME,
		"client" TEXT,
		"message" TEXT)
`); err != nil {
		return nil, err
	}

	return db, nil
}

// insertTrack inserts a new tracking records
func (t *Tracker) insertTrack(ctx context.Context, msg string) (int, error) {
	statement, err := t.db.PrepareContext(ctx, `INSERT INTO track(message,client) VALUES (?,?) RETURNING id;`)
	if err != nil {
		return 0, err
	}
	var id int
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	err = statement.QueryRowContext(ctx, msg, t.opts.ClientID).Scan(&id)
	if err != nil {
		info(err.Error())
	}
	return id, err
}

// completeTrack completes an existing tracking record by setting the busy_end date
func (t *Tracker) completeTrack(ctx context.Context, id int) error {
	statement, err := t.db.PrepareContext(ctx, `UPDATE track set busy_end=datetime(CURRENT_TIMESTAMP, 'localtime') WHERE id=(?)`)
	if err != nil {
		return err
	}
	_, err = statement.ExecContext(ctx, id)
	if err != nil {
		info(err.Error())
	}
	return err
}

// Input for select:
// select sum(ROUND((JULIANDAY(busy_end) - JULIANDAY(busy_start)) * 86400)) || ' secs' AS total from track

func info(format string, v ...any) {
	log.Printf(format+"\n", v...)
}

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
