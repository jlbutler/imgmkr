.PHONY: build clean test install lint

# Build the binary
build:
	go build -o imgmkr .

# Clean build artifacts
clean:
	rm -f imgmkr

# Run tests
test:
	go test ./...

# Install the binary
install:
	go install .

# Run linter
lint:
	go vet ./...
	go fmt ./...

# Run all checks
check: lint test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o dist/imgmkr-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/imgmkr-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/imgmkr-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/imgmkr-windows-amd64.exe .