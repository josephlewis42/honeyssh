
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
