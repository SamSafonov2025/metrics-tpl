# StaticLint - Static Analyzer for Go Code Quality

Статический анализатор для проверки качества кода в Go проектах.

## Описание

`staticlint` - это кастомный статический анализатор, построенный на базе `golang.org/x/tools/go/analysis`, который проверяет код на наличие небезопасных практик обработки ошибок.

## Проверки

Анализатор выполняет следующие проверки:

1. **Запрет использования `panic()`** - обнаруживает прямые вызовы встроенной функции `panic` в любом месте кода
2. **Ограничение `os.Exit`** - запрещает вызовы `os.Exit` вне функции `main` пакета `main`
3. **Ограничение `log.Fatal*`** - запрещает вызовы `log.Fatal`, `log.Fatalf`, `log.Fatalln` вне функции `main` пакета `main`

## Установка и запуск

### Запуск через go run (рекомендуется)

```bash
# Проверка всего проекта
go run ./cmd/staticlint ./...

# Проверка конкретного пакета
go run ./cmd/staticlint ./internal/storage/...
```

### Сборка и запуск бинарника

```bash
# Сборка
go build -o staticlint ./cmd/staticlint

# Запуск (Linux/macOS)
./staticlint ./...

# Запуск (Windows)
go build -o staticlint.exe ./cmd/staticlint
.\staticlint.exe .\...
```

## Запуск тестов

```bash
# Запуск тестов анализатора
go test ./cmd/staticlint -v

# Краткий вывод
go test ./cmd/staticlint
```

## Результаты первоначального запуска

При первом запуске на проекте были обнаружены следующие нарушения:

```
D:\AutoGPT\workdir\yandex_go\metrics-tpl\cmd\agent\main.go:399:3: direct call to panic is not allowed
D:\AutoGPT\workdir\yandex_go\metrics-tpl\internal\storage\dbstorage\dbstorage.go:159:5: log.Fatalf must only be called in main function of main package
D:\AutoGPT\workdir\yandex_go\metrics-tpl\internal\storage\dbstorage\dbstorage.go:182:3: log.Fatalf must only be called in main function of main package
D:\AutoGPT\workdir\yandex_go\metrics-tpl\cmd\server\main.go:26:3: direct call to panic is not allowed
```

## Внесённые исправления

### 1. cmd/agent/main.go:399

**Было:**
```go
if err := logger.Init(); err != nil {
    panic(err)
}
```

**Стало:**
```go
if err := logger.Init(); err != nil {
    log.Fatalf("Failed to initialize logger: %v", err)
}
```

**Обоснование:** Использование `panic()` в production коде затрудняет graceful shutdown и обработку ошибок. Заменено на `log.Fatalf()` в функции `main`, что соответствует требованиям анализатора.

---

### 2. cmd/server/main.go:26

**Было:**
```go
if err := logger.Init(); err != nil {
    panic(err)
}
```

**Стало:**
```go
if err := logger.Init(); err != nil {
    log.Fatalf("Failed to initialize logger: %v", err)
}
```

**Обоснование:** Аналогично исправлению в agent - замена `panic()` на `log.Fatalf()` в функции `main`.

---

### 3. internal/storage/dbstorage/dbstorage.go:159

**Было:**
```go
if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
    log.Fatalf("Unable to rollback transaction: %v", rollbackErr)
}
```

**Стало:**
```go
if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
    log.Printf("Unable to rollback transaction: %v", rollbackErr)
}
```

**Обоснование:** Вызов `log.Fatalf()` в библиотечном коде (вне функции `main`) является плохой практикой - он не даёт вызывающему коду возможность обработать ошибку. Заменено на `log.Printf()` для логирования ошибки без прерывания программы (основная ошибка всё равно возвращается через переменную `err`).

---

### 4. internal/storage/dbstorage/dbstorage.go:182

**Было:**
```go
err = tx.Commit(context.Background())
if err != nil {
    log.Fatalf("Unable to commit transaction: %v", err)
}
```

**Стало:**
```go
err = tx.Commit(context.Background())
if err != nil {
    return fmt.Errorf("unable to commit transaction: %w", err)
}
```

**Обоснование:** Использование `log.Fatalf()` в библиотечном коде недопустимо. Заменено на корректный возврат ошибки с использованием `fmt.Errorf` и обёртки ошибки через `%w` для сохранения цепочки ошибок.

---

## Текущий статус

После внесения всех исправлений проект успешно проходит все проверки:

```bash
$ go run ./cmd/staticlint ./...
# Нет вывода - все проверки пройдены успешно
```
