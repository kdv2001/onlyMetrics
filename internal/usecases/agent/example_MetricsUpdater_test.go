package agent

import (
	"context"
	"fmt"
	"time"
)

func ExampleMetricsUpdater_GetMetrics() {
	ctx := context.Background()
	updatePeriod := 5 * time.Second
	// создаем объект сборщика метрик
	updater := NewMetricsUpdater(ctx, updatePeriod)

	time.Sleep(updatePeriod)
	// получаем собранные метрики
	metrics, err := updater.GetMetrics(ctx)
	if err != nil {
		fmt.Println("Something went wrong: ", err)
		return
	}

	fmt.Println("result metrics: ", metrics)
}
