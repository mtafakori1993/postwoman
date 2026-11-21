import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Download, Pencil, Plus, RefreshCw, Trash2, Upload } from "lucide-react"
import { cn } from "@/lib/utils"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { Cookie, StoredCookie } from "@/types"

function emptyCookie(): StoredCookie {
  return { url: "", cookie: { name: "", value: "", path: "/", secure: false, httpOnly: false, sameSite: "lax" } }
}

function originOf(url: string): string {
  try {
    return new URL(url.includes("://") ? url : `https://${url}`).origin
  } catch {
    return url
  }
}

export function CookieManagerPanel() {
  const { cookies, refreshCookies, saveCookie, deleteCookie, clearCookies } = useWorkspaceStore()
  const [editing, setEditing] = useState<StoredCookie | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [filter, setFilter] = useState("")

  useEffect(() => {
    void refreshCookies()
  }, [refreshCookies])

  const filtered = useMemo(() => {
    const needle = filter.trim().toLowerCase()
    if (!needle) return cookies
    return cookies.filter((entry) => `${entry.cookie.name} ${entry.cookie.domain ?? ""} ${entry.url}`.toLowerCase().includes(needle))
  }, [cookies, filter])

  const grouped = useMemo(() => {
    const map = new Map<string, StoredCookie[]>()
    for (const entry of filtered) {
      const key = originOf(entry.url)
      map.set(key, [...(map.get(key) ?? []), entry])
    }
    return [...map.entries()].sort((a, b) => a[0].localeCompare(b[0]))
  }, [filtered])

  const save = async () => {
    if (!editing) return
    if (!editing.url.trim() || !editing.cookie.name.trim()) {
      setError("A URL and a cookie name are required")
      return
    }
    try {
      await saveCookie(editing)
      setEditing(null)
      setError(null)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const exportCookies = async () => {
    await api.saveTextFile("postwoman-cookies.json", JSON.stringify(cookies, null, 2)).catch((err) => setError(api.errorMessage(err)))
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 border-b border-border px-2 py-1.5">
        <span className="flex-1 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
          Cookies {cookies.length > 0 && `(${cookies.length})`}
        </span>
        <Button variant="ghost" size="icon" className="h-7 w-7" title="Refresh" aria-label="Refresh cookies" onClick={() => void refreshCookies()}>
          <RefreshCw className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-7 w-7" title="Add cookie" aria-label="Add cookie" onClick={() => setEditing(emptyCookie())}>
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <div className="border-b border-border px-2 py-1.5">
        <Input value={filter} onChange={(e) => setFilter(e.target.value)} placeholder="Filter cookies…" className="h-7 text-xs" />
      </div>

      {error && <div className="bg-destructive/10 px-3 py-1.5 text-xs text-destructive">{error}</div>}

      <div className="flex-1 overflow-y-auto">
        {cookies.length === 0 && (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">
            No cookies yet. Cookies set by servers you call appear here and persist between sessions.
          </div>
        )}
        {grouped.map(([origin, entries]) => (
          <div key={origin}>
            <div className="sticky top-0 bg-muted/60 px-2 py-1 pw-text-3xs font-semibold uppercase tracking-wide text-muted-foreground">{origin}</div>
            {entries.map((entry) => (
              <div key={`${entry.url}-${entry.cookie.name}-${entry.cookie.path ?? ""}`} className="group flex items-center gap-2 border-b border-border/40 px-2 py-1.5">
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm">{entry.cookie.name}</div>
                  <div className="truncate font-mono pw-text-2xs text-muted-foreground">{entry.cookie.value}</div>
                  <div className="flex gap-2 pw-text-3xs text-muted-foreground">
                    <span>{entry.cookie.path || "/"}</span>
                    {entry.cookie.secure && <span>secure</span>}
                    {entry.cookie.httpOnly && <span>httpOnly</span>}
                    {entry.cookie.sameSite && <span>{entry.cookie.sameSite}</span>}
                    {entry.cookie.expires ? <span>expires {new Date(entry.cookie.expires).toLocaleDateString()}</span> : <span>session</span>}
                  </div>
                </div>
                <Button variant="ghost" size="icon" className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100" aria-label="Edit cookie" onClick={() => setEditing(entry)}>
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100 hover:text-destructive"
                  aria-label="Delete cookie"
                  onClick={() => void deleteCookie(entry.url, entry.cookie.name)}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}
          </div>
        ))}
      </div>

      <div className="flex items-center gap-1 border-t border-border px-2 py-2">
        <Button variant="outline" size="sm" className="gap-1.5" onClick={exportCookies} disabled={cookies.length === 0}>
          <Download className="h-3.5 w-3.5" /> Export
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="gap-1.5"
          onClick={async () => {
            try {
              const path = await api.openFileDialog("Import cookies JSON", "*.json")
              if (!path) return
              const content = await api.readTextFile(path)
              const parsed = JSON.parse(content) as StoredCookie[]
              for (const entry of parsed) await saveCookie(entry)
            } catch (err) {
              setError(api.errorMessage(err))
            }
          }}
        >
          <Upload className="h-3.5 w-3.5" /> Import
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="ml-auto gap-1.5 text-destructive"
          disabled={cookies.length === 0}
          onClick={() => void clearCookies()}
        >
          <Trash2 className="h-3.5 w-3.5" /> Clear all
        </Button>
      </div>

      <Dialog open={editing !== null} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{editing?.cookie.name ? "Edit cookie" : "Add cookie"}</DialogTitle>
          </DialogHeader>
          {editing && (
            <div className="space-y-2">
              <div>
                <Label className="text-xs">URL</Label>
                <Input value={editing.url} onChange={(e) => setEditing({ ...editing, url: e.target.value })} placeholder="https://api.example.com" className="font-mono text-xs" />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <Label className="text-xs">Name</Label>
                  <Input value={editing.cookie.name} onChange={(e) => setEditing({ ...editing, cookie: { ...editing.cookie, name: e.target.value } })} />
                </div>
                <div>
                  <Label className="text-xs">Value</Label>
                  <Input value={editing.cookie.value} onChange={(e) => setEditing({ ...editing, cookie: { ...editing.cookie, value: e.target.value } })} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <Label className="text-xs">Domain</Label>
                  <Input value={editing.cookie.domain ?? ""} onChange={(e) => setEditing({ ...editing, cookie: { ...editing.cookie, domain: e.target.value } })} />
                </div>
                <div>
                  <Label className="text-xs">Path</Label>
                  <Input value={editing.cookie.path ?? ""} onChange={(e) => setEditing({ ...editing, cookie: { ...editing.cookie, path: e.target.value } })} />
                </div>
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-xs">
                  <Checkbox checked={editing.cookie.secure} onCheckedChange={(c) => setEditing({ ...editing, cookie: { ...editing.cookie, secure: c === true } })} /> Secure
                </label>
                <label className="flex items-center gap-2 text-xs">
                  <Checkbox checked={editing.cookie.httpOnly} onCheckedChange={(c) => setEditing({ ...editing, cookie: { ...editing.cookie, httpOnly: c === true } })} /> HttpOnly
                </label>
              </div>
            </div>
          )}
          {error && <div className={cn("rounded border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive")}>{error}</div>}
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditing(null)}>Cancel</Button>
            <Button onClick={save}>Save</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
