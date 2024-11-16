package tracker

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func Test_UpsertPunchUpdate(t *testing.T) {
	tracker, mock := DBMock(t)
	day := trucateDay(time.Now())
	sql1 := wildcardStatement("UPDATE " + tablePunch + " SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, float64(3600), "test").
		WillReturnResult(sqlmock.NewResult(0, 88))
	err := tracker.UpsertPunchRecord(context.Background(), time.Second*3600, day)
	assert.NoError(t, err)
}

func Test_UpsertPunchInsert(t *testing.T) {
	tracker, mock := DBMock(t)
	day := trucateDay(time.Now())
	sql1 := wildcardStatement("UPDATE " + tablePunch + " SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, float64(3600), "test").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("INSERT INTO "+tablePunch).
		WithArgs(day, float64(3600), "test").
		WillReturnRows(mock.NewRows([]string{"id"}).
			AddRow("44"))
	err := tracker.UpsertPunchRecord(context.Background(), time.Second*3600, day)
	assert.NoError(t, err)
}
