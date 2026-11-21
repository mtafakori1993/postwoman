import React from "react"
import { Button } from "@/components/ui/button"
import { Trash2, Clock, RotateCcw } from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"

function formatTimestamp(ts: number): string {
  if (!ts) return "—"
  const d = new Date(ts)
  return d.toLocaleString()
}

export function HistoryPanel() {
  const { history, openHistoryEntry, deleteHistoryEntry, clearHistory } = useWorkspaceStore()

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">
          History {history.length > 0 && `(${history.length})`}
        </span>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 text-xs"
          disabled={history.length === 0}
          onClick={clearHistory}
        >
          Clear
        </Button>
      </div>

      <div className="flex-1 overflow-y-auto py-1">
        {history.length === 0 && (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">No request history yet.</div>
        )}
        {history.map((entry) => (
          <div
            key={entry.id}
            className="group flex items-start gap-2 px-2 py-1.5 hover:bg-muted/50 cursor-pointer border-b border-border/40 last:border-0"
            onClick={() => openHistoryEntry(entry)}
            title="Open a copy of this request"
          >
            <span
              className={cn(
                "font-mono text-[10px] font-bold w-9 shrink-0 mt-0.5",
                entry.request?.method === "GET" ? "text-sky-500" : "text-green-600",
              )}
            >
              {entry.request?.method}
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-sm truncate">{entry.request?.name || entry.request?.url}</div>
              <div className="text-[11px] text-muted-foreground truncate font-mono">{entry.request?.url}</div>
              <div className="flex items-center gap-2 text-[10px] text-muted-foreground mt-0.5">
                <Clock className="h-3 w-3" />
                <span>{formatTimestamp(entry.timestamp)}</span>
                {entry.response && (
                  <span
                    className={cn(
                      "font-mono",
                      entry.response.status >= 200 && entry.response.status < 300
                        ? "text-green-600 dark:text-green-400"
                        : "text-red-500",
                    )}
                  >
                    {entry.response.status}
                  </span>
                )}
                <span>{entry.duration} ms</span>
              </div>
            </div>
            <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100">
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 p-0"
                title="Reopen"
                aria-label="Reopen request"
                onClick={(e) => {
                  e.stopPropagation()
                  openHistoryEntry(entry)
                }}
              >
                <RotateCcw className="h-3.5 w-3.5" />
              </Button>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 p-0"
                title="Delete entry"
                aria-label="Delete history entry"
                onClick={(e) => {
                  e.stopPropagation()
                  deleteHistoryEntry(entry.id)
                }}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
