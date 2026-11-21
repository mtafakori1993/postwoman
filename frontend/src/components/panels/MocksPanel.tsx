import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { Plus, Trash2, Save, Play, Square, RefreshCw } from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import * as api from "@/lib/api"
import type { MockRequestLog, MockServer } from "@/types"

export function MocksPanel() {
  const { mockServers, setMockServers } = useWorkspaceStore()
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [draft, setDraft] = useState<MockServer | null>(null)
  const [dirty, setDirty] = useState(false)
  const [logs, setLogs] = useState<MockRequestLog[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!selectedId && mockServers.length > 0) setSelectedId(mockServers[0].id)
  }, [mockServers, selectedId])

  useEffect(() => {
    const server = mockServers.find((s) => s.id === selectedId) ?? null
    if (server) {
      setDraft({ ...server, routes: server.routes ?? [] })
      setDirty(false)
    } else {
      setDraft(null)
    }
  }, [mockServers, selectedId])

  const refresh = async () => {
    try {
      setMockServers(await api.listMockServers())
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const loadLogs = async (id: string) => {
    try {
      setLogs(await api.mockServerLogs(id))
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  useEffect(() => {
    if (draft?.id && draft.isRunning) loadLogs(draft.id)
  }, [draft?.id, draft?.isRunning])

  const createServer = async () => {
    setError(null)
    try {
      const created = await api.saveMockServer({
        id: "",
        name: "New Mock Server",
        port: 8787,
        routes: [],
        isRunning: false,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      await refresh()
      setSelectedId(created.id)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const saveDraft = async () => {
    if (!draft) return
    setError(null)
    try {
      const saved = await api.saveMockServer(draft)
      await refresh()
      setSelectedId(saved.id)
      setDirty(false)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const toggle = async (server: MockServer) => {
    setError(null)
    try {
      if (server.isRunning) await api.stopMockServer(server.id)
      else await api.startMockServer(server.id)
      await refresh()
      if (server.id === selectedId) loadLogs(server.id)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const remove = async (id: string) => {
    setError(null)
    try {
      await api.deleteMockServer(id)
      if (selectedId === id) setSelectedId(null)
      await refresh()
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">Mock Servers</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={refresh} title="Refresh" aria-label="Refresh mock servers">
          <RefreshCw className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={createServer} title="New mock server" aria-label="New mock server">
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {error && <div className="px-3 py-1.5 text-xs text-destructive bg-destructive/10">{error}</div>}

      <div className="max-h-36 overflow-y-auto border-b border-border py-1">
        {mockServers.length === 0 && (
          <div className="px-3 py-3 text-xs text-muted-foreground">No mock servers yet.</div>
        )}
        {mockServers.map((server) => (
          <div
            key={server.id}
            className={cn(
              "group flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-muted/50",
              selectedId === server.id && "bg-muted",
            )}
            onClick={() => setSelectedId(server.id)}
          >
            <Badge variant={server.isRunning ? "default" : "secondary"} className="pw-text-3xs px-1.5 py-0">
              {server.isRunning ? "live" : "off"}
            </Badge>
            <span className="flex-1 text-sm truncate">{server.name}</span>
            <span className="pw-text-2xs text-muted-foreground font-mono">:{server.port}</span>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 p-0"
              title={server.isRunning ? "Stop" : "Start"}
              aria-label={server.isRunning ? "Stop mock server" : "Start mock server"}
              onClick={(e) => {
                e.stopPropagation()
                toggle(server)
              }}
            >
              {server.isRunning ? <Square className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
              aria-label="Delete mock server"
              onClick={(e) => {
                e.stopPropagation()
                remove(server.id)
              }}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {draft ? (
          <>
            <div className="flex items-center gap-2">
              <Input
                value={draft.name}
                onChange={(e) => {
                  setDraft({ ...draft, name: e.target.value })
                  setDirty(true)
                }}
                className="h-8 text-sm"
                placeholder="Name"
              />
              <Input
                type="number"
                value={draft.port}
                onChange={(e) => {
                  setDraft({ ...draft, port: parseInt(e.target.value) || 0 })
                  setDirty(true)
                }}
                className="h-8 w-24 text-sm"
                placeholder="Port"
              />
              <Button size="sm" className="h-8 gap-1" disabled={!dirty} onClick={saveDraft}>
                <Save className="h-3.5 w-3.5" /> Save
              </Button>
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label className="text-xs font-medium text-muted-foreground">Routes</Label>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 text-xs gap-1"
                  onClick={() => {
                    setDraft({
                      ...draft,
                      routes: [
                        ...draft.routes,
                        {
                          id: `route-${Date.now()}`,
                          method: "GET",
                          path: "/example",
                          statusCode: 200,
                          headers: [],
                          body: "{}",
                          delay: 0,
                          matchQuery: false,
                        },
                      ],
                    })
                    setDirty(true)
                  }}
                >
                  <Plus className="h-3 w-3" /> Add route
                </Button>
              </div>

              {draft.routes.length === 0 && (
                <div className="text-xs text-muted-foreground">No routes. Add one to respond to requests.</div>
              )}

              {draft.routes.map((route, index) => (
                <div key={route.id || index} className="border border-border rounded-md p-2 space-y-2">
                  <div className="flex items-center gap-2">
                    <Input
                      value={route.method}
                      onChange={(e) => {
                        const routes = [...draft.routes]
                        routes[index] = { ...route, method: e.target.value.toUpperCase() }
                        setDraft({ ...draft, routes })
                        setDirty(true)
                      }}
                      className="h-7 w-20 text-xs font-mono"
                    />
                    <Input
                      value={route.path}
                      onChange={(e) => {
                        const routes = [...draft.routes]
                        routes[index] = { ...route, path: e.target.value }
                        setDraft({ ...draft, routes })
                        setDirty(true)
                      }}
                      className="h-7 flex-1 text-xs font-mono"
                      placeholder="/path"
                    />
                    <Input
                      type="number"
                      value={route.statusCode}
                      onChange={(e) => {
                        const routes = [...draft.routes]
                        routes[index] = { ...route, statusCode: parseInt(e.target.value) || 200 }
                        setDraft({ ...draft, routes })
                        setDirty(true)
                      }}
                      className="h-7 w-16 text-xs"
                      title="Status code"
                    />
                    <Input
                      type="number"
                      value={route.delay}
                      onChange={(e) => {
                        const routes = [...draft.routes]
                        routes[index] = { ...route, delay: parseInt(e.target.value) || 0 }
                        setDraft({ ...draft, routes })
                        setDirty(true)
                      }}
                      className="h-7 w-16 text-xs"
                      title="Delay (ms)"
                    />
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7"
                      aria-label="Remove route"
                      onClick={() => {
                        setDraft({ ...draft, routes: draft.routes.filter((_, i) => i !== index) })
                        setDirty(true)
                      }}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                  <Textarea
                    value={route.body}
                    onChange={(e) => {
                      const routes = [...draft.routes]
                      routes[index] = { ...route, body: e.target.value }
                      setDraft({ ...draft, routes })
                      setDirty(true)
                    }}
                    className="font-mono text-xs min-h-[60px]"
                    placeholder="Response body"
                  />
                </div>
              ))}
            </div>

            {draft.isRunning && (
              <div className="pt-2 border-t border-border">
                <div className="flex items-center justify-between mb-1">
                  <Label className="text-xs font-medium text-muted-foreground">Request log</Label>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-6 text-xs"
                    onClick={async () => {
                      await api.clearMockServerLogs(draft.id)
                      setLogs([])
                    }}
                  >
                    Clear
                  </Button>
                </div>
                <div className="text-xs font-mono space-y-0.5 max-h-40 overflow-y-auto">
                  {logs.length === 0 && <div className="text-muted-foreground">No requests yet.</div>}
                  {logs.map((log) => (
                    <div key={log.id} className="flex gap-2">
                      <span className="text-muted-foreground">{new Date(log.timestamp).toLocaleTimeString()}</span>
                      <span className="font-bold">{log.method}</span>
                      <span className="truncate">{log.path}</span>
                      <span className={cn(log.status < 400 ? "text-green-600" : "text-red-500")}>{log.status}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="text-xs text-muted-foreground">Create a mock server to serve canned responses on a local port.</div>
        )}
      </div>
    </div>
  )
}
