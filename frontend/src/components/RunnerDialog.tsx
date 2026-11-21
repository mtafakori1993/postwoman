import React, { useEffect, useState } from "react"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Play, FolderOpen, CheckCircle2, XCircle } from "lucide-react"
import { cn } from "@/lib/utils"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { RunResult } from "@/types"

interface RunnerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function RunnerDialog({ open, onOpenChange }: RunnerDialogProps) {
  const { collections, environments, activeEnvironmentId } = useWorkspaceStore()
  const [collectionId, setCollectionId] = useState("")
  const [environmentId, setEnvironmentId] = useState(activeEnvironmentId ?? "")
  const [iterations, setIterations] = useState(1)
  const [delay, setDelay] = useState(0)
  const [dataFile, setDataFile] = useState("")
  const [stopOnError, setStopOnError] = useState(false)
  const [running, setRunning] = useState(false)
  const [progress, setProgress] = useState(0)
  const [result, setResult] = useState<RunResult | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setResult(null)
      setError(null)
      setProgress(0)
      if (!collectionId && collections.length > 0) setCollectionId(collections[0].id)
    }
  }, [open, collections, collectionId])

  useEffect(() => {
    if (!open) return
    const unsub = api.onEvent("runner:progress", () => setProgress((p) => p + 1))
    return unsub
  }, [open])

  const pickDataFile = async () => {
    try {
      const path = await api.openFileDialog("Select CSV or JSON data file", "*.csv;*.json")
      if (path) setDataFile(path)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const run = async () => {
    if (!collectionId) return
    setRunning(true)
    setError(null)
    setResult(null)
    setProgress(0)
    try {
      const res = await api.runCollection({
        collectionId,
        environmentId: environmentId || undefined,
        iterations,
        delay,
        dataFile: dataFile || undefined,
        stopOnError,
      })
      setResult(res)
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setRunning(false)
    }
  }

  const requests = result?.iterations?.flatMap((it) => it.requests) ?? []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Collection Runner</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs">Collection</Label>
              <Select value={collectionId} onValueChange={setCollectionId}>
                <SelectTrigger className="h-9 mt-1">
                  <SelectValue placeholder="Select collection" />
                </SelectTrigger>
                <SelectContent>
                  {collections.map((c) => (
                    <SelectItem key={c.id} value={c.id}>
                      {c.info.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label className="text-xs">Environment</Label>
              <Select value={environmentId || "none"} onValueChange={(v) => setEnvironmentId(v === "none" ? "" : v)}>
                <SelectTrigger className="h-9 mt-1">
                  <SelectValue placeholder="No environment" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">No environment</SelectItem>
                  {environments.map((e) => (
                    <SelectItem key={e.id} value={e.id}>
                      {e.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label className="text-xs">Iterations</Label>
              <Input
                type="number"
                min={1}
                value={iterations}
                onChange={(e) => setIterations(Math.max(1, parseInt(e.target.value) || 1))}
                className="h-9 mt-1"
              />
            </div>
            <div>
              <Label className="text-xs">Delay between requests (ms)</Label>
              <Input
                type="number"
                min={0}
                value={delay}
                onChange={(e) => setDelay(Math.max(0, parseInt(e.target.value) || 0))}
                className="h-9 mt-1"
              />
            </div>
          </div>

          <div>
            <Label className="text-xs">Data file (optional, CSV or JSON)</Label>
            <div className="flex items-center gap-2 mt-1">
              <Input value={dataFile} readOnly placeholder="No data file" className="h-9 font-mono text-xs" />
              <Button variant="outline" size="sm" className="h-9 gap-1" onClick={pickDataFile}>
                <FolderOpen className="h-4 w-4" /> Choose
              </Button>
              {dataFile && (
                <Button variant="ghost" size="sm" className="h-9" onClick={() => setDataFile("")}>
                  Clear
                </Button>
              )}
            </div>
          </div>

          <label className="flex items-center gap-2 cursor-pointer text-sm">
            <Checkbox checked={stopOnError} onCheckedChange={(c) => setStopOnError(c === true)} />
            Stop on first error
          </label>

          {error && <div className="text-sm text-destructive">{error}</div>}

          <div className="flex items-center gap-3">
            <Button onClick={run} disabled={running || !collectionId} className="gap-1">
              <Play className="h-4 w-4" /> {running ? `Running (${progress})...` : "Run"}
            </Button>
            {result && (
              <div className="flex items-center gap-3 text-sm">
                <Badge variant="secondary">{result.totalRequests} requests</Badge>
                <Badge variant={result.failedTests > 0 ? "destructive" : "secondary"}>
                  {result.passedTests} passed / {result.failedTests} failed
                </Badge>
              </div>
            )}
          </div>

          {requests.length > 0 && (
            <ScrollArea className="max-h-[280px] border border-border rounded-md">
              <div className="divide-y divide-border">
                {requests.map((r, i) => (
                  <div key={`${r.requestId}-${i}`} className="flex items-center gap-2 px-3 py-1.5 text-sm">
                    {r.status === "error" || r.status === "failed" ? (
                      <XCircle className="h-4 w-4 text-red-500 shrink-0" />
                    ) : (
                      <CheckCircle2 className="h-4 w-4 text-green-500 shrink-0" />
                    )}
                    <span className="flex-1 truncate">{r.requestName}</span>
                    <span className="text-xs text-muted-foreground font-mono">
                      {r.response?.status ?? "—"}
                    </span>
                    <span className={cn("text-xs font-mono", r.status === "error" && "text-red-500")}>{r.duration} ms</span>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
