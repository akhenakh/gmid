#!/bin/sh
# Assemble a release archive from a binary built by `goreleaser build`.
#
# Usage: packaging/package.sh <platform> <version> <binary>
#
# platform is one of Linux_x86_64, Linux_arm64, Darwin_x86_64, Darwin_arm64,
# matching the archive names consumed by scripts/brew-bump.sh.
set -eu

PLATFORM="${1:?usage: package.sh <platform> <version> <binary>}"
VERSION="${2:?usage: package.sh <platform> <version> <binary>}"
BIN="${3:?usage: package.sh <platform> <version> <binary>}"

mkdir -p build
OUT="build/gmid_${VERSION}_${PLATFORM}.tar.gz"

case "$PLATFORM" in
Linux_*)
	STAGE="$(mktemp -d)"
	trap 'rm -rf "$STAGE"' EXIT
	cp "$BIN" "$STAGE/gmid"
	chmod +x "$STAGE/gmid"
	cp -R assets "$STAGE/assets"
	cp README.md LICENSE THIRD_PARTY_LICENSES.md "$STAGE/"
	cp -R docs "$STAGE/docs"
	tar -czf "$OUT" -C "$STAGE" \
		gmid assets docs README.md LICENSE THIRD_PARTY_LICENSES.md
	;;
Darwin_*)
	APP="$(packaging/macos/bundle.sh "$BIN" "$VERSION")"
	tar -czf "$OUT" -C "$(dirname "$APP")" "$(basename "$APP")"
	;;
*)
	echo "unknown platform: $PLATFORM" >&2
	exit 1
	;;
esac

echo "$OUT"
