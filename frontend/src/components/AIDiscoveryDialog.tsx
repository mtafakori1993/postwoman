import React, { useCallback, useEffect, useMemo, useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { AlertTriangle, Check, Compass, Loader2 } from "lucide-react"
import * as api from "@/lib/api"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import { baseName, buildEdits, defaultDiscoveryOptions } from "./discovery/shared"
import { DiscoverySetupStep } from "./discovery/DiscoverySetupStep"
import { DiscoveryRunningStep } from "./discovery/DiscoveryRunningStep"
import { DiscoveryReviewStep } from "./discovery/DiscoveryReviewStep"
import { DiscoveryPreviewStep } from "./discovery/DiscoveryPreviewStep"
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

const STEPS: { id: Stage; label: string }[] = [
  { id: "setup", label: "Setup" },
  { id: "running", label: "Analyze" },
  { id: "review", label: "Review" },
  { id: "preview", label: "Import" },
]

export function AIDiscoveryDialog({ open, onOpenChange }: AIDiscoveryDialogProps) {
  const aiEnabled = useWorkspaceStore((s) => s.settings.aiEnabled)
  const refreshCollections = useWorkspaceStore((s) => s.refreshCollections)

  const [stage, setStage] = useState<Stage>("setup")
  const [options, setOptions] = useState<DiscoveryOptions>(defaultDiscoveryOptions)
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

  const patchOptions = (patch: Partial<DiscoveryOptions>) => setOptions((prev) => ({ ...prev, ...patch }))

  const refreshSession = useCallback(async (id: string) => {
    try {
      const next = await api.discoveryGet(id)
      setSession(next)
      if (next.progress && next.progress.stage) {
        setLog((prev) => (prev.length ? prev : [next.progress]))
      }
      if (next.status !== "running") {
        setStage("review")
        const endpoints = next.endpoints ?? []
        setEdits((prev) => (Object.keys(prev).length > 0 ? prev : buildEdits(endpoints)))
        setActiveID((prev) => prev || endpoints[0]?.id || "")
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

  const selectAll = (selected: boolean, ids?: string[]) =>
    setEdits((prev) => {
      const next: Record<string, DiscoveryEndpointEdit> = { ...prev }
      for (const id of ids ?? Object.keys(prev)) {
        if (next[id]) next[id] = { ...next[id], selected }
      }
      return next
    })

  const openSource = useCallback(
    async (endpoint: DiscoveredEndpoint) => {
      if (!endpoint.sourceFile || !sessionID) return
      try {
        const line = Math.max(1, endpoint.line || 1)
        setSource(await api.discoverySourceFile(sessionID, endpoint.sourceFile, Math.max(1, line - 5), line + 45))
      } catch (err) {
        setError(api.errorMessage(err))
      }
    },
    [sessionID],
  )

  const selectedEdits = useMemo(() => Object.values(edits).filter((edit) => edit.selected), [edits])

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


  const canVisit = (step: Stage) => {
    if (step === "setup") return true
    if (step === "running") return stage === "running"
    if (step === "review") return Boolean(session) && session?.status !== "running"
    return collectionJSON.length > 0
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex h-[92vh] max-w-[95vw] flex-col gap-0 overflow-hidden p-0 sm:max-w-[95vw]">
        <DialogHeader className="shrink-0 space-y-2 border-b px-6 py-4">
          <DialogTitle className="flex items-center gap-2">
            <Compass className="h-5 w-5 text-primary" /> API Discovery Agent
          </DialogTitle>
          <DialogDescription className="sr-only">
            Discover API endpoints in a local project and import them as a collection.
          </DialogDescription>
          <nav className="flex flex-wrap items-center gap-1.5">
            {STEPS.map((step, index) => {
              const active = stage === step.id
              const done = STEPS.findIndex((entry) => entry.id === stage) > index
              const enabled = canVisit(step.id)
              return (
                <button
                  key={step.id}
                  type="button"
                  disabled={!enabled}
                  onClick={() => enabled && setStage(step.id)}
                  className={cn(
                    "flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs transition-colors",
                    active && "border-primary bg-primary/10 text-foreground",
                    !active && done && "border-emerald-500/40 bg-emerald-500/5 text-foreground",
                    !active && !done && "text-muted-foreground",
                    enabled ? "cursor-pointer hover:bg-muted" : "cursor-not-allowed opacity-60",
                  )}
                >
                  {done && !active ? (
                    <Check className="h-3 w-3 text-emerald-500" />
                  ) : (
                    <span className="tabular-nums">{index + 1}</span>
                  )}
                  {step.label}
                </button>
              )
            })}
          </nav>
        </DialogHeader>

        {error && (
          <div className="mx-6 mt-3 flex shrink-0 items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-xs text-destructive">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="break-words">{error}</span>
          </div>
        )}

        <div className="min-h-0 flex-1">
          {stage === "setup" && (
            <DiscoverySetupStep
              options={options}
              patchOptions={patchOptions}
              browse={browse}
              start={start}
              busy={busy}
              aiEnabled={aiEnabled}
            />
          )}

          {stage === "running" && (
            <DiscoveryRunningStep
              session={session}
              log={log}
              cancel={cancel}
              refresh={() => void refreshSession(sessionID)}
            />
          )}
          {stage === "review" &&
            (session ? (
              <DiscoveryReviewStep
                session={session}
                edits={edits}
                updateEdit={updateEdit}
                selectAll={selectAll}
                activeID={activeID}
                setActiveID={setActiveID}
                source={source}
                openSource={openSource}
                collectionName={options.collectionName}
                baseUrl={options.baseUrl}
                patchOptions={patchOptions}
                preview={preview}
                runImport={runImport}
                busy={busy}
              />
            ) : (
              <div className="flex h-full items-center justify-center">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            ))}

          {stage === "preview" && (
            <DiscoveryPreviewStep
              preview={collectionJSON}
              validation={validation}
              imported={imported}
              collectionName={options.collectionName || baseName(options.directory)}
              selectedCount={selectedEdits.length}
              busy={busy}
              back={() => setStage("review")}
              runImport={runImport}
            />
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

