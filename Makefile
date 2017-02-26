GOOPTS=GO15VENDOREXPERIMENT=1
GO=${GOOPTS} go
SOURCES=$(wildcard src/cw/*.go src/cw/**/*.go)
OSX=GOOS=darwin GOARCH=amd64 ${GO}
LINUX=GOOS=linux GOARCH=amd64 ${GO}
VERSION=$(shell git rev-parse --short HEAD)

.PHONY: release

./bin/cw: ${SOURCES}
	$(GO) build -o cw -ldflags "-X main.version=${VERSION}"

release: ${SOURCES}
	mkdir -p ./release/linux ./release/osx
	${OSX} build -o ./release/osx/bellhop
	${LINUX} build -o ./release/linux/bellhop
