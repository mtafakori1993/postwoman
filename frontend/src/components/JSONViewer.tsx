import React, { useEffect, useMemo, useState } from "react"
import { cn } from "@/lib/utils"
import { Check, ChevronDown, ChevronRight, Copy, CopyPlus, Maximize2, Minimize2 } from "lucide-react"
import * as api from "@/lib/api"

interface JSONViewerProps {
  json: string
  searchQuery?: string
  onCopyPath?: (path: string) => void
}

interface JSONNode {
  key?: string
  value: unknown
  path: string
  pointer: string
  type: "object" | "array" | "string" | "number" | "boolean" | "null" | "undefined"
  expanded: boolean
  children?: JSONNode[]
}

/** Large arrays/objects are rendered lazily to keep the UI responsive. */
const LAZY_LIMIT = 300

function parseJSON(json: string): unknown {
  try {
    return JSON.parse(json)
  } catch {
    return json
  }
}

function pointerFor(parent: string, key?: string, index?: number): string {
  const escape = (value: string) => value.replace(/~/g, "~0").replace(/\//g, "~1")
  if (typeof index === "number") return `${parent}/${index}`
  if (key === undefined) return parent || ""
  return `${parent}/${escape(key)}`
}

function getType(value: unknown): JSONNode["type"] {
  if (value === null) return "null"
  if (value === undefined) return "undefined"
  if (Array.isArray(value)) return "array"
  return typeof value as JSONNode["type"]
}

function buildTree(value: unknown, path = "$", pointer = "", expanded = true, key?: string): JSONNode {
  const type = getType(value)
  if (type === "object" && value !== null) {
    const obj = value as Record<string, unknown>
    const children = Object.entries(obj).map(([childKey, childValue]) =>
      buildTree(childValue, `${path}.${childKey}`, pointerFor(pointer, childKey), expanded, childKey),
    )
    return { key, value: obj, path, pointer, type, expanded, children }
  }
  if (type === "array") {
    const arr = value as unknown[]
    const children = arr.map((childValue, index) =>
      buildTree(childValue, `${path}[${index}]`, pointerFor(pointer, undefined, index), expanded, String(index)),
    )
    return { key, value: arr, path, pointer, type, expanded, children }
  }
  return { key, value, path, pointer, type, expanded }
}

function formatValue(value: unknown, type: JSONNode["type"]): string {
  if (type === "string") return `"${value}"`
  if (type === "null") return "null"
  if (type === "undefined") return "undefined"
  return String(value)
}

function JSONNodeComponent({
  node,
  searchQuery,
  onCopyPath,
  expandSignal,
  level = 0,
}: {
  node: JSONNode
  searchQuery?: string
  onCopyPath?: (path: string) => void
  expandSignal: { mode: "expand" | "collapse"; token: number }
  level?: number
}) {
  const [expanded, setExpanded] = useState(node.expanded)
  const [showAll, setShowAll] = useState(false)
  const [copied, setCopied] = useState<string | null>(null)

  const matchesSearch = useMemo(() => {
    if (!searchQuery) return false
    const query = searchQuery.toLowerCase()
    return (
      node.key?.toLowerCase().includes(query) ||
      String(node.value).toLowerCase().includes(query) ||
      node.path.toLowerCase().includes(query)
    )
  }, [node, searchQuery])

  useEffect(() => {
    if (matchesSearch) setExpanded(true)
  }, [matchesSearch])

  useEffect(() => {
    if (expandSignal.token === 0) return
    setExpanded(expandSignal.mode === "expand")
  }, [expandSignal])

  const isComplex = node.type === "object" || node.type === "array"
  const childCount = node.children?.length || 0
  const visibleChildren = node.children
    ? showAll || childCount <= LAZY_LIMIT
      ? node.children
      : node.children.slice(0, LAZY_LIMIT)
    : []

  const copy = async (label: string, value: string) => {
    await api.copyToClipboard(value)
    setCopied(label)
    setTimeout(() => setCopied(null), 1200)
  }

  return (
    <div className={cn("font-mono text-sm", matchesSearch && "bg-yellow-100/30 dark:bg-yellow-900/30")}>
      <div className="group flex items-start gap-2" style={{ paddingLeft: `${level * 16}px` }}>
        {isComplex ? (
          <button
            onClick={() => setExpanded((value) => !value)}
            className="flex h-5 w-5 items-center justify-center rounded text-muted-foreground hover:text-foreground"
            aria-label={expanded ? "Collapse" : "Expand"}
          >
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>
        ) : (
          <span className="w-5" />
        )}

        {node.key !== undefined && <span className="text-purple-600 dark:text-purple-400">{node.key}</span>}
        {node.key !== undefined && <span className="text-muted-foreground">: </span>}

        {isComplex ? (
          <>
            <span className={cn(node.type === "object" ? "text-blue-600 dark:text-blue-400" : "text-orange-600 dark:text-orange-400")}>
              {node.type === "object" ? "{" : "["}
            </span>
            <span className="ml-1 text-muted-foreground">
              {childCount} {node.type === "object" ? "keys" : "items"}
            </span>
            <span className={cn(node.type === "object" ? "text-blue-600 dark:text-blue-400" : "text-orange-600 dark:text-orange-400")}>
              {node.type === "object" ? "}" : "]"}
            </span>
          </>
        ) : (
          <span
            className={cn(
              node.type === "string" && "text-green-600 dark:text-green-400",
              node.type === "number" && "text-cyan-600 dark:text-cyan-400",
              node.type === "boolean" && "text-purple-600 dark:text-purple-400",
              (node.type === "null" || node.type === "undefined") && "text-muted-foreground",
            )}
          >
            {formatValue(node.value, node.type)}
          </span>
        )}

        <span className="ml-1 pw-text-3xs text-muted-foreground/60">{node.type}</span>

        <div className="ml-auto flex items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
          {!isComplex && (
            <button className="rounded p-1 text-muted-foreground hover:text-foreground" title="Copy value" onClick={() => copy("value", formatValue(node.value, node.type))}>
              {copied === "value" ? <Check className="h-3 w-3 text-green-500" /> : <Copy className="h-3 w-3" />}
            </button>
          )}
          {node.key !== undefined && (
            <>
              <button
                className="rounded p-1 text-muted-foreground hover:text-foreground"
                title="Copy JSON path"
                onClick={() => {
                  void copy("path", node.path)
                  onCopyPath?.(node.path)
                }}
              >
                {copied === "path" ? <Check className="h-3 w-3 text-green-500" /> : <CopyPlus className="h-3 w-3" />}
              </button>
              <button className="rounded p-1 text-muted-foreground hover:text-foreground" title="Copy JSON Pointer" onClick={() => copy("pointer", node.pointer)}>
                {copied === "pointer" ? <Check className="h-3 w-3 text-green-500" /> : <span className="pw-text-4xs font-bold">/</span>}
              </button>
            </>
          )}
        </div>
      </div>

      {isComplex && expanded && node.children && (
        <div className="ml-5 border-l-2 border-border/50">
          {visibleChildren.map((child, index) => (
            <JSONNodeComponent
              key={`${child.path}-${index}`}
              node={child}
              searchQuery={searchQuery}
              onCopyPath={onCopyPath}
              expandSignal={expandSignal}
              level={level + 1}
            />
          ))}
          {!showAll && childCount > LAZY_LIMIT && (
            <button
              className="px-2 py-1 pw-text-2xs text-primary underline"
              onClick={() => setShowAll(true)}
            >
              Show the remaining {childCount - LAZY_LIMIT} {node.type === "object" ? "keys" : "items"}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

export function JSONViewer({ json, searchQuery, onCopyPath }: JSONViewerProps) {
  const [expandSignal, setExpandSignal] = useState<{ mode: "expand" | "collapse"; token: number }>({ mode: "expand", token: 0 })
  const rootNode = useMemo(() => buildTree(parseJSON(json)), [json])

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 border-b border-border bg-muted/20 px-2 py-1">
        <button
          className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 pw-text-2xs text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={() => setExpandSignal((s) => ({ mode: "expand", token: s.token + 1 }))}
        >
          <Maximize2 className="h-3 w-3" /> Expand all
        </button>
        <button
          className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 pw-text-2xs text-muted-foreground hover:bg-muted hover:text-foreground"
          onClick={() => setExpandSignal((s) => ({ mode: "collapse", token: s.token + 1 }))}
        >
          <Minimize2 className="h-3 w-3" /> Collapse all
        </button>
        <span className="ml-auto pw-text-3xs text-muted-foreground">hover a value to copy its value, path or JSON Pointer</span>
      </div>
      <div className="flex-1 overflow-auto p-3">
        <JSONNodeComponent node={rootNode} searchQuery={searchQuery} onCopyPath={onCopyPath} expandSignal={expandSignal} />
      </div>
    </div>
  )
}
