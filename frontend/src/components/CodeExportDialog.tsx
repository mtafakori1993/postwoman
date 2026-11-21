import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Check, Copy, Download, Loader2 } from "lucide-react"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { Request } from "@/types"

// Target id -> menu label. Using a select keeps the dialog usable at any width
// (seven tabs used to overflow narrow dialogs).
const TARGETS: { value: string; label: string }[] = [
  { value: "curl", label: "cURL" },
  { value: "http", label: "Raw HTTP" },
  { value: "fetch", label: "JavaScript (fetch)" },
  { value: "node", label: "Node.js" },
  { value: "python", label: "Python (requests)" },
  { value: "go", label: "Go" },
  { value: "php", label: "PHP" },
]

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
      {/* Fixed height + internal scrolling keeps long snippets inside the modal. */}
      <DialogContent className="flex max-h-[85vh] max-w-3xl flex-col gap-3 overflow-hidden">
        <DialogHeader>
          <DialogTitle>Generate code</DialogTitle>
          <DialogDescription>Runnable snippets that reflect the current request configuration.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap items-center gap-2 pr-6">
          <Select value={target} onValueChange={setTarget}>
            <SelectTrigger className="h-8 w-[200px]" aria-label="Code language">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TARGETS.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <div className="ml-auto flex items-center gap-1">
            <Button variant="ghost" size="sm" className="gap-1.5" onClick={copy} disabled={!code}>
              {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />} Copy
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="gap-1.5"
              disabled={!code}
              onClick={() => api.saveTextFile(`request.${target}`, code).catch(console.error)}
            >
              <Download className="h-3.5 w-3.5" /> Save
            </Button>
          </div>
        </div>

        {busy && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="h-3.5 w-3.5 animate-spin" /> Generating…
          </div>
        )}
        {error && (
          <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div>
        )}

        <pre
          className="min-h-0 flex-1 overflow-auto whitespace-pre rounded-md border border-border bg-muted/20 p-3 font-mono text-xs leading-relaxed"
          tabIndex={0}
        >
          {code || "—"}
        </pre>
      </DialogContent>
    </Dialog>
  )
}
