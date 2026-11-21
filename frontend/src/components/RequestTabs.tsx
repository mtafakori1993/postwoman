import React, { useEffect, useRef } from "react"
import { Button } from "@/components/ui/button"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"
import { ChevronLeft, ChevronRight, Copy, Layers, Plus, X, XCircle } from "lucide-react"
import { cn } from "@/lib/utils"
import { TabState, TabPreferences } from "@/types"

interface RequestTabsProps {
  tabs: TabState[]
  activeTabId: string | null
  onTabChange: (id: string) => void
  onTabClose: (id: string) => void
  onNewTab: () => void
  onTabReorder: (tabs: TabState[]) => void
  onTabCloseOthers: (id: string) => void
  onTabCloseToRight: (id: string) => void
  onTabCloseAll: () => void
  onTabDuplicate: (id: string) => void
  preferences: TabPreferences
}

export function RequestTabs({
  tabs,
  activeTabId,
  onTabChange,
  onTabClose,
  onNewTab,
  onTabReorder,
  onTabCloseOthers,
  onTabCloseToRight,
  onTabCloseAll,
  onTabDuplicate,
  preferences,
}: RequestTabsProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  const scroll = (delta: number) => scrollRef.current?.scrollBy({ left: delta, behavior: "smooth" })

  useEffect(() => {
    const active = scrollRef.current?.querySelector<HTMLElement>('[data-tab-active="true"]')
    active?.scrollIntoView({ block: "nearest", inline: "nearest" })
  }, [activeTabId, tabs.length])

  const handleDragStart = (e: React.DragEvent, tabId: string) => {
    e.dataTransfer.setData("text/plain", tabId)
    e.dataTransfer.effectAllowed = "move"
  }

  const handleDrop = (e: React.DragEvent, targetTabId: string) => {
    e.preventDefault()
    const sourceTabId = e.dataTransfer.getData("text/plain")
    if (!sourceTabId || sourceTabId === targetTabId) return
    const next = [...tabs]
    const from = next.findIndex((t) => t.id === sourceTabId)
    const to = next.findIndex((t) => t.id === targetTabId)
    if (from === -1 || to === -1) return
    const [moved] = next.splice(from, 1)
    next.splice(to, 0, moved)
    onTabReorder(next)
  }

  return (
    <div className="flex h-10 bg-muted/40 border-b border-border shrink-0">
      <Button
        variant="ghost"
        size="icon"
        className="h-10 w-7 shrink-0 rounded-none"
        onClick={() => scroll(-200)}
        aria-label="Scroll tabs left"
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <div
        ref={scrollRef}
        role="tablist"
        aria-label="Request tabs"
        className="flex-1 overflow-x-auto flex items-center gap-1 px-1"
      >
        {tabs.map((tab, index) => {
          const isActive = activeTabId === tab.id
          const closeButtonVisible = preferences.tabCloseButton === "always" || isActive
          return (
            <ContextMenu key={tab.id}>
              <ContextMenuTrigger asChild>
                <div
                  role="tab"
                  aria-selected={isActive}
                  data-tab-active={isActive ? "true" : "false"}
                  data-tab-id={tab.id}
                  tabIndex={isActive ? 0 : -1}
                  draggable
                  onDragStart={(e) => handleDragStart(e, tab.id)}
                  onDragOver={(e) => e.preventDefault()}
                  onDrop={(e) => handleDrop(e, tab.id)}
                  onClick={() => onTabChange(tab.id)}
                  onAuxClick={(e) => {
                    if (e.button === 1 && preferences.closeTabOnMiddleClick) {
                      e.preventDefault()
                      onTabClose(tab.id)
                    }
                  }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault()
                      onTabChange(tab.id)
                      return
                    }
                    if (e.key === "ArrowRight" || e.key === "ArrowLeft" || e.key === "Home" || e.key === "End") {
                      e.preventDefault()
                      let nextIndex = index
                      if (e.key === "ArrowRight") nextIndex = (index + 1) % tabs.length
                      else if (e.key === "ArrowLeft") nextIndex = (index - 1 + tabs.length) % tabs.length
                      else if (e.key === "Home") nextIndex = 0
                      else if (e.key === "End") nextIndex = tabs.length - 1
                      const nextTab = tabs[nextIndex]
                      if (!nextTab) return
                      onTabChange(nextTab.id)
                      requestAnimationFrame(() => {
                        scrollRef.current
                          ?.querySelector<HTMLElement>(`[data-tab-id="${nextTab.id}"]`)
                          ?.focus()
                      })
                    }
                  }}
                  title={tab.title}
                  className={cn(
                    "group relative flex items-center gap-2 px-3 py-1.5 text-sm font-medium min-w-[150px] max-w-[240px] cursor-pointer rounded-t-md border-b-2 transition-colors outline-none",
                    "focus-visible:ring-1 focus-visible:ring-ring",
                    isActive
                      ? "text-foreground bg-background border-primary shadow-[inset_0_1px_0_hsl(var(--border))]"
                      : "text-foreground/70 hover:text-foreground hover:bg-muted/60 border-transparent",
                  )}
                >
                  {tab.request && (
                    <span className={cn("font-mono text-[10px] font-bold shrink-0", methodColor(tab.request.method))}>
                      {tab.request.method}
                    </span>
                  )}
                  <span className="truncate flex-1" title={tab.title}>
                    {tab.title}
                    {tab.isDirty && (
                      <span
                        className="ml-1.5 inline-block h-1.5 w-1.5 rounded-full bg-orange-500 align-middle"
                        aria-label="Unsaved changes"
                        title="Unsaved changes"
                      />
                    )}
                  </span>
                  <button
                    className={cn(
                      "h-5 w-5 p-0 hover:bg-muted rounded shrink-0 flex items-center justify-center transition-opacity",
                      closeButtonVisible
                        ? "opacity-100"
                        : "opacity-0 group-hover:opacity-100 group-focus-within:opacity-100",
                    )}
                    onClick={(e) => {
                      e.stopPropagation()
                      onTabClose(tab.id)
                    }}
                    aria-label={`Close ${tab.title}`}
                    tabIndex={-1}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              </ContextMenuTrigger>

              <ContextMenuContent className="w-56">
                <ContextMenuItem onSelect={() => onTabClose(tab.id)}>
                  <X className="h-4 w-4" />
                  Close Tab
                  <ContextMenuShortcut>Ctrl+W</ContextMenuShortcut>
                </ContextMenuItem>
                <ContextMenuItem
                  onSelect={() => onTabCloseOthers(tab.id)}
                  disabled={tabs.length <= 1}
                >
                  <XCircle className="h-4 w-4" />
                  Close Other Tabs
                </ContextMenuItem>
                <ContextMenuItem
                  onSelect={() => onTabCloseToRight(tab.id)}
                  disabled={index === tabs.length - 1}
                >
                  <ChevronRight className="h-4 w-4" />
                  Close Tabs to the Right
                </ContextMenuItem>
                <ContextMenuItem onSelect={onTabCloseAll} disabled={tabs.length === 0}>
                  <Layers className="h-4 w-4" />
                  Close All Tabs
                </ContextMenuItem>
                <ContextMenuSeparator />
                <ContextMenuItem onSelect={() => onTabDuplicate(tab.id)}>
                  <Copy className="h-4 w-4" />
                  Duplicate Tab
                </ContextMenuItem>
              </ContextMenuContent>
            </ContextMenu>
          )
        })}

        <Button variant="ghost" size="icon" className="h-8 w-8 p-0 shrink-0" onClick={onNewTab} aria-label="New tab">
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <Button
        variant="ghost"
        size="icon"
        className="h-10 w-7 shrink-0 rounded-none"
        onClick={() => scroll(200)}
        aria-label="Scroll tabs right"
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  )
}

function methodColor(method: string): string {
  switch (method.toUpperCase()) {
    case "GET":
      return "text-sky-500"
    case "POST":
      return "text-green-600"
    case "PUT":
    case "PATCH":
      return "text-amber-500"
    case "DELETE":
      return "text-red-500"
    default:
      return "text-muted-foreground"
  }
}
