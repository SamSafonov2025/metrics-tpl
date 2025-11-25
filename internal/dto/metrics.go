// Package dto содержит объекты передачи данных (Data Transfer Objects) для метрик.
package dto

// Metrics представляет структуру данных для передачи метрики между клиентом и сервером.
// Поддерживает два типа метрик: gauge (вещественные значения) и counter (целочисленные счетчики).
//
// Поля Delta и Value являются указателями, чтобы различать отсутствие значения (nil) от нулевого значения.
//
// Пример для gauge метрики:
//
//	value := 123.45
//	m := Metrics{ID: "temperature", MType: "gauge", Value: &value}
//
// Пример для counter метрики:
//
//	delta := int64(10)
//	m := Metrics{ID: "requests", MType: "counter", Delta: &delta}
type Metrics struct {
	// ID содержит уникальное имя метрики
	ID string `json:"id"`
	// MType определяет тип метрики: "gauge" или "counter"
	MType string `json:"type"`
	// Delta содержит значение для counter метрик (абсолютное значение счетчика)
	Delta *int64 `json:"delta,omitempty"`
	// Value содержит значение для gauge метрик (вещественное число)
	Value *float64 `json:"value,omitempty"`
}
