# Releasing Postwoman

This document describes the version source, local build commands and the automated
GitHub release pipeline.

## Version source of truth

The repository root contains a `VERSION` file, for example:

```
1.3.3
```

Never edit version strings in individual files by hand. Use:

```bash
node scripts/update-version.mjs 1.4.0   # write
node scripts/update-version.mjs --check # verify
```

`update-version.mjs` updates `app.go`, `frontend/package.json`,
`frontend/package-lock.json`, `wails.json` (`info.productVersion`) and the README
version block. Release builds also inject the version into the executable:

```bash
go build -ldflags "-X main.appVersion=1.4.0"
```

## Normal release

Prerequisites: a clean working tree, `git`, `node`, and push access to `origin`.

```bash
./scripts/release.sh 1.4.0
```

The script:

1. validates `1.4.0` as MAJOR.MINOR.PATCH;
2. refuses to run with uncommitted changes or an existing tag;
3. updates all version metadata;
4. generates `docs/RELEASE_NOTES_v1.4.0.md` and updates `CHANGELOG.md`;
5. commits `chore(release): v1.4.0`;
6. creates annotated tag `v1.4.0`;
7. pushes the branch and tag.

The tag push starts `.github/workflows/release.yml`.

Use `--dry-run` to inspect the changes without committing, or `--no-push` to create
the commit and tag locally.

## Manual tag flow

```bash
node scripts/update-version.mjs 1.4.0
node scripts/generate-release-notes.mjs 1.4.0
git add VERSION app.go frontend/package.json frontend/package-lock.json \
  wails.json README.md CHANGELOG.md docs/RELEASE_NOTES_v1.4.0.md
git commit -m "chore(release): v1.4.0"
git tag -a v1.4.0 -m "Postwoman v1.4.0"
git push origin HEAD
git push origin v1.4.0
```

The workflow fails if the tag is not exactly `v<VERSION>`.

## Workflow jobs

| Job | Runner | Output |
| --- | --- | --- |
| `validate` | Ubuntu | Checks tag/VERSION/metadata consistency |
| `windows` | `windows-latest` | Portable ZIP + NSIS installer |
| `linux` | `ubuntu-24.04` | `.deb` + AppImage |
| `publish` | Ubuntu | Validates assets, writes checksums, creates the release |

Windows builds use:

```powershell
pwsh ./scripts/build-windows.ps1 -Version <version>
```

Prerequisites on Windows: Go 1.27+, Node 20+, Wails CLI v2.17.1, NSIS 3, and
MinGW-w64 `gcc` (the SQLite driver uses CGO). The script sets `CGO_ENABLED=1`
and locates common MinGW/NSIS installation directories automatically.

This runs `npm ci`, `npm run build`, then:

```text
wails build -platform windows/amd64 -trimpath -webview2 embed \
  -installscope user -nsis -ldflags "-X main.appVersion=<version>" -m -s
```

The NSIS template is `build/windows/installer/project.nsi`. It installs per-user
(`%LOCALAPPDATA%\Programs\Postwoman`), allows choosing the directory, creates a Start
Menu shortcut, offers an optional desktop shortcut, registers an uninstaller, and
never removes `%APPDATA%\Postwoman`.

The portable ZIP contains `Postwoman.exe`, `portable.txt`, `README.txt` and `LICENSE`.
`portable.txt` switches the database and WebView2 cache to `<folder>\data`.

Linux builds use `scripts/build-linux.sh`; packaging output goes to `dist/linux` by
default.

## Local verification checklist

Before tagging:

- [ ] `go test ./...` passes.
- [ ] `cd frontend && npx tsc --noEmit && npm run build` passes.
- [ ] `node scripts/update-version.mjs --check` passes.
- [ ] `node scripts/generate-release-notes.mjs <version> --stdout` shows sensible,
      categorized notes.
- [ ] `./scripts/build-linux.sh` produces the `.deb` and AppImage.
- [ ] `pwsh ./scripts/build-windows.ps1 -Version <version>` produces both Windows
      assets (or let CI build them).
- [ ] `./scripts/verify-release-assets.sh dist <version> all` passes.

Windows manual smoke test:

1. Install the new setup over an older version; confirm the version in the
   executable's file properties (or on the GitHub release page) updates and only one
   Start Menu entry exists.
2. Launch, create a request/collection, close, reinstall/upgrade, and confirm the
   data is preserved.
3. Uninstall and confirm `%APPDATA%\Postwoman` still exists.
4. Extract the portable ZIP to a writable folder (and a USB drive if available).
5. Confirm `data\postwoman.db` is created after launch, and that moving the folder
   keeps the data.
6. Remove/rename `portable.txt` and confirm the app uses `%APPDATA%\Postwoman`
   instead.
7. Verify `SHA256SUMS.txt` with PowerShell `Get-FileHash`.

## Security

- The workflow uses the automatic `GITHUB_TOKEN` with `permissions: contents: write`.
  No personal access token is required.
- Tokens are passed only through the `GH_TOKEN` environment variable and are never
  echoed. Do not add `set -x` to release steps.
- Release notes and checksums are generated from checked-out source, never from
  user-provided input.
- Do not commit build outputs, installers or API keys. `dist/`, `build/bin/`,
  NSIS temporary output and root packaging artifacts are ignored.

## Troubleshooting

| Symptom | Fix |
| --- | --- |
| `makensis.exe not found` | Install NSIS 3 (`choco install nsis`) and ensure its directory is on `PATH`. |
| Installer build fails on Linux | NSIS installers are built on the Windows runner; use `scripts/build-windows.ps1` on Windows. |
| `tag does not match VERSION` | Run `node scripts/update-version.mjs <version>`, commit, retag. |
| Portable starts with no data | Ensure `portable.txt` is next to `Postwoman.exe` and the folder is writable. |
| App does not start on Windows | Install the WebView2 Runtime; the installer does this automatically, and the portable executable embeds the bootstrapper. |
| Release asset validation fails | Run `./scripts/verify-release-assets.sh dist <version> all` locally and inspect the reported file. |

## Artifacts and retention

Workflow artifacts are retained for 7 days. Published GitHub release assets are
permanent until the release is deleted. The publish job updates an existing release
in place (assets are uploaded with `--clobber`), so rerunning `workflow_dispatch`
for the same tag repairs a partial release without creating duplicates.