Eversor's first numbered release: a local Ubuntu utility for previewing and
cleaning supported caches and package-manager-confirmed obsolete items.

- Compact `eversor dry-run` and confirmed `eversor clean`; optional `--tree` view.
- Detect supported browsers, editors and development tools on each machine.
- Include Cargo download archives, Kubernetes client cache and Helm indexes.
- Preserve personal files, credentials, source, installed tools and unknown data.
- Recheck active processes, directory identity, ownership and mount boundaries.
- Protect running/newest kernels and active Snap revisions during system cleanup.
- Report separate filesystems and support cancellation at confirmation.
- Supply static Linux amd64/arm64 archives and Ubuntu/Debian `.deb` packages.

Download the package matching `dpkg --print-architecture`, verify it against
`SHA256SUMS`, then install with `sudo apt install ./eversor_0.1.0_amd64.deb`
(use the arm64 filename when appropriate). If switching from a source install,
first run `make -C ~/Projects/Eversor uninstall` to remove the old
`/usr/local/bin/eversor` override. Debian packages install `/usr/bin/eversor`.

Run Eversor as your normal account. System cleanup requests sudo separately.
Preview first; cleanup can require later downloads or rebuilds. There are no
automatic updates, schedules, telemetry or persistent report logs. Support is
limited to reviewed providers and documented default cache locations; unknown
folders are not purged. Process detection is best effort under Linux permissions.

See the repository's usage and provider documentation for complete boundaries.
