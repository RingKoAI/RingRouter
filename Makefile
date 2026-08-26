.PHONY: build run test clean dev

APP_NAME := ringrouter
GO := go
GOFLAGS := -trimpath

build:
	$(GO) build $(GOFLAGS) -o $(APP_NAME) .

run:
	$(GO) run $(GOFLAGS) .

test:
	$(GO) test -v -race ./...

clean:
	rm -f $(APP_NAME)
	rm -rf data/

dev:
	$(GO) run $(GOFLAGS) .

lint:
	$(GO) vet ./...