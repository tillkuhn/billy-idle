package tracker

import (
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// DBMock returns instances of mockDB (compatible with sql.DB)
// https://github.com/jmoiron/sqlx/issues/204#issuecomment-187641445
func DBMock(t *testing.T) (*Tracker, sqlmock.Sqlmock) {
	mockDB, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	tr := NewWithDB(&Options{ClientID: "test"}, sqlxDB)
	return tr, sqlMock
}

func wildcardStatement(stmt string) string {
	return stmt + " (.+)"
}
