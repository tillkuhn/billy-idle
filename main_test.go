package main

import (
	"os"
	"testing"
	"time"
)

func Test_Tracker(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func(path string) {
		t.Log("Removing " + path)
		err := os.RemoveAll(path)
		if err != nil {
			t.Log(err.Error())
		}
	}(dir)
	dbDirectory = dir // overwrite with tempdir

	checkInterval = 100 * time.Millisecond
	idleAfter = 100 * time.Millisecond
	cmd = "testdata/ioreg-mock.sh"
	go func() { main() }()
	time.Sleep(1 * time.Second)
	c <- os.Interrupt
	t.Log("Test finished")
}
