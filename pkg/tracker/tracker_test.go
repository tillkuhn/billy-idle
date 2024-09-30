package tracker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_RandomTask(t *testing.T) {
	ta := randomTask()
	t.Log(ta)
	assert.NotEmpty(t, ta)
}

func Test_State(t *testing.T) {
	a := CurrentState{
		idle:       false,
		lastCheck:  time.Now(),
		lastSwitch: time.Now(),
	}
	assert.False(t, a.idle)
	assert.True(t, a.busy())
	assert.NotEmpty(t, a.String())
}
