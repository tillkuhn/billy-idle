package tracker

import "time"

type Options struct {
	CheckInterval time.Duration
	ClientID      string
	Cmd           string
	DbDirectory   string
	Debug         bool
	DropCreate    bool
	Env           string
	IdleTolerance time.Duration
}
