package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

const (
	interval   = 1 * time.Second
	idleAfter  = 5 * time.Second
	dateLayout = time.RFC1123
)

var idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")

// main runs the tracker
func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	busyStart := time.Now()
	idle := false
	fmt.Printf("🐝 Starting in busy mode at %v\n", time.Now().Format(dateLayout))
	go func() {
		for {
			cmd := exec.Command("ioreg", "-c", "IOHIDSystem")
			stdout, err := cmd.Output()
			if err != nil {
				log.Fatal(err.Error())
			}

			match := idleMatcher.FindStringSubmatch(string(stdout))
			var idleMillis int64
			if match != nil {
				if i, err := strconv.Atoi(match[1]); err == nil {
					idleMillis = int64(i / 1000000)
				}
			} else {
				log.Fatal("Can't parse HIDIdleTime from output")
			}

			if idle == false && idleMillis >= idleAfter.Milliseconds() {
				idle = true
				fmt.Printf("💤 Switched to idle after %ds idle and %v busy time at %v\n",
					idleMillis/1000, time.Now().Sub(busyStart).Round(time.Second), time.Now().Format(dateLayout))
			} else if idle == true && idleMillis < idleAfter.Milliseconds() {
				idle = false
				busyStart = time.Now()
				fmt.Printf("🐝 Switched to busy mode at %s\n", time.Now().Format(dateLayout))
			}
			time.Sleep(interval)
		}
	}()
	<-c
	fmt.Printf("🐝 Stopped at %v\n", time.Now().Format(dateLayout))

}
