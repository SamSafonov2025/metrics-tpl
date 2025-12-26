// cmd/agent/grpc_sender.go
package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/SamSafonov2025/metrics-tpl/proto"
)

// MetricsGRPCSender отправляет метрики через gRPC
type MetricsGRPCSender struct {
	grpcAddress string
	client      pb.MetricsClient
	conn        *grpc.ClientConn
}

// NewMetricsGRPCSender создает новый gRPC отправитель метрик
func NewMetricsGRPCSender(grpcAddress string) (*MetricsGRPCSender, error) {
	// Устанавливаем соединение с gRPC сервером
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := pb.NewMetricsClient(conn)

	return &MetricsGRPCSender{
		grpcAddress: grpcAddress,
		client:      client,
		conn:        conn,
	}, nil
}

// Close закрывает соединение с gRPC сервером
func (s *MetricsGRPCSender) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// SendBatchGRPCCtx отправляет батч метрик через gRPC
func (s *MetricsGRPCSender) SendBatchGRPCCtx(ctx context.Context, batch []Metrics) error {
	if len(batch) == 0 {
		return nil
	}

	// Конвертируем Metrics в proto.Metric
	protoMetrics := make([]*pb.Metric, 0, len(batch))
	for _, m := range batch {
		protoMetric := &pb.Metric{
			Id: m.ID,
		}

		switch m.MType {
		case "gauge":
			protoMetric.Type = pb.Metric_GAUGE
			if m.Value != nil {
				protoMetric.Value = *m.Value
			}
		case "counter":
			protoMetric.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				protoMetric.Delta = *m.Delta
			}
		}

		protoMetrics = append(protoMetrics, protoMetric)
	}

	// Получаем локальный IP для сервера
	localIP := getLocalIPForServer(s.grpcAddress)

	// Создаем метаданные с IP-адресом
	md := metadata.New(map[string]string{
		"x-real-ip": localIP,
	})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Отправляем запрос
	req := &pb.UpdateMetricsRequest{
		Metrics: protoMetrics,
	}

	fmt.Printf("agent: sending gRPC batch (%d metrics) to %s with IP=%s\n", len(batch), s.grpcAddress, localIP)
	start := time.Now()
	_, err := s.client.UpdateMetrics(ctx, req)
	dur := time.Since(start)

	if err != nil {
		fmt.Printf("agent: gRPC send failed in %s: %v\n", dur, err)
		return err
	}

	fmt.Printf("agent: gRPC batch sent successfully in %s\n", dur)
	return nil
}

// getLocalIPForServerGRPC возвращает локальный IP-адрес для gRPC соединения
// (можно переиспользовать функцию из main.go, но дублируем для изоляции)
func getLocalIPForServerGRPC(serverAddr string) string {
	// Извлекаем хост из адреса сервера
	host := serverAddr
	if idx := strings.Index(serverAddr, ":"); idx >= 0 {
		host = serverAddr[:idx]
	}

	// Пытаемся установить UDP-соединение с сервером
	conn, err := net.Dial("udp", net.JoinHostPort(host, "80"))
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
