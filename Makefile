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

.PHONY: bench
bench:
	k6 run bench.js

.PHONY: mocks
mocks:
	mockery --name=WalletUsecase --dir=./internal/port/handler --output=./mocks/usecase --outpkg=mocks
	mockery --name=WalletRepository --dir=./internal/usecase --output=./mocks/usecase --outpkg=mocks
	mockery --name=TxManager --dir=./internal/usecase --output=./mocks/usecase --outpkg=mocks

.PHONY: test
test:
	go test -race -timeout 30s ./...