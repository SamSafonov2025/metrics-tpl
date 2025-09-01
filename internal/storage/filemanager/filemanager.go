package filemanager

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/interfaces"
)

type StorageInterface = interfaces.Store

type FileManager struct {
	FilePath  string
	Done      chan struct{}
	closeOnce sync.Once
}

func New(filePath string) *FileManager {
	return &FileManager{
		FilePath: filePath,
		Done:     make(chan struct{}),
	}
}

func (fm *FileManager) SaveData(storage StorageInterface) error {
	if fm.FilePath == "" {
		return errors.New("filemanager: empty FilePath")
	}

	// собираем метрики из стораджа
	var out []dto.Metrics
	for k, v := range storage.GetAllCounters() {
		val := v
		out = append(out, dto.Metrics{ID: k, MType: "counter", Delta: &val})
	}
	for k, v := range storage.GetAllGauges() {
		val := v
		out = append(out, dto.Metrics{ID: k, MType: "gauge", Value: &val})
	}

	f, err := os.Create(fm.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(out)
}

func (fm *FileManager) LoadData(storage StorageInterface) error {
	if fm.FilePath == "" {
		return errors.New("filemanager: empty FilePath")
	}

	f, err := os.Open(fm.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var items []dto.Metrics
	dec := json.NewDecoder(f)
	if err := dec.Decode(&items); err != nil {
		return err
	}

	for _, m := range items {
		switch m.MType {
		case "gauge":
			if m.Value == nil {
				continue
			}
			storage.SetGauge(m.ID, *m.Value)

		case "counter":
			if m.Delta == nil {
				continue
			}
			want := *m.Delta
			cur, ok := storage.GetCounter(m.ID)
			var inc int64
			if ok {
				inc = want - cur // доводим до абсолютного занчения
			} else {
				inc = want
			}
			if inc != 0 {
				storage.IncrementCounter(m.ID, inc)
			}
		}
	}
	return nil
}

func (fm *FileManager) RunBackup(interval time.Duration, storage StorageInterface) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = fm.SaveData(storage) // не паникуем на ошибках бэкапа
		case <-fm.Done:
			return
		}
	}
}

func (fm *FileManager) Close(storage StorageInterface) {
	fm.closeOnce.Do(func() {
		// закрываем сигнал завершения один раз
		if fm.Done != nil {
			close(fm.Done)
		}
		// финальный сейв — игнорируем ошибку в тестах
		_ = fm.SaveData(storage)
	})
}
