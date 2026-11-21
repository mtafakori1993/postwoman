import React, { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Send, Save, X } from "lucide-react"
import { Request, Header, QueryParam, KeyValuePair, AuthConfig } from "@/types"
import { KeyValueEditor } from "./KeyValueEditor"
import { BodyEditor } from "./BodyEditor"
import { AuthEditor } from "./AuthEditor"

interface RequestEditorProps {
  request: Request | null
  onSend: () => void
  onSave: () => void
  onCancel?: () => void
  onChange: (request: Partial<Request>) => void
  isLoading: boolean
}

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"]

export function RequestEditor({ request, onSend, onSave, onCancel, onChange, isLoading }: RequestEditorProps) {
  const [url, setUrl] = useState("")
  const [method, setMethod] = useState<Request["method"]>("GET")
  const [activeTab, setActiveTab] = useState("params")
  const [headers, setHeaders] = useState<Header[]>([])
  const [queryParams, setQueryParams] = useState<QueryParam[]>([])
  const [pathVariables, setPathVariables] = useState<KeyValuePair[]>([])
  const [body, setBody] = useState<Request["body"]>({ mode: "none" })
  const [auth, setAuth] = useState<AuthConfig>({ type: "noauth" })
  const [scripts, setScripts] = useState<Request["scripts"]>([])
  const [settings, setSettings] = useState<Request["settings"]>({
    followRedirects: true,
    followOriginalMethod: false,
    maxRedirects: 10,
    timeout: 30000,
    encoding: "auto",
  })

  // Sync with request prop
  useEffect(() => {
    if (request) {
      setUrl(request.url || "")
      setMethod(request.method || "GET")
      setHeaders(request.headers || [])
      setQueryParams(request.queryParams || [])
      setPathVariables(request.pathVariables || [])
      setBody(request.body || { mode: "none" })
      setAuth(request.auth || { type: "noauth" })
      setScripts(request.scripts || [])
      setSettings(request.settings || {
        followRedirects: true,
        followOriginalMethod: false,
        maxRedirects: 10,
        timeout: 30000,
        encoding: "auto",
      })
    }
  }, [request])

  const handleUrlChange = (newUrl: string) => {
    setUrl(newUrl)
    onChange({ url: newUrl })
  }

  const handleMethodChange = (newMethod: Request["method"]) => {
    setMethod(newMethod)
    onChange({ method: newMethod })
  }

  const handleHeadersChange = (newHeaders: Header[]) => {
    setHeaders(newHeaders)
    onChange({ headers: newHeaders })
  }

  const handleQueryParamsChange = (newParams: QueryParam[]) => {
    setQueryParams(newParams)
    onChange({ queryParams: newParams })
  }

  const handlePathVariablesChange = (newVars: KeyValuePair[]) => {
    setPathVariables(newVars)
    onChange({ pathVariables: newVars })
  }

  const handleBodyChange = (newBody: Request["body"]) => {
    setBody(newBody)
    onChange({ body: newBody })
  }

  const handleAuthChange = (newAuth: AuthConfig) => {
    setAuth(newAuth)
    onChange({ auth: newAuth })
  }

  const handleScriptsChange = (newScripts: Request["scripts"]) => {
    setScripts(newScripts)
    onChange({ scripts: newScripts })
  }

  const handleSettingsChange = (newSettings: Partial<Request["settings"]>) => {
    const updated = { ...settings, ...newSettings }
    setSettings(updated)
    onChange({ settings: updated })
  }

  return (
    <div className="flex flex-col h-full bg-card">
      {/* Request Line */}
      <div className="flex flex-wrap items-center gap-2 p-3 border-b border-border bg-muted/30">
        {/* Method Selector */}
        <Select value={method} onValueChange={handleMethodChange}>
          <SelectTrigger className="w-24 h-9">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {HTTP_METHODS.map(m => (
              <SelectItem key={m} value={m}>{m}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        {/* URL Input */}
        <div className="flex-1 relative">
          <Input
            value={url}
            onChange={e => handleUrlChange(e.target.value)}
            placeholder="Enter request URL"
            className="font-mono text-sm"
            onKeyDown={e => e.key === "Enter" && (e.ctrlKey || e.metaKey) && onSend()}
          />
        </div>

        {/* Send Button */}
        {isLoading ? (
          <Button
            onClick={onCancel}
            variant="destructive"
            className="h-9 px-6 gap-2"
          >
            <X className="h-4 w-4" />
            <span>Cancel</span>
          </Button>
        ) : (
          <Button
            onClick={onSend}
            className="pw-btn-primary h-9 px-6 gap-2"
            style={{ backgroundColor: "hsl(var(--pw-orange))", borderColor: "hsl(var(--pw-orange))" }}
          >
            <Send className="h-4 w-4" />
            <span>Send</span>
          </Button>
        )}

        {/* Save Button */}
        <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onSave} aria-label="Save (Ctrl+S)">
          <Save className="h-4 w-4" />
        </Button>
      </div>

      {/* Tab Navigation */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="border-b border-border bg-muted/30">
        <TabsList className="p-1 bg-transparent overflow-x-auto flex-nowrap justify-start w-full">
          <TabsTrigger value="params">Params</TabsTrigger>
          <TabsTrigger value="auth">Auth</TabsTrigger>
          <TabsTrigger value="headers">
            Headers
            {headers.length > 0 && <span className="ml-1 text-[10px] text-muted-foreground">{headers.length}</span>}
          </TabsTrigger>
          <TabsTrigger value="body">
            Body
            {body.mode && body.mode !== "none" && (
              <span className="ml-1 rounded bg-primary/15 px-1 text-[9px] font-semibold uppercase text-primary">
                {body.mode === "formdata" ? "form" : body.mode === "urlencoded" ? "url" : body.mode}
              </span>
            )}
          </TabsTrigger>
          <TabsTrigger value="scripts">Scripts</TabsTrigger>
          <TabsTrigger value="settings">Settings</TabsTrigger>
        </TabsList>

        {/* Params Tab */}
        <TabsContent value="params" className="p-3">
          <div className="space-y-4">
            {/* Path Variables */}
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-2 block">Path Variables</Label>
              <KeyValueEditor
                items={pathVariables}
                onChange={handlePathVariablesChange}
                columns={[
                  { key: "key", header: "Key", width: "30%" },
                  { key: "value", header: "Value", width: "50%" },
                  { key: "description", header: "Description", width: "20%" },
                ]}
              />
            </div>

            {/* Query Parameters */}
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-2 block">Query Parameters</Label>
              <KeyValueEditor
                items={queryParams}
                onChange={handleQueryParamsChange}
                columns={[
                  { key: "enabled", header: "", width: "40px", type: "checkbox" },
                  { key: "key", header: "Key", width: "30%" },
                  { key: "value", header: "Value", width: "50%" },
                  { key: "description", header: "Description", width: "20%" },
                ]}
              />
            </div>
          </div>
        </TabsContent>

        {/* Auth Tab */}
        <TabsContent value="auth" className="p-3">
          <AuthEditor auth={auth} onChange={handleAuthChange} />
        </TabsContent>

        {/* Headers Tab */}
        <TabsContent value="headers" className="p-3">
          <KeyValueEditor
            items={headers}
            onChange={handleHeadersChange}
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Key", width: "30%" },
              { key: "value", header: "Value", width: "50%" },
              { key: "description", header: "Description", width: "20%" },
            ]}
          />
        </TabsContent>

        {/* Body Tab */}
        <TabsContent value="body" className="p-3">
          <BodyEditor
            body={body}
            onChange={handleBodyChange}
            method={method}
            headers={headers}
            onHeadersChange={handleHeadersChange}
          />
        </TabsContent>

        {/* Scripts Tab */}
        <TabsContent value="scripts" className="p-3">
          <div className="space-y-4">
            {/* Pre-request Script */}
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-2 block">Pre-request Script</Label>
              <Textarea
                value={scripts.find(s => s.type === "prerequest")?.script || ""}
                onChange={e => handleScriptsChange([
                  ...scripts.filter(s => s.type !== "prerequest"),
                  { type: "prerequest", script: e.target.value }
                ])}
                className="font-mono text-sm min-h-[200px]"
                placeholder="// Pre-request script runs before the request is sent&#10;pm.environment.set('timestamp', Date.now().toString());"
              />
            </div>

            {/* Test Script */}
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-2 block">Test Script</Label>
              <Textarea
                value={scripts.find(s => s.type === "test")?.script || ""}
                onChange={e => handleScriptsChange([
                  ...scripts.filter(s => s.type !== "test"),
                  { type: "test", script: e.target.value }
                ])}
                className="font-mono text-sm min-h-[200px]"
                placeholder="// Test script runs after the response is received&#10;pm.test('Status is 200', () => {&#10;  pm.response.to.have.status(200);&#10;});"
              />
            </div>
          </div>
        </TabsContent>

        {/* Settings Tab */}
        <TabsContent value="settings" className="p-3 space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="followRedirects">Follow Redirects</Label>
              <Checkbox
                id="followRedirects"
                checked={settings.followRedirects}
                onCheckedChange={(checked) => handleSettingsChange({ followRedirects: checked === true })}
              />
            </div>
            <div>
              <Label htmlFor="followOriginalMethod">Follow Original Method</Label>
              <Checkbox
                id="followOriginalMethod"
                checked={settings.followOriginalMethod}
                onCheckedChange={(checked) => handleSettingsChange({ followOriginalMethod: checked === true })}
              />
            </div>
            <div>
              <Label htmlFor="maxRedirects">Max Redirects</Label>
              <Input
                id="maxRedirects"
                type="number"
                value={settings.maxRedirects}
                onChange={e => handleSettingsChange({ maxRedirects: parseInt(e.target.value) || 10 })}
                className="w-full mt-1"
              />
            </div>
            <div>
              <Label htmlFor="timeout">Timeout (ms)</Label>
              <Input
                id="timeout"
                type="number"
                value={settings.timeout}
                onChange={e => handleSettingsChange({ timeout: parseInt(e.target.value) || 30000 })}
                className="w-full mt-1"
              />
            </div>
            <div>
              <Label htmlFor="encoding">Response Encoding</Label>
              <Select value={settings.encoding} onValueChange={value => handleSettingsChange({ encoding: value as any })}>
                <SelectTrigger className="w-full mt-1">
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
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}