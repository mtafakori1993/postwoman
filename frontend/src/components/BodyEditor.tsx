import React, { useState, useRef, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { KeyValueEditor } from "./KeyValueEditor"
import { KeyValuePair, BodyType, RequestBody } from "@/types"
import { cn } from "@/lib/utils"
import { FileText, Code, Image, Upload, Globe, Download, Copy } from "lucide-react"

const BODY_TYPES: { value: BodyType; label: string; icon: React.ReactNode }[] = [
  { value: "none", label: "None", icon: <FileText className="h-4 w-4" /> },
  { value: "json", label: "JSON", icon: <Code className="h-4 w-4" /> },
  { value: "xml", label: "XML", icon: <Code className="h-4 w-4" /> },
  { value: "text", label: "Text", icon: <FileText className="h-4 w-4" /> },
  { value: "javascript", label: "JavaScript", icon: <Code className="h-4 w-4" /> },
  { value: "html", label: "HTML", icon: <Globe className="h-4 w-4" /> },
  { value: "formdata", label: "Form Data", icon: <Upload className="h-4 w-4" /> },
  { value: "urlencoded", label: "URL Encoded", icon: <Globe className="h-4 w-4" /> },
  { value: "binary", label: "Binary", icon: <Image className="h-4 w-4" /> },
  { value: "graphql", label: "GraphQL", icon: <Code className="h-4 w-4" /> },
]

interface BodyEditorProps {
  body: RequestBody
  onChange: (body: RequestBody) => void
  method: string
}

export function BodyEditor({ body, onChange, method }: BodyEditorProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [activeType, setActiveType] = useState<BodyType>(body.mode || "none")
  const [rawText, setRawText] = useState(body.raw || "")
  const [formData, setFormData] = useState<KeyValuePair[]>(body.formdata || [])
  const [urlEncoded, setUrlEncoded] = useState<KeyValuePair[]>(body.urlencoded || [])
  const [graphqlQuery, setGraphqlQuery] = useState(body.graphql?.query || "")
  const [graphqlVariables, setGraphqlVariables] = useState(body.graphql?.variables || "{}")
  const [binaryFile, setBinaryFile] = useState<File | null>(null)

  // Sync with body prop
  useEffect(() => {
    setActiveType(body.mode || "none")
    setRawText(body.raw || "")
    setFormData(body.formdata || [])
    setUrlEncoded(body.urlencoded || [])
    setGraphqlQuery(body.graphql?.query || "")
    setGraphqlVariables(body.graphql?.variables || "{}")
  }, [body])

  const handleTypeChange = (type: BodyType) => {
    setActiveType(type)
    onChange({ ...body, mode: type })
  }

  const handleRawChange = (text: string) => {
    setRawText(text)
    onChange({ ...body, mode: activeType, raw: text })
  }

  const handleFormDataChange = (items: KeyValuePair[]) => {
    setFormData(items)
    onChange({ ...body, mode: "formdata", formdata: items })
  }

  const handleUrlEncodedChange = (items: KeyValuePair[]) => {
    setUrlEncoded(items)
    onChange({ ...body, mode: "urlencoded", urlencoded: items })
  }

  const handleGraphqlQueryChange = (query: string) => {
    setGraphqlQuery(query)
    onChange({ ...body, mode: "graphql", graphql: { query, variables: body.graphql?.variables || "{}" } })
  }

  const handleGraphqlVariablesChange = (variables: string) => {
    setGraphqlVariables(variables)
    onChange({ ...body, mode: "graphql", graphql: { query: body.graphql?.query || "", variables } })
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setBinaryFile(file)
      const reader = new FileReader()
      reader.onload = () => {
        onChange({ ...body, mode: "binary", binary: reader.result as string })
      }
      reader.readAsDataURL(file)
    }
  }

  // Disable body for methods that typically don't have a body
  const hasBody = !["GET", "HEAD", "DELETE"].includes(method.toUpperCase())

  if (!hasBody) {
    return (
      <div className="text-center text-muted-foreground py-8">
        <p className="mb-2">Request body is not allowed for {method} requests</p>
        <Select value={activeType} onValueChange={handleTypeChange} disabled>
          <SelectTrigger className="w-[200px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {BODY_TYPES.map(t => (
              <SelectItem key={t.value} value={t.value} disabled>
                <span className="flex items-center gap-2">{t.icon} {t.label}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Body Type Selector */}
      <div className="flex items-center gap-2">
        <Label className="text-xs font-medium text-muted-foreground">Body Type</Label>
        <Select value={activeType} onValueChange={handleTypeChange}>
          <SelectTrigger className="w-[200px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {BODY_TYPES.map(t => (
              <SelectItem key={t.value} value={t.value}>
                <span className="flex items-center gap-2">{t.icon} {t.label}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {/* Body Editor */}
      <div className="border border-border rounded-md overflow-hidden">
        {activeType === "none" && (
          <div className="p-8 text-center text-muted-foreground">
            <p>No request body</p>
          </div>
        )}

        {["json", "xml", "text", "javascript", "html"].includes(activeType) && (
          <div className="p-2">
            <Textarea
              value={rawText}
              onChange={e => handleRawChange(e.target.value)}
              className="font-mono text-sm min-h-[200px] bg-background border-0 focus:ring-0 resize-none"
              placeholder={getPlaceholder(activeType)}
              spellCheck={false}
            />
          </div>
        )}

        {activeType === "formdata" && (
          <KeyValueEditor
            items={formData}
            onChange={handleFormDataChange}
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Key", width: "25%" },
              { key: "value", header: "Value", width: "40%" },
              { key: "description", header: "Description", width: "15%" },
              { key: "type", header: "Type", width: "10%" },
            ]}
          />
        )}

        {activeType === "urlencoded" && (
          <KeyValueEditor
            items={urlEncoded}
            onChange={handleUrlEncodedChange}
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Key", width: "35%" },
              { key: "value", header: "Value", width: "50%" },
              { key: "description", header: "Description", width: "15%" },
            ]}
          />
        )}

        {activeType === "graphql" && (
          <div className="p-2 space-y-3">
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-1 block">Query</Label>
              <Textarea
                value={graphqlQuery}
                onChange={e => handleGraphqlQueryChange(e.target.value)}
                className="font-mono text-sm min-h-[150px] border border-border rounded-md"
                placeholder="query {&#10;  users {&#10;    id&#10;    name&#10;  }&#10;}"
              />
            </div>
            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-1 block">Variables</Label>
              <Textarea
                value={graphqlVariables}
                onChange={e => handleGraphqlVariablesChange(e.target.value)}
                className="font-mono text-sm min-h-[80px] border border-border rounded-md"
                placeholder='{&#10;  "limit": 10&#10;}'
              />
            </div>
          </div>
        )}

        {activeType === "binary" && (
          <div className="p-4 space-y-3 text-center">
            <input
              type="file"
              onChange={handleFileChange}
              className="hidden"
              id="binary-file-input"
              ref={fileInputRef}
            />
            <Label htmlFor="binary-file-input" className="cursor-pointer">
              <Button variant="outline" className="gap-2">
                <Upload className="h-4 w-4" />
                Select File
              </Button>
            </Label>
            {binaryFile && (
              <div className="flex items-center justify-center gap-2 text-sm">
                <FileText className="h-4 w-4" />
                <span className="truncate max-w-[200px]">{binaryFile.name}</span>
                <span className="text-muted-foreground">({formatBytes(binaryFile.size)})</span>
                <Button variant="ghost" size="icon" onClick={() => setBinaryFile(null)}>
                  <Download className="h-3 w-3" />
                </Button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

function getPlaceholder(type: BodyType): string {
  switch (type) {
    case "json":
      return "{\n  \"key\": \"value\"\n}"
    case "xml":
      return "<?xml version=\"1.0\"?>\n<root>\n  <element>value</element>\n</root>"
    case "javascript":
      return "// JavaScript code\nconsole.log('Hello');"
    case "html":
      return "<!DOCTYPE html>\n<html>\n  <body>Hello</body>\n</html>"
    default:
      return "Enter text content..."
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes"
  const k = 1024
  const sizes = ["Bytes", "KB", "MB", "GB"]
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
}