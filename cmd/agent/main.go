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
	"github.com/kdv2001/onlyMetrics/pkg/network"
	"github.com/kdv2001/onlyMetrics/pkg/operators"
)

var buildVersion string
var buildDate string
var buildCommit string

const na = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", operators.OpIf(buildVersion != "", buildVersion, na))
	fmt.Printf("Build date: %s\n", operators.OpIf(buildDate != "", buildDate, na))
	fmt.Printf("Build commit: %s\n", operators.OpIf(buildCommit != "", buildCommit, na))

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

	metric := agent.NewMetricsUpdater(ctx, parsedFlags.PollInterval.AsTimeDuration())

	scheme := clients.HTTP
	if parsedFlags.SymmetricEncryptionKey != "" {
		scheme = clients.HTTPS
	}

	ip, err := network.GetLocalIP()
	if err != nil {
		sugarLogger.Errorf("failed to get local IP address %v", err)
	}

	metricsHTTPClient := metricsHTTP.NewBodyClient(
		httpClient,
		parsedFlags.ServerAddr.AsURL(),
		metricsHTTP.CompresGZIPOpt(),
		metricsHTTP.WithSHA256Opt(parsedFlags.CryptKey),
		metricsHTTP.SetRequestScheme(scheme),
		metricsHTTP.SetRealIPHeader(ip),
	)

	metricsUC := agent.NewUseCase(metricsHTTPClient,
		metric,
		parsedFlags.ReportInterval.AsTimeDuration(),
		parsedFlags.MaxGoroutineNum)
	err = metricsUC.SendMetrics(ctx)
	if err != nil {
		log.Fatalf("failed to send metrics: %v", err)
	}
}

func getHTTPTransport(TLSCertificatePath string) (http.RoundTripper, error) {
	transport := http.DefaultTransport
	if TLSCertificatePath == "" {
		return transport, nil
	}

	caCert, err := os.ReadFile(TLSCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("error reading server certificate: %w", err)
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
