GO ?= go
BINARY ?= schh
CMD ?= ./cmd/schh
BIN_DIR ?= bin

.PHONY: build install run test fmt tidy clean

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(BINARY) $(CMD)

install:
	$(GO) install $(CMD)

run: build
	$(BIN_DIR)/$(BINARY)

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)
