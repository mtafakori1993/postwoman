// Ambient declarations for the globals injected by the Wails v2 runtime.
//
// The generated bindings in `wailsjs/go/main/App.js` call `window.go.main.App.*`
// and `wailsjs/runtime/runtime.js` uses `window.runtime`. These only exist when
// the page is served by the Wails webview.

export {}

declare global {
  interface Window {
    go: {
      main: {
        App: Record<string, (...args: unknown[]) => Promise<unknown>>
      }
    }
    runtime: Record<string, (...args: unknown[]) => unknown>
  }
}
