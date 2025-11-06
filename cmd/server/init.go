package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"path"
	"sync"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/kdv2001/onlyMetrics/docs"
	pb "github.com/kdv2001/onlyMetrics/internal/gen/protogen/only_metrics/grpc"
	serviceGRPC "github.com/kdv2001/onlyMetrics/internal/handlers/grpc"
	sericeHttp "github.com/kdv2001/onlyMetrics/internal/handlers/http"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/memory"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/postgres"
	"github.com/kdv2001/onlyMetrics/internal/usecases/metrics"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

func initService() error {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer cancel()

	parsedFlags, err := initFlags()
	if err != nil {
		return fmt.Errorf("failed to init flags: %w", err)
	}

	var metricsStorage metrics.MetricStorage
	if parsedFlags.PostgresDSN != "" {
		conn, iErr := pgx.Connect(ctx, parsedFlags.PostgresDSN)
		if iErr != nil {
			return iErr
		}
		defer conn.Close(ctx)

		err = conn.Ping(ctx)
		if err != nil {
			return err
		}
		postgresStorage, iErr := postgres.NewStorage(ctx, conn)
		if iErr != nil {
			return iErr
		}
		defer postgresStorage.Close(ctx)
		metricsStorage = postgresStorage
	} else {
		memoryStorage := memory.NewStorage(ctx,
			parsedFlags.FileStoragePath,
			parsedFlags.StoreInterval.AsTimeDuration(),
			parsedFlags.RestoreData)
		defer memoryStorage.Close(ctx)
		metricsStorage = memoryStorage
	}

	log, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to init looger: %w", err)
	}
	sugarLogger := log.Sugar()

	metricsUC := metrics.NewUseCases(metricsStorage)
	httpHandlers := sericeHttp.NewHandlers(metricsUC)
	grpcHandlers := serviceGRPC.NewHandlers(metricsUC)

	wg := &sync.WaitGroup{}
	errorChan := make(chan error)
	err = startHTTPServer(
		ctx,
		wg,
		errorChan,
		parsedFlags,
		httpHandlers,
		sugarLogger)
	if err != nil {
		return err
	}

	err = startGRPCServer(
		ctx,
		wg,
		errorChan,
		parsedFlags,
		grpcHandlers,
		sugarLogger)
	if err != nil {
		return err
	}

	finalErrChan := make(chan error)
	go func() {
		var errs []error
		internalErr := <-errorChan
		errs = append(errs, internalErr)
		cancel()

		for internalErr = range errorChan {
			errs = append(errs, internalErr)
		}

		finalErrChan <- errors.Join(errs...)
	}()

	wg.Wait()
	close(errorChan)

	err = <-finalErrChan
	if err != nil {
		return err
	}
	close(finalErrChan)

	return nil
}

func startGRPCServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	errorChan chan error,
	parsedFlags *flags,
	handlers *serviceGRPC.Handlers,
	sugarLogger *zap.SugaredLogger,
) error {
	interceptors := make([]grpc.UnaryServerInterceptor, 0)
	if parsedFlags.TrustedSubnet != "" {
		subnet, err := serviceGRPC.NewSubNetInterceptor(parsedFlags.TrustedSubnet)
		if err != nil {
			return err
		}

		interceptors = append(interceptors, subnet)
	}

	interceptors = append(interceptors,
		serviceGRPC.AddLoggerToContextInterceptor(sugarLogger),
		serviceGRPC.RequestInterceptor(),
		serviceGRPC.ResponseInterceptor(),
		serviceGRPC.NewErrorInterceptor())

	credsOpt, err := getGRPCCreds(parsedFlags)
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		credsOpt,
		grpc.ChainUnaryInterceptor(interceptors...),
	)

	pb.RegisterOnlyMetricsServer(server, handlers)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		server.GracefulStop()
	}()

	listen, err := net.Listen("tcp", parsedFlags.GRPCServerAddr)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = server.Serve(listen)
		if err != nil {
			errorChan <- err
		}
		logger.Infof(ctx, "gRPC server stopped")
	}()

	logger.Infof(ctx, "serving grpc metrics on port %s", parsedFlags.GRPCServerAddr)

	return nil
}

func startHTTPServer(
	ctx context.Context,
	wg *sync.WaitGroup,
	errorChan chan error,
	parsedFlags *flags,
	httpHandlers *sericeHttp.Handlers,
	sugarLogger *zap.SugaredLogger,
) error {
	chiMux := chi.NewMux()

	chiMux.Use(
		sericeHttp.ResponseMiddleware(),
		sericeHttp.RequestMiddleware())

	if parsedFlags.TrustedSubnet != "" {
		mw, err := sericeHttp.NewSubNetMiddleware(parsedFlags.TrustedSubnet)
		if err != nil {
			return fmt.Errorf("failed to init ubNetMiddleware: %w", err)
		}
		chiMux.Use(mw)

	}
	if parsedFlags.CryptKey != "" {
		chiMux.Use(sericeHttp.NewSha256Middleware(parsedFlags.CryptKey))
	}

	chiMux.Use(
		sericeHttp.CompressMiddleware(sericeHttp.GetDefaultAcceptedEncodingData()),
		sericeHttp.DecompressMiddleware(),
		sericeHttp.AddLoggerToContextMiddleware(sugarLogger))

	chiMux.Get("/", httpHandlers.GetAllMetric)

	chiMux.Route("/ping", func(r chi.Router) {
		r.Get("/", httpHandlers.GetPing)
	})

	chiMux.Route("/updates", func(r chi.Router) {
		r.Post("/", httpHandlers.UpdateMetrics)
	})

	chiMux.Route("/update", func(r chi.Router) {
		r.Post("/", httpHandlers.CollectBodyMetric)
		r.Route(fmt.Sprintf("/{%s}/{%s}/{%s}",
			sericeHttp.MetricTypePathKey,
			sericeHttp.MetricNamePathKey,
			sericeHttp.ValuePathKey,
		), func(r chi.Router) {
			r.Post("/", httpHandlers.CollectMetric)
		})
	})

	chiMux.Route("/value", func(r chi.Router) {
		r.Post("/", httpHandlers.GetBodyMetric)
		r.Route(fmt.Sprintf("/{%s}/{%s}",
			sericeHttp.MetricTypePathKey,
			sericeHttp.MetricNamePathKey,
		), func(r chi.Router) {
			r.Get("/", httpHandlers.GetMetric)
		})
	})

	chiMux.Get("/swagger/*", httpSwagger.Handler())

	tlsConfig, err := getTLSConfig(parsedFlags.SymmetricEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to init tls: %w", err)
	}

	server := &http.Server{
		Handler:   chiMux,
		Addr:      parsedFlags.ServerAddr,
		TLSConfig: tlsConfig,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutDownErr := server.Shutdown(ctx)
		if shutDownErr != nil {
			sugarLogger.Errorf("failed to shut down http server: %v", shutDownErr)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if tlsConfig != nil {
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errorChan <- err
			}
		}
		logger.Infof(ctx, "http server stopped")
	}()

	logger.Infof(ctx, "serving http metrics on port %s", parsedFlags.ServerAddr)

	return nil
}

const (
	certificateName = "CERTIFICATE.pem"
	privateKeyName  = "PRIVATE_KEY.pem"
)

func getTLSConfig(privateKeyPath string) (*tls.Config, error) {
	if privateKeyPath == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(
		path.Join(privateKeyPath, certificateName),
		path.Join(privateKeyPath, privateKeyName))
	if err != nil {
		return nil, fmt.Errorf("error reading server certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}

	return tlsConfig, nil
}

func getGRPCCreds(parsedFlags *flags) (grpc.ServerOption, error) {
	var credsOpt = grpc.Creds(insecure.NewCredentials())
	if parsedFlags.SymmetricEncryptionKey == "" {
		return credsOpt, nil
	}

	tlsConf, err := credentials.NewServerTLSFromFile(
		path.Join(parsedFlags.SymmetricEncryptionKey, certificateName),
		path.Join(parsedFlags.SymmetricEncryptionKey, privateKeyName))
	if err != nil {
		return nil, err
	}

	return grpc.Creds(tlsConf), nil
}
