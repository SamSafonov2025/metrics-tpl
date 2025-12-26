# Спринт 8 - Результаты тестирования

## Инкременты

- **Инкремент 24**: RSA шифрование данных
- **Инкремент 25**: JSON конфигурация
- **Инкремент 26**: Graceful shutdown

---

## Инкремент 24 - RSA шифрование

### Запуск с шифрованием

**Сервер**:
```bash
$ go run cmd/server/main.go -crypto-key=keys/private.pem
INFO  Loaded RSA private key  {"path": "keys/private.pem"}
INFO  Server started
```

**Агент**:
```bash
$ go run cmd/agent/main.go -crypto-key=keys/public.pem
agent: loaded RSA public key from keys/public.pem
agent: started | rsa=true | workers=4
agent: encrypted data: gz=394B -> enc=2048B
agent: response /updates/ -> 200 in 45ms
agent: batch sent successfully
```

### Результат

| Проверка | Статус |
|----------|--------|
| Генерация ключей | ✅ PASS |
| Шифрование на агенте | ✅ PASS |
| Расшифровка на сервере | ✅ PASS |
| Обратная совместимость | ✅ PASS |

---

## Инкремент 25 - JSON конфигурация

### Файлы конфигурации

**configs/server.json**:
```json
{
  "address": "localhost:8080",
  "restore": true,
  "store_interval": "300s",
  "store_file": "/tmp/metrics-db.json",
  "database_dsn": "",
  "crypto_key": "keys/private.pem"
}
```

**configs/agent.json**:
```json
{
  "address": "localhost:8080",
  "report_interval": "10s",
  "poll_interval": "2s",
  "crypto_key": "keys/public.pem"
}
```

### Запуск с JSON

**Сервер**:
```bash
$ go run cmd/server/main.go -c=configs/server.json
INFO  Server config loaded  {
  "address": "localhost:8080",
  "store_interval": "5m0s",
  "restore": true
}
INFO  Server started
```

**Агент**:
```bash
$ go run cmd/agent/main.go -config=configs/agent.json
INFO  Agent config loaded  {
  "server_address": "localhost:8080",
  "poll_interval": "2s",
  "report_interval": "10s"
}
agent: started
```

### Проверка приоритетов

**Флаг переопределяет JSON**:
```bash
$ go run cmd/server/main.go -c=configs/server.json -a=0.0.0.0:9090
INFO  Server config loaded  {"address": "0.0.0.0:9090"}  # <- из флага
```

**ENV переопределяет JSON**:
```bash
$ export ADDRESS="0.0.0.0:8888"
$ go run cmd/server/main.go -c=configs/server.json
INFO  Server config loaded  {"address": "0.0.0.0:8888"}  # <- из env
```

### Результат

| Проверка | Статус |
|----------|--------|
| Загрузка JSON сервера | ✅ PASS |
| Загрузка JSON агента | ✅ PASS |
| Приоритет Flag > JSON | ✅ PASS |
| Приоритет ENV > JSON | ✅ PASS |
| Обратная совместимость | ✅ PASS |

---

## Инкремент 26 - Graceful Shutdown

### Автоматические тесты

**Тесты агента**:
```bash
$ go test -v ./cmd/agent -run TestAgent
=== RUN   TestAgentGracefulShutdown
agent: started | poll=100ms report=200ms | workers=2
agent: batch sent successfully
agent: batch sent successfully
agent: received shutdown signal, finishing current work...
agent: all metrics sent, shutdown completed successfully
    graceful_shutdown_test.go:56: Agent shutdown successfully after sending 6 requests
    graceful_shutdown_test.go:63: Total requests sent: 6
--- PASS: TestAgentGracefulShutdown (0.50s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/cmd/agent	0.514s
```

**Тесты сервера**:
```bash
$ go test -v ./cmd/server -run TestServer
=== RUN   TestServerGracefulShutdown
    graceful_shutdown_test.go:77: Storage file created successfully
--- PASS: TestServerGracefulShutdown (0.10s)
=== RUN   TestServerShutdownWithRequest
    graceful_shutdown_test.go:129: Request completed successfully during shutdown
--- PASS: TestServerShutdownWithRequest (0.60s)
=== RUN   TestDataSaveOnShutdown
    graceful_shutdown_test.go:173: Storage file contains 85 bytes
--- PASS: TestDataSaveOnShutdown (0.01s)
PASS
ok  	github.com/SamSafonov2025/metrics-tpl/cmd/server	0.718s
```
