import React, { useEffect, useMemo, useRef } from "react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  CheckCircle2,
  FileCode2,
  FileSearch,
  Layers,
  Loader2,
  RefreshCw,
  Route,
  ShieldCheck,
  Sparkles,
  Square,
  Wand2,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { StatCard } from "./shared"
import type { DiscoveryProgress, DiscoverySession } from "@/types"

interface DiscoveryRunningStepProps {
  session: DiscoverySession | null
  log: DiscoveryProgress[]
  cancel: () => void
  refresh: () => void
}

const STAGE_LABELS: Record<string, string> = {
  init: "Starting",
  scanning: "Scanning files",
  framework: "Detecting framework",
  routes: "Extracting routes",
  schemas: "Inferring schemas",
  auth: "Detecting auth",
  relationships: "Resolving relations",
  ai: "AI enrichment",
  generating: "Generating collection",
  done: "Complete",
  cancelled: "Cancelled",
  failed: "Failed",
}

const STAGE_ORDER = [
  "scanning",
  "framework",
  "routes",
  "schemas",
  "auth",
  "relationships",
  "ai",
  "generating",
]

export function DiscoveryRunningStep({ session, log, cancel, refresh }: DiscoveryRunningStepProps) {
  const logRef = useRef<HTMLDivElement>(null)
  const percent = Math.max(0, Math.min(100, session?.progress.percent ?? 0))
  const currentStage = session?.progress.stage || "init"
  const reached = useMemo(() => new Set(log.map((entry) => entry.stage)), [log])
  const currentIndex = STAGE_ORDER.indexOf(currentStage)

  useEffect(() => {
    const node = logRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [log])

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 flex-1 overflow-y-auto p-6">
        <div className="mx-auto max-w-[1300px] space-y-4">
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard label="Files seen" value={session?.progress.filesSeen ?? session?.stats.filesSeen ?? 0} icon={FileSearch} />
            <StatCard
              label="Endpoints found"
              value={session?.progress.endpoints ?? session?.endpoints?.length ?? 0}
              icon={Route}
              tone="success"
            />
            <StatCard
              label="Route files"
              value={session?.stats.routeFiles ?? 0}
              icon={FileCode2}
              hint={`${session?.stats.filesAnalyzed ?? 0} analyzed`}
            />
            <StatCard
              label="AI files sent"
              value={session?.stats.aiFilesSent ?? 0}
              icon={Sparkles}
              hint={session?.options.useAi ? "Enrichment enabled" : "Static only"}
            />
          </div>

          <div className="rounded-lg border bg-card/40 p-5">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <Loader2 className="h-5 w-5 animate-spin text-primary" />
                <div>
                  <p className="text-sm font-semibold">{session?.progress.message || "Starting the discovery agent…"}</p>
                  <p className="pw-text-2xs text-muted-foreground">
                    {STAGE_LABELS[currentStage] || currentStage} · {session?.files?.length ?? 0} files considered
                  </p>
                </div>
              </div>
              <span className="text-3xl font-semibold tabular-nums">{percent}%</span>
            </div>
            <div className="mt-4 h-2 w-full overflow-hidden rounded-full bg-muted">
              <div className="h-full rounded-full bg-primary transition-all duration-500" style={{ width: `${percent}%` }} />
            </div>

            <ol className="mt-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
              {STAGE_ORDER.map((stage, index) => {
                const done = reached.has(stage) || (currentIndex > index && currentStage !== "failed")
                const active = currentStage === stage
                return (
                  <li
                    key={stage}
                    className={cn(
                      "flex items-center gap-2 rounded-md border px-2.5 py-2 text-xs",
                      active && "border-primary/60 bg-primary/5 text-foreground",
                      !active && done && "border-emerald-500/40 bg-emerald-500/5",
                      !active && !done && "text-muted-foreground",
                    )}
                  >
                    {done && !active ? (
                      <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-500" />
                    ) : active ? (
                      <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-primary" />
                    ) : (
                      <span className="h-3.5 w-3.5 shrink-0 rounded-full border" />
                    )}
                    <span className="truncate">{STAGE_LABELS[stage] || stage}</span>
                  </li>
                )
              })}
            </ol>

            {session?.progress.current && (
              <p className="mt-3 truncate rounded bg-muted/50 px-2 py-1 font-mono pw-text-2xs text-muted-foreground">
                {session.progress.current}
              </p>
            )}
          </div>

          <div className="rounded-lg border bg-card/40">
            <header className="flex items-center justify-between gap-3 border-b px-4 py-2.5">
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-semibold">Live agent log</span>
                <Badge variant="outline" className="pw-text-2xs">
                  {log.length} updates
                </Badge>
              </div>
              <span className="pw-text-2xs text-muted-foreground">Auto-scrolling</span>
            </header>
            <div ref={logRef} className="h-[260px] overflow-auto bg-muted/20 p-3 font-mono pw-text-2xs leading-5">
              {log.length === 0 && <p className="text-muted-foreground">Waiting for the first update…</p>}
              {log.map((entry, index) => (
                <div key={`${index}-${entry.stage}`} className="flex gap-2">
                  <span className="w-10 shrink-0 text-right text-muted-foreground">{entry.percent}%</span>
                  <span className="w-24 shrink-0 truncate text-primary">{entry.stage}</span>
                  <span className="min-w-0 break-words">{entry.message}</span>
                </div>
              ))}
            </div>
          </div>

          <p className="flex items-center gap-1.5 pw-text-2xs text-muted-foreground">
            <ShieldCheck className="h-3.5 w-3.5" />
            The agent reads files locally. AI enrichment only receives small, secret-redacted snippets.
          </p>
        </div>
      </div>

      <footer className="flex shrink-0 items-center justify-between gap-3 border-t px-6 py-3">
        <span className="flex items-center gap-1.5 pw-text-2xs text-muted-foreground">
          <Wand2 className="h-3.5 w-3.5" /> You can cancel at any time; partial results are discarded.
        </span>
        <div className="flex gap-2">
          <Button variant="ghost" className="gap-1.5" onClick={refresh}>
            <RefreshCw className="h-4 w-4" /> Refresh
          </Button>
          <Button variant="outline" className="gap-1.5" onClick={cancel}>
            <Square className="h-3.5 w-3.5" /> Cancel discovery
          </Button>
        </div>
      </footer>
    </div>
  )
}