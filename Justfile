fixtures:
	bash -c 'cd test/testdata && ./build-fixtures.go'

protos:
  buf lint && buf generate

proto-lint:
	buf lint

proto-generate:
  buf generate

test:
  go test -race -v ./...

test-failfast:
  go test -race -v ./...

bench:
  go test -race -bench='.+' -v ./...

build:
  go build ./cmd/gonix

lint:
  golangci-lint run
