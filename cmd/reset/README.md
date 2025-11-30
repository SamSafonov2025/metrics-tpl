# Reset Generator

Утилита для автоматической генерации методов `Reset()` для структур с комментарием `// generate:reset`.

## Описание

`cmd/reset` - это кодогенератор, который сканирует весь проект и создаёт методы сброса состояния для структур, помеченных специальным комментарием. Сгенерированные методы приводят все поля структуры к начальным (нулевым) значениям.

## Использование

### Запуск генератора

```bash
# Запуск из корня проекта
go run ./cmd/reset

# Через go generate (если добавлена директива //go:generate)
go generate ./...
```

### Пометка структур для генерации

Добавьте комментарий `// generate:reset` над определением структуры:

```go
package mypackage

//go:generate go run ../../cmd/reset

// generate:reset
type MyStruct struct {
    ID       int
    Name     string
    Items    []string
    Metadata map[string]interface{}
    Parent   *MyStruct
}
```

После запуска генератора в той же директории появится файл `reset.gen.go` с методом:

```go
func (r *MyStruct) Reset() {
    if r == nil {
        return
    }

    r.ID = 0
    r.Name = ""
    r.Items = r.Items[:0]
    clear(r.Metadata)
    if r.Parent != nil {
        if resetter, ok := interface{}(r.Parent).(interface{ Reset() }); ok {
            resetter.Reset()
        }
    }
}
```

## Правила сброса полей

Метод `Reset()` обрабатывает различные типы полей по следующим правилам:

### 1. Примитивные типы

Приводятся к нулевым значениям:
- `int`, `int8`, `int16`, `int32`, `int64` → `0`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` → `0`
- `float32`, `float64` → `0`
- `string` → `""`
- `bool` → `false`

```go
// generate:reset
type Example struct {
    Count  int
    Name   string
    Active bool
}

// Сгенерирует:
// r.Count = 0
// r.Name = ""
// r.Active = false
```

### 2. Слайсы

Обрезаются по длине без изменения capacity:

```go
// generate:reset
type Example struct {
    Items []string
}

// Сгенерирует:
// r.Items = r.Items[:0]
```

**Почему не `nil`?** Это позволяет избежать лишних аллокаций при повторном использовании структуры.

### 3. Мапы

Очищаются встроенной функцией `clear()`:

```go
// generate:reset
type Example struct {
    Data map[string]int
}

// Сгенерирует:
// clear(r.Data)
```

### 4. Указатели

Для не-nil указателей сбрасываются значения по тем же правилам:

```go
// generate:reset
type Example struct {
    Count *int
    Name  *string
}

// Сгенерирует:
// if r.Count != nil {
//     *r.Count = 0
// }
// if r.Name != nil {
//     *r.Name = ""
// }
```

### 5. Вложенные структуры

Если структура имеет метод `Reset()`, он будет вызван:

```go
// generate:reset
type Child struct {
    Value int
}

// generate:reset
type Parent struct {
    Child  Child
    ChildP *Child
}

// Сгенерирует:
// if resetter, ok := interface{}(&r.Child).(interface{ Reset() }); ok {
//     resetter.Reset()
// }
// if r.ChildP != nil {
//     if resetter, ok := interface{}(r.ChildP).(interface{ Reset() }); ok {
//         resetter.Reset()
//     }
// }
```

## Примеры использования

### Пример 1: Простая структура

```go
// generate:reset
type Config struct {
    Host     string
    Port     int
    Debug    bool
    Features []string
}

func main() {
    cfg := &Config{
        Host:     "localhost",
        Port:     8080,
        Debug:    true,
        Features: []string{"auth", "logging"},
    }

    cfg.Reset()

    // Теперь:
    // cfg.Host = ""
    // cfg.Port = 0
    // cfg.Debug = false
    // len(cfg.Features) = 0
}
```

### Пример 2: Пул объектов

```go
// generate:reset
type Request struct {
    ID      string
    Headers map[string]string
    Body    []byte
}

var requestPool = sync.Pool{
    New: func() interface{} {
        return &Request{
            Headers: make(map[string]string),
            Body:    make([]byte, 0, 1024),
        }
    },
}

func GetRequest() *Request {
    return requestPool.Get().(*Request)
}

func PutRequest(r *Request) {
    r.Reset() // Очистка перед возвратом в пул
    requestPool.Put(r)
}
```

### Пример 3: Сложная структура с вложенностью

```go
// generate:reset
type Metrics struct {
    Counts map[string]int64
    Gauges map[string]float64
}

// generate:reset
type Session struct {
    UserID   string
    Token    string
    Data     map[string]interface{}
    Metrics  *Metrics
    Created  time.Time
}

func main() {
    s := &Session{
        UserID:  "user123",
        Token:   "secret",
        Data:    map[string]interface{}{"key": "value"},
        Metrics: &Metrics{
            Counts: map[string]int64{"requests": 100},
            Gauges: map[string]float64{"cpu": 0.5},
        },
    }

    s.Reset()

    // Все поля сброшены:
    // s.UserID = ""
    // s.Token = ""
    // len(s.Data) = 0
    // s.Metrics.Counts и s.Metrics.Gauges очищены
}
```

## Архитектура

### Основные компоненты

```
cmd/reset/
└── main.go              # Основной файл генератора

Генератор состоит из:
1. Сканера директорий (filepath.Walk)
2. AST парсера (go/ast)
3. Анализатора структур
4. Генератора кода методов Reset()
5. Форматтера кода (go/format)
```

### Алгоритм работы

1. **Сканирование**: Рекурсивный обход всех `.go` файлов проекта
2. **Парсинг**: Построение AST для каждого файла
3. **Поиск**: Определение структур с комментарием `// generate:reset`
4. **Анализ**: Извлечение информации о полях структур
5. **Генерация**: Создание кода методов `Reset()`
6. **Группировка**: Объединение методов по пакетам
7. **Форматирование**: Применение go/format
8. **Запись**: Сохранение в файлы `reset.gen.go`

### Структуры данных

```go
type StructInfo struct {
    Name    string      // Имя структуры
    Fields  []FieldInfo // Поля структуры
    PkgName string      // Имя пакета
    PkgPath string      // Путь к пакету
}

type FieldInfo struct {
    Name      string // Имя поля
    Type      string // Тип поля
    IsPointer bool   // Является ли указателем
    IsSlice   bool   // Является ли слайсом
    IsMap     bool   // Является ли мапой
    IsStruct  bool   // Является ли структурой
}
```
## Примеры сгенерированного кода

### Входные данные

```go
// generate:reset
type User struct {
    ID       int
    Name     string
    Email    *string
    Tags     []string
    Settings map[string]interface{}
    Profile  *Profile
}

// generate:reset
type Profile struct {
    Bio   string
    Avatar string
}
```

### Выходные данные (reset.gen.go)

```go
// Code generated by cmd/reset; DO NOT EDIT.

package mypackage

func (r *User) Reset() {
    if r == nil {
        return
    }

    r.ID = 0
    r.Name = ""
    if r.Email != nil {
        *r.Email = ""
    }
    r.Tags = r.Tags[:0]
    clear(r.Settings)
    if r.Profile != nil {
        if resetter, ok := interface{}(r.Profile).(interface{ Reset() }); ok {
            resetter.Reset()
        }
    }
}

func (r *Profile) Reset() {
    if r == nil {
        return
    }

    r.Bio = ""
    r.Avatar = ""
}
```