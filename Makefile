VERSION=0.0.5

# Darwin or Linux
ARCH=$(shell bash -c "uname | tr '[:upper:]' '[:lower:]'")
DESTDIR="bin"

.PHONY: all release-darwin-linux release-linux-darwin deps test racetest

ifeq ($(ARCH), darwin)
all: release-darwin-linux
endif

ifeq ($(ARCH), linux)
all: release-linux-darwin
endif

all: deps
	$(info Compiling $(ARCH) binary...)
	@go build -o $(DESTDIR)/web-content-change-detector-$(VERSION)-$(ARCH) .

release-darwin-linux: deps
	$(info Compiling Linux binary...)
	@CGO_LDFLAGS="-static" CC=x86_64-linux-musl-gcc CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o $(DESTDIR)/web-content-change-detector-$(VERSION)-linux .

release-linux-darwin: deps
	$(info Compiling Darwin binary...)
	@echo "Crosscompilation to Darwin currently not supported!"
#	@CGO_LDFLAGS="-static" GOARCH=amd64 CGO_ENABLED=1 GOOS=darwin go build -a -installsuffix cgo -o $(DESTDIR)/web-content-change-detector-$(VERSION)-darwin .

deps:
	$(info Downloading dependencies...)
	@go get -v -t ./...

test: deps
	$(info Running tests...)
	@cd difflib; go test
	@go test -v -coverprofile coverage.txt -covermode=atomic
	@go tool cover -func coverage.txt

coverage: test
	@go tool cover -html=coverage.txt

racetest: deps
	$(info Running tests...)
	@go test -race -v -coverprofile coverage.txt -covermode=atomic
	@go tool cover -func coverage.txt
