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

const (
	recreateSchema = true // CAUTION !!!!
)

var (
	c           = make(chan os.Signal, 1)
	clientID    = "default"
	dbDirectory = "./sqlite"
	idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")
)

type Options struct {
	checkInterval time.Duration
	idleAfter     time.Duration
	cmd           string
}

// main runs the tracker
func main() {
	var trackerWG sync.WaitGroup
	var opts Options
	flag.DurationVar(&opts.checkInterval, "interval", 2*time.Second, "Interval to check for idle time")
	flag.DurationVar(&opts.idleAfter, "idle", 10*time.Second, "Max time before client is considered idle")
	flag.StringVar(&opts.cmd, "cmd", "ioreg", "Command to retrieve HIDIdleTime")
	if len(os.Args) > 1 && os.Args[1] == "help" {
		flag.PrintDefaults()
		return
	}
	flag.Parse()

	ctx, ctxCancel := context.WithCancel(context.Background())
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	db, err := initDB()
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
	info("🛑 Received Signal %v\n", sig)
	ctxCancel()
	trackerWG.Wait()
}

func tracker(ctx context.Context, db *sql.DB, opts *Options) {
	idle := false
	lastEvent := time.Now()

	info("🎬 %s tracker started version=%s commit=%s", filepath.Base(os.Args[0]), version.Version, version.GitCommit)
	id, _ := insertTrack(ctx, db, fmt.Sprintf("🐝 Start tracking in busy mode, idle time kicks in after %vs", opts.idleAfter.Seconds()))
loop:
	for {
		select {
		case <-ctx.Done():
			// make sure latest status is written to db, must use a fresh context
			if err := completeTrack(context.Background(), db, id); err != nil {
				info(err.Error())
			}
			break loop
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
func initDB() (*sql.DB, error) {
	fn := filepath.Join(dbDirectory, "db")

	db, err := sql.Open("sqlite", fn)
	if err != nil {
		return nil, err
	}

	clientID, err = os.Hostname()
	if err != nil {
		return nil, err
	}

	// drop table if exists t; insert into t values(42), (314);
	var dropStmt string
	if recreateSchema {
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

func info(format string, v ...any) {
	log.Printf("["+clientID+"] "+format+"\n", v...)
}
