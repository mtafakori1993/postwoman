# Changelog

All notable changes to Postwoman are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and
[Semantic Versioning](https://semver.org/).

## [1.4.1] - 2026-11-22

> Supersedes v1.4.0, whose Windows build failed before any release assets were published.

### Features

- **release:** Add Windows installer and portable release pipeline ([a63e1b5](https://github.com/mtafakori1993/postwoman/commit/a63e1b5))
- Add AI providers and API discovery agent ([a59464f](https://github.com/mtafakori1993/postwoman/commit/a59464f))

### Improvements

- Restore natural interface typography using the platform system font and default rendering ([a59464f](https://github.com/mtafakori1993/postwoman/commit/a59464f))
- Harden the Windows release build with reliable MinGW-w64/NSIS discovery, toolchain verification and artifact validation.
- Fix a PowerShell parse error (`$out:` expansion) that made the Windows build fail immediately, and upload a build log for diagnosis.

[1.4.1]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.4.1

## [1.4.0] - 2026-11-22

### Features

- **release:** Add Windows installer and portable release pipeline ([a63e1b5](https://github.com/mtafakori1993/postwoman/commit/a63e1b5))
- Add AI providers and API discovery agent ([a59464f](https://github.com/mtafakori1993/postwoman/commit/a59464f))

[1.4.0]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.4.0

## [1.3.3] - 2026-11-21

### Added
- Configurable interface font and font-size settings with migration for older values.
- Bundled program/code font and offline font rendering.

### Fixed
- Modal overflow in the Generate-code dialog and a global viewport-height cap.
- Font-size controls now apply consistently to small labels and editors.

## [1.3.2] - 2026-11-21

### Fixed
- Release packaging and installer documentation for the v1.3.1 fixes.
- Interface text uses a clean sans stack instead of a synthetic-bold monospace face.

## [1.3.1] - 2026-11-21

### Added
- Working interface font-size stepper (11–20 px) and code-editor size stepper (9–24 px).

### Fixed
- Distorted chrome typography and the Generate-code dialog overflow.

## [1.3.0] - 2026-11-21

### Added
- Advanced API client workbench: tabs, request body editor, response views,
  collections, environments, scripts, mock servers, WebSocket sessions, history,
  import/export and command palette.

[1.3.3]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.3.3
[1.3.2]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.3.2
[1.3.1]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.3.1
[1.3.0]: https://github.com/mtafakori1993/postwoman/releases/tag/v1.3.0
