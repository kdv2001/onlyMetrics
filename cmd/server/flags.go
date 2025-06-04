package main

import (
	"flag"
)

type flags struct {
	serverAddr string
}

func initFlags() flags {
	serverAddr := flag.String("a", ":8080", "The address to bind the server to")

	flag.Parse()

	return flags{
		serverAddr: *serverAddr,
	}
}
