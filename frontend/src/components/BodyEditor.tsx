import React, { useEffect, useMemo, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { KeyValueEditor } from "./KeyValueEditor"
import { SyntaxEditor } from "./SyntaxEditor"
import { Header, KeyValuePair, BodyType, RequestBody, Request } from "@/types"
import { cn } from "@/lib/utils"
import {
  AlertCircle,
  Braces,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Code2,
  FileCode2,
  FileText,
  FileUp,
  Globe,
  HardDrive,
  Loader2,
  Minimize2,
  RefreshCw,
  Upload,
  Wand2,
  X,
} from "lucide-react"
import * as api from "../lib/api"
import {
  defaultContentTypeForBody,
  formatContent,
  languageForBody,
  minifyContent,
  validateContent,
} from "@/lib/format"

const BODY_TYPES: { value: BodyType; label: string; icon: React.ReactNode }[] = [
  { value: "none", label: "None", icon: <FileText className="h-4 w-4" /> },
  { value: "raw", label: "Raw", icon: <FileCode2 className="h-4 w-4" /> },
  { value: "json", label: "JSON", icon: <Braces className="h-4 w-4" /> },
  { value: "xml", label: "XML", icon: <Code2 className="h-4 w-4" /> },
  { value: "text", label: "Text", icon: <FileText className="h-4 w-4" /> },
  { value: "javascript", label: "JavaScript", icon: <Code2 className="h-4 w-4" /> },
  { value: "html", label: "HTML", icon: <Globe className="h-4 w-4" /> },
  { value: "formdata", label: "Form Data", icon: <Upload className="h-4 w-4" /> },
  { value: "urlencoded", label: "URL Encoded", icon: <Globe className="h-4 w-4" /> },
  { value: "binary", label: "Binary", icon: <HardDrive className="h-4 w-4" /> },
  { value: "graphql", label: "GraphQL", icon: <Braces className="h-4 w-4" /> },
]

interface BodyEditorProps {
  body: RequestBody
  onChange: (body: RequestBody) => void
  method: string
  headers: Header[]
  onHeadersChange: (headers: Header[]) => void
  request?: Request | null
  environmentId?: string | null
}

interface GraphQLTypeInfo {
  name: string
  kind: string
  description?: string
  fields: { name: string; type: string; description?: string }[]
  enumValues?: string[]
}

export function BodyEditor({ body, onChange, method, headers, onHeadersChange, request, environmentId }: BodyEditorProps) {
  const activeType = (body.mode || "none") as BodyType
  const rawText = body.raw || ""
  const formData = body.formdata || []
  const urlEncoded = body.urlencoded || []
  const graphqlQuery = body.graphql?.query || ""
  const graphqlVariables = body.graphql?.variables || "{}"

  const binaryInputRef = useRef<HTMLInputElement>(null)
  const [binaryMeta, setBinaryMeta] = useState<{ name: string; size: number } | null>(null)
  const [bodyError, setBodyError] = useState<string | null>(null)
  const [formatError, setFormatError] = useState<string | null>(null)
  const [schemaTypes, setSchemaTypes] = useState<GraphQLTypeInfo[]>([])
  const [schemaError, setSchemaError] = useState<string | null>(null)
  const [schemaBusy, setSchemaBusy] = useState(false)
  const [schemaOpen, setSchemaOpen] = useState(true)
  const [openType, setOpenType] = useState<string | null>(null)
  const schemaStorageKey = request?.url ? `postwoman.graphqlSchema:${request.url}` : ""

  useEffect(() => {
    if (!schemaStorageKey) {
      setSchemaTypes([])
      return
    }
    try {
      const raw = localStorage.getItem(schemaStorageKey)
      setSchemaTypes(raw ? (JSON.parse(raw) as GraphQLTypeInfo[]) : [])
    } catch {
      setSchemaTypes([])
    }
    setSchemaError(null)
  }, [schemaStorageKey])

  const fetchSchema = async () => {
    if (!request) {
      setSchemaError("Open a request before introspecting its schema.")
      return
    }
    setSchemaBusy(true)
    setSchemaError(null)
    try {
      const raw = await api.graphqlIntrospect(request, environmentId ?? "")
      const parsed = JSON.parse(raw) as {
        data?: { __schema?: { types?: any[] } }
        errors?: { message: string }[]
      }
      if (parsed.errors?.length) throw new Error(parsed.errors.map((e) => e.message).join("; "))
      const types = (parsed.data?.__schema?.types ?? []).filter((t: any) => !String(t.name).startsWith("__"))
      const mapped: GraphQLTypeInfo[] = types
        .filter((t: any) => t.fields || t.enumValues)
        .map((t: any) => ({
          name: String(t.name),
          kind: String(t.kind ?? ""),
          description: t.description ?? undefined,
          fields: (t.fields ?? []).map((f: any) => ({ name: String(f.name), type: renderTypeRef(f.type), description: f.description })),
          enumValues: (t.enumValues ?? []).map((v: any) => String(v.name)),
        }))
        .sort((a, b) => a.name.localeCompare(b.name))
      setSchemaTypes(mapped)
      if (schemaStorageKey) localStorage.setItem(schemaStorageKey, JSON.stringify(mapped))
    } catch (err) {
      setSchemaError(api.errorMessage(err))
    } finally {
      setSchemaBusy(false)
    }
  }

  const graphqlIssue = useMemo(() => {
    if (activeType !== "graphql" || !graphqlQuery.trim()) return null
    let depth = 0
    for (const char of graphqlQuery) {
      if (char === "{") depth += 1
      else if (char === "}") {
        depth -= 1
        if (depth < 0) return "Unbalanced braces: unexpected '}'"
      }
    }
    if (depth !== 0) return "Unbalanced braces: a '{' is not closed"
    const rootField = /(?:query|mutation|subscription)?[^{]*{\s*([A-Za-z_][A-Za-z0-9_]*)/.exec(graphqlQuery)?.[1]
    if (rootField && schemaTypes.length > 0 && !schemaTypes.some((t) => t.fields.some((f) => f.name === rootField))) {
      return `Root field "${rootField}" was not found in the introspected schema`
    }
    return null
  }, [activeType, graphqlQuery, schemaTypes])

  const language = languageForBody(activeType)
  const isRawEditor = ["raw", "json", "xml", "text", "javascript", "html"].includes(activeType)
  const canFormat = ["json", "xml", "html"].includes(activeType)
  const canMinify = activeType === "json"

  const validationError = useMemo(() => {
    if (!isRawEditor) return null
    return validateContent(rawText, language)
  }, [isRawEditor, rawText, language])

  const contentHeader = getContentType(headers)
  const autoContentType = defaultContentTypeForBody(activeType)

  useEffect(() => {
    setBodyError(null)
    setFormatError(null)
  }, [activeType])

  const updateBody = (patch: Partial<RequestBody>) => onChange({ ...body, ...patch })

  const handleTypeChange = (type: BodyType) => {
    const previousType = activeType
    updateBody({ mode: type })
    onHeadersChange(syncContentTypeHeader(headers, previousType, type))
  }

  const handleRawChange = (text: string) => {
    setFormatError(null)
    updateBody({ mode: activeType, raw: text })
  }

  const handleFormat = () => {
    const result = formatContent(rawText, language)
    if (result.ok) {
      setFormatError(null)
      updateBody({ mode: activeType, raw: result.output })
    } else {
      setFormatError(result.error || "Could not format content")
    }
  }

  const handleMinify = () => {
    const result = minifyContent(rawText, language)
    if (result.ok) {
      setFormatError(null)
      updateBody({ mode: activeType, raw: result.output })
    } else {
      setFormatError(result.error || "Could not minify content")
    }
  }

  const handleContentTypeChange = (value: string) => {
    onHeadersChange(setContentType(headers, value))
  }

  const chooseFormDataFile = async (row: KeyValuePair, index: number) => {
    try {
      const path = await api.openFileDialog("Choose a file to upload", "*")
      if (!path) return
      const next = formData.map((item, i) =>
        i === index
          ? {
              ...item,
              type: "file",
              src: path,
              fileName: path.split(/[\\/]/).pop() || path,
              enabled: true,
            }
          : item,
      )
      updateBody({ mode: "formdata", formdata: next })
      setBodyError(null)
    } catch (error) {
      setBodyError(api.errorMessage(error))
    }
  }

  const addFileRow = () => {
    updateBody({
      mode: "formdata",
      formdata: [
        ...formData,
        { key: "", value: "", description: "", enabled: true, type: "file", src: "", fileName: "" },
      ],
    })
  }

  const handleBinaryFile = (file: File) => {
    if (file.size > 100 * 1024 * 1024) {
      setBodyError("File exceeds the 100 MiB upload limit.")
      return
    }
    setBodyError(null)
    const reader = new FileReader()
    reader.onload = () => {
      const dataUrl = reader.result as string
      setBinaryMeta({ name: file.name, size: file.size })
      updateBody({ mode: "binary", binary: dataUrl, fileName: file.name })
      const currentContentType = getContentType(headers)
      if (!currentContentType || currentContentType.toLowerCase() === defaultContentTypeForBody("binary")) {
        onHeadersChange(setContentType(headers, file.type || "application/octet-stream"))
      }
    }
    reader.onerror = () => setBodyError("Could not read the selected file.")
    reader.readAsDataURL(file)
  }

  const clearBinary = () => {
    setBinaryMeta(null)
    updateBody({ mode: "binary", binary: "", fileName: "" })
  }

  const handleDrop = (event: React.DragEvent) => {
    event.preventDefault()
    const file = event.dataTransfer.files?.[0]
    if (file) handleBinaryFile(file)
  }

  return (
    <div className="space-y-3">
      {/* Body type + content type ------------------------------------------- */}
      <div className="flex flex-col gap-2 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex items-center gap-2">
          <Label htmlFor="body-type-select" className="text-xs font-medium text-muted-foreground">Body Type</Label>
          <Select value={activeType} onValueChange={(value) => handleTypeChange(value as BodyType)}>
            <SelectTrigger id="body-type-select" className="w-[210px] h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {BODY_TYPES.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  <span className="flex items-center gap-2">
                    {type.icon}
                    {type.label}
                  </span>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-2 min-w-0">
          {activeType !== "none" && (
            <>
              <Label htmlFor="body-content-type" className="text-xs font-medium text-muted-foreground shrink-0">
                Content-Type
              </Label>
              {activeType === "formdata" ? (
                <div className="flex h-8 min-w-0 flex-1 items-center gap-2 rounded-md border border-input bg-muted/30 px-2 text-xs text-muted-foreground lg:w-[300px]">
                  <span className="truncate font-mono">multipart/form-data</span>
                  <span className="shrink-0 rounded bg-primary/10 px-1.5 py-0.5 pw-text-3xs font-medium text-primary">
                    boundary auto
                  </span>
                </div>
              ) : (
                <div className="flex min-w-0 flex-1 items-center gap-1 lg:w-[300px]">
                  <Input
                    id="body-content-type"
                    value={contentHeader}
                    onChange={(event) => handleContentTypeChange(event.target.value)}
                    placeholder={autoContentType ? `Auto · ${autoContentType}` : "Auto"}
                    className="h-8 font-mono text-xs"
                  />
                  {contentHeader ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 shrink-0 px-2 pw-text-2xs"
                      onClick={() => handleContentTypeChange("")}
                      title="Use the automatic content type"
                    >
                      Auto
                    </Button>
                  ) : (
                    <span className="shrink-0 rounded bg-muted px-1.5 py-0.5 pw-text-3xs font-medium text-muted-foreground">
                      Auto
                    </span>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>

      {["GET", "HEAD"].includes(method.toUpperCase()) && activeType !== "none" && (
        <div className="flex items-center gap-2 rounded-md border border-amber-500/30 bg-amber-500/10 px-2.5 py-1.5 text-xs text-amber-700 dark:text-amber-400">
          <AlertCircle className="h-3.5 w-3.5 shrink-0" />
          {method.toUpperCase()} requests usually do not include a body. It will still be sent as configured.
        </div>
      )}

      {/* Mode-specific editor ----------------------------------------------- */}
      {activeType === "none" && (
        <div className="rounded-md border border-dashed border-border bg-muted/10 py-12 text-center">
          <FileText className="mx-auto mb-2 h-8 w-8 text-muted-foreground/50" />
          <p className="text-sm text-muted-foreground">This request has no body</p>
          <p className="mt-0.5 text-xs text-muted-foreground/70">Choose a body type above to add one</p>
        </div>
      )}

      {isRawEditor && (
        <div className="overflow-hidden rounded-md border border-border">
          <div className="flex h-9 items-center gap-2 border-b border-border bg-muted/30 px-2.5">
            <span className="pw-text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
              {activeType === "raw" ? "Plain text" : activeType}
            </span>
            {validationError ? (
              <span className="flex items-center gap-1 pw-text-2xs text-destructive">
                <AlertCircle className="h-3 w-3" /> Invalid
              </span>
            ) : rawText.trim() ? (
              <span className="flex items-center gap-1 pw-text-2xs text-green-600 dark:text-green-400">
                <CheckCircle2 className="h-3 w-3" /> Valid
              </span>
            ) : null}

            <div className="ml-auto flex items-center gap-1">
              <span className="hidden pw-text-2xs text-muted-foreground sm:inline">
                {rawText.length.toLocaleString()} chars
              </span>
              {canFormat && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 gap-1.5 px-2 pw-text-2xs"
                  onClick={handleFormat}
                  title="Format body"
                >
                  <Wand2 className="h-3.5 w-3.5" />
                  Format
                </Button>
              )}
              {canMinify && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 gap-1.5 px-2 pw-text-2xs"
                  onClick={handleMinify}
                  title="Minify body"
                >
                  <Minimize2 className="h-3.5 w-3.5" />
                  Minify
                </Button>
              )}
            </div>
          </div>
          <div style={{ height: "clamp(220px, 42vh, 560px)" }}>
            <SyntaxEditor
              value={rawText}
              onChange={handleRawChange}
              language={language}
              placeholder={getPlaceholder(activeType)}
              minHeight="100%"
              ariaLabel="Request body"
            />
          </div>
          {(validationError || formatError) && (
            <div className="flex items-start gap-2 border-t border-destructive/20 bg-destructive/5 px-2.5 py-1.5 pw-text-2xs text-destructive">
              <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span className="break-all font-mono">{formatError || validationError}</span>
            </div>
          )}
        </div>
      )}

      {activeType === "formdata" && (
        <div className="space-y-2">
          <KeyValueEditor
            items={formData}
            onChange={(items) => updateBody({ mode: "formdata", formdata: items })}
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Key", width: "24%", minWidth: 140 },
              { key: "value", header: "Value", width: "32%", minWidth: 190, type: "autoValue" },
              { key: "type", header: "Type", width: "92px", type: "rowtype" },
              { key: "description", header: "Description", width: "18%", minWidth: 120 },
            ]}
            onChooseFile={chooseFormDataFile}
            footerExtra={
              <Button variant="outline" size="sm" className="gap-1.5" onClick={addFileRow}>
                <FileUp className="h-3.5 w-3.5" />
                Add File
              </Button>
            }
            emptyMessage="No form fields yet. Add a text field or a file."
          />
          <p className="pw-text-2xs text-muted-foreground">
            File fields can be repeated with the same key to upload multiple files.
          </p>
          {bodyError && <InlineError message={bodyError} onClose={() => setBodyError(null)} />}
        </div>
      )}

      {activeType === "urlencoded" && (
        <KeyValueEditor
          items={urlEncoded}
          onChange={(items) => updateBody({ mode: "urlencoded", urlencoded: items })}
          columns={[
            { key: "enabled", header: "", width: "40px", type: "checkbox" },
            { key: "key", header: "Key", width: "32%", minWidth: 160 },
            { key: "value", header: "Value", width: "46%", minWidth: 220 },
            { key: "description", header: "Description", width: "22%", minWidth: 130 },
          ]}
          emptyMessage="No parameters yet. Add a URL-encoded field."
        />
      )}

      {activeType === "graphql" && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <Button variant="outline" size="sm" className="gap-1.5" onClick={fetchSchema} disabled={schemaBusy}>
              {schemaBusy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              Introspect schema
            </Button>
            {schemaTypes.length > 0 && <span className="pw-text-2xs text-muted-foreground">{schemaTypes.length} types loaded</span>}
            {graphqlIssue ? (
              <span className="flex items-center gap-1 pw-text-2xs text-destructive">
                <AlertCircle className="h-3 w-3" /> {graphqlIssue}
              </span>
            ) : graphqlQuery.trim() ? (
              <span className="flex items-center gap-1 pw-text-2xs text-green-600 dark:text-green-400">
                <CheckCircle2 className="h-3 w-3" /> Query looks valid
              </span>
            ) : null}
          </div>

          <div className="overflow-hidden rounded-md border border-border">
            <div className="border-b border-border bg-muted/30 px-2.5 py-1.5 pw-text-2xs font-semibold uppercase tracking-wide text-muted-foreground">Query</div>
            <div style={{ height: "clamp(160px, 30vh, 360px)" }}>
              <SyntaxEditor
                value={graphqlQuery}
                onChange={(query) => updateBody({ mode: "graphql", graphql: { query, variables: graphqlVariables } })}
                language="javascript"
                placeholder={"query {\n  users {\n    id\n    name\n  }\n}"}
                minHeight="100%"
                ariaLabel="GraphQL query"
              />
            </div>
          </div>

          <div className="overflow-hidden rounded-md border border-border">
            <div className="border-b border-border bg-muted/30 px-2.5 py-1.5 pw-text-2xs font-semibold uppercase tracking-wide text-muted-foreground">Variables</div>
            <div style={{ height: "140px" }}>
              <SyntaxEditor
                value={graphqlVariables}
                onChange={(variables) => updateBody({ mode: "graphql", graphql: { query: graphqlQuery, variables } })}
                language="json"
                placeholder={'{\n  "limit": 10\n}'}
                minHeight="100%"
                ariaLabel="GraphQL variables"
              />
            </div>
          </div>
          {validateContent(graphqlVariables, "json") && (
            <InlineError message={`GraphQL variables: ${validateContent(graphqlVariables, "json")}`} />
          )}

          {schemaError && <InlineError message={schemaError} onClose={() => setSchemaError(null)} />}

          {schemaTypes.length > 0 && (
            <div className="overflow-hidden rounded-md border border-border">
              <button
                type="button"
                className="flex w-full items-center gap-2 border-b border-border bg-muted/30 px-2.5 py-1.5 pw-text-2xs font-semibold uppercase tracking-wide text-muted-foreground"
                onClick={() => setSchemaOpen((v) => !v)}
              >
                {schemaOpen ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
                Schema explorer
              </button>
              {schemaOpen && (
                <div className="max-h-64 divide-y divide-border/50 overflow-y-auto">
                  {schemaTypes.map((type) => (
                    <div key={type.name}>
                      <button
                        type="button"
                        className="flex w-full items-center gap-2 px-2.5 py-1 text-left text-xs hover:bg-muted/40"
                        onClick={() => setOpenType((current) => (current === type.name ? null : type.name))}
                      >
                        {openType === type.name ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
                        <span className="font-mono font-medium">{type.name}</span>
                        <span className="ml-auto rounded bg-muted px-1 pw-text-3xs text-muted-foreground">
                          {type.kind.replace(/_/g, " ").toLowerCase()}
                        </span>
                      </button>
                      {openType === type.name && (
                        <div className="bg-muted/10 px-6 py-1.5 font-mono pw-text-2xs">
                          {type.description && <p className="mb-1 text-muted-foreground italic">{type.description}</p>}
                          {type.fields.map((field) => (
                            <div key={field.name} className="flex gap-2">
                              <span>{field.name}</span>
                              <span className="text-muted-foreground">{field.type}</span>
                            </div>
                          ))}
                          {type.enumValues && type.enumValues.length > 0 && (
                            <div className="text-muted-foreground">{type.enumValues.join(" | ")}</div>
                          )}
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {activeType === "binary" && (
        <div
          className="rounded-md border border-dashed border-border bg-muted/10 p-5 text-center transition-colors hover:border-primary/50 hover:bg-muted/20"
          onDragOver={(event) => event.preventDefault()}
          onDrop={handleDrop}
        >
          <input
            ref={binaryInputRef}
            type="file"
            className="hidden"
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) handleBinaryFile(file)
              event.target.value = ""
            }}
          />
          {body.binary ? (
            <div className="mx-auto flex max-w-xl items-center gap-3 rounded-md border border-border bg-background px-3 py-2 text-left">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded bg-primary/10 text-primary">
                <FileUp className="h-4 w-4" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">{body.fileName || "Selected file"}</p>
                <p className="pw-text-2xs text-muted-foreground">
                  {binaryMeta && binaryMeta.name === body.fileName ? `${formatBytes(binaryMeta.size)} · ` : ""}Binary payload
                </p>
              </div>
              <Button variant="outline" size="sm" onClick={() => binaryInputRef.current?.click()}>
                Change
              </Button>
              <Button variant="ghost" size="icon" className="h-8 w-8" onClick={clearBinary} aria-label="Remove file">
                <X className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <>
              <Upload className="mx-auto mb-2 h-8 w-8 text-muted-foreground/60" />
              <p className="text-sm font-medium">Select a binary file</p>
              <p className="mt-0.5 text-xs text-muted-foreground">Drag a file here or choose one from disk</p>
              <Button variant="outline" size="sm" className="mt-3 gap-1.5" onClick={() => binaryInputRef.current?.click()}>
                <Upload className="h-3.5 w-3.5" />
                Choose File
              </Button>
            </>
          )}
          {bodyError && <InlineError className="mt-3 text-left" message={bodyError} onClose={() => setBodyError(null)} />}
        </div>
      )}
    </div>
  )
}

function InlineError({
  message,
  onClose,
  className,
}: {
  message: string
  onClose?: () => void
  className?: string
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/5 px-2.5 py-1.5 pw-text-2xs text-destructive",
        className,
      )}
    >
      <AlertCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span className="min-w-0 flex-1 break-all font-mono">{message}</span>
      {onClose && (
        <button type="button" onClick={onClose} aria-label="Dismiss error" className="shrink-0 opacity-70 hover:opacity-100">
          <X className="h-3 w-3" />
        </button>
      )}
    </div>
  )
}

function getContentType(headers: Header[]): string {
  return (
    headers.find((header) => header.enabled !== false && header.key.toLowerCase() === "content-type")?.value ?? ""
  )
}

function setContentType(headers: Header[], value: string): Header[] {
  const cleaned = value.trim()
  const next = headers.filter((header) => header.key.toLowerCase() !== "content-type")
  if (!cleaned) return next
  return [...next, { key: "Content-Type", value: cleaned, enabled: true, description: "" }]
}

/** Update Content-Type only when it is missing or still equals the value the
 * previous body mode generated automatically. Custom overrides are preserved. */
function syncContentTypeHeader(headers: Header[], previousMode: BodyType, nextMode: BodyType): Header[] {
  const current = getContentType(headers)
  const previousDefault = defaultContentTypeForBody(previousMode)?.toLowerCase()
  const nextDefault = defaultContentTypeForBody(nextMode)

  // Multipart boundaries are generated at request time; a stale header would
  // break the upload.
  if (nextMode === "formdata") return setContentType(headers, "")

  if (!current || current.toLowerCase() === previousDefault) {
    return setContentType(headers, nextDefault ?? "")
  }
  return headers
}

function getPlaceholder(type: BodyType): string {
  switch (type) {
    case "json":
      return '{\n  "key": "value"\n}'
    case "xml":
      return '<?xml version="1.0" encoding="UTF-8"?>\n<root>\n  <element>value</element>\n</root>'
    case "javascript":
      return "// JavaScript payload\nconsole.log('Hello');"
    case "html":
      return "<!DOCTYPE html>\n<html>\n  <body>Hello</body>\n</html>"
    default:
      return "Enter request body..."
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const k = 1024
  const sizes = ["B", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function renderTypeRef(ref: any): string {
  if (!ref) return "Unknown"
  if (ref.kind === "NON_NULL") return `${renderTypeRef(ref.ofType)}!`
  if (ref.kind === "LIST") return `[${renderTypeRef(ref.ofType)}]`
  return String(ref.name ?? "Unknown")
}
