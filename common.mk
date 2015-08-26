MAINTAINER := 'INF ENG <inf-eng@xoom.com>'
ARCH := amd64
DATE := $(shell date)
COMMIT_ID := $(shell git rev-parse --short HEAD)
SDK_INFO := $(shell go version)
LD_FLAGS := -X main.buildInfo 'Version: $(VERSION), commitID: $(COMMIT_ID), build date: $(DATE), SDK: $(SDK_INFO)'

all: clean binaries 

test:
	godep go test ./...

binaries: tools test 
	GOOS=darwin GOARCH=$(ARCH) godep go build -ldflags "$(LD_FLAGS)" -o $(NAME)-darwin-$(ARCH)
	GOOS=linux GOARCH=$(ARCH) godep go build -ldflags "$(LD_FLAGS)" -o $(NAME)-linux-$(ARCH)
	GOOS=windows GOARCH=$(ARCH) godep go build -ldflags "$(LD_FLAGS)" -o $(NAME)-windows-$(ARCH).exe

tools:
	type godep > /dev/null 2>&1 || go get -v github.com/tools/godep

package: all
	mkdir -p packaging
	cp $(NAME)-linux-$(ARCH) packaging/$(NAME)
	fpm -s dir -t deb -v $(VERSION) -n $(NAME) -a amd64 --description $(DESCRIPTION) -m $(MAINTAINER) --vendor $(MAINTAINER) --url $(DOCUMENTATION) --iteration 1 --prefix /usr/local/bin -C packaging .
	fpm -s dir -t rpm --rpm-os linux -v $(VERSION) -n $(NAME) -a amd64 --description $(DESCRIPTION) -m $(MAINTAINER) --vendor $(MAINTAINER) --url $(DOCUMENTATION) --iteration 1 --prefix /usr/local/bin -C packaging .

clean: 
	go clean
	rm -rf *.deb *.rpm packaging
	rm -f $(NAME)-darwin-$(ARCH) $(NAME)-linux-$(ARCH) $(NAME)-windows-$(ARCH).exe
