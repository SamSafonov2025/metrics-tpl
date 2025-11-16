package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	"github.com/SamSafonov2025/metrics-tpl/internal/storage/memstorage"
)

const (
	pprofAddr     = ":6060"
	numMetrics    = 1000
	numIterations = 10000
)

func main() {
	// Запускаем HTTP сервер для pprof
	go func() {
		log.Println("Starting pprof server on", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			log.Fatal(err)
		}
	}()

	// Даем серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	log.Println("Starting profiling workload...")

	// Создаем storage, который будет жить во время всего профилирования
	storage := memstorage.New()

	// Запускаем фоновую нагрузку
	go generateLoad(storage)
	go serializeMetrics(storage)

	// Ждем, пока накопятся данные
	time.Sleep(2 * time.Second)

	log.Println("Collecting memory profile during active workload...")

	// Сохраняем профиль памяти во время активной работы
	if err := saveMemProfile("profiles/base.pprof"); err != nil {
		log.Fatal("Failed to save memory profile:", err)
	}

	log.Println("Memory profile saved to profiles/base.pprof")
	log.Println("Press Ctrl+C to exit or wait...")

	// Держим программу запущенной для возможности дополнительного профилирования
	time.Sleep(10 * time.Second)
}

func runWorkload() {
	storage := memstorage.New()
	svc := service.NewMetricsService(storage, 0, nil)
	ctx := context.Background()

	log.Printf("Running %d iterations with %d metrics...\n", numIterations, numMetrics)

	// Эмулируем реальную нагрузку: обновление метрик
	for i := 0; i < numIterations; i++ {
		// Одиночные обновления
		for j := 0; j < 10; j++ {
			metricName := fmt.Sprintf("metric_%d", rand.Intn(numMetrics))
			value := rand.Float64() * 1000

			metric := dto.Metrics{
				ID:    metricName,
				MType: "gauge",
				Value: &value,
			}

			_, _ = svc.Update(ctx, metric)
		}

		// Батч-обновления
		if i%10 == 0 {
			metrics := generateMetricsBatch(100)
			_ = svc.UpdateBatch(ctx, metrics)
		}

		// Чтение метрик
		if i%50 == 0 {
			_, _, _ = svc.List(ctx)
		}

		// JSON encoding/decoding (эмулируем HTTP запросы)
		if i%5 == 0 {
			metric := dto.Metrics{
				ID:    fmt.Sprintf("metric_%d", rand.Intn(numMetrics)),
				MType: "gauge",
			}
			data, _ := json.Marshal(metric)
			_ = json.Unmarshal(data, &metric)
		}
	}

	log.Println("Workload completed")
}

func generateMetricsBatch(size int) []dto.Metrics {
	metrics := make([]dto.Metrics, size)
	for i := 0; i < size; i++ {
		if i%2 == 0 {
			value := rand.Float64() * 1000
			metrics[i] = dto.Metrics{
				ID:    fmt.Sprintf("gauge_%d", rand.Intn(numMetrics)),
				MType: "gauge",
				Value: &value,
			}
		} else {
			delta := rand.Int63n(100)
			metrics[i] = dto.Metrics{
				ID:    fmt.Sprintf("counter_%d", rand.Intn(numMetrics)),
				MType: "counter",
				Delta: &delta,
			}
		}
	}
	return metrics
}

func saveMemProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	runtime.GC() // Запускаем сборщик мусора для точной статистики
	if err := pprof.WriteHeapProfile(f); err != nil {
		return err
	}

	return nil
}

func saveCPUProfile(filename string, duration time.Duration) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}

	time.Sleep(duration)
	pprof.StopCPUProfile()

	return nil
}

// generateLoad создает дополнительную нагрузку для более реалистичного профиля
func generateLoad(storage *memstorage.MemStorage) {
	ctx := context.Background()

	// Заполняем storage данными
	for i := 0; i < numMetrics; i++ {
		_ = storage.SetGauge(ctx, fmt.Sprintf("gauge_%d", i), rand.Float64()*1000)
		_ = storage.IncrementCounter(ctx, fmt.Sprintf("counter_%d", i), rand.Int63n(1000))
	}

	// Постоянно читаем и обновляем метрики
	for {
		// Случайные операции
		switch rand.Intn(3) {
		case 0:
			// Чтение
			_, _ = storage.GetGauge(ctx, fmt.Sprintf("gauge_%d", rand.Intn(numMetrics)))
		case 1:
			// Запись gauge
			_ = storage.SetGauge(ctx, fmt.Sprintf("gauge_%d", rand.Intn(numMetrics)), rand.Float64()*1000)
		case 2:
			// Инкремент counter
			_ = storage.IncrementCounter(ctx, fmt.Sprintf("counter_%d", rand.Intn(numMetrics)), rand.Int63n(10))
		}

		// Периодически читаем все метрики (создаем аллокации)
		if rand.Intn(100) == 0 {
			_ = storage.GetAllGauges(ctx)
			_ = storage.GetAllCounters(ctx)
		}

		// Периодически делаем батч-обновления
		if rand.Intn(50) == 0 {
			metrics := generateMetricsBatch(50)
			_ = storage.SetMetrics(ctx, metrics)
		}

		time.Sleep(time.Microsecond)
	}
}

// serializeMetrics создает дополнительные аллокации через JSON
func serializeMetrics(storage *memstorage.MemStorage) {
	ctx := context.Background()

	for {
		gauges := storage.GetAllGauges(ctx)
		counters := storage.GetAllCounters(ctx)

		// Создаем структуру для сериализации
		type MetricsResponse struct {
			Gauges   map[string]float64 `json:"gauges"`
			Counters map[string]int64   `json:"counters"`
		}

		resp := MetricsResponse{
			Gauges:   gauges,
			Counters: counters,
		}

		// Сериализуем в JSON
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		_ = encoder.Encode(resp)

		// Десериализуем обратно
		decoder := json.NewDecoder(&buf)
		var decoded MetricsResponse
		_ = decoder.Decode(&decoded)

		time.Sleep(10 * time.Millisecond)
	}
}
