.PHONY: build
build:
	go build -v ./cmd/wallet

.PHONY: start
start:
	./wallet.exe

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: gofmt
gofmt:
	gofmt -w -s .

.PHONY: test
test:
	go test -v -race -timeout 30s ./...