package storage

import (
	"log"
	"sync"

	metrics2 "github.com/SamSafonov2025/metrics-tpl/internal/metrics"
	"github.com/SamSafonov2025/metrics-tpl/internal/postgres"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SamSafonov2025/metrics-tpl/internal/config"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage/dbstorage"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage/filemanager"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage/memstorage"
)

// Общий интерфейс стораджа (совпадает с тем, что использует FileManager)
//type Store = filemanager.StorageInterface

var (
	once     sync.Once
	curFM    *filemanager.FileManager
	curStore interfaces.Store
)

// NewStorage — единая точка инициализации стораджа + FileManager из конфигурации.
// Выбор backend-а:
//   - если cfg.Database непустой и подключение успешно → dbstorage (PostgreSQL),
//   - иначе → memstorage.
//
// Также: делает restore (если cfg.Restore) и запускает бэкап (если cfg.StoreInterval>0),
// либо выполняет разовый сейв на старте, если задан cfg.FileStoragePath и интервал = 0.
//
// Вызывайте один раз. Повторные вызовы вернут уже созданный store.
func NewStorage(cfg *config.ServerConfig) interfaces.Store {
	once.Do(func() {
		// Выбор реализации ТОЛЬКО по cfg.Database (никакой магии от чужого Pool)
		if cfg.Database != "" {
			if postgres.Pool == nil {
				if _, err := postgres.Connect(cfg.Database); err != nil {
					// логируем и мягко откатываемся на in-memory
					log.Printf("storage: postgres connect failed, fallback to memstorage: %v", err)
				}
			}
			if postgres.Pool != nil {
				curStore = NewDB(postgres.Pool)
			} else {
				curStore = NewMem()
			}
		} else {
			// Явно in-memory для тестов/локалки без БД,
			// даже если где-то уже инициализировали глобальный Pool.
			curStore = NewMem()
		}

		// FileManager + restore/backup
		curFM = filemanager.New(cfg.FileStoragePath)
		if cfg.Restore {
			_ = curFM.LoadData(curStore)
		}
		if cfg.StoreInterval > 0 {
			go curFM.RunBackup(cfg.StoreInterval, curStore)
		} else if cfg.FileStoragePath != "" {
			_ = curFM.SaveData(curStore)
		}
	})
	return curStore
}

// Close — аккуратно завершает FileManager и соединение с БД.
// Рекомендуется вызывать в main: defer storage.Close()
func Close() {
	if curFM != nil && curStore != nil {
		curFM.Close(curStore)
	}
	if postgres.Pool != nil {
		postgres.Close()
	}
}

// --- Вспомогательные фабрики (если нужно напрямую) ---

func NewMem() interfaces.Store {
	return &memstorage.MemStorage{
		Counters: make(map[string]metrics2.Counter),
		Gauges:   make(map[string]metrics2.Gauge),
	}
}

func NewDB(pool *pgxpool.Pool) interfaces.Store {
	return &dbstorage.DBStorage{Pool: pool}
}

func TestReset() {
	if curFM != nil && curStore != nil {
		curFM.Close(curStore)
	}
	curFM = nil
	curStore = nil
	once = sync.Once{}

	// важно: чтобы следующий NewStorage с cfg.Database=="" точно выбрал memstorage
	postgres.Close() // внутри выставляет Pool = nil
}
