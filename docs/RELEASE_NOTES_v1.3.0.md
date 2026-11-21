# Postwoman v1.3.0

A large feature release that turns Post into a full API-development workbench
while keeping the existing architecture, UI language and data.

## Downloads

| File | Notes |
| --- | --- |
| `postwoman_1.3.0_amd64.deb` | Debian/Ubuntu installer (`sudo apt install ./postwoman_1.3.0_amd64.deb`) |
| `Postwoman-1.3.0-x86_64.AppImage` | Portable, no installation (`chmod +x` then run) |

## Highlights

### Requests
- **Custom HTTP methods** in addition to the standard set.
- **URL ⇄ Params two-way synchronisation**: pasting a URL populates the params
  table, editing params rewrites the URL, and query strings are no longer
  duplicated on the wire.
- **Path variables** (`/:id` or `{{id}}`) detected automatically, with
  missing-value warnings and a resolved-URL preview.
- **Bulk edit** for params/headers/form-data/variables (`key: value` lines).
- Per-request **proxy** and **TLS verification** settings; request name and
  Markdown-style documentation notes.

### Authentication
- **Auth inheritance** — request → folder → collection, with an explicit "No Auth"
  opt-out. (`inherit` was previously never resolved, so inherited auth silently
  did nothing.)
- **Real OAuth 2.0**: authorization code, **authorization code + PKCE**,
  client credentials, password and refresh-token grants, with token expiry
  tracking and automatic refresh. The previous "Get New Access Token" button
  was a stub.
- **JWT decoding** with expiry indicator for bearer/JWT tokens.
- Secret masking and reveal controls throughout.

### Responses
- **Wrap / no-wrap toggle** for body views (Pretty, Raw, Preview).
- **Meta tab** with method, URL, status, content-type, sizes, redirect state and
  request headers; one-click "copy response headers".
- **Response tools**: JSONPath extraction that saves results as
  environment/collection/global variables, structural diff against any
  historical response, and JSON Schema generation/validation.
- **JSON explorer**: expand/collapse all, copy value, JSON path and JSON
  Pointer, type labels and lazy rendering for very large payloads.
- Optional **AI assistant** (OpenAI-compatible endpoint, opt-in, consent-gated).

### Workflow
- **Command palette** (`Ctrl+K`) across requests, collections, environments,
  history, panels and actions.
- **Rebindable keyboard shortcuts** with a shortcut reference dialog.
- **Tabs**: pin/unpin, copy request URL, reopen closed tab; pinned tabs are
  protected from "close others"/"close all".
- **Cancellable requests** (real cancellation instead of a no-op).
- **Code generation** for cURL, raw HTTP, `fetch`, Node.js, Python, Go and PHP;
  **cURL import** straight into an editable request.
- **Cookie manager** with persistence between sessions.
- **History** search, method/status filters, pinning, replay and copy as cURL.
- **Collections**: drag-and-drop of requests between collections/folders,
  favourites, recents, per-collection settings (variables/auth/description) and
  generated Markdown documentation.
- **GraphQL** schema introspection with a types/fields explorer and query
  validation; **WebSocket** ping, message filtering and clearing.
- Import Postman/OpenAPI/Swagger/HAR from a file **or URL**.

### Look and feel
- **Source Code Pro** is now the bundled program font (SIL OFL 1.1) for the UI
  and the code editors — the app renders identically offline.
- Accent colours, editor font and size, compact mode and layout presets
  (default / minimal / API testing / JSON focus / wide response).

## Compatibility

- Databases from 1.1.x are migrated automatically; existing collections,
  environments, tabs and history are preserved.
- Settings gain new fields (accent, editor font, layout preset, AI
  configuration) with production defaults.
- Verified with `go vet ./...`, `go test ./...` (10 packages), `tsc --noEmit`
  and `vite build`; the packaged binary is built with
  `go build -tags desktop,production -ldflags "-w -s"`.

## Known limitations

- Response timing reports the total duration only; the Go transport does not
  expose DNS/TCP/TLS phase timings, so none are invented.
- Credentials are stored in the local SQLite settings file rather than an OS
  keyring; the UI masks values and warns before exporting collections that
  contain secrets.
- OpenAPI documents are imported into browsable requests + generated docs, but
  there is no separate endpoints tree browser yet.
- Request chaining is provided through variable extraction and scripts; there
  is no drag-and-drop workflow builder yet.
