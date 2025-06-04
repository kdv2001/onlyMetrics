package main

import (
	"flag"
	"net/url"
	"time"
)

type flags struct {
	serverAddr     url.URL
	reportInterval time.Duration
	pollInterval   time.Duration
}

func initFlags() flags {
	serverAddr := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
	}
	flag.Func("a", "metric server address", func(s string) error {
		if s == "" {
			return nil
		}

		serverAddr = url.URL{
			Scheme: "http",
			Host:   s,
		}

		return nil
	})
	reportInterval := flag.Int64("r", 10, "report interval duration")
	pollInterval := flag.Int64("p", 2, "report poll duration")

	flag.Parse()

	return flags{
		serverAddr:     serverAddr,
		reportInterval: time.Duration(*reportInterval) * time.Second,
		pollInterval:   time.Duration(*pollInterval) * time.Second,
	}
}
