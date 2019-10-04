.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	go vet
	go test

.PHONY: test-integration
test-integration:
	go run -race ./examples/simple 2>/dev/null
	go run -race ./examples/dirwalk 2>/dev/null
	go run -race ./examples/abort 2>/dev/null

