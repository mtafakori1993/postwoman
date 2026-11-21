// Central keyboard-shortcut registry.
//
// Every shortcut is declared once here with an id, a human label, a default
// combo and a group. The App component matches events against the resolved
// map (defaults merged with user overrides) and the Settings panel renders the
// same registry, which keeps documentation and behaviour in sync.

export interface ShortcutDef {
  id: string
  label: string
  defaultCombo: string
  group: string
}

export const SHORTCUTS: ShortcutDef[] = [
  { id: "send", label: "Send request", defaultCombo: "Ctrl+Enter", group: "Requests" },
  { id: "save", label: "Save request", defaultCombo: "Ctrl+S", group: "Requests" },
  { id: "focusUrl", label: "Focus URL bar", defaultCombo: "Ctrl+L", group: "Requests" },
  { id: "copyRequest", label: "Copy request as cURL", defaultCombo: "Ctrl+Shift+C", group: "Requests" },
  { id: "codeExport", label: "Generate code snippet", defaultCombo: "Ctrl+Shift+E", group: "Requests" },
  { id: "newRequest", label: "New request tab", defaultCombo: "Ctrl+N", group: "Tabs" },
  { id: "newTab", label: "New tab", defaultCombo: "Ctrl+T", group: "Tabs" },
  { id: "closeTab", label: "Close tab", defaultCombo: "Ctrl+W", group: "Tabs" },
  { id: "closeAllTabs", label: "Close all tabs", defaultCombo: "Ctrl+Shift+W", group: "Tabs" },
  { id: "restoreTab", label: "Reopen closed tab", defaultCombo: "Ctrl+Shift+T", group: "Tabs" },
  { id: "nextTab", label: "Next tab", defaultCombo: "Ctrl+Tab", group: "Tabs" },
  { id: "prevTab", label: "Previous tab", defaultCombo: "Ctrl+Shift+Tab", group: "Tabs" },
  { id: "palette", label: "Command palette", defaultCombo: "Ctrl+K", group: "Navigation" },
  { id: "toggleSidebar", label: "Toggle sidebar", defaultCombo: "Ctrl+B", group: "Navigation" },
  { id: "importCurl", label: "Import cURL", defaultCombo: "Ctrl+Shift+I", group: "Navigation" },
  { id: "shortcuts", label: "Show shortcuts", defaultCombo: "Ctrl+/", group: "Navigation" },
]

export type ShortcutOverrides = Record<string, string>

/** Merge defaults with user overrides (an empty override means "unbound"). */
export function resolveShortcuts(overrides: ShortcutOverrides = {}): Record<string, string> {
  const out: Record<string, string> = {}
  for (const def of SHORTCUTS) {
    const override = overrides[def.id]
    out[def.id] = override === undefined ? def.defaultCombo : override
  }
  return out
}

export function normalizeKey(key: string): string {
  if (key === " ") return "Space"
  if (key === "Esc") return "Escape"
  if (key.length === 1) return key.toUpperCase()
  return key
}

/** Minimal shape shared by DOM and React keyboard events. */
export interface KeyLike {
  key: string
  ctrlKey: boolean
  metaKey: boolean
  altKey: boolean
  shiftKey: boolean
}

/** Canonical combo string for a keyboard event, e.g. "Ctrl+Shift+K". */
export function comboFromEvent(event: KeyLike): string {
  const parts: string[] = []
  if (event.ctrlKey || event.metaKey) parts.push("Ctrl")
  if (event.altKey) parts.push("Alt")
  if (event.shiftKey) parts.push("Shift")
  parts.push(normalizeKey(event.key))
  return parts.join("+")
}

export function comboMatches(event: KeyLike, combo: string): boolean {
  if (!combo) return false
  const parts = combo.split("+")
  const key = parts[parts.length - 1]
  const needsCtrl = parts.includes("Ctrl")
  const needsAlt = parts.includes("Alt")
  const needsShift = parts.includes("Shift")
  return (
    (event.ctrlKey || event.metaKey) === needsCtrl &&
    event.altKey === needsAlt &&
    event.shiftKey === needsShift &&
    normalizeKey(event.key) === key
  )
}

/** Human-friendly combo with a platform-appropriate primary modifier. */
export function formatCombo(combo: string): string {
  if (!combo) return "—"
  const isMac = typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || "")
  return combo
    .replace(/\bCtrl\b/g, isMac ? "⌘" : "Ctrl")
    .replace(/\bAlt\b/g, isMac ? "⌥" : "Alt")
    .replace(/\bShift\b/g, isMac ? "⇧" : "Shift")
}

/** Find shortcut ids that share the same combo (user overrides included). */
export function findShortcutConflicts(resolved: Record<string, string>): Record<string, string[]> {
  const byCombo = new Map<string, string[]>()
  for (const [id, combo] of Object.entries(resolved)) {
    if (!combo) continue
    byCombo.set(combo, [...(byCombo.get(combo) ?? []), id])
  }
  const conflicts: Record<string, string[]> = {}
  for (const [combo, ids] of byCombo) {
    if (ids.length > 1) {
      for (const id of ids) conflicts[id] = ids.filter((other) => other !== id)
    }
  }
  return conflicts
}
