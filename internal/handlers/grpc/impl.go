package grpc

import (
	"context"

	metrics_adapter "github.com/kdv2001/onlyMetrics/internal/adapters/grpc/metrics"
	"github.com/kdv2001/onlyMetrics/internal/domain"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
)

type useCases interface {
	UpdateMetric(ctx context.Context, value domain.MetricValue) error
	GetMetric(ctx context.Context, value domain.MetricType,
		name string) (domain.MetricValue, error)
	GetAllMetrics(ctx context.Context) ([]domain.MetricValue, error)
	Ping(ctx context.Context) error
	UpdateMetrics(ctx context.Context, metrics []domain.MetricValue) error
}

// Handlers grpc обработчики для сбора метрик и их последующего предоставления клиенту.
type Handlers struct {
	pb.UnimplementedOnlyMetricsServer
	metricUseCases useCases
}

// NewHandlers создает объект grpc обработчиков для сбора метрик и их последующего предоставления клиенту.
func NewHandlers(useCases useCases) *Handlers {
	return &Handlers{
		metricUseCases: useCases,
	}
}

// GetAllMetrics обработчик для получения всех метрик.
func (h *Handlers) GetAllMetrics(ctx context.Context, _ *pb.Empty) (*pb.Metrics, error) {
	values, err := h.metricUseCases.GetAllMetrics(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]*pb.Metric, 0, len(values))
	for _, v := range values {
		res = append(res, metrics_adapter.DomainToPB(v))
	}

	return &pb.Metrics{
		Values: res,
	}, nil
}

// GetMetric обработчик для получения метрики.
func (h *Handlers) GetMetric(ctx context.Context, req *pb.Metric) (*pb.Metric, error) {
	t, err := metrics_adapter.PBToDomainType(req.GetType())
	if err != nil {
		return nil, err
	}

	val, err := h.metricUseCases.GetMetric(ctx, t, req.Name)
	if err != nil {
		return nil, err
	}

	res := metrics_adapter.DomainToPB(val)

	return res, nil
}

// UpdateMetric обработчик для обновления метрики.
func (h *Handlers) UpdateMetric(ctx context.Context, req *pb.Metric) (*pb.Empty, error) {
	r, err := metrics_adapter.PBToDomain(req)
	if err != nil {
		return nil, err
	}

	err = h.metricUseCases.UpdateMetric(ctx, r)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// UpdateMetrics обработчик для обновления нескольких метрик.
func (h *Handlers) UpdateMetrics(ctx context.Context, req *pb.Metrics) (*pb.Empty, error) {
	r := make([]domain.MetricValue, 0, len(req.GetValues()))
	for _, v := range req.GetValues() {
		vv, err := metrics_adapter.PBToDomain(v)
		if err != nil {
			return nil, err
		}
		r = append(r, vv)
	}

	if err := h.metricUseCases.UpdateMetrics(ctx, r); err != nil {
		return nil, err
	}

	return nil, nil
}

// Ping обработчик для проверки работоспособности сервиса.
func (h *Handlers) Ping(ctx context.Context, _ *pb.Empty) (*pb.Status, error) {
	if err := h.metricUseCases.Ping(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}
