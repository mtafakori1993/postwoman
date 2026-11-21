# Postwoman Implementation Plan

Ordered by user impact. Each item lists the concrete change and the verification
used before it is considered done.

## P0 — Reliability & integration

1. **Wire the frontend to the real Wails backend**
   - Remove the hand-written fake bridge (`src/wailsjs/go/runtime.js`), stop loading it from `index.html`.
   - Add `src/lib/api.ts` typed wrapper around `wailsjs/go/main/App` with a browser-dev fallback.
   - Verify: request round-trip in the packaged app; no `Wails runtime not available` errors.
2. **Fix database bootstrapping**
   - Implement `ensureDir`; create the default workspace; run migrations; add indexes.
   - Verify: delete `~/.local/share/Postwoman/postwoman.db`, launch, confirm schema + workspace.
3. **Single source of truth**
   - Hydrate zustand from `Bootstrap()`; every mutation goes to SQLite; layout/settings persisted in `app_settings`.
   - Verify: create data, restart, data present.
4. **Layout fixes**
   - Flex sidebar (no fixed + margin), single response resizer, min sizes, no clipping at 1280×720.
   - Verify: screenshots at 1280×720, 1366×768, 1920×1080.

## P1 — Core API client

5. **Request engine rewrite**
   - Typed `model.Request`; variable resolution (env/collection/global/local/dynamic/path); all body modes; multipart files; duplicate headers/params; per-request timeout/cancel; redirects; proxy incl. SOCKS5; TLS certs/CA; response size cap; binary-safe responses; structured errors.
   - Verify: Go tests against `httptest` servers for each body mode, auth header, redirects, timeout, cancellation.
6. **Authentication**
   - Real Basic, Bearer, API key, OAuth2 (token endpoint), JWT, OAuth1 HMAC, AWS SigV4, Digest challenge. Clear errors for unsupported NTLM/Hawk (hidden in UI). Request → collection inheritance.
   - Verify: header/signature unit tests.
7. **Request editor**
   - Controlled form, duplicate-safe key/value editors, body modes incl. file picker, GraphQL envelope, scripts editor, auth inheritance, settings, save/update/duplicate.
   - Verify: UI interaction + persistence tests.
8. **Response viewer**
   - Correct header/cookie mapping, status/timing/size, pretty JSON/XML/HTML/text/binary, search highlight, tests tab, copy/download/save.
   - Verify: local test server responses of each type.

## P2 — Data management

9. **Collections tree**
   - Create/rename/delete/duplicate collection, nested folders, requests; context menus; drag & drop; selection opens tab; persisted.
   - Verify: CRUD test + restart.
10. **Environments & variables**
    - Manager dialog, secret masking, active env switching, globals, resolution preview.
    - Verify: variable resolution test, restart.
11. **History**
    - Backend records every send; panel with open/delete/clear; retention.
12. **Import/export**
    - Formats: Postman 2.0/2.1, Postman environment, OpenAPI 3.0/3.1, Swagger 2.0, cURL, HAR, GraphQL, AsyncAPI.
    - Format detection, validation, preview, warnings for unsupported fields, nested folders preserved, round-trip export.
    - Verify: fixture-based Go tests + round-trip.
13. **Search**
    - Global search across collections, folders, requests, environments, history.
14. **Settings**
    - Theme, editor, timeout, SSL verification, proxy, history retention, auto-save, notifications; persisted.

## P3 — Automation

15. **Scripting**
    - Real `pm.test`/`pm.expect`, response/request objects, variables, cookies, `pm.sendRequest`, console capture, timeout via `vm.Interrupt`, results returned to UI.
    - Collection/folder scripts executed.
16. **Collection runner**
    - Iterations, delay, data file (JSON/CSV), environment, progress events, cancellation, pass/fail summary, exportable report.

## P4 — Advanced

17. **WebSocket** — real connect/send/receive via gorilla/websocket, persisted sessions.
18. **Mock servers** — real local HTTP servers with configurable routes, start/stop, request log.
19. **Monitors** — scheduled request execution with assertions (basic implementation).
20. **gRPC / MQTT / Socket.IO** — not feasible within scope without large new dependencies; **do not present fake UI**. Document as limitations.

## P5 — Packaging & verification

21. Unit/integration tests (`go test ./...`), frontend build/typecheck.
22. AppImage via `appimagetool`, desktop file, icon, AppRun.
23. Launch packaged app under X, screenshot, execute request against local test server, verify DB persistence after restart.
24. Final report `docs/FINAL_AUDIT_REPORT.md`.

## Explicit non-goals / accepted limitations

- Cloud sync, team collaboration, accounts.
- gRPC, MQTT, Socket.IO (no fake UI will be shown).
- NTLM and Hawk request signing (clear error, hidden from auth picker).
- Full OpenAPI code generation; import converts operations to requests and reports unsupported constructs.
