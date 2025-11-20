// Package service содержит бизнес-логику для работы с метриками.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
)

// Стандартные ошибки сервиса метрик
var (
	// ErrInvalidType возвращается при указании неподдерживаемого типа метрики.
	// Поддерживаемые типы: "gauge" и "counter".
	ErrInvalidType = errors.New("invalid metric type")

	// ErrNotFound возвращается при попытке получить несуществующую метрику.
	ErrNotFound = errors.New("metric not found")

	// ErrBadValue возвращается при некорректном значении метрики.
	// Например, nil значение для gauge или counter.
	ErrBadValue = errors.New("bad metric value")
)

// MetricsService определяет интерфейс сервиса для работы с метриками.
// Предоставляет методы для обновления, получения и управления метриками.
//
// Все методы принимают context.Context для управления временем выполнения
// и поддержки отмены операций.
type MetricsService interface {
	// Ping проверяет доступность хранилища данных.
	// Возвращает ошибку, если хранилище недоступно.
	Ping(ctx context.Context) error

	// List возвращает все gauge и counter метрики.
	// Возвращает два map: первый для gauge, второй для counter метрик.
	List(ctx context.Context) (gauges map[string]float64, counters map[string]int64, err error)

	// Update обновляет одну метрику.
	// Для counter выполняет инкремент, для gauge устанавливает новое значение.
	// Возвращает обновленную метрику с актуальным значением.
	Update(ctx context.Context, m dto.Metrics) (dto.Metrics, error)

	// Get возвращает метрику по типу и имени.
	// Возвращает ErrNotFound, если метрика не существует.
	// Возвращает ErrInvalidType, если тип метрики некорректен.
	Get(ctx context.Context, typ, id string) (dto.Metrics, error)

	// UpdateBatch атомарно обновляет несколько метрик.
	// Все метрики должны быть валидными, иначе операция отменяется целиком.
	UpdateBatch(ctx context.Context, items []dto.Metrics) error
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
