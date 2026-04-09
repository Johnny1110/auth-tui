BINARY_NAME=authc

.PHONY: all build run clean help

all: build

build:
	@echo "compiling $(BINARY_NAME)..."
	go build -o built/$(BINARY_NAME) main.go
	@echo "compile success"

run: build
	./$(BINARY_NAME)

clean:
	@echo "cleaning..."
	rm -rf built
	@echo "clean done"

install: build
	@echo "installing -> /usr/local/bin..."
	cp $(BINARY_NAME) /usr/local/bin/
	@echo "install success"