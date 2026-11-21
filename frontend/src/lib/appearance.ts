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
