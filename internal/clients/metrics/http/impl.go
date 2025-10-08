// Package http предоставляет методы для отправки метрик на сервер по http.
package http

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/kdv2001/onlyMetrics/internal/domain"
)

const retryNums = 3

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client клиент для отправки метрик.
type Client struct {
	client    httpClient
	serverURL url.URL
}

// NewClient создает клиент для отправки метрик.
func NewClient(client httpClient, serverURL url.URL) *Client {
	return &Client{
		client:    client,
		serverURL: serverURL,
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
	hh        func([]byte) ([]byte, error)
	withGzip  bool
}

// clientOption опция клиента.
type clientOption func(c *BodyClient)

// CompresGZIPOpt включает gzip сжатие.
func CompresGZIPOpt() clientOption {
	return func(c *BodyClient) {
		c.withGzip = true
	}
}

// WithSHA256Opt включает добавление подписи sha256.
func WithSHA256Opt(key string) clientOption {
	if key == "" {
		return func(c *BodyClient) {
			c.hh = nil
		}
	}

	return func(c *BodyClient) {
		c.hh = func(body []byte) ([]byte, error) {
			hh := hmac.New(sha256.New, []byte(key))
			if _, err := hh.Write(body); err != nil {
				return nil, err

			}
			return hh.Sum(nil), nil
		}
	}
}

// NewBodyClient создает клиент для отправки метрик на сервер в теле запроса в формате JSON.
func NewBodyClient(client httpClient, serverURL url.URL, opts ...clientOption) *BodyClient {
	bc := &BodyClient{
		client:    client,
		serverURL: serverURL,
	}

	for _, opt := range opts {
		opt(bc)
	}

	return bc
}

// SendGauge отправляет метрику типа "Градусник".
func (c *BodyClient) SendGauge(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

// SendCounter отправляет метрику типа "Счетчик".
func (c *BodyClient) SendCounter(ctx context.Context, value domain.MetricValue) error {
	return c.send(ctx, value)
}

func (c *BodyClient) send(ctx context.Context, value domain.MetricValue) error {
	sendMetricURL := c.serverURL.JoinPath("update")
	type metric struct {
		Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
		ID    string   `json:"id"`              // Имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
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

	var buf *bytes.Buffer
	switch {
	case c.withGzip:
		buf = bytes.NewBuffer(nil)
		gzipWriter := gzip.NewWriter(buf)
		_, err = gzipWriter.Write(b)
		if err != nil {
			return err
		}
		err = gzipWriter.Close()
		if err != nil {
			return err
		}
	default:
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendMetricURL.String(), buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
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

// SendMetrics отправляет набор метрик.
func (c *BodyClient) SendMetrics(ctx context.Context, metrics []domain.MetricValue) error {
	sendMetricURL := c.serverURL.JoinPath("updates")
	type metric struct {
		Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
		ID    string   `json:"id"`              // Имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	}

	res := make([]metric, 0, len(metrics))
	for _, dm := range metrics {
		m := metric{
			ID:    dm.Name,
			MType: dm.Type.String(),
		}

		switch dm.Type {
		case domain.GaugeMetricType:
			m.Value = &dm.GaugeValue
		case domain.CounterMetricType:
			m.Delta = &dm.CounterValue
		default:
			return fmt.Errorf("unknown metric type: %v", dm.Type)
		}

		res = append(res, m)
	}

	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	var buf *bytes.Buffer
	switch {
	case c.withGzip:
		buf = bytes.NewBuffer(nil)
		gzipWriter := gzip.NewWriter(buf)
		_, err = gzipWriter.Write(b)
		if err != nil {
			return err
		}
		err = gzipWriter.Close()
		if err != nil {
			return err
		}
	default:
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendMetricURL.String(), buf)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if c.hh != nil {
		bufSHA, err := c.hh(buf.Bytes())
		if err != nil {
			return err
		}
		str := hex.EncodeToString(bufSHA)
		req.Header.Set("HashSHA256", str)
	}

	var timeSleep time.Duration
	for i := 0; i < retryNums; i++ {
		time.Sleep(timeSleep)
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		switch resp.StatusCode {
		case http.StatusLocked:
			timeSleep = time.Duration(i*2+1) * time.Second
			continue
		case http.StatusOK:
			return nil
		default:
			return fmt.Errorf("server error")
		}
	}

	return nil
}
