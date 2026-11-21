import React, { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Download, Copy, Search, Code, FileText, Eye, X, CheckCircle2, XCircle, Terminal } from "lucide-react"
import { cn } from "@/lib/utils"
import { Response, Cookie, Header } from "@/types"
import { JSONViewer } from "./JSONViewer"
import * as api from "../lib/api"

interface ResponseViewerProps {
  response: Response | null
  requestHeaders?: Header[]
  height: number
  onHeightChange: (height: number) => void
}

export function ResponseViewer({ response, requestHeaders, height, onHeightChange }: ResponseViewerProps) {
  const [activeTab, setActiveTab] = useState<"body" | "cookies" | "headers" | "tests" | "console">("body")
  const [viewMode, setViewMode] = useState<"pretty" | "raw" | "preview">("pretty")
  const [searchQuery, setSearchQuery] = useState("")
  const [showSearch, setShowSearch] = useState(false)
  const [copied, setCopied] = useState(false)

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

  const bodyText = useMemo(() => response?.body ?? "", [response])
  const isJSON = useMemo(() => {
    if (!response) return false
    const ct = response.contentType || ""
    if (ct.includes("json")) return true
    const trimmed = bodyText.trim()
    return (trimmed.startsWith("{") && trimmed.endsWith("}")) || (trimmed.startsWith("[") && trimmed.endsWith("]"))
  }, [response, bodyText])

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
      await api.copyToClipboard(response.body)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch (err) {
      console.error(err)
    }
  }

  const handleSave = async () => {
    try {
      const ext = isJSON ? "json" : "txt"
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
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => setShowSearch((v) => !v)} aria-label="Search in response">
            <Search className="h-4 w-4" />
          </Button>
          <Select value={viewMode} onValueChange={(v) => setViewMode(v as typeof viewMode)}>
            <SelectTrigger className="w-[120px] h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pretty"><Eye className="h-4 w-4 mr-2" /> Pretty</SelectItem>
              <SelectItem value="raw"><FileText className="h-4 w-4 mr-2" /> Raw</SelectItem>
              <SelectItem value="preview"><Code className="h-4 w-4 mr-2" /> Preview</SelectItem>
            </SelectContent>
          </Select>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleSave} aria-label="Save response">
            <Download className="h-4 w-4" />
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={handleCopy} aria-label="Copy response">
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
          />
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => { setShowSearch(false); setSearchQuery("") }}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      )}

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as typeof activeTab)} className="flex-1 flex flex-col min-h-0">
        <TabsList className="p-1 bg-muted/30 border-b border-border justify-start">
          <TabsTrigger value="body">Body</TabsTrigger>
          <TabsTrigger value="cookies">Cookies{response.cookies?.length ? ` (${response.cookies.length})` : ""}</TabsTrigger>
          <TabsTrigger value="headers">Headers{response.headers?.length ? ` (${response.headers.length})` : ""}</TabsTrigger>
          <TabsTrigger value="tests">Tests{tests.length ? ` (${passed}/${tests.length})` : ""}</TabsTrigger>
          <TabsTrigger value="console">Console{consoleEntries.length ? ` (${consoleEntries.length})` : ""}</TabsTrigger>
        </TabsList>

        <TabsContent value="body" className="flex-1 min-h-0 m-0">
          {response.error ? (
            <div className="p-4 text-sm text-destructive font-mono">{response.error}</div>
          ) : response.isBinary ? (
            <div className="p-4 text-sm text-muted-foreground">
              Binary response ({formatBytes(response.bodySize)}) — use the download button to save it.
            </div>
          ) : viewMode === "pretty" ? (
            isJSON ? (
              <JSONViewer json={response.body} searchQuery={searchQuery} onCopyPath={(p) => api.copyToClipboard(p)} />
            ) : (
              <ScrollArea className="h-full p-3">
                <pre className="font-mono text-sm whitespace-pre-wrap break-all">{response.body}</pre>
              </ScrollArea>
            )
          ) : viewMode === "raw" ? (
            <ScrollArea className="h-full p-3">
              <pre className="font-mono text-sm whitespace-pre-wrap break-all">{response.body}</pre>
            </ScrollArea>
          ) : (
            <div className="h-full border-t border-border">
              {/* No allow-same-origin: untrusted HTML must not access app storage. */}
              <iframe title="Response preview" srcDoc={response.body} className="w-full h-full bg-white" sandbox="allow-scripts" />
            </div>
          )}
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
    </div>
  )
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
