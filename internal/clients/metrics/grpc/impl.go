package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	metrics_adapter "github.com/kdv2001/onlyMetrics/internal/adapters/grpc/metrics"
	"github.com/kdv2001/onlyMetrics/internal/domain"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
)

type OnlyMetricsClient interface {
	GetAllMetrics(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.Metrics, error)
	GetMetric(ctx context.Context, in *pb.Metric, opts ...grpc.CallOption) (*pb.Metric, error)
	UpdateMetric(ctx context.Context, in *pb.Metric, opts ...grpc.CallOption) (*pb.Empty, error)
	UpdateMetrics(ctx context.Context, in *pb.Metrics, opts ...grpc.CallOption) (*pb.Empty, error)
	Ping(ctx context.Context, in *pb.Empty, opts ...grpc.CallOption) (*pb.Status, error)
}

// Client клиент обертка над GRPC
type Client struct {
	client OnlyMetricsClient
}

// NewClient создает клиент обертка над GRPC
func NewClient(client OnlyMetricsClient) *Client {
	return &Client{
		client: client,
	}
}

// SendGauge отправляет метрику типа "Градусник".
func (c *Client) SendGauge(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

// SendCounter отправляет метрику типа "Счетчик".
func (c *Client) SendCounter(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

func (c *Client) send(ctx context.Context, value domain.MetricValue) error {
	req := metrics_adapter.DomainToPB(value)
	if req == nil {
		return fmt.Errorf("unknown metric, type: %v", value.Type)

	}

	_, err := c.client.UpdateMetric(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// SendMetrics отправляет набор метрик.
func (c *Client) SendMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	res := make([]*pb.Metric, 0, len(metrics))
	for _, dm := range metrics {
		pbM := metrics_adapter.DomainToPB(dm)
		if pbM == nil {
			return fmt.Errorf("unknown metric, type: %v", dm.Type)
		}

		res = append(res, pbM)
	}

	_, err := c.client.UpdateMetrics(ctx, &pb.Metrics{
		Values: res,
	})
	if err != nil {
		return err
	}

	return nil
}
