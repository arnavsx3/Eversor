# Development

## Requirements

- Linux; Ubuntu is the initial supported distribution.
- Go 1.25 or newer. No third-party Go modules are required.
- Make and a C compiler for race-enabled tests. Runtime binaries do not need Go
  or a C compiler installed on the destination machine.
- GNU tar, coreutils and `dpkg-deb` for Linux release packaging.

```sh
make check
make build
./bin/eversor dry-run
```

`make check` verifies formatting, runs vet and race-enabled tests, and builds a
static Linux binary. Tests use disposable synthetic homes and fake command
runners. Never use real user directories as destructive test fixtures.

`make dist-check` builds Linux amd64/arm64 archives and `.deb` packages, verifies
checksums, package architecture/version, extracted binaries, and a native binary
smoke test. Build outputs are ignored by Git. `VERSION` supplies the numbered
version; the build also embeds its Git revision and dirty state. Release metadata
uses `SOURCE_DATE_EPOCH` from the source commit.

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

## Publishing a release

1. Update `VERSION` and `docs/release-notes.md` for the intended release.
2. Run `make check dist-check`, review, commit and push. Wait for CI to pass.
3. Tag the tested commit with `v` followed by `VERSION`, then push that tag.

The release workflow requires the tag to match `VERSION`, repeats the checks and
publishes the packages and checksums to GitHub Releases. Published versions and
tags should not be overwritten; make a new version for corrections. Install the
verified package locally and use dry-run/cancellation for live validation.

This does not certify every Ubuntu version or third-party app layout. Unknown
layouts are left alone. Add upstream evidence and preservation tests when
supporting another layout; do not broaden cleanup to a whole data directory.
