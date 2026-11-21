import React, { useEffect, useMemo, useRef, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import {
  FileText,
  Folder,
  Globe,
  History,
  Keyboard,
  LayoutTemplate,
  Moon,
  PanelLeft,
  Play,
  Plus,
  Search as SearchIcon,
  Settings as SettingsIcon,
  Sparkles,
  Sun,
  Terminal,
  Trash2,
} from "lucide-react"
import { useWorkspaceStore } from "@/stores/workspace"

interface CommandPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onNewRequest: () => void
  onImport: () => void
  onImportCurl: () => void
  onOpenRunner: () => void
  onOpenShortcuts: () => void
  onOpenAI: () => void
}

interface PaletteItem {
  id: string
  label: string
  hint?: string
  group: string
  icon: React.ElementType
  run: () => void
}

export function CommandPalette({ open, onOpenChange, onNewRequest, onImport, onImportCurl, onOpenRunner, onOpenShortcuts, onOpenAI }: CommandPaletteProps) {
  const [query, setQuery] = useState("")
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const store = useWorkspaceStore()
  const {
    collections,
    requests,
    environments,
    history,
    workspace,
    settings,
  } = store

  useEffect(() => {
    if (open) {
      setQuery("")
      setActiveIndex(0)
      setTimeout(() => inputRef.current?.focus(), 30)
    }
  }, [open])

  const close = () => onOpenChange(false)

  const items = useMemo<PaletteItem[]>(() => {
    const list: PaletteItem[] = []

    const action = (id: string, label: string, group: string, icon: React.ElementType, run: () => void, hint?: string) =>
      list.push({ id, label, group, icon, run, hint })

    action("new-request", "New request", "Actions", Plus, () => { close(); onNewRequest() })
    action("import", "Import collection / OpenAPI / HAR", "Actions", FileText, () => { close(); onImport() })
    action("import-curl", "Import cURL", "Actions", Terminal, () => { close(); onImportCurl() })
    action("run-collection", "Run collection", "Actions", Play, () => { close(); onOpenRunner() })
    action("shortcuts", "Keyboard shortcuts", "Actions", Keyboard, () => { close(); onOpenShortcuts() })
    action("ai", "AI assistant", "Actions", Sparkles, () => { close(); onOpenAI() })
    action("clear-history", "Clear history", "Actions", Trash2, () => { close(); void store.clearHistory() })
    action("toggle-theme", `Switch to ${settings.theme === "dark" ? "light" : "dark"} theme`, "Actions", settings.theme === "dark" ? Sun : Moon, () => {
      close()
      store.updateSettings({ theme: settings.theme === "dark" ? "light" : "dark" })
    })
    action("toggle-sidebar", "Toggle sidebar", "Actions", PanelLeft, () => { close(); store.toggleSidebar() })
    for (const preset of ["default", "minimal", "testing", "json", "wide"]) {
      action(`layout-${preset}`, `Layout: ${preset}`, "Layout", LayoutTemplate, () => { close(); store.applyLayoutPreset(preset) })
    }
    for (const tab of ["collections", "environments", "history", "mocks", "websocket", "cookies", "settings"] as const) {
      action(`sidebar-${tab}`, `Open ${tab} panel`, "Panels", SettingsIcon, () => {
        close()
        if (!store.sidebarOpen) store.toggleSidebar()
        store.setActiveSidebarTab(tab)
      })
    }

    for (const request of requests) {
      const collection = collections.find((c) => c.id === request.collectionId)
      action(`request-${request.id}`, request.name || "Untitled Request", "Requests", FileText, () => { close(); void store.openRequest(request.id) }, `${request.method} ${request.url}`.slice(0, 90) + (collection ? ` · ${collection.info.name}` : ""))
    }
    for (const collection of collections) {
      action(`collection-${collection.id}`, collection.info.name, "Collections", Folder, () => {
        close()
        if (!store.sidebarOpen) store.toggleSidebar()
        store.setActiveSidebarTab("collections")
      }, collection.info.description || "")
    }
    for (const environment of environments) {
      action(`env-${environment.id}`, `Environment: ${environment.name}`, "Environments", Globe, () => { close(); void store.setActiveEnvironment(environment.id) }, store.activeEnvironmentId === environment.id ? "active" : `workspace: ${workspace?.name ?? ""}`)
    }
    action("env-none", "Environment: none", "Environments", Globe, () => { close(); void store.setActiveEnvironment(null) })
    for (const entry of history.slice(0, 40)) {
      action(
        `history-${entry.id}`,
        `${entry.request?.method} ${entry.request?.url}`.slice(0, 80),
        "History",
        History,
        () => { close(); store.openHistoryEntry(entry) },
        new Date(entry.timestamp).toLocaleString(),
      )
    }
    return list
  }, [collections, requests, environments, history, settings.theme, workspace?.name, store])

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase()
    if (!needle) return items.slice(0, 60)
    return items.filter((item) => `${item.label} ${item.hint ?? ""} ${item.group}`.toLowerCase().includes(needle)).slice(0, 80)
  }, [items, query])

  useEffect(() => {
    setActiveIndex(0)
  }, [query])

  const grouped = useMemo(() => {
    const map = new Map<string, PaletteItem[]>()
    for (const item of filtered) {
      map.set(item.group, [...(map.get(item.group) ?? []), item])
    }
    return [...map.entries()]
  }, [filtered])

  const flat = filtered

  useEffect(() => {
    if (filtered.length === 0) setActiveIndex(0)
    else setActiveIndex((i) => Math.min(i, filtered.length - 1))
  }, [filtered.length])

  const choose = (item: PaletteItem) => item.run()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl gap-0 overflow-hidden p-0 top-[15%] translate-y-0">
        <DialogHeader className="sr-only">
          <DialogTitle>Command palette</DialogTitle>
        </DialogHeader>
        <div className="flex items-center gap-2 border-b border-border px-3">
          <SearchIcon className="h-4 w-4 text-muted-foreground" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "ArrowDown") {
                e.preventDefault()
                setActiveIndex((i) => Math.min(i + 1, flat.length - 1))
              } else if (e.key === "ArrowUp") {
                e.preventDefault()
                setActiveIndex((i) => Math.max(i - 1, 0))
              } else if (e.key === "Enter" && flat[activeIndex]) {
                e.preventDefault()
                choose(flat[activeIndex])
              }
            }}
            placeholder="Search requests, collections, environments, actions…"
            className="h-11 border-0 focus-visible:ring-0"
          />
        </div>
        <div className="max-h-[420px] overflow-y-auto py-1">
          {flat.length === 0 && <div className="px-4 py-6 text-center text-sm text-muted-foreground">No matches</div>}
          {grouped.map(([group, groupItems]) => (
            <div key={group}>
              <div className="px-3 pb-0.5 pt-2 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">{group}</div>
              {groupItems.map((item) => {
                const index = flat.indexOf(item)
                const Icon = item.icon
                return (
                  <button
                    key={item.id}
                    className={cn("flex w-full items-center gap-3 px-3 py-1.5 text-left", index === activeIndex ? "bg-muted" : "hover:bg-muted/60")}
                    onMouseEnter={() => setActiveIndex(index)}
                    onClick={() => choose(item)}
                  >
                    <Icon className="h-4 w-4 shrink-0 text-muted-foreground" />
                    <span className="min-w-0 flex-1 truncate text-sm">{item.label}</span>
                    {item.hint && <span className="max-w-[45%] shrink-0 truncate font-mono text-[10px] text-muted-foreground">{item.hint}</span>}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}
