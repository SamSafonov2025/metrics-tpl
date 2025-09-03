// internal/service/metrics.go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
)

var (
	ErrInvalidType = errors.New("invalid metric type")
	ErrNotFound    = errors.New("metric not found")
	ErrBadValue    = errors.New("bad metric value")
)

type MetricsService interface {
	Ping(ctx context.Context) error
	List(ctx context.Context) (gauges map[string]float64, counters map[string]int64, err error)
	Update(ctx context.Context, m dto.Metrics) (dto.Metrics, error) // одиночная операция
	Get(ctx context.Context, typ, id string) (dto.Metrics, error)   // чтение одной метрики
	UpdateBatch(ctx context.Context, items []dto.Metrics) error     // батч/транзакция
}

type metricsService struct {
	repo    interfaces.Store
	timeout time.Duration
	pinger  func(ctx context.Context) error // абстракция ping; можно внедрить postgres или заглушку
}

func NewMetricsService(repo interfaces.Store, timeout time.Duration, pinger func(ctx context.Context) error) MetricsService {
	return &metricsService{repo: repo, timeout: timeout, pinger: pinger}
}

func (s *metricsService) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, s.timeout)
}

func (s *metricsService) Ping(ctx context.Context) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	if s.pinger != nil {
		return s.pinger(ctx)
	}
	return nil
}

func (s *metricsService) List(ctx context.Context) (map[string]float64, map[string]int64, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	return s.repo.GetAllGauges(ctx), s.repo.GetAllCounters(ctx), nil
}

func (s *metricsService) Update(ctx context.Context, m dto.Metrics) (dto.Metrics, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	switch m.MType {
	case "gauge":
		if m.Value == nil {
			return m, ErrBadValue
		}
		if err := s.repo.SetGauge(ctx, m.ID, *m.Value); err != nil {
			return m, err
		}
		if v, ok := s.repo.GetGauge(ctx, m.ID); ok {
			m.Value = &v
		}
	case "counter":
		if m.Delta == nil {
			return m, ErrBadValue
		}
		if err := s.repo.IncrementCounter(ctx, m.ID, *m.Delta); err != nil {
			return m, err
		}
		if v, ok := s.repo.GetCounter(ctx, m.ID); ok {
			m.Delta = &v
		}
	default:
		return m, ErrInvalidType
	}
	return m, nil
}

func (s *metricsService) Get(ctx context.Context, typ, id string) (dto.Metrics, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	switch typ {
	case "gauge":
		if v, ok := s.repo.GetGauge(ctx, id); ok {
			return dto.Metrics{ID: id, MType: "gauge", Value: &v}, nil
		}
		return dto.Metrics{}, ErrNotFound
	case "counter":
		if v, ok := s.repo.GetCounter(ctx, id); ok {
			return dto.Metrics{ID: id, MType: "counter", Delta: &v}, nil
		}
		return dto.Metrics{}, ErrNotFound
	default:
		return dto.Metrics{}, ErrInvalidType
	}
}

func (s *metricsService) UpdateBatch(ctx context.Context, items []dto.Metrics) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	// Валидация списка (типы/поля)
	for _, it := range items {
		if it.MType == "gauge" && it.Value == nil {
			return ErrBadValue
		}
		if it.MType == "counter" && it.Delta == nil {
			return ErrBadValue
		}
		if it.MType != "gauge" && it.MType != "counter" {
			return ErrInvalidType
		}
	}
	// Делегируем атомарность в репозиторий (транзакция в БД / единый блок в памяти/файле)
	return s.repo.SetMetrics(ctx, items)
}
