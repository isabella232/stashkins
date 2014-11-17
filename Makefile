NAME := stashkins
ARCH := amd64
VERSION := 1.1-dev
DATE := $(shell date)
COMMIT_ID := $(shell git rev-parse --short HEAD)
SDK_INFO := $(shell go version)
LD_FLAGS := -X main.version $(VERSION) -X main.commit $(COMMIT_ID) -X main.buildTime '$(DATE)' -X main.sdkInfo '$(SDK_INFO)'

all: clean binaries package

test:
	godep go test

binaries: tools deps test 
	GOOS=darwin GOARCH=$(ARCH) godep go build -ldflags "$(LD_FLAGS)" -o $(NAME)-darwin-$(ARCH)
	GOOS=linux GOARCH=$(ARCH) godep go build -ldflags "$(LD_FLAGS)" -o $(NAME)-linux-$(ARCH)

deps:
	go get -v -u github.com/xoom/stash
	go get -v -u github.com/xoom/jenkins

tools:
	type godep > /dev/null 2>&1 || go get -v github.com/tools/godep

clean: 
	go clean
	rm -f *.deb

package:
	which fpm && fpm -s dir -t deb -v $(VERSION) -n stashkins -a amd64 -m"Mark Petrovic <mark.petrovic@xoom.com>" --prefix /usr/local/bin --description "https://github.com/xoom/stashkins" stashkins-linux-amd64
