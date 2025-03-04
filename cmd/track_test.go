package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExecuteTrackerCommandHelp(t *testing.T) {
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)

	rootCmd.SetArgs([]string{trackCmd.Use, "-h"})
	assert.NoError(t, rootCmd.Execute())
	expected := "Starts the tracker in daemon mode"
	assert.Contains(t, actual.String(), expected, "actual is not expected")
}
