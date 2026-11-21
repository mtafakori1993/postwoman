import React, { useMemo, useRef, useState } from "react"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { maskValue, useVariableScopes, type ScopedVariable } from "@/lib/variables"

interface VariableInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "onChange"> {
  value: string
  onChange: (value: string) => void
  /** Override the automatically derived scope list (useful in tests/previews). */
  variables?: ScopedVariable[]
  /** Hide the `{{ }}` trigger for fields that never take variables. */
  allowVariables?: boolean
}

const TRIGGER = /{{\s*([A-Za-z0-9_$.-]*)$/

/**
 * Text input that offers `{{variable}}` autocompletion. The popup opens while
 * the caret sits inside an unfinished placeholder, which makes the request
 * builder feel like the desktop API clients users expect.
 */
export function VariableInput({ value, onChange, variables, allowVariables = true, className, ...rest }: VariableInputProps) {
  const scoped = useVariableScopes()
  const list = variables ?? scoped
  const inputRef = useRef<HTMLInputElement>(null)
  const [query, setQuery] = useState<string | null>(null)
  const [activeIndex, setActiveIndex] = useState(0)

  const suggestions = useMemo(() => {
    if (query === null) return []
    const needle = query.toLowerCase()
    return list.filter((v) => v.key.toLowerCase().includes(needle)).slice(0, 12)
  }, [query, list])

  const open = query !== null && suggestions.length > 0

  const recompute = (next: string, caret: number | null) => {
    if (!allowVariables) {
      setQuery(null)
      return
    }
    const before = next.slice(0, caret ?? next.length)
    const match = TRIGGER.exec(before)
    if (match) {
      setQuery(match[1])
      setActiveIndex(0)
    } else {
      setQuery(null)
    }
  }

  const accept = (variable: ScopedVariable) => {
    const input = inputRef.current
    const caret = input?.selectionStart ?? value.length
    const before = value.slice(0, caret)
    if (!TRIGGER.test(before)) {
      setQuery(null)
      return
    }
    const insertStart = before.lastIndexOf("{{")
    const next = `${value.slice(0, insertStart)}{{${variable.key}}}${value.slice(caret)}`
    const nextCaret = insertStart + variable.key.length + 4
    onChange(next)
    setQuery(null)
    requestAnimationFrame(() => {
      input?.focus()
      input?.setSelectionRange(nextCaret, nextCaret)
    })
  }

  return (
    <div className="relative min-w-0 flex-1">
      <Input
        {...rest}
        ref={inputRef}
        value={value}
        className={cn("font-mono text-sm", className)}
        onChange={(e) => {
          onChange(e.target.value)
          recompute(e.target.value, e.target.selectionStart)
        }}
        onKeyDown={(e) => {
          if (open) {
            if (e.key === "ArrowDown") {
              e.preventDefault()
              setActiveIndex((i) => (i + 1) % suggestions.length)
              return
            }
            if (e.key === "ArrowUp") {
              e.preventDefault()
              setActiveIndex((i) => (i - 1 + suggestions.length) % suggestions.length)
              return
            }
            if (e.key === "Enter" || e.key === "Tab") {
              e.preventDefault()
              accept(suggestions[activeIndex])
              return
            }
            if (e.key === "Escape") {
              e.preventDefault()
              setQuery(null)
              return
            }
          }
          rest.onKeyDown?.(e)
        }}
        onBlur={(e) => {
          // Delay so a click on a suggestion registers before the popup closes.
          setTimeout(() => setQuery(null), 150)
          rest.onBlur?.(e)
        }}
      />

      {open && (
        <div
          role="listbox"
          aria-label="Variable suggestions"
          className="absolute left-0 right-0 top-full z-30 mt-1 max-h-56 overflow-y-auto rounded-md border border-border bg-popover p-1 shadow-lg"
        >
          {suggestions.map((variable, index) => (
            <button
              key={variable.key}
              type="button"
              role="option"
              aria-selected={index === activeIndex}
              onMouseDown={(e) => {
                e.preventDefault()
                accept(variable)
              }}
              onMouseEnter={() => setActiveIndex(index)}
              className={cn(
                "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-xs",
                index === activeIndex ? "bg-muted" : "hover:bg-muted/60",
              )}
            >
              <span className="font-mono truncate">{`{{${variable.key}}}`}</span>
              <span className="ml-auto truncate font-mono text-muted-foreground">
                {variable.secret ? maskValue(variable.value) : variable.value || "—"}
              </span>
              <span className="shrink-0 rounded bg-muted px-1 pw-text-3xs uppercase text-muted-foreground">
                {variable.scope.slice(0, 3)}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

interface VariableChipsProps {
  text: string
  className?: string
}

/**
 * Inline inspector for the variables referenced in a string. Secret values are
 * masked until the user deliberately reveals them.
 */
export function VariableChips({ text, className }: VariableChipsProps) {
  const variables = useVariableScopes()
  const [revealed, setRevealed] = useState<string | null>(null)
  const names = useMemo(() => {
    const found = new Set<string>()
    for (const match of text.matchAll(/{{\s*([^}\s]+)\s*}}/g)) found.add(match[1])
    return [...found]
  }, [text])

  if (names.length === 0) return null

  return (
    <div className={cn("flex flex-wrap items-center gap-1.5", className)}>
      {names.map((name) => {
        const variable = variables.find((v) => v.key === name)
        const missing = !variable
        const secret = variable?.secret ?? false
        const shown = missing
          ? "not defined"
          : secret && revealed !== name
            ? maskValue(variable!.value)
            : variable!.value || "(empty)"
        return (
          <span
            key={name}
            title={
              missing
                ? `{{${name}}} is not defined in any environment, collection or global scope`
                : `${variable!.scope} · {{${name}}} = ${secret ? "secret" : variable!.value}`
            }
            className={cn(
              "inline-flex max-w-full items-center gap-1 rounded border px-1.5 py-0.5 font-mono pw-text-2xs",
              missing
                ? "border-destructive/40 bg-destructive/10 text-destructive"
                : "border-border bg-muted/40 text-muted-foreground",
            )}
          >
            <span className="truncate">{`{{${name}}}`}</span>
            <span className="text-muted-foreground/60">=</span>
            <span className="max-w-[160px] truncate">{shown}</span>
            {secret && !missing && (
              <button
                type="button"
                className="shrink-0 pw-text-3xs underline"
                onClick={() => setRevealed((current) => (current === name ? null : name))}
              >
                {revealed === name ? "hide" : "reveal"}
              </button>
            )}
          </span>
        )
      })}
    </div>
  )
}
