package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"

	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	"github.com/kdv2001/onlyMetrics/internal/usecases/agent"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

var buildVersion string
var buildDate string
var buildCommit string

const na = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", opIf(buildVersion != "", buildVersion, na))
	fmt.Printf("Build date: %s\n", opIf(buildDate != "", buildDate, na))
	fmt.Printf("Build commit: %s\n", opIf(buildCommit != "", buildCommit, na))

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

func opIf[T comparable](cond bool, a T, b T) T {
	if cond {
		return a
	}

	return b
}
