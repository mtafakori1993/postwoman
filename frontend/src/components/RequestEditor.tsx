import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import {
  AlertTriangle,
  ArrowDownUp,
  Braces,
  Code2,
  FileText,
  Info,
  Save,
  Send,
  X,
} from "lucide-react"
import type { AuthConfig, Header, KeyValuePair, QueryParam, Request } from "@/types"
import { KeyValueEditor } from "./KeyValueEditor"
import { BodyEditor } from "./BodyEditor"
import { AuthEditor } from "./AuthEditor"
import { MethodSelector } from "./MethodSelector"
import { VariableChips, VariableInput } from "./VariableInput"
import { SyntaxEditor } from "./SyntaxEditor"
import { cn } from "@/lib/utils"
import { composeUrl, missingPathVariableNames, paramsFromUrl, syncPathVariables } from "@/lib/url"
import { resolveTemplate, unresolvedPlaceholders, type ScopedVariable } from "@/lib/variables"

interface RequestEditorProps {
  request: Request | null
  onSend: () => void
  onSave: () => void
  onCancel?: () => void
  onChange: (request: Partial<Request>) => void
  isLoading: boolean
  environmentId?: string | null
  inheritedAuth?: AuthConfig
  inheritedAuthSource?: string
  variableScopes?: ScopedVariable[]
  focusUrlSignal?: number
  onOpenCodeExport?: () => void
}

const DEFAULT_SETTINGS: Request["settings"] = {
  followRedirects: true,
  followOriginalMethod: false,
  maxRedirects: 10,
  timeout: 30000,
  encoding: "auto",
}

export function RequestEditor({
  request,
  onSend,
  onSave,
  onCancel,
  onChange,
  isLoading,
  environmentId,
  inheritedAuth,
  inheritedAuthSource,
  variableScopes = [],
  focusUrlSignal = 0,
  onOpenCodeExport,
}: RequestEditorProps) {
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [url, setUrl] = useState("")
  const [method, setMethod] = useState("GET")
  const [activeTab, setActiveTab] = useState("params")
  const [headers, setHeaders] = useState<Header[]>([])
  const [queryParams, setQueryParams] = useState<QueryParam[]>([])
  const [pathVariables, setPathVariables] = useState<KeyValuePair[]>([])
  const [body, setBody] = useState<Request["body"]>({ mode: "none" })
  const [auth, setAuth] = useState<AuthConfig>({ type: "inherit" })
  const [scripts, setScripts] = useState<Request["scripts"]>([])
  const [settings, setSettings] = useState<Request["settings"]>(DEFAULT_SETTINGS)
  const [showDocs, setShowDocs] = useState(false)

  // Sync with the request prop. The parent updates the store on every change,
  // so these values always mirror the active tab.
  useEffect(() => {
    if (!request) return
    setName(request.name || "")
    setDescription(request.description || "")
    setUrl(request.url || "")
    setMethod(request.method || "GET")
    setHeaders(request.headers || [])
    setQueryParams(request.queryParams || [])
    setPathVariables(request.pathVariables || [])
    setBody(request.body || { mode: "none" })
    setAuth(request.auth || { type: "inherit" })
    setScripts(request.scripts || [])
    setSettings(request.settings || DEFAULT_SETTINGS)
  }, [request])

  // Ctrl+L focuses the URL bar.
  useEffect(() => {
    if (!focusUrlSignal) return
    const input = document.getElementById("pw-url-input") as HTMLInputElement | null
    input?.focus()
    input?.select()
  }, [focusUrlSignal])

  const handleUrlChange = (next: string) => {
    setUrl(next)
    // Keep the Params and Path Variables tables in step with the URL.
    const nextParams = paramsFromUrl(next, queryParams)
    const nextPathVars = syncPathVariables(next, pathVariables)
    setQueryParams(nextParams)
    setPathVariables(nextPathVars)
    onChange({ url: next, queryParams: nextParams, pathVariables: nextPathVars })
  }

  const handleQueryParamsChange = (next: QueryParam[]) => {
    setQueryParams(next)
    const nextUrl = composeUrl(url, next)
    setUrl(nextUrl)
    onChange({ queryParams: next, url: nextUrl })
  }

  const importParamsFromUrl = () => {
    const next = paramsFromUrl(url, queryParams)
    setQueryParams(next)
    onChange({ queryParams: next, url: composeUrl(url, next) })
  }

  const handlePathVariablesChange = (next: KeyValuePair[]) => {
    setPathVariables(next)
    onChange({ pathVariables: next })
  }

  const handleScriptsChange = (next: Request["scripts"]) => {
    setScripts(next)
    onChange({ scripts: next })
  }

  const handleSettingsChange = (patch: Partial<Request["settings"]>) => {
    const updated = { ...settings, ...patch }
    setSettings(updated)
    onChange({ settings: updated })
  }

  const resolvedUrl = useMemo(() => resolveTemplate(url, variableScopes), [url, variableScopes])

  const warnings = useMemo(() => {
    const out: string[] = []
    const missingPath = missingPathVariableNames(url, pathVariables)
    if (missingPath.length > 0) {
      out.push(`Path variables without a value: ${missingPath.map((v) => `{{${v}}}`).join(", ")}`)
    }
    const unresolved = unresolvedPlaceholders(url, variableScopes)
    if (unresolved.length > 0) {
      out.push(`Undefined variables: ${unresolved.map((v) => `{{${v}}}`).join(", ")}`)
    }
    return out
  }, [url, pathVariables, variableScopes])

  const hasPathVariables = pathVariables.length > 0

  return (
    <div className="flex flex-col h-full bg-card">
      {/* Name + description ------------------------------------------------- */}
      <div className="flex items-center gap-2 px-3 pt-2">
        <Input
          value={name}
          onChange={(e) => {
            setName(e.target.value)
            onChange({ name: e.target.value })
          }}
          placeholder="Request name"
          aria-label="Request name"
          className="h-8 max-w-sm text-sm font-medium"
        />
        <Button
          variant="ghost"
          size="sm"
          className={cn("h-8 gap-1.5 text-xs", showDocs && "bg-muted")}
          onClick={() => setShowDocs((v) => !v)}
          title="Request documentation and notes"
        >
          <FileText className="h-3.5 w-3.5" />
          Docs
        </Button>
        {onOpenCodeExport && (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 gap-1.5 text-xs ml-auto"
            onClick={onOpenCodeExport}
            title="Generate code snippet (Ctrl+Shift+E)"
          >
            <Code2 className="h-3.5 w-3.5" />
            Code
          </Button>
        )}
      </div>

      {showDocs && (
        <div className="px-3 pt-2">
          <Textarea
            value={description}
            onChange={(e) => {
              setDescription(e.target.value)
              onChange({ description: e.target.value })
            }}
            placeholder="Describe this request: purpose, auth notes, expected responses…"
            className="min-h-[72px] text-sm"
          />
        </div>
      )}

      {/* Request line ------------------------------------------------------- */}
      <div className="flex flex-wrap items-start gap-2 p-3 border-b border-border bg-muted/30">
        <MethodSelector value={method} onChange={(next) => { setMethod(next); onChange({ method: next }) }} />

        <div className="flex-1 min-w-[240px] space-y-1">
          <VariableInput
            id="pw-url-input"
            value={url}
            onChange={handleUrlChange}
            placeholder="https://api.example.com/users/{{userId}}"
            aria-label="Request URL"
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                e.preventDefault()
                onSend()
              }
            }}
          />
          <VariableChips text={url} />
          {resolvedUrl !== url && (
            <div className="truncate font-mono pw-text-2xs text-muted-foreground" title={resolvedUrl}>
              → {resolvedUrl}
            </div>
          )}
        </div>

        {isLoading ? (
          <Button onClick={onCancel} variant="destructive" className="h-9 px-6 gap-2">
            <X className="h-4 w-4" />
            <span>Cancel</span>
          </Button>
        ) : (
          <Button
            onClick={onSend}
            className="pw-btn-primary h-9 px-6 gap-2"
            style={{ backgroundColor: "hsl(var(--pw-orange))", borderColor: "hsl(var(--pw-orange))" }}
            title="Send (Ctrl+Enter)"
          >
            <Send className="h-4 w-4" />
            <span>Send</span>
          </Button>
        )}

        <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onSave} aria-label="Save (Ctrl+S)" title="Save (Ctrl+S)">
          <Save className="h-4 w-4" />
        </Button>
      </div>

      {warnings.length > 0 && (
        <div className="flex flex-col gap-1 border-b border-amber-500/30 bg-amber-500/10 px-3 py-1.5">
          {warnings.map((warning) => (
            <div key={warning} className="flex items-center gap-2 pw-text-2xs text-amber-700 dark:text-amber-400">
              <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
              {warning}
            </div>
          ))}
        </div>
      )}

      {/* Tabs -------------------------------------------------------------- */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="border-b border-border bg-muted/30">
        <TabsList className="p-1 bg-transparent overflow-x-auto flex-nowrap justify-start w-full">
          <TabsTrigger value="params">
            Params
            {queryParams.length > 0 && <span className="ml-1 pw-text-3xs text-muted-foreground">{queryParams.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="auth">
            Auth
            {auth.type && auth.type !== "noauth" && auth.type !== "inherit" && (
              <span className="ml-1 rounded bg-primary/15 px-1 pw-text-4xs font-semibold uppercase text-primary">{auth.type}</span>
            )}
          </TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            {headers.length > 0 && <span className="ml-1 pw-text-3xs text-muted-foreground">{headers.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="body">
            Body
            {body.mode && body.mode !== "none" && (
              <span className="ml-1 rounded bg-primary/15 px-1 pw-text-4xs font-semibold uppercase text-primary">
                {body.mode === "formdata" ? "form" : body.mode === "urlencoded" ? "url" : body.mode}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="scripts">
            Scripts
            {scripts.some((s) => (s.script || "").trim()) && (
              <span className="ml-1 h-1.5 w-1.5 rounded-full bg-primary" aria-hidden />
            )}
          </TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        {/* Params --------------------------------------------------------- */}
        <TabsContent value="params" className="p-3 space-y-4">
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-2">
              <Label className="text-xs font-medium text-muted-foreground">Query Parameters</Label>
              <span className="rounded bg-muted px-1.5 py-0.5 pw-text-3xs text-muted-foreground">
                synced with URL
              </span>
              <Button variant="ghost" size="sm" className="ml-auto h-7 gap-1.5 pw-text-2xs" onClick={importParamsFromUrl} title="Re-parse the query string from the URL">
                <ArrowDownUp className="h-3.5 w-3.5" />
                Import from URL
              </Button>
            </div>
            <KeyValueEditor
              items={queryParams}
              onChange={handleQueryParamsChange}
              variableAutocomplete
              columns={[
                { key: "enabled", header: "", width: "40px", type: "checkbox" },
                { key: "key", header: "Key", width: "30%" },
                { key: "value", header: "Value", width: "50%" },
                { key: "description", header: "Description", width: "20%" },
              ]}
              emptyMessage="No query parameters. Type them in the URL or add a row."
            />
          </div>

          <div>
            <div className="mb-2 flex items-center gap-2">
              <Label className="text-xs font-medium text-muted-foreground">Path Variables</Label>
              {hasPathVariables ? (
                <span className="rounded bg-muted px-1.5 py-0.5 pw-text-3xs text-muted-foreground">
                  detected in URL
                </span>
              ) : (
                <span className="pw-text-2xs text-muted-foreground">
                  Add <code>{"/:id"}</code> or <code>{"{{id}}"}</code> to the URL path and it appears here.
                </span>
              )}
            </div>
            {hasPathVariables ? (
              <KeyValueEditor
                items={pathVariables}
                onChange={handlePathVariablesChange}
                variableAutocomplete
                columns={[
                  { key: "key", header: "Variable", width: "26%" },
                  { key: "value", header: "Value", width: "54%" },
                  { key: "description", header: "Description", width: "20%" },
                ]}
                emptyMessage="No path variables detected."
              />
            ) : (
              <div className="flex items-center gap-2 rounded-md border border-dashed border-border bg-muted/10 px-3 py-3 text-xs text-muted-foreground">
                <Info className="h-3.5 w-3.5 shrink-0" />
                Path variables are detected automatically from the URL path.
              </div>
            )}
          </div>
        </TabsContent>

        {/* Auth ----------------------------------------------------------- */}
        <TabsContent value="auth" className="p-3">
          <AuthEditor
            auth={auth}
            onChange={(next) => { setAuth(next); onChange({ auth: next }) }}
            inheritedAuth={inheritedAuth}
            inheritedSource={inheritedAuthSource}
            request={request}
            environmentId={environmentId}
          />
        </TabsContent>

        {/* Headers -------------------------------------------------------- */}
        <TabsContent value="headers" className="p-3">
          <KeyValueEditor
            items={headers}
            onChange={(next) => { setHeaders(next); onChange({ headers: next }) }}
            variableAutocomplete
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Key", width: "30%" },
              { key: "value", header: "Value", width: "50%" },
              { key: "description", header: "Description", width: "20%" },
            ]}
            emptyMessage="No headers. Add one, or set default headers in Settings."
          />
        </TabsContent>

        {/* Body ----------------------------------------------------------- */}
        <TabsContent value="body" className="p-3">
          <BodyEditor
            body={body}
            onChange={(next) => { setBody(next); onChange({ body: next }) }}
            method={method}
            headers={headers}
            onHeadersChange={(next) => { setHeaders(next); onChange({ headers: next }) }}
            request={request}
            environmentId={environmentId}
          />
        </TabsContent>

        {/* Scripts -------------------------------------------------------- */}
        <TabsContent value="scripts" className="p-3 space-y-4">
          <div>
            <div className="mb-1 flex items-center gap-2">
              <Label className="text-xs font-medium text-muted-foreground">Pre-request script</Label>
              <span className="pw-text-2xs text-muted-foreground">runs before the request is sent</span>
            </div>
            <div className="h-[200px] overflow-hidden rounded-md border border-border">
              <SyntaxEditor
                value={scripts.find((s) => s.type === "prerequest")?.script || ""}
                onChange={(value) =>
                  handleScriptsChange([
                    ...scripts.filter((s) => s.type !== "prerequest"),
                    { type: "prerequest", script: value },
                  ])
                }
                language="javascript"
                ariaLabel="Pre-request script"
                placeholder={"// pm.environment.set('timestamp', Date.now().toString());"}
                minHeight="100%"
              />
            </div>
          </div>
          <div>
            <div className="mb-1 flex items-center gap-2">
              <Label className="text-xs font-medium text-muted-foreground">Test script</Label>
              <span className="pw-text-2xs text-muted-foreground">runs after the response arrives</span>
            </div>
            <div className="h-[200px] overflow-hidden rounded-md border border-border">
              <SyntaxEditor
                value={scripts.find((s) => s.type === "test")?.script || ""}
                onChange={(value) =>
                  handleScriptsChange([
                    ...scripts.filter((s) => s.type !== "test"),
                    { type: "test", script: value },
                  ])
                }
                language="javascript"
                ariaLabel="Test script"
                placeholder={"pm.test('Status is 200', () => pm.response.to.have.status(200));"}
                minHeight="100%"
              />
            </div>
            <p className="mt-1 pw-text-2xs text-muted-foreground">
              Tests are listed in the response&apos;s Tests tab. Use <code>pm.test</code>, <code>pm.response.to.have.status</code> and the
              <code> expect(...)</code> helpers.
            </p>
          </div>
        </TabsContent>

        {/* Settings ------------------------------------------------------- */}
        <TabsContent value="settings" className="p-3 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <label className="flex items-center justify-between gap-2 text-sm">
              <span>Follow redirects</span>
              <Checkbox checked={settings.followRedirects} onCheckedChange={(c) => handleSettingsChange({ followRedirects: c === true })} />
            </label>
            <label className="flex items-center justify-between gap-2 text-sm">
              <span>Follow original method</span>
              <Checkbox checked={settings.followOriginalMethod} onCheckedChange={(c) => handleSettingsChange({ followOriginalMethod: c === true })} />
            </label>
            <div>
              <Label htmlFor="maxRedirects" className="text-xs">Max redirects</Label>
              <Input
                id="maxRedirects"
                type="number"
                value={settings.maxRedirects}
                onChange={(e) => handleSettingsChange({ maxRedirects: parseInt(e.target.value) || 10 })}
                className="mt-1 h-8"
              />
            </div>
            <div>
              <Label htmlFor="timeout" className="text-xs">Timeout (ms)</Label>
              <Input
                id="timeout"
                type="number"
                value={settings.timeout}
                onChange={(e) => handleSettingsChange({ timeout: parseInt(e.target.value) || 30000 })}
                className="mt-1 h-8"
              />
            </div>
            <div>
              <Label className="text-xs">Response encoding</Label>
              <Select value={settings.encoding || "auto"} onValueChange={(value) => handleSettingsChange({ encoding: value })}>
                <SelectTrigger className="mt-1 h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="auto">Auto</SelectItem>
                  <SelectItem value="utf8">UTF-8</SelectItem>
                  <SelectItem value="latin1">Latin-1</SelectItem>
                  <SelectItem value="binary">Binary</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <label className="flex items-center justify-between gap-2 text-sm">
              <span>Verify TLS certificate</span>
              <Checkbox
                checked={settings.tls?.rejectUnauthorized !== false}
                onCheckedChange={(c) => handleSettingsChange({ tls: { ...(settings.tls || {}), rejectUnauthorized: c === true } })}
              />
            </label>
          </div>

          <div className="rounded-md border border-border p-3">
            <div className="mb-2 flex items-center gap-2">
              <ArrowDownUp className="h-3.5 w-3.5 text-muted-foreground" />
              <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">Proxy (this request)</span>
            </div>
            <div className="grid grid-cols-4 gap-2">
              <Input
                placeholder="Host"
                value={settings.proxy?.host || ""}
                onChange={(e) => handleSettingsChange({ proxy: { ...(settings.proxy || { port: 8080, protocol: "http" }), host: e.target.value } })}
                className="h-8"
              />
              <Input
                type="number"
                placeholder="Port"
                value={settings.proxy?.port ?? ""}
                onChange={(e) => handleSettingsChange({ proxy: { ...(settings.proxy || { host: "", protocol: "http" }), port: parseInt(e.target.value) || 0 } })}
                className="h-8"
              />
              <Select
                value={settings.proxy?.protocol || "http"}
                onValueChange={(value) => handleSettingsChange({ proxy: { ...(settings.proxy || { host: "", port: 8080 }), protocol: value } })}
              >
                <SelectTrigger className="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="http">HTTP</SelectItem>
                  <SelectItem value="https">HTTPS</SelectItem>
                  <SelectItem value="socks5">SOCKS5</SelectItem>
                </SelectContent>
              </Select>
              <Input
                placeholder="Bypass (host1,host2)"
                value={(settings.proxy?.bypass || []).join(",")}
                onChange={(e) =>
                  handleSettingsChange({
                    proxy: {
                      ...(settings.proxy || { host: "", port: 8080, protocol: "http" }),
                      bypass: e.target.value.split(",").map((v) => v.trim()).filter(Boolean),
                    },
                  })
                }
                className="h-8"
              />
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
