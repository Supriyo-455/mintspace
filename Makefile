build: *.go
	go build -o bin/mintspace

debug: *go
	go build -gcflags="all=-N -l" -o debug/mintspace

run: build
	./bin/mintspace

test:
	go test -v ./...