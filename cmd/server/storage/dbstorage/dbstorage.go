package dbstorage

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBStorage struct {
	Pool *pgxpool.Pool
}

type Gauge struct {
	ID    string  `json:"id"`
	Value float64 `json:"value"`
}

type Counter struct {
	ID    string `json:"id"`
	Value int64  `json:"value"`
}

func (db *DBStorage) SetGauge(metricName string, value float64) {
	ctx := context.Background()
	const q = `
		INSERT INTO gauge (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;
	`
	if _, err := db.Pool.Exec(ctx, q, metricName, value); err != nil {
		log.Printf("SetGauge: failed to upsert %q=%v: %v", metricName, value, err)
	}
}

func (db *DBStorage) IncrementCounter(metricName string, value int64) {
	ctx := context.Background()
	const q = `
		INSERT INTO counter (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = counter.value + EXCLUDED.value;
	`
	if _, err := db.Pool.Exec(ctx, q, metricName, value); err != nil {
		log.Printf("IncrementCounter: failed to upsert %q by %v: %v", metricName, value, err)
	}
}

func (db *DBStorage) GetGauge(metricName string) (float64, bool) {
	ctx := context.Background()
	const q = `SELECT value FROM gauge WHERE id = $1;`

	var v float64
	err := db.Pool.QueryRow(ctx, q, metricName).Scan(&v)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false
		}
		log.Printf("GetGauge: query %q: %v", metricName, err)
		return 0, false
	}
	return v, true
}

func (db *DBStorage) GetCounter(metricName string) (int64, bool) {
	ctx := context.Background()
	const q = `SELECT value FROM counter WHERE id = $1;`

	var v int64
	err := db.Pool.QueryRow(ctx, q, metricName).Scan(&v)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false
		}
		log.Printf("GetCounter: query %q: %v", metricName, err)
		return 0, false
	}
	return v, true
}

func (db *DBStorage) GetAllGauges() map[string]float64 {
	ctx := context.Background()
	const q = `SELECT id, value FROM gauge;`

	rows, err := db.Pool.Query(ctx, q)
	if err != nil {
		log.Printf("GetAllGauges: query: %v", err)
		return nil
	}
	defer rows.Close()

	gauges := make(map[string]float64)
	for rows.Next() {
		var id string
		var v float64
		if err := rows.Scan(&id, &v); err != nil {
			log.Printf("GetAllGauges: scan: %v", err)
			return nil
		}
		gauges[id] = v
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetAllGauges: rows err: %v", err)
		return nil
	}
	return gauges
}

func (db *DBStorage) GetAllCounters() map[string]int64 {
	ctx := context.Background()
	const q = `SELECT id, value FROM counter;`

	rows, err := db.Pool.Query(ctx, q)
	if err != nil {
		log.Printf("GetAllCounters: query: %v", err)
		return nil
	}
	defer rows.Close()

	counters := make(map[string]int64)
	for rows.Next() {
		var id string
		var v int64
		if err := rows.Scan(&id, &v); err != nil {
			log.Printf("GetAllCounters: scan: %v", err)
			return nil
		}
		counters[id] = v
	}
	if err := rows.Err(); err != nil {
		log.Printf("GetAllCounters: rows err: %v", err)
		return nil
	}
	return counters
}
