import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Check, Copy, Download, Loader2 } from "lucide-react"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { Request } from "@/types"

const TARGETS = ["curl", "http", "fetch", "node", "python", "go", "php"] as const

interface CodeExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: Request | null
}

export function CodeExportDialog({ open, onOpenChange, request }: CodeExportDialogProps) {
  const activeEnvironmentId = useWorkspaceStore((s) => s.activeEnvironmentId)
  const [target, setTarget] = useState<string>("curl")
  const [code, setCode] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!open || !request) return
    let cancelled = false
    setBusy(true)
    setError(null)
    api
      .exportRequestCode(request, target)
      .then((generated) => {
        if (!cancelled) setCode(generated)
      })
      .catch((err) => {
        if (!cancelled) setError(api.errorMessage(err))
      })
      .finally(() => {
        if (!cancelled) setBusy(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, request, target, activeEnvironmentId])

  const copy = async () => {
    await api.copyToClipboard(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Generate code</DialogTitle>
          <DialogDescription>Runnable snippets that reflect the current request configuration.</DialogDescription>
        </DialogHeader>
        <div className="flex items-center gap-2">
          <Tabs value={target} onValueChange={setTarget}>
            <TabsList>
              {TARGETS.map((item) => (
                <TabsTrigger key={item} value={item} className="text-xs capitalize">{item}</TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
          <div className="ml-auto flex items-center gap-1">
            <Button variant="ghost" size="sm" className="gap-1.5" onClick={copy} disabled={!code}>
              {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />} Copy
            </Button>
            <Button variant="ghost" size="sm" className="gap-1.5" disabled={!code} onClick={() => api.saveTextFile(`request.${target}`, code).catch(console.error)}>
              <Download className="h-3.5 w-3.5" /> Save
            </Button>
          </div>
        </div>
        {busy && <div className="flex items-center gap-2 text-xs text-muted-foreground"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Generating…</div>}
        {error && <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div>}
        <pre className="max-h-96 overflow-auto rounded-md border border-border bg-muted/20 p-3 font-mono text-xs">{code}</pre>
      </DialogContent>
    </Dialog>
  )
}
