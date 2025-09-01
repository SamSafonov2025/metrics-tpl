package dbstorage

import (
	"context"
	"errors"
	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	//"github.com/jackc/pgx/v5/pgconn"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
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

func (db *DBStorage) SetGauge(ctx context.Context, metricName string, value float64) error {
	//ctx := context.Background()
	const q = `
		INSERT INTO gauge (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;
	`
	return retryCtx(ctx, func(ctx context.Context) error {
		_, err := db.Pool.Exec(ctx, q, metricName, value)
		return err
	})
}

func (db *DBStorage) IncrementCounter(ctx context.Context, metricName string, value int64) error {
	//ctx := context.Background()
	const q = `
		INSERT INTO counter (id, value)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET value = counter.value + EXCLUDED.value;
	`
	return retryCtx(ctx, func(ctx context.Context) error {
		_, err := db.Pool.Exec(ctx, q, metricName, value)
		return err
	})
}

func (db *DBStorage) GetGauge(ctx context.Context, metricName string) (float64, bool) {
	//ctx := context.Background()
	const q = `SELECT value FROM gauge WHERE id = $1;`

	var v float64
	err := db.Pool.QueryRow(ctx, q, metricName).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false
		}
		log.Printf("GetGauge: query %q: %v", metricName, err)
		return 0, false
	}
	return v, true
}

func (db *DBStorage) GetCounter(ctx context.Context, metricName string) (int64, bool) {
	//ctx := context.Background()
	const q = `SELECT value FROM counter WHERE id = $1;`

	var v int64
	err := db.Pool.QueryRow(ctx, q, metricName).Scan(&v)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false
		}
		log.Printf("GetCounter: query %q: %v", metricName, err)
		return 0, false
	}
	return v, true
}

func (db *DBStorage) GetAllGauges(ctx context.Context) map[string]float64 {
	//ctx := context.Background()
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

func (db *DBStorage) GetAllCounters(ctx context.Context) map[string]int64 {
	//ctx := context.Background()
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

/*
// Пакетная запись метрик с ретраями
func (db *DBStorage) SetMetrics(ctx context.Context, metrics []dto.Metrics) error {
	once := func(ctx context.Context) error {
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
				// skip unknown
			}
		}

		tx, err := db.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		if len(gaugeIDs) > 0 {
			const qGauge = `
				INSERT INTO public.gauge (id, value)
				SELECT * FROM unnest($1::text[], $2::double precision[])
				ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value;
			`
			if _, err := tx.Exec(ctx, qGauge, gaugeIDs, gaugeVals); err != nil {
				return fmt.Errorf("upsert gauge: %w", err)
			}
		}
		if len(counterIDs) > 0 {
			const qCounter = `
				INSERT INTO public.counter (id, value)
				SELECT * FROM unnest($1::text[], $2::bigint[])
				ON CONFLICT (id) DO UPDATE SET value = public.counter.value + EXCLUDED.value;
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

	return retryCtx(ctx, once)
}
*/

func (db *DBStorage) SetMetrics(ctx context.Context, metrics []dto.Metrics) {
	tx, err := db.Pool.Begin(context.Background())
	if err != nil {
		log.Printf("Error starting transaction: %s", err)
		return
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
				log.Fatalf("Unable to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	for _, metric := range metrics {
		if metric.MType == dto.MetricTypeGauge && metric.Value != nil {
			err = db.InsertOrUpdateGauge(ctx, metric.ID, *metric.Value)
			if err != nil {
				log.Printf("Error inserting gauge metric: %v", err)
			}
		} else if metric.MType == dto.MetricTypeCounter && metric.Delta != nil {
			err = db.InsertOrUpdateCounter(ctx, metric.ID, *metric.Delta)
			if err != nil {
				log.Printf("Error inserting counter metric: %v", err)
			}
		} else {
			log.Printf("Unknown metric type or metric value is nil: %s, %s", metric.MType, metric.ID)
		}
	}

	err = tx.Commit(context.Background())
	if err != nil {
		log.Fatalf("Unable to commit transaction: %v", err)
	}
}

func (db *DBStorage) InsertOrUpdateGauge(ctx context.Context, metricID string, value float64) error {
	q := `INSERT INTO gauge (id, value)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE
			SET value = excluded.value;`
	_, err := db.Pool.Exec(ctx, q, metricID, value)
	return err
}

func (db *DBStorage) InsertOrUpdateCounter(ctx context.Context, metricID string, delta int64) error {
	q := `INSERT INTO counter (id, value)
			VALUES ($1, $2)
			ON CONFLICT (id) DO UPDATE
			SET value = public.counter.value + excluded.value;`
	_, err := db.Pool.Exec(ctx, q, metricID, delta)
	return err
}

func (db *DBStorage) StorageType() string {
	return "db"
}

// ------ retry policy ------
var backoffs = []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

func retryCtx(ctx context.Context, fn func(context.Context) error) error {
	for i := 0; i < len(backoffs)+1; i++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if !isRetryablePgErr(err) || i == len(backoffs) {
			return err
		}
		select {
		case <-time.After(backoffs[i]):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func isRetryablePgErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// ретраи только для Class 08 — Connection Exception
		if strings.HasPrefix(pgErr.Code, "08") {
			return true
		}
		// пример из задания: UniqueViolation — не ретраем
		if pgErr.Code == pgerrcode.UniqueViolation {
			return false
		}
	}
	return false
}
