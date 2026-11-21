import React, { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Clock, Copy, Pin, PinOff, Play, RotateCcw, Search, Trash2 } from "lucide-react"
import { cn } from "@/lib/utils"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"

const PINNED_KEY = "postwoman.pinnedHistory"

function loadPinned(): string[] {
  try {
    const raw = localStorage.getItem(PINNED_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.filter((v) => typeof v === "string") : []
  } catch {
    return []
  }
}

function formatTimestamp(ts: number): string {
  if (!ts) return "—"
  return new Date(ts).toLocaleString()
}

function statusClass(status?: number): string {
  if (!status) return "text-muted-foreground"
  if (status < 300) return "text-green-600 dark:text-green-400"
  if (status < 400) return "text-yellow-600 dark:text-yellow-500"
  return "text-red-600 dark:text-red-400"
}

export function HistoryPanel() {
  const { history, openHistoryEntry, deleteHistoryEntry, clearHistory, sendActiveRequest } = useWorkspaceStore()
  const [query, setQuery] = useState("")
  const [method, setMethod] = useState("all")
  const [status, setStatus] = useState("all")
  const [pinned, setPinned] = useState<string[]>(loadPinned)

  const togglePin = (id: string) => {
    setPinned((current) => {
      const next = current.includes(id) ? current.filter((v) => v !== id) : [...current, id]
      try {
        localStorage.setItem(PINNED_KEY, JSON.stringify(next))
      } catch {
        /* best effort */
      }
      return next
    })
  }

  const entries = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return history
      .filter((entry) => {
        if (method !== "all" && (entry.request?.method || "").toUpperCase() !== method) return false
        const code = entry.response?.status ?? 0
        if (status === "success" && !(code >= 200 && code < 300)) return false
        if (status === "redirect" && !(code >= 300 && code < 400)) return false
        if (status === "client" && !(code >= 400 && code < 500)) return false
        if (status === "server" && !(code >= 500)) return false
        if (status === "error" && entry.response?.error === undefined) return false
        if (!needle) return true
        return `${entry.request?.method} ${entry.request?.url} ${entry.request?.name}`.toLowerCase().includes(needle)
      })
      .sort((a, b) => {
        const ap = pinned.includes(a.id) ? 1 : 0
        const bp = pinned.includes(b.id) ? 1 : 0
        if (ap !== bp) return bp - ap
        return b.timestamp - a.timestamp
      })
  }, [history, query, method, status, pinned])

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 border-b border-border px-2 py-1.5">
        <span className="flex-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          History {history.length > 0 && `(${history.length})`}
        </span>
        <Button variant="ghost" size="sm" className="h-7 text-xs" disabled={history.length === 0} onClick={clearHistory}>
          Clear
        </Button>
      </div>

      <div className="space-y-1.5 border-b border-border px-2 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Search history…" className="h-7 pl-7 text-xs" />
        </div>
        <div className="flex gap-1.5">
          <Select value={method} onValueChange={setMethod}>
            <SelectTrigger className="h-7 flex-1 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All methods</SelectItem>
              {["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"].map((m) => (
                <SelectItem key={m} value={m}>{m}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger className="h-7 flex-1 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Any status</SelectItem>
              <SelectItem value="success">2xx</SelectItem>
              <SelectItem value="redirect">3xx</SelectItem>
              <SelectItem value="client">4xx</SelectItem>
              <SelectItem value="server">5xx</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {history.length === 0 && <div className="px-3 py-6 text-center text-xs text-muted-foreground">No request history yet.</div>}
        {history.length > 0 && entries.length === 0 && (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">No entries match the current filters.</div>
        )}
        {entries.map((entry) => (
          <div
            key={entry.id}
            className="group flex items-start gap-2 border-b border-border/40 px-2 py-1.5 last:border-0 hover:bg-muted/50"
          >
            <span className={cn("mt-0.5 w-9 shrink-0 font-mono pw-text-3xs font-bold", entry.request?.method === "GET" ? "text-sky-500" : "text-green-600")}>
              {entry.request?.method}
            </span>
            <button className="min-w-0 flex-1 text-left" onClick={() => openHistoryEntry(entry)} title={entry.request?.url}>
              <div className="truncate text-sm">{entry.request?.name || entry.request?.url}</div>
              <div className="truncate font-mono pw-text-2xs text-muted-foreground">{entry.request?.url}</div>
              <div className="mt-0.5 flex items-center gap-2 pw-text-3xs text-muted-foreground">
                <Clock className="h-3 w-3" />
                <span>{formatTimestamp(entry.timestamp)}</span>
                {entry.response && <span className={cn("font-mono", statusClass(entry.response.status))}>{entry.response.status}</span>}
                <span>{entry.duration} ms</span>
                {pinned.includes(entry.id) && <Pin className="h-3 w-3 text-primary" />}
              </div>
            </button>
            <div className="flex flex-col gap-0.5 opacity-0 group-hover:opacity-100">
              <Button variant="ghost" size="icon" className="h-6 w-6 p-0" title="Toggle pin" aria-label="Toggle pin" onClick={() => togglePin(entry.id)}>
                {pinned.includes(entry.id) ? <PinOff className="h-3.5 w-3.5" /> : <Pin className="h-3.5 w-3.5" />}
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 p-0"
                title="Replay request"
                aria-label="Replay request"
                onClick={async () => {
                  openHistoryEntry(entry)
                  setTimeout(() => void sendActiveRequest(), 60)
                }}
              >
                <Play className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 p-0"
                title="Copy as cURL"
                aria-label="Copy as cURL"
                onClick={async () => {
                  const curl = await api.exportRequestCurl({ ...entry.request, id: "", collectionId: "" })
                  await api.copyToClipboard(curl)
                }}
              >
                <Copy className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6 p-0" title="Reopen" aria-label="Reopen" onClick={() => openHistoryEntry(entry)}>
                <RotateCcw className="h-3.5 w-3.5" />
              </Button>
              <Button variant="ghost" size="icon" className="h-6 w-6 p-0 hover:text-destructive" title="Delete" aria-label="Delete" onClick={() => deleteHistoryEntry(entry.id)}>
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
