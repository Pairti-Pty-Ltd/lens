#!/bin/bash
# Runs on an ephemeral macOS release runner after make app.
set -euo pipefail
umask 077
: "${CERT_P12_BASE64:?}" "${CERT_PASSWORD:?}" "${APPLE_ID:?}" "${APPLE_TEAM_ID:?}" "${APPLE_APP_PASSWORD:?}"
app="$RUNNER_TEMP/package/Lens.app"
keychain="$RUNNER_TEMP/lens-signing.keychain-db"
certificate="$RUNNER_TEMP/lens-signing.p12"
password=$(openssl rand -hex 32)
trap 'rm -f "$certificate"; security delete-keychain "$keychain" >/dev/null 2>&1 || true' EXIT

printf '%s' "$CERT_P12_BASE64" | base64 -D > "$certificate"
security create-keychain -p "$password" "$keychain"
security unlock-keychain -p "$password" "$keychain"
security import "$certificate" -k "$keychain" -P "$CERT_PASSWORD" -T /usr/bin/codesign
security set-key-partition-list -S apple-tool:,apple: -s -k "$password" "$keychain" >/dev/null
security list-keychains -d user -s "$keychain" "$HOME/Library/Keychains/login.keychain-db"
identity=$(security find-identity -v -p codesigning "$keychain" | awk '/"Developer ID Application:/ {print $2}')
[[ "$identity" =~ ^[A-Fa-f0-9]{40}$ ]]

for tool in typst pandoc pdftoppm; do
  codesign --force --options runtime --timestamp --sign "$identity" "$app/Contents/MacOS/$tool"
done
codesign --force --options runtime --timestamp --sign "$identity" "$app"
codesign --verify --deep --strict "$app"
submission="$RUNNER_TEMP/lens-notarization.zip"
ditto -c -k --keepParent "$app" "$submission"
xcrun notarytool submit "$submission" --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" \
  --password "$APPLE_APP_PASSWORD" --wait --output-format json > "$RUNNER_TEMP/notary.json"
test "$(plutil -extract status raw "$RUNNER_TEMP/notary.json")" = Accepted
xcrun stapler staple "$app"
xcrun stapler validate "$app"
spctl --assess --type execute --verbose "$app"
mkdir -p dist
ditto -c -k --keepParent "$app" "dist/Lens-$GITHUB_REF_NAME-$ARCH.zip"
