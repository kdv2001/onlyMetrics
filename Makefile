
run-server:
	go run ./cmd/server -a "localhost:8080"

run-agent:
	go run ./cmd/agent -a "localhost:8080" -r 2 -p 2

run-godoc:
	godoc -http=:8080 -play

build-server:
	go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildCommit=$(git rev-parse HEAD) -X 'main.buildDate=$(date)'" -o ./cmd/server ./cmd/server

build-agent:
	go build -ldflags "-X main.buildVersion=v1.0.0 -X main.buildCommit=$(git rev-parse HEAD) -X 'main.buildDate=$(date)'" -o ./cmd/agent ./cmd/agent

build: build-agent build-server