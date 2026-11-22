import React, { useCallback, useEffect, useMemo, useState } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  Compass,
  FolderOpen,
  Loader2,
  Play,
  Search,
  Sparkles,
  Square,
  Upload,
} from "lucide-react"
import * as api from "@/lib/api"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import type {
  DiscoveredEndpoint,
  DiscoveryEndpointEdit,
  DiscoveryGenerateRequest,
  DiscoveryOptions,
  DiscoveryProgress,
  DiscoverySession,
  DiscoverySourceFile,
} from "@/types"

interface AIDiscoveryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type Stage = "setup" | "running" | "review" | "preview"

function baseName(path: string): string {
  const clean = path.replace(/[\\/]+$/, "")
  const parts = clean.split(/[\\/]/)
  return parts[parts.length - 1] || "discovered-api"
}

function defaultOptions(): DiscoveryOptions {
  return {
    directory: "",
    collectionName: "",
    baseUrl: "http://localhost:3000",
    excludePatterns: [],
    includeTests: false,
    respectGitignore: true,
    useAi: false,
    maxFiles: 2000,
    maxFileSizeKb: 512,
    maxAiFiles: 40,
    followSymlinks: false,
    includeHidden: false,
  }
}

function buildEdits(endpoints: DiscoveredEndpoint[]): Record<string, DiscoveryEndpointEdit> {
  const edits: Record<string, DiscoveryEndpointEdit> = {}
  for (const endpoint of endpoints) {
    edits[endpoint.id] = {
      id: endpoint.id,
      selected: endpoint.selected !== false,
      name: endpoint.name || `${endpoint.method} ${endpoint.path}`,
      description: endpoint.description ?? "",
      folder: endpoint.folder ?? "",
    }
  }
  return edits
}

function methodClass(method: string): string {
  switch (method.toUpperCase()) {
    case "GET":
      return "bg-emerald-500/15 text-emerald-600"
    case "POST":
      return "bg-blue-500/15 text-blue-600"
    case "PUT":
    case "PATCH":
      return "bg-amber-500/15 text-amber-600"
    case "DELETE":
      return "bg-rose-500/15 text-rose-600"
    default:
      return "bg-muted text-muted-foreground"
  }
}

export function AIDiscoveryDialog({ open, onOpenChange }: AIDiscoveryDialogProps) {
  const aiEnabled = useWorkspaceStore((s) => s.settings.aiEnabled)
  const refreshCollections = useWorkspaceStore((s) => s.refreshCollections)

  const [stage, setStage] = useState<Stage>("setup")
  const [options, setOptions] = useState<DiscoveryOptions>(defaultOptions)
  const [sessionID, setSessionID] = useState("")
  const [session, setSession] = useState<DiscoverySession | null>(null)
  const [log, setLog] = useState<DiscoveryProgress[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [edits, setEdits] = useState<Record<string, DiscoveryEndpointEdit>>({})
  const [activeID, setActiveID] = useState("")
  const [source, setSource] = useState<DiscoverySourceFile | null>(null)
  const [collectionJSON, setCollectionJSON] = useState("")
  const [validation, setValidation] = useState<string[]>([])
  const [imported, setImported] = useState<string[]>([])

  const patchOptions = (patch: Partial<DiscoveryOptions>) =>
    setOptions((prev) => ({ ...prev, ...patch }))

  const refreshSession = useCallback(async (id: string) => {
    try {
      const next = await api.discoveryGet(id)
      setSession(next)
      if (next.progress && next.progress.stage) {
        setLog((prev) => (prev.length ? prev : [next.progress]))
      }
      if (next.status !== "running") {
        setStage("review")
        setEdits((prev) => (Object.keys(prev).length > 0 ? prev : buildEdits(next.endpoints)))
        setActiveID((prev) => prev || next.endpoints[0]?.id || "")
        if (next.status === "failed" && next.error) setError(next.error)
      }
    } catch (err) {
      setError(api.errorMessage(err))
      setStage("review")
    }
  }, [])

  // Stream agent progress while a run is in flight.
  useEffect(() => {
    if (!sessionID) return
    return api.onEvent<DiscoveryProgress>("discovery:progress", (progress) => {
      if (progress.sessionId && progress.sessionId !== sessionID) return
      setLog((prev) => [...prev.slice(-199), progress])
      if (progress.done) void refreshSession(sessionID)
    })
  }, [sessionID, refreshSession])

  // Poll as a fallback in case an event is missed (or the run finishes fast).
  useEffect(() => {
    if (!open || !sessionID || stage !== "running") return
    const timer = setInterval(() => void refreshSession(sessionID), 900)
    return () => clearInterval(timer)
  }, [open, sessionID, stage, refreshSession])

  useEffect(() => {
    if (open) setError(null)
  }, [open])

  const browse = async () => {
    try {
      const directory = await api.openDirectoryDialog("Choose a project directory")
      if (directory) {
        patchOptions({
          directory,
          collectionName: options.collectionName || `${baseName(directory)} API`,
        })
      }
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const start = async () => {
    const directory = options.directory.trim()
    if (!directory) {
      setError("Choose a project directory first.")
      return
    }
    setBusy(true)
    setError(null)
    setSession(null)
    setSessionID("")
    setLog([])
    setEdits({})
    setSource(null)
    setCollectionJSON("")
    setValidation([])
    setImported([])
    try {
      const id = await api.discoveryStart({
        ...options,
        directory,
        collectionName: options.collectionName.trim() || `${baseName(directory)} API`,
        useAi: options.useAi && aiEnabled,
      })
      setSessionID(id)
      setStage("running")
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const cancel = async () => {
    if (!sessionID) return
    try {
      await api.discoveryCancel(sessionID)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const updateEdit = (id: string, patch: Partial<DiscoveryEndpointEdit>) =>
    setEdits((prev) => ({ ...prev, [id]: { ...prev[id], ...patch } }))

  const selectAll = (selected: boolean) =>
    setEdits((prev) => {
      const next: Record<string, DiscoveryEndpointEdit> = {}
      for (const [id, edit] of Object.entries(prev)) next[id] = { ...edit, selected }
      return next
    })

  const openSource = async (endpoint: DiscoveredEndpoint) => {
    if (!endpoint.sourceFile) return
    try {
      const line = Math.max(1, endpoint.line || 1)
      setSource(
        await api.discoverySourceFile(sessionID, endpoint.sourceFile, Math.max(1, line - 5), line + 45),
      )
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const selectedEdits = useMemo(
    () => Object.values(edits).filter((edit) => edit.selected),
    [edits],
  )

  const generateRequest = (): DiscoveryGenerateRequest => ({
    sessionId: sessionID,
    collectionName:
      options.collectionName.trim() ||
      (options.directory ? `${baseName(options.directory)} API` : "Discovered API"),
    baseUrl: options.baseUrl,
    endpoints: selectedEdits,
  })

  const preview = async () => {
    setBusy(true)
    setError(null)
    try {
      const json = await api.discoveryGenerate(generateRequest())
      setCollectionJSON(json)
      setValidation(await api.discoveryValidate(json))
      setStage("preview")
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const runImport = async () => {
    setBusy(true)
    setError(null)
    try {
      const summary = await api.discoveryImport(generateRequest())
      setImported([summary.message || "Collection imported.", ...(summary.warnings ?? [])])
      try {
        setCollectionJSON(await api.discoveryGenerate(generateRequest()))
      } catch {
        // The import already succeeded; a preview failure must not hide that.
      }
      await refreshCollections()
      setStage("preview")
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const activeEndpoint = session?.endpoints.find((endpoint) => endpoint.id === activeID) ?? null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-5xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Compass className="h-5 w-5 text-primary" /> API Discovery Agent
          </DialogTitle>
          <DialogDescription>
            Point the agent at a project folder. It reads route definitions, optionally asks your
            configured AI to fill the gaps, then lets you review every endpoint before importing.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <div className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="break-words">{error}</span>
          </div>
        )}

        {stage === "setup" && (
          <div className="space-y-3">
            <div>
              <span className="text-xs text-muted-foreground">Project directory</span>
              <div className="mt-1 flex gap-2">
                <Input
                  value={options.directory}
                  onChange={(e) => patchOptions({ directory: e.target.value })}
                  placeholder="/path/to/project"
                  className="h-8 font-mono text-xs"
                />
                <Button variant="outline" size="sm" className="h-8 gap-1" onClick={browse}>
                  <FolderOpen className="h-3.5 w-3.5" /> Browse
                </Button>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-2">
              <div>
                <span className="text-xs text-muted-foreground">Collection name</span>
                <Input
                  value={options.collectionName}
                  onChange={(e) => patchOptions({ collectionName: e.target.value })}
                  placeholder="My API"
                  className="mt-1 h-8 text-xs"
                />
              </div>
              <div>
                <span className="text-xs text-muted-foreground">Base URL</span>
                <Input
                  value={options.baseUrl}
                  onChange={(e) => patchOptions({ baseUrl: e.target.value })}
                  placeholder="http://localhost:3000"
                  className="mt-1 h-8 font-mono text-xs"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-x-6 gap-y-1.5 text-sm">
              <label className="flex items-center justify-between gap-2">
                <span>Include test files</span>
                <Checkbox
                  checked={options.includeTests}
                  onCheckedChange={(c) => patchOptions({ includeTests: c === true })}
                />
              </label>
              <label className="flex items-center justify-between gap-2">
                <span>Respect .gitignore</span>
                <Checkbox
                  checked={options.respectGitignore}
                  onCheckedChange={(c) => patchOptions({ respectGitignore: c === true })}
                />
              </label>
              <label className="flex items-center justify-between gap-2">
                <span>Include hidden files</span>
                <Checkbox
                  checked={options.includeHidden}
                  onCheckedChange={(c) => patchOptions({ includeHidden: c === true })}
                />
              </label>
              <label className="flex items-center justify-between gap-2">
                <span>Follow symlinks</span>
                <Checkbox
                  checked={options.followSymlinks}
                  onCheckedChange={(c) => patchOptions({ followSymlinks: c === true })}
                />
              </label>
              <label className="flex items-center justify-between gap-2">
                <span className="flex items-center gap-1">
                  <Sparkles className="h-3.5 w-3.5" /> AI enrichment
                </span>
                <Checkbox
                  checked={options.useAi && aiEnabled}
                  disabled={!aiEnabled}
                  onCheckedChange={(c) => patchOptions({ useAi: c === true })}
                />
              </label>
            </div>

            {!aiEnabled && (
              <p className="pw-text-2xs text-muted-foreground">
                AI enrichment is off because no AI provider is enabled. Static analysis still runs.
                Configure a provider under Settings → AI.
              </p>
            )}

            <div className="grid grid-cols-3 gap-2">
              <div>
                <span className="text-xs text-muted-foreground">Max files</span>
                <Input
                  type="number"
                  min="1"
                  value={options.maxFiles}
                  onChange={(e) => patchOptions({ maxFiles: parseInt(e.target.value) || 2000 })}
                  className="mt-1 h-8 text-xs"
                />
              </div>
              <div>
                <span className="text-xs text-muted-foreground">Max file size (KB)</span>
                <Input
                  type="number"
                  min="1"
                  value={options.maxFileSizeKb}
                  onChange={(e) => patchOptions({ maxFileSizeKb: parseInt(e.target.value) || 512 })}
                  className="mt-1 h-8 text-xs"
                />
              </div>
              <div>
                <span className="text-xs text-muted-foreground">Max files to AI</span>
                <Input
                  type="number"
                  min="0"
                  value={options.maxAiFiles}
                  onChange={(e) => patchOptions({ maxAiFiles: parseInt(e.target.value) || 0 })}
                  className="mt-1 h-8 text-xs"
                />
              </div>
            </div>

            <div>
              <span className="text-xs text-muted-foreground">
                Exclude patterns (comma or newline separated)
              </span>
              <Input
                value={options.excludePatterns.join(", ")}
                onChange={(e) =>
                  patchOptions({
                    excludePatterns: e.target.value
                      .split(/[,\n]/)
                      .map((part) => part.trim())
                      .filter(Boolean),
                  })
                }
                placeholder="vendor, generated, *.min.js"
                className="mt-1 h-8 font-mono text-xs"
              />
            </div>

            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <p className="pw-text-2xs text-muted-foreground">
                The agent only reads files inside the chosen directory. Nothing is uploaded unless AI
                enrichment is enabled, and detected secrets are redacted before sending.
              </p>
              <Button onClick={start} disabled={busy || !options.directory.trim()} className="gap-1">
                {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                Start discovery
              </Button>
            </div>
          </div>
        )}

        {stage === "running" && (
          <div className="space-y-3">
            <div className="space-y-2 rounded-md border p-3">
              <div className="flex items-center justify-between gap-2">
                <span className="flex items-center gap-2 text-sm font-medium">
                  <Loader2 className="h-4 w-4 animate-spin text-primary" />
                  {session?.progress.message || "Starting the discovery agent…"}
                </span>
                <span className="text-xs text-muted-foreground">
                  {session?.progress.percent ?? 0}%
                </span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded bg-muted">
                <div
                  className="h-full bg-primary transition-all"
                  style={{
                    width: `${Math.max(0, Math.min(100, session?.progress.percent ?? 0))}%`,
                  }}
                />
              </div>
              <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
                <span>Stage: {session?.progress.stage || "init"}</span>
                <span>Files: {session?.progress.filesSeen ?? session?.stats.filesSeen ?? 0}</span>
                <span>Endpoints: {session?.progress.endpoints ?? session?.endpoints?.length ?? 0}</span>
                {session?.progress.current && (
                  <span className="max-w-[320px] truncate font-mono">{session.progress.current}</span>
                )}
              </div>
            </div>
            <ScrollArea className="h-[260px] rounded-md border p-2">
              <div className="space-y-1 font-mono text-xs">
                {log.length === 0 && (
                  <p className="text-muted-foreground">Waiting for the first update…</p>
                )}
                {log.map((entry, index) => (
                  <div key={`${index}-${entry.stage}`} className="flex gap-2">
                    <span className="w-10 shrink-0 text-right text-muted-foreground">
                      {entry.percent}%
                    </span>
                    <span className="w-20 shrink-0 truncate text-primary">{entry.stage}</span>
                    <span className="min-w-0 break-words">{entry.message}</span>
                  </div>
                ))}
              </div>
            </ScrollArea>
            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <p className="pw-text-2xs text-muted-foreground">
                {session?.files?.length ?? 0} files considered so far. You can cancel at any time.
              </p>
              <div className="flex gap-2">
                <Button variant="outline" onClick={cancel} className="gap-1">
                  <Square className="h-3.5 w-3.5" /> Cancel
                </Button>
                <Button
                  variant="ghost"
                  onClick={() => void refreshSession(sessionID)}
                  disabled={!sessionID}
                >
                  Refresh
                </Button>
              </div>
            </div>
          </div>
        )}
        {stage === "review" && session && (
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-1.5 text-xs">
              <Badge variant="secondary">{session.projectName || baseName(session.root)}</Badge>
              <Badge variant="outline">{session.endpoints.length} endpoints</Badge>
              <Badge variant="outline">
                {session.stats.filesAnalyzed}/{session.stats.filesSeen} files
              </Badge>
              {session.frameworks?.map((framework) => (
                <Badge key={framework} variant="outline">
                  {framework}
                </Badge>
              ))}
              {session.stats.lowConfidence > 0 && (
                <Badge variant="outline" className="text-amber-600">
                  {session.stats.lowConfidence} low confidence
                </Badge>
              )}
              <span className="ml-auto text-muted-foreground">
                {selectedEdits.length} selected
              </span>
            </div>

            {session.warnings?.length > 0 && (
              <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
                <p className="flex items-center gap-1 font-medium">
                  <AlertTriangle className="h-3.5 w-3.5" /> Agent warnings
                </p>
                <ul className="mt-1 list-inside list-disc space-y-0.5">
                  {session.warnings.slice(0, 5).map((warning) => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </div>
            )}

            <div className="grid grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)] gap-3">
              <div className="space-y-2">
                <div className="flex items-center gap-1.5">
                  <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => selectAll(true)}>
                    Select all
                  </Button>
                  <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => selectAll(false)}>
                    None
                  </Button>
                  <span className="ml-auto flex items-center gap-1 pw-text-2xs text-muted-foreground">
                    <Search className="h-3 w-3" /> Click a row to edit it
                  </span>
                </div>
                <ScrollArea className="h-[340px] rounded-md border">
                  <div className="divide-y">
                    {session.endpoints.length === 0 && (
                      <p className="p-3 text-xs text-muted-foreground">
                        No endpoints were found. Try enabling AI enrichment or widening the scan
                        options.
                      </p>
                    )}
                    {session.endpoints.map((endpoint) => {
                      const edit = edits[endpoint.id]
                      const selected = edit?.selected ?? true
                      return (
                        <div
                          key={endpoint.id}
                          role="button"
                          tabIndex={0}
                          onClick={() => {
                            setActiveID(endpoint.id)
                            setSource(null)
                          }}
                          onKeyDown={(event) => {
                            if (event.key === "Enter" || event.key === " ") {
                              event.preventDefault()
                              setActiveID(endpoint.id)
                              setSource(null)
                            }
                          }}
                          className={cn(
                            "flex cursor-pointer items-start gap-2 p-2 hover:bg-muted/50",
                            activeID === endpoint.id && "bg-muted",
                          )}
                        >
                          <span onClick={(event) => event.stopPropagation()} className="pt-0.5">
                            <Checkbox
                              checked={selected}
                              onCheckedChange={(checked) =>
                                updateEdit(endpoint.id, { selected: checked === true })
                              }
                            />
                          </span>
                          <span
                            className={cn(
                              "mt-0.5 rounded px-1.5 py-0.5 pw-text-2xs font-semibold",
                              methodClass(endpoint.method),
                            )}
                          >
                            {endpoint.method}
                          </span>
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-xs font-medium">
                              {edit?.name || endpoint.path}
                            </p>
                            <p className="truncate font-mono pw-text-2xs text-muted-foreground">
                              {endpoint.path}
                            </p>
                          </div>
                          <span
                            title={`${endpoint.confidence}% confidence`}
                            className={cn(
                              "mt-1 h-2 w-2 shrink-0 rounded-full",
                              endpoint.confidence >= 70
                                ? "bg-emerald-500"
                                : endpoint.confidence >= 40
                                  ? "bg-amber-500"
                                  : "bg-rose-500",
                            )}
                          />
                        </div>
                      )
                    })}
                  </div>
                </ScrollArea>
              </div>

              <div className="space-y-2">
                {!activeEndpoint || !edits[activeEndpoint.id] ? (
                  <div className="flex h-[340px] items-center justify-center rounded-md border text-xs text-muted-foreground">
                    Select an endpoint to review it.
                  </div>
                ) : (
                  <>
                    <div className="space-y-2 rounded-md border p-2">
                      <div className="flex items-center gap-2 pw-text-2xs text-muted-foreground">
                        <span
                          className={cn(
                            "rounded px-1.5 py-0.5 font-semibold",
                            methodClass(activeEndpoint.method),
                          )}
                        >
                          {activeEndpoint.method}
                        </span>
                        <span className="min-w-0 flex-1 truncate font-mono">
                          {activeEndpoint.path}
                        </span>
                        <span>{activeEndpoint.confidence}%</span>
                      </div>
                      <div>
                        <span className="text-xs text-muted-foreground">Request name</span>
                        <Input
                          className="mt-1 h-8 text-xs"
                          value={edits[activeEndpoint.id].name}
                          onChange={(event) =>
                            updateEdit(activeEndpoint.id, { name: event.target.value })
                          }
                        />
                      </div>
                      <div>
                        <span className="text-xs text-muted-foreground">Folder</span>
                        <Input
                          className="mt-1 h-8 font-mono text-xs"
                          list="discovery-folders"
                          value={edits[activeEndpoint.id].folder}
                          onChange={(event) =>
                            updateEdit(activeEndpoint.id, { folder: event.target.value })
                          }
                          placeholder="Module / controller"
                        />
                        <datalist id="discovery-folders">
                          {(session.folders ?? []).map((folder) => (
                            <option key={folder} value={folder} />
                          ))}
                        </datalist>
                      </div>
                      <div>
                        <span className="text-xs text-muted-foreground">Description</span>
                        <Textarea
                          className="mt-1 min-h-[52px] text-xs"
                          rows={2}
                          value={edits[activeEndpoint.id].description}
                          onChange={(event) =>
                            updateEdit(activeEndpoint.id, { description: event.target.value })
                          }
                        />
                      </div>
                      <div className="flex flex-wrap items-center gap-1.5 pw-text-2xs text-muted-foreground">
                        {activeEndpoint.sourceFile && (
                          <span className="max-w-[220px] truncate font-mono">
                            {activeEndpoint.sourceFile}
                            {activeEndpoint.line ? `:${activeEndpoint.line}` : ""}
                          </span>
                        )}
                        {activeEndpoint.authRequired && (
                          <Badge variant="outline">
                            auth: {activeEndpoint.authType || "required"}
                          </Badge>
                        )}
                        {activeEndpoint.origin && (
                          <Badge variant="outline">{activeEndpoint.origin}</Badge>
                        )}
                      </div>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-7 text-xs"
                        onClick={() => void openSource(activeEndpoint)}
                        disabled={!activeEndpoint.sourceFile}
                      >
                        View source
                      </Button>
                      {activeEndpoint.confidence < 40 && (
                        <p className="pw-text-2xs text-amber-600">
                          Low confidence — verify the method, path and parameters before importing.
                        </p>
                      )}
                    </div>
                    {source && (
                      <ScrollArea className="h-[150px] rounded-md border bg-muted/30">
                        <pre className="p-2 font-mono pw-text-2xs leading-4">
                          {source.content}
                          {source.truncated ? "\n… (truncated)" : ""}
                        </pre>
                      </ScrollArea>
                    )}
                  </>
                )}
              </div>
            </div>

            <div className="space-y-2 border-t pt-3">
              <div className="grid grid-cols-2 gap-2">
                <div>
                  <span className="text-xs text-muted-foreground">Collection name</span>
                  <Input
                    className="mt-1 h-8 text-xs"
                    value={options.collectionName}
                    onChange={(event) => patchOptions({ collectionName: event.target.value })}
                  />
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">Base URL</span>
                  <Input
                    className="mt-1 h-8 font-mono text-xs"
                    value={options.baseUrl}
                    onChange={(event) => patchOptions({ baseUrl: event.target.value })}
                  />
                </div>
              </div>
              <div className="flex items-center justify-between gap-2">
                <Button variant="ghost" className="gap-1" onClick={() => setStage("setup")}>
                  <ArrowLeft className="h-4 w-4" /> Back
                </Button>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    className="gap-1"
                    onClick={() => void preview()}
                    disabled={busy || selectedEdits.length === 0}
                  >
                    {busy ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Search className="h-4 w-4" />
                    )}
                    Preview
                  </Button>
                  <Button
                    className="gap-1"
                    onClick={() => void runImport()}
                    disabled={busy || selectedEdits.length === 0}
                  >
                    {busy ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Upload className="h-4 w-4" />
                    )}
                    Import to Postwoman
                  </Button>
                </div>
              </div>
            </div>
          </div>
        )}
        {stage === "preview" && (
          <div className="space-y-3">
            {imported.length > 0 && (
              <div className="rounded-md border border-emerald-500/40 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-400">
                <p className="flex items-center gap-1 font-medium">
                  <CheckCircle2 className="h-3.5 w-3.5" /> {imported[0]}
                </p>
                {imported.length > 1 && (
                  <ul className="mt-1 list-inside list-disc space-y-0.5">
                    {imported.slice(1).map((line) => (
                      <li key={line}>{line}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {validation.length > 0 && (
              <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
                <p className="flex items-center gap-1 font-medium">
                  <AlertTriangle className="h-3.5 w-3.5" /> Validation warnings
                </p>
                <ul className="mt-1 list-inside list-disc space-y-0.5">
                  {validation.map((warning) => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </div>
            )}

            <ScrollArea className="h-[360px] rounded-md border bg-muted/20">
              <pre className="whitespace-pre-wrap p-3 font-mono pw-text-2xs leading-4">
                {collectionJSON || "No preview generated."}
              </pre>
            </ScrollArea>

            <div className="flex items-center justify-between gap-2 border-t pt-3">
              <Button variant="ghost" className="gap-1" onClick={() => setStage("review")}>
                <ArrowLeft className="h-4 w-4" /> Back to review
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    api
                      .copyToClipboard(collectionJSON)
                      .catch((err) => setError(api.errorMessage(err)))
                  }}
                  disabled={!collectionJSON}
                >
                  Copy JSON
                </Button>
                <Button
                  className="gap-1"
                  onClick={() => void runImport()}
                  disabled={busy || imported.length > 0 || !collectionJSON}
                >
                  {busy ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : imported.length > 0 ? (
                    <CheckCircle2 className="h-4 w-4" />
                  ) : (
                    <Upload className="h-4 w-4" />
                  )}
                  {imported.length > 0 ? "Imported" : "Import to Postwoman"}
                </Button>
              </div>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}