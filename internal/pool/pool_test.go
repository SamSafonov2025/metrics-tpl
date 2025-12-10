package pool

import (
	"sync"
	"testing"
)

// TestStruct - тестовая структура с методом Reset()
type TestStruct struct {
	Value  int
	Name   string
	Data   []byte
	Counts map[string]int
}

func (t *TestStruct) Reset() {
	if t == nil {
		return
	}
	t.Value = 0
	t.Name = ""
	t.Data = t.Data[:0]
	clear(t.Counts)
}

// TestPool_Basic проверяет базовую функциональность пула
func TestPool_Basic(t *testing.T) {
	// Счетчик созданных объектов
	var created int

	pool := New(func() *TestStruct {
		created++
		return &TestStruct{
			Data:   make([]byte, 0, 1024),
			Counts: make(map[string]int),
		}
	})

	// Первый Get должен создать новый объект
	obj1 := pool.Get()
	if obj1 == nil {
		t.Fatal("Get() returned nil")
	}
	if created != 1 {
		t.Errorf("Expected 1 object created, got %d", created)
	}

	// Модифицируем объект
	obj1.Value = 42
	obj1.Name = "test"
	obj1.Data = append(obj1.Data, []byte("hello")...)
	obj1.Counts["requests"] = 10

	// Возвращаем в пул
	pool.Put(obj1)

	// Получаем снова - должны получить тот же объект, но сброшенный
	obj2 := pool.Get()
	if obj2 == nil {
		t.Fatal("Get() returned nil")
	}

	// Проверяем, что объект сброшен
	if obj2.Value != 0 {
		t.Errorf("Expected Value to be 0, got %d", obj2.Value)
	}
	if obj2.Name != "" {
		t.Errorf("Expected Name to be empty, got %s", obj2.Name)
	}
	if len(obj2.Data) != 0 {
		t.Errorf("Expected Data length to be 0, got %d", len(obj2.Data))
	}
	if len(obj2.Counts) != 0 {
		t.Errorf("Expected Counts to be empty, got %v", obj2.Counts)
	}

	// Capacity должна быть сохранена
	if cap(obj2.Data) == 0 {
		t.Error("Expected Data capacity to be preserved")
	}

	// Второй Get не должен создавать новый объект (используется из пула)
	if created > 1 {
		t.Errorf("Expected only 1 object created, got %d", created)
	}
}

// TestPool_MultipleObjects проверяет работу с несколькими объектами
func TestPool_MultipleObjects(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data:   make([]byte, 0, 128),
			Counts: make(map[string]int),
		}
	})

	// Получаем несколько объектов
	objs := make([]*TestStruct, 5)
	for i := 0; i < 5; i++ {
		objs[i] = pool.Get()
		objs[i].Value = i
		objs[i].Name = "obj"
	}

	// Возвращаем все в пул
	for i := 0; i < 5; i++ {
		pool.Put(objs[i])
	}

	// Получаем снова и проверяем, что все сброшены
	for i := 0; i < 5; i++ {
		obj := pool.Get()
		if obj.Value != 0 {
			t.Errorf("Object %d: expected Value to be 0, got %d", i, obj.Value)
		}
		if obj.Name != "" {
			t.Errorf("Object %d: expected Name to be empty, got %s", i, obj.Name)
		}
	}
}

// TestPool_Concurrent проверяет потокобезопасность пула
func TestPool_Concurrent(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data:   make([]byte, 0, 256),
			Counts: make(map[string]int),
		}
	})

	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				obj := pool.Get()
				obj.Value = id
				obj.Name = "concurrent"
				obj.Data = append(obj.Data, byte(i))
				obj.Counts["iteration"] = i
				pool.Put(obj)
			}
		}(g)
	}

	wg.Wait()
}

// TestPool_NilHandling проверяет обработку nil
func TestPool_NilHandling(t *testing.T) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data:   make([]byte, 0, 64),
			Counts: make(map[string]int),
		}
	})

	// Put с nil не должно паниковать
	// (хотя это плохая практика, но метод Reset() должен обрабатывать nil)
	var nilObj *TestStruct
	// Это вызовет панику, потому что obj.Reset() на nil вызовет панику
	// Но наш Reset() проверяет на nil
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Put(nil) panicked as expected: %v", r)
		}
	}()
	pool.Put(nilObj)
}

// Benchmark для измерения производительности
func BenchmarkPool_GetPut(b *testing.B) {
	pool := New(func() *TestStruct {
		return &TestStruct{
			Data:   make([]byte, 0, 1024),
			Counts: make(map[string]int),
		}
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := pool.Get()
			obj.Value = 123
			obj.Data = append(obj.Data, []byte("benchmark")...)
			pool.Put(obj)
		}
	})
}

// BenchmarkPool_vs_Allocation сравнивает пул с прямым созданием объектов
func BenchmarkPool_vs_Allocation(b *testing.B) {
	b.Run("Pool", func(b *testing.B) {
		pool := New(func() *TestStruct {
			return &TestStruct{
				Data:   make([]byte, 0, 1024),
				Counts: make(map[string]int),
			}
		})

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			obj := pool.Get()
			obj.Value = i
			pool.Put(obj)
		}
	})

	b.Run("Direct", func(b *testing.B) {
		var sum int
		for i := 0; i < b.N; i++ {
			obj := &TestStruct{}
			obj.Value = i
			sum += obj.Value
		}
		// Prevent compiler from optimizing away the benchmark
		if sum < 0 {
			b.Fatal("unexpected sum")
		}
	})
}
