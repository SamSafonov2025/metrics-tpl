# Protocol Buffers для метрик

## Установка protoc

### Linux
```bash
# Скачайте последнюю версию protoc
wget https://github.com/protocolbuffers/protobuf/releases/download/v28.3/protoc-28.3-linux-x86_64.zip
unzip protoc-28.3-linux-x86_64.zip -d $HOME/.local
export PATH="$PATH:$HOME/.local/bin"
```

### Windows
Скачайте protoc с https://github.com/protocolbuffers/protobuf/releases и добавьте в PATH.

### macOS
```bash
brew install protobuf
```

## Установка плагинов Go

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Генерация кода

```bash
make proto
```

Будут созданы файлы:
- `proto/metrics.pb.go` - код для сериализации/десериализации
- `proto/metrics_grpc.pb.go` - код gRPC сервиса
