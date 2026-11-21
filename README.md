<p align="center">
  <img src="assets/postwoman-logo.png" width="150" alt="Postwoman logo" />
</p>

<h1 align="center">Postwoman</h1>

<p align="center">A desktop API development platform for building, sending and testing HTTP requests.</p>

> **Logo.** The image above is loaded from
> [`assets/postwoman-logo.png`](assets/postwoman-logo.png) (also committed as
> [`assets/postwoman-logo.base64`](assets/postwoman-logo.base64)). A relative path is
> used instead of an inline `data:` URI because GitHub's HTML sanitizer strips
> `data:` URLs from `<img src>`, which is why the image previously did not appear.
> The same PNG is reused for the application icon when packaging.

---

## Contents

- [Features](#features)
- [Install](#install)
- [Build from source](#build-from-source)
- [Packaging](#packaging)
- [Development](#development)
- [Usage](#usage)
- [Keyboard shortcuts](#keyboard-shortcuts)
- [Where your data lives](#where-your-data-lives)
- [Project layout](#project-layout)
- [Tech stack](#tech-stack)
- [License](#license)

## Features

- **Request builder** — every HTTP method, URL, query parameters, path variables,
  headers, and a complete request-body editor.
- **Body types** — None, Raw, JSON, XML, Text, JavaScript, HTML,
  Form Data (`multipart/form-data`, text fields and multi-file uploads),
  URL Encoded, Binary and GraphQL. Syntax highlighting, validation and
  JSON format/minify are built in.
- **Response viewer** — Pretty (syntax-highlighted), Raw and Preview
  (HTML, images, PDF, audio, video, XML) in one compact bottom bar, plus
  Cookies, Headers, Tests and Console tabs.
- **Tabs** — right-click any tab for *Close*, *Close Others*, *Close to the Right*,
  *Close All* and *Duplicate*. Unsaved changes are protected by a confirmation
  dialog with **Save & Close**.
- **Authentication** — Basic, Bearer, API Key, Digest, OAuth 1.0/2.0 and AWS SigV4.
- **Collections & folders** — organise and persist your requests.
- **Environments & variables** — global and environment scopes with
  `{{variable}}` interpolation.
- **Scripts** — pre-request and test scripts with a Postman-style `pm.*` API,
  test results and console output.
- **Mock servers** — define routes and serve canned responses locally.
- **WebSocket** — connect and exchange messages interactively.
- **History** — automatically records sent requests.
- **Import / export** — Postman collections and environments, OpenAPI 3, HAR and
  cURL in; Postman format out.
- **Theming** — light, dark and system themes.

## Install

### Debian / Ubuntu (`.deb`)

Runtime dependencies (GTK 3 and WebKitGTK 4.1) are pulled in automatically.

```bash
# resolve dependencies automatically
sudo apt install ./postwoman_1.1.2_amd64.deb
```

Or, on a system without `apt`:

```bash
sudo dpkg -i postwoman_1.1.2_amd64.deb
sudo apt-get -f install     # install any missing dependencies
```

This installs the `postwoman` launcher, a desktop entry and the application icon.
Launch it from your application menu, or run `postwoman` in a terminal.

Uninstall:

```bash
sudo apt remove postwoman
```

### AppImage (Linux)

No installation required — a single self-contained executable:

```bash
chmod +x Postwoman-1.1.2-x86_64.AppImage
./Postwoman-1.1.2-x86_64.AppImage
```

If your system lacks FUSE you can still run it:

```bash
./Postwoman-1.1.2-x86_64.AppImage --appimage-extract-and-run
```

## Build from source

### Prerequisites

- **Go 1.27+**
- **Node.js 20+** with npm
- **A C toolchain (CGO)** — the SQLite driver needs `gcc`
- **Linux development headers**:

```bash
sudo apt install build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
```

- *(optional)* the **Wails CLI**, only needed for `wails dev` / `wails build`:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Build

```bash
# 1. build the web UI
cd frontend && npm install && npm run build && cd ..

# 2. compile the desktop application (frontend/dist is embedded)
go build -tags desktop,production -ldflags "-w -s" -o build/bin/Postwoman .

# 3. run it
./build/bin/Postwoman
```

## Packaging

Both installables are built from the version in `app.go`, so the artifacts always
match the application.

```bash
# Debian / Ubuntu package  -> postwoman_1.1.2_amd64.deb
./packaging/debian/build-deb.sh

# Portable AppImage        -> Postwoman-1.1.2-x86_64.AppImage
./packaging/appimage/build-appimage.sh
```

The AppImage runtime is vendored at `packaging/appimage/runtime-x86_64`, so no
external `appimagetool` is required.

## Development

```bash
wails dev            # desktop app with live reload

cd frontend
npm run dev          # frontend only (Vite dev server)
npm run build        # type-check + production bundle

go test ./...        # backend test suite
```

## Usage

1. **Create a request** — `Ctrl+T` or the `+` button in the tab strip.
2. **Compose it** — pick a method, enter the URL, and use the *Params*, *Auth*,
   *Headers*, *Body*, *Scripts* and *Settings* tabs.
3. **Choose a body type** — the *Body* tab offers every standard mode; the
   appropriate `Content-Type` header is set automatically and can be overridden.
4. **Send** — `Ctrl+Enter`. The response appears below in the *Body* tab.
5. **Inspect** — switch the bottom bar between **Pretty**, **Raw** and **Preview**,
   or open *Cookies*, *Headers*, *Tests* and *Console*.
6. **Save & organise** — `Ctrl+S` stores the request in a collection.

### Tabs

Right-click a tab for the context menu:

| Action | Description |
| --- | --- |
| Close Tab | Close the tab (`Ctrl+W`) |
| Close Other Tabs | Keep only this tab |
| Close Tabs to the Right | Close every tab after this one |
| Close All Tabs | Close every tab |
| Duplicate Tab | Copy the tab next to the original |

Tabs can be dragged to reorder and middle-clicked to close (configurable in
*Settings → Tabs*).

### Request body types

| Mode | Editor | Notes |
| --- | --- | --- |
| None | — | No body sent |
| Raw | code editor | `text/plain` by default |
| JSON | code editor | validation, format & minify |
| XML | code editor | validation & format |
| Text | code editor | plain text |
| JavaScript | code editor | syntax highlighting |
| HTML | code editor | syntax highlighting |
| Form Data | key/value table | text fields and multi-file uploads |
| URL Encoded | key/value table | `application/x-www-form-urlencoded` |
| Binary | file picker | upload a file as the body |
| GraphQL | query + variables | JSON payload |

## Keyboard shortcuts

| Shortcut | Action |
| --- | --- |
| `Ctrl+B` | Toggle the sidebar |
| `Ctrl+K` | Open search |
| `Ctrl+T` | New tab |
| `Ctrl+W` | Close the current tab |
| `Ctrl+S` | Save the current request |
| `Ctrl+Enter` | Send the current request |

## Where your data lives

Requests, collections, environments, history and settings are stored in a local
SQLite database:

- `$XDG_DATA_HOME/Postwoman/postwoman.db`
- or `~/.local/share/Postwoman/postwoman.db`

## Project layout

```
app.go                  Wails-bound backend API
main.go                 Wails entry point (embeds frontend/dist)
internal/
  database/             SQLite storage + migrations
  httpclient/           HTTP engine (auth, redirects, cookies, multipart)
  importexport/         Postman / OpenAPI / HAR / cURL import + export
  mockserver/           Local mock HTTP server
  scripting/            Postman-style pm.* script engine
  variables/            {{variable}} resolution
  websocket/            WebSocket client
frontend/               React + TypeScript UI (Vite)
  src/components/       UI components
  src/stores/           Zustand state
packaging/debian/       .deb build script, control data, icons
packaging/appimage/     AppImage build script + vendored runtime
assets/                 Logo (PNG + base64)
```

## Tech stack

- **Wails v2.17** · **Go 1.27**
- **SQLite** (go-sqlite3) with goose migrations
- **React 19** · **TypeScript** · **Vite 7**
- **Tailwind CSS 3** · **Radix UI** (shadcn-style components)
- **Zustand** · **CodeMirror 6**

## License

MIT — see [`packaging/debian/copyright`](packaging/debian/copyright).
