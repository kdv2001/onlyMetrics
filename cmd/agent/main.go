package main

import (
	"context"
	"log"
	"net/http"
	"time"

	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	"github.com/kdv2001/onlyMetrics/internal/usecases/agent"
)

func main() {
	httpClient := &http.Client{
		Timeout: time.Second * 5,
	}

	parsedFlags, err := initFlags()
	if err != nil {
		log.Fatal(err)
	}

	metric := agent.NewMetricsUpdater(parsedFlags.pollInterval)
	metricsHTTPClient := metricsHTTP.NewClient(httpClient, parsedFlags.serverAddr)
	metricsUC := agent.NewUseCase(metricsHTTPClient, metric, parsedFlags.reportInterval)
	_ = metricsUC.SendMetrics(context.TODO())
}
