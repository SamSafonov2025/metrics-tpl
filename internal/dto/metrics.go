package dto

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"` // counter (абсолютное значение)
	Value *float64 `json:"value,omitempty"` // gauge
}
