import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { AlertCircle, CheckCircle2, GitCompare, Plus, Sigma, Wand2 } from "lucide-react"
import { cn } from "@/lib/utils"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { ExtractionResult, HistoryEntry, JSONDiff, Request, Response } from "@/types"

interface ResponseToolsProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  response: Response | null
  request: Request | null
  history: HistoryEntry[]
}

type SaveScope = "environment" | "collection" | "global"

export function ResponseTools({ open, onOpenChange, response, request, history }: ResponseToolsProps) {
  const [tab, setTab] = useState("extract")
  const body = response?.body ?? ""

  // --- extract -----------------------------------------------------------
  const [path, setPath] = useState("$")
  const [extraction, setExtraction] = useState<ExtractionResult | null>(null)
  const [extractError, setExtractError] = useState<string | null>(null)
  const [variableName, setVariableName] = useState("")
  const [scope, setScope] = useState<SaveScope>("environment")
  const [saveMessage, setSaveMessage] = useState<string | null>(null)

  const runExtract = async () => {
    setExtractError(null)
    setExtraction(null)
    setSaveMessage(null)
    try {
      const result = await api.extractJSONPath(body, path)
      setExtraction(result)
      const last = path.split(/[.[]/).filter(Boolean).pop() || "value"
      setVariableName(last.replace(/[^A-Za-z0-9_]/g, ""))
    } catch (err) {
      setExtractError(api.errorMessage(err))
    }
  }

  const saveVariable = async () => {
    if (!extraction || !variableName.trim()) return
    const state = useWorkspaceStore.getState()
    try {
      if (scope === "global") {
        await state.setGlobalVariable(variableName.trim(), extraction.display)
      } else if (scope === "environment") {
        const env = state.environments.find((e) => e.id === state.activeEnvironmentId)
        if (!env) throw new Error("Select an environment first")
        const values = [...(env.values ?? [])]
        const index = values.findIndex((v) => v.key === variableName.trim())
        const entry = { key: variableName.trim(), value: extraction.display, type: "default", enabled: true }
        if (index === -1) values.push(entry)
        else values[index] = { ...values[index], value: extraction.display }
        await state.saveEnvironment({ ...env, values })
      } else {
        if (!request?.collectionId) throw new Error("This request is not saved in a collection")
        const collection = state.collections.find((c) => c.id === request.collectionId)
        if (!collection) throw new Error("Collection not found")
        const variables = [...(collection.variables ?? [])]
        const index = variables.findIndex((v) => v.key === variableName.trim())
        const entry = { key: variableName.trim(), value: extraction.display, type: "default", enabled: true }
        if (index === -1) variables.push(entry)
        else variables[index] = { ...variables[index], value: extraction.display }
        await state.updateCollectionMeta({ ...collection, variables })
      }
      setSaveMessage(`Saved {{${variableName.trim()}}} to ${scope}`)
    } catch (err) {
      setExtractError(api.errorMessage(err))
    }
  }

  // --- compare -----------------------------------------------------------
  const candidates = useMemo(
    () => history.filter((entry) => entry.response?.body && /^\s*[{[]/.test(entry.response.body)),
    [history],
  )
  const [compareId, setCompareId] = useState<string>("")
  const [diff, setDiff] = useState<JSONDiff | null>(null)
  const [compareError, setCompareError] = useState<string | null>(null)

  useEffect(() => {
    setDiff(null)
    setCompareError(null)
  }, [compareId, body])

  const runCompare = async () => {
    const entry = candidates.find((c) => c.id === compareId)
    if (!entry?.response?.body) {
      setCompareError("Pick a history response to compare against")
      return
    }
    setCompareError(null)
    try {
      setDiff(await api.compareJSON(entry.response.body, body))
    } catch (err) {
      setCompareError(api.errorMessage(err))
    }
  }

  // --- schema ------------------------------------------------------------
  const [schema, setSchema] = useState("")
  const [schemaErrors, setSchemaErrors] = useState<string[] | null>(null)
  const [schemaMessage, setSchemaMessage] = useState<string | null>(null)

  const generateSchema = async () => {
    setSchemaMessage(null)
    setSchemaErrors(null)
    try {
      setSchema(await api.generateJSONSchema(body))
      setSchemaMessage("Schema generated from the current response")
    } catch (err) {
      setSchemaMessage(api.errorMessage(err))
    }
  }

  const validateSchema = async () => {
    setSchemaMessage(null)
    setSchemaErrors(null)
    try {
      const errors = await api.validateJSONSchema(body, schema)
      setSchemaErrors(errors)
      if (errors.length === 0) setSchemaMessage("Response matches the schema")
    } catch (err) {
      setSchemaMessage(api.errorMessage(err))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Response tools</DialogTitle>
          <DialogDescription>Extract variables, compare responses and validate against a JSON Schema.</DialogDescription>
        </DialogHeader>

        <Tabs value={tab} onValueChange={setTab}>
          <TabsList>
            <TabsTrigger value="extract" className="gap-1.5"><Sigma className="h-3.5 w-3.5" /> Extract</TabsTrigger>
            <TabsTrigger value="compare" className="gap-1.5"><GitCompare className="h-3.5 w-3.5" /> Compare</TabsTrigger>
            <TabsTrigger value="schema" className="gap-1.5"><Wand2 className="h-3.5 w-3.5" /> Schema</TabsTrigger>
          </TabsList>

          <TabsContent value="extract" className="space-y-3">
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <Label className="text-xs">JSONPath</Label>
                <Input value={path} onChange={(e) => setPath(e.target.value)} placeholder="$.user.profile.email" className="font-mono" />
              </div>
              <Button onClick={runExtract}>Extract</Button>
            </div>
            {extractError && <ErrorLine message={extractError} />}
            {extraction && (
              <div className="space-y-2 rounded-md border border-border p-3">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span className="rounded bg-muted px-1.5 py-0.5">{extraction.type}</span>
                  <span className="font-mono truncate">{extraction.path}</span>
                </div>
                <pre className="max-h-32 overflow-auto rounded bg-muted/30 p-2 font-mono text-xs">{extraction.display}</pre>
                <div className="flex flex-wrap items-end gap-2">
                  <div className="flex-1">
                    <Label className="text-xs">Variable name</Label>
                    <Input value={variableName} onChange={(e) => setVariableName(e.target.value)} className="font-mono" />
                  </div>
                  <div className="w-40">
                    <Label className="text-xs">Save to</Label>
                    <Select value={scope} onValueChange={(v) => setScope(v as SaveScope)}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="environment">Environment</SelectItem>
                        <SelectItem value="collection">Collection</SelectItem>
                        <SelectItem value="global">Global</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <Button variant="outline" className="gap-1.5" onClick={saveVariable} disabled={!variableName.trim()}>
                    <Plus className="h-3.5 w-3.5" /> Save variable
                  </Button>
                </div>
                {saveMessage && <p className="text-xs text-green-600 dark:text-green-400">{saveMessage}</p>}
              </div>
            )}
          </TabsContent>

          <TabsContent value="compare" className="space-y-3">
            <div className="flex items-end gap-2">
              <div className="flex-1">
                <Label className="text-xs">Compare the current response with a history entry</Label>
                <Select value={compareId} onValueChange={setCompareId}>
                  <SelectTrigger><SelectValue placeholder="Select a historical response" /></SelectTrigger>
                  <SelectContent>
                    {candidates.slice(0, 50).map((entry) => (
                      <SelectItem key={entry.id} value={entry.id}>
                        {entry.request?.method} {entry.request?.url?.slice(0, 60)} · {new Date(entry.timestamp).toLocaleTimeString()}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={runCompare} disabled={!compareId}>Compare</Button>
            </div>
            {compareError && <ErrorLine message={compareError} />}
            {diff && (
              <div className="space-y-2">
                <div className="flex items-center gap-3 text-xs">
                  {diff.equal ? (
                    <span className="flex items-center gap-1 text-green-600 dark:text-green-400"><CheckCircle2 className="h-3.5 w-3.5" /> Identical</span>
                  ) : (
                    <>
                      <span className="rounded bg-green-500/15 px-1.5 py-0.5 text-green-700 dark:text-green-400">+{diff.added} added</span>
                      <span className="rounded bg-red-500/15 px-1.5 py-0.5 text-red-700 dark:text-red-400">-{diff.removed} removed</span>
                      <span className="rounded bg-amber-500/15 px-1.5 py-0.5 text-amber-700 dark:text-amber-400">~{diff.changed} changed</span>
                    </>
                  )}
                  {diff.truncated && <span className="text-muted-foreground">(truncated)</span>}
                </div>
                <ScrollArea className="max-h-72 rounded-md border border-border">
                  <div className="divide-y divide-border/50 font-mono text-xs">
                    {diff.entries.map((entry, index) => (
                      <div key={`${entry.path}-${index}`} className="flex items-start gap-2 p-2">
                        <span className={cn("shrink-0 rounded px-1 pw-text-3xs font-semibold", entry.kind === "added" && "bg-green-500/15 text-green-700 dark:text-green-400", entry.kind === "removed" && "bg-red-500/15 text-red-700 dark:text-red-400", entry.kind === "changed" && "bg-amber-500/15 text-amber-700 dark:text-amber-400")}>
                          {entry.kind}
                        </span>
                        <div className="min-w-0">
                          <div className="truncate">{entry.path}</div>
                          {entry.kind === "changed" ? (
                            <div className="text-muted-foreground">
                              <span className="text-red-600 dark:text-red-400">{entry.left}</span> → <span className="text-green-600 dark:text-green-400">{entry.right}</span>
                            </div>
                          ) : (
                            <div className="truncate text-muted-foreground">{entry.right || entry.left}</div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </div>
            )}
          </TabsContent>

          <TabsContent value="schema" className="space-y-3">
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={generateSchema} className="gap-1.5"><Wand2 className="h-3.5 w-3.5" /> Generate from response</Button>
              <Button onClick={validateSchema} disabled={!schema.trim()}>Validate response</Button>
              {schemaMessage && <span className="text-xs text-muted-foreground">{schemaMessage}</span>}
            </div>
            <textarea
              value={schema}
              onChange={(e) => setSchema(e.target.value)}
              spellCheck={false}
              placeholder='{ "type": "object", "required": ["id"] }'
              className="h-64 w-full rounded-md border border-border bg-background p-2 font-mono text-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
            />
            {schemaErrors && schemaErrors.length > 0 && (
              <div className="space-y-1 rounded-md border border-destructive/30 bg-destructive/5 p-2">
                {schemaErrors.map((error) => (
                  <div key={error} className="flex items-start gap-2 text-xs text-destructive"><AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" /> {error}</div>
                ))}
              </div>
            )}
            {schemaErrors && schemaErrors.length === 0 && (
              <p className="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400"><CheckCircle2 className="h-3.5 w-3.5" /> Response matches the schema</p>
            )}
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  )
}

function ErrorLine({ message }: { message: string }) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">
      <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 flex-1 break-all font-mono">{message}</span>
    </div>
  )
}
