package main

import "log"

func main() {
	if err := initService(); err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}
}
