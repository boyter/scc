.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	go vet
	go test

.PHONY: test-integration
test-integration:
	go run ./examples/simple 2>/dev/null
	go run ./examples/dirwalk 2>/dev/null

