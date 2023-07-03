GO=/usr/local/go/bin/go
BINARY_NAME=dmud
BINARY_PATH=bin/

default: build

prep:
	mkdir -p $(BINARY_PATH)

build: prep
	$(GO) build -o $(BINARY_PATH)$(BINARY_NAME) -v ./cmd/dmud

clean: 
	$(GO) clean
	rm -rf $(BINARY_PATH)

run: build
	./$(BINARY_PATH)$(BINARY_NAME)
