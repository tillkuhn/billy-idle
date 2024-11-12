package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/tillkuhn/billy-idle/pkg/tracker"

	"github.com/stretchr/testify/assert"
)

func Test_Tracker(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
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
	opts := &tracker.Options{
		CheckInterval: 100 * time.Millisecond,
		IdleTolerance: 150 * time.Millisecond, // fixed mock value is 125ms
		AppDir:        dir,                    // overwrite with tempdir
		Cmd:           "pkg/tracker/ioreg_mock.sh",
		Debug:         true,
		DropCreate:    true,
	}
	tr := tracker.New(opts)
	assert.NoError(t, err)
	go func() { tr.Track(ctx) }()
	time.Sleep(1 * time.Second)
	ctxCancel()
	tr.WaitClose()
	t.Log("Test finished")
}

func Test_Help(_ *testing.T) {
	os.Args = []string{"app", "help"}
	main()
}
