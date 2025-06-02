package main

import (
	"context"
	"net/http"
	"net/url"
	"time"

	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	"github.com/kdv2001/onlyMetrics/internal/usecases/agent"
)

func main() {
	httpClient := &http.Client{
		Timeout: time.Second * 5,
	}
	metric := agent.NewMetricsUpdater(time.Second * 2)
	metricsHTTPClient := metricsHTTP.NewClient(httpClient, url.URL{
		Scheme: "http",
		Path:   "localhost:8080",
	})
	metricsUC := agent.NewUseCase(metricsHTTPClient, metric, time.Second*10)
	_ = metricsUC.SendMetrics(context.TODO())
}
