package tracker

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func Test_UpsertPunchUpdate(t *testing.T) {
	tracker, mock := DBMock(t)
	day := TruncateDay(time.Now())
	sql1 := wildcardStatement("UPDATE " + tablePunch + " SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, float64(3600), "test", tracker.opts.RegBusy.Seconds(), "just a test note").
		WillReturnResult(sqlmock.NewResult(0, 88))
	err := tracker.UpsertPunchRecord(context.Background(), time.Second*3600, day, "just a test note")
	assert.NoError(t, err)
}

func Test_UpsertPunchUpdateWithPlanned(t *testing.T) {
	tracker, mock := DBMock(t)
	day := TruncateDay(time.Now())
	sql1 := wildcardStatement("UPDATE " + tablePunch + " SET")
	// mock.ExpectPrepare(sql1)
	planned := time.Second * 7200
	mock.ExpectExec(sql1).WithArgs(day, float64(3600), "test", planned.Seconds(), "test with planned duration").
		WillReturnResult(sqlmock.NewResult(0, 88))
	err := tracker.UpsertPunchRecordWithPlannedDuration(context.Background(), time.Second*3600, day, planned, "test with planned duration")
	assert.NoError(t, err)
}

func Test_UpsertPunchInsert(t *testing.T) {
	tracker, mock := DBMock(t)
	day := TruncateDay(time.Now())
	sql1 := wildcardStatement("UPDATE " + tablePunch + " SET")
	// mock.ExpectPrepare(sql1)
	mock.ExpectExec(sql1).WithArgs(day, float64(3600), "test", tracker.opts.RegBusy.Seconds(), "").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("INSERT "+"INTO "+tablePunch).
		WithArgs(day, float64(3600), "test", tracker.opts.RegBusy.Seconds(), "").
		WillReturnRows(mock.NewRows([]string{"id"}).
			AddRow("44"))
	err := tracker.UpsertPunchRecord(context.Background(), time.Second*3600, day, "")
	assert.NoError(t, err)
}

func Test_SelectPunch(t *testing.T) {
	tr, mock := DBMock(t)

	today := TruncateDay(time.Now())
	mock.ExpectQuery("SELECT (.*)").
		WillReturnRows(
			mock.NewRows([]string{"day", "busy_secs", "planned_secs"}).
				AddRow(today, 3600, 28080).
				AddRow(today, 7200, 28080),
		)
	mock.ExpectClose()
	recs, err := tr.PunchRecords(context.Background(), 0)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(recs))
}

func Test_PunchReport(t *testing.T) {
	tr, mock := DBMock(t)
	var output bytes.Buffer
	tr.opts.Out = &output

	// day, err := time.Parse("2006-01-02 15:04:05", "2024-01-23 13:14:15") // is a tuesday
	// assert.NoError(t, err)
	day := TruncateDay(time.Now())
	mock.ExpectQuery("SELECT (.*)").
		WillReturnRows(
			mock.NewRows([]string{"day", "busy_secs", "planned_secs"}).
				AddRow(day, 3600, 28080).
				AddRow(day, 7200, 28080),
		)
	mock.ExpectClose()
	err := tr.PunchReport(context.Background(), 0)
	assert.NoError(t, err)
	assert.Contains(t, output.String(), day.Weekday().String())
}
