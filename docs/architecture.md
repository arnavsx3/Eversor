# Architecture

Eversor is a local, offline-first Ubuntu CLI written in Go. Its public operations
are `eversor dry-run` and `eversor clean`. A single binary discovers the current
account and installed software; no username, application list, or project path
from the developer's laptop is required on another machine.

## Pipeline

1. **Inventory:** inspect executable paths, installed Debian packages, Snap and
   Flatpak application IDs, desktop app state, and running user processes.
2. **Inspect (optional `--tree`):** walk the user's home without following symlinks or crossing mount
   points. Build an allocated-size tree, deduplicate hard links, and report
   inaccessible paths without treating them as cleanup candidates.
3. **Detect:** application providers map inventory evidence to documented cache
   paths or native cleanup operations. Unknown folders remain inspection-only.
4. **Plan:** attach an explanation, size estimate, privilege scope, and current
   eligibility to each candidate. A plan is generated afresh for every invocation.
5. **Execute:** `clean` displays the plan and requires confirmation, unless `--yes`
   was supplied. Recheck activity, ownership, symlinks, mount boundaries, and file
   identity before each action. Continue reporting independent failures.
6. **Report:** print each result and measured filesystem space recovery. JSON is
   available for scripting. There are no background jobs, telemetry, or automatic
   persistent logs; reports remain local unless the user saves or shares them.

## Boundaries

- A directory being large, old, named `cache`, or inside `.cache` is insufficient
  evidence for deletion. Providers must have an explicit supported rule.
- Preserve source code, credentials, browser cookies/history/profiles, databases,
  downloaded models, installed toolchains, Docker images/containers/volumes,
  application state, and unknown directories.
- Skip relevant open applications and active package/build tools. Unknown process
  state or a failed inspection must not be interpreted as permission to clean.
- File cleanup runs as the ordinary user through Go's traversal-resistant
  `os.Root` API, using the inspected directory identity as an additional guard.
  Keep the cache directory itself; clear only its approved contents.
- Native cleanup commands are fixed by provider code and receive argument arrays,
  never shell strings. Missing tools and unavailable daemons become skipped items.
- System cleanup uses a narrow root helper in the installed, root-owned binary.
  The helper accepts only fixed operation IDs, independently rechecks package
  state, and never accepts arbitrary paths or commands from a user plan.
- APT decides unused packages; additionally protect the running and newest kernel.
  Snap removal applies only to revisions still marked disabled immediately before
  execution. Never uninstall an application merely because a newer version exists.
- Dry-run performs metadata inspection and read-only inventory queries only.
  It does not execute cleanup commands, request sudo, or save state.

## Code layout

```text
cmd/eversor/          CLI entry point
internal/app/        command parsing and orchestration
internal/core/       inventory, scan, plan, cleanup policy, and native backends
internal/report/     human-readable and JSON output
docs/                architecture, usage, provider rules, and development guide
.github/workflows/   verification and build automation
packaging/           Debian package metadata
scripts/             package assembly and verification
```

The initial implementation uses the Go standard library. Provider definitions and
tests ship with the application; downloaded rules cannot introduce executable
cleanup actions. New apps get dedicated evidence, paths, activity guards, and
synthetic preservation tests rather than a blanket cache sweep.

## Delivery and verification

Ship Linux binaries with a numbered version and source commit. Debian packages
install `/usr/bin/eversor`; source installation uses `/usr/local/bin/eversor`.
Both enable user cleanup and narrowly elevated package cleanup. Development
binaries support dry-run and user-only cleanup. Tagged releases are verified and
published by CI; upgrades remain explicit, with no background updater.

Test with disposable synthetic directories and fake command runners. Cover
symlink substitution, hard links, mount boundaries, active apps, preserved files,
read-only preview, partial failures, package parsing, and privilege boundaries.
Run formatting, unit/race tests, vet, and a build before pushing implementation
changes. Verify the installed binary with a live dry-run, never by using personal
data as deletion-test fixtures.

## References

- [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/0.8/)
- [Go os.Root](https://pkg.go.dev/os#Root)
- [Docker build-cache pruning](https://docs.docker.com/reference/cli/docker/builder/prune/)
