import React, { useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { AlertCircle, ClipboardPaste, Loader2 } from "lucide-react"
import * as api from "../lib/api"
import type { Request } from "@/types"

interface ImportCurlDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onOpen: (request: Request) => void
}

export function ImportCurlDialog({ open, onOpenChange, onOpen }: ImportCurlDialogProps) {
  const [command, setCommand] = useState("")
  const [preview, setPreview] = useState<Request | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const parse = async () => {
    setBusy(true)
    setError(null)
    setPreview(null)
    try {
      const request = await api.parseCurl(command)
      setPreview(request)
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const pasteFromClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText()
      if (text) setCommand(text)
    } catch (err) {
      setError("Clipboard access was denied; paste manually instead.")
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import cURL</DialogTitle>
          <DialogDescription>Paste a `curl …` command to convert it into an editable request.</DialogDescription>
        </DialogHeader>
        <textarea
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          spellCheck={false}
          placeholder={`curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{"name":"Ada"}'`}
          className="h-32 w-full resize-none rounded-md border border-border bg-background p-2 font-mono text-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
        {error && (
          <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
            <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" /> {error}
          </div>
        )}
        {preview && (
          <div className="space-y-1 rounded-md border border-border bg-muted/20 p-3 text-xs">
            <div className="font-mono"><span className="font-bold">{preview.method}</span> {preview.url}</div>
            <div className="text-muted-foreground">
              {preview.headers.filter((h) => h.enabled !== false).length} headers · {preview.queryParams.length} params · body: {preview.body.mode}
              {preview.auth.type !== "noauth" ? ` · auth: ${preview.auth.type}` : ""}
            </div>
          </div>
        )}
        <DialogFooter>
          <Button variant="outline" className="gap-1.5 mr-auto" onClick={pasteFromClipboard}>
            <ClipboardPaste className="h-3.5 w-3.5" /> Paste
          </Button>
          <Button variant="outline" onClick={parse} disabled={busy || !command.trim()}>
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Parse"}
          </Button>
          <Button
            disabled={!preview}
            onClick={() => {
              if (preview) onOpen(preview)
              onOpenChange(false)
            }}
          >
            Open in tab
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
