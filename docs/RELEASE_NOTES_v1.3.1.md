# Postwoman v1.3.1

Bug-fix release for v1.3.0.

## Downloads

| File | Notes |
| --- | --- |
| `postwoman_1.3.1_amd64.deb` | Debian/Ubuntu installer (`sudo apt install ./postwoman_1.3.1_amd64.deb`) |
| `Postwoman-1.3.1-x86_64.AppImage` | Portable, no installation (`chmod +x` then run) |

## Fixed

- **Font size control** — the interface font size is now an actual setting in
  *Settings → Appearance → Interface font size* (`−` / `+` / `Reset`,
  11–20 px). It scales the whole UI, including the small labels that were
  previously pinned to absolute pixel sizes. The code-editor font size gained
  the same stepper (9–24 px) and is applied through a CSS variable as well as
  the CodeMirror theme, so changes take effect immediately, including in
  read-only editors.
- **Interface typeface** — the UI no longer forces a monospace face with
  synthesised (fake) bold at a small base size, which made the chrome look
  distorted. The interface now defaults to a clean system sans stack, the code
  editors default to the bundled Source Code Pro, and *Interface font* lets you
  choose "System UI (recommended)" or "Source Code Pro (monospace)".
- **Generate-code dialog overflow** — the seven language tabs overflowed the
  modal. Languages are now a select, the dialog is capped at 85 % of the
  viewport height with an internally scrolling, non-wrapping code block, and
  every dialog in the app is capped at 92 % viewport height so no modal can
  spill off screen.

## Notes

- The app-wide text scale default moved from 13 px (which shrank the interface)
  to 16 px, matching the previous look. Existing installations keep their stored
  value and can adjust it with the new stepper.
- No database or settings migration is required; the new `uiFont` setting
  defaults to `system`.
