import React from "react"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import type {
  DiscoveredEndpoint,
  DiscoveryEndpointEdit,
  DiscoveryOptions,
  DiscoverySchemaField,
} from "@/types"

export type DiscoveryStage = "setup" | "running" | "review" | "preview"

export function baseName(path: string): string {
  const clean = path.replace(/[\\/]+$/, "")
  const parts = clean.split(/[\\/]/)
  return parts[parts.length - 1] || "discovered-api"
}

export function defaultDiscoveryOptions(): DiscoveryOptions {
  return {
    directory: "",
    collectionName: "",
    baseUrl: "http://localhost:3000",
    excludePatterns: [],
    includeTests: false,
    respectGitignore: true,
    useAi: false,
    maxFiles: 2000,
    maxFileSizeKb: 512,
    maxAiFiles: 40,
    followSymlinks: false,
    includeHidden: false,
  }
}

export function buildEdits(endpoints: DiscoveredEndpoint[]): Record<string, DiscoveryEndpointEdit> {
  const edits: Record<string, DiscoveryEndpointEdit> = {}
  for (const endpoint of endpoints) {
    edits[endpoint.id] = {
      id: endpoint.id,
      selected: endpoint.selected !== false,
      name: endpoint.name || `${endpoint.method} ${endpoint.path}`,
      description: endpoint.description ?? "",
      folder: endpoint.folder ?? "",
    }
  }
  return edits
}

export function methodClass(method: string): string {
  switch (method.toUpperCase()) {
    case "GET":
      return "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400"
    case "POST":
      return "bg-blue-500/15 text-blue-600 dark:text-blue-400"
    case "PUT":
    case "PATCH":
      return "bg-amber-500/15 text-amber-600 dark:text-amber-400"
    case "DELETE":
      return "bg-rose-500/15 text-rose-600 dark:text-rose-400"
    case "HEAD":
    case "OPTIONS":
      return "bg-violet-500/15 text-violet-600 dark:text-violet-400"
    default:
      return "bg-muted text-muted-foreground"
  }
}

export function confidenceTone(confidence: number): string {
  if (confidence >= 70) return "bg-emerald-500"
  if (confidence >= 40) return "bg-amber-500"
  return "bg-rose-500"
}

export function confidenceText(confidence: number): string {
  if (confidence >= 70) return "text-emerald-600 dark:text-emerald-400"
  if (confidence >= 40) return "text-amber-600 dark:text-amber-400"
  return "text-rose-600 dark:text-rose-400"
}

export function MethodBadge({ method, className }: { method: string; className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex min-w-[54px] items-center justify-center rounded px-1.5 py-0.5 pw-text-2xs font-semibold tracking-wide",
        methodClass(method),
        className,
      )}
    >
      {method.toUpperCase()}
    </span>
  )
}
export function ConfidenceMeter({
  value,
  showLabel = true,
  className,
}: {
  value: number
  showLabel?: boolean
  className?: string
}) {
  const clamped = Math.max(0, Math.min(100, value))
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div className="h-1.5 w-16 overflow-hidden rounded-full bg-muted">
        <div className={cn("h-full rounded-full", confidenceTone(clamped))} style={{ width: `${clamped}%` }} />
      </div>
      {showLabel && <span className={cn("pw-text-2xs tabular-nums", confidenceText(clamped))}>{clamped}%</span>}
    </div>
  )
}



export function StatCard({
  label,
  value,
  hint,
  icon: Icon,
  tone,
  className,
}: {
  label: string
  value: React.ReactNode
  hint?: string
  icon?: React.ElementType
  tone?: "default" | "success" | "warning" | "danger"
  className?: string
}) {
  const toneClass =
    tone === "success"
      ? "text-emerald-600 dark:text-emerald-400"
      : tone === "warning"
        ? "text-amber-600 dark:text-amber-400"
        : tone === "danger"
          ? "text-rose-600 dark:text-rose-400"
          : "text-foreground"
  return (
    <div className={cn("rounded-lg border bg-card/60 px-3 py-2.5", className)}>
      <div className="flex items-center justify-between gap-2 text-muted-foreground">
        <span className="pw-text-2xs font-medium uppercase tracking-wide">{label}</span>
        {Icon && <Icon className="h-3.5 w-3.5" />}
      </div>
      <p className={cn("mt-0.5 text-xl font-semibold tabular-nums leading-tight", toneClass)}>{value}</p>
      {hint && <p className="pw-text-2xs text-muted-foreground">{hint}</p>}
    </div>
  )
}

export function Panel({
  title,
  description,
  actions,
  children,
  className,
  bodyClassName,
}: {
  title: string
  description?: string
  actions?: React.ReactNode
  children: React.ReactNode
  className?: string
  bodyClassName?: string
}) {
  return (
    <section className={cn("rounded-lg border bg-card/40", className)}>
      <header className="flex items-start justify-between gap-3 border-b px-4 py-3">
        <div className="min-w-0">
          <h3 className="text-sm font-semibold">{title}</h3>
          {description && <p className="mt-0.5 pw-text-2xs text-muted-foreground">{description}</p>}
        </div>
        {actions}
      </header>
      <div className={cn("p-4", bodyClassName)}>{children}</div>
    </section>
  )
}

export function OptionRow({
  label,
  hint,
  checked,
  disabled,
  onChange,
}: {
  label: React.ReactNode
  hint?: string
  checked: boolean
  disabled?: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <label
      className={cn(
        "flex cursor-pointer items-start justify-between gap-3 rounded-md border border-transparent px-2 py-1.5 hover:border-border hover:bg-muted/40",
        disabled && "cursor-not-allowed opacity-60",
      )}
    >
      <span className="min-w-0">
        <span className="block text-sm">{label}</span>
        {hint && <span className="block pw-text-2xs text-muted-foreground">{hint}</span>}
      </span>
      <input
        type="checkbox"
        className="mt-1 h-4 w-4 shrink-0 accent-[hsl(var(--primary))]"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
      />
    </label>
  )
}

export function SchemaTree({
  fields,
  depth = 0,
}: {
  fields: DiscoverySchemaField[]
  depth?: number
}) {
  if (!fields || fields.length === 0) {
    return <p className="pw-text-2xs text-muted-foreground">No schema inferred.</p>
  }
  return (
    <div className={cn("space-y-1", depth > 0 && "mt-1 border-l border-border/70 pl-3")}>
      {fields.map((field) => (
        <div key={`${field.name}-${depth}`} className="rounded-md border bg-background/60 px-2 py-1.5">
          <div className="flex flex-wrap items-center gap-1.5">
            <span className="font-mono text-xs font-medium">{field.name}</span>
            <Badge variant="outline" className="pw-text-2xs">
              {field.type || "any"}
            </Badge>
            {field.required && (
              <Badge variant="outline" className="pw-text-2xs text-amber-600 dark:text-amber-400">
                required
              </Badge>
            )}
            {field.example && (
              <span className="min-w-0 truncate pw-text-2xs text-muted-foreground">e.g. {field.example}</span>
            )}
          </div>
          {field.description && <p className="mt-0.5 pw-text-2xs text-muted-foreground">{field.description}</p>}
          {field.children && field.children.length > 0 && <SchemaTree fields={field.children} depth={depth + 1} />}
        </div>
      ))}
    </div>
  )
}

export function EmptyState({
  title,
  description,
  icon: Icon,
  action,
}: {
  title: string
  description: string
  icon?: React.ElementType
  action?: React.ReactNode
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-center">
      {Icon && <Icon className="h-8 w-8 text-muted-foreground/60" />}
      <p className="text-sm font-medium">{title}</p>
      <p className="max-w-md pw-text-2xs text-muted-foreground">{description}</p>
      {action}
    </div>
  )
}