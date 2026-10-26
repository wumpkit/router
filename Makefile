.PHONY: all fmt lint lint-fix test bench

all: fmt lint test

# format go source code
fmt:
	go fmt ./...

# run linter checks
lint:
	golangci-lint run ./...

# run linter and apply automated fixes
lint-fix:
	golangci-lint run --fix ./...

# run unit tests with race detection
test:
	go test -v -race ./...

# run memory and cpu benchmarks
bench:
	go test -bench=. -benchmem ./...
