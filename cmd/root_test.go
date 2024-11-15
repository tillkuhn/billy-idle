package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultAppRoot(t *testing.T) {
	h, err := os.UserHomeDir()
	assert.NoError(t, err)
	assert.Equal(t, defaultAppRoot(), h+"/.billy-idle")
}

func Test_DefaultEnd(t *testing.T) {
	assert.Equal(t, defaultEnv(), "test")
}
