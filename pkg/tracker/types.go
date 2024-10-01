package tracker

import (
	"fmt"
	"time"
)

type Options struct {
	CheckInterval time.Duration
	ClientID      string
	Cmd           string
	AppDir        string
	Debug         bool
	DropCreate    bool
	Env           string
	IdleTolerance time.Duration
}

type TrackRecord struct {
	ID        int       `db:"id"`
	BusyStart time.Time `db:"busy_start"`
	BusyEnd   time.Time `db:"busy_end"`
	Message   string    `db:"message"`
	Task      string    `db:"task"`
	Client    string    `db:"client"`
}

func (t TrackRecord) String() string {
	return fmt.Sprintf("%s %s: Spent %v %s", t.BusyStart.Weekday(), t.BusyStart.Format("15:04:05"), t.Duration(), t.Task)
}

func (t TrackRecord) Duration() time.Duration {
	return t.BusyEnd.Sub(t.BusyStart)
}

type IdleState struct {
	id         int
	idle       bool
	lastCheck  time.Time
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

func (i *IdleState) SwitchState() {
	i.idle = !i.idle
	i.lastSwitch = time.Now()
}

func (i *IdleState) Busy() bool {
	return !i.idle
}

func (i *IdleState) TimeSinceLastSwitch() time.Duration {
	// see https://stackoverflow.com/a/50061223/4292075 discussion how to avoid issues after OS sleep
	// "Try stripping monotonic clock from one of your Time variables using now = now.Round(0) before calling Sub.
	// That should fix it by forcing it to use wall clock."
	return time.Since(i.lastSwitch.Round(0)).Round(time.Second)
}

func (i *IdleState) TimeSinceLastCheck() time.Duration {
	if i.lastCheck.IsZero() {
		return 0
	}
	return time.Since(i.lastCheck.Round(0)).Round(time.Second)
}

func (i *IdleState) ExceedsIdleTolerance(idleMillis int64, idleTolerance time.Duration) bool {
	return i.Busy() && idleMillis >= idleTolerance.Milliseconds()
}

func (i *IdleState) ExceedsCheckTolerance(idleTolerance time.Duration) bool {
	return i.Busy() && i.TimeSinceLastCheck() >= idleTolerance
}

func (i *IdleState) ApplicableForBusy(idleMillis int64, idleTolerance time.Duration) bool {
	return i.idle && idleMillis < idleTolerance.Milliseconds()
}
