package main

import (
	"context"
	"fmt"
	"net/http"

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
	if parsedFlags.postgresDSN != "" {
		conn, iErr := pgx.Connect(ctx, parsedFlags.postgresDSN)
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
		memoryStorage := memory.NewStorage(ctx, parsedFlags.fileStoragePath,
			parsedFlags.storeInterval, parsedFlags.restoreData)
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
	if parsedFlags.cryptKey != "" {
		chiMux.Use(sericeHttp.NewSha256Middleware(parsedFlags.cryptKey))
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

	logger.Infof(ctx, "serving metrics on port %s", parsedFlags.serverAddr)

	err = http.ListenAndServe(parsedFlags.serverAddr, chiMux)
	if err != nil {
		return err
	}

	return nil
}
