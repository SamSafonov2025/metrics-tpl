# Pool - Generic Object Pool with Auto-Reset

Generic-пул объектов с автоматическим сбросом состояния для эффективного переиспользования "тяжёлых" объектов.

## Описание

`pool.Pool[T]` - это типобезопасная обёртка вокруг `sync.Pool`, которая автоматически вызывает метод `Reset()` перед возвратом объекта в пул. Это гарантирует, что следующий вызов `Get()` получит объект в чистом начальном состоянии.

## Особенности

- ✅ **Type-safe**: Использование дженериков гарантирует типобезопасность на этапе компиляции
- ✅ **Auto-reset**: Автоматический вызов `Reset()` при возврате объекта в пул
- ✅ **Zero allocations**: При активном использовании пула аллокации памяти минимальны
- ✅ **Thread-safe**: Потокобезопасная реализация на основе `sync.Pool`
- ✅ **Simple API**: Всего 3 метода: `New()`, `Get()`, `Put()`

## Требования к типам

Тип `T` должен реализовывать интерфейс `Resettable`:

```go
type Resettable interface {
    Reset()
}
```

Метод `Reset()` должен:
- Сбрасывать все поля к нулевым значениям
- Обрезать слайсы: `s = s[:0]` (сохраняя capacity!)
- Очищать мапы: `clear(m)`
- Обрабатывать nil-ресивер

Для генерации методов `Reset()` используйте утилиту `cmd/reset`.

## Установка

```go
import "github.com/SamSafonov2025/metrics-tpl/internal/pool"
```

## Использование

### Базовый пример

```go
package main

import (
    "fmt"
    "github.com/SamSafonov2025/metrics-tpl/internal/pool"
)

// Определяем структуру
type Request struct {
    ID      int
    Method  string
    Headers map[string]string
    Body    []byte
}

// Реализуем метод Reset()
func (r *Request) Reset() {
    if r == nil {
        return
    }
    r.ID = 0
    r.Method = ""
    clear(r.Headers)
    r.Body = r.Body[:0]
}

func main() {
    // Создаём пул с конструктором
    requestPool := pool.New(func() *Request {
        return &Request{
            Headers: make(map[string]string),
            Body:    make([]byte, 0, 1024),
        }
    })

    // Получаем объект из пула
    req := requestPool.Get()
    req.ID = 1
    req.Method = "GET"
    req.Headers["Content-Type"] = "application/json"

    // Используем объект...
    fmt.Printf("Request: %+v\n", req)

    // Возвращаем в пул - автоматически вызывается Reset()
    requestPool.Put(req)

    // Следующий Get() получит чистый объект
    req2 := requestPool.Get()
    fmt.Printf("Clean request: %+v\n", req2)
}
```

### Использование с генератором Reset()

```go
package mypackage

//go:generate go run ../../cmd/reset

// generate:reset
type User struct {
    ID       int64
    Username string
    Email    string
    Tags     []string
    Settings map[string]interface{}
}

// Метод Reset() сгенерируется автоматически в reset.gen.go

func main() {
    // Создаём пул пользователей
    userPool := pool.New(func() *User {
        return &User{
            Tags:     make([]string, 0, 10),
            Settings: make(map[string]interface{}),
        }
    })

    user := userPool.Get()
    user.ID = 123
    user.Username = "john"
    // ... используем объект

    userPool.Put(user) // автоматический сброс
}
```

### Пример с HTTP-сервером

```go
type Response struct {
    StatusCode int
    Headers    map[string]string
    Body       []byte
}

func (r *Response) Reset() {
    if r == nil {
        return
    }
    r.StatusCode = 0
    clear(r.Headers)
    r.Body = r.Body[:0]
}

var responsePool = pool.New(func() *Response {
    return &Response{
        Headers: make(map[string]string),
        Body:    make([]byte, 0, 4096),
    }
})

func handler(w http.ResponseWriter, r *http.Request) {
    resp := responsePool.Get()
    defer responsePool.Put(resp)

    // Используем resp...
    resp.StatusCode = 200
    resp.Headers["Content-Type"] = "application/json"
    resp.Body = append(resp.Body, []byte(`{"status":"ok"}`)...)

    // Пишем ответ
    for k, v := range resp.Headers {
        w.Header().Set(k, v)
    }
    w.WriteHeader(resp.StatusCode)
    w.Write(resp.Body)
}
```

## API

### New[T Resettable](newFunc func() T) *Pool[T]

Создаёт новый пул с указанной функцией-конструктором.

**Параметры:**
- `newFunc` - функция, которая создаёт новый экземпляр типа T

**Возвращает:**
- Указатель на новый `Pool[T]`

**Пример:**
```go
pool := pool.New(func() *MyStruct {
    return &MyStruct{
        Data: make([]byte, 0, 1024),
    }
})
```

### Get() T

Получает объект из пула.

Если пул пуст, создаётся новый объект с помощью конструктора.
Объект может быть ранее использован и сброшен.

**Возвращает:**
- Объект типа T в начальном состоянии

**Пример:**
```go
obj := pool.Get()
obj.Field = "value"
```

### Put(obj T)

Возвращает объект в пул после сброса.

Автоматически вызывает `obj.Reset()` перед помещением в пул.
Безопасно для вызова с nil (метод Reset должен это обрабатывать).

**Параметры:**
- `obj` - объект для возврата в пул

**Пример:**
```go
pool.Put(obj) // obj.Reset() вызывается автоматически
```
