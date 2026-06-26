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

## docs: Regenerate docs/ from the content registries (spells, races, classes, …)
docs:
	$(GO) run ./cmd/gendocs

## dev: Start dev server with hot reload + file persistence (saves to data/)
dev: prep
	@DMUD_PERSISTENCE=file $(AIR) -c .air.toml

## dev-persist: Alias for `make dev` (file persistence); kept for muscle memory
dev-persist: dev

## dev-nopersist: Start dev server with hot reload and NO persistence
dev-nopersist: prep
	@DMUD_PERSISTENCE=none $(AIR) -c .air.toml

## dev-redis: Start Redis via Docker Compose, then dev server connected to it
dev-redis: prep
	$(DOCKER_COMPOSE) --profile redis up -d redis
	DMUD_PERSISTENCE=redis DMUD_LOG_LEVEL=debug $(AIR) -c .air.toml

## run: Build and run locally
run: build
	./$(BINARY_PATH)$(BINARY_NAME)

## client: Build and run the terminal client (WebSocket localhost:8080; pass ARGS="-tcp")
client: prep
	$(GO) build -o $(BINARY_PATH)dmud-client ./cmd/client
	./$(BINARY_PATH)dmud-client $(ARGS)

## client-build: Compile the terminal client to bin/dmud-client
client-build: prep
	$(GO) build -o $(BINARY_PATH)dmud-client -v ./cmd/client

## web: Serve the static web client at http://localhost:8090 (server must run on :8080)
web:
	@echo "Open http://localhost:8090/?ws=ws://localhost:8080/ws"
	@python3 -m http.server 8090 -d web

## webtty: Run the real TUI client in a browser via a PTY→xterm.js bridge (http://localhost:8091; server on :8080)
webtty: prep client-build
	$(GO) build -o $(BINARY_PATH)webtty ./cmd/webtty
	@echo "Open http://localhost:8091  (local/dev only — see cmd/webtty/main.go)"
	./$(BINARY_PATH)webtty -listen :8091 -client $(BINARY_PATH)dmud-client -mud localhost:8080

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
	$(GO) test -race -v -count=1 -run TestServer_BotsSmoke ./internal/game/

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

.PHONY: default setup prep build docs client client-build web webtty dev dev-persist dev-nopersist dev-redis run watch test test-race smoke vet race clean connect \
	docker-build docker-run docker-stop docker-clean \
	dc-up dc-down dc-logs help
