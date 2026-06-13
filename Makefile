BIN     := rollops-plugin-gatewayapi
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X 'main.version=$(VERSION)'

.PHONY: build test vet checksum dist clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/$(BIN)

test:
	go test -race ./...

vet:
	go vet ./...

# checksum prints the sha256 to paste into a rollout's featureFlags.sha256.
checksum: build
	shasum -a 256 bin/$(BIN)

# dist builds release archives for the supported platforms with checksums.
dist:
	rm -rf dist && mkdir -p dist
	@for pair in linux/amd64 linux/arm64 darwin/arm64; do \
		os=$${pair%/*}; arch=$${pair#*/}; \
		echo "building $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o dist/$(BIN)_$(VERSION)_$${os}_$${arch} ./cmd/$(BIN); \
	done
	cd dist && shasum -a 256 * > checksums.txt

clean:
	rm -rf bin dist
