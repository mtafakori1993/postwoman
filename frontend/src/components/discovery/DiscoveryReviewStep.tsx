import React, { useEffect, useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AlertTriangle,
  ArrowLeftRight,
  Copy,
  Eye,
  FileCode2,
  FolderTree,
  ListChecks,
  Loader2,
  Search,
  ShieldCheck,
  Upload,
  XCircle,
} from "lucide-react"
import { cn } from "@/lib/utils"
import * as api from "@/lib/api"
import { ConfidenceMeter, MethodBadge, SchemaTree, StatCard, confidenceText } from "./shared"
import type {
  DiscoveredEndpoint,
  DiscoveryEndpointEdit,
  DiscoveryOptions,
  DiscoveryParam,
  DiscoverySession,
  DiscoverySourceFile,
} from "@/types"

interface DiscoveryReviewStepProps {
  session: DiscoverySession
  edits: Record<string, DiscoveryEndpointEdit>
  updateEdit: (id: string, patch: Partial<DiscoveryEndpointEdit>) => void
  selectAll: (selected: boolean, ids?: string[]) => void
  activeID: string
  setActiveID: (id: string) => void
  source: DiscoverySourceFile | null
  openSource: (endpoint: DiscoveredEndpoint) => void
  collectionName: string
  baseUrl: string
  patchOptions: (patch: Partial<DiscoveryOptions>) => void
  preview: () => void
  runImport: () => void
  busy: boolean
}

type DetailTab = "details" | "params" | "schema" | "source"
type ConfidenceFilter = "all" | "high" | "low"

export function DiscoveryReviewStep({
  session,
  edits,
  updateEdit,
  selectAll,
  activeID,
  setActiveID,
  source,
  openSource,
  collectionName,
  baseUrl,
  patchOptions,
  preview,
  runImport,
  busy,
}: DiscoveryReviewStepProps) {
  const [search, setSearch] = useState("")
  const [methodFilter, setMethodFilter] = useState("ALL")
  const [confidenceFilter, setConfidenceFilter] = useState<ConfidenceFilter>("all")
  const [folderFilter, setFolderFilter] = useState("all")
  const [detailTab, setDetailTab] = useState<DetailTab>("details")

  const endpoints = session.endpoints ?? []
  const folders = useMemo(() => {
    const set = new Set<string>(session.folders ?? [])
    for (const edit of Object.values(edits)) if (edit.folder) set.add(edit.folder)
    return [...set].sort((a, b) => a.localeCompare(b))
  }, [session.folders, edits])
  const methods = useMemo(
    () => [...new Set(endpoints.map((endpoint) => endpoint.method.toUpperCase()))].sort(),
    [endpoints],
  )

  const filtered = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return endpoints.filter((endpoint) => {
      if (methodFilter !== "ALL" && endpoint.method.toUpperCase() !== methodFilter) return false
      if (confidenceFilter === "high" && endpoint.confidence < 70) return false
      if (confidenceFilter === "low" && endpoint.confidence >= 70) return false
      if (folderFilter !== "all" && (edits[endpoint.id]?.folder ?? endpoint.folder ?? "") !== folderFilter) return false
      if (!needle) return true
      return `${endpoint.method} ${endpoint.path} ${edits[endpoint.id]?.name ?? endpoint.name ?? ""} ${endpoint.handler ?? ""}`
        .toLowerCase()
        .includes(needle)
    })
  }, [endpoints, search, methodFilter, confidenceFilter, folderFilter, edits])

  const visible = filtered.slice(0, 400)
  const selectedCount = useMemo(() => Object.values(edits).filter((edit) => edit.selected).length, [edits])
  const highCount = endpoints.filter((endpoint) => endpoint.confidence >= 70).length
  const lowCount = endpoints.length - highCount
  const activeEndpoint = endpoints.find((endpoint) => endpoint.id === activeID) ?? null
  const activeEdit = activeEndpoint ? edits[activeEndpoint.id] : undefined

  useEffect(() => {
    if (detailTab === "source" && activeEndpoint && !source) void openSource(activeEndpoint)
  }, [detailTab, activeEndpoint, source, openSource])

  const invertFiltered = () => {
    for (const endpoint of filtered) {
      updateEdit(endpoint.id, { selected: !(edits[endpoint.id]?.selected ?? true) })
    }
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="shrink-0 space-y-3 border-b px-6 py-3">
        <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <StatCard label="Endpoints" value={endpoints.length} />
          <StatCard label="Selected" value={selectedCount} tone="success" />
          <StatCard label="High confidence" value={highCount} tone="success" hint="≥ 70%" />
          <StatCard label="Low confidence" value={lowCount} tone={lowCount > 0 ? "warning" : "default"} hint="review these" />
          <StatCard label="Files" value={`${session.stats.filesAnalyzed}/${session.stats.filesSeen}`} hint="analyzed / seen" />
          <StatCard label="Frameworks" value={session.frameworks?.length ?? 0} hint={(session.frameworks ?? []).join(", ") || "—"} />
        </div>
        {session.warnings?.length > 0 && (
          <div className="rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
            <p className="flex items-center gap-1 font-medium">
              <AlertTriangle className="h-3.5 w-3.5" /> Agent warnings
            </p>
            <ul className="mt-1 list-inside list-disc space-y-0.5">
              {session.warnings.slice(0, 4).map((warning) => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <div className="flex min-h-0 flex-1">
        <div className="flex w-[46%] min-w-[380px] flex-col border-r">
          <div className="space-y-2 border-b p-3">
            <div className="relative">
              <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder="Search method, path, name or handler…"
                aria-label="Search endpoints"
                className="h-8 pl-7 text-xs"
              />
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <select
                value={methodFilter}
                onChange={(event) => setMethodFilter(event.target.value)}
                aria-label="Filter by HTTP method"
                className="h-8 rounded-md border bg-background px-2 text-xs"
              >
                <option value="ALL">All methods</option>
                {methods.map((method) => (
                  <option key={method} value={method}>
                    {method}
                  </option>
                ))}
              </select>
              <select
                value={confidenceFilter}
                onChange={(event) => setConfidenceFilter(event.target.value as ConfidenceFilter)}
                aria-label="Filter by confidence"
                className="h-8 rounded-md border bg-background px-2 text-xs"
              >
                <option value="all">All confidence</option>
                <option value="high">High (≥ 70%)</option>
                <option value="low">Low (&lt; 70%)</option>
              </select>
              <select
                value={folderFilter}
                onChange={(event) => setFolderFilter(event.target.value)}
                aria-label="Filter by folder"
                className="h-8 max-w-[180px] rounded-md border bg-background px-2 text-xs"
              >
                <option value="all">All folders</option>
                {folders.map((folder) => (
                  <option key={folder} value={folder}>
                    {folder}
                  </option>
                ))}
              </select>
            </div>
            <div className="flex flex-wrap items-center gap-1.5">
              <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => selectAll(true)}>
                <ListChecks className="h-3.5 w-3.5" /> Select all
              </Button>
              <Button size="sm" variant="outline" className="h-7 text-xs" onClick={() => selectAll(false)}>
                <XCircle className="h-3.5 w-3.5" /> None
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-7 text-xs"
                onClick={() => selectAll(true, filtered.map((endpoint) => endpoint.id))}
              >
                Select filtered
              </Button>
              <Button size="sm" variant="outline" className="h-7 text-xs" onClick={invertFiltered}>
                <ArrowLeftRight className="h-3.5 w-3.5" /> Invert filtered
              </Button>
              <span className="ml-auto pw-text-2xs text-muted-foreground">
                {filtered.length} shown · {selectedCount} selected
              </span>
            </div>
          </div>
          <ScrollArea className="min-h-0 flex-1">
            <div className="divide-y">
              {visible.length === 0 && (
                <p className="p-4 pw-text-2xs text-muted-foreground">
                  {endpoints.length === 0
                    ? "No endpoints were found. Try enabling AI enrichment or widening the scan options."
                    : "No endpoints match the current filters."}
                </p>
              )}
              {visible.map((endpoint) => {
                const edit = edits[endpoint.id]
                const selected = edit?.selected ?? true
                const folder = edit?.folder ?? endpoint.folder ?? ""
                return (
                  <div
                    key={endpoint.id}
                    role="button"
                    tabIndex={0}
                    aria-current={activeID === endpoint.id ? "true" : undefined}
                    onClick={() => setActiveID(endpoint.id)}
                    onKeyDown={(event) => {
                      if (event.key === "Enter" || event.key === " ") {
                        event.preventDefault()
                        setActiveID(endpoint.id)
                      }
                    }}
                    className={cn(
                      "flex cursor-pointer items-start gap-2.5 px-3 py-2.5 hover:bg-muted/50",
                      activeID === endpoint.id && "bg-muted",
                    )}
                  >
                    <span onClick={(event) => event.stopPropagation()} className="pt-0.5">
                      <Checkbox
                        checked={selected}
                        onCheckedChange={(checked) => updateEdit(endpoint.id, { selected: checked === true })}
                      />
                    </span>
                    <MethodBadge method={endpoint.method} className="mt-0.5" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium">{edit?.name || endpoint.path}</p>
                      <p className="truncate font-mono pw-text-2xs text-muted-foreground">{endpoint.path}</p>
                      <div className="mt-1 flex items-center gap-2">
                        {folder && (
                          <span className="flex items-center gap-1 pw-text-2xs text-muted-foreground">
                            <FolderTree className="h-3 w-3" /> {folder}
                          </span>
                        )}
                        {endpoint.authRequired && (
                          <Badge variant="outline" className="pw-text-2xs">
                            auth
                          </Badge>
                        )}
                      </div>
                    </div>
                    <ConfidenceMeter value={endpoint.confidence} />
                  </div>
                )
              })}
              {filtered.length > visible.length && (
                <p className="p-3 pw-text-2xs text-muted-foreground">
                  Showing the first {visible.length} of {filtered.length} endpoints. Narrow the filters to see the rest.
                </p>
              )}
            </div>
          </ScrollArea>
        </div>
        <div className="flex min-h-0 flex-1 flex-col">
          {!activeEndpoint ? (
            <div className="flex h-full items-center justify-center p-6 text-center">
              <p className="pw-text-2xs text-muted-foreground">Select an endpoint on the left to review and edit it.</p>
            </div>
          ) : (
            <>
              <div className="shrink-0 space-y-2 border-b px-4 py-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <MethodBadge method={activeEndpoint.method} />
                      <span className="truncate font-mono text-xs">{activeEndpoint.path}</span>
                    </div>
                    <p className="mt-1 truncate pw-text-2xs text-muted-foreground">
                      {activeEndpoint.sourceFile || "unknown source"}
                      {(activeEndpoint.line ?? 0) > 0 ? `:${activeEndpoint.line}` : ""}
                      {activeEndpoint.framework ? ` · ${activeEndpoint.framework}` : ""}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-3">
                    <div className="text-right">
                      <p className={cn("text-xs font-semibold", confidenceText(activeEndpoint.confidence))}>
                        {activeEndpoint.confidence}%
                      </p>
                      <p className="pw-text-2xs text-muted-foreground">confidence</p>
                    </div>
                    <label className="flex cursor-pointer items-center gap-1.5 text-xs">
                      <Checkbox
                        checked={activeEdit?.selected ?? true}
                        onCheckedChange={(checked) => updateEdit(activeEndpoint.id, { selected: checked === true })}
                      />
                      Include
                    </label>
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-1.5">
                  {activeEndpoint.origin && (
                    <Badge variant="outline" className="pw-text-2xs">
                      {activeEndpoint.origin}
                    </Badge>
                  )}
                  {activeEndpoint.authRequired && (
                    <Badge variant="outline" className="pw-text-2xs">
                      <ShieldCheck className="mr-1 h-3 w-3" />
                      {activeEndpoint.authType || "auth required"}
                    </Badge>
                  )}
                  {(activeEndpoint.statusCodes ?? []).map((code) => (
                    <Badge key={code} variant="outline" className="pw-text-2xs">
                      {code}
                    </Badge>
                  ))}
                </div>
              </div>
              <Tabs
                value={detailTab}
                onValueChange={(value) => setDetailTab(value as DetailTab)}
                className="flex min-h-0 flex-1 flex-col"
              >
                <TabsList className="mx-4 mt-3 w-fit">
                  <TabsTrigger value="details">Details</TabsTrigger>
                  <TabsTrigger value="params">Params</TabsTrigger>
                  <TabsTrigger value="schema">Schema</TabsTrigger>
                  <TabsTrigger value="source">Source</TabsTrigger>
                </TabsList>
                <TabsContent value="details" className="min-h-0 flex-1 overflow-auto p-4 pt-3">
                  <div className="grid gap-3">
                    <div className="grid gap-1">
                      <label className="pw-text-2xs font-medium text-muted-foreground">Request name</label>
                      <Input
                        className="h-8 text-xs"
                        value={activeEdit?.name ?? ""}
                        onChange={(event) => updateEdit(activeEndpoint.id, { name: event.target.value })}
                      />
                    </div>
                    <div className="grid gap-1">
                      <label className="pw-text-2xs font-medium text-muted-foreground">Folder</label>
                      <Input
                        className="h-8 text-xs"
                        value={activeEdit?.folder ?? activeEndpoint.folder ?? ""}
                        onChange={(event) => updateEdit(activeEndpoint.id, { folder: event.target.value })}
                      />
                    </div>
                    <div className="grid gap-1">
                      <label className="pw-text-2xs font-medium text-muted-foreground">Description</label>
                      <Textarea
                        className="min-h-[70px] text-xs"
                        value={activeEdit?.description ?? ""}
                        onChange={(event) => updateEdit(activeEndpoint.id, { description: event.target.value })}
                      />
                    </div>
                    <dl className="grid grid-cols-2 gap-x-4 gap-y-2 rounded-md border p-3 pw-text-2xs">
                      {[
                        ["Handler", activeEndpoint.handler || "—"],
                        ["Controller", activeEndpoint.controller || "—"],
                        ["Framework", activeEndpoint.framework || "—"],
                        ["Content type", activeEndpoint.contentType || "—"],
                        ["Source", activeEndpoint.sourceFile ? `${activeEndpoint.sourceFile}:${activeEndpoint.line ?? 0}` : "—"],
                        ["Origin", activeEndpoint.origin || "—"],
                      ].map(([label, value]) => (
                        <React.Fragment key={label}>
                          <dt className="text-muted-foreground">{label}</dt>
                          <dd className="truncate font-mono" title={value}>
                            {value}
                          </dd>
                        </React.Fragment>
                      ))}
                    </dl>
                    {(activeEndpoint.notes ?? []).length > 0 && (
                      <div>
                        <p className="pw-text-2xs font-medium text-muted-foreground">Notes</p>
                        <ul className="mt-1 list-inside list-disc space-y-0.5 pw-text-2xs">
                          {(activeEndpoint.notes ?? []).map((note) => (
                            <li key={note}>{note}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                    {(activeEndpoint.warnings ?? []).length > 0 && (
                      <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-2">
                        <p className="flex items-center gap-1 pw-text-2xs font-medium text-amber-700 dark:text-amber-400">
                          <AlertTriangle className="h-3 w-3" /> Endpoint warnings
                        </p>
                        <ul className="mt-1 list-inside list-disc space-y-0.5 pw-text-2xs text-amber-700 dark:text-amber-400">
                          {(activeEndpoint.warnings ?? []).map((warning) => (
                            <li key={warning}>{warning}</li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </div>
                </TabsContent>
                <TabsContent value="params" className="min-h-0 flex-1 overflow-auto p-4 pt-3">
                  <div className="space-y-4">
                    {(
                      [
                        ["Path parameters", activeEndpoint.pathParams ?? []],
                        ["Query parameters", activeEndpoint.queryParams ?? []],
                        ["Headers", activeEndpoint.headers ?? []],
                      ] as [string, DiscoveryParam[]][]
                    ).map(([label, params]) => (
                      <div key={label}>
                        <p className="pw-text-2xs font-medium text-muted-foreground">{label}</p>
                        {params.length === 0 ? (
                          <p className="mt-1 pw-text-2xs text-muted-foreground">None detected.</p>
                        ) : (
                          <table className="mt-1 w-full border-collapse pw-text-2xs">
                            <thead>
                              <tr className="text-left text-muted-foreground">
                                <th className="border-b py-1 pr-2 font-medium">Name</th>
                                <th className="border-b py-1 pr-2 font-medium">Type</th>
                                <th className="border-b py-1 pr-2 font-medium">Required</th>
                                <th className="border-b py-1 font-medium">Description</th>
                              </tr>
                            </thead>
                            <tbody>
                              {params.map((param) => (
                                <tr key={`${param.in}-${param.name}`}>
                                  <td className="border-b py-1 pr-2 font-mono">{param.name}</td>
                                  <td className="border-b py-1 pr-2">{param.type || "string"}</td>
                                  <td className="border-b py-1 pr-2">{param.required ? "yes" : "no"}</td>
                                  <td className="border-b py-1">{param.description || "—"}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        )}
                      </div>
                    ))}
                    <div>
                      <p className="pw-text-2xs font-medium text-muted-foreground">Request body example</p>
                      <pre className="mt-1 max-h-[260px] overflow-auto rounded-md border bg-muted/30 p-2 font-mono pw-text-2xs">
                        {activeEndpoint.requestBody || "No request body detected."}
                      </pre>
                    </div>
                  </div>
                </TabsContent>
                <TabsContent value="schema" className="min-h-0 flex-1 overflow-auto p-4 pt-3">
                  <div className="space-y-4">
                    <div>
                      <p className="pw-text-2xs font-medium text-muted-foreground">Request schema</p>
                      <div className="mt-1 rounded-md border p-2">
                        <SchemaTree fields={activeEndpoint.requestSchema ?? []} />
                      </div>
                    </div>
                    <div>
                      <p className="pw-text-2xs font-medium text-muted-foreground">Response schema</p>
                      <div className="mt-1 rounded-md border p-2">
                        <SchemaTree fields={activeEndpoint.responseSchema ?? []} />
                      </div>
                    </div>
                  </div>
                </TabsContent>
                <TabsContent value="source" className="min-h-0 flex-1 overflow-hidden p-0">
                  {!source ? (
                    <div className="flex h-full items-center justify-center p-6">
                      <p className="pw-text-2xs text-muted-foreground">Loading source…</p>
                    </div>
                  ) : (
                    <div className="flex h-full min-h-0 flex-col">
                      <div className="flex items-center justify-between gap-2 border-b px-4 py-2">
                        <span className="truncate font-mono pw-text-2xs text-muted-foreground">
                          {source.path} · lines {source.startLine}–{source.endLine} of {source.totalLines}
                        </span>
                        <div className="flex shrink-0 gap-1.5">
                          <Button
                            size="sm"
                            variant="ghost"
                            className="h-7 gap-1.5 text-xs"
                            onClick={() => api.copyToClipboard(source.content)}
                          >
                            <Copy className="h-3.5 w-3.5" /> Copy
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 gap-1.5 text-xs"
                            onClick={() => void openSource(activeEndpoint)}
                          >
                            <FileCode2 className="h-3.5 w-3.5" /> Reload
                          </Button>
                        </div>
                      </div>
                      <pre className="min-h-0 flex-1 overflow-auto bg-muted/20 p-3 font-mono pw-text-2xs leading-5">
                        {source.content.split("\n").map((line, index) => (
                          <div key={index} className="flex">
                            <span className="mr-3 w-10 shrink-0 select-none text-right text-muted-foreground">
                              {source.startLine + index}
                            </span>
                            <span className="whitespace-pre-wrap break-words">{line || " "}</span>
                          </div>
                        ))}
                      </pre>
                      {source.truncated && (
                        <p className="border-t px-3 py-1 pw-text-2xs text-muted-foreground">
                          Showing a window around the route. Open the file in your editor for full context.
                        </p>
                      )}
                    </div>
                  )}
                </TabsContent>
              </Tabs>
            </>
          )}
        </div>
      </div>

      <footer className="flex shrink-0 flex-wrap items-center justify-between gap-3 border-t px-6 py-3">
        <div className="flex flex-1 items-center gap-2">
          <div className="min-w-[200px] flex-1">
            <Input
              className="h-8 text-xs"
              value={collectionName}
              onChange={(event) => patchOptions({ collectionName: event.target.value })}
              placeholder="Collection name"
            />
          </div>
          <div className="min-w-[200px] flex-1">
            <Input
              className="h-8 font-mono text-xs"
              value={baseUrl}
              onChange={(event) => patchOptions({ baseUrl: event.target.value })}
              placeholder="http://localhost:3000"
            />
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" className="gap-1.5" onClick={preview} disabled={busy || selectedCount === 0}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Eye className="h-4 w-4" />}
            Preview collection
          </Button>
          <Button className="gap-1.5" onClick={runImport} disabled={busy || selectedCount === 0}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
            Import {selectedCount} endpoint{selectedCount === 1 ? "" : "s"}
          </Button>
        </div>
      </footer>
    </div>
  )
}