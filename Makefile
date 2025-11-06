
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

generate-certs:
	go run ./cmd/cert

# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
generate:
	protoc --go_out=. --go_opt=paths=import \
      --go-grpc_out=. --go-grpc_opt=paths=import \
      api/onlyMetrics.proto --go_opt=default_api_level=API_OPAQUE
