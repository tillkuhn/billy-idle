package cmd

import (
	"bytes"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tillkuhn/billy-idle/pkg/tracker"
)

// Todo align with WSP test, should be less redundant
func TestIdle(t *testing.T) {
	var grpcTestPort = 50054 // use different port for each test to avoid conflicts
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	wspArgs := []string{"--port", strconv.Itoa(grpcTestPort)}
	rootCmd.SetArgs(slices.Insert(wspArgs, 0, idleCmd.Use))
	// Returns "Error: rpc error: code = DeadlineExceeded desc = context deadline exceeded\nUsage:\n  b
	// if no server
	opts := &tracker.Options{
		Port:     grpcTestPort,
		ClientID: "test",
		AppRoot:  defaultAppRoot(),
	}
	tr := tracker.New(opts)
	go func() {
		if err := tr.ServeGRCP(); err != nil {
			t.Log(err)
			t.Fail()
		}
	}()
	// assert.NoError(t, tr.ServeGRCP())
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, actual.String(), "idle until")
}
