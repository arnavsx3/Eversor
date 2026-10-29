# Eversor

An open-source Linux cleanup tool for understanding disk usage and reclaiming
space from known caches and obsolete files.

The architecture is documented in [docs/architecture.md](docs/architecture.md).
Eversor provides a read-only storage preview and a confirmed cleanup command.
It detects supported software and known cache artifacts on the current machine.
Unknown directories are shown in the storage tree and never automatically purged.

Download an architecture-matched `.deb` from [Releases](https://github.com/arnavsx3/Eversor/releases/latest),
verify it against `SHA256SUMS`, then install it with APT:

```sh
sudo apt install ./eversor_0.1.0_amd64.deb
eversor version
```

Use the arm64 file on ARM systems. If switching from a source installation,
remove the old `/usr/local/bin/eversor` with `make uninstall` first. Release
binaries do not require Go to be installed.

Alternatively, build and install with Go 1.25 or newer:

```sh
make check
make install
eversor dry-run
eversor clean
```

Development binaries can clean user caches. Installation to a root-owned system
path enables the separate administrator-authorized package cleanup helper.
See [installation and usage](docs/usage.md) for flags, reports, and uninstalling.
CI produces Linux amd64 and arm64 archives and `.deb` packages with checksums.
The version comes from `VERSION`; the command also prints the source commit.

Updates are manual. Install a newer downloaded `.deb` with APT, or run
`make update` inside a clean source checkout to pull, verify and reinstall it.
APT does not fetch GitHub releases automatically; no Eversor APT repository is
configured. Use one installation method to avoid an older binary masking a newer
one. See the usage guide for upgrades and removal.

## Workflow

```text
eversor dry-run
eversor clean
```

- **Inspect:** build a directory tree showing where storage is being used.
- **Identify:** detect installed tools and known, rebuildable caches, with an
  explanation of each cleanup candidate.
- **Preview:** show the proposed cleanup, estimated reclaimable space, and skipped
  items without deleting files.
- **Clean:** revalidate the plan, use native cleanup commands where appropriate,
  and report what was removed and the space actually recovered.

Large or unfamiliar directories should remain inspection results until there is
an explicit, supported rule for cleaning them. Personal files, project source,
credentials, browser profiles, and databases must be preserved.

The initial target is Ubuntu. Eversor uses Go and an application-provider registry
to detect the software installed on each machine. Unsupported folders remain
inspection-only rather than being guessed safe to delete.

See [supported providers](docs/providers.md) for the exact cleanup boundaries.
Both commands show a compact preview by default. Add `--tree` for the storage
overview. Both accept `--no-tree`, `--tree-depth 1` through `4`, `--no-projects`,
`--user-only`, and `--json`. `clean` shows a fresh preview and asks for confirmation;
`--yes` confirms it non-interactively. JSON cleanup requires `--yes`.

There is no scheduler, telemetry, or automatic report file. To save a preview,
redirect `eversor dry-run --json` to a file you control. Such reports contain local
paths and should be treated as private. Cache regeneration can make later builds
and downloads take longer. Measured recovery includes concurrent disk activity.

## Contributing

Keep each change focused and review its diff before committing. Use Conventional
Commits, validate the change, and push each completed change before starting the
next one. Read [CONTRIBUTING.md](CONTRIBUTING.md), the
[development guide](docs/development.md), and [AGENTS.md](AGENTS.md).

## License

Eversor is licensed under the [MIT License](LICENSE).
