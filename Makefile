all: build

build:
	@go build -v -x -ldflags="-s -w" .
	@strip ./openva-client

regen:
	@echo 'module openva-client' > ./go.mod
	@rm -f ./go.sum
	@go mod why

clean:
	@rm ./openva-client

rebuild: clean regen build
