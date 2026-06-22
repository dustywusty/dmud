BINARY_NAME=dmud
BINARY_PATH=bin/
IMAGE_NAME=dmud
IMAGE_TAG?=latest

GO := $(shell which go)
AIR := $(shell go env GOPATH)/bin/air
DOCKER_COMPOSE := docker compose

default: build

## setup: Install tools and dependencies (first-time setup)
setup:
	@bash scripts/setup.sh

prep:
	mkdir -p $(BINARY_PATH)

## build: Compile binary to bin/dmud
build: prep
	$(GO) build -o $(BINARY_PATH)$(BINARY_NAME) -v ./cmd/dmud

## dev: Start dev server with hot reload (no persistence)
dev: prep
	@$(AIR) -c .air.toml

## dev-persist: Start dev server with file persistence (data saved to data/)
dev-persist: prep
	DMUD_PERSISTENCE=file DMUD_LOG_LEVEL=debug $(AIR) -c .air.toml

## dev-redis: Start Redis via Docker Compose, then dev server connected to it
dev-redis: prep
	$(DOCKER_COMPOSE) --profile redis up -d redis
	DMUD_PERSISTENCE=redis DMUD_LOG_LEVEL=debug $(AIR) -c .air.toml

## run: Build and run locally
run: build
	./$(BINARY_PATH)$(BINARY_NAME)

## watch: Hot-reload dev server (alias for dev)
watch: dev

## test: Run all tests
test:
	$(GO) test -v ./...

## test-race: Run tests with race detector
test-race:
	$(GO) test -race -v ./...

## smoke: Run the bot smoke test under the race detector (verbose transcript)
smoke:
	$(GO) test -race -v -run TestServer_BotsSmoke ./internal/game/

## vet: Run go vet
vet:
	$(GO) vet ./...

## race: Run binary with race detector
race:
	$(GO) run -race ./cmd/dmud

## clean: Remove build artifacts
clean:
	$(GO) clean
	rm -rf $(BINARY_PATH)

## connect: Connect to TCP server via netcat
connect:
	while true; do nc localhost 3333 || sleep 10; done

# --- Docker ---

## docker-build: Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## docker-run: Build and run in Docker (port 8080)
docker-run: docker-build
	docker run --rm -it -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

## docker-stop: Stop running dmud containers
docker-stop:
	docker stop $$(docker ps -q --filter ancestor=$(IMAGE_NAME):$(IMAGE_TAG)) 2>/dev/null || true

## docker-clean: Remove dmud Docker image
docker-clean:
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true

# --- Docker Compose ---

## dc-up: Start services via Docker Compose
dc-up:
	$(DOCKER_COMPOSE) up --build

## dc-down: Stop Docker Compose services
dc-down:
	$(DOCKER_COMPOSE) down

## dc-logs: Tail Docker Compose logs
dc-logs:
	$(DOCKER_COMPOSE) logs -f

## help: Show available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## //' | column -t -s ':'

.PHONY: default setup prep build dev dev-persist dev-redis run watch test test-race vet race clean connect \
	docker-build docker-run docker-stop docker-clean \
	dc-up dc-down dc-logs help
