.PHONY: build run clean cov fmt help vet test

## build: build for all platforms
build: 
	go build -o bin/ziond cmd/ziond/main.go

## build-cli: build CLI for all platforms
build-cli: 
	go build -o bin/zion cmd/zion/main.go


## clean: cleans the binary
clean:
	@echo "Cleaning..."
	@go clean

## cov: generates coverage report
cov:
	@echo "Coverage..."
	go test -cover ./...

## fmt: Go Format
fmt:
	@echo "Gofmt..."
	@if [ -n "$(gofmt -l .)" ]; then echo "Go code is not formatted"; exit 1; fi


## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## run: Run locally with default configuration in regtest
run: clean
	export ZION_NETWORK=regtest; \
	go run ./cmd/ziond

## vet: code analysis
vet:
	@echo "Vet..."
	@go vet ./...

## create-cert: creates localhost ssl certficate and key
create-cert:
	chmod u+x ./scripts/sslcert
	bash ./scripts/sslcert