package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRmInvalidArg(t *testing.T) {
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	// wspArgs := []string{"--port", strconv.Itoa(grpcTestPort)}
	rootCmd.SetArgs([]string{rmCmd.Use, "not-a-string"})
	// Returns "Error: rpc error: code = DeadlineExceeded desc = context deadline exceeded\nUsage:\n  b
	// if no server
	err := rootCmd.Execute()
	assert.ErrorContains(t, err, " invalid syntax")
	// assert.Contains(t, actual.String(), "state=busy")
}
