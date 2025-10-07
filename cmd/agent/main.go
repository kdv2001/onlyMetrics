package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	"github.com/kdv2001/onlyMetrics/internal/usecases/agent"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

func main() {
	httpClient := &http.Client{
		Timeout: time.Second * 5,
	}
	zapLog, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to init logger: %w", err)
	}

	ctx := logger.ToContext(context.Background(), zapLog.Sugar())

	parsedFlags, err := initFlags()
	if err != nil {
		log.Fatal(err)
	}

	metric := agent.NewMetricsUpdater(ctx, parsedFlags.pollInterval)
	metricsHTTPClient := metricsHTTP.NewBodyClient(
		httpClient,
		parsedFlags.serverAddr,
		metricsHTTP.CompresGZIPOpt(),
		metricsHTTP.WithSHA256Opt(parsedFlags.cryptKey),
	)

	metricsUC := agent.NewUseCase(metricsHTTPClient, metric, parsedFlags.reportInterval, parsedFlags.maxGoroutineNum)
	_ = metricsUC.SendMetrics(context.TODO())
}
