package main

import (
	"os"
	"testing"
	"time"
)

func Test_Tracker(t *testing.T) {
	checkInterval = 100 * time.Millisecond
	idleAfter = 200 * time.Millisecond
	go func() { main() }()
	time.Sleep(500 * time.Millisecond)
	c <- os.Interrupt
}
