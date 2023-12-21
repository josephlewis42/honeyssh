
all: build test

.PHONY: build
build:
	go generate ./...
	go build -o honeyssh ./main.go

.PHONY: test
test:
	go test -cover ./...

.PHONY: run
run: build
	./honeyssh serve  --config honeycfg

.PHONY: play
play: build
	./honeyssh playground

.PHONY: setup
setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install github.com/mitchellh/protoc-gen-go-json@latest