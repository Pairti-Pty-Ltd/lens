# macOS releases

The release workflow runs on tags of the form `vMAJOR.MINOR.PATCH`. It builds on native Intel and Apple Silicon runners. Complete the clean-Mac Finder checks on both architectures before tagging a release.

Configure these Actions secrets in the Lens repository:

| Secret | Value |
| --- | --- |
| `MACOS_CERT_P12_BASE64` | Base64-encoded PKCS#12 export of the Developer ID Application identity and private key |
| `MACOS_CERT_PASSWORD` | Password for that export |
| `APPLE_ID`, `APPLE_TEAM_ID`, `APPLE_APP_PASSWORD` | Apple account, developer team and app-specific password used by notarytool |
| `HOMEBREW_TAP_TOKEN` | Credential with Contents write access to `Pairti-Pty-Ltd/homebrew-tap` |

Initialize the tap repository with a default branch before the first release. Each runner signs the nested report tools and Lens.app, submits the app to Apple, staples the ticket, and checks Gatekeeper. Once both archives pass, the publish job uploads them and `SHA256SUMS` to a draft release before making it public.

The final job downloads the published archives, verifies their checksums, and writes `Casks/pairti-lens.rb` in the tap. If a job fails, rerun the failed jobs for the same tag. The cask update refuses to downgrade an existing version.
