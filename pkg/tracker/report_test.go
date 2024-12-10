package tracker

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Report(t *testing.T) {
	tr, mock := DBMock(t)

	start := time.Now()
	mock.ExpectQuery("SELECT (.*)").
		WillReturnRows(
			mock.NewRows([]string{"id", "busy_start", "busy_end", "task"}).
				AddRow("1", start, start.Add(6*time.Minute), "Having a DejaVu").
				AddRow("2", start, start.Add(3*time.Minute), "Debugging Code but only for s short time").
				AddRow("3", start, nil, "Unfinished business"),
		)
	mock.ExpectClose()
	var output bytes.Buffer
	assert.NoError(t, tr.Report(context.Background(), &output))
	assert.Contains(t, output.String(), "DejaVu")
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
