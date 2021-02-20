GOSRC:=$(shell find . -name '*.go')

.PHONY: texst

all: texst

# → https://blog.golang.org/cover
cover: coverage.html

coverage.html: coverage.out
	go tool cover -html=$< -o $@

coverage.out: $(GOSRC)
	go test -coverprofile=$@ ./... || true
#	go test -covermode=count -coverprofile=$@ || true

texst:
	go build ./cmd/texst
