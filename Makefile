static-check:
	go vet ./...
	golangci-lint run

test:
	go test ./... -vet=all -race -count=1 -cover -coverprofile=coverage.out

build:
	go build -o bin/uproxy cmd/uproxy.go

all: clean static-check test build

clean:
	rm bin/*
