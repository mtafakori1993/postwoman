import React, { useMemo } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { AlertTriangle, ArrowLeft, Check, CheckCircle2, Copy, Download, FileJson, Loader2, Upload } from "lucide-react"
import * as api from "@/lib/api"

interface DiscoveryPreviewStepProps {
  preview: string
  validation: string[]
  imported: string[]
  collectionName: string
  selectedCount: number
  busy: boolean
  back: () => void
  runImport: () => void
}

export function DiscoveryPreviewStep({
  preview,
  validation,
  imported,
  collectionName,
  selectedCount,
  busy,
  back,
  runImport,
}: DiscoveryPreviewStepProps) {
  const stats = useMemo(() => {
    try {
      const parsed = JSON.parse(preview)
      const items = Array.isArray(parsed?.item) ? parsed.item : []
      const requests = items.flatMap((entry: { item?: unknown[] }) => (Array.isArray(entry?.item) ? entry.item : []))
      return {
        collections: items.length,
        requests: requests.length,
        bytes: new Blob([preview]).size,
      }
    } catch {
      return { collections: 0, requests: selectedCount, bytes: new Blob([preview]).size }
    }
  }, [preview, selectedCount])

  const fileName = `${(collectionName || "discovery").replace(/[^\w.-]+/g, "-")}.postman_collection.json`

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-b px-6 py-3">
        <div className="flex items-center gap-2">
          <FileJson className="h-4 w-4 text-primary" />
          <span className="text-sm font-semibold">Collection preview</span>
          <Badge variant="outline">{stats.collections} collection(s)</Badge>
          <Badge variant="outline">{stats.requests} requests</Badge>
          <Badge variant="outline">{(stats.bytes / 1024).toFixed(1)} KB</Badge>
        </div>
        <div className="flex gap-2">
          <Button variant="ghost" size="sm" className="gap-1.5" onClick={() => api.copyToClipboard(preview)}>
            <Copy className="h-4 w-4" /> Copy JSON
          </Button>
          <Button variant="outline" size="sm" className="gap-1.5" onClick={() => api.saveTextFile(fileName, preview)}>
            <Download className="h-4 w-4" /> Export
          </Button>
        </div>
      </div>

      {imported.length > 0 && (
        <div className="mx-6 mt-3 shrink-0 rounded-md border border-emerald-500/40 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-700 dark:text-emerald-400">
          <p className="flex items-center gap-1 font-medium">
            <CheckCircle2 className="h-3.5 w-3.5" /> {imported[0]}
          </p>
          {imported.slice(1).map((message) => (
            <p key={message} className="mt-0.5">
              {message}
            </p>
          ))}
        </div>
      )}

      {validation.length > 0 && (
        <div className="mx-6 mt-3 shrink-0 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
          <p className="flex items-center gap-1 font-medium">
            <AlertTriangle className="h-3.5 w-3.5" /> Collection validation
          </p>
          <ul className="mt-1 list-inside list-disc space-y-0.5">
            {validation.slice(0, 5).map((message) => (
              <li key={message}>{message}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-hidden p-4">
        <pre className="h-full overflow-auto rounded-md border bg-muted/20 p-3 font-mono pw-text-2xs leading-5">
          {preview || "// No preview generated yet."}
        </pre>
      </div>

      <footer className="flex shrink-0 items-center justify-between gap-3 border-t px-6 py-3">
        <Button variant="ghost" className="gap-1.5" onClick={back}>
          <ArrowLeft className="h-4 w-4" /> Back to review
        </Button>
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1 pw-text-2xs text-muted-foreground">
            <Check className="h-3.5 w-3.5" /> Nothing is imported until you confirm.
          </span>
          <Button className="gap-1.5" onClick={runImport} disabled={busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
            Import {selectedCount} endpoint{selectedCount === 1 ? "" : "s"}
          </Button>
        </div>
      </footer>
    </div>
  )
}