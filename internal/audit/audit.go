// internal/audit/audit.go
package audit

import (
	"sync"

	"go.uber.org/zap"

	"github.com/SamSafonov2025/metrics-tpl/internal/logger"
)

// AuditEvent представляет событие аудита
type AuditEvent struct {
	Timestamp int64    `json:"ts"`         // unix timestamp события
	Metrics   []string `json:"metrics"`    // наименование полученных метрик
	IPAddress string   `json:"ip_address"` // IP адрес входящего запроса
}

// Observer интерфейс наблюдателя (подписчика)
type Observer interface {
	Notify(event AuditEvent) error
	Close() error
}

// Publisher интерфейс издателя (publisher)
type Publisher interface {
	Register(observer Observer)
	Deregister(observer Observer)
	NotifyAll(event AuditEvent)
}

// AuditPublisher реализация publisher для аудита
type AuditPublisher struct {
	mu        sync.RWMutex
	observers []Observer
}

// NewAuditPublisher создает новый publisher
func NewAuditPublisher() *AuditPublisher {
	return &AuditPublisher{
		observers: make([]Observer, 0),
	}
}

// Register регистрирует нового наблюдателя
func (p *AuditPublisher) Register(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

// Deregister удаляет наблюдателя
func (p *AuditPublisher) Deregister(observer Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, obs := range p.observers {
		if obs == observer {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
			break
		}
	}
}

// NotifyAll оповещает всех наблюдателей о событии
func (p *AuditPublisher) NotifyAll(event AuditEvent) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for _, observer := range p.observers {
		// Запускаем в горутине, чтобы не блокировать основной поток
		go func(obs Observer) {
			if err := obs.Notify(event); err != nil {
				logger.GetLogger().Error("Failed to notify audit observer", zap.Error(err))
			}
		}(observer)
	}
}

// Close закрывает всех наблюдателей
func (p *AuditPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, observer := range p.observers {
		if err := observer.Close(); err != nil {
			logger.GetLogger().Error("Failed to close audit observer", zap.Error(err))
		}
	}
	return nil
}
