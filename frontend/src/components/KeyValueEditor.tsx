import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Plus, Trash2, Upload, X } from "lucide-react"
import { cn } from "@/lib/utils"
import { KeyValuePair } from "@/types"

interface Column {
  key: string
  header: string
  width?: string
  type?: "text" | "checkbox" | "file"
}

interface KeyValueEditorProps {
  items: KeyValuePair[]
  onChange: (items: KeyValuePair[]) => void
  columns: Column[]
  autoAddCommon?: boolean
}

export function KeyValueEditor({ items, onChange, columns }: KeyValueEditorProps) {
  const [rows, setRows] = useState<KeyValuePair[]>(items)

  useEffect(() => {
    setRows(items)
  }, [items])

  const commit = (next: KeyValuePair[]) => {
    setRows(next)
    onChange(next)
  }

  const updateRow = (index: number, key: string, value: string | boolean) => {
    const next = rows.map((row, i) => {
      if (i !== index) return row
      if (key === "enabled") return { ...row, enabled: value as boolean }
      return { ...row, [key]: value }
    })
    commit(next)
  }

  const addRow = () => {
    commit([...rows, { key: "", value: "", description: "", enabled: true, type: "text" }])
  }

  const removeRow = (index: number) => {
    commit(rows.filter((_, i) => i !== index))
  }

  return (
    <div className="border border-border rounded-md overflow-hidden">
      {/* Header */}
      <div className="flex items-center bg-muted/50 border-b border-border px-2 py-1 text-xs font-medium text-muted-foreground">
        {columns.map((col) => (
          <div
            key={col.key}
            className={cn("flex items-center px-1", col.type === "checkbox" && "justify-center")}
            style={{ width: col.width }}
          >
            {col.header}
          </div>
        ))}
        <div className="flex-1" />
        <div className="w-8 text-center">Actions</div>
      </div>

      {/* Rows */}
      <div className="max-h-[320px] overflow-y-auto">
        {rows.length === 0 && (
          <div className="px-3 py-4 text-xs text-muted-foreground">No entries yet.</div>
        )}
        {rows.map((row, index) => (
          <div
            key={index}
            className="flex items-center px-2 py-1 border-b border-border/50 last:border-0 hover:bg-muted/30 transition-colors"
          >
            {columns.map((col) => (
              <div
                key={col.key}
                className={cn("flex items-center px-1", col.type === "checkbox" && "justify-center")}
                style={{ width: col.width }}
              >
                {col.type === "checkbox" ? (
                  <Checkbox
                    checked={row.enabled !== false}
                    onCheckedChange={(checked) => updateRow(index, "enabled", checked === true)}
                    className="h-4 w-4"
                    aria-label="Enable entry"
                  />
                ) : col.type === "file" ? (
                  <FileCell
                    row={row}
                    onChange={(patch) => {
                      const next = rows.map((r, i) => (i === index ? { ...r, ...patch } : r))
                      commit(next)
                    }}
                  />
                ) : (
                  <Input
                    value={(row[col.key as keyof KeyValuePair] as string) || ""}
                    onChange={(e) => updateRow(index, col.key, e.target.value)}
                    placeholder={col.header}
                    className="h-8 text-sm bg-transparent border-0 focus-visible:ring-1"
                  />
                )}
              </div>
            ))}
            <div className="flex-1" />
            <div className="flex items-center gap-1 w-8 justify-center">
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 p-0 opacity-50 hover:opacity-100"
                onClick={() => removeRow(index)}
                aria-label="Remove row"
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          </div>
        ))}

        <div className="px-2 py-2">
          <Button variant="outline" size="sm" className="w-full" onClick={addRow}>
            <Plus className="h-3 w-3 mr-1" />
            Add Row
          </Button>
        </div>
      </div>
    </div>
  )
}

/**
 * File cell for multipart form-data. The user types an absolute path to the
 * file; the Go backend reads it when building the multipart body. No fake file
 * picker is used because the request is executed by the backend, not the
 * browser.
 */
function FileCell({ row, onChange }: { row: KeyValuePair; onChange: (patch: Partial<KeyValuePair>) => void }) {
  return (
    <div className="flex items-center gap-1 w-full">
      <Input
        value={row.src || ""}
        onChange={(e) => onChange({ src: e.target.value, type: "file" })}
        placeholder="Absolute file path"
        className="h-8 text-sm bg-transparent border-0 focus-visible:ring-1"
      />
      {row.src && (
        <Button variant="ghost" size="icon" className="h-6 w-6 p-0" onClick={() => onChange({ src: "" })} aria-label="Clear file">
          <X className="h-3 w-3" />
        </Button>
      )}
      <Upload className="h-3 w-3 text-muted-foreground" />
    </div>
  )
}
