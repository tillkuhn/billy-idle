package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

const (
	dateLayout     = time.RFC1123
	recreateSchema = true // CAUTION !!!!
)

var (
	c             = make(chan os.Signal, 1)
	cmd           = "ioreg"
	clientID      = "default"
	dbDirectory   = "./sqlite"
	checkInterval = 1 * time.Second
	idleAfter     = 3 * time.Second
	idleMatcher   = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")
)

// main runs the tracker
func main() {
	ctx, ctxCancel := context.WithCancel(context.Background())
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer func(db *sql.DB) { _ = db.Close() }(db)
	go func() {
		idle := false
		lastEvent := time.Now()

		id, _ := insertTrack(ctx, db, fmt.Sprintf("🐝 Start tracking in busy mode, idle time kicks in after %vs", idleAfter.Seconds()))
		for {
			idleMillis, err := currentIdleTime(ctx)
			switch {
			case err != nil:
				log.Println(err.Error())
			case !idle && idleMillis >= idleAfter.Milliseconds():
				idle = true
				if err := completeTrack(ctx, db, id); err != nil {
					log.Println(err.Error())
				}
				log.Printf("[%s] 💤 Switched to idle mode after %v of busy time, completing record #%d\n",
					clientID, time.Since(lastEvent).Round(time.Second), id)
				lastEvent = time.Now()
			case idle && idleMillis < idleAfter.Milliseconds():
				idle = false
				id, err = insertTrack(ctx, db, fmt.Sprintf("🐝 Resuming busy mode after %v of idle time, creating new record", time.Since(lastEvent).Round(time.Second)))
				if err != nil {
					log.Println(err.Error())
				}
				lastEvent = time.Now()
			}
			time.Sleep(checkInterval)
		}
	}()
	<-c
	ctxCancel()
	log.Printf("🐝 Stopped at %v\n", time.Now().Format(dateLayout))
}

// currentIdleTime gets the current idle time in milliseconds from the external ioreg command
func currentIdleTime(ctx context.Context) (int64, error) {
	cmd := exec.CommandContext(ctx, cmd, "-c", "IOHIDSystem")
	stdout, err := cmd.Output()
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
	log.Printf("[%s] %s", clientID, msg)
	statement, err := db.PrepareContext(ctx, `INSERT INTO track(message,client) VALUES (?,?) RETURNING id;`)
	if err != nil {
		return 0, err
	}
	var id int
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	err = statement.QueryRowContext(ctx, msg, clientID).Scan(&id)
	if err != nil {
		log.Println(err.Error())
	}
	return id, err
}

// completeTrack completes an existing tracking record by setting the busy_end date
func completeTrack(ctx context.Context, db *sql.DB, id int) error {
	statement, err := db.PrepareContext(ctx, `UPDATE track set busy_end=datetime(CURRENT_TIMESTAMP, 'localtime') WHERE id=(?)`)
	if err != nil {
		return err
	}
	// Golang SQL insert row and get returning ID example: https://gist.github.com/miguelmota/d54814683346c4c98cec432cf99506c0
	_, err = statement.ExecContext(ctx, id)
	return err
}
