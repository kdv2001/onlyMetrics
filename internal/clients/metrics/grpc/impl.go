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
	UpdateMetric(ctx context.Context, in *pb.UpdateMetricReq, opts ...grpc.CallOption) (*pb.UpdateMetricResp, error)
	UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsReq, opts ...grpc.CallOption) (*pb.UpdateMetricsResp, error)
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
	metrics := metrics_adapter.DomainMetricToPB(value)
	if metrics == nil {
		return fmt.Errorf("unknown metric, type: %v", value.Type)

	}

	req := &pb.UpdateMetricReq{}
	req.SetMetric(metrics)

	_, err := c.client.UpdateMetric(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// SendMetrics отправляет набор метрик.
func (c *Client) SendMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, dm := range metrics {
		pbM := metrics_adapter.DomainMetricToPB(dm)
		if pbM == nil {
			return fmt.Errorf("unknown metric, type: %v", dm.Type)
		}

		pbMetrics = append(pbMetrics, pbM)
	}

	req := &pb.UpdateMetricsReq{}
	reqMetrics := &pb.Metrics{}
	reqMetrics.SetValues(pbMetrics)
	req.SetMetrics(reqMetrics)

	_, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return err
	}

	return nil
}
