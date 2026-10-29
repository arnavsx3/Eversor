# Supported cleanup providers

Eversor inventories Debian packages, executable paths, Snap revisions, Flatpak app
IDs, and user processes. An installed-app signal explains a match; a documented
cache artifact can also be recognized after its application was uninstalled.
There is no universal rule that labels every application's data safe to delete.

| Provider | Eligible data | Preserved data |
| --- | --- | --- |
| Chrome, Chromium, Firefox, Brave | Documented browser cache paths, including supported Snap/Flatpak locations | Cookies, history, passwords, bookmarks, profiles and extensions |
| VS Code, VSCodium, Codex, Claude, Antigravity, Discord, Slack | Named Cache, Code Cache, GPUCache and supported compiled-data directories | Settings, sessions, authentication, extension storage and installed extensions |
| Go | Default compiler cache and default module-download cache | Installed Go toolchains, source projects and installed Go executables |
| npm, Yarn, pnpm, pip | Supported downloaded-package/cache locations | Project dependencies, environments, global tools and configuration |
| Python uv | Native pruning of unused records in its verified default cache directory | Installed environments and toolchains; uv may retain shared or required records |
| Trivy, pre-commit, Go lint, mise | Known scan, hook-environment, lint and installer-download caches | Installed toolchains and custom source/configuration |
| Desktop thumbnails | Generated thumbnail cache | Original pictures and videos |
| Rust projects | A `target` directory with Cargo's cache marker and an adjacent `Cargo.toml`, discovered under non-hidden home directories | Source, manifests, lockfiles, databases and installed Rust toolchains |
| Docker | Native pruning of unused build cache through a verified and pinned local Unix socket | Images, containers, networks, volumes and database contents; remote contexts |
| APT | Downloaded installers and packages APT itself proposes for autoremove/purge | Running and newest installed kernels; manually installed or required packages |
| Snap | Revisions confirmed disabled again immediately before removal | Active revisions and application user data |
| systemd journal | Archived journals beyond the 14-day / 500 MB retention policy | Active journals; arbitrary application log directories |

Eligible directories must be owned by the current account. Symlinked cache roots,
mounted filesystems, special files, foreign-owned entries, active related apps,
and caches containing an open executable or file are skipped. Activity and mount
checks run again immediately before each cleanup action. Process inspection is
best effort under Linux permissions; these checks supplement explicit provider
boundaries rather than turn arbitrary directories into safe targets.

Shared files with additional hard links are excluded from reclaimable-size
estimates. Native tools may retain data, so their complete cache size is not a
promise of free space. `clean` measures the filesystem's actual available-space
change after all selected actions.

## Adding an application

1. Establish which locations contain rebuildable cache data, using upstream
   documentation or source. Do not select a whole profile or application-data root.
2. Add a provider in `internal/core/catalog.go`, including commands/packages/app
   IDs, supported XDG-relative paths, and related process names.
3. For a native cleanup action, add a fixed implementation in the executor. Never
   execute a command supplied by a downloaded rule or serialized plan.
4. Add synthetic tests for cache discovery, preservation of adjacent important
   data, active use, symlinks and identity changes. Update this table.

User-defined arbitrary cleanup paths and downloaded executable plugins are
intentionally unsupported. Additional applications can be added through reviewed
providers without weakening existing boundaries.
