package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	sericeHttp "github.com/kdv2001/onlyMetrics/internal/handlers/http"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/memory"
	"github.com/kdv2001/onlyMetrics/internal/usecases/metrics"
)

func initService() error {
	parsedFlags := initFlags()

	metricsStorage := memory.NewStorage()
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

	sugarLogger.Infof("serving metrics on port %s", parsedFlags.serverAddr)

	return http.ListenAndServe(parsedFlags.serverAddr, chiMux)
}
