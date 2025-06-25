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
	chiMux.Use(sericeHttp.AddLoggerToContextMiddleware(sugarLogger),
		sericeHttp.ResponseMiddleware(), sericeHttp.RequestMiddleware())

	chiMux.HandleFunc(fmt.Sprintf("/update/{%s}/{%s}/{%s}",
		sericeHttp.MetricTypePathKey,
		sericeHttp.MetricNamePathKey,
		sericeHttp.ValuePathKey,
	), httpHandlers.CollectMetric)

	chiMux.HandleFunc(fmt.Sprintf("/value/{%s}/{%s}",
		sericeHttp.MetricTypePathKey,
		sericeHttp.MetricNamePathKey,
	), httpHandlers.GetMetric)

	sugarLogger.Infof("serving metrics on port %s", parsedFlags.serverAddr)

	return http.ListenAndServe(parsedFlags.serverAddr, chiMux)
}
