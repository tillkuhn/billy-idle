package tracker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
)

// DBMock returns instances of mockDB (compatible with sql.DB)
// https://github.com/jmoiron/sqlx/issues/204#issuecomment-187641445
func DBMock(t *testing.T) (*Tracker, sqlmock.Sqlmock) {
	mockDB, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tr := &Tracker{
		opts: &Options{ClientID: "test"},
		db:   sqlxDB,
		wg:   sync.WaitGroup{},
	}
	return tr, sqlMock
}

func Test_UpsertBusyUpdate(t *testing.T) {
	tracker, mock := DBMock(t)
	day := time.Now().Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	sql1 := wildcardStatement("UPDATE busy SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, 3600, "test").
		WillReturnResult(sqlmock.NewResult(0, 88))
	err := tracker.UpsertBusyRecord(context.Background(), 3600)
	assert.NoError(t, err)
}

func Test_UpsertBusyInsert(t *testing.T) {
	tracker, mock := DBMock(t)
	day := time.Now().Format("2006-01-02") // Mon Jan 2 15:04:05 MST 2006
	sql1 := wildcardStatement("UPDATE busy SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, 3600, "test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("INSERT INTO busy").
		WithArgs(day, 3600, "test").
		WillReturnRows(mock.NewRows([]string{"id"}).
			AddRow("44"))
	err := tracker.UpsertBusyRecord(context.Background(), 3600)
	assert.NoError(t, err)
}
func wildcardStatement(stmt string) string {
	return stmt + " (.+)"
}
