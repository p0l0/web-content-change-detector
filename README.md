# Web Content Change Detector

This program scans a given website and notifies you if the content have changed to last check

## Build

`CGO_LDFLAGS="-static" CC=x86_64-linux-musl-gcc CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .`

### Build dependencies

`brew install FiloSottile/musl-cross/musl-cross`