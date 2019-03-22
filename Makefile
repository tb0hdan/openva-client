.PHONY: build

BUILD = $(shell git rev-parse HEAD)
BDATE = $(shell date -u '+%Y-%m-%d_%I:%M:%S%p_UTC')
VERSION = $(shell cat ./VERSION)
GO_VERSION = $(shell go version|awk '{print $$3}')
PROJECT_URL = "https://openva.dev"


all: lint build

lint:
	@golangci-lint run

build:
	@go build -v -x -ldflags="-s -w -X main.Build=$(BUILD) -X main.BuildDate=$(BDATE) -X main.GoVersion=$(GO_VERSION) -X main.Version=$(VERSION) -X main.ProjectURL=$(PROJECT_URL)" .
	@strip ./openva-client

regen:
	@echo 'module openva-client' > ./go.mod
	@rm -f ./go.sum
	@go mod why

clean:
	@rm ./openva-client

rebuild: clean regen build
