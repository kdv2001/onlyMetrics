package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	client    httpClient
	serverURL url.URL
}

func NewClient(client httpClient, serverURL url.URL) *Client {
	return &Client{
		client:    client,
		serverURL: serverURL,
	}
}

func (c *Client) SendGauge(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

func (c *Client) SendCounter(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

func (c *Client) send(ctx context.Context, value domain.MetricValue) error {
	sendMetricURL := c.serverURL.JoinPath("update", value.Type.String(), value.Name)
	switch value.Type {
	case domain.GaugeMetricType:
		sendMetricURL = sendMetricURL.JoinPath(fmt.Sprint(value.GaugeValue))
	case domain.CounterMetricType:
		sendMetricURL = sendMetricURL.JoinPath(fmt.Sprint(value.CounterValue))
	default:
		return fmt.Errorf("unknown metric type: %v", value.Type)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendMetricURL.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("internal server error")
	}

	return nil
}
