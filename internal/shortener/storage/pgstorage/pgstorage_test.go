package pgstorage

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPgStorage_Ping_WithMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil) // эмулируем успешный ping

	pg := &PgStorage{DB: db}

	err = pg.Ping()
	assert.NoError(t, err, "Ожидалось, что Ping() не вернет ошибку")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}

func TestPgStorage_Ping_Fail_WithMock(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Включаем мониторинг Ping()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPing().WillReturnError(context.DeadlineExceeded) // Ожидаем ошибку пинга

	pg := &PgStorage{DB: db}

	err = pg.Ping()
	assert.Error(t, err, "Ожидалась ошибка из-за тайм-аута базы данных")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидаемые вызовы должны быть выполнены")
}
