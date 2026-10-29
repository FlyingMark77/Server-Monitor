#!/bin/bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
VERSION="${VERSION:-0.1.0}"
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]] || { echo 'Invalid VERSION' >&2; exit 1; }
[[ "$(uname -s)" == Darwin ]] || { echo 'Packaging requires macOS.' >&2; exit 1; }
export MACOSX_DEPLOYMENT_TARGET=12.0
export CGO_CFLAGS="${CGO_CFLAGS:-} -mmacosx-version-min=12.0"
export CGO_LDFLAGS="${CGO_LDFLAGS:-} -mmacosx-version-min=12.0"
export COPYFILE_DISABLE=1
wails build -platform darwin/arm64 -ldflags "-X main.appVersion=$VERSION"
STAGE="$(mktemp -d "${TMPDIR:-/tmp}/server-monitor-package.XXXXXX")"
trap 'rm -rf "$STAGE"' EXIT
ditto --norsrc build/bin/ServerMonitor.app "$STAGE/ServerMonitor.app"
cp LICENSE "$STAGE/ServerMonitor.app/Contents/Resources/LICENSE"
cp NOTICE "$STAGE/ServerMonitor.app/Contents/Resources/NOTICE"
xattr -cr "$STAGE/ServerMonitor.app"
dot_clean -m "$STAGE/ServerMonitor.app"
codesign --force --deep --sign - "$STAGE/ServerMonitor.app"
codesign --verify --deep --strict "$STAGE/ServerMonitor.app"
ln -s /Applications "$STAGE/Applications"
mkdir -p dist
hdiutil create -volname 'Server Monitor' -srcfolder "$STAGE" -ov -format UDZO "dist/ServerMonitor-$VERSION-macos-arm64.dmg"
ditto --norsrc "$STAGE/ServerMonitor.app" dist/ServerMonitor.app
find dist/ServerMonitor.app ! -name '._*' -exec xattr -c {} +
dot_clean -m dist/ServerMonitor.app
codesign --verify --deep --strict dist/ServerMonitor.app
shasum -a 256 "dist/ServerMonitor-$VERSION-macos-arm64.dmg" > "dist/ServerMonitor-$VERSION-macos-arm64.dmg.sha256"
echo "Built $ROOT/dist/ServerMonitor-$VERSION-macos-arm64.dmg (ad-hoc signed, not notarized)."
