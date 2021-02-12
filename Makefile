GOSRC:=$(shell find . -name '*.go')

all: coverage.html version.go

# → https://blog.golang.org/cover
cover: coverage.html

coverage.html: coverage.out
	go tool cover -html=$< -o $@

coverage.out: $(GOSRC)
	go test -coverprofile=$@ ./... || true
#	go test -covermode=count -coverprofile=$@ || true

version.go: VERSION
	go generate
