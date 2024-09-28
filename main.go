package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/tillkuhn/billy-idle/internal/version"

	_ "modernc.org/sqlite"
)

var (
	c           = make(chan os.Signal, 1)
	clientID    = "default"
	idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")
)

type Options struct {
	checkInterval time.Duration
	cmd           string
	dbDirectory   string
	dropCreate    bool
	env           string
	idleAfter     time.Duration
}

// main runs the tracker
func main() {
	var trackerWG sync.WaitGroup
	var opts Options
	flag.StringVar(&opts.cmd, "cmd", "ioreg", "Command to retrieve HIDIdleTime")
	flag.StringVar(&opts.dbDirectory, "db-dir", "./sqlite", "SQLite directory")
	flag.BoolVar(&opts.dropCreate, "drop-create", false, "Drop and re-create db schema (CAUTION!)")
	flag.StringVar(&opts.env, "env", "default", "Environment")
	flag.DurationVar(&opts.checkInterval, "interval", 2*time.Second, "Interval to check for idle time")
	flag.DurationVar(&opts.idleAfter, "idle", 10*time.Second, "Max time before client is considered idle")
	if len(os.Args) > 1 && os.Args[1] == "help" {
		flag.PrintDefaults()
		return
	}
	flag.Parse()

	ctx, ctxCancel := context.WithCancel(context.Background())
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	db, err := initDB(&opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)
	trackerWG.Add(1)
	go func() {
		tracker(ctx, db, &opts)
		trackerWG.Done()
	}()
	sig := <-c
	info("🛑 Received Signal %v", sig)
	ctxCancel()
	trackerWG.Wait()
}

func tracker(ctx context.Context, db *sql.DB, opts *Options) {
	var done, idle bool
	lastEvent := time.Now()

	info("🎬 %s tracker started version=%s commit=%s", filepath.Base(os.Args[0]), version.Version, version.GitCommit)
	id, _ := insertTrack(ctx, db, fmt.Sprintf("🐝 Start tracking in busy mode, idle time kicks in after %vs", opts.idleAfter.Seconds()))
	for !done {
		select {
		case <-ctx.Done():
			// make sure latest status is written to db, must use a fresh context
			if err := completeTrack(context.Background(), db, id); err != nil {
				info(err.Error())
			}
			done = true
		default:
			idleMillis, err := currentIdleTime(ctx, opts.cmd)
			switch {
			case err != nil:
				info(err.Error())
			case !idle && idleMillis >= opts.idleAfter.Milliseconds():
				idle = true
				info("💤 Entering idle mode after %v of busy time, completing record #%d", time.Since(lastEvent).Round(time.Second), id)
				_ = completeTrack(ctx, db, id)
				lastEvent = time.Now()
			case idle && idleMillis < opts.idleAfter.Milliseconds():
				idle = false
				msg := fmt.Sprintf("🐝 Resuming busy mode after %v of idle time, creating new record", time.Since(lastEvent).Round(time.Second))
				id, _ = insertTrack(ctx, db, msg)
				info(msg + " #" + strconv.Itoa(id))
				lastEvent = time.Now()
			}
			time.Sleep(opts.checkInterval)
		}
	}
	info("🛑 tracker stopped")
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

// initDB initializes SQLite DB in local filesystem
func initDB(opts *Options) (*sql.DB, error) {
	dbFile := filepath.Join(opts.dbDirectory, "db_"+opts.env)
	info("Using Database %s", dbFile)
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("cannot open db %s: %w", dbFile, err)
	}

	clientID, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	// drop table if exists t; insert into t values(42), (314);
	var dropStmt string
	if opts.dropCreate {
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
func insertTrack(ctx context.Context, db *sql.DB, msg string) (int, error) {
	statement, err := db.PrepareContext(ctx, `INSERT INTO track(message,client) VALUES (?,?) RETURNING id;`)
	if err != nil {
		return 0, err
	}
	var id int
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	err = statement.QueryRowContext(ctx, msg, clientID).Scan(&id)
	if err != nil {
		info(err.Error())
	}
	return id, err
}

// completeTrack completes an existing tracking record by setting the busy_end date
func completeTrack(ctx context.Context, db *sql.DB, id int) error {
	statement, err := db.PrepareContext(ctx, `UPDATE track set busy_end=datetime(CURRENT_TIMESTAMP, 'localtime') WHERE id=(?)`)
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
	log.Printf("["+clientID+"] "+format+"\n", v...)
}
