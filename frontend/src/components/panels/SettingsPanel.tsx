import React, { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { AlertTriangle, Check, Keyboard, Plus, RotateCcw, Sparkles } from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import { ACCENTS, ACCENT_LABELS, EDITOR_FONTS, EDITOR_FONT_LABELS } from "@/lib/appearance"
import { SHORTCUTS, comboFromEvent, findShortcutConflicts, formatCombo, resolveShortcuts } from "@/lib/shortcuts"

const LAYOUT_PRESETS = ["default", "minimal", "testing", "json", "wide"]

function ShortcutRecorder() {
  const overrides = useWorkspaceStore((s) => s.settings.keyboardShortcuts || {})
  const updateSettings = useWorkspaceStore((s) => s.updateSettings)
  const [recording, setRecording] = useState<string | null>(null)
  const resolved = useMemo(() => resolveShortcuts(overrides), [overrides])
  const conflicts = useMemo(() => findShortcutConflicts(resolved), [resolved])

  const groups = useMemo(() => {
    const map = new Map<string, typeof SHORTCUTS>()
    for (const shortcut of SHORTCUTS) map.set(shortcut.group, [...(map.get(shortcut.group) ?? []), shortcut])
    return [...map.entries()]
  }, [])

  const setCombo = (id: string, combo: string) => {
    updateSettings({ keyboardShortcuts: { ...overrides, [id]: combo } })
    setRecording(null)
  }

  return (
    <div className="space-y-3">
      {recording && (
        <div
          tabIndex={-1}
          autoFocus
          onKeyDown={(event) => {
            event.preventDefault()
            event.stopPropagation()
            if (event.key === "Escape") {
              setRecording(null)
              return
            }
            const combo = comboFromEvent(event)
            if (!combo || combo === "Ctrl" || combo === "Shift" || combo === "Alt") return
            setCombo(recording, combo)
          }}
          className="rounded-md border border-primary/40 bg-primary/5 px-3 py-2 text-xs"
        >
          Recording <span className="font-semibold">{SHORTCUTS.find((s) => s.id === recording)?.label}</span> — press a key combination, or Escape to cancel.
        </div>
      )}
      {groups.map(([group, items]) => (
        <div key={group} className="space-y-0.5">
          <div className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">{group}</div>
          {items.map((shortcut) => (
            <div key={shortcut.id} className="flex items-center justify-between gap-2 text-xs">
              <span className="text-muted-foreground">{shortcut.label}</span>
              <span className="flex items-center gap-1.5">
                {conflicts[shortcut.id] && (
                  <span title="Conflicts with another shortcut">
                    <AlertTriangle className="h-3.5 w-3.5 text-amber-500" />
                  </span>
                )}
                <button
                  className={cn(
                    "rounded border px-1.5 py-0.5 font-mono text-[10px] hover:bg-muted",
                    recording === shortcut.id ? "border-primary text-primary" : "border-border",
                  )}
                  onClick={() => setRecording(shortcut.id)}
                  title="Click to rebind"
                >
                  {formatCombo(resolved[shortcut.id]) || "unbound"}
                </button>
                {overrides[shortcut.id] !== undefined && (
                  <button
                    className="text-[10px] text-muted-foreground underline"
                    onClick={() => {
                      const next = { ...overrides }
                      delete next[shortcut.id]
                      updateSettings({ keyboardShortcuts: next })
                    }}
                  >
                    reset
                  </button>
                )}
              </span>
            </div>
          ))}
        </div>
      ))}
    </div>
  )
}

/** Settings panel: appearance, layout, requests, scripts, workspaces, shortcuts, AI. */
export function SettingsPanel() {
  const {
    settings,
    updateSettings,
    resetSettings,
    workspaces,
    workspace,
    createWorkspace,
    setActiveWorkspace,
    applyLayoutPreset,
    tabPreferences,
    setTabPreferences,
  } = useWorkspaceStore()

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 border-b border-border px-2 py-1.5">
        <span className="flex-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">Settings</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={resetSettings} title="Reset to defaults" aria-label="Reset settings">
          <RotateCcw className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 space-y-4 overflow-y-auto p-3 text-sm">
        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Appearance</Label>
          <div className="flex items-center justify-between gap-2">
            <span>Theme</span>
            <Select value={settings.theme} onValueChange={(v) => updateSettings({ theme: v })}>
              <SelectTrigger className="h-8 w-[130px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="light">Light</SelectItem>
                <SelectItem value="dark">Dark</SelectItem>
                <SelectItem value="system">System</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between gap-2">
            <span>Accent colour</span>
            <div className="flex items-center gap-1.5">
              {Object.keys(ACCENTS).map((accent) => (
                <button
                  key={accent}
                  aria-label={ACCENT_LABELS[accent]}
                  title={ACCENT_LABELS[accent]}
                  onClick={() => updateSettings({ accentColor: accent })}
                  className={cn(
                    "h-5 w-5 rounded-full border transition-transform hover:scale-110",
                    settings.accentColor === accent && "ring-2 ring-ring ring-offset-1 ring-offset-background",
                  )}
                  style={{ backgroundColor: `hsl(${ACCENTS[accent]})` }}
                />
              ))}
            </div>
          </div>
          <div className="flex items-center justify-between gap-2">
            <span>Editor font</span>
            <Select value={settings.editorFontFamily || "mono"} onValueChange={(v) => updateSettings({ editorFontFamily: v })}>
              <SelectTrigger className="h-8 w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.keys(EDITOR_FONTS).map((font) => (
                  <SelectItem key={font} value={font}>{EDITOR_FONT_LABELS[font]}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center justify-between gap-2">
            <span>Editor font size</span>
            <Input
              type="number"
              value={settings.editorFontSize}
              onChange={(e) => updateSettings({ editorFontSize: parseInt(e.target.value) || 13 })}
              className="h-8 w-[90px]"
            />
          </div>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Compact mode</span>
            <Checkbox checked={!!settings.compactMode} onCheckedChange={(c) => updateSettings({ compactMode: c === true })} />
          </label>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Layout</Label>
          <div className="flex flex-wrap gap-1.5">
            {LAYOUT_PRESETS.map((preset) => (
              <Button
                key={preset}
                variant={settings.layoutPreset === preset ? "default" : "outline"}
                size="sm"
                className="h-7 text-[11px] capitalize"
                onClick={() => applyLayoutPreset(preset)}
              >
                {preset}
              </Button>
            ))}
          </div>
          <p className="text-[11px] text-muted-foreground">Presets set the sidebar width, response height and active panel.</p>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Tabs</Label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Confirm before closing unsaved tabs</span>
            <Checkbox checked={tabPreferences.confirmCloseDirtyTabs} onCheckedChange={(c) => setTabPreferences({ confirmCloseDirtyTabs: c === true })} />
          </label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Middle-click closes tabs</span>
            <Checkbox checked={tabPreferences.closeTabOnMiddleClick} onCheckedChange={(c) => setTabPreferences({ closeTabOnMiddleClick: c === true })} />
          </label>
          <div className="flex items-center justify-between gap-2">
            <span>Close button</span>
            <Select value={tabPreferences.tabCloseButton} onValueChange={(v) => setTabPreferences({ tabCloseButton: v as "hover" | "always" })}>
              <SelectTrigger className="h-8 w-[130px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="hover">On hover</SelectItem>
                <SelectItem value="always">Always visible</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Requests</Label>
          <div className="flex items-center justify-between gap-2">
            <span>Timeout (ms)</span>
            <Input
              type="number"
              value={settings.requestTimeout}
              onChange={(e) => updateSettings({ requestTimeout: parseInt(e.target.value) || 30000 })}
              className="h-8 w-[110px]"
            />
          </div>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Verify TLS certificates</span>
            <Checkbox checked={settings.sslVerification} onCheckedChange={(c) => updateSettings({ sslVerification: c === true })} />
          </label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Follow redirects by default</span>
            <Checkbox checked={settings.followRedirects} onCheckedChange={(c) => updateSettings({ followRedirects: c === true })} />
          </label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Send no-cache header</span>
            <Checkbox checked={settings.sendNoCacheHeader} onCheckedChange={(c) => updateSettings({ sendNoCacheHeader: c === true })} />
          </label>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Scripts &amp; data</Label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Enable pre-request &amp; test scripts</span>
            <Checkbox checked={settings.scriptExecutionEnabled} onCheckedChange={(c) => updateSettings({ scriptExecutionEnabled: c === true })} />
          </label>
          <div className="flex items-center justify-between gap-2">
            <span>History retention (days)</span>
            <Input
              type="number"
              value={settings.historyRetentionDays}
              onChange={(e) => updateSettings({ historyRetentionDays: parseInt(e.target.value) || 30 })}
              className="h-8 w-[90px]"
            />
          </div>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="flex items-center gap-1.5 text-xs font-semibold uppercase text-muted-foreground">
            <Sparkles className="h-3.5 w-3.5" /> AI assistant (optional)
          </Label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Enable AI assistant</span>
            <Checkbox checked={!!settings.aiEnabled} onCheckedChange={(c) => updateSettings({ aiEnabled: c === true })} />
          </label>
          <div className="flex items-center justify-between gap-2">
            <span>Provider</span>
            <Select value={settings.aiProvider || "openai"} onValueChange={(v) => updateSettings({ aiProvider: v })}>
              <SelectTrigger className="h-8 w-[150px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="openai">OpenAI-compatible</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">Base URL</span>
            <Input
              value={settings.aiBaseUrl || ""}
              onChange={(e) => updateSettings({ aiBaseUrl: e.target.value })}
              placeholder="https://api.openai.com/v1"
              className="mt-1 h-8 font-mono text-xs"
            />
          </div>
          <div>
            <span className="text-xs text-muted-foreground">Model</span>
            <Input
              value={settings.aiModel || ""}
              onChange={(e) => updateSettings({ aiModel: e.target.value })}
              placeholder="gpt-4o-mini"
              className="mt-1 h-8 font-mono text-xs"
            />
          </div>
          <div>
            <span className="text-xs text-muted-foreground">API key</span>
            <Input
              type="password"
              value={settings.aiApiKey || ""}
              onChange={(e) => updateSettings({ aiApiKey: e.target.value })}
              placeholder="sk-…"
              className="mt-1 h-8 font-mono text-xs"
            />
          </div>
          <p className="text-[11px] text-muted-foreground">
            Data is only sent when you explicitly run a prompt from the AI assistant. Keys are stored locally with your app settings.
          </p>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="flex items-center gap-1.5 text-xs font-semibold uppercase text-muted-foreground">
            <Keyboard className="h-3.5 w-3.5" /> Keyboard shortcuts
          </Label>
          <ShortcutRecorder />
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Workspaces</Label>
          {workspaces.map((ws) => (
            <div
              key={ws.id}
              className={cn("flex cursor-pointer items-center gap-2 rounded px-2 py-1 hover:bg-muted/50", workspace?.id === ws.id && "bg-muted")}
              onClick={() => setActiveWorkspace(ws.id)}
            >
              {workspace?.id === ws.id ? <Check className="h-3.5 w-3.5 text-primary" /> : <span className="w-3.5" />}
              <span className="flex-1 truncate text-sm">{ws.name}</span>
            </div>
          ))}
          <Button
            variant="outline"
            size="sm"
            className="w-full gap-1"
            onClick={() => {
              const name = window.prompt?.("Workspace name")
              if (name) createWorkspace(name)
            }}
          >
            <Plus className="h-3.5 w-3.5" /> New workspace
          </Button>
        </section>
      </div>
    </div>
  )
}
