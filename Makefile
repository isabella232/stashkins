# This build uses godep, which manages which commits of our dependencies we build against.
# https://github.com/tools/godep

all:
	type godep > /dev/null 2>&1 || go get -v github.com/tools/godep
	go clean
	godep go test
	godep go build
	godep go install
