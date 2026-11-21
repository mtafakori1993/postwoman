import React, { useMemo, useState } from "react"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Check, Copy, Download } from "lucide-react"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import { isSecretValue } from "@/lib/variables"
import type { Collection, Folder, Request } from "@/types"

interface CollectionDocsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  collection: Collection | null
}

function describeRequest(request: Request, depth: number): string {
  const lines: string[] = []
  const indent = "  ".repeat(depth)
  lines.push(`${indent}### ${request.name || "Untitled Request"}`)
  lines.push("")
  lines.push(`${indent}\`\`\`http`)
  lines.push(`${indent}${request.method} ${request.url}`)
  const headers = (request.headers ?? []).filter((h) => h.enabled !== false && h.key)
  for (const header of headers) lines.push(`${indent}${header.key}: ${header.value}`)
  lines.push(`${indent}\`\`\``)
  if (request.description) lines.push(`\n${indent}${request.description}`)
  const params = (request.queryParams ?? []).filter((p) => p.enabled !== false && p.key)
  if (params.length > 0) {
    lines.push("")
    lines.push(`${indent}| Query | Value | Description |`)
    lines.push(`${indent}| --- | --- | --- |`)
    for (const param of params) lines.push(`${indent}| ${param.key} | ${param.value} | ${param.description ?? ""} |`)
  }
  const body = request.body
  if (body && body.mode && body.mode !== "none") {
    lines.push("")
    lines.push(`${indent}**Body** (${body.mode})`)
    if (body.raw) lines.push(`${indent}\`\`\`${body.mode}\n${body.raw}\n${indent}\`\`\``)
    if (body.mode === "urlencoded") {
      for (const row of (body.urlencoded ?? []).filter((r) => r.enabled !== false)) lines.push(`${indent}- ${row.key} = ${row.value}`)
    }
    if (body.mode === "formdata") {
      for (const row of (body.formdata ?? []).filter((r) => r.enabled !== false)) {
        lines.push(`${indent}- ${row.key}: ${row.type === "file" ? `(file) ${row.fileName ?? row.src}` : row.value}`)
      }
    }
    if (body.mode === "graphql") {
      lines.push(`${indent}\`\`\`graphql\n${body.graphql?.query ?? ""}\n${indent}\`\`\``)
      if (body.graphql?.variables) lines.push(`${indent}Variables: \`${body.graphql.variables}\``)
    }
  }
  const auth = request.auth
  if (auth && auth.type && auth.type !== "inherit" && auth.type !== "noauth") lines.push(`\n${indent}Auth: ${auth.type}`)
  return lines.join("\n")
}

export function CollectionDocsDialog({ open, onOpenChange, collection }: CollectionDocsDialogProps) {
  const folders = useWorkspaceStore((s) => s.folders)
  const requests = useWorkspaceStore((s) => s.requests)
  const [copied, setCopied] = useState(false)

  const markdown = useMemo(() => {
    if (!collection) return ""
    const lines: string[] = [`# ${collection.info.name}`, ""]
    if (collection.info.description) lines.push(collection.info.description, "")
    const variables = (collection.variables ?? []).filter((v) => v.enabled !== false)
    if (variables.length > 0) {
      lines.push("## Variables", "", "| Name | Value | Type |", "| --- | --- | --- |")
      for (const variable of variables) {
        lines.push(`| ${variable.key} | ${isSecretValue(variable.key, variable.type) ? "<secret>" : variable.value} | ${variable.type ?? "default"} |`)
      }
      lines.push("")
    }
    if (collection.auth && collection.auth.type && collection.auth.type !== "noauth" && collection.auth.type !== "inherit") {
      lines.push("## Authentication", "", `Inherited auth: **${collection.auth.type}**`, "")
    }
    const collectionFolders = folders.filter((f) => f.collectionId === collection.id)
    const collectionRequests = requests.filter((r) => r.collectionId === collection.id)

    const renderFolder = (folder: Folder, depth: number) => {
      lines.push(`${"#".repeat(Math.min(depth + 2, 6))} ${folder.name}`, "")
      if (folder.description) lines.push(folder.description, "")
      if (folder.auth && folder.auth.type && folder.auth.type !== "inherit" && folder.auth.type !== "noauth") {
        lines.push(`Auth: **${folder.auth.type}**`, "")
      }
      for (const child of collectionFolders.filter((f) => f.parentId === folder.id)) renderFolder(child, depth + 1)
      for (const request of collectionRequests.filter((r) => r.folderId === folder.id)) lines.push(describeRequest(request, depth), "")
    }

    lines.push("## Requests", "")
    for (const request of collectionRequests.filter((r) => !r.folderId)) lines.push(describeRequest(request, 0), "")
    for (const folder of collectionFolders.filter((f) => !f.parentId)) renderFolder(folder, 0)
    return lines.join("\n")
  }, [collection, folders, requests])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Documentation — {collection?.info.name}</DialogTitle>
          <DialogDescription>Generated Markdown from the collection, its variables and every request.</DialogDescription>
        </DialogHeader>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            className="gap-1.5"
            onClick={async () => {
              await api.copyToClipboard(markdown)
              setCopied(true)
              setTimeout(() => setCopied(false), 1500)
            }}
          >
            {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />} Copy Markdown
          </Button>
          <Button variant="outline" size="sm" className="gap-1.5" onClick={() => api.saveTextFile(`${collection?.info.name ?? "collection"}.md`, markdown).catch(console.error)}>
            <Download className="h-3.5 w-3.5" /> Export .md
          </Button>
        </div>
        <pre className="max-h-[420px] overflow-auto whitespace-pre-wrap rounded-md border border-border bg-muted/20 p-3 text-xs">{markdown}</pre>
      </DialogContent>
    </Dialog>
  )
}
