# Postwoman v1.3.2

Installers for the v1.3.1 fixes, now attached to the GitHub release as assets.

## Downloads

| File | Notes |
| --- | --- |
| `postwoman_1.3.2_amd64.deb` | Debian/Ubuntu installer (`sudo apt install ./postwoman_1.3.2_amd64.deb`) |
| `Postwoman-1.3.2-x86_64.AppImage` | Portable, no installation (`chmod +x` then run) |

## Included fixes (from v1.3.1)

- **Font size actually works** — *Settings → Appearance → Interface font size*
  (`−` / `+` / `Reset`, 11–20 px) scales the whole interface, and *Code editor
  font size* (9–24 px) scales every editor immediately, including read-only
  response bodies. Small labels are rem-based, so they follow the setting.
- **No more distorted UI text** — the chrome uses a clean system sans stack by
  default instead of a synthetic-bold monospace face. *Interface font* lets you
  switch to Source Code Pro (monospace); Source Code Pro remains the bundled
  program/code font.
- **No modal overflow** — the Generate-code dialog (language select + 85vh cap
  with an internally scrolling code block) and every dialog (92vh global cap)
  now stay inside the window.

## Release engineering

- Releases are published automatically: pushing a `v*` tag runs the
  `Release` GitHub Actions workflow, which creates the GitHub release and
  attaches the installer artifacts from that tag.
- For older tags whose releases predate this, run **Actions → Release → Run
  workflow** and pass the tag (for example `v1.1.2`) to attach its artifacts.

## Compatibility

- No migration required; settings and databases from 1.1.x – 1.3.1 are reused.
- If your interface text still looks small after upgrading, use
  *Settings → Appearance → Interface font size → Reset* (16 px).
