package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DefaultAppDir(t *testing.T) {
	assert.NotEmpty(t, defaultAppDir("test"))
}
