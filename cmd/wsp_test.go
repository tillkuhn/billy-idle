package cmd

import (
	"bytes"
	"context"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

// Todo align with Idle test, should be less redundant
func TestWSPStatusError(t *testing.T) {
	var grpcTestPort = 50052 // use different port for test to avoid conflicts
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	wspArgs := []string{"--port", strconv.Itoa(grpcTestPort)}
	rootCmd.SetArgs(slices.Insert(wspArgs, 0, wspCmd.Use))
	// Returns "Error: rpc error: code = DeadlineExceeded desc = context deadline exceeded\nUsage:\n  b
	// if no server
	opts := &tracker.Options{
		Port:     grpcTestPort,
		ClientID: "test",
		AppRoot:  defaultAppRoot(),
	}
	tr := tracker.New(opts)
	go func() {
		if err := tr.ServeGRCP(context.Background()); err != nil {
			t.Log(err)
			t.Fail()
		}
	}()
	// assert.NoError(t, tr.ServeGRCP())
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, actual.String(), "state=busy")
}
