# Simple Makefile for a Go project

# Build the application
all: build

build:
	@echo "Building..."
	@go build -o main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

protogen:
	@echo "Generating proto files..."
	@buf generate

# Live Reload
watch:
	@if [ -x "$(GOPATH)/bin/air" ]; then \
	    "$(GOPATH)/bin/air"; \
		@echo "Watching...";\
	else \
	    read -p "air is not installed. Do you want to install it now? (y/n) " choice; \
	    if [ "$$choice" = "y" ]; then \
			go install github.com/cosmtrek/air@latest; \
	        "$(GOPATH)/bin/air"; \
				@echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

key:
	@echo "Generating keys..."
	@openssl genpkey -algorithm RSA -out ./keys/private_key.pem -pkeyopt rsa_keygen_bits:2048
	@openssl rsa -pubout -in ./keys/private_key.pem -out ./keys/public_key.pem
	@awk 'BEGIN {ORS="\\n"} {print} END {print ""}' ./keys/private_key.pem > ./keys/private_key_formatted.pem
