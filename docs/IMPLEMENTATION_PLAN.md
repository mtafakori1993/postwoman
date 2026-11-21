# Postwoman Implementation Plan

> Status: implemented. See `docs/FINAL_AUDIT_REPORT.md` for verification results.
> Items marked ✅ are done; ⛔ are intentionally out of scope and are not
> surfaced as fake UI.

Ordered by user impact. Each item lists the concrete change and the verification
used before it is considered done.

## P0 — Reliability & integration

1. ✅ **Wire the frontend to the real Wails backend**
   - Removed the hand-written fake bridge, stopped loading it from `index.html`.
   - Added `src/lib/api.ts` typed wrapper around `wailsjs/go/main/App`.
   - Verified: root-package Go tests call the same bound methods; packaged app
     renders live data.
2. ✅ **Fix database bootstrapping**
   - Implemented `ensureDir`; default workspace; migrations; indexes.
   - Verified: fresh temp databases in tests; app migrates to goose version 3.
3. ✅ **Single source of truth**
   - zustand hydrated from `Bootstrap()`; mutations go to SQLite; tabs and
     settings persisted in the backend.
   - Verified: persistence tests + `TestPersistenceAcrossReopen`.
4. ✅ **Layout fixes**
   - Flex sidebar (no fixed + margin), single response resizer, min sizes.
   - Verified: screenshots of the running app.

## P1 — Core API client

5. ✅ **Request engine rewrite** — typed `model.Request`; scoped + dynamic
   variables; all body modes; multipart files; duplicate headers/params;
   per-request timeout; redirects; proxy incl. SOCKS5; TLS certs/CA; response
   cap; structured errors. Verified by `internal/httpclient` tests.
6. ✅ **Authentication** — Basic, Bearer, API key, OAuth2, JWT, OAuth1 HMAC,
   AWS SigV4, Digest; inheritance request → folder → collection. ⛔ NTLM and
   Hawk signing (clear error, with an in-editor notice).
7. ✅ **Request editor** — controlled form, duplicate-safe editors, body modes,
   GraphQL envelope, scripts, auth, settings, save/update.
8. ✅ **Response viewer** — header/cookie mapping, status/timing/size, JSON/
   raw/preview, tests tab, console tab, copy/save.

## P2 — Data management

9. ✅ **Collections tree** — create/rename/delete/duplicate, nested folders,
   requests, context menus, open-in-tab, persisted.
10. ✅ **Environments & variables** — manager panel, activation, globals,
    persisted.
11. ✅ **History** — recorded on send; open/delete/clear.
12. ✅ **Import/export** — Postman 2.0/2.1, Postman env, OpenAPI 3.0/3.1,
    Swagger 2.0, cURL, HAR, GraphQL, AsyncAPI; nested folders preserved;
    round-trip verified.
13. ✅ **Search** — backend search palette.
14. ✅ **Settings** — theme, timeout, TLS verification, redirects, script
    execution, retention; persisted.

## P3 — Automation

15. ✅ **Scripting** — `pm.test`/`pm.expect`, response/request objects, variables,
    `pm.sendRequest`, console capture, interrupt timeout, results in the UI;
    collection/folder scripts executed.
16. ✅ **Collection runner** — iterations, delay, JSON/CSV data files,
    environment, progress events, pass/fail summary.

## P4 — Advanced

17. ✅ **WebSocket** — real connect/send/receive via gorilla/websocket.
18. ✅ **Mock servers** — real local HTTP servers, start/stop, request log.
19. ⛔ **Monitors** — table only; not surfaced.
20. ⛔ **gRPC / MQTT / Socket.IO** — not implemented; no fake UI.

## P5 — Packaging & verification

21. ✅ Tests (`go test ./...`, `-race`), frontend typecheck + build.
22. ✅ AppImage via `appimagetool` with desktop file, icon and `AppRun`.
23. ✅ Launched packaged app under X, screenshots, verified DB persistence.
24. ✅ Final report `docs/FINAL_AUDIT_REPORT.md`.

## Explicit non-goals / accepted limitations

- Cloud sync, team collaboration, accounts.
- gRPC, MQTT, Socket.IO (no fake UI is shown).
- NTLM and Hawk request signing (clear error; shown as "not supported" in the
  auth editor).
- Full OpenAPI code generation; import converts operations to requests and
  reports unsupported constructs as warnings.
