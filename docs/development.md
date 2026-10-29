# Development

## Requirements

- Linux; Ubuntu is the initial supported distribution.
- Go 1.25 or newer. No third-party Go modules are required.
- Make and a C compiler for race-enabled tests. Runtime binaries do not need Go
  or a C compiler installed on the destination machine.

```sh
make check
make build
./bin/eversor dry-run
```

`make check` verifies formatting, runs vet and race-enabled tests, and builds a
static Linux binary. Tests use disposable synthetic homes and fake command
runners. Never use real user directories as destructive test fixtures.

`make dist` builds Linux amd64 and arm64 archives plus `dist/SHA256SUMS`.
Build outputs are ignored by Git. Set `VERSION` explicitly for a packaged build;
otherwise the version identifies the current Git revision and a dirty tree.

## Components

- `internal/core/environment_linux.go`: account/XDG discovery, installed-software
  inventory, process inspection and mount boundaries.
- `internal/core/catalog.go`: reviewed app-specific cache definitions.
- `internal/core/scan.go`: storage tree, ownership/identity validation and
  traversal-resistant cache roots.
- `internal/core/plan.go`: read-only candidate construction and eligibility.
- `internal/core/clean.go`: revalidation, scoped user cleanup and local Docker
  endpoint enforcement. Native uv pruning receives an inherited directory handle.
- `internal/core/system.go`: fixed APT/Snap/journal operations and kernel guards.
- `internal/app`: public CLI, confirmation, locking and privileged helper dispatch.
- `internal/report`: stdout reports; no automatic persistent logs.

The hidden system endpoint is not a general root command runner. It accepts only
the built-in operation IDs and rechecks package state. A development binary or
one inside a user-writable directory cannot be elevated by `eversor clean`.
Install a root-owned binary first or use `clean --user-only` during development.

## Verification scope

The automated suite exercises cache discovery, clean/preview/cancellation,
adjacent data preservation, hard links, symlinks, replaced directories, mount
boundaries, active apps/generated executables, concurrent cleanup, malformed
package data, remote Docker rejection and running/newest kernel protection.
CI tests Go 1.25 and 1.27 on Ubuntu and cross-builds both CPU architectures.

This does not certify every Ubuntu version or third-party app layout. Unknown
layouts are left alone. Add upstream evidence and preservation tests when
supporting another layout; do not broaden cleanup to a whole data directory.
