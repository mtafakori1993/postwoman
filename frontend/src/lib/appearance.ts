// Theme accent colours and editor font stacks.
//
// Accents are expressed as HSL triples so they can be dropped straight into
// the CSS custom properties the Tailwind theme reads (`hsl(var(--primary))`).

export const ACCENTS: Record<string, string> = {
  orange: "24.6 95% 53.1%",
  blue: "217 91% 60%",
  violet: "262 83% 58%",
  green: "142 71% 45%",
  rose: "346 77% 50%",
  slate: "215 25% 45%",
}

export const ACCENT_LABELS: Record<string, string> = {
  orange: "Orange",
  blue: "Blue",
  violet: "Violet",
  green: "Green",
  rose: "Rose",
  slate: "Slate",
}

export const EDITOR_FONTS: Record<string, string> = {
  mono: "\"Source Code Pro\", ui-monospace, \"JetBrains Mono\", \"Fira Code\", monospace",
  system: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
  sans: "\"Source Code Pro\", Inter, system-ui, sans-serif",
}

export const EDITOR_FONT_LABELS: Record<string, string> = {
  mono: "Source Code Pro",
  system: "System monospace",
  sans: "Sans-serif",
}

/**
 * Interface typefaces. The default is the operating system's own UI font so
 * text keeps its natural weight and proportions; Source Code Pro remains
 * available for users who prefer a monospace interface.
 */
export const DEFAULT_UI_FONT = "system"

export const UI_FONTS: Record<string, string> = {
  system: "system-ui, -apple-system, \"Segoe UI\", Roboto, \"Helvetica Neue\", Arial, sans-serif",
  "source-code-pro": "\"Source Code Pro\", ui-monospace, \"JetBrains Mono\", \"Fira Code\", monospace",
}

export const UI_FONT_LABELS: Record<string, string> = {
  system: "System UI (default)",
  "source-code-pro": "Source Code Pro (monospace)",
}

export const UI_FONT_SIZES = { min: 11, max: 20, default: 15 }
export const EDITOR_FONT_SIZES = { min: 9, max: 24, default: 13 }

export function clampFontSize(value: number | undefined, bounds: { min: number; max: number; default: number }): number {
  const size = Number.isFinite(value) ? Number(value) : bounds.default
  return Math.min(bounds.max, Math.max(bounds.min, Math.round(size)))
}

export function applyAccent(root: HTMLElement, accent: string) {
  const hsl = ACCENTS[accent] ?? ACCENTS.orange
  root.style.setProperty("--primary", hsl)
  root.style.setProperty("--ring", hsl)
  root.style.setProperty("--pw-orange", hsl)
  root.style.setProperty("--sidebar-primary", hsl)
  root.style.setProperty("--sidebar-ring", hsl)
  root.dataset.accent = accent
}

export function applyEditorFont(root: HTMLElement, family: string) {
  root.style.setProperty("--pw-editor-font", EDITOR_FONTS[family] ?? EDITOR_FONTS.mono)
}

/** Apply the interface typeface (the platform system sans by default). */
export function applyUiFont(root: HTMLElement, font: string) {
  root.style.setProperty("--pw-ui-font", UI_FONTS[font] ?? UI_FONTS[DEFAULT_UI_FONT])
  root.dataset.uiFont = font
}

/** Apply the code-editor font size through a CSS variable (live updates). */
export function applyEditorFontSize(root: HTMLElement, size: number) {
  root.style.setProperty("--pw-editor-font-size", `${clampFontSize(size, EDITOR_FONT_SIZES)}px`)
}

/** Apply the app-wide text scale; Tailwind's rem sizes follow the root size. */
export function applyUiFontSize(root: HTMLElement, size: number) {
  root.style.fontSize = `${clampFontSize(size, UI_FONT_SIZES)}px`
}
