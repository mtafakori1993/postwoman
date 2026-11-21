import React, { useMemo } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { AlertTriangle } from "lucide-react"
import { SHORTCUTS, findShortcutConflicts, formatCombo, resolveShortcuts } from "@/lib/shortcuts"
import { useWorkspaceStore } from "@/stores/workspace"

interface ShortcutsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ShortcutsDialog({ open, onOpenChange }: ShortcutsDialogProps) {
  const overrides = useWorkspaceStore((s) => s.settings.keyboardShortcuts || {})
  const resolved = useMemo(() => resolveShortcuts(overrides), [overrides])
  const conflicts = useMemo(() => findShortcutConflicts(resolved), [resolved])
  const groups = useMemo(() => {
    const map = new Map<string, typeof SHORTCUTS>()
    for (const shortcut of SHORTCUTS) {
      map.set(shortcut.group, [...(map.get(shortcut.group) ?? []), shortcut])
    }
    return [...map.entries()]
  }, [])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Keyboard shortcuts</DialogTitle>
          <DialogDescription>Customise these in Settings → Keyboard shortcuts.</DialogDescription>
        </DialogHeader>
        <div className="max-h-[420px] space-y-4 overflow-y-auto">
          {groups.map(([group, items]) => (
            <div key={group} className="space-y-1">
              <div className="pw-text-2xs font-semibold uppercase tracking-wide text-muted-foreground">{group}</div>
              {items.map((shortcut) => {
                const combo = resolved[shortcut.id]
                return (
                  <div key={shortcut.id} className="flex items-center justify-between gap-4 py-0.5 text-sm">
                    <span className="text-muted-foreground">{shortcut.label}</span>
                    <span className="flex items-center gap-1">
                      {conflicts[shortcut.id] && (
                        <span title="Conflicts with another shortcut">
                          <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
                        </span>
                      )}
                      <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono pw-text-3xs">{formatCombo(combo) || "unbound"}</kbd>
                    </span>
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
