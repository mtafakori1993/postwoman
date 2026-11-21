import React, { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Copy, FileUp, FolderOpen, NotebookPen, Plus, Trash2, X } from "lucide-react"
import { cn } from "@/lib/utils"
import { VariableInput } from "./VariableInput"
import { KeyValuePair } from "@/types"

interface Column {
  key: string
  header: string
  width?: string
  minWidth?: number
  type?: "text" | "checkbox" | "file" | "rowtype" | "autoValue"
  placeholder?: string
}

interface KeyValueEditorProps {
  items: KeyValuePair[]
  onChange: (items: KeyValuePair[]) => void
  columns: Column[]
  addLabel?: string
  emptyMessage?: string
  footerExtra?: React.ReactNode
  onChooseFile?: (row: KeyValuePair, index: number) => void
  onAdd?: () => void
  /** Offer `{{variable}}` autocompletion in text cells. */
  variableAutocomplete?: boolean
}

function emptyRow(): KeyValuePair {
  return { key: "", value: "", description: "", enabled: true, type: "text" }
}

/**
 * Structured key/value editor used by params, headers, form-data and
 * URL-encoded bodies. The `autoValue` column swaps between a text input and a
 * file picker based on the row's `type`, which is what makes multipart rows
 * feel like a real API client.
 */
export function KeyValueEditor({
  items,
  onChange,
  columns,
  addLabel = "Add Row",
  emptyMessage = "No entries yet.",
  footerExtra,
  onChooseFile,
  onAdd,
  variableAutocomplete = false,
}: KeyValueEditorProps) {
  const rows = items
  const [bulkOpen, setBulkOpen] = useState(false)
  const [bulkText, setBulkText] = useState("")

  const commit = (next: KeyValuePair[]) => onChange(next)

  const updateRow = (index: number, patch: Partial<KeyValuePair>) => {
    commit(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)))
  }

  const addRow = () => {
    if (onAdd) {
      onAdd()
      return
    }
    commit([...rows, emptyRow()])
  }

  const removeRow = (index: number) => {
    commit(rows.filter((_, i) => i !== index))
  }

  const duplicateRow = (index: number) => {
    const next = [...rows]
    next.splice(index + 1, 0, { ...rows[index] })
    commit(next)
  }

  return (
    <div className="border border-border rounded-md overflow-hidden bg-background">
      <div className="overflow-x-auto">
        <div className="min-w-[640px]">
          {/* Header */}
          <div className="flex items-center bg-muted/50 border-b border-border px-2 py-1.5 pw-text-2xs font-medium uppercase tracking-wide text-muted-foreground">
            {columns.map((col) => (
              <div
                key={col.key}
                className={cn("flex items-center px-1 shrink-0", col.type === "checkbox" && "justify-center")}
                style={{ width: col.width, minWidth: col.minWidth }}
              >
                {col.header}
              </div>
            ))}
            <div className="flex-1" />
            <div className="w-14 text-center shrink-0">Actions</div>
          </div>

          {/* Rows */}
          <div className="max-h-[min(420px,50vh)] overflow-y-auto">
            {rows.length === 0 && (
              <div className="px-3 py-5 text-xs text-muted-foreground">{emptyMessage}</div>
            )}
            {rows.map((row, index) => (
              <div
                key={index}
                className="flex items-center px-2 py-1 border-b border-border/50 last:border-0 hover:bg-muted/20 transition-colors"
              >
                {columns.map((col) => (
                  <div
                    key={col.key}
                    className={cn("flex items-center px-1 shrink-0", col.type === "checkbox" && "justify-center")}
                    style={{ width: col.width, minWidth: col.minWidth }}
                  >
                    {col.type === "checkbox" ? (
                      <Checkbox
                        checked={row.enabled !== false}
                        onCheckedChange={(checked) => updateRow(index, { enabled: checked === true })}
                        className="h-4 w-4"
                        aria-label="Enable entry"
                      />
                    ) : col.type === "rowtype" ? (
                      <Select
                        value={row.type === "file" ? "file" : "text"}
                        onValueChange={(value) =>
                          updateRow(index, {
                            type: value,
                            ...(value === "text" ? { src: "", fileName: "" } : {}),
                          })
                        }
                      >
                        <SelectTrigger className="h-8 text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="text">Text</SelectItem>
                          <SelectItem value="file">File</SelectItem>
                        </SelectContent>
                      </Select>
                    ) : col.type === "autoValue" && row.type === "file" ? (
                      <FileValue
                        row={row}
                        onPick={() => onChooseFile?.(row, index)}
                        onChange={(patch) => updateRow(index, patch)}
                      />
                    ) : (
                      variableAutocomplete ? (
                        <VariableInput
                          value={(row[col.key as keyof KeyValuePair] as string) || ""}
                          onChange={(next) => updateRow(index, { [col.key]: next })}
                          type={col.key === "value" && row.type === "secret" ? "password" : undefined}
                          placeholder={col.placeholder ?? col.header}
                          aria-label={col.header || col.key}
                          className="h-8 text-sm bg-transparent border-0 focus-visible:ring-1"
                        />
                      ) : (
                        <Input
                          value={(row[col.key as keyof KeyValuePair] as string) || ""}
                          onChange={(e) => updateRow(index, { [col.key]: e.target.value })}
                          placeholder={col.placeholder ?? col.header}
                          type={col.key === "value" && row.type === "secret" ? "password" : undefined}
                          className="h-8 text-sm bg-transparent border-0 focus-visible:ring-1"
                        />
                      )
                    )}
                  </div>
                ))}
                <div className="flex-1" />
                <div className="flex items-center gap-0.5 w-14 justify-center shrink-0">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                    onClick={() => duplicateRow(index)}
                    aria-label="Duplicate row"
                    title="Duplicate row"
                  >
                    <Copy className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-6 w-6 p-0 text-muted-foreground hover:text-destructive"
                    onClick={() => removeRow(index)}
                    aria-label="Remove row"
                    title="Remove row"
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {bulkOpen && (
        <div className="space-y-2 border-t border-border bg-muted/20 p-2">
          <textarea
            value={bulkText}
            onChange={(event) => setBulkText(event.target.value)}
            spellCheck={false}
            placeholder={"Paste one entry per line, for example:\ncontent-type: application/json\npage=2\n# lines starting with # are ignored"}
            className="h-28 w-full resize-none rounded-md border border-border bg-background p-2 font-mono text-xs outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              onClick={() => {
                const parsed: KeyValuePair[] = []
                for (const rawLine of bulkText.split(/\r?\n/)) {
                  const line = rawLine.trim()
                  if (!line || line.startsWith("#")) continue
                  const separator = line.includes(":") ? line.indexOf(":") : line.indexOf("=")
                  if (separator === -1) {
                    parsed.push({ key: line, value: "", description: "", enabled: true, type: "text" })
                    continue
                  }
                  parsed.push({
                    key: line.slice(0, separator).trim(),
                    value: line.slice(separator + 1).trim(),
                    description: "",
                    enabled: true,
                    type: "text",
                  })
                }
                if (parsed.length > 0) commit([...rows, ...parsed])
                setBulkText("")
                setBulkOpen(false)
              }}
            >
              Apply
            </Button>
            <Button variant="ghost" size="sm" onClick={() => setBulkOpen(false)}>
              Cancel
            </Button>
            <span className="pw-text-2xs text-muted-foreground">Use `key: value` or `key=value` per line.</span>
          </div>
        </div>
      )}

      <div className="flex items-center gap-2 px-2 py-2 border-t border-border bg-muted/20">
        <Button variant="outline" size="sm" className="gap-1.5" onClick={addRow}>
          <Plus className="h-3.5 w-3.5" />
          {addLabel}
        </Button>
        <Button variant="ghost" size="sm" className="gap-1.5" onClick={() => setBulkOpen((value) => !value)}>
          <NotebookPen className="h-3.5 w-3.5" />
          Bulk Edit
        </Button>
        {footerExtra}
      </div>
    </div>
  )
}

function FileValue({
  row,
  onPick,
  onChange,
}: {
  row: KeyValuePair
  onPick: () => void
  onChange: (patch: Partial<KeyValuePair>) => void
}) {
  return (
    <div className="flex items-center gap-1 w-full min-w-0">
      <Input
        value={row.src || ""}
        onChange={(e) =>
          onChange({
            src: e.target.value,
            type: "file",
            fileName: e.target.value.split(/[\\/]/).pop() || "",
          })
        }
        placeholder="Path to file"
        title={row.src || undefined}
        className="h-8 min-w-0 flex-1 text-sm bg-transparent border-0 focus-visible:ring-1 font-mono"
      />
      <Input
        value={row.contentType || ""}
        onChange={(e) => onChange({ contentType: e.target.value, type: "file" })}
        placeholder="auto"
        title="Content-Type sent for this file part"
        aria-label="Part content type"
        className="h-8 w-[104px] shrink-0 text-xs bg-transparent border-0 focus-visible:ring-1 font-mono"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-7 w-7 p-0 shrink-0 text-muted-foreground hover:text-foreground"
        onClick={onPick}
        aria-label="Choose file"
        title="Choose file"
      >
        <FolderOpen className="h-3.5 w-3.5" />
      </Button>
      {row.src && (
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="h-7 w-7 p-0 shrink-0 text-muted-foreground hover:text-destructive"
          onClick={() => onChange({ src: "", fileName: "" })}
          aria-label="Clear file"
          title="Clear file"
        >
          <X className="h-3 w-3" />
        </Button>
      )}
      {!row.src && <FileUp className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden />}
    </div>
  )
}
