package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// Connect открывает пул соединений и инициализирует таблицы.
// Возвращает ошибку вместо log.Fatalf — можно мягко откатиться на memstorage.
func Connect(dsn string) (*pgxpool.Pool, error) {
	dsn = normalizeDSN(dsn)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	// Проверим соединение заранее
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	Pool = pool

	// Инициализация схемы
	if err := initDB(ctx); err != nil {
		pool.Close()
		Pool = nil
		return nil, err
	}

	return Pool, nil
}

// initDB создаёт таблицы, если их нет.
func initDB(ctx context.Context) error {
	if Pool == nil {
		return fmt.Errorf("pool is nil")
	}

	_, err := Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS gauge (
			id    varchar(256) PRIMARY KEY,
			value double precision NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create table gauge: %w", err)
	}

	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS counter (
			id    varchar(256) PRIMARY KEY,
			value BIGINT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create table counter: %w", err)
	}

	return nil
}

// Close безопасно закрывает пул.
func Close() {
	if Pool != nil {
		Pool.Close()
		Pool = nil
	}
}

// normalizeDSN:
// - убирает случайный префикс "//";
// - заменяет schema= на search_path= (рабочий параметр для Postgres);
// - если не указан sslmode, добавляет sslmode=disable (актуально для локальной БД без TLS).
func normalizeDSN(dsn string) string {
	dsn = strings.TrimPrefix(dsn, "//")
	// только первая замена достаточно (обычно параметр один)
	if strings.Contains(dsn, "schema=") && !strings.Contains(dsn, "search_path=") {
		dsn = strings.Replace(dsn, "schema=", "search_path=", 1)
	}
	if !strings.Contains(dsn, "sslmode=") {
		sep := "?"
		if strings.Contains(dsn, "?") {
			sep = "&"
		}
		dsn = dsn + sep + "sslmode=disable"
	}
	return dsn
}
