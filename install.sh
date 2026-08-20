#!/bin/sh
# Installs the socialsight CLI.
#
#   curl -fsSL https://raw.githubusercontent.com/SocialSight/cli/main/install.sh | sh
#
# Override the install location or version:
#
#   curl -fsSL .../install.sh | sh -s -- --prefix "$HOME/.local" --version v0.2.0
set -eu

repo="SocialSight/cli"
prefix="${PREFIX:-/usr/local}"
version="${VERSION:-}"

error() {
  echo "error: $1" >&2
  exit 1
}

while [ $# -gt 0 ]; do
  case "$1" in
    --prefix)
      [ $# -ge 2 ] || error "--prefix requires a value"
      prefix="$2"
      shift 2
      ;;
    --version)
      [ $# -ge 2 ] || error "--version requires a value"
      version="$2"
      shift 2
      ;;
    *)
      error "unknown argument: $1"
      ;;
  esac
done

os="$(uname -s)"
case "$os" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) error "unsupported OS: $os (see README.md for manual install / npm)" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) error "unsupported architecture: $arch" ;;
esac

if [ -z "$version" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" \
    | grep '"tag_name":' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
  [ -n "$version" ] || error "could not determine the latest release (none tagged yet?); pass --version vX.Y.Z"
fi

version_num="$(echo "$version" | sed 's/^v//')"
archive="socialsight_${version_num}_${os}_${arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${version}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "Downloading socialsight ${version} for ${os}/${arch}..."
curl -fsSL "${base_url}/${archive}" -o "${tmpdir}/${archive}" \
  || error "failed to download ${base_url}/${archive} (does this release/platform combination exist?)"
curl -fsSL "${base_url}/checksums.txt" -o "${tmpdir}/checksums.txt" \
  || error "failed to download checksums.txt for ${version}"

expected="$(grep " ${archive}\$" "${tmpdir}/checksums.txt" | awk '{print $1}')"
[ -n "$expected" ] || error "no checksum found for ${archive} in checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmpdir}/${archive}" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmpdir}/${archive}" | awk '{print $1}')"
else
  error "need sha256sum or shasum to verify the download"
fi
[ "$expected" = "$actual" ] || error "checksum mismatch for ${archive}: expected ${expected}, got ${actual}"

tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}"
[ -f "${tmpdir}/socialsight" ] || error "archive did not contain a socialsight binary"

mkdir -p "${prefix}/bin"
install -m 755 "${tmpdir}/socialsight" "${prefix}/bin/socialsight"

# Unsigned/unnotarized binaries get Gatekeeper-quarantined on macOS, which
# otherwise blocks the first run with an "unidentified developer" prompt.
if [ "$os" = "darwin" ] && command -v xattr >/dev/null 2>&1; then
  xattr -d com.apple.quarantine "${prefix}/bin/socialsight" 2>/dev/null || true
fi

echo "Installed socialsight ${version} to ${prefix}/bin/socialsight"

case ":$PATH:" in
  *":${prefix}/bin:"*) ;;
  *) echo "Note: ${prefix}/bin is not on your PATH. Add it, e.g.: export PATH=\"${prefix}/bin:\$PATH\"" ;;
esac
