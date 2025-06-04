package main

import (
	"context"
	"net/http"
	"time"

	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	"github.com/kdv2001/onlyMetrics/internal/usecases/agent"
)

func main() {
	httpClient := &http.Client{
		Timeout: time.Second * 5,
	}

	parsedFlags := initFlags()

	metric := agent.NewMetricsUpdater(parsedFlags.pollInterval)
	metricsHTTPClient := metricsHTTP.NewClient(httpClient, parsedFlags.serverAddr)
	metricsUC := agent.NewUseCase(metricsHTTPClient, metric, parsedFlags.reportInterval)
	_ = metricsUC.SendMetrics(context.TODO())
}
