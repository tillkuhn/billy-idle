package tracker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_DayTimeIcon(t *testing.T) {
	ti := map[int]string{
		1:  "💤",
		4:  "💤",
		6:  "☕",
		9:  "☕",
		12: "🌞",
		15: "🌞",
		18: "🌙",
		20: "🌙",
	}
	for hour, icon := range ti {
		tt := time.Date(2021, 8, 15, hour, 30, 0, 0, time.Local)
		assert.Equal(t, icon, DayTimeIcon(tt))
	}
}
