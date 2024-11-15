package tracker

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

func Test_Insert(t *testing.T) {
	tr, mock := DBMock(t)
	sql1 := `INSERT INTO track(.*)`
	mock.ExpectPrepare(sql1)
	// Error row: https://github.com/DATA-DOG/go-sqlmock/blob/master/rows_test.go#L53
	mock.ExpectQuery(sql1).WithArgs("nur der RWE", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id"}).
			AddRow("42"))
	mock.ExpectClose()
	id, err := tr.newTrackRecord(context.Background(), "nur der RWE")
	assert.NoError(t, err)
	assert.Equal(t, 42, id)
}

func Test_Update(t *testing.T) {
	tr, mock := DBMock(t)
	sql1 := "UPDATE track(.*)"
	mock.ExpectPrepare(sql1)
	// Error row: https://github.com/DATA-DOG/go-sqlmock/blob/master/rows_test.go#L53
	mock.ExpectExec(sql1).WithArgs(sqlmock.AnyArg(), "nur der RWE", 42).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectClose()
	err := tr.completeTrackRecord(context.Background(), 42, "nur der RWE")
	assert.NoError(t, err)
}

func Test_mandatoryBreak(t *testing.T) {
	assert.Equal(t, 0*time.Minute, mandatoryBreak(5*time.Hour))
	assert.Equal(t, 9*time.Minute, mandatoryBreak(6*time.Hour+9*time.Minute))
	assert.Equal(t, 30*time.Minute, mandatoryBreak(6*time.Hour+31*time.Minute))
	assert.Equal(t, 30*time.Minute, mandatoryBreak(8*time.Hour))
	assert.Equal(t, 39*time.Minute, mandatoryBreak(9*time.Hour+9*time.Minute))
	assert.Equal(t, 45*time.Minute, mandatoryBreak(9*time.Hour+18*time.Minute))
}
