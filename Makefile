VERSION=0.0.2

release: deps
	CGO_LDFLAGS="-static" CC=x86_64-linux-musl-gcc CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o web-content-change-detector .

deps:
	go get .

test: deps
	go test -v

.PHONY: release deps test

