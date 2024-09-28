package tracker

import "time"

type Options struct {
	CheckInterval time.Duration
	ClientID      string
	Cmd           string
	DbDirectory   string
	DropCreate    bool
	Env           string
	IdleAfter     time.Duration
}
