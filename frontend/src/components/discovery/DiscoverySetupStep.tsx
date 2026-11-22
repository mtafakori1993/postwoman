import React from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  AlertTriangle,
  Compass,
  FileSearch,
  FolderOpen,
  Layers,
  Loader2,
  Play,
  ShieldCheck,
  Sparkles,
  Wand2,
} from "lucide-react"
import { OptionRow, Panel } from "./shared"
import type { DiscoveryOptions } from "@/types"

interface DiscoverySetupStepProps {
  options: DiscoveryOptions
  patchOptions: (patch: Partial<DiscoveryOptions>) => void
  browse: () => void
  start: () => void
  busy: boolean
  aiEnabled: boolean
}

export function DiscoverySetupStep({
  options,
  patchOptions,
  browse,
  start,
  busy,
  aiEnabled,
}: DiscoverySetupStepProps) {
  const ready = options.directory.trim().length > 0
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="min-h-0 flex-1 overflow-y-auto p-6">
        <div className="mx-auto grid max-w-[1300px] gap-4 xl:grid-cols-[minmax(0,1.05fr)_minmax(0,1fr)]">
          <div className="space-y-4">
            <Panel
              title="Project"
              description="Choose the folder the agent should read. Nothing outside it is scanned."
            >
              <div className="space-y-3">
                <div>
                  <span className="text-xs font-medium text-muted-foreground">Project directory</span>
                  <div className="mt-1 flex gap-2">
                    <Input
                      value={options.directory}
                      onChange={(event) => patchOptions({ directory: event.target.value })}
                      placeholder="/path/to/project"
                      className="h-10 font-mono text-xs"
                    />
                    <Button variant="outline" className="h-10 gap-1.5" onClick={browse}>
                      <FolderOpen className="h-4 w-4" /> Browse
                    </Button>
                  </div>
                </div>
                <div className="grid gap-3 md:grid-cols-2">
                  <div>
                    <span className="text-xs font-medium text-muted-foreground">Collection name</span>
                    <Input
                      value={options.collectionName}
                      onChange={(event) => patchOptions({ collectionName: event.target.value })}
                      placeholder="My API"
                      className="mt-1 h-10 text-sm"
                    />
                  </div>
                  <div>
                    <span className="text-xs font-medium text-muted-foreground">Base URL</span>
                    <Input
                      value={options.baseUrl}
                      onChange={(event) => patchOptions({ baseUrl: event.target.value })}
                      placeholder="http://localhost:3000"
                      className="mt-1 h-10 font-mono text-xs"
                    />
                  </div>
                </div>
              </div>
            </Panel>

            <Panel title="Scan options" description="Control what the agent considers and what it skips.">
              <div className="grid gap-1 md:grid-cols-2">
                <OptionRow
                  label="Include test files"
                  hint="Scan spec/test files for additional route usage."
                  checked={options.includeTests}
                  onChange={(checked) => patchOptions({ includeTests: checked })}
                />
                <OptionRow
                  label="Respect .gitignore"
                  hint="Skip files and folders ignored by git."
                  checked={options.respectGitignore}
                  onChange={(checked) => patchOptions({ respectGitignore: checked })}
                />
                <OptionRow
                  label="Include hidden files"
                  hint="Consider dot-files and hidden directories."
                  checked={options.includeHidden}
                  onChange={(checked) => patchOptions({ includeHidden: checked })}
                />
                <OptionRow
                  label="Follow symlinks"
                  hint="Resolve symbolic links while walking the tree."
                  checked={options.followSymlinks}
                  onChange={(checked) => patchOptions({ followSymlinks: checked })}
                />
                <OptionRow
                  label={
                    <span className="flex items-center gap-1.5">
                      <Sparkles className="h-3.5 w-3.5" /> AI enrichment
                    </span>
                  }
                  hint={
                    aiEnabled
                      ? "Ask the configured provider to fill unresolved handlers, params and schemas."
                      : "Requires an enabled provider in Settings → AI providers."
                  }
                  checked={options.useAi && aiEnabled}
                  disabled={!aiEnabled}
                  onChange={(checked) => patchOptions({ useAi: checked })}
                />
              </div>
            </Panel>

            <Panel title="Limits and exclusions" description="Keep large repositories fast and focused.">
              <div className="space-y-3">
                <div className="grid gap-3 md:grid-cols-3">
                  <div>
                    <span className="text-xs font-medium text-muted-foreground">Max files</span>
                    <Input
                      type="number"
                      min="1"
                      value={options.maxFiles}
                      onChange={(event) => patchOptions({ maxFiles: parseInt(event.target.value) || 2000 })}
                      className="mt-1 h-10 text-sm"
                    />
                  </div>
                  <div>
                    <span className="text-xs font-medium text-muted-foreground">Max file size (KB)</span>
                    <Input
                      type="number"
                      min="1"
                      value={options.maxFileSizeKb}
                      onChange={(event) => patchOptions({ maxFileSizeKb: parseInt(event.target.value) || 512 })}
                      className="mt-1 h-10 text-sm"
                    />
                  </div>
                  <div>
                    <span className="text-xs font-medium text-muted-foreground">Max files to AI</span>
                    <Input
                      type="number"
                      min="0"
                      value={options.maxAiFiles}
                      onChange={(event) => patchOptions({ maxAiFiles: parseInt(event.target.value) || 0 })}
                      className="mt-1 h-10 text-sm"
                    />
                  </div>
                </div>
                <div>
                  <span className="text-xs font-medium text-muted-foreground">
                    Exclude patterns (comma or newline separated)
                  </span>
                  <Input
                    value={options.excludePatterns.join(", ")}
                    onChange={(event) =>
                      patchOptions({
                        excludePatterns: event.target.value
                          .split(/[,\n]/)
                          .map((part) => part.trim())
                          .filter(Boolean),
                      })
                    }
                    placeholder="vendor, generated, *.min.js"
                    className="mt-1 h-10 font-mono text-xs"
                  />
                </div>
              </div>
            </Panel>
          </div>

          <div className="space-y-4">
            <Panel title="How it works" description="Four local steps, with AI only where you allow it.">
              <ol className="space-y-3">
                {[
                  {
                    icon: FileSearch,
                    title: "Walk the project",
                    text: "Respects .gitignore, skips dependencies/build output and secret files, and caps large repositories.",
                  },
                  {
                    icon: Layers,
                    title: "Detect frameworks and routes",
                    text: "Understands common route styles plus OpenAPI/Swagger/AsyncAPI documents, handlers, params, bodies and auth.",
                  },
                  {
                    icon: Wand2,
                    title: "Optionally enrich with AI",
                    text: "Sends only small, secret-redacted snippets for endpoints the static pass could not resolve.",
                  },
                  {
                    icon: Compass,
                    title: "Review before import",
                    text: "Every endpoint is editable, scored by confidence, and can be inspected against its source before you import.",
                  },
                ].map((step) => (
                  <li key={step.title} className="flex gap-3">
                    <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border bg-background">
                      <step.icon className="h-3.5 w-3.5 text-primary" />
                    </span>
                    <span>
                      <span className="block text-sm font-medium">{step.title}</span>
                      <span className="block pw-text-2xs text-muted-foreground">{step.text}</span>
                    </span>
                  </li>
                ))}
              </ol>
            </Panel>

            <Panel title="Current selection" description="A quick summary before you start.">
              <dl className="space-y-2 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <dt className="text-muted-foreground">Directory</dt>
                  <dd className="min-w-0 truncate font-mono pw-text-2xs">
                    {options.directory || "Not selected"}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <dt className="text-muted-foreground">Collection</dt>
                  <dd className="min-w-0 truncate">
                    {options.collectionName || (options.directory ? `${options.directory.split(/[\\/]/).pop()} API` : "Auto")}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <dt className="text-muted-foreground">Base URL</dt>
                  <dd className="min-w-0 truncate font-mono pw-text-2xs">{options.baseUrl || "—"}</dd>
                </div>
                <div className="flex items-center justify-between gap-3">
                  <dt className="text-muted-foreground">AI enrichment</dt>
                  <dd className="flex items-center gap-1.5">
                    <Sparkles className="h-3.5 w-3.5" />
                    {options.useAi && aiEnabled ? "Enabled" : "Static analysis only"}
                  </dd>
                </div>
              </dl>
            </Panel>

            {!aiEnabled && (
              <div className="flex items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
                <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                <span>
                  AI enrichment is disabled because no provider is enabled. Static discovery still runs.
                  Configure one under <strong>Settings → AI providers</strong> to resolve uncertain endpoints.
                </span>
              </div>
            )}
          </div>
        </div>
      </div>

      <footer className="flex shrink-0 items-center justify-between gap-4 border-t px-6 py-3">
        <p className="flex items-center gap-1.5 pw-text-2xs text-muted-foreground">
          <ShieldCheck className="h-3.5 w-3.5" />
          Files stay local. Secrets are redacted before any AI request.
        </p>
        <Button onClick={start} disabled={busy || !ready} className="gap-1.5">
          {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
          Start discovery
        </Button>
      </footer>
    </div>
  )
}