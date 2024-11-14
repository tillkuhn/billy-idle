package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ExecuteTrackCommand(t *testing.T) {
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{punchCmd.Use})
	assert.ErrorContains(t, rootCmd.Execute(), "expected a duration argument")
}
