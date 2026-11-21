import React, { useEffect, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu"
import {
  BookOpen,
  ChevronRight,
  ChevronDown,
  Folder as FolderIcon,
  FolderOpen,
  FileText,
  MoreVertical,
  Plus,
  Download,
  Upload,
  Copy,
  Pencil,
  Settings2,
  Star,
  Trash2,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import { CollectionSettingsDialog } from "../CollectionSettingsDialog"
import { CollectionDocsDialog } from "../CollectionDocsDialog"
import { isSecretValue } from "@/lib/variables"
import type { Collection, Folder, Request } from "@/types"

// HTML5 drag-and-drop cannot reliably read custom MIME types during dragover in
// every webview, so the dragged request id is also kept in module scope.
let draggedRequestId: string | null = null

type CreateTarget =
  | { kind: "collection" }
  | { kind: "folder"; collectionId: string; parentId?: string }
  | { kind: "request"; collectionId: string; folderId?: string }

/** A single-line input that commits on Enter/blur and cancels on Escape. */
function InlineInput({
  initial,
  placeholder,
  onSubmit,
  onCancel,
  className,
}: {
  initial?: string
  placeholder?: string
  onSubmit: (value: string) => void
  onCancel: () => void
  className?: string
}) {
  const [value, setValue] = useState(initial ?? "")
  const ref = useRef<HTMLInputElement>(null)
  const done = useRef(false)

  useEffect(() => {
    ref.current?.focus()
    ref.current?.select()
  }, [])

  const commit = () => {
    if (done.current) return
    done.current = true
    const trimmed = value.trim()
    if (trimmed) onSubmit(trimmed)
    else onCancel()
  }

  return (
    <Input
      ref={ref}
      value={value}
      onChange={(e) => setValue(e.target.value)}
      placeholder={placeholder}
      className={cn("h-7 text-sm", className)}
      onKeyDown={(e) => {
        if (e.key === "Enter") commit()
        if (e.key === "Escape") {
          done.current = true
          onCancel()
        }
      }}
      onBlur={commit}
    />
  )
}

function MethodDot({ method }: { method: string }) {
  const color =
    method === "GET"
      ? "text-sky-500"
      : method === "POST"
        ? "text-green-600"
        : method === "PUT" || method === "PATCH"
          ? "text-amber-500"
          : method === "DELETE"
            ? "text-red-500"
            : "text-muted-foreground"
  return <span className={cn("font-mono text-[10px] font-bold w-9 shrink-0", color)}>{method}</span>
}

function RequestRow({
  request,
  depth,
}: {
  request: Request
  depth: number
}) {
  const { openRequest, duplicateRequest, deleteRequest, favoriteRequestIds, toggleFavoriteRequest } = useWorkspaceStore()
  const [renaming, setRenaming] = useState(false)
  const isFavorite = favoriteRequestIds.includes(request.id)

  if (renaming) {
    return (
      <div className="px-2 py-1" style={{ paddingLeft: depth * 12 + 8 }}>
        <InlineInput
          initial={request.name}
          onSubmit={async (name) => {
            setRenaming(false)
            await useWorkspaceStore.getState().saveRequest({ ...request, name })
          }}
          onCancel={() => setRenaming(false)}
        />
      </div>
    )
  }

  return (
    <div
      className="group flex items-center gap-1 px-2 py-1 hover:bg-muted/50 rounded cursor-pointer"
      style={{ paddingLeft: depth * 12 + 8 }}
      onClick={() => openRequest(request.id)}
      draggable
      onDragStart={(event) => {
        draggedRequestId = request.id
        event.dataTransfer.setData("text/plain", request.id)
        event.dataTransfer.effectAllowed = "move"
      }}
      onDragEnd={() => {
        draggedRequestId = null
      }}
    >
      <MethodDot method={request.method} />
      <span className="flex-1 text-sm truncate" title={request.description || request.name}>
        {request.name}
      </span>
      <Button
        variant="ghost"
        size="icon"
        className={cn("h-6 w-6 p-0 shrink-0", isFavorite ? "text-amber-500" : "opacity-0 group-hover:opacity-100 text-muted-foreground")}
        aria-label={isFavorite ? "Remove from favourites" : "Add to favourites"}
        title={isFavorite ? "Remove from favourites" : "Add to favourites"}
        onClick={(event) => {
          event.stopPropagation()
          toggleFavoriteRequest(request.id)
        }}
      >
        <Star className={cn("h-3.5 w-3.5", isFavorite && "fill-current")} />
      </Button>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
            onClick={(e) => e.stopPropagation()}
            aria-label="Request actions"
          >
            <MoreVertical className="h-3.5 w-3.5" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuItem onClick={() => openRequest(request.id)}>
            <FileText className="h-3.5 w-3.5 mr-2" /> Open
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => setRenaming(true)}>
            <Pencil className="h-3.5 w-3.5 mr-2" /> Rename
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => duplicateRequest(request.id)}>
            <Copy className="h-3.5 w-3.5 mr-2" /> Duplicate
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem className="text-destructive" onClick={() => deleteRequest(request.id)}>
            <Trash2 className="h-3.5 w-3.5 mr-2" /> Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  )
}

function FolderNode({
  folder,
  collectionId,
  folders,
  requests,
  depth,
  createTarget,
  setCreateTarget,
}: {
  folder: Folder
  collectionId: string
  folders: Folder[]
  requests: Request[]
  depth: number
  createTarget: CreateTarget | null
  setCreateTarget: (t: CreateTarget | null) => void
}) {
  const { deleteFolder } = useWorkspaceStore()
  const [expanded, setExpanded] = useState(true)
  const [renaming, setRenaming] = useState(false)

  const childFolders = folders.filter((f) => f.parentId === folder.id)
  const childRequests = requests.filter((r) => r.folderId === folder.id)
  const creating = createTarget && createTarget.kind !== "collection" && "parentId" in createTarget && createTarget.parentId === folder.id
  const creatingRequest =
    createTarget?.kind === "request" && createTarget.collectionId === collectionId && createTarget.folderId === folder.id

  return (
    <div>
      <div
        className="group flex items-center gap-1 px-2 py-1 hover:bg-muted/50 rounded cursor-pointer"
        style={{ paddingLeft: depth * 12 + 8 }}
        onClick={() => setExpanded((v) => !v)}
        onDragOver={(event) => {
          if (draggedRequestId) event.preventDefault()
        }}
        onDrop={(event) => {
          const id = draggedRequestId || event.dataTransfer.getData("text/plain")
          draggedRequestId = null
          if (!id) return
          event.preventDefault()
          event.stopPropagation()
          void useWorkspaceStore.getState().moveRequest(id, { collectionId, folderId: folder.id })
        }}
      >
        <Button
          variant="ghost"
          size="icon"
          className="h-4 w-4 p-0 shrink-0"
          onClick={(e) => {
            e.stopPropagation()
            setExpanded((v) => !v)
          }}
          aria-label={expanded ? "Collapse" : "Expand"}
        >
          {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
        </Button>
        {expanded ? (
          <FolderOpen className="h-4 w-4 text-amber-500 shrink-0" />
        ) : (
          <FolderIcon className="h-4 w-4 text-amber-500 shrink-0" />
        )}
        {renaming ? (
          <InlineInput
            initial={folder.name}
            onSubmit={async (name) => {
              setRenaming(false)
              await useWorkspaceStore.getState().renameFolder(folder.id, name)
            }}
            onCancel={() => setRenaming(false)}
          />
        ) : (
          <span className="flex-1 text-sm truncate font-medium" title={folder.name}>
            {folder.name}
          </span>
        )}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
              onClick={(e) => e.stopPropagation()}
              aria-label="Folder actions"
            >
              <MoreVertical className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => setCreateTarget({ kind: "request", collectionId, folderId: folder.id })}>
              <Plus className="h-3.5 w-3.5 mr-2" /> Add Request
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setCreateTarget({ kind: "folder", collectionId, parentId: folder.id })}>
              <Plus className="h-3.5 w-3.5 mr-2" /> Add Folder
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setRenaming(true)}>
              <Pencil className="h-3.5 w-3.5 mr-2" /> Rename
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={() => deleteFolder(folder.id)}>
              <Trash2 className="h-3.5 w-3.5 mr-2" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {expanded && (
        <div>
          {creating && (
            <div className="px-2 py-1" style={{ paddingLeft: (depth + 1) * 12 + 8 }}>
              <InlineInput
                placeholder="Folder name"
                onSubmit={async (name) => {
                  setCreateTarget(null)
                  await useWorkspaceStore.getState().createFolder(collectionId, folder.id, name)
                }}
                onCancel={() => setCreateTarget(null)}
              />
            </div>
          )}
          {creatingRequest && (
            <div className="px-2 py-1" style={{ paddingLeft: (depth + 1) * 12 + 8 }}>
              <InlineInput
                placeholder="Request name"
                onSubmit={async (name) => {
                  setCreateTarget(null)
                  await useWorkspaceStore.getState().createRequest({ collectionId, folderId: folder.id, name })
                }}
                onCancel={() => setCreateTarget(null)}
              />
            </div>
          )}
          {childFolders.map((f) => (
            <FolderNode
              key={f.id}
              folder={f}
              collectionId={collectionId}
              folders={folders}
              requests={requests}
              depth={depth + 1}
              createTarget={createTarget}
              setCreateTarget={setCreateTarget}
            />
          ))}
          {childRequests.map((r) => (
            <RequestRow key={r.id} request={r} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  )
}

export function CollectionsPanel() {
  const {
    collections,
    folders,
    requests,
    favoriteRequestIds,
    recentRequestIds,
    createCollection,
    createFolder,
    createRequest,
    renameCollection,
    deleteCollection,
    duplicateCollection,
    exportCollectionToFile,
    importFromFile,
  } = useWorkspaceStore()

  const [createTarget, setCreateTarget] = useState<CreateTarget | null>(null)
  const [settingsFor, setSettingsFor] = useState<Collection | null>(null)
  const [docsFor, setDocsFor] = useState<Collection | null>(null)
  const [renamingCollection, setRenamingCollection] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [importMessage, setImportMessage] = useState<string | null>(null)
  const [importError, setImportError] = useState<string | null>(null)

  const isExpanded = (id: string) => expanded[id] !== false
  const toggle = (id: string) => setExpanded((s) => ({ ...s, [id]: !isExpanded(id) }))

  const handleImport = async () => {
    setImportMessage(null)
    setImportError(null)
    try {
      const result = await importFromFile()
      if (result) setImportMessage(result.message || `Imported ${result.format}`)
    } catch (err) {
      setImportError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">Collections</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => setCreateTarget({ kind: "collection" })} aria-label="New collection" title="New collection">
          <Plus className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={handleImport} aria-label="Import" title="Import collection / environment / OpenAPI / HAR">
          <Upload className="h-4 w-4" />
        </Button>
      </div>

      {(importMessage || importError) && (
        <div className={cn("px-3 py-1.5 text-xs", importError ? "text-destructive bg-destructive/10" : "text-green-700 dark:text-green-400 bg-green-500/10")}>
          {importError || importMessage}
        </div>
      )}

      <div className="flex-1 overflow-y-auto py-1">
        {createTarget?.kind === "collection" && (
          <div className="px-2 py-1">
            <InlineInput
              placeholder="Collection name"
              onSubmit={async (name) => {
                setCreateTarget(null)
                await createCollection(name)
              }}
              onCancel={() => setCreateTarget(null)}
            />
          </div>
        )}

        {(favoriteRequestIds.length > 0 || recentRequestIds.length > 0) && (
          <div className="border-b border-border/60 pb-1">
            {favoriteRequestIds.length > 0 && (
              <>
                <div className="px-3 pt-2 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">Favourites</div>
                {favoriteRequestIds
                  .map((id) => requests.find((r) => r.id === id))
                  .filter((r): r is Request => Boolean(r))
                  .map((request) => (
                    <RequestRow key={`fav-${request.id}`} request={request} depth={0} />
                  ))}
              </>
            )}
            {recentRequestIds.length > 0 && (
              <>
                <div className="px-3 pt-2 text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">Recent</div>
                {recentRequestIds
                  .slice(0, 5)
                  .map((id) => requests.find((r) => r.id === id))
                  .filter((r): r is Request => Boolean(r))
                  .map((request) => (
                    <RequestRow key={`recent-${request.id}`} request={request} depth={0} />
                  ))}
              </>
            )}
          </div>
        )}

        {collections.length === 0 && createTarget?.kind !== "collection" && (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground">
            No collections yet. Create one or import a Postman collection, OpenAPI spec or HAR file.
          </div>
        )}

        {collections.map((collection) => (
          <div key={collection.id}>
            <div
              className="group flex items-center gap-1 px-2 py-1 hover:bg-muted/50 rounded"
              onDragOver={(event) => {
                if (draggedRequestId) event.preventDefault()
              }}
              onDrop={(event) => {
                const id = draggedRequestId || event.dataTransfer.getData("text/plain")
                draggedRequestId = null
                if (!id) return
                event.preventDefault()
                void useWorkspaceStore.getState().moveRequest(id, { collectionId: collection.id, folderId: undefined })
              }}
            >
              <Button
                variant="ghost"
                size="icon"
                className="h-4 w-4 p-0 shrink-0"
                onClick={() => toggle(collection.id)}
                aria-label={isExpanded(collection.id) ? "Collapse" : "Expand"}
              >
                {isExpanded(collection.id) ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
              </Button>
              {renamingCollection === collection.id ? (
                <InlineInput
                  initial={collection.info.name}
                  onSubmit={async (name) => {
                    setRenamingCollection(null)
                    await renameCollection(collection.id, name)
                  }}
                  onCancel={() => setRenamingCollection(null)}
                />
              ) : (
                <span
                  className="min-w-0 flex-1 cursor-pointer"
                  title={collection.info.description || collection.info.name}
                  onClick={() => toggle(collection.id)}
                >
                  <span className="block truncate text-sm font-semibold">{collection.info.name}</span>
                  {collection.info.description && (
                    <span className="block truncate text-[10px] text-muted-foreground">{collection.info.description}</span>
                  )}
                </span>
              )}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100" aria-label="Collection actions">
                    <MoreVertical className="h-3.5 w-3.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  <DropdownMenuItem onClick={() => setCreateTarget({ kind: "request", collectionId: collection.id })}>
                    <Plus className="h-3.5 w-3.5 mr-2" /> Add Request
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setCreateTarget({ kind: "folder", collectionId: collection.id })}>
                    <Plus className="h-3.5 w-3.5 mr-2" /> Add Folder
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setRenamingCollection(collection.id)}>
                    <Pencil className="h-3.5 w-3.5 mr-2" /> Rename
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setSettingsFor(collection)}>
                    <Settings2 className="h-3.5 w-3.5 mr-2" /> Settings & variables
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setDocsFor(collection)}>
                    <BookOpen className="h-3.5 w-3.5 mr-2" /> Documentation
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => duplicateCollection(collection.id)}>
                    <Copy className="h-3.5 w-3.5 mr-2" /> Duplicate
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => {
                      const secrets = (collection.variables ?? []).filter(
                        (v) => v.enabled !== false && isSecretValue(v.key, v.type),
                      )
                      if (secrets.length > 0) {
                        const confirmed = window.confirm(
                          `This collection contains ${secrets.length} secret variable(s) (${secrets
                            .map((v) => v.key)
                            .join(", ")}).\n\nExported files include variable values in clear text. Export anyway?`,
                        )
                        if (!confirmed) return
                      }
                      void exportCollectionToFile(collection.id)
                    }}
                  >
                    <Download className="h-3.5 w-3.5 mr-2" /> Export to file
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem className="text-destructive" onClick={() => deleteCollection(collection.id)}>
                    <Trash2 className="h-3.5 w-3.5 mr-2" /> Delete
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>

            {isExpanded(collection.id) && (
              <div>
                {createTarget?.kind === "folder" &&
                  createTarget.collectionId === collection.id &&
                  !createTarget.parentId && (
                    <div className="px-2 py-1" style={{ paddingLeft: 20 }}>
                      <InlineInput
                        placeholder="Folder name"
                        onSubmit={async (name) => {
                          setCreateTarget(null)
                          await createFolder(collection.id, undefined, name)
                        }}
                        onCancel={() => setCreateTarget(null)}
                      />
                    </div>
                  )}
                {createTarget?.kind === "request" &&
                  createTarget.collectionId === collection.id &&
                  !createTarget.folderId && (
                    <div className="px-2 py-1" style={{ paddingLeft: 20 }}>
                      <InlineInput
                        placeholder="Request name"
                        onSubmit={async (name) => {
                          setCreateTarget(null)
                          await createRequest({ collectionId: collection.id, name })
                        }}
                        onCancel={() => setCreateTarget(null)}
                      />
                    </div>
                  )}

                {folders
                  .filter((f) => f.collectionId === collection.id && !f.parentId)
                  .map((folder) => (
                    <FolderNode
                      key={folder.id}
                      folder={folder}
                      collectionId={collection.id}
                      folders={folders}
                      requests={requests}
                      depth={1}
                      createTarget={createTarget}
                      setCreateTarget={setCreateTarget}
                    />
                  ))}

                {requests
                  .filter((r) => r.collectionId === collection.id && !r.folderId)
                  .map((request) => (
                    <RequestRow key={request.id} request={request} depth={1} />
                  ))}
              </div>
            )}
          </div>
        ))}
      </div>

      <CollectionSettingsDialog open={settingsFor !== null} onOpenChange={(open) => !open && setSettingsFor(null)} collection={settingsFor} />
      <CollectionDocsDialog open={docsFor !== null} onOpenChange={(open) => !open && setDocsFor(null)} collection={docsFor} />
    </div>
  )
}
