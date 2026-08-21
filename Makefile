.PHONY: build install test clean run

BINARY_NAME=agpass
BUILD_DIR=bin

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) .

install:
	go install .

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)