package tracker

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var idleMatcher = regexp.MustCompile("\"HIDIdleTime\"\\s*=\\s*(\\d+)")

// IdleTime gets the current idle time (HIDIdleTime) in milliseconds from the external ioreg command
// todo optimize by limit depth, e.g. -d 4 should be sufficient ...	-d limit tree to the given depth
func IdleTime(ctx context.Context, cmd string) (int64, error) {
	cmdExec := exec.CommandContext(ctx, cmd, "-d", "4", "-c", "IOHIDSystem")
	stdout, err := cmdExec.Output()
	if err != nil {
		return 0, err
	}
	match := idleMatcher.FindStringSubmatch(string(stdout))
	var t int64
	if match != nil {
		if i, err := strconv.Atoi(match[1]); err == nil {
			t = int64(i) / time.Second.Microseconds()
		}
		return t, err
	}
	return t, fmt.Errorf("%w can't parse HIDIdleTime from output %s", err, string(stdout))
}
