# Postwoman v1.4.1

**Release date:** 2026-11-22

Postwoman v1.4.1 is an API development desktop release for Windows and Linux.

> **Note:** v1.4.1 supersedes v1.4.0, whose Windows build failed before any release
> assets were published. This release ships the downloadable Windows installer,
> Windows portable ZIP, Debian package and AppImage.

## Features

- **release:** Add Windows installer and portable release pipeline ([a63e1b5](https://github.com/mtafakori1993/postwoman/commit/a63e1b5))
- Add AI providers and API discovery agent ([a59464f](https://github.com/mtafakori1993/postwoman/commit/a59464f))

## Improvements

- Restored natural interface typography: the platform system font is used instead of a bold-only Open Sans face, non-default font feature settings were removed, and forced leading overrides were relaxed ([a59464f](https://github.com/mtafakori1993/postwoman/commit/a59464f)).
- Hardened the Windows release build with reliable MinGW-w64/NSIS discovery, a toolchain verification step, execution-policy bypass, build-log artifacts and error annotations. The workflow now validates the executable/installer versions and portable ZIP contents before publishing.
- Fixed the Windows build script parse error (`$out:` variable expansion) that made the job fail immediately, and added a build-log artifact for future diagnosis.
- Fixed the empty Windows executable version metadata: `build/windows/info.json` now embeds a four-part file/product version, and the build stamps the executable with `scripts/windows-version` before verification so the app and installer always report 1.4.1.

## Discovery agent

- The **API Discovery Agent** is now a full-size workspace: a stage stepper (Setup → Analyze → Review → Import), a searchable and filterable endpoint list, per-endpoint tabs for details, parameters, schemas and the exact source window, and a collection preview before anything is imported.
- Fixed a crash after discovering projects that yield no detectable endpoints (for example a WordPress tree without recognisable routes). Discovery sessions now always return arrays instead of `null`, the review UI tolerates empty results, the app and dialog are wrapped in error boundaries, and a panic inside the agent is reported as a failed session instead of terminating the application.
- Large repositories now release the source of files that produced no routes and re-read them on demand, which keeps WordPress-sized checkouts within a comfortable memory budget.

**Full changelog:** https://github.com/mtafakori1993/postwoman/compare/v1.3.3...v1.4.1

## Downloads

| Platform | Asset | Notes |
| --- | --- | --- |
| Windows (installer) | `Postwoman-v1.4.1-windows-x64-setup.exe` | Recommended. Per-user install, no administrator privileges, Start Menu shortcut, optional desktop shortcut, clean upgrades. |
| Windows (portable) | `Postwoman-v1.4.1-windows-x64-portable.zip` | Extract anywhere and run `Postwoman.exe`. Data stays in the adjacent `data` folder. |
| Debian / Ubuntu | `postwoman_1.4.1_amd64.deb` | Native package with GTK/WebKitGTK dependencies resolved by apt. |
| Linux (portable) | `Postwoman-1.4.1-x86_64.AppImage` | Single portable executable; no installation required. |
| Checksums | `SHA256SUMS.txt` | SHA-256 hashes for every attached asset. |

## Installation

### Windows — installer (recommended)

Download `Postwoman-v1.4.1-windows-x64-setup.exe` and run it. The installer is per-user, installs under `%LOCALAPPDATA%\Programs\Postwoman`, creates a Start Menu shortcut and lets you choose whether to add a desktop shortcut. It installs the Microsoft WebView2 Runtime when missing. Upgrades overwrite the previous installation; your data in `%APPDATA%\Postwoman` is preserved.

### Windows — portable

Download `Postwoman-v1.4.1-windows-x64-portable.zip`, extract the whole folder to a writable location (USB drive included) and run `Postwoman.exe`. The included `portable.txt` marker keeps the database and WebView2 cache in the folder's `data` directory. Keep `Postwoman.exe`, `portable.txt` and `data` together when moving the app.

### Linux

Install the Debian package with `sudo apt install ./postwoman_1.4.1_amd64.deb`, or make the AppImage executable with `chmod +x Postwoman-1.4.1-x86_64.AppImage` and run it.

## Verifying downloads

Download `SHA256SUMS.txt` alongside the assets and verify a file with:

```bash
sha256sum -c SHA256SUMS.txt
```

On Windows PowerShell:

```powershell
Get-FileHash .\Postwoman-v1.4.1-windows-x64-setup.exe -Algorithm SHA256
```

Requires Windows 10/11 x64 or a 64-bit Linux distribution with GTK 3 and WebKitGTK 4.1.

