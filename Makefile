
all: build test

.PHONY: build
build:
	go build -o osshit ./main.go

.PHONY: test
test:
	go test ./...

.PHONY: run
run: build
	./osshit serve  --host-key ~/.ssh/id_rsa --root-fs fs.tar
