# This build uses godep, which manages which commits of our dependencies we build against.
# https://github.com/tools/godep

LD_FLAGS := -X main.commit $(shell git rev-parse --short HEAD)

all: tools deps
	go clean
	godep go test
	godep go build -ldflags "$(LD_FLAGS)"
	godep go install

deps:
	go get -v -u github.com/xoom/stash
	go get -v -u github.com/xoom/jenkins

tools:
	type godep > /dev/null 2>&1 || go get -v github.com/tools/godep
