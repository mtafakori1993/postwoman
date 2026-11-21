import { useEffect, useState } from "react"
import { useWorkspaceStore } from "@/stores/workspace"

/** Resolve the effective theme (including the `system` setting) and keep it
 * in sync with OS changes. Used by the CodeMirror editor, which needs a
 * concrete theme extension rather than CSS variables. */
export function useIsDarkMode(): boolean {
  const theme = useWorkspaceStore((s) => s.settings.theme)
  const [systemDark, setSystemDark] = useState(() => {
    if (typeof window === "undefined" || !window.matchMedia) return false
    return window.matchMedia("(prefers-color-scheme: dark)").matches
  })

  useEffect(() => {
    if (typeof window === "undefined" || !window.matchMedia) return
    const query = window.matchMedia("(prefers-color-scheme: dark)")
    const onChange = (event: MediaQueryListEvent) => setSystemDark(event.matches)
    setSystemDark(query.matches)
    query.addEventListener?.("change", onChange)
    return () => query.removeEventListener?.("change", onChange)
  }, [])

  if (theme === "dark") return true
  if (theme === "light") return false
  return systemDark
}
