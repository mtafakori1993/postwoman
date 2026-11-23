# Postwoman v1.5.1

**Release date:** 2026-11-23

Postwoman v1.5.1 is an API development desktop release for Windows and Linux.

## Bug Fixes

- **ai:** Parse OpenAI-compatible responses leniently and explain failures ([183dcd1](https://github.com/mtafakori1993/postwoman/commit/183dcd1))

**Full changelog:** https://github.com/mtafakori1993/postwoman/compare/v1.5.0...v1.5.1

## Downloads

| Platform | Asset | Notes |
| --- | --- | --- |
| Windows (installer) | `Postwoman-v1.5.1-windows-x64-setup.exe` | Recommended. Per-user install, no administrator privileges, Start Menu shortcut, optional desktop shortcut, clean upgrades. |
| Windows (portable) | `Postwoman-v1.5.1-windows-x64-portable.zip` | Extract anywhere and run `Postwoman.exe`. Data stays in the adjacent `data` folder. |
| Debian / Ubuntu | `postwoman_1.5.1_amd64.deb` | Native package with GTK/WebKitGTK dependencies resolved by apt. |
| Linux (portable) | `Postwoman-1.5.1-x86_64.AppImage` | Single portable executable; no installation required. |
| Checksums | `SHA256SUMS.txt` | SHA-256 hashes for every attached asset. |

## Installation

### Windows — installer (recommended)

Download `Postwoman-v1.5.1-windows-x64-setup.exe` and run it. The installer is per-user, installs under `%LOCALAPPDATA%\Programs\Postwoman`, creates a Start Menu shortcut and lets you choose whether to add a desktop shortcut. It installs the Microsoft WebView2 Runtime when missing. Upgrades overwrite the previous installation; your data in `%APPDATA%\Postwoman` is preserved.

### Windows — portable

Download `Postwoman-v1.5.1-windows-x64-portable.zip`, extract the whole folder to a writable location (USB drive included) and run `Postwoman.exe`. The included `portable.txt` marker keeps the database and WebView2 cache in the folder's `data` directory. Keep `Postwoman.exe`, `portable.txt` and `data` together when moving the app.

### Linux

Install the Debian package with `sudo apt install ./postwoman_1.5.1_amd64.deb`, or make the AppImage executable with `chmod +x Postwoman-1.5.1-x86_64.AppImage` and run it.

## Verifying downloads

Download `SHA256SUMS.txt` alongside the assets and verify a file with:

```bash
sha256sum -c SHA256SUMS.txt
```

On Windows PowerShell:

```powershell
Get-FileHash .\Postwoman-v1.5.1-windows-x64-setup.exe -Algorithm SHA256
```

Requires Windows 10/11 x64 or a 64-bit Linux distribution with GTK 3 and WebKitGTK 4.1.

