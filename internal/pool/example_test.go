package pool_test

import (
	"fmt"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/pool"
)

// Example демонстрирует базовое использование Pool
func Example() {
	// Создаем пул для структур SimpleStruct из dto
	simplePool := pool.New(func() *dto.SimpleStruct {
		return &dto.SimpleStruct{}
	})

	// Получаем объект из пула
	obj := simplePool.Get()
	obj.Value = 42
	obj.Label = "example"

	fmt.Printf("Before Put: Value=%d, Label=%s\n", obj.Value, obj.Label)

	// Возвращаем в пул - автоматически вызывается Reset()
	simplePool.Put(obj)

	// Получаем снова - объект сброшен
	obj2 := simplePool.Get()
	fmt.Printf("After Get: Value=%d, Label=%s\n", obj2.Value, obj2.Label)

	// Output:
	// Before Put: Value=42, Label=example
	// After Get: Value=0, Label=
}

// Example_complex демонстрирует использование с более сложной структурой
func Example_complex() {
	// Создаем пул для ComplexStruct
	complexPool := pool.New(func() *dto.ComplexStruct {
		return &dto.ComplexStruct{
			Tags:     make([]string, 0, 10),
			Metadata: make(map[string]interface{}),
		}
	})

	// Получаем и используем объект
	obj := complexPool.Get()
	obj.ID = 100
	obj.Name = "test"
	obj.Active = true
	obj.Tags = append(obj.Tags, "tag1", "tag2")
	obj.Metadata["key"] = "value"

	fmt.Printf("Before Put: ID=%d, Name=%s, Tags=%v\n", obj.ID, obj.Name, obj.Tags)

	// Возвращаем в пул
	complexPool.Put(obj)

	// Получаем снова
	obj2 := complexPool.Get()
	fmt.Printf("After Get: ID=%d, Name=%s, Tags=%v, Metadata=%v\n",
		obj2.ID, obj2.Name, obj2.Tags, obj2.Metadata)

	// Output:
	// Before Put: ID=100, Name=test, Tags=[tag1 tag2]
	// After Get: ID=0, Name=, Tags=[], Metadata=map[]
}

// Example_reuse демонстрирует переиспользование объектов
func Example_reuse() {
	pool := pool.New(func() *dto.ResetableStruct {
		return &dto.ResetableStruct{}
	})

	// Используем объект несколько раз
	for i := 0; i < 3; i++ {
		obj := pool.Get()
		// Используем объект
		fmt.Printf("Iteration %d: got object from pool\n", i+1)
		pool.Put(obj)
	}

	// Output:
	// Iteration 1: got object from pool
	// Iteration 2: got object from pool
	// Iteration 3: got object from pool
}
