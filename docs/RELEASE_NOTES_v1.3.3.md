# Postwoman v1.3.3

Open Sans is now the default interface font at 15 px.

## Downloads

| File | Notes |
| --- | --- |
| `postwoman_1.3.3_amd64.deb` | Debian/Ubuntu installer (`sudo apt install ./postwoman_1.3.3_amd64.deb`) |
| `Postwoman-1.3.3-x86_64.AppImage` | Portable, no installation (`chmod +x` then run) |

## Changed

- **Default interface font: Open Sans** — the bundled `OpenSans-Bold.ttf` face is
  registered for both the normal and bold weights, so the browser never
  synthetically emboldens it (fake-bolding a bold face is what made UI text look
  smeared in earlier builds). Light/regular/semibold/extra-bold files were not
  bundled, so every weight renders with the same crisp face and hierarchy comes
  from size and colour.
- **Default interface size: 15 px** (was 16 px), adjustable 11–20 px in
  *Settings → Appearance → Interface font size*.
- *Settings → Appearance → Interface font* now offers **Open Sans (default)**,
  System UI and Source Code Pro (monospace). Source Code Pro remains the bundled
  program/code font for every editor.
- Existing installations whose stored settings still hold a previous default
  (13 px or 16 px, font “system”) are migrated to the new defaults once;
  deliberate choices such as 12 px or 18 px are preserved.

## Bundled fonts

| Font | Role | Licence |
| --- | --- | --- |
| Open Sans (Bold) | Interface | Apache License 2.0 (`frontend/src/assets/fonts/OpenSans-LICENSE.txt`) |
| Source Code Pro (Regular) | Code editors / monospace | SIL OFL 1.1 (`frontend/src/assets/fonts/SourceCodePro-OFL.txt`) |

## Compatibility

- No data migration required; collections, environments, tabs and history are untouched.
- If the interface text still uses the old face after upgrading, pick
  *Settings → Appearance → Interface font → Open Sans (default)* once.
