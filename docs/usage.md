# Usage and installation

## Install a release

Download the `.deb` matching `dpkg --print-architecture` and `SHA256SUMS` from
[GitHub Releases](https://github.com/arnavsx3/Eversor/releases/latest). Keep only
the selected package and the manifest in your download directory, then verify
its checksum (replace the filename for ARM systems):

```sh
sha256sum --ignore-missing --check SHA256SUMS
sudo apt install ./eversor_0.1.0_amd64.deb
eversor version
```

Check that verification reports `OK` for the package you are installing. A
checksum verifies integrity against the accompanying trusted manifest; it is not
a cryptographic publisher signature.

The package installs a root-owned `/usr/bin/eversor` without Go or Make on the
destination machine. Downloading a later `.deb` and installing it with APT
upgrades the package. Remove it with `sudo apt remove eversor`; removal does not
undo earlier cleanup or remove your source checkout. Ubuntu's regular APT updates
do not download new Eversor versions from GitHub. No updater or schedule is added.

If switching from a source installation, first run
`make -C ~/Projects/Eversor uninstall`. The source installer uses
`/usr/local/bin/eversor`, which would otherwise take precedence over the package's
`/usr/bin/eversor`. Use `type -a eversor` to see installed copies.

## Build from source

Install [Go 1.25 or newer](https://go.dev/doc/install), then:

```sh
git clone https://github.com/arnavsx3/Eversor.git
cd Eversor
make check
make install
eversor version
```

Installation places a root-owned binary at `/usr/local/bin/eversor`. The shell
finds it without changing your home directory or PATH. Source code and local
build outputs remain in the checkout. Use `make uninstall` to remove the binary.

Update a source installation with:

```sh
make -C ~/Projects/Eversor update
```

This requires a clean checkout, fast-forwards its tracked branch, runs checks and
reinstalls. Local edits or a divergent branch stop the update. Do not use it to
update a `.deb` installation; install the newer package instead.

CI also builds Linux amd64 and arm64 archives containing the binary and licence,
with a SHA-256 manifest. When installing a prebuilt archive, verify its checksum
against the accompanying trusted manifest, extract the appropriate architecture,
then use `sudo install -o root -g root -m 0755 eversor /usr/local/bin/eversor`.

## Inspect

```sh
eversor dry-run
eversor dry-run --tree
eversor dry-run --tree-depth 1
eversor dry-run --no-tree
eversor dry-run --json
```

Preview does not delete files, request administrator access, or save a report.
The default output is a compact list of supported cleanup candidates with
reasons and estimates. Use `--tree` to add a full home storage overview; an
explicit `--tree-depth` also enables it. Missing apps require no configuration;
known artifacts can be found even after their app was uninstalled. Unknown folders in the tree stay untouched.

The tree does not follow symlinks or cross mounts. It inspects up to one million
entries and reports if the tree is partial. The general storage walk is skipped
by default; `--no-tree` remains supported for existing scripts. It cannot be
combined with `--tree` or `--tree-depth`. `--no-projects` also omits Rust
build-cache discovery.

## Clean

```sh
eversor clean
eversor clean --user-only
eversor clean --no-projects
eversor clean --yes --json
```

`clean` makes a new plan rather than trusting a previously saved preview. Confirm
the displayed plan with `y` or use `--yes`. Empty input or EOF cancels cleanup.
JSON cleanup requires `--yes` so the output is not mixed with a confirmation
question. Use `--user-only` to omit system package and archived-journal cleanup.
Ctrl+C cancels a pending confirmation. An interrupted cleanup exits with status 1
and reports interruption; actions that already completed are not rolled back.

Run the public commands from your normal account. Only fixed system operations
are elevated through sudo; the password prompt is for those operations. Cache
files and project build outputs are never deleted as root.

Open supported apps, active builds, cache-root substitutions, unknown ownership,
and mounted caches are skipped. Files detected as in use are also skipped. Close
apps/builds and run again if you want their eligible caches included.

Each action reports `cleaned`, `skipped` or `failed`. Independent failures do not
prevent other eligible actions from being reported. Exit status is 0 for a
successful preview, cancellation or cleanup without failed actions; 1 for an
inspection/execution failure; and 2 for invalid arguments. A failed run can have
partially completed independent actions; read the per-action results.

The final space delta samples the root and home filesystems, eligible cache paths,
and standard native-tool storage locations before and after cleanup. Shared
filesystems are counted once, so a separate home or boot drive is accounted for.
Unavailable or changed filesystems produce measurement notes. Custom Docker
storage on a separate drive may be outside these observed locations.
The delta includes concurrent disk activity and can be negative when other apps
write data. Shared hard-linked files and native tools make estimates differ from
actual recovery. No fixed amount of recovered space is promised.

## Privacy and persistence

Eversor has no scheduler, telemetry, network service, or automatic report log.
Cleanup creates an empty concurrency-lock file in the runtime directory (or the
system temporary directory when no runtime directory is available).

To save a report yourself:

```sh
eversor dry-run --json > eversor-preview.json
eversor clean --yes --json > eversor-cleanup.json
```

Reports include local filesystem paths and command output. Keep real reports
private; use synthetic paths in bug reports. Never post credentials, browser
databases, or application data to demonstrate an issue.
