package dbstorage

import (
	"context"
	"fmt"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
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

// SetMetrics — совместима со старым кодом: логирует ошибку и не паникует.
func (db *DBStorage) SetMetrics(metrics []dto.Metrics) {
	if err := db.setMetrics(context.Background(), metrics); err != nil {
		log.Printf("SetMetrics error: %v", err)
	}
}

// setMetrics — основная логика с контекстом и возвратом ошибки.
func (db *DBStorage) setMetrics(ctx context.Context, metrics []dto.Metrics) error {
	// Разносим по двум батчам: gauge и counter.
	gaugeIDs := make([]string, 0, len(metrics))
	gaugeVals := make([]float64, 0, len(metrics))

	counterIDs := make([]string, 0, len(metrics))
	counterVals := make([]int64, 0, len(metrics))

	for _, m := range metrics {
		switch m.MType {
		case dto.MetricTypeGauge:
			if m.Value == nil {
				continue
			}
			gaugeIDs = append(gaugeIDs, m.ID)
			gaugeVals = append(gaugeVals, *m.Value)

		case dto.MetricTypeCounter:
			if m.Delta == nil {
				continue
			}
			counterIDs = append(counterIDs, m.ID)
			counterVals = append(counterVals, *m.Delta)

		default:
			// Неподдерживаемый тип — просто пропускаем.
			log.Printf("unknown metric type: %s (id=%s)", m.MType, m.ID)
		}
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	// Безопасный откат — неважно, уже был commit или нет.
	defer func() { _ = tx.Rollback(ctx) }()

	// Массовый upsert для gauge
	if len(gaugeIDs) > 0 {
		const qGauge = `
			INSERT INTO public.gauge (id, value)
			SELECT * FROM unnest($1::text[], $2::double precision[])
			ON CONFLICT (id) DO UPDATE
			SET value = EXCLUDED.value;
		`
		if _, err := tx.Exec(ctx, qGauge, gaugeIDs, gaugeVals); err != nil {
			return fmt.Errorf("upsert gauge: %w", err)
		}
	}

	// Массовый upsert для counter (+= delta)
	if len(counterIDs) > 0 {
		const qCounter = `
			INSERT INTO public.counter (id, value)
			SELECT * FROM unnest($1::text[], $2::bigint[])
			ON CONFLICT (id) DO UPDATE
			SET value = public.counter.value + EXCLUDED.value;
		`
		if _, err := tx.Exec(ctx, qCounter, counterIDs, counterVals); err != nil {
			return fmt.Errorf("upsert counter: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func (db *DBStorage) StorageType() string {
	return "db"
}
