package main

import (
	"flag"
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

func initFlags() flags {
	scheme := "http"
	serverAddr := url.URL{
		Scheme: scheme,
		Host:   "localhost:8080",
	}
	flag.Func("a", "metric server address", func(s string) error {
		if s == "" {
			return nil
		}

		serverAddr = url.URL{
			Scheme: scheme,
			Host:   s,
		}

		return nil
	})
	reportInterval := flag.Int64("r", 10, "report interval duration")
	pollInterval := flag.Int64("p", 2, "report poll duration")

	flag.Parse()

	if value := os.Getenv("ADDRESS"); value != "" {
		serverAddr = url.URL{
			Scheme: scheme,
			Host:   value,
		}
	}

	if value := os.Getenv("REPORT_INTERVAL"); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			reportInterval = &intValue
		}
	}

	if value := os.Getenv("POLL_INTERVAL"); value != "" {
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			pollInterval = &intValue
		}
	}

	return flags{
		serverAddr:     serverAddr,
		reportInterval: time.Duration(*reportInterval) * time.Second,
		pollInterval:   time.Duration(*pollInterval) * time.Second,
	}
}
