.PHONY: build build-go build-front run run-go run-front \
        run-front-prod dev test test-go lint clean

GO      := go
GOFLAGS := -ldflags="-s -w"
BIN     := server
FRONT   := front
PORT    ?= 8888

build: build-go build-front

build-go:
	$(GO) build $(GOFLAGS) -o $(BIN) ./cmd/server

build-front:
	cd $(FRONT) && yarn build:prod

run: run-go

run-go: build-go
	./$(BIN)

run-front:
	cd $(FRONT) && yarn dev

run-front-prod: build-front
	cd $(FRONT) && yarn start

dev:
	@echo "Start Go backend and frontend dev server in parallel"
	@echo "  Go backend:  http://localhost:$(PORT)"
	@echo "  Frontend:    http://localhost:3000"
	$(GO) run ./cmd/server &
	cd $(FRONT) && yarn dev

test: test-go

test-go:
	$(GO) test ./...

lint:
	cd $(FRONT) && yarn lint

clean:
	rm -f $(BIN)
	rm -rf $(FRONT)/.next
	rm -rf $(FRONT)/dist
	rm -rf build/
