import React, { useState, useMemo, useEffect } from "react"
import { cn } from "@/lib/utils"
import { ChevronRight, ChevronDown, Copy, FileText, Minimize2, Maximize2 } from "lucide-react"

interface JSONViewerProps {
  json: string
  searchQuery?: string
  onCopyPath?: (path: string) => void
}

interface JSONNode {
  key?: string
  value: unknown
  path: string
  type: "object" | "array" | "string" | "number" | "boolean" | "null" | "undefined"
  expanded: boolean
  children?: JSONNode[]
  parent?: JSONNode
}

function parseJSON(json: string): unknown {
  try {
    return JSON.parse(json)
  } catch {
    return json
  }
}

function buildTree(value: unknown, path: string = "$", expanded: boolean = true, parent?: JSONNode): JSONNode {
  const type = getType(value)
  
  if (type === "object" && value !== null) {
    const obj = value as Record<string, unknown>
    const children = Object.entries(obj).map(([key, val]) => 
      buildTree(val, `${path}.${key}`, expanded, { key: path.split(".").pop(), value: obj, path, type, expanded } as JSONNode)
    )
    return { key: path.split(".").pop(), value: obj, path, type, expanded, children }
  }
  
  if (type === "array") {
    const arr = value as unknown[]
    const children = arr.map((val, index) => 
      buildTree(val, `${path}[${index}]`, expanded, { key: path.split(".").pop(), value: arr, path, type, expanded } as JSONNode)
    )
    return { key: path.split(".").pop(), value: arr, path, type, expanded, children }
  }
  
  return { key: path.split(".").pop(), value, path, type, expanded }
}

function getType(value: unknown): JSONNode["type"] {
  if (value === null) return "null"
  if (value === undefined) return "undefined"
  if (Array.isArray(value)) return "array"
  return typeof value as JSONNode["type"]
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
  level = 0 
}: { 
  node: JSONNode
  searchQuery?: string
  onCopyPath?: (path: string) => void
  level?: number
}) {
  const [expanded, setExpanded] = useState(node.expanded)
  const matchesSearch = useMemo(() => {
    if (!searchQuery) return false
    const query = searchQuery.toLowerCase()
    return (
      node.key?.toLowerCase().includes(query) ||
      String(node.value).toLowerCase().includes(query) ||
      node.path.toLowerCase().includes(query)
    )
  }, [node, searchQuery])

  const isComplex = node.type === "object" || node.type === "array"
  const childCount = node.children?.length || 0

  // Expand automatically when the node (or a descendant path) matches the
  // search term. This runs in an effect rather than during render.
  useEffect(() => {
    if (matchesSearch) setExpanded(true)
  }, [matchesSearch])

  const handleToggle = () => {
    setExpanded(!expanded)
  }

  const handleCopyPath = () => {
    onCopyPath?.(node.path)
  }

  return (
    <div className={cn("font-mono text-sm", matchesSearch && "bg-yellow-100/30 dark:bg-yellow-900/30")}>
      <div className="flex items-start gap-2" style={{ paddingLeft: `${level * 16}px` }}>
        {isComplex && (
          <button
            onClick={handleToggle}
            className="flex items-center justify-center w-5 h-5 text-muted-foreground hover:text-foreground rounded"
            aria-label={expanded ? "Collapse" : "Expand"}
          >
            {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>
        )}
        {!isComplex && <span className="w-5" />}

        {node.key && (
          <span className="text-purple-600 dark:text-purple-400">{node.key}</span>
        )}
        {node.key && <span className="text-muted-foreground">: </span>}

        {isComplex ? (
          <>
            <span className={cn(
              node.type === "object" ? "text-blue-600 dark:text-blue-400" : "text-orange-600 dark:text-orange-400"
            )}>
              {node.type === "object" ? "{" : "["}
            </span>
            <span className="text-muted-foreground ml-1">
              {childCount} {node.type === "object" ? "keys" : "items"}
            </span>
            <span className={cn(
              node.type === "object" ? "text-blue-600 dark:text-blue-400" : "text-orange-600 dark:text-orange-400"
            )}>
              {node.type === "object" ? "}" : "]"}
            </span>
          </>
        ) : (
          <span className={cn(
            node.type === "string" && "text-green-600 dark:text-green-400",
            node.type === "number" && "text-cyan-600 dark:text-cyan-400",
            node.type === "boolean" && "text-purple-600 dark:text-purple-400",
            node.type === "null" && "text-muted-foreground",
            node.type === "undefined" && "text-muted-foreground"
          )}>
            {formatValue(node.value, node.type)}
          </span>
        )}

        <div className="ml-auto flex items-center gap-1 opacity-0 hover:opacity-100 transition-opacity">
          {node.key && (
            <button
              onClick={handleCopyPath}
              className="p-1 text-muted-foreground hover:text-foreground rounded"
              title="Copy path"
            >
              <Copy className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>

      {isComplex && expanded && node.children && (
        <div className="border-l-2 border-border/50 ml-5">
          {node.children.map((child, index) => (
            <JSONNodeComponent
              key={index}
              node={child}
              searchQuery={searchQuery}
              onCopyPath={onCopyPath}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function JSONViewer({ json, searchQuery, onCopyPath }: JSONViewerProps) {
  const rootNode = useMemo(() => buildTree(parseJSON(json)), [json])

  return (
    <div className="h-full overflow-auto p-3 bg-background">
      <JSONNodeComponent
        node={rootNode}
        searchQuery={searchQuery}
        onCopyPath={onCopyPath}
      />
    </div>
  )
}