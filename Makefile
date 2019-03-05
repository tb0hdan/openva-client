all: build

build:
	@go build -v -x -ldflags="-s -w" .
	@strip ./openva-client

