package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"

	_ "github.com/kdv2001/onlyMetrics/docs"
	sericeHttp "github.com/kdv2001/onlyMetrics/internal/handlers/http"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/memory"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/postgres"
	"github.com/kdv2001/onlyMetrics/internal/usecases/metrics"
	"github.com/kdv2001/onlyMetrics/pkg/logger"
)

func initService() error {
	ctx := context.Background()
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
			parsedFlags.StoreInterval.asTimeDuration(),
			parsedFlags.RestoreData)
		defer memoryStorage.Close(ctx)
		metricsStorage = memoryStorage
	}

	metricsUC := metrics.NewUseCases(metricsStorage)
	httpHandlers := sericeHttp.NewHandlers(metricsUC)

	chiMux := chi.NewMux()
	log, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to init looger: %w", err)
	}
	if parsedFlags.CryptKey != "" {
		chiMux.Use(sericeHttp.NewSha256Middleware(parsedFlags.CryptKey))
	}

	sugarLogger := log.Sugar()
	chiMux.Use(
		sericeHttp.CompressMiddleware(sericeHttp.GetDefaultAcceptedEncodingData()),
		sericeHttp.DecompressMiddleware(),
		sericeHttp.AddLoggerToContextMiddleware(sugarLogger),
		sericeHttp.ResponseMiddleware(),
		sericeHttp.RequestMiddleware())

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

	logger.Infof(ctx, "serving metrics on port %s", parsedFlags.ServerAddr)

	tlsConfig, err := getTLSConfig(parsedFlags.SymmetricEncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to init tls: %w", err)
	}

	server := http.Server{
		Handler:   chiMux,
		Addr:      parsedFlags.ServerAddr,
		TLSConfig: tlsConfig,
	}

	if tlsConfig != nil {
		err = server.ListenAndServeTLS("", "")
	} else {
		err = server.ListenAndServe()
	}
	if err != nil {
		return err
	}

	return nil
}

func getTLSConfig(privateKeyPath string) (*tls.Config, error) {
	if privateKeyPath == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(path.Join(privateKeyPath, "CERTIFICATE.pem"),
		path.Join(privateKeyPath, "PRIVATE_KEY.pem"))
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
