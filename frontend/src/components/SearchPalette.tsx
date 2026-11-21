import React, { useEffect, useRef, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { FileText, Folder, Globe, History, Search as SearchIcon } from "lucide-react"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { SearchResult } from "@/types"

interface SearchPaletteProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const typeIcon: Record<string, React.ElementType> = {
  request: FileText,
  folder: Folder,
  collection: Folder,
  environment: Globe,
  history: History,
}

export function SearchPalette({ open, onOpenChange }: SearchPaletteProps) {
  const [query, setQuery] = useState("")
  const [results, setResults] = useState<SearchResult[]>([])
  const [activeIndex, setActiveIndex] = useState(0)
  const [loading, setLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)
  const { openRequest, setActiveSidebarTab, toggleSidebar, sidebarOpen } = useWorkspaceStore()

  useEffect(() => {
    if (open) {
      setQuery("")
      setResults([])
      setActiveIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [open])

  useEffect(() => {
    if (!open) return
    const trimmed = query.trim()
    if (!trimmed) {
      setResults([])
      return
    }
    let cancelled = false
    setLoading(true)
    const timer = setTimeout(async () => {
      try {
        const found = await api.search(trimmed)
        if (!cancelled) {
          setResults(found ?? [])
          setActiveIndex(0)
        }
      } catch (err) {
        if (!cancelled) setResults([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }, 180)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [query, open])

  const choose = (result: SearchResult) => {
    onOpenChange(false)
    if (result.type === "request" || result.type === "history") {
      openRequest(result.id)
    } else if (result.type === "collection" || result.type === "folder") {
      if (!sidebarOpen) toggleSidebar()
      setActiveSidebarTab("collections")
    } else if (result.type === "environment") {
      if (!sidebarOpen) toggleSidebar()
      setActiveSidebarTab("environments")
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl p-0 gap-0 overflow-hidden top-[20%] translate-y-0">
        <DialogHeader className="sr-only">
          <DialogTitle>Search</DialogTitle>
        </DialogHeader>
        <div className="flex items-center gap-2 px-3 border-b border-border">
          <SearchIcon className="h-4 w-4 text-muted-foreground" />
          <Input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "ArrowDown") {
                e.preventDefault()
                setActiveIndex((i) => Math.min(i + 1, results.length - 1))
              } else if (e.key === "ArrowUp") {
                e.preventDefault()
                setActiveIndex((i) => Math.max(i - 1, 0))
              } else if (e.key === "Enter" && results[activeIndex]) {
                choose(results[activeIndex])
              }
            }}
            placeholder="Search collections, requests, environments..."
            className="border-0 focus-visible:ring-0 h-11"
          />
        </div>
        <div className="max-h-[400px] overflow-y-auto py-1">
          {query.trim() && !loading && results.length === 0 && (
            <div className="px-4 py-6 text-center text-sm text-muted-foreground">No matches</div>
          )}
          {results.map((result, index) => {
            const Icon = typeIcon[result.type] || FileText
            return (
              <button
                key={`${result.type}-${result.id}`}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-muted/60",
                  index === activeIndex && "bg-muted",
                )}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => choose(result)}
              >
                <Icon className="h-4 w-4 text-muted-foreground shrink-0" />
                <div className="min-w-0 flex-1">
                  <div className="text-sm truncate">{result.name}</div>
                  <div className="text-[11px] text-muted-foreground truncate font-mono">{result.match || result.parent}</div>
                </div>
                <span className="text-[10px] uppercase text-muted-foreground shrink-0">{result.type}</span>
              </button>
            )
          })}
          {!query.trim() && (
            <div className="px-4 py-6 text-center text-xs text-muted-foreground">
              Type to search. Use ↑/↓ to navigate and Enter to open.
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
