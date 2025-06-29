package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	sericeHttp "github.com/kdv2001/onlyMetrics/internal/handlers/http"
	"github.com/kdv2001/onlyMetrics/internal/pkg/logger"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/memory"
	"github.com/kdv2001/onlyMetrics/internal/usecases/metrics"
)

func initService() error {
	ctx := context.Background()
	parsedFlags, err := initFlags()
	if err != nil {
		return fmt.Errorf("failed to init flags: %w", err)
	}

	metricsStorage := memory.NewStorage(ctx, parsedFlags.fileStoragePath,
		parsedFlags.storeInterval, parsedFlags.restoreData)
	defer metricsStorage.Close(ctx)

	metricsUC := metrics.NewUseCases(metricsStorage)
	httpHandlers := sericeHttp.NewHandlers(metricsUC)

	chiMux := chi.NewMux()
	log, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to init looger: %w", err)
	}

	sugarLogger := log.Sugar()
	chiMux.Use(
		sericeHttp.CompressMiddleware(sericeHttp.GetDefaultAcceptedEncodingData()),
		sericeHttp.DecompressMiddleware(),
		sericeHttp.AddLoggerToContextMiddleware(sugarLogger),
		sericeHttp.ResponseMiddleware(),
		sericeHttp.RequestMiddleware())

	chiMux.Get("/", httpHandlers.GetAllMetric)
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

	logger.Infof(ctx, "serving metrics on port %s", parsedFlags.serverAddr)

	err = http.ListenAndServe(parsedFlags.serverAddr, chiMux)
	if err != nil {
		return err
	}

	return nil
}
