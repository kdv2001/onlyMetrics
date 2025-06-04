
run-server:
	go run ./cmd/server -a "localhost:8080"

run-agent:
	go run ./cmd/agent -a "localhost:8080" -r 2 -p 2