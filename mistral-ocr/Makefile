.PHONY: build install clean test

# Binary name
BINARY_NAME=mistral-ocr

# Build the binary
build:
	go build -o $(BINARY_NAME) main.go

# Install the binary
install:
	go install

# Clean build files
clean:
	go clean
	rm -f $(BINARY_NAME)

# Run tests
test:
	go test ./...

# Run the program
run:
	go run main.go

# Build for multiple platforms
build-all:
	# Linux
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 main.go
	# macOS
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 main.go
	# Windows
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe main.go
