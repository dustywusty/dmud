BINARY_NAME=dmud
BINARY_PATH=bin/
IMAGE_NAME=dmud
IMAGE_TAG?=latest

GO := $(shell which go)

default: build

prep:
	mkdir -p $(BINARY_PATH)

build: prep
	$(GO) build -o $(BINARY_PATH)$(BINARY_NAME) -v ./cmd/dmud

clean:
	$(GO) clean
	rm -rf $(BINARY_PATH)

test:
	$(GO) test -v ./...

test-race:
	$(GO) test -race -v ./...

vet:
	$(GO) vet ./...

connect:
	while true; do nc localhost 3333 || sleep 10; done

run: build
	./$(BINARY_PATH)$(BINARY_NAME)

race:
	$(GO) run -race ./cmd/dmud

# go install github.com/air-verse/air@latest
AIR := $(shell go env GOPATH)/bin/air

watch:
	@$(AIR) -c .air.toml

# --- Docker ---

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-run: docker-build
	docker run --rm -it -p 8080:8080 $(IMAGE_NAME):$(IMAGE_TAG)

docker-stop:
	docker stop $$(docker ps -q --filter ancestor=$(IMAGE_NAME):$(IMAGE_TAG)) 2>/dev/null || true

docker-clean:
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true

.PHONY: default prep build clean test test-race vet connect run race watch \
	docker-build docker-run docker-stop docker-clean
