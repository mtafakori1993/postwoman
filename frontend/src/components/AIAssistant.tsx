import React, { useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Loader2, Send, Sparkles, ShieldAlert } from "lucide-react"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { Request, Response } from "@/types"

interface AIAssistantProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  request: Request | null
  response: Response | null
}

const PRESETS: { label: string; prompt: string }[] = [
  { label: "Explain this request", prompt: "Explain what this request does and what each part contributes." },
  { label: "Analyse the error", prompt: "Explain why this request failed and how to fix it." },
  { label: "Explain the response", prompt: "Summarise the response payload and describe its important fields." },
  { label: "Generate tests", prompt: "Write pm.test assertions for this response (status, schema and important fields)." },
  { label: "Generate JSON Schema", prompt: "Produce a JSON Schema (draft-07) that describes this response." },
  { label: "Suggest improvements", prompt: "Suggest improvements to this request (headers, auth, pagination, error handling)." },
]

export function AIAssistant({ open, onOpenChange, request, response }: AIAssistantProps) {
  const aiEnabled = useWorkspaceStore((s) => s.settings.aiEnabled)
  const [prompt, setPrompt] = useState("")
  const [output, setOutput] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const send = async (text: string) => {
    const finalPrompt = text.trim() || prompt.trim()
    if (!finalPrompt) return
    setBusy(true)
    setError(null)
    setOutput("")
    try {
      const result = await api.aiAssist({
        prompt: finalPrompt,
        requestJson: request ? JSON.stringify({ method: request.method, url: request.url, headers: request.headers, body: request.body, auth: { type: request.auth.type } }, null, 2) : "",
        responseBody: response?.body ? response.body.slice(0, 20000) : "",
      })
      setOutput(result)
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2"><Sparkles className="h-4 w-4" /> AI assistant</DialogTitle>
          <DialogDescription>
            Optional and opt-in. The configured provider receives the request configuration and response body only when you press Send.
          </DialogDescription>
        </DialogHeader>

        {!aiEnabled && (
          <div className="flex items-start gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400">
            <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0" />
            <span>The assistant is disabled. Enable it and add a provider API key in Settings → AI providers.</span>
          </div>
        )}

        <div className="flex flex-wrap gap-1.5">
          {PRESETS.map((preset) => (
            <Button key={preset.label} variant="outline" size="sm" className="h-7 pw-text-2xs" disabled={!aiEnabled || busy} onClick={() => { setPrompt(preset.prompt); void send(preset.prompt) }}>
              {preset.label}
            </Button>
          ))}
        </div>

        <div className="flex items-end gap-2">
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Ask anything about this request or response…"
            className="h-20 flex-1 resize-none rounded-md border border-border bg-background p-2 text-sm outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
          <Button className="gap-1.5" disabled={!aiEnabled || busy} onClick={() => void send(prompt)}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            Send
          </Button>
        </div>

        {error && <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div>}
        {output && <pre className="max-h-80 overflow-auto whitespace-pre-wrap rounded-md border border-border bg-muted/20 p-3 text-xs">{output}</pre>}
      </DialogContent>
    </Dialog>
  )
}
