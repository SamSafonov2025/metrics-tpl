package filemanager

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

type StorageInterface interface {
	SetGauge(metricName string, value float64)
	IncrementCounter(metricName string, value int64)
	GetGauge(metricName string) (float64, bool)
	GetCounter(metricName string) (int64, bool)
	GetAllGauges() map[string]float64
	GetAllCounters() map[string]int64
}

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"` // counter (абсолютное значение)
	Value *float64 `json:"value,omitempty"` // gauge
}

type FileManager struct {
	FilePath string
	Done     chan struct{}
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
	var out []Metrics
	for k, v := range storage.GetAllCounters() {
		val := v
		out = append(out, Metrics{ID: k, MType: "counter", Delta: &val})
	}
	for k, v := range storage.GetAllGauges() {
		val := v
		out = append(out, Metrics{ID: k, MType: "gauge", Value: &val})
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

	var items []Metrics
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
	if fm.Done != nil {
		close(fm.Done)
	}
	_ = fm.SaveData(storage)
}
