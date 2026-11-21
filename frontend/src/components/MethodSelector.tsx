import React, { useEffect, useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Check, Pencil } from "lucide-react"

const COMMON_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE", "CONNECT"]

interface MethodSelectorProps {
  value: string
  onChange: (method: string) => void
}

/**
 * HTTP method picker with first-class support for custom methods (WebDAV, etc.).
 * The custom mode keeps the exact casing the user types but uppercases on blur,
 * matching what actually goes on the wire.
 */
export function MethodSelector({ value, onChange }: MethodSelectorProps) {
  const normalised = (value || "GET").toUpperCase()
  const [custom, setCustom] = useState(!COMMON_METHODS.includes(normalised))
  const [draft, setDraft] = useState(normalised)

  useEffect(() => {
    setDraft(normalised)
    setCustom(!COMMON_METHODS.includes(normalised))
  }, [normalised])

  const commitCustom = () => {
    const next = draft.trim().toUpperCase()
    if (!next) {
      setDraft(normalised)
      return
    }
    onChange(next)
    if (COMMON_METHODS.includes(next)) setCustom(false)
  }

  if (custom) {
    return (
      <div className="flex items-center gap-1">
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value.toUpperCase())}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              e.preventDefault()
              commitCustom()
            }
          }}
          onBlur={commitCustom}
          aria-label="Custom HTTP method"
          className="h-9 w-24 font-mono text-sm font-bold"
          placeholder="METHOD"
        />
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-7"
          title="Pick from the standard methods"
          aria-label="Pick from the standard methods"
          onClick={() => {
            commitCustom()
            setCustom(false)
            onChange("GET")
          }}
        >
          <Check className="h-4 w-4" />
        </Button>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-1">
      <Select
        value={normalised}
        onValueChange={(next) => {
          if (next === "__custom__") {
            setCustom(true)
            return
          }
          onChange(next)
        }}
      >
        <SelectTrigger className="h-9 w-28 font-mono font-bold" aria-label="HTTP method">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {COMMON_METHODS.map((method) => (
            <SelectItem key={method} value={method} className="font-mono">
              {method}
            </SelectItem>
          ))}
          <SelectItem value="__custom__">
            <span className="flex items-center gap-2">
              <Pencil className="h-3.5 w-3.5" /> Custom…
            </span>
          </SelectItem>
        </SelectContent>
      </Select>
    </div>
  )
}
