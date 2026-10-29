#!/usr/bin/env bash
set -euo pipefail
release_version=${1:?release version required}
staging=$(mktemp -d "${TMPDIR:-/tmp}/eversor-package-check.XXXXXXXX")
trap 'rm -r -- "$staging"' EXIT
(cd dist && sha256sum --check SHA256SUMS)
for arch in amd64 arm64; do
    package="dist/eversor_${release_version}_${arch}.deb"
    [[ $(dpkg-deb --field "$package" Package) == eversor ]]
    [[ $(dpkg-deb --field "$package" Version) == "$release_version" ]]
    [[ $(dpkg-deb --field "$package" Architecture) == "$arch" ]]
    dpkg-deb --extract "$package" "$staging/$arch"
    [[ -x "$staging/$arch/usr/bin/eversor" ]]
    cmp "dist/eversor_linux_$arch/eversor" "$staging/$arch/usr/bin/eversor"
    tar -xzf "dist/eversor_linux_$arch.tar.gz" -C "$staging/$arch"
    cmp "$staging/$arch/eversor" "$staging/$arch/usr/bin/eversor"
    [[ -f "$staging/$arch/usr/share/doc/eversor/copyright" ]]
done
case $(uname -m) in
    x86_64) arch=amd64 ;;
    aarch64) arch=arm64 ;;
    *) echo 'Skipping native binary smoke test on this architecture.'; exit 0 ;;
esac
actual=$("$staging/$arch/usr/bin/eversor" version)
[[ $actual == "eversor $release_version ("*")" ]]
"$staging/$arch/usr/bin/eversor" --help > /dev/null
echo "Package verification passed: $actual"
