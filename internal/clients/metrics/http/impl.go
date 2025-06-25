package http

import (
	"bytes"
	"context"
	"encoding/json"
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

// BodyClient клиент для отправки метрик на сервер в теле запроса в формате JSON.
type BodyClient struct {
	client    httpClient
	serverURL url.URL
}

// NewBodyClient ...
func NewBodyClient(client httpClient, serverURL url.URL) *BodyClient {
	return &BodyClient{
		client:    client,
		serverURL: serverURL,
	}
}

// SendGauge ...
func (c *BodyClient) SendGauge(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

// SendCounter ...
func (c *BodyClient) SendCounter(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

func (c *BodyClient) send(ctx context.Context, value domain.MetricValue) error {
	sendMetricURL := c.serverURL.JoinPath("update")
	type metric struct {
		ID    string   `json:"id"`              // Имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
		Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
	}

	m := metric{
		ID:    value.Name,
		MType: value.Type.String(),
	}

	switch value.Type {
	case domain.GaugeMetricType:
		m.Value = &value.GaugeValue
	case domain.CounterMetricType:
		m.Delta = &value.CounterValue
	default:
		return fmt.Errorf("unknown metric type: %v", value.Type)
	}

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	buf := bytes.NewReader(b)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendMetricURL.String(), buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
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
