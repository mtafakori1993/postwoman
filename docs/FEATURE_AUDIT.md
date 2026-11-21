# Postwoman Feature Audit

> Initial audit performed before changes. Statuses are updated at the end of the
> project in `docs/FINAL_AUDIT_REPORT.md`.

## 1. Project state at audit time

| Item | Finding |
| --- | --- |
| Stack | Wails v2.17 + Go 1.27 backend, React 19 + TypeScript + Vite + Tailwind + Radix UI frontend, SQLite (goose migrations) |
| Git | The directory is **not** a git repository. No history, no uncommitted state to preserve. |
| Backend packages | `internal/database`, `internal/httpclient`, `internal/scripting`, `internal/collections` |
| Frontend | 10 components, one zustand store persisted to **localStorage**, no router |
| Tests | **None** (backend or frontend) |
| Build | `wails build` succeeds; AppImage had not been produced (only a raw binary in `AppDir/usr/bin`) |
| Runtime | Application launches; SQLite database created at `~/.local/share/Postwoman/postwoman.db` |
| Integration | **Broken.** Frontend declared `window.__POSTWOMAN__` but never wired it to Wails. A hand-written fake bridge in `src/wailsjs/go/runtime.js` calls `window.external.Invoke` (a Wails v1 API that does not exist in v2). Every backend call failed with "Wails runtime not available". |

## 2. Feature matrix (initial)

Legend: COMPLETE / PARTIAL / BROKEN / MISSING / UNTESTED / BLOCKED

### Application shell

| Feature | Expected | Current implementation | Status | Problems | Required changes | Test strategy | Deps |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Startup | Window opens, DB migrated, state loaded | Wails opens; DB migrated | PARTIAL | `ensureDir` is a no-op; no default workspace; frontend never loads backend state | Fix dir creation, bootstrap default workspace, hydrate UI | launch, check DB, inspect UI | database |
| Main window | 1024x768+ usable layout | 1024x768 | PARTIAL | Sidebar is `fixed` **and** content gets a margin equal to its width → double offset; response panel duplicates resizer | Flex layout | visual at 3 resolutions | React |
| Sidebar | Collections tree, envs, history, settings | 7 tabs; only collections renders, 6 say "coming soon" | BROKEN | Placeholder panels, dead buttons, no resize persistence | Implement panels, remove dead tabs | interaction tests | store |
| Tabs | Create/close/switch/reorder/persist | Local state only, not persisted; close button nested in `<button>` | PARTIAL | Not persisted; invalid nesting; hidden-tab logic drops last tab | Backend persistence, fix markup | UI test + restart | backend |
| Workspace selection | Multiple workspaces | Hard-coded "Default Workspace", no-op select | MISSING | — | Workspace CRUD | UI + DB | repository |
| Layout persistence | Sidebar width, panel height | In zustand/localStorage | PARTIAL | Persisted only in browser storage, not DB | Move to settings | restart test | settings |
| Keyboard shortcuts | Ctrl+S/T/W/B/K/E/Enter | None implemented | MISSING | — | Global handler | manual/UI | — |
| Menus/dialogs | Functional menus | User menu items are no-ops; notification dot is fake | BROKEN | Fake controls | Implement or remove | click every item | — |
| Settings | Theme, timeout, SSL, proxy, etc. | Store defaults only, no UI | MISSING | — | Settings dialog + persistence | restart test | settings |

### HTTP client

| Feature | Expected | Current | Status | Problems |
| --- | --- | --- | --- | --- |
| All methods | GET…TRACE | Accepted by Go client | UNTESTED | Frontend never reaches backend |
| URL parsing | scheme-less, spaces, unicode | `url.Parse` only | PARTIAL | No scheme default; no invalid URL UX |
| Query params | enable/disable, duplicates | `map[string]string` loses duplicates and `enabled` | BROKEN | Disabled params are still sent |
| Path variables | `:id` substitution | Type exists only | MISSING | Not implemented |
| Headers | enable/disable, duplicates | `map[string]string` loses duplicates/enabled | BROKEN | Disabled headers sent; duplicate headers collapsed |
| Cookies | send/receive, editor | Jar only; response cookies from jar (wrong URL) | PARTIAL | Cookies tab gets jar cookies for target, not Set-Cookie |
| Bodies | json/xml/text/js/html/formdata/urlencoded/binary/graphql | Frontend sends only `body.raw` | BROKEN | All other body modes ignored |
| Auth | basic/bearer/apikey/digest/oauth1/oauth2/jwt/awsv4 | Basic/bearer/apikey/oauth2 token/jwt only; digest sends bogus header; oauth1 realm only; awsv4 nothing | BROKEN | Fake implementations |
| Auth inheritance | request → folder → collection | None | MISSING | — |
| Redirects | follow toggle, max, method | Custom loop that loses body and leaks responses | BROKEN | Body replay broken |
| TLS | verification toggle, client certs, CA | Toggle only; cert TODO | PARTIAL | No cert/CA loading |
| Proxy | http/https/socks5 + bypass | http/https only | PARTIAL | SOCKS5 missing |
| Timeout | per request | Only when settings != nil | PARTIAL | — |
| Cancellation | cancel in-flight | None | MISSING | — |
| File uploads | multipart file parts | `BuildFormData` unused, content only | BROKEN | No disk reads |
| Downloads | save response | Buttons are no-ops | MISSING | — |
| Error handling | structured errors | Returns `Response{Error}` with status 0 | PARTIAL | Frontend treats it as a normal response |

### Request editor

| Feature | Status | Problems |
| --- | --- | --- |
| URL/method | PARTIAL | Works locally; not persisted |
| Params/headers | BROKEN | Duplicate local state + `enabled` lost; dynamic Tailwind `w-[${width}]` classes never generated |
| Auth | PARTIAL | Fake "Get New Access Token" alert; NTLM/Hawk presented as working |
| Body | PARTIAL | Only raw text; binary file never uploaded; GET body blocked (Postman allows) |
| Scripts | PARTIAL | Plain textarea; scripts never executed |
| GraphQL | PARTIAL | Stored but not sent as GraphQL envelope |
| Saving | BROKEN | Only creates; never updates; no collection selection; uses fake bridge |

### Response viewer

| Feature | Status | Problems |
| --- | --- | --- |
| JSON | PARTIAL | Custom tree has `setState` during render (React warning/loop); no virtualization |
| XML/HTML/Text | PARTIAL | Only pretty JSON or raw text |
| Binary | MISSING | Body assumed UTF-8 |
| Raw/pretty | PARTIAL | No format selection persistence |
| Headers | BROKEN | Expects `Header[]`; Go returns `http.Header` object → `headers.map` crash |
| Cookies | BROKEN | Go `http.Cookie` JSON ≠ `Cookie` type (sameSite etc.) |
| Status/timing/size | PARTIAL | Duration is ns number → rendered as "0.00 s" |
| Search | PARTIAL | Filters nothing in raw mode |
| Copy/download/save | BROKEN | Copy works; download/save are no-ops |
| Tests | MISSING | Placeholder text |

### Collections / folders / requests

| Feature | Status | Problems |
| --- | --- | --- |
| Collection CRUD | BROKEN | Frontend-only localStorage; backend exists but unused; no UI dialogs |
| Folders | BROKEN | "Add Request/Add Folder" prompts do nothing; delete does nothing |
| Nested folders | BROKEN | Import creates all folders at root; export drops nested requests |
| Request CRUD | BROKEN | Sidebar selection does nothing; duplicate/delete no-ops; no save-to-collection |
| Drag & drop | MISSING | — |
| Context menus | BROKEN | Dead menu items |
| Persistence | BROKEN | localStorage only; SQLite unused |
| Export | BROKEN | Backend export drops folder-contained requests |
| Import | BROKEN | Panics on collections without `item`; no nesting; no params; no preview; no formats |

### Environments / variables

| Feature | Status | Problems |
| --- | --- | --- |
| Environment management | MISSING | No UI; "Manage Environments…" no-op |
| Variable management | MISSING | — |
| Scopes | PARTIAL | Types only |
| Resolution | BROKEN | `ResolveVariables` naive string replace; never called by request path |
| Secrets | MISSING | Stored plaintext, no masking |
| Dynamic variables | PARTIAL | Generated only inside script sandbox, not in requests |
| Switching | PARTIAL | Select updates store; request ignores it |
| Persistence | BROKEN | localStorage only |

### Import / export

| Format | Status | Notes |
| --- | --- | --- |
| Postman 2.0/2.1 | BROKEN | Partial, panics, loses params/folders/scripts/auth details |
| Postman Environment | MISSING | — |
| OpenAPI 3.0/3.1 | MISSING | — |
| Swagger 2.0 | MISSING | — |
| cURL | MISSING | — |
| HAR | MISSING | — |
| GraphQL | MISSING | — |
| AsyncAPI | MISSING | — |
| Preview/conflict/errors | MISSING | — |
| Export round-trip | BROKEN | No UI; lossy |

### Auth, scripting, runner

| Feature | Status | Problems |
| --- | --- | --- |
| Pre-request scripts | BROKEN | Stored, never run |
| Post-response scripts | BROKEN | Stored, never run |
| Assertions | BROKEN | `pm.test` logs only, never returns results; `pm.expect` stub |
| Script timeout | PARTIAL | `time.After` returns but leaves runaway goroutine alive |
| Sandbox | PARTIAL | goja is safe, but `pm.sendRequest` is fake |
| Collection/folder scripts | MISSING | Imported but never executed |
| Runner | MISSING | Types only |
| Data files | MISSING | — |
| Reports | MISSING | — |

### Protocols / other

| Feature | Status | Problems |
| --- | --- | --- |
| REST | PARTIAL | See HTTP |
| GraphQL | PARTIAL | Body mode only, not transmitted |
| WebSocket | MISSING | Types + DB table only |
| gRPC | MISSING | Types + DB table only |
| MQTT | MISSING | Types + DB table only |
| Socket.IO | MISSING | Not in requirements list beyond "if included" |
| History | BROKEN | Backend records only if requests reached backend (never); UI tab placeholder |
| Global search | MISSING | `console.log` |
| Mock servers | MISSING | DB table only |
| API docs / specs | MISSING | DB table only |
| Monitors | MISSING | DB table only |
| Notifications | BROKEN | Emits a frontend event only; no UI listener |
| Logging | PARTIAL | zerolog to stderr |
| Secret storage | MISSING | Plaintext |
| AppImage | MISSING | Only a raw binary was copied into AppDir |

## 3. Cross-cutting technical debt

1. **Fake bridge.** Hand-written `src/wailsjs/go/runtime.js` shadows the real Wails runtime and is loaded from `index.html`. Must be deleted and real generated bindings used.
2. **Two sources of truth.** SQLite backend vs localStorage store. The app must use the backend.
3. **Schema drift.** DB columns store untyped JSON; typed model + normalization needed.
4. **`ensureDir` no-op** → fresh installs fail if the parent directory is missing.
5. **Migration/DB correctness.** No indexes beyond history; no default workspace row.
6. **Duplicate UI.** Sidebar `fixed` + content `marginLeft`; response panel has two resizers.
7. **Dynamic Tailwind classes** (`w-[${col.width}]`) are never emitted by JIT.
8. **Invalid HTML** (button inside button in tabs).
9. **No tests, no CI.**
10. **No request cancellation / no context propagation.**
11. **Secrets logged/stored plaintext**; scripts unrestricted HTTP but that is intended; no OS access.
12. **XSS risk** in response HTML preview: `sandbox="allow-scripts allow-same-origin"` on `srcDoc` allows same-origin access to app DOM. Must drop `allow-same-origin`.
13. **No virtualization** for large JSON/collections.
14. **Hard-coded values** (workspace, environment "manage" no-op, app version, platform).

## 4. Priorities

See `docs/IMPLEMENTATION_PLAN.md`.
