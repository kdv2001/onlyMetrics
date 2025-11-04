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
	"google.golang.org/grpc"
	grpcCredentials "google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/kdv2001/onlyMetrics/internal/clients"
	metricsGRPC "github.com/kdv2001/onlyMetrics/internal/clients/metrics/grpc"
	metricsHTTP "github.com/kdv2001/onlyMetrics/internal/clients/metrics/http"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
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

	zapLog, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("failed to init logger: %w", err)
	}

	sugarLogger := zapLog.Sugar()
	ctx = logger.ToContext(ctx, sugarLogger)

	metric := agent.NewMetricsUpdater(ctx, parsedFlags.PollInterval.AsTimeDuration())

	var metricsSender agent.SendClient
	if parsedFlags.GRPCServerAddr.String() != "" {
		metricsSender, err = getGRPCClient(ctx, parsedFlags)
	} else {
		metricsSender, err = getHttpClient(ctx, parsedFlags)
	}
	if err != nil {
		log.Fatal(err)
	}

	metricsUC := agent.NewUseCase(metricsSender,
		metric,
		parsedFlags.ReportInterval.AsTimeDuration(),
		parsedFlags.MaxGoroutineNum)
	err = metricsUC.SendMetrics(ctx)
	if err != nil {
		log.Fatalf("failed to send metrics: %v", err)
	}
}

func getGRPCClient(_ context.Context, parsedFlags *flags) (*metricsGRPC.Client, error) {
	opts := make([]grpc.DialOption, 0)
	if parsedFlags.SymmetricEncryptionKey != "" {
		tlsConf, err := grpcCredentials.NewClientTLSFromFile(parsedFlags.SymmetricEncryptionKey, "")
		if err != nil {
			return nil, err
		}

		opts = append(opts, grpc.WithTransportCredentials(tlsConf))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithUnaryInterceptor(metricsGRPC.NewErrorInterceptor()))

	addr := ""
	if parsedFlags.GRPCServerAddr.Path != "" {
		addr = parsedFlags.GRPCServerAddr.Path
	}

	if parsedFlags.GRPCServerAddr.Port() != "" {
		addr = fmt.Sprintf("%s:%s", addr, parsedFlags.GRPCServerAddr.Port())
	}

	conn, err := grpc.NewClient(addr,
		opts...)
	if err != nil {
		return nil, err
	}

	client := pb.NewOnlyMetricsClient(conn)

	return metricsGRPC.NewClient(client), nil
}

func getHttpClient(ctx context.Context, parsedFlags *flags) (*metricsHTTP.BodyClient, error) {
	transport, err := getHTTPTransport(parsedFlags.SymmetricEncryptionKey)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout:   time.Second * 5,
		Transport: transport,
	}

	scheme := clients.HTTP
	if parsedFlags.SymmetricEncryptionKey != "" {
		scheme = clients.HTTPS
	}

	ip, err := network.GetLocalIP()
	if err != nil {
		logger.Errorf(ctx, "failed to get local IP address %v", err)
	}

	metricsHTTPClient := metricsHTTP.NewBodyClient(
		httpClient,
		parsedFlags.ServerAddr.AsURL(),
		metricsHTTP.CompresGZIPOpt(),
		metricsHTTP.WithSHA256Opt(parsedFlags.CryptKey),
		metricsHTTP.SetRequestScheme(scheme),
		metricsHTTP.SetRealIPHeader(ip),
	)

	return metricsHTTPClient, nil
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
