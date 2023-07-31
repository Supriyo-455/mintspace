RELEASE=bin/release/mintspace
DEBUG=bin/debug/mintspace

run: $(RELEASE)
	./$^

release: $(RELEASE)

debug: $(DEBUG)

$(RELEASE): **.go
	go build -o $@

$(DEBUG): **.go
	go build -gcflags="all=-N -l" -o $@

test:
	go test -v ./...

clean:
	rm -rf bin/