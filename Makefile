
all: build test

.PHONY: build
build:
	go generate ./...
	go build -o osshit ./main.go

.PHONY: test
test:
	go test -cover ./...

.PHONY: run
run: build
	./osshit serve  --config honeycfg
