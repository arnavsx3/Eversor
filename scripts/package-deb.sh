#!/usr/bin/env bash
set -euo pipefail
release_version=${1:?release version required}
if [[ ! $release_version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo 'Package version must be MAJOR.MINOR.PATCH.' >&2
    exit 1
fi
staging=$(mktemp -d "${TMPDIR:-/tmp}/eversor-deb.XXXXXXXX")
trap 'rm -r -- "$staging"' EXIT
for arch in amd64 arm64; do
    root="$staging/$arch"
    install -d -m 0755 "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/doc/eversor/docs"
    sed -e "s/@VERSION@/$release_version/" -e "s/@ARCH@/$arch/" packaging/debian/control.in > "$root/DEBIAN/control"
    chmod 0644 "$root/DEBIAN/control"
    install -m 0755 "dist/eversor_linux_$arch/eversor" "$root/usr/bin/eversor"
    install -m 0644 LICENSE "$root/usr/share/doc/eversor/copyright"
    install -m 0644 LICENSE README.md AGENTS.md CONTRIBUTING.md "$root/usr/share/doc/eversor/"
    install -m 0644 docs/*.md "$root/usr/share/doc/eversor/docs/"
    dpkg-deb --root-owner-group -Zxz --build "$root" "dist/eversor_${release_version}_${arch}.deb"
done
