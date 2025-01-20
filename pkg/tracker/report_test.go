package tracker

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Report(t *testing.T) {
	// https://go.dev/wiki/TableDrivenTests
	tests := map[string]struct {
		duration time.Duration
		result   string
	}{
		"lessThanReg": {
			duration: time.Hour * 6,
			result:   "expected to be busy for another",
		},
		"overReg": {
			duration: time.Hour * 9,
			result:   "longer than the expected",
		},
		"overMax": {
			duration: time.Hour * 11,
			result:   "capped",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tr, mock := DBMock(t)
			start := time.Now()
			mock.ExpectQuery("SELECT (.*)").
				WillReturnRows(
					mock.NewRows([]string{"id", "busy_start", "busy_end", "task"}).
						AddRow("1", start, start.Add(5*time.Minute), "Having a DejaVu").
						AddRow("2", start.Add(10*time.Minute), start.Add(10*time.Minute), "Debugging Code but only for s short time").
						AddRow("2", start.Add(25*time.Minute), start.Add(test.duration), "Keeping test cases green"),
					// AddRow("3", start.Add(50*time.Minute), nil, "Unfinished business"),
				)
			mock.ExpectClose()
			var output bytes.Buffer
			tr.opts.Out = &output
			tr.opts.MaxBusy = 10 * time.Hour
			tr.opts.RegBusy = 7*time.Hour + 48*time.Minute

			err := tr.Report(context.Background(), &output)
			assert.NoError(t, err)
			assert.Contains(t, output.String(), "DejaVu")
			assert.Contains(t, output.String(), test.result)
			t.Log(output.String())
		})
	}
}

func Test_FormatDuration(t *testing.T) {
	assert.Equal(t, "-59m", FDur(-59*time.Minute))
	assert.Equal(t, "2h", FDur(2*time.Hour))
	assert.Equal(t, "-2h", FDur(-2*time.Hour))
	assert.Equal(t, "2h5m", FDur(2*time.Hour+5*time.Minute))
	assert.Equal(t, "5m", FDur(5*time.Minute))
	assert.Equal(t, "2h", FDur(2*time.Hour))
	assert.Equal(t, "0m", FDur(0*time.Minute))
	assert.Equal(t, "-3h7m", FDur(-3*time.Hour+7*time.Minute*-1))
}
