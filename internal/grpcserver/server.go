// Package grpcserver содержит реализацию gRPC сервера для метрик
package grpcserver

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/SamSafonov2025/metrics-tpl/internal/dto"
	"github.com/SamSafonov2025/metrics-tpl/internal/service"
	pb "github.com/SamSafonov2025/metrics-tpl/proto"
)

// MetricsGRPCServer реализует gRPC сервер для метрик
type MetricsGRPCServer struct {
	pb.UnimplementedMetricsServer
	service service.MetricsService
}

// NewMetricsGRPCServer создает новый gRPC сервер для метрик
func NewMetricsGRPCServer(svc service.MetricsService) *MetricsGRPCServer {
	return &MetricsGRPCServer{
		service: svc,
	}
}

// UpdateMetrics обрабатывает пакетное обновление метрик через gRPC
func (s *MetricsGRPCServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	// Конвертируем proto метрики в dto метрики
	dtoMetrics := make([]dto.Metrics, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		dtoMetric := protoToDTO(m)
		dtoMetrics = append(dtoMetrics, dtoMetric)
	}

	// Вызываем метод сервиса для обновления метрик
	if err := s.service.UpdateBatch(ctx, dtoMetrics); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}

// protoToDTO конвертирует proto.Metric в dto.Metrics
func protoToDTO(m *pb.Metric) dto.Metrics {
	result := dto.Metrics{
		ID: m.Id,
	}

	switch m.Type {
	case pb.Metric_GAUGE:
		result.MType = "gauge"
		result.Value = &m.Value
	case pb.Metric_COUNTER:
		result.MType = "counter"
		result.Delta = &m.Delta
	}

	return result
}

// TrustedSubnetInterceptor создает унарный интерсептор для проверки доверенной подсети
func TrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если trusted_subnet пустой, пропускаем проверку
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		// Получаем метаданные из контекста
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		// Извлекаем IP-адрес из метаданных с ключом "x-real-ip"
		ips := md.Get("x-real-ip")
		if len(ips) == 0 {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip in metadata")
		}

		clientIP := ips[0]

		// Проверяем, входит ли IP в доверенную подсеть
		if !isIPInSubnet(clientIP, trustedSubnet) {
			return nil, status.Error(codes.PermissionDenied, "IP address not in trusted subnet")
		}

		// Продолжаем обработку запроса
		return handler(ctx, req)
	}
}

// isIPInSubnet проверяет, входит ли IP-адрес в указанную подсеть CIDR
func isIPInSubnet(ipStr, cidr string) bool {
	// Парсим CIDR
	_, subnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}

	// Парсим IP-адрес
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}

	// Проверяем, содержится ли IP в подсети
	return subnet.Contains(ip)
}
