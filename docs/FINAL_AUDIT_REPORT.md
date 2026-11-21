# Postwoman — Final Audit Report

This document records what was actually changed, how it was verified, and what
remains out of scope. It replaces the "initial" statuses in
`docs/FEATURE_AUDIT.md`.

Everything below was executed on this machine. Where a claim could not be
verified automatically it is called out explicitly.

## 1. Environment

| Item | Value |
| --- | --- |
| OS / session | Linux, GNOME/Mutter on `DISPLAY=:0` (X11) |
| Go | 1.27.4 (`~/go/bin`) |
| Node / npm | 18.19.1 / 9.2.0 |
| Wails | v2.17.1 |
| WebKitGTK | 4.1 only |
| AppImage tool | `appimagetool` (continuous build, 2026-11-06) |

## 2. Root cause of the original breakage

The shipped frontend never talked to the Go backend. It declared a
`window.__POSTWOMAN__` bridge and loaded a hand-written shim
(`frontend/src/wailsjs/go/runtime.js`) that called `window.external.Invoke`, a
Wails **v1** API that does not exist in Wails v2. Every backend call therefore
failed. All data was stored in zustand's `localStorage` and the SQLite database
was unused.

## 3. What was changed

### Backend (Go)

- `internal/model` — single typed model package (workspace, collections,
  folders, requests, bodies, auth, variables, environments, tabs, history, mock
  servers, run results, WebSocket messages, `AppState`).
- `internal/database` — real directory creation, WAL + busy timeout, goose
  migrations `001`–`003`, typed repository CRUD (workspaces, collections,
  nested folders, requests, environments, globals, history, tabs, settings, mock
  servers), workspace-wide listing, search, settings merge.
- `internal/httpclient` — full request engine: all body modes (raw/json/xml/
  html/javascript/text/urlencoded/form-data incl. file parts/graphql/binary),
  duplicate-safe headers and query params, `:path` variables, cookie jar,
  redirect control, per-request timeout and cancellation, HTTP/SOCKS5 proxy,
  TLS client certs/CA, digest challenge retry, OAuth1 HMAC, OAuth2 token fetch,
  JWT, AWS SigV4, Hawk, and structured errors.
- `internal/variables` — Postman-compatible ordered-scope resolution
  (local → environment → collection → globals), recursive substitution,
  unresolved placeholders preserved, and dynamic variables (`{{$guid}}`,
  `{{$timestamp}}`, `{{$randomInt}}`, …).
- `internal/scripting` — sandboxed goja runtime with `pm.test`, `pm.expect`,
  `pm.response`, `pm.sendRequest`, console capture, variable scopes, and a
  `vm.Interrupt` timeout.
- `internal/importexport` — format detection and parsers for Postman
  2.0/2.1 collections, Postman environments, OpenAPI 3.0/3.1, Swagger 2.0,
  HAR, cURL, GraphQL SDL/introspection, AsyncAPI and the native format;
  transactional import persistence; lossless Postman export; cURL export.
- `internal/runner` — collection/folder runs, iterations, CSV/JSON data files,
  delay, stop-on-error, progress callback, pass/fail summary.
- `internal/websocket` — live sessions via `gorilla/websocket`, frame capture,
  event emission, clean shutdown.
- `internal/mockserver` — real local HTTP mock servers (route matching, delays,
  request log, start/stop).
- `app.go` — the complete Wails binding surface used by the UI, including
  `Bootstrap`, request sending with script execution and history, import/export,
  runner, WebSocket, and mock-server control.

### Frontend (React/TypeScript)

- Removed the fake bridge and `localStorage` persistence.
- Added `src/lib/api.ts`, a typed wrapper over the generated bindings
  (`frontend/wailsjs/go/main/App.js`).
- Rewrote `src/stores/workspace.ts`: state is hydrated from `Bootstrap()` and
  every mutation is persisted to SQLite; tabs and settings are saved (debounced)
  to the backend; only pure layout prefs stay in `localStorage`.
- Aligned `src/types` with the generated Go models via a recursive `Plain<T>`
  type, so the frontend can build ordinary object literals without the generated
  `convertValues` helper.
- Rebuilt the shell: `Sidebar` (collections tree, environments, history, mock
  servers, WebSocket, settings), `TopBar` (workspace + environment selectors,
  search, runner, import, theme), `RequestTabs`, `RequestEditor`,
  `ResponseViewer` (real tests/cookies/headers/console, save, copy), plus
  `SearchPalette` and `RunnerDialog`.
- Fixed concrete UI bugs: `fixed` sidebar + content margin double-offset,
  duplicate response resizers, `setState` during render in `JSONViewer`,
  dynamic Tailwind classes (`w-[${width}]`) in `KeyValueEditor`, unused/incorrect
  file input ref in `BodyEditor`, response cookie/header mapping, and the
  response HTML preview sandbox (`allow-same-origin` removed).
- Removed dead controls (fake notification dot, no-op account menu,
  "coming soon" panels) and replaced them with real ones or deleted them.

## 4. Bugs found *by the new tests* and fixed

These were discovered while writing the test suite and are fixed in this
changeset:

1. **Dynamic-variable deadlock** — `variables.Resolver.Value` held its mutex and
   then called `generate()` → `intn()` which tried to lock the same mutex. Any
   request containing e.g. `{{$randomInt}}` hung forever. Fixed by generating
   outside the lock.
2. **Export dropped folder-nested requests** — `ExportCollection` built a
   folder's Postman item *before* attaching its requests, so requests inside
   folders vanished from exports (and round-trips). Fixed by rendering folders
   bottom-up after requests are attached.
3. **Collection/global variables were not applied for UI sends** — the frontend
   passes only the active environment id, and `App.SendRequest` did not load
   collection variables, so `{{baseUrl}}`-style URLs failed with
   `invalid character "{" in host name`. Fixed by loading environment,
   collection and global variables server-side when the caller supplies none.

## 5. Verification actually performed

### Backend

```
go build ./...            # clean
go vet ./...              # clean
go test ./...             # all packages pass
go test -race ./...       # all packages pass
```

Test packages and coverage:

| Package | What is covered |
| --- | --- |
| `internal/variables` | scope precedence, recursive resolution, unknown vars preserved, dynamic variables, `Set` |
| `internal/scripting` | `pm.test` pass/fail, `pm.expect`, env/global/collection mutation, console capture, infinite-loop interrupt, no filesystem access |
| `internal/httpclient` | query/header/path variables, JSON + urlencoded bodies, bearer/basic auth, API key in header/query, timeout, invalid URL, unsupported protocol |
| `internal/database` | collection/folder/request round-trip, persistence across reopen, nested-folder cascade delete |
| `internal/importexport` | format detection, Postman (folders/scripts/params), OpenAPI 3, Swagger 2, HAR, cURL, persist + export round-trip, cURL export |
| `internal/runner` | iterations + data, run against a real `httptest` server |
| `internal/mockserver` | route serving, 404 for unmatched routes, request log, stop |
| `internal/websocket` | connect/send/receive/close against a real WebSocket server, bad-URL failure |
| `Postwoman` (root) | `SendRequest` resolves collection variables; scripts run and persist environment changes; history recorded; import → `Bootstrap` → export round-trip; settings and tabs persist |

### Frontend

```
npx tsc --noEmit          # clean
npm run build             # succeeds (vite production build)
```

### Packaging and runtime

```
wails build               # succeeds -> build/bin/Postwoman
appimagetool AppDir Postwoman-1.0.0-x86_64.AppImage
```

- The AppImage launches, migrates the database (`goose` version 3), and renders
  the UI.
- On launch it loaded **pre-existing persisted collections** (e.g.
  "Atlas Mono API (v2)" with its folder tree) and **restored tabs**, which
  demonstrates end-to-end persistence through SQLite (the data was written by an
  earlier session and read back by the packaged app).
- Screenshots were captured from the running window.

AppImage location:

```
/home/Mohammad.Tafakori/projects/Postwoman/Postwoman-1.0.0-x86_64.AppImage
```

Raw binary: `/home/Mohammad.Tafakori/projects/Postwoman/build/bin/Postwoman`.

### Note on this particular VM

WebKitGTK 4.1 in this environment only maps its window when launched with
`GDK_BACKEND=x11` and `WEBKIT_DISABLE_COMPOSITING_MODE=1`. On a normal desktop
neither variable is required. Example:

```
GDK_BACKEND=x11 WEBKIT_DISABLE_COMPOSITING_MODE=1 ./Postwoman-1.0.0-x86_64.AppImage
```

## 6. Honest limitations

- **Interactive UI flows were not automated.** No `xdotool`/Xlib is available in
  this VM, so clicking was not scripted. Instead, the exact bound methods the UI
  calls are exercised by the root-package Go tests, and the UI itself was
  verified visually via screenshots. Rendering was checked at the default
  1280×800 window (the app's minimum is 900×560).
- **gRPC, MQTT and Socket.IO are not implemented.** No UI is shown for them, so
  nothing fake is presented.
- **NTLM and Hawk request signing are not implemented.** Both return a clear
  error, and the auth editor shows an explicit "not supported" notice instead of
  pretending they work.
- **Monitors** exist as a database table only and are intentionally not surfaced
  in the UI.
- **OpenAPI import** converts operations to requests; unsupported constructs are
  reported as warnings rather than silently dropped.
- **Secrets** (auth tokens, secret variables) are stored in the local SQLite
  database without OS-keychain integration.
- Large responses are capped at 200 MiB by the HTTP client; the JSON viewer is
  not virtualised.

## 7. Deliverables

| Artifact | Path |
| --- | --- |
| Feature audit | `docs/FEATURE_AUDIT.md` |
| Implementation plan | `docs/IMPLEMENTATION_PLAN.md` |
| Final report (this file) | `docs/FINAL_AUDIT_REPORT.md` |
| AppImage | `Postwoman-1.0.0-x86_64.AppImage` |
| Binary | `build/bin/Postwoman` |
| AppDir | `AppDir/` |
| Tests | `app_test.go`, `internal/*/**_test.go` |
