VERSION=0.0.3

# Darwin or Linux
ARCH=$(shell bash -c "uname | tr '[:upper:]' '[:lower:]'")
DESTDIR="bin"

ifeq ($(ARCH), darwin)
release: release-darwin-linux
endif

ifeq ($(ARCH), linux)
release: release-linux-darwin
endif

release: deps
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
	@go get .

test: deps
	go test -v

.PHONY: release release-darwin-linux release-linux-darwin deps test
