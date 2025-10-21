package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
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

	parsedFlags, err := initFlags()
	if err != nil {
		log.Fatal(err)
	}

	transport, err := getHTTPTransport(parsedFlags.symmetricEncryptionKey)
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

	ctx := logger.ToContext(context.Background(), zapLog.Sugar())

	metric := agent.NewMetricsUpdater(ctx, parsedFlags.pollInterval)

	scheme := clients.HTTP
	if parsedFlags.symmetricEncryptionKey != "" {
		scheme = clients.HTTPS
	}

	metricsHTTPClient := metricsHTTP.NewBodyClient(
		httpClient,
		parsedFlags.serverAddr,
		metricsHTTP.CompresGZIPOpt(),
		metricsHTTP.WithSHA256Opt(parsedFlags.cryptKey),
		metricsHTTP.SetRequestScheme(scheme),
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
