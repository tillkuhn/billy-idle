package cmd

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/tillkuhn/billy-idle/pkg/tracker"

	"github.com/stretchr/testify/assert"
)

func TestSum(t *testing.T) {
	tests := map[string]struct {
		args  []string
		out   string
		error string
	}{
		// "missing-args": {
		//	[]string{},
		//	"",
		//	"requires at least 1 arg(s)",
		// },
		"too-many-args": {
			[]string{"1", "2", "3"},
			"",
			"accepts at most 2 arg(s)",
		},
		"invalid-time": {
			[]string{"morning"},
			"",
			"invalid duration",
		},
		"invalid-date": {
			[]string{"4h5m", "last year"},
			"",
			"cannot parse",
		},
	}

	for name, te := range tests {
		t.Run(name, func(t *testing.T) {
			actual := new(bytes.Buffer)
			rootCmd.SetOut(actual)
			rootCmd.SetErr(actual)
			rootCmd.SetArgs(slices.Insert(te.args, 0, punchCmd.Use))
			err := rootCmd.Execute()
			if te.error != "" {
				assert.ErrorContains(t, err, te.error)
			}
			assert.Contains(t, actual.String(), te.out)
		})
	}
}

func Test_PunchReport(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	today := tracker.TruncateDay(time.Now())
	mock.ExpectQuery("SELECT (.*)").
		WillReturnRows(
			mock.NewRows([]string{"day", "busy_secs"}).
				AddRow(today, 3600).
				AddRow(today, 7200),
		)
	mock.ExpectClose()
	tr := tracker.NewWithDB(&punchOpts, sqlxDB)
	err = punchReport(context.Background(), tr)
	assert.NoError(t, err)
}

/*
func Test_ExecuteTrackCommandMissingArg(t *testing.T) {
	assert.ErrorContains(t, rootCmd.Execute(), "expected a duration")
}

func Test_ExecuteTrackCommand(t *testing.T) {
	actual := new(bytes.Buffer)
	rootCmd.SetOut(actual)
	rootCmd.SetErr(actual)
	rootCmd.SetArgs([]string{punchCmd.Use, "2"})
	assert.NoError(t, rootCmd.Execute())
}

*/
