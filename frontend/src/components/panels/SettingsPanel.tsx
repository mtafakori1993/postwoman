import React from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { RotateCcw, Plus, Check } from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"

export function SettingsPanel() {
  const { settings, updateSettings, resetSettings, workspaces, workspace, createWorkspace, setActiveWorkspace } =
    useWorkspaceStore()

  const shortcuts: { keys: string; action: string }[] = [
    { keys: "Ctrl+B", action: "Toggle sidebar" },
    { keys: "Ctrl+K", action: "Search" },
    { keys: "Ctrl+T", action: "New tab" },
    { keys: "Ctrl+S", action: "Save request" },
    { keys: "Ctrl+Enter", action: "Send request" },
    { keys: "Ctrl+W", action: "Close tab" },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">Settings</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={resetSettings} title="Reset to defaults" aria-label="Reset settings">
          <RotateCcw className="h-4 w-4" />
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-4 text-sm">
        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Appearance</Label>
          <div className="flex items-center justify-between gap-2">
            <span>Theme</span>
            <Select value={settings.theme} onValueChange={(v) => updateSettings({ theme: v })}>
              <SelectTrigger className="w-[130px] h-8">
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
            <span>Editor font size</span>
            <Input
              type="number"
              value={settings.editorFontSize}
              onChange={(e) => updateSettings({ editorFontSize: parseInt(e.target.value) || 13 })}
              className="w-[90px] h-8"
            />
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
              className="w-[110px] h-8"
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
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Scripts & data</Label>
          <label className="flex items-center justify-between gap-2 cursor-pointer">
            <span>Enable pre-request & test scripts</span>
            <Checkbox
              checked={settings.scriptExecutionEnabled}
              onCheckedChange={(c) => updateSettings({ scriptExecutionEnabled: c === true })}
            />
          </label>
          <div className="flex items-center justify-between gap-2">
            <span>History retention (days)</span>
            <Input
              type="number"
              value={settings.historyRetentionDays}
              onChange={(e) => updateSettings({ historyRetentionDays: parseInt(e.target.value) || 30 })}
              className="w-[90px] h-8"
            />
          </div>
        </section>

        <Separator />

        <section className="space-y-2">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Workspaces</Label>
          {workspaces.map((ws) => (
            <div
              key={ws.id}
              className={cn(
                "flex items-center gap-2 px-2 py-1 rounded cursor-pointer hover:bg-muted/50",
                workspace?.id === ws.id && "bg-muted",
              )}
              onClick={() => setActiveWorkspace(ws.id)}
            >
              {workspace?.id === ws.id ? <Check className="h-3.5 w-3.5 text-primary" /> : <span className="w-3.5" />}
              <span className="flex-1 text-sm truncate">{ws.name}</span>
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

        <Separator />

        <section className="space-y-1">
          <Label className="text-xs font-semibold uppercase text-muted-foreground">Keyboard shortcuts</Label>
          {shortcuts.map((s) => (
            <div key={s.keys} className="flex items-center justify-between text-xs py-0.5">
              <span className="text-muted-foreground">{s.action}</span>
              <kbd className="px-1.5 py-0.5 rounded border border-border bg-muted font-mono text-[10px]">{s.keys}</kbd>
            </div>
          ))}
        </section>
      </div>
    </div>
  )
}
