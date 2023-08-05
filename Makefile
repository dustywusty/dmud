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

connect:
	while true; do nc localhost 3333 || sleep 10; done


run: build
	./$(BINARY_PATH)$(BINARY_NAME)
