GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || printf dev)
LDFLAGS = -X github.com/arnavsx3/Eversor/internal/app.Version=$(VERSION)

.PHONY: build test vet fmt-check check install uninstall dist

build:
	@mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/eversor ./cmd/eversor

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (echo 'Run gofmt before committing.'; exit 1)

check: fmt-check vet test build

install: build
	sudo install -o root -g root -m 0755 bin/eversor /usr/local/bin/eversor

uninstall:
	sudo rm -f /usr/local/bin/eversor

dist:
	@mkdir -p dist
	@for arch in amd64 arm64; do \
		name=eversor_linux_$$arch; \
		mkdir -p dist/$$name; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o dist/$$name/eversor ./cmd/eversor || exit 1; \
		cp LICENSE dist/$$name/LICENSE; \
		tar -C dist/$$name -czf dist/$$name.tar.gz eversor LICENSE || exit 1; \
	done
	@cd dist && sha256sum eversor_linux_*.tar.gz > SHA256SUMS
