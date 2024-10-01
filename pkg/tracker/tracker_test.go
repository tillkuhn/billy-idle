package tracker

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
)

func Test_RandomTask(t *testing.T) {
	for i := 0; i < 4; i++ {
		ta := randomTask()
		t.Log(t.Name() + ": " + ta) // let's make test output more fun
		assert.NotEmpty(t, ta)
	}
}

func Test_State(t *testing.T) {
	a := IdleState{
		idle:       false,
		lastCheck:  time.Now(),
		lastSwitch: time.Now(),
	}
	assert.False(t, a.idle)
	assert.True(t, a.Busy())
	assert.NotEmpty(t, a.String())
}

func Test_Report(t *testing.T) {
	mockDB, sqlMock := DBMock(t)
	tr := Tracker{
		opts: &Options{},
		db:   mockDB,
		wg:   sync.WaitGroup{},
	}
	start := time.Now()
	sqlMock.ExpectQuery("SELECT (.*)").
		WillReturnRows(
			sqlMock.NewRows([]string{"id", "busy_start", "busy_end", "task"}).
				AddRow("1", start, start.Add(5*time.Minute), "Having a DejaVu").
				AddRow("2", start, start.Add(3*time.Minute), "Debugging Code"))
	sqlMock.ExpectClose()
	var output bytes.Buffer
	assert.NoError(t, tr.Report(context.Background(), &output))
	assert.Contains(t, output.String(), "DejaVu")
}

// DBMock returns instances of mockDB (compatible with sql.DB) and sql mock:
// https://github.com/jmoiron/sqlx/issues/204#issuecomment-187641445
func DBMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, sqlMock
}
