BINARY_NAME=codeplugs

.PHONY: all build clean test run vet fmt frontend-build frontend-install

all: build

frontend-install:
	cd frontend && bun install

frontend-build: frontend-install
	cd frontend && bun run build

# Build the Go binary (including embedded frontend)
build: frontend-build
	go build -o $(BINARY_NAME) main.go

# Quick build without rebuilding frontend (if you know it hasn't changed)
fast-build:
	go build -o $(BINARY_NAME) main.go

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -f codeplugs.db*
	rm -f *.csv
	rm -f *.txt
	rm -rf frontend/dist

test:
	go test ./...

run: build
	./$(BINARY_NAME) --serve

vet:
	go vet ./...

fmt:
	go fmt ./...
