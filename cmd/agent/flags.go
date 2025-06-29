package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type flags struct {
	serverAddr     url.URL
	reportInterval time.Duration
	pollInterval   time.Duration
}

func initFlags() (flags, error) {
	scheme := "http"
	serverAddr := url.URL{
		Scheme: scheme,
		Host:   "localhost:8080",
	}
	flag.Func("a", "metric server address", func(address string) error {
		if address == "" {
			return nil
		}

		serverAddr = url.URL{
			Scheme: scheme,
			Host:   address,
		}

		return nil
	})
	reportInterval := flag.Int64("r", 10, "report interval duration")
	pollInterval := flag.Int64("p", 2, "report poll duration")

	flag.Parse()

	if value, exist := os.LookupEnv("ADDRESS"); exist {
		if value == "" {
			return flags{}, fmt.Errorf("ADDRESS environment variable not set")
		}

		serverAddr = url.URL{
			Scheme: scheme,
			Host:   value,
		}
	}

	reportIntervalKey := "REPORT_INTERVAL"
	if value, exist := os.LookupEnv(reportIntervalKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", reportIntervalKey)
		}

		val, err := parseIntervalValue(value)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", reportIntervalKey, err)
		}
		reportInterval = &val
	}

	poolIntervalKey := "POLL_INTERVAL"
	if value, exist := os.LookupEnv(poolIntervalKey); exist {
		if value == "" {
			return flags{}, fmt.Errorf("%s environment variable not set", poolIntervalKey)
		}

		val, err := parseIntervalValue(value)
		if err != nil {
			return flags{}, fmt.Errorf("failed to parse %s: %w", poolIntervalKey, err)
		}
		pollInterval = &val
	}

	return flags{
		serverAddr:     serverAddr,
		reportInterval: time.Duration(*reportInterval) * time.Second,
		pollInterval:   time.Duration(*pollInterval) * time.Second,
	}, nil
}

func parseIntervalValue(value string) (int64, error) {
	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s: %w", value, err)
	}
	if intValue <= 0 {
		return 0, fmt.Errorf("invalid POLL_INTERVAL: %s", value)
	}

	return intValue, nil
}
