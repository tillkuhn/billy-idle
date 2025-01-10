package tracker

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_CurrentState(t *testing.T) {
	cs := IdleState{}
	assert.False(t, cs.idle)
	cs.SwitchState()
	assert.True(t, cs.idle)
	assert.Equal(t, "💤", cs.Icon())
	assert.Equal(t, "idle", cs.State())
	assert.False(t, cs.Busy())
	assert.Contains(t, cs.String(), "idle")

	cs.SwitchState()
	assert.True(t, cs.Busy())
	assert.Equal(t, "🐝", cs.Icon())
	assert.Equal(t, "busy", cs.State())
	assert.True(t, cs.ExceedsIdleTolerance(10_000, 5*time.Second))
	assert.False(t, cs.ExceedsIdleTolerance(1_000, 5*time.Second))

	assert.GreaterOrEqual(t, cs.TimeSinceLastSwitch().Milliseconds(), int64(0))
	assert.GreaterOrEqual(t, cs.TimeSinceLastCheck().Milliseconds(), int64(0))
}

func Test_AppDir(t *testing.T) {
	o := Options{Env: "test", AppRoot: "/tmp"}
	assert.Equal(t, filepath.Join("/tmp", "test"), o.AppDir())
}

func Test_String(t *testing.T) {
	tr := TrackRecord{
		ID:        22,
		BusyStart: time.Time{},
		BusyEnd:   sql.NullTime{},
		Message:   "hello",
		Task:      "123",
		Client:    "test",
	}
	assert.Contains(t, tr.String(), "22")
}

// Test add-break-on-top calc
//
// 05:00 -> 0m
// 06:00 -> 0m
// 06:01 -> 1m
// 06:29 -> 29m
// 06:30 -> 30
// 06:55 -> 30
// 09:00 -> 30
// 09:05 -> 35
// 09:15 -> 45 = 10:00
// 09:55 -> 45
