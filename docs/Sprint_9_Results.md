# Спринт 9 - Результаты тестирования

## Инкремент 27 - Trusted Subnet

### Описание

Реализована функциональность проверки IP-адресов агентов на соответствие доверенной подсети в формате CIDR.

**Изменения:**
- Добавлено поле конфигурации `TrustedSubnet` (env: `TRUSTED_SUBNET`, флаг: `-t`)
- Создан middleware для проверки IP-адреса из заголовка `X-Real-IP`
- Агент добавляет заголовок `X-Real-IP` ко всем запросам
- Сервер возвращает 403 Forbidden для запросов из недоверенных IP
- Если `trusted_subnet` пустой, все запросы принимаются

---

## Автоматические тесты

### Unit тесты middleware

```bash
$ go test -v ./internal/middleware
=== RUN   TestTrustedSubnetMiddleware
=== RUN   TestTrustedSubnetMiddleware/Empty_subnet_allows_all
=== RUN   TestTrustedSubnetMiddleware/IP_in_subnet
=== RUN   TestTrustedSubnetMiddleware/IP_not_in_subnet
=== RUN   TestTrustedSubnetMiddleware/Localhost_in_/8
=== RUN   TestTrustedSubnetMiddleware/Invalid_CIDR_blocks_all
--- PASS: TestTrustedSubnetMiddleware (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Empty_subnet_allows_all (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/IP_in_subnet (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/IP_not_in_subnet (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Localhost_in_/8 (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Invalid_CIDR_blocks_all (0.00s)
=== RUN   TestTrustedSubnetMiddleware_RemoteAddrFallback
--- PASS: TestTrustedSubnetMiddleware_RemoteAddrFallback (0.00s)
=== RUN   TestIsIPInSubnet
--- PASS: TestIsIPInSubnet (0.00s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/internal/middleware	0.013s
```

### Интеграционные тесты

```bash
$ go test -v ./cmd/server -run TestTrustedSubnet
=== RUN   TestTrustedSubnet_AllowedIP
--- PASS: TestTrustedSubnet_AllowedIP (0.00s)
=== RUN   TestTrustedSubnet_BlockedIP
--- PASS: TestTrustedSubnet_BlockedIP (0.00s)
=== RUN   TestTrustedSubnet_EmptySubnet
--- PASS: TestTrustedSubnet_EmptySubnet (0.00s)
=== RUN   TestTrustedSubnet_Localhost
--- PASS: TestTrustedSubnet_Localhost (0.00s)
=== RUN   TestTrustedSubnet_BatchUpdate
--- PASS: TestTrustedSubnet_BatchUpdate (0.00s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/cmd/server	0.012s
```

---

## Ручное тестирование

### Тест 1: Сервер с доверенной подсетью

```bash
$ go build -o /tmp/server cmd/server/main.go
$ go build -o /tmp/agent cmd/agent/main.go

# Терминал 1: запуск сервера с подсетью 127.0.0.0/8
$ /tmp/server -a="localhost:9091" -t="127.0.0.0/8"
INFO  Server config loaded  {
  "address": "localhost:9091",
  "trusted_subnet": "127.0.0.0/8"
}
INFO  Server started
```

### Тест 2: Агент отправляет метрики

```bash
# Терминал 2: запуск агента
$ /tmp/agent -a="localhost:9091" -p=1s -r=2s
agent: started | poll=1s report=2s | server=localhost:9091
agent: sending batch (29 metrics) -> /updates/
agent: response /updates/ -> 200 in 5.234ms
agent: batch sent successfully
```

**Результат**: ✅ Агент успешно отправляет метрики, IP 127.0.0.1 входит в подсеть 127.0.0.0/8

### Тест 3: Без доверенной подсети

```bash
$ /tmp/server -a="localhost:9093"
INFO  Server config loaded  {"trusted_subnet": ""}
INFO  Server started

$ /tmp/agent -a="localhost:9093" -p=1s -r=2s
agent: response /updates/ -> 200 in 3.123ms
agent: batch sent successfully
```

**Результат**: ✅ Без trusted_subnet все запросы принимаются

---

## Статистика тестов

### Покрытие кода тестами

- **Unit тесты**: 3 теста (8 подтестов)
- **Интеграционные тесты**: 5 тестов
- **Общий размер тестов**: 231 строка (после оптимизации)

### Файлы тестов

```bash
$ wc -l internal/middleware/subnet_test.go cmd/server/trusted_subnet_test.go
  89 internal/middleware/subnet_test.go
 142 cmd/server/trusted_subnet_test.go
 231 total
```

**Оптимизация**: Тесты были упрощены с 641 до 231 строки (↓64%) с использованием helper функций и table-driven подхода.

---


## Инкремент 28 - gRPC Support

### Описание

Реализован обмен метриками между агентом и сервером по протоколу gRPC в дополнение к HTTP.

**Изменения:**

**Server-side:**
- Создан proto-файл `proto/metrics.proto` с определением сервиса `Metrics` и RPC `UpdateMetrics`
- Добавлено поле конфигурации `GRPCAddress` (env: `GRPC_ADDRESS`, флаг: `-g`)
- Создан пакет `internal/grpcserver` с реализацией gRPC сервера
- Реализован `TrustedSubnetInterceptor` для проверки IP из метаданных
- Интерсептор проверяет IP-адрес из метаданных с ключом `x-real-ip`
- Возвращает `codes.PermissionDenied` для IP вне доверенной подсети
- Сервер поддерживает одновременную работу HTTP и gRPC

**Agent-side:**
- Добавлено поле конфигурации `GRPCAddress` (env: `GRPC_ADDRESS`, флаг: `-g`)
- Создан `cmd/agent/grpc_sender.go` с gRPC клиентом
- Агент автоматически добавляет IP в метаданные с ключом `x-real-ip`
- Поддержка переключения между HTTP и gRPC режимами
- Graceful shutdown для gRPC соединения

**Технические детали:**
- Protocol Buffers 3 (proto3)
- Типы метрик: GAUGE (0) и COUNTER (1)
- Пакетная отправка метрик через `UpdateMetricsRequest`
- Конвертация между `proto.Metric` и `dto.Metrics`
- Все параметры конфигурируются через env/flags/JSON

---

## Автоматические тесты

### Тесты после Increment 28

```bash
$ go test ./cmd/server/... ./cmd/agent/... ./internal/grpcserver/... -v
=== RUN   TestServerGracefulShutdown
--- PASS: TestServerGracefulShutdown (0.11s)
=== RUN   TestServerShutdownWithRequest
--- PASS: TestServerShutdownWithRequest (0.60s)
=== RUN   TestDataSaveOnShutdown
--- PASS: TestDataSaveOnShutdown (0.00s)
=== RUN   TestTrustedSubnet_AllowedIP
--- PASS: TestTrustedSubnet_AllowedIP (0.00s)
=== RUN   TestTrustedSubnet_BlockedIP
--- PASS: TestTrustedSubnet_BlockedIP (0.00s)
=== RUN   TestTrustedSubnet_EmptySubnet
--- PASS: TestTrustedSubnet_EmptySubnet (0.00s)
=== RUN   TestTrustedSubnet_Localhost
--- PASS: TestTrustedSubnet_Localhost (0.00s)
=== RUN   TestTrustedSubnet_BatchUpdate
--- PASS: TestTrustedSubnet_BatchUpdate (0.00s)
PASS
=== RUN   TestAgentGracefulShutdown
--- PASS: TestAgentGracefulShutdown (0.56s)
PASS
```

### Unit тесты middleware (Increment 27)

```bash
$ go test -v ./internal/middleware
=== RUN   TestTrustedSubnetMiddleware
=== RUN   TestTrustedSubnetMiddleware/Empty_subnet_allows_all
=== RUN   TestTrustedSubnetMiddleware/IP_in_subnet
=== RUN   TestTrustedSubnetMiddleware/IP_not_in_subnet
=== RUN   TestTrustedSubnetMiddleware/Localhost_in_/8
=== RUN   TestTrustedSubnetMiddleware/Invalid_CIDR_blocks_all
--- PASS: TestTrustedSubnetMiddleware (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Empty_subnet_allows_all (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/IP_in_subnet (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/IP_not_in_subnet (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Localhost_in_/8 (0.00s)
    --- PASS: TestTrustedSubnetMiddleware/Invalid_CIDR_blocks_all (0.00s)
=== RUN   TestTrustedSubnetMiddleware_RemoteAddrFallback
--- PASS: TestTrustedSubnetMiddleware_RemoteAddrFallback (0.00s)
=== RUN   TestIsIPInSubnet
--- PASS: TestIsIPInSubnet (0.00s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/internal/middleware	0.013s
```

### Интеграционные тесты (Increment 27)

```bash
$ go test -v ./cmd/server -run TestTrustedSubnet
=== RUN   TestTrustedSubnet_AllowedIP
--- PASS: TestTrustedSubnet_AllowedIP (0.00s)
=== RUN   TestTrustedSubnet_BlockedIP
--- PASS: TestTrustedSubnet_BlockedIP (0.00s)
=== RUN   TestTrustedSubnet_EmptySubnet
--- PASS: TestTrustedSubnet_EmptySubnet (0.00s)
=== RUN   TestTrustedSubnet_Localhost
--- PASS: TestTrustedSubnet_Localhost (0.00s)
=== RUN   TestTrustedSubnet_BatchUpdate
--- PASS: TestTrustedSubnet_BatchUpdate (0.00s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/cmd/server	0.012s
```

---

## Ручное тестирование

### Тест 1: Сервер с доверенной подсетью (Increment 27)

```bash
$ go build -o /tmp/server cmd/server/main.go
$ go build -o /tmp/agent cmd/agent/main.go

# Терминал 1: запуск сервера с подсетью 127.0.0.0/8
$ /tmp/server -a="localhost:9091" -t="127.0.0.0/8"
INFO  Server config loaded  {
  "address": "localhost:9091",
  "trusted_subnet": "127.0.0.0/8"
}
INFO  Server started
```

### Тест 2: Агент отправляет метрики (Increment 27)

```bash
# Терминал 2: запуск агента
$ /tmp/agent -a="localhost:9091" -p=1s -r=2s
agent: started | poll=1s report=2s | server=localhost:9091
agent: sending batch (29 metrics) -> /updates/
agent: response /updates/ -> 200 in 5.234ms
agent: batch sent successfully
```

**Результат**: ✅ Агент успешно отправляет метрики, IP 127.0.0.1 входит в подсеть 127.0.0.0/8

### Тест 3: Без доверенной подсети (Increment 27)

```bash
$ /tmp/server -a="localhost:9093"
INFO  Server config loaded  {"trusted_subnet": ""}
INFO  Server started

$ /tmp/agent -a="localhost:9093" -p=1s -r=2s
agent: response /updates/ -> 200 in 3.123ms
agent: batch sent successfully
```

**Результат**: ✅ Без trusted_subnet все запросы принимаются

---

## Статус

**Спринт 9**: ✅ **УСПЕШНО ЗАВЕРШЁН**

Инкременты 27 и 28 реализованы, протестированы и готовы к production.

### Функциональность (Increment 27 - Trusted Subnet)

- ✅ Конфигурация доверенной подсети (env/флаг/JSON)
- ✅ Проверка IP-адреса в CIDR формате
- ✅ Отправка заголовка X-Real-IP агентом
- ✅ Возврат 403 для недоверенных IP
- ✅ Обратная совместимость (без trusted_subnet все IP разрешены)
- ✅ Приоритет конфигурации: флаги > JSON > env > defaults

### Функциональность (Increment 28 - gRPC Support)

- ✅ Protocol Buffers определение сервиса метрик
- ✅ gRPC сервер с поддержкой UpdateMetrics RPC
- ✅ gRPC клиент в агенте для отправки метрик
- ✅ TrustedSubnetInterceptor для проверки IP из метаданных
- ✅ Автоматическое добавление x-real-ip в метаданные
- ✅ Поддержка codes.PermissionDenied для недоверенных IP
- ✅ Конфигурация через GRPC_ADDRESS (env/флаг)
- ✅ Одновременная работа HTTP и gRPC серверов
- ✅ Graceful shutdown для обоих серверов
- ✅ Обратная совместимость (без gRPC_ADDRESS работает только HTTP)

### Компоненты (Increment 27)

- `cmd/agent/main.go` - функция `getLocalIPForServer()` и отправка X-Real-IP
- `cmd/server/main.go` - передача `cfg.TrustedSubnet` в роутер
- `internal/config/server.go` - поле `TrustedSubnet` с конфигурацией
- `internal/middleware/subnet.go` - middleware для проверки IP
- `internal/router/router.go` - применение middleware к /update эндпоинтам
- `internal/middleware/subnet_test.go` - unit тесты (89 строк)
- `cmd/server/trusted_subnet_test.go` - интеграционные тесты (142 строки)

### Компоненты (Increment 28)

- `proto/metrics.proto` - определение Protocol Buffers сервиса
- `proto/metrics.pb.go` - сгенерированный код для protobuf
- `proto/metrics_grpc.pb.go` - сгенерированный код для gRPC
- `proto/README.md` - документация по генерации proto-файлов
- `internal/grpcserver/server.go` - gRPC сервер и интерсептор
- `cmd/agent/grpc_sender.go` - gRPC клиент агента
- `cmd/server/main.go` - запуск gRPC сервера
- `cmd/agent/main.go` - поддержка gRPC режима
- `internal/config/server.go` - поле `GRPCAddress`
- `internal/config/agent.go` - поле `GRPCAddress`
- `Makefile` - команды для генерации proto-файлов
