package main

import (
	"fmt"
	"log"

	"github.com/kdv2001/onlyMetrics/pkg/operators"
)

var buildVersion string
var buildDate string
var buildCommit string

const na = "N/A"

func main() {
	fmt.Printf("Build version: %s\n", operators.OpIf(buildVersion != "", buildVersion, na))
	fmt.Printf("Build date: %s\n", operators.OpIf(buildDate != "", buildDate, na))
	fmt.Printf("Build commit: %s\n", operators.OpIf(buildCommit != "", buildCommit, na))

	if err := initService(); err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}
}
