package main

import (
	"flag"
	"os"
)

type flags struct {
	serverAddr string
}

func initFlags() flags {
	serverAddr := flag.String("a", ":8080", "The address to bind the server to")

	flag.Parse()

	if value := os.Getenv("ADDRESS"); value != "" {
		serverAddr = &value
	}

	return flags{
		serverAddr: *serverAddr,
	}
}
