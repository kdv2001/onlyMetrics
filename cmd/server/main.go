package main

import (
	"fmt"
	"log"
)

var buildVersion string
var buildDate string
var buildCommit string

const na = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", opIf(buildVersion != "", buildVersion, na))
	fmt.Printf("Build date: %s\n", opIf(buildDate != "", buildDate, na))
	fmt.Printf("Build commit: %s\n", opIf(buildCommit != "", buildCommit, na))

	if err := initService(); err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}
}

func opIf[T comparable](cond bool, a T, b T) T {
	if cond {
		return a
	}

	return b
}
