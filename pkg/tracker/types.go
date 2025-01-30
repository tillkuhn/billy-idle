package tracker

import (
	"database/sql"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// Options to configure the Tracker
type Options struct {
	CheckInterval time.Duration
	ClientID      string
	Cmd           string
	AppRoot       string
	Debug         bool
	DropCreate    bool
	Env           string
	IdleTolerance time.Duration
	MinBusy       time.Duration
	MaxBusy       time.Duration
	RegBusy       time.Duration
	Out           io.Writer
	// Port for gRPC Communication
	Port int
}

func (o Options) AppDir() string {
	return filepath.Join(o.AppRoot, o.Env)
}

// TrackRecord representation of a database record
type TrackRecord struct {
	ID        int          `db:"id"`
	BusyStart time.Time    `db:"busy_start"`
	BusyEnd   sql.NullTime `db:"busy_end"`
	Message   string       `db:"message"`
	Task      string       `db:"task"`
	Client    string       `db:"client"`
}

type PunchRecord struct {
	Day         time.Time `db:"day"`
	BusySecs    float64   `db:"busy_secs"`
	PlannedSecs float64   `db:"planned_secs"`
}

// String returns a string representation of the TrackRecord
func (t TrackRecord) String() string {
	// 2025-10-09 Wed 09:01:30 #144 Spent 3h31m20s Eating a Frosted rhubarb cookies topped with Honeydew until 12:32PM
	return fmt.Sprintf("%s → %v: %s#%d",
		t.BusyStart.Format("15:04:05"), t.BusyEnd, t.Task, t.ID)
}

// Duration returns the duration of a punch record, calculated as the difference between the end and start times.
func (t TrackRecord) Duration() time.Duration {
	if t.BusyEnd.Valid {
		return t.BusyEnd.Time.Sub(t.BusyStart)
	}
	return 0
}

// IdleState represents the current state
type IdleState struct {
	// id current record id for the active record
	id int
	// idle if false, client is considered busy
	idle bool
	// lastCheck holds the timestamp when the last idle check took place
	lastCheck time.Time
	// lastSwitch holds the timestamp when state last switch from idle to busy or vice versa
	lastSwitch time.Time
}

// State returns a single string indicating the current state, either idle or busy
func (i *IdleState) State() string {
	if i.idle {
		return "idle"
	}
	return "busy"
}

// Icon returns an emoji that represents the current state
func (i *IdleState) Icon() string {
	if i.idle {
		return "💤"
	}
	return "🐝"
}

// String returns a string summary representing all important state fields
func (i *IdleState) String() string {
	return fmt.Sprintf("state=%s lastSwitch=%v ago lastCheck=%v ago", i.State(), i.TimeSinceLastSwitch(), i.TimeSinceLastCheck())
}

// SwitchState switch between idly and busy
func (i *IdleState) SwitchState() {
	i.idle = !i.idle
	i.lastSwitch = time.Now()
}

// Busy convenience method for ! idle
func (i *IdleState) Busy() bool {
	return !i.idle
}

// TimeSinceLastSwitch returns the duration since the last state switch
func (i *IdleState) TimeSinceLastSwitch() time.Duration {
	// see https://stackoverflow.com/a/50061223/4292075 discussion how to avoid issues after OS sleep
	// "Try stripping monotonic clock from one of your Time variables using now = now.Round(0) before calling Sub.
	// That should fix it by forcing it to use wall clock."
	return time.Since(i.lastSwitch.Round(0)).Round(time.Second)
}

// TimeSinceLastCheck returns the duration since the last idle checkpoint
func (i *IdleState) TimeSinceLastCheck() time.Duration {
	if i.lastCheck.IsZero() {
		return 0
	}
	return time.Since(i.lastCheck.Round(0)).Round(time.Second)
}

// ExceedsIdleTolerance returns true if the idle time is greater than the tolerated duration
func (i *IdleState) ExceedsIdleTolerance(idleMillis int64, idleTolerance time.Duration) bool {
	return i.Busy() && idleMillis >= idleTolerance.Milliseconds()
}

// ExceedsCheckTolerance returns true if the duration since the last checkpoint is greater than the tolerated duration
func (i *IdleState) ExceedsCheckTolerance(idleTolerance time.Duration) bool {
	return i.Busy() && i.TimeSinceLastCheck() >= idleTolerance
}

// IsBusy returns true if current idle time is lower than the idle tolerance
func (i *IdleState) IsBusy(idleMillis int64, idleTolerance time.Duration) bool {
	return i.idle && idleMillis < idleTolerance.Milliseconds()
}
