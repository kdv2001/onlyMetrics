package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/kdv2001/onlyMetrics/internal/clients"
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

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer cancel()

	parsedFlags, err := initFlags()
	if err != nil {
		log.Fatal(err)
	}

	transport, err := getHTTPTransport(parsedFlags.SymmetricEncryptionKey)
	if err != nil {
		log.Fatal(err)
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 5,
		Transport: transport,
	}

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to init logger: %w", err)
	}

	sugarLogger := zapLog.Sugar()
	ctx = logger.ToContext(ctx, sugarLogger)

	metric := agent.NewMetricsUpdater(ctx, parsedFlags.PollInterval.asTimeDuration())

	scheme := clients.HTTP
	if parsedFlags.SymmetricEncryptionKey != "" {
		scheme = clients.HTTPS
	}

	metricsHTTPClient := metricsHTTP.NewBodyClient(
		httpClient,
		parsedFlags.ServerAddr,
		metricsHTTP.CompresGZIPOpt(),
		metricsHTTP.WithSHA256Opt(parsedFlags.CryptKey),
		metricsHTTP.SetRequestScheme(scheme),
	)

	metricsUC := agent.NewUseCase(metricsHTTPClient,
		metric,
		parsedFlags.ReportInterval.asTimeDuration(),
		parsedFlags.MaxGoroutineNum)
	err = metricsUC.SendMetrics(ctx)
	if err != nil {
		sugarLogger.Errorf("failed to send metrics: %v", err)
	}
}

func opIf[T comparable](cond bool, a T, b T) T {
	if cond {
		return a
	}

	return b
}

func getHTTPTransport(TLSCertificatePath string) (http.RoundTripper, error) {
	transport := http.DefaultTransport
	if TLSCertificatePath == "" {
		return transport, nil
	}

	caCert, err := os.ReadFile(TLSCertificatePath)
	if err != nil {
		log.Fatalf("Error reading server certificate: %v", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	return &http.Transport{
		TLSClientConfig: tlsConfig,
	}, nil

}
