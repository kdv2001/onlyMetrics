package main

import (
	"fmt"
	"log"
	"net/http"

	sericeHttp "github.com/kdv2001/onlyMetrics/internal/handlers/http"
	"github.com/kdv2001/onlyMetrics/internal/storage/metrics/memory"
	"github.com/kdv2001/onlyMetrics/internal/usecases/metrics"
)

func initService() error {
	metricsStorage := memory.NewStorage()
	metricsUC := metrics.NewUseCases(metricsStorage)

	httpHandlers := sericeHttp.NewHandlers(metricsUC)

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/update/{%s}/{%s}/{%s}",
		sericeHttp.MetricTypePathKey,
		sericeHttp.MetricNamePathKey,
		sericeHttp.ValuePathKey,
	),
		httpHandlers.CollectMetric)

	log.Print("serving metrics on port 8080")
	return http.ListenAndServe(":8080", mux)
}
