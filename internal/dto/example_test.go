package dto_test

import (
	"encoding/json"
	"fmt"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
)

// Example демонстрирует создание и использование метрик разных типов.
func Example() {
	// Создание gauge метрики
	gaugeValue := 23.5
	gauge := dto.Metrics{
		ID:    "temperature",
		MType: "gauge",
		Value: &gaugeValue,
	}

	// Создание counter метрики
	counterDelta := int64(10)
	counter := dto.Metrics{
		ID:    "requests",
		MType: "counter",
		Delta: &counterDelta,
	}

	// Сериализация в JSON
	gaugeJSON, _ := json.Marshal(gauge)
	counterJSON, _ := json.Marshal(counter)

	fmt.Printf("Gauge JSON: %s\n", gaugeJSON)
	fmt.Printf("Counter JSON: %s\n", counterJSON)

	// Output:
	// Gauge JSON: {"id":"temperature","type":"gauge","value":23.5}
	// Counter JSON: {"id":"requests","type":"counter","delta":10}
}

// ExampleMetrics_gauge демонстрирует создание gauge метрики.
func ExampleMetrics_gauge() {
	value := 42.7
	metric := dto.Metrics{
		ID:    "cpu_usage",
		MType: "gauge",
		Value: &value,
	}

	fmt.Printf("ID: %s\n", metric.ID)
	fmt.Printf("Type: %s\n", metric.MType)
	fmt.Printf("Value: %.1f\n", *metric.Value)

	// Output:
	// ID: cpu_usage
	// Type: gauge
	// Value: 42.7
}

// ExampleMetrics_counter демонстрирует создание counter метрики.
func ExampleMetrics_counter() {
	delta := int64(100)
	metric := dto.Metrics{
		ID:    "total_requests",
		MType: "counter",
		Delta: &delta,
	}

	fmt.Printf("ID: %s\n", metric.ID)
	fmt.Printf("Type: %s\n", metric.MType)
	fmt.Printf("Delta: %d\n", *metric.Delta)

	// Output:
	// ID: total_requests
	// Type: counter
	// Delta: 100
}

// ExampleMetrics_jsonSerialization демонстрирует сериализацию и десериализацию метрик.
func ExampleMetrics_jsonSerialization() {
	// Создаем метрику
	value := 75.3
	original := dto.Metrics{
		ID:    "memory_usage",
		MType: "gauge",
		Value: &value,
	}

	// Сериализуем в JSON
	jsonData, _ := json.Marshal(original)
	fmt.Printf("JSON: %s\n", jsonData)

	// Десериализуем из JSON
	var restored dto.Metrics
	json.Unmarshal(jsonData, &restored)

	fmt.Printf("Restored ID: %s\n", restored.ID)
	fmt.Printf("Restored Type: %s\n", restored.MType)
	fmt.Printf("Restored Value: %.1f\n", *restored.Value)

	// Output:
	// JSON: {"id":"memory_usage","type":"gauge","value":75.3}
	// Restored ID: memory_usage
	// Restored Type: gauge
	// Restored Value: 75.3
}

// ExampleMetrics_nilValues демонстрирует использование указателей для различения отсутствия значения.
func ExampleMetrics_nilValues() {
	// Метрика без значения (для запроса)
	request := dto.Metrics{
		ID:    "temperature",
		MType: "gauge",
	}

	// Проверяем наличие значения
	if request.Value == nil {
		fmt.Println("Value is not set")
	}

	if request.Delta == nil {
		fmt.Println("Delta is not set")
	}

	// Output:
	// Value is not set
	// Delta is not set
}
