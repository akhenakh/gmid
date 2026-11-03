#!/bin/sh
# Build gmid.app around an already-built binary.
#
# Usage: packaging/macos/bundle.sh <binary> <version>
#
# The binary is copied to Contents/MacOS/gmid and the bundled assets to
# Contents/Resources/assets. internal/config.AssetsDir finds that layout from
# the executable path, so gmid launched from the bundle loads its soundfont,
# fonts and demo songs without environment variables.
set -eu

BIN="${1:?usage: bundle.sh <binary> <version>}"
VERSION="${2:-${GORELEASER_CURRENT_TAG:-0.0.0}}"
VERSION="${VERSION#v}"

APP="build/gmid.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

cp "$BIN" "$APP/Contents/MacOS/gmid"
chmod +x "$APP/Contents/MacOS/gmid"
cp -R assets "$APP/Contents/Resources/assets"
cp assets/icons/gmid.icns "$APP/Contents/Resources/gmid.icns"

cat >"$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>gmid</string>
	<key>CFBundleDisplayName</key>
	<string>gmid</string>
	<key>CFBundleIdentifier</key>
	<string>space.inair.gmid</string>
	<key>CFBundleExecutable</key>
	<string>gmid</string>
	<key>CFBundleIconFile</key>
	<string>gmid.icns</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleShortVersionString</key>
	<string>${VERSION}</string>
	<key>CFBundleVersion</key>
	<string>${VERSION}</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
</dict>
</plist>
PLIST

printf 'APPL????' >"$APP/Contents/PkgInfo"

# The release is not notarised. An ad-hoc signature gives macOS a stable
# identity; the Homebrew cask clears the quarantine flag at install time.
codesign --force --sign - "$APP" 2>/dev/null || true

echo "$APP"
