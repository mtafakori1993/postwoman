import React, { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  AlignLeft,
  CheckCircle2,
  Code,
  Copy,
  Download,
  Eye,
  FileCode2,
  FileText,
  Image as ImageIcon,
  Info,
  Search,
  Sparkles,
  Terminal,
  Wrench,
  X,
  XCircle,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { Response, Cookie, Header, ResponseViewMode, Request, HistoryEntry } from "@/types"
import { JSONViewer } from "./JSONViewer"
import { SyntaxEditor } from "./SyntaxEditor"
import { ResponseTools } from "./ResponseTools"
import { AIAssistant } from "./AIAssistant"
import { detectLanguage, extForContentType, formatContent, prettyXML } from "@/lib/format"
import * as api from "../lib/api"

interface ResponseViewerProps {
  response: Response | null
  requestHeaders?: Header[]
  request?: Request | null
  environmentId?: string | null
  history?: HistoryEntry[]
  height: number
  onHeightChange: (height: number) => void
}

const WRAP_KEY = "postwoman.responseWrap"

export function ResponseViewer({ response, requestHeaders, request, environmentId, history = [], height, onHeightChange }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<"body" | "cookies" | "headers" | "meta" | "tests" | "console">("body")
  const [viewMode, setViewMode] = useState<ResponseViewMode>("pretty")
  const [searchQuery, setSearchQuery] = useState("")
  const [showSearch, setShowSearch] = useState(false)
  const [copied, setCopied] = useState(false)
  const [copiedHeaders, setCopiedHeaders] = useState(false)
  const [toolsOpen, setToolsOpen] = useState(false)
  const [aiOpen, setAiOpen] = useState(false)
  const [wrap, setWrap] = useState(() => {
    try {
      const stored = localStorage.getItem(WRAP_KEY)
      return stored === null ? true : stored === "true"
    } catch {
      return true
    }
  })

  const toggleWrap = () => {
    setWrap((value) => {
      try {
        localStorage.setItem(WRAP_KEY, String(!value))
      } catch {
        /* best effort */
      }
      return !value
    })
  }

  const handleResize = (e: React.MouseEvent) => {
    e.preventDefault()
    const startY = e.clientY
    const startHeight = height
    const handleMouseMove = (moveEvent: MouseEvent) => {
      const newHeight = Math.max(120, Math.min(window.innerHeight - 200, startHeight + (startY - moveEvent.clientY)))
      onHeightChange(newHeight)
    }
    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
    }
    document.addEventListener("mousemove", handleMouseMove)
    document.addEventListener("mouseup", handleMouseUp)
  }

  const tests = response?.tests ?? []
  const consoleEntries = response?.console ?? []
  const passed = tests.filter((t) => t.passed).length
  const failed = tests.length - passed

  const bodyText = response?.body ?? ""
  const contentType = response?.contentType ?? ""
  const language = useMemo(() => detectLanguage(contentType, bodyText), [contentType, bodyText])
  const formattedBody = useMemo(() => {
    if (!bodyText || viewMode !== "pretty" || language === "json") return ""
    return formatContent(bodyText, language).output
  }, [bodyText, language, viewMode])

  if (!response) {
    return (
      <div className="flex flex-col h-full bg-card">
        <div className="h-1 w-full bg-border hover:bg-primary/50 cursor-row-resize transition-colors" onMouseDown={handleResize} />
        <div className="flex-1 flex items-center justify-center text-muted-foreground">
          <div className="text-center">
            <Code className="h-12 w-12 mx-auto mb-4 opacity-30" />
            <p className="text-lg">No response yet</p>
            <p className="text-sm">Send a request to see the response</p>
          </div>
        </div>
      </div>
    )
  }

  const statusColor =
    response.error || response.status === 0
      ? "text-red-600 dark:text-red-400"
      : response.status >= 200 && response.status < 300
        ? "text-green-600 dark:text-green-400"
        : response.status >= 300 && response.status < 400
          ? "text-yellow-600 dark:text-yellow-400"
          : "text-red-600 dark:text-red-400"

  const handleCopy = async () => {
    try {
      await api.copyToClipboard(response.body || response.bodyBase64 || "")
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch (err) {
      console.error(err)
    }
  }

  const handleCopyHeaders = async () => {
    try {
      const text = (response?.headers ?? []).map((h) => `${h.key}: ${h.value}`).join("\n")
      await api.copyToClipboard(text)
      setCopiedHeaders(true)
      setTimeout(() => setCopiedHeaders(false), 1500)
    } catch (err) {
      console.error(err)
    }
  }

  const handleSave = async () => {
    try {
      if (response.isBinary && response.bodyBase64) {
        const blob = base64ToBlob(response.bodyBase64, contentType || "application/octet-stream")
        const url = URL.createObjectURL(blob)
        const anchor = document.createElement("a")
        anchor.href = url
        anchor.download = `response.${extForContentType(contentType)}`
        anchor.click()
        setTimeout(() => URL.revokeObjectURL(url), 1000)
        return
      }
      const ext = extForContentType(contentType)
      await api.saveTextFile(`response.${ext}`, response.body)
    } catch (err) {
      console.error(err)
    }
  }

  return (
    <div className="flex flex-col h-full bg-card">
      <div className="h-1 w-full bg-border hover:bg-primary/50 cursor-row-resize transition-colors" onMouseDown={handleResize} />

      {/* Header */}
      <div className="flex items-center justify-between gap-2 p-2 border-b border-border bg-muted/30 flex-wrap">
        <div className="flex items-center gap-3 flex-wrap">
          <span className={cn("font-mono text-base font-semibold", statusColor)}>
            {response.error ? "Error" : `${response.status} ${response.statusText}`}
          </span>
          <span className="text-xs text-muted-foreground font-mono">{formatDuration(response.responseTime)}</span>
          <span className="text-xs text-muted-foreground font-mono">{formatBytes(response.bodySize)}</span>
          {response.redirected && <Badge variant="secondary" className="text-xs">Redirected</Badge>}
          {tests.length > 0 && (
            <Badge variant={failed > 0 ? "destructive" : "secondary"} className="text-xs">
              {passed}/{tests.length} tests
            </Badge>
          )}
          {response.error && (
            <Badge variant="destructive" className="text-xs max-w-[420px] truncate" title={response.error}>
              {response.error}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-1">
          {activeTab === "body" && !response.error && (
            <Button
              variant="ghost"
              size="icon"
              className={cn("h-8 w-8", !wrap && "text-muted-foreground")}
              onClick={toggleWrap}
              aria-label={wrap ? "Disable text wrapping" : "Enable text wrapping"}
              title={wrap ? "Disable text wrapping" : "Enable text wrapping"}
            >
              <AlignLeft className="h-4 w-4" />
            </Button>
          )}
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={handleCopyHeaders}
            aria-label="Copy response headers"
            title="Copy response headers"
            disabled={!response.headers?.length}
          >
            <Copy className={cn("h-4 w-4", copiedHeaders && "text-green-500")} />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => setToolsOpen(true)}
            aria-label="Response tools"
            title="Extract, compare and validate"
            disabled={!response.body}
          >
            <Wrench className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => setAiOpen(true)}
            aria-label="AI assistant"
            title="AI assistant"
          >
            <Sparkles className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setShowSearch((v) => !v)} aria-label="Search in response">
            <Search className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleSave} aria-label="Save response" title="Save response">
            <Download className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleCopy} aria-label="Copy response" title="Copy response">
            <Copy className={cn("h-4 w-4", copied && "text-green-500")} />
          </Button>
        </div>
      </div>

      {showSearch && (
        <div className="flex items-center gap-2 p-2 border-b border-border bg-muted/30">
          <Input
            type="search"
            placeholder="Search in response..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="flex-1 max-w-md h-8"
            autoFocus
          />
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setShowSearch(false); setSearchQuery("") }}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      )}

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as typeof activeTab)} className="flex-1 flex flex-col min-h-0">
        <TabsList className="p-1 bg-muted/30 border-b border-border justify-start overflow-x-auto flex-nowrap shrink-0">
          <TabsTrigger value="body">Body</TabsTrigger>
          <TabsTrigger value="cookies">Cookies{response.cookies?.length ? ` (${response.cookies.length})` : ""}</TabsTrigger>
          <TabsTrigger value="headers">Headers{response.headers?.length ? ` (${response.headers.length})` : ""}</TabsTrigger>
          <TabsTrigger value="meta">Meta</TabsTrigger>
          <TabsTrigger value="tests">Tests{tests.length ? ` (${passed}/${tests.length})` : ""}</TabsTrigger>
          <TabsTrigger value="console">Console{consoleEntries.length ? ` (${consoleEntries.length})` : ""}</TabsTrigger>
        </TabsList>

        <TabsContent value="body" className="flex-1 min-h-0 m-0">
          {response.error ? (
            <div className="p-4 text-sm text-destructive font-mono">{response.error}</div>
          ) : viewMode === "pretty" ? (
            <PrettyBody
              response={response}
              language={language}
              formattedBody={formattedBody}
              searchQuery={searchQuery}
              wrap={wrap}
            />
          ) : viewMode === "raw" ? (
            <RawBody response={response} searchQuery={searchQuery} wrap={wrap} />
          ) : (
            <PreviewBody response={response} language={language} wrap={wrap} />
          )}
        </TabsContent>

        <TabsContent value="meta" className="flex-1 min-h-0 m-0">
          <ResponseMeta response={response} request={request} requestHeaders={requestHeaders} />
        </TabsContent>

        <TabsContent value="cookies" className="flex-1 min-h-0 m-0">
          <CookieTable cookies={response.cookies ?? []} />
        </TabsContent>

        <TabsContent value="headers" className="flex-1 min-h-0 m-0">
          <HeaderTable headers={response.headers ?? []} />
        </TabsContent>

        <TabsContent value="tests" className="flex-1 min-h-0 m-0">
          <TestResults tests={tests} />
        </TabsContent>

        <TabsContent value="console" className="flex-1 min-h-0 m-0">
          <ConsoleOutput entries={consoleEntries} />
        </TabsContent>
      </Tabs>

      {activeTab === "body" && !response.error && (
        <ResponseViewTabs value={viewMode} onChange={setViewMode} />
      )}

      <ResponseTools open={toolsOpen} onOpenChange={setToolsOpen} response={response} request={request ?? null} history={history} />
      <AIAssistant open={aiOpen} onOpenChange={setAiOpen} request={request ?? null} response={response} />
    </div>
  )
}

function ResponseMeta({
  response,
  request,
  requestHeaders,
}: {
  response: Response
  request?: Request | null
  requestHeaders?: Header[]
}) {
  const rows: { label: string; value: string }[] = [
    { label: "Request method", value: request?.method || "—" },
    { label: "Request URL", value: request?.url || response.url || "—" },
    { label: "Final URL", value: response.url || "—" },
    { label: "Status", value: `${response.status} ${response.statusText}` },
    { label: "Content-Type", value: response.contentType || "—" },
    { label: "Response time", value: response.responseTime ? `${response.responseTime} ms` : "—" },
    { label: "Body size", value: `${response.bodySize.toLocaleString()} bytes` },
    { label: "Header size", value: `${response.headerSize.toLocaleString()} bytes` },
    { label: "Redirected", value: response.redirected ? "yes" : "no" },
    { label: "Received at", value: response.timestamp ? new Date(response.timestamp).toLocaleString() : "—" },
  ]
  const enabledRequestHeaders = (requestHeaders ?? []).filter((h) => h.enabled !== false && h.key)
  return (
    <ScrollArea className="h-full">
      <div className="space-y-4 p-3">
        <section>
          <h3 className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            <Info className="h-3.5 w-3.5" /> Network information
          </h3>
          <div className="divide-y divide-border/40 rounded-md border border-border">
            {rows.map((row) => (
              <div key={row.label} className="flex gap-3 px-3 py-1.5 text-xs">
                <span className="w-40 shrink-0 text-muted-foreground">{row.label}</span>
                <span className="min-w-0 flex-1 break-all font-mono">{row.value}</span>
              </div>
            ))}
          </div>
          <p className="mt-1 pw-text-2xs text-muted-foreground">
            A detailed DNS/TCP/TLS timing breakdown is only available when the networking stack exposes it; this build reports
            total response time.
          </p>
        </section>

        <section>
          <h3 className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
            Request headers ({enabledRequestHeaders.length})
          </h3>
          {enabledRequestHeaders.length === 0 ? (
            <p className="text-xs text-muted-foreground">No request headers configured.</p>
          ) : (
            <table className="w-full text-xs">
              <tbody>
                {enabledRequestHeaders.map((header, index) => (
                  <tr key={`${header.key}-${index}`} className="border-b border-border/40">
                    <td className="py-1 pr-3 font-medium align-top">{header.key}</td>
                    <td className="py-1 font-mono break-all">{header.value}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </div>
    </ScrollArea>
  )
}

/** Bottom navigation for the response body views. Kept in a single,
 * non-wrapping row so it stays compact on narrow layouts. */
function ResponseViewTabs({
  value,
  onChange,
}: {
  value: ResponseViewMode
  onChange: (mode: ResponseViewMode) => void
}) {
  const views: { value: ResponseViewMode; label: string; icon: React.ReactNode }[] = [
    { value: "pretty", label: "Pretty", icon: <Sparkles className="h-3.5 w-3.5" /> },
    { value: "raw", label: "Raw", icon: <FileText className="h-3.5 w-3.5" /> },
    { value: "preview", label: "Preview", icon: <Eye className="h-3.5 w-3.5" /> },
  ]

  const handleKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return
    event.preventDefault()
    const index = views.findIndex((view) => view.value === value)
    const delta = event.key === "ArrowRight" ? 1 : -1
    const next = views[(index + delta + views.length) % views.length]
    onChange(next.value)
    const buttons = event.currentTarget.querySelectorAll<HTMLButtonElement>('[role="tab"]')
    buttons[(index + delta + views.length) % views.length]?.focus()
  }

  return (
    <div
      role="tablist"
      aria-label="Response body view"
      onKeyDown={handleKeyDown}
      className="flex h-9 shrink-0 items-center gap-1 overflow-x-auto border-t border-border bg-muted/30 px-2"
    >
      {views.map((view) => {
        const active = view.value === value
        return (
          <button
            key={view.value}
            role="tab"
            aria-selected={active}
            tabIndex={active ? 0 : -1}
            onClick={() => onChange(view.value)}
            className={cn(
              "inline-flex h-7 shrink-0 items-center gap-1.5 rounded px-2.5 text-xs font-medium transition-colors outline-none",
              "focus-visible:ring-1 focus-visible:ring-ring",
              active
                ? "border border-border bg-background text-foreground shadow-sm"
                : "border border-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
            )}
          >
            {view.icon}
            {view.label}
          </button>
        )
      })}
    </div>
  )
}

function PrettyBody({
  response,
  language,
  formattedBody,
  searchQuery,
  wrap,
}: {
  response: Response
  language: ReturnType<typeof detectLanguage>
  formattedBody: string
  searchQuery: string
  wrap: boolean
}) {
  if (response.isBinary) {
    return (
      <div className="flex h-full items-center justify-center p-6 text-center text-sm text-muted-foreground">
        <div>
          <ImageIcon className="mx-auto mb-2 h-8 w-8 opacity-40" />
          <p>Binary response ({formatBytes(response.bodySize)})</p>
          <p className="mt-1 text-xs">Use Preview to render it or the download button to save it.</p>
        </div>
      </div>
    )
  }
  if (!response.body) {
    return <EmptyBody />
  }
  if (language === "json") {
    return <JSONViewer json={response.body} searchQuery={searchQuery} onCopyPath={(p) => api.copyToClipboard(p)} />
  }
  if (language === "text") {
    return (
      <ScrollArea className="h-full">
        <HighlightedPre text={formattedBody} query={searchQuery} wrap={wrap} className="p-3" />
      </ScrollArea>
    )
  }
  return (
    <div className="h-full min-h-0">
      <SyntaxEditor
        value={formattedBody}
        language={language}
        readOnly
        lineNumbers
        wrap={wrap}
        height="100%"
        ariaLabel="Pretty response body"
      />
    </div>
  )
}

function RawBody({ response, searchQuery, wrap }: { response: Response; searchQuery: string; wrap: boolean }) {
  const raw = response.isBinary ? response.bodyBase64 ?? "" : response.body
  if (!raw) {
    return response.isBinary ? (
      <div className="flex h-full items-center justify-center p-6 text-center text-sm text-muted-foreground">
        <div>
          <FileCode2 className="mx-auto mb-2 h-8 w-8 opacity-40" />
          <p>Binary response ({formatBytes(response.bodySize)})</p>
          <p className="mt-1 text-xs">Use Preview to render it or the download button to save it.</p>
        </div>
      </div>
    ) : (
      <EmptyBody />
    )
  }
  return (
    <ScrollArea className="h-full">
      <HighlightedPre text={raw} query={searchQuery} wrap={wrap} className="p-3" />
    </ScrollArea>
  )
}

function PreviewBody({
  response,
  language,
  wrap,
}: {
  response: Response
  language: ReturnType<typeof detectLanguage>
  wrap: boolean
}) {
  const ct = (response.contentType || "").toLowerCase()
  const dataUrl = response.bodyBase64
    ? `data:${response.contentType || "application/octet-stream"};base64,${response.bodyBase64}`
    : ""

  if (ct.startsWith("image/") && dataUrl) {
    return (
      <div className="flex h-full items-center justify-center overflow-auto bg-muted/20 p-4">
        <img src={dataUrl} alt="Response preview" className="max-h-full max-w-full object-contain shadow-sm" />
      </div>
    )
  }

  if (ct.includes("application/pdf") && dataUrl) {
    return (
      <div className="h-full">
        <iframe title="Response preview" src={dataUrl} className="h-full w-full border-0 bg-white" />
      </div>
    )
  }

  if (ct.startsWith("audio/") && dataUrl) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <audio controls src={dataUrl} className="w-full max-w-xl" />
      </div>
    )
  }

  if (ct.startsWith("video/") && dataUrl) {
    return (
      <div className="flex h-full items-center justify-center overflow-auto bg-black p-2">
        <video controls src={dataUrl} className="max-h-full max-w-full" />
      </div>
    )
  }

  if (language === "html" || ct.includes("text/html")) {
    return (
      <div className="h-full border-t border-border">
        {/* No allow-same-origin: untrusted HTML must not access app storage. */}
        <iframe
          title="Response preview"
          srcDoc={response.body}
          className="h-full w-full border-0 bg-white"
          sandbox="allow-scripts allow-forms allow-popups"
        />
      </div>
    )
  }

  if (!response.body) {
    return response.isBinary ? (
      <div className="flex h-full items-center justify-center p-6 text-center text-sm text-muted-foreground">
        <div>
          <FileCode2 className="mx-auto mb-2 h-8 w-8 opacity-40" />
          <p>No preview available for this binary response.</p>
          <p className="mt-1 text-xs">Use the download button to save it.</p>
        </div>
      </div>
    ) : (
      <EmptyBody />
    )
  }

  if (language === "text") {
    return (
      <ScrollArea className="h-full">
        <pre className={cn("p-3 font-mono text-sm", wrap ? "whitespace-pre-wrap break-all" : "whitespace-pre")}>{response.body}</pre>
      </ScrollArea>
    )
  }

  // Structured but non-renderable payloads (XML, etc.) are shown pretty-printed.
  if (language === "xml") {
    return (
      <ScrollArea className="h-full">
        <pre className={cn("p-3 font-mono text-sm", wrap ? "whitespace-pre-wrap break-all" : "whitespace-pre")}>{prettyXML(response.body)}</pre>
      </ScrollArea>
    )
  }

  return (
    <div className="flex h-full items-center justify-center p-6 text-center text-sm text-muted-foreground">
      <div>
        <Eye className="mx-auto mb-2 h-8 w-8 opacity-40" />
        <p>Preview is not available for this content type.</p>
        <p className="mt-1 text-xs">Switch to Pretty or Raw to inspect the response.</p>
      </div>
    </div>
  )
}

function EmptyBody() {
  return (
    <div className="flex h-full items-center justify-center p-6 text-sm text-muted-foreground">
      The response body is empty.
    </div>
  )
}

function HighlightedPre({ text, query, wrap = true, className }: { text: string; query?: string; wrap?: boolean; className?: string }) {
  const content = useMemo(() => {
    if (!query?.trim()) return text
    try {
      const pattern = new RegExp(`(${escapeRegExp(query)})`, "ig")
      return text.split(pattern).map((part, index) =>
        part.toLowerCase() === query.toLowerCase() ? (
          <mark key={index} className="rounded bg-yellow-300/70 px-0.5 text-foreground dark:bg-yellow-500/40">
            {part}
          </mark>
        ) : (
          <React.Fragment key={index}>{part}</React.Fragment>
        ),
      )
    } catch {
      return text
    }
  }, [text, query])

  return (
    <pre className={cn("font-mono text-sm", wrap ? "whitespace-pre-wrap break-all" : "whitespace-pre", className)}>{content}</pre>
  )
}

function base64ToBlob(base64: string, contentType: string): Blob {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i)
  return new Blob([bytes], { type: contentType })
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

function CookieTable({ cookies }: { cookies: Cookie[] }) {
  if (cookies.length === 0) {
    return <div className="text-center text-muted-foreground py-8">No cookies in response</div>
  }
  return (
    <ScrollArea className="h-full">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border">
            <th className="pw-table-header">Name</th>
            <th className="pw-table-header">Value</th>
            <th className="pw-table-header">Domain</th>
            <th className="pw-table-header">Path</th>
            <th className="pw-table-header">Expires</th>
            <th className="pw-table-header">Secure</th>
            <th className="pw-table-header">HttpOnly</th>
            <th className="pw-table-header">SameSite</th>
          </tr>
        </thead>
        <tbody>
          {cookies.map((cookie, index) => (
            <tr key={`${cookie.name}-${index}`} className="border-b border-border/50">
              <td className="pw-table-cell font-mono">{cookie.name}</td>
              <td className="pw-table-cell font-mono truncate max-w-[220px]" title={cookie.value}>{cookie.value}</td>
              <td className="pw-table-cell">{cookie.domain || "—"}</td>
              <td className="pw-table-cell">{cookie.path || "—"}</td>
              <td className="pw-table-cell">{cookie.expires ? new Date(cookie.expires * 1000).toLocaleString() : "Session"}</td>
              <td className="pw-table-cell">{cookie.secure ? "Yes" : "No"}</td>
              <td className="pw-table-cell">{cookie.httpOnly ? "Yes" : "No"}</td>
              <td className="pw-table-cell">{cookie.sameSite || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </ScrollArea>
  )
}

function HeaderTable({ headers }: { headers: Header[] }) {
  if (headers.length === 0) {
    return <div className="text-center text-muted-foreground py-8">No headers in response</div>
  }
  return (
    <ScrollArea className="h-full">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-border">
            <th className="pw-table-header">Header</th>
            <th className="pw-table-header">Value</th>
          </tr>
        </thead>
        <tbody>
          {headers.map((h, i) => (
            <tr key={`${h.key}-${i}`} className="border-b border-border/50">
              <td className="pw-table-cell font-medium align-top">{h.key}</td>
              <td className="pw-table-cell font-mono break-all">{h.value}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </ScrollArea>
  )
}

function TestResults({ tests }: { tests: Response["tests"] }) {
  if (!tests || tests.length === 0) {
    return (
      <div className="text-center text-muted-foreground py-8">
        <p>No tests were run.</p>
        <p className="text-xs mt-1">Add a test script (e.g. <code>pm.test("Status is 200", () =&gt; pm.response.to.have.status(200))</code>) to the request.</p>
      </div>
    )
  }
  return (
    <ScrollArea className="h-full p-3">
      <div className="space-y-1">
        {tests.map((t, i) => (
          <div key={i} className="flex items-start gap-2 text-sm py-1 border-b border-border/40 last:border-0">
            {t.passed ? (
              <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400 mt-0.5 shrink-0" />
            ) : (
              <XCircle className="h-4 w-4 text-red-600 dark:text-red-400 mt-0.5 shrink-0" />
            )}
            <div className="min-w-0">
              <div className={cn("font-medium", t.passed ? "text-foreground" : "text-red-600 dark:text-red-400")}>{t.name}</div>
              {t.error && <div className="text-xs text-muted-foreground font-mono break-all">{t.error}</div>}
            </div>
          </div>
        ))}
      </div>
    </ScrollArea>
  )
}

function ConsoleOutput({ entries }: { entries: Response["console"] }) {
  if (!entries || entries.length === 0) {
    return <div className="text-center text-muted-foreground py-8">No console output</div>
  }
  return (
    <ScrollArea className="h-full p-3">
      <div className="font-mono text-xs space-y-0.5">
        {entries.map((e, i) => (
          <div
            key={i}
            className={cn(
              "flex gap-2",
              e.level === "error" && "text-red-600 dark:text-red-400",
              e.level === "warn" && "text-yellow-600 dark:text-yellow-400",
            )}
          >
            <Terminal className="h-3 w-3 mt-0.5 shrink-0 opacity-50" />
            <span className="break-all">{e.args?.join(" ")}</span>
          </div>
        ))}
      </div>
    </ScrollArea>
  )
}

function formatDuration(ms: number): string {
  if (!ms) return "—"
  if (ms < 1000) return `${ms} ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(2)} s`
  return `${(ms / 60000).toFixed(2)} min`
}

function formatBytes(bytes: number): string {
  if (!bytes) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}
