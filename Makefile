GO ?= go
VERSION ?= $(shell cat VERSION)
REVISION ?= $(shell git describe --always --dirty --exclude '*' 2>/dev/null || printf unknown)
SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct 2>/dev/null || date +%s)
export SOURCE_DATE_EPOCH
LDFLAGS = -s -w -X github.com/arnavsx3/Eversor/internal/app.Version=$(VERSION) -X github.com/arnavsx3/Eversor/internal/app.Revision=$(REVISION)

.PHONY: build test vet fmt-check check install uninstall update dist dist-check

build:
	@mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o bin/eversor ./cmd/eversor

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt-check:
	@files=$$(gofmt -l cmd internal) && test -z "$$files" || (echo 'Run gofmt before committing.'; exit 1)

check: fmt-check vet test build

install: build
	sudo install -o root -g root -m 0755 bin/eversor /usr/local/bin/eversor

uninstall:
	sudo rm -- /usr/local/bin/eversor

update:
	@test -z "$$(git status --porcelain)" || (echo 'Commit or set aside local changes before updating.'; exit 1)
	git pull --ff-only
	$(MAKE) check
	$(MAKE) install

dist:
	@mkdir -p dist
	@for arch in amd64 arm64; do \
		name=eversor_linux_$$arch; \
		mkdir -p dist/$$name; \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o dist/$$name/eversor ./cmd/eversor || exit 1; \
		cp LICENSE dist/$$name/LICENSE; \
		tar --sort=name --owner=0 --group=0 --numeric-owner --mtime=@$(SOURCE_DATE_EPOCH) -C dist/$$name -czf dist/$$name.tar.gz eversor LICENSE || exit 1; \
	done
	@bash scripts/package-deb.sh '$(VERSION)'
	@cd dist && sha256sum eversor_linux_amd64.tar.gz eversor_linux_arm64.tar.gz eversor_$(VERSION)_amd64.deb eversor_$(VERSION)_arm64.deb > SHA256SUMS

dist-check: dist
	bash scripts/check-packages.sh '$(VERSION)'
