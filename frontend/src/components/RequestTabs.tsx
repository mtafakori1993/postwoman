import React, { useRef } from "react"
import { Button } from "@/components/ui/button"
import { Plus, X, ChevronLeft, ChevronRight } from "lucide-react"
import { cn } from "@/lib/utils"
import { TabState } from "@/types"

interface RequestTabsProps {
  tabs: TabState[]
  activeTabId: string | null
  onTabChange: (id: string) => void
  onTabClose: (id: string) => void
  onNewTab: () => void
  onTabReorder: (tabs: TabState[]) => void
  onCloseOthers?: (id: string) => void
}

export function RequestTabs({
  tabs,
  activeTabId,
  onTabChange,
  onTabClose,
  onNewTab,
  onTabReorder,
}: RequestTabsProps) {
  const scrollRef = useRef<HTMLDivElement>(null)

  const scroll = (delta: number) => scrollRef.current?.scrollBy({ left: delta, behavior: "smooth" })

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
      <Button variant="ghost" size="icon" className="h-10 w-7 shrink-0" onClick={() => scroll(-200)} aria-label="Scroll tabs left">
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <div ref={scrollRef} className="flex-1 overflow-x-auto flex items-center gap-1 px-1">
        {tabs.map((tab) => (
          <div
            key={tab.id}
            role="tab"
            aria-selected={activeTabId === tab.id}
            draggable
            onDragStart={(e) => handleDragStart(e, tab.id)}
            onDragOver={(e) => e.preventDefault()}
            onDrop={(e) => handleDrop(e, tab.id)}
            onClick={() => onTabChange(tab.id)}
            className={cn(
              "group relative flex items-center gap-2 px-3 py-1.5 text-sm font-medium min-w-[150px] max-w-[240px] cursor-pointer rounded-t-md border-b-2",
              activeTabId === tab.id
                ? "text-foreground bg-background border-primary"
                : "text-foreground/70 hover:text-foreground hover:bg-muted/50 border-transparent",
            )}
          >
            {tab.request && (
              <span className={cn("font-mono text-[10px] font-bold shrink-0", methodColor(tab.request.method))}>
                {tab.request.method}
              </span>
            )}
            <span className="truncate flex-1" title={tab.title}>
              {tab.title}
              {tab.isDirty && <span className="ml-1 text-orange-500">●</span>}
            </span>
            <button
              className="h-5 w-5 p-0 opacity-0 group-hover:opacity-100 hover:bg-muted rounded shrink-0 flex items-center justify-center"
              onClick={(e) => {
                e.stopPropagation()
                onTabClose(tab.id)
              }}
              aria-label="Close tab"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        ))}

        <Button variant="ghost" size="icon" className="h-8 w-8 p-0 shrink-0" onClick={onNewTab} aria-label="New tab">
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <Button variant="ghost" size="icon" className="h-10 w-7 shrink-0" onClick={() => scroll(200)} aria-label="Scroll tabs right">
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
