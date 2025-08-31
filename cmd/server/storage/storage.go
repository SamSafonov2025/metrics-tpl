package storage

import (
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/metrics"
	"github.com/SamSafonov2025/metrics-tpl/cmd/server/storage/memstorage"
	"time"
)

func NewStorage(filePath string, storeInterval time.Duration, restore bool) *memstorage.MemStorage {
	s := &memstorage.MemStorage{
		Counters: make(map[string]metrics.Counter),
		Gauges:   make(map[string]metrics.Gauge),
		//FilePath:  filePath,
		//StoreSync: storeInterval == 0,

		//Done:      make(chan struct{}),
	}

	/*
		if restore {
			err := s.LoadData()
			if err != nil {
				return nil
			}
		}
	*/

	return s
}
