build: *.go
	go build -o bin/mintspace

run: build
	./bin/mintspace

test:
	go test -v ./...