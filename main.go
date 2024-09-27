package main

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

const (
	interval  = 1 * time.Second
	idleAfter = 5 * time.Second
)

var idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")

// main runs the tracker
func main() {
	started := time.Now()
	idle := false
	fmt.Printf("🐝 Starting in busy mode at %v\n", time.Now().Format(time.Kitchen))
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
			fmt.Printf("💤 Switched to idle after %ds idle time (running since %v)\n", idleMillis/1000, time.Now().Sub(started).Round(time.Second))
		} else if idle == true && idleMillis < idleAfter.Milliseconds() {
			idle = false
			fmt.Printf("🐝 Switched to busy (running since %v)\n", time.Now().Sub(started).Round(time.Second))
		}
		time.Sleep(interval)
	}
}
