package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Tracker(t *testing.T) {
	ctx := context.Background()
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
	opts := Options{
		checkInterval: 100 * time.Millisecond,
		idleAfter:     100 * time.Millisecond,
		dbDirectory:   dir, // overwrite with tempdir
		cmd:           "testdata/ioreg-mock.sh",
	}
	db, err := initDB(&opts)
	assert.NoError(t, err)
	go func() { tracker(ctx, db, &opts) }()
	time.Sleep(1 * time.Second)
	c <- os.Interrupt
	t.Log("Test finished")
}

func Test_Help(_ *testing.T) {
	os.Args = []string{"hase", "help"}
	main()
}
