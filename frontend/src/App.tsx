import React, { useCallback, useEffect, useState } from "react"
import { Sidebar } from "./components/Layout/Sidebar"
import { TopBar } from "./components/Layout/TopBar"
import { RequestTabs } from "./components/RequestTabs"
import { RequestEditor } from "./components/RequestEditor"
import { ResponseViewer } from "./components/ResponseViewer"
import { SearchPalette } from "./components/SearchPalette"
import { RunnerDialog } from "./components/RunnerDialog"
import { useWorkspaceStore, newRequestDraft } from "./stores/workspace"
import * as api from "./lib/api"
import { Plus, AlertTriangle, Loader2 } from "lucide-react"
import type { Request } from "./types"

function App() {
  const {
    ready,
    initError,
    initialize,
    tabs,
    activeTabId,
    setActiveTab,
    closeTab,
    newTab,
    reorderTabs,
    updateTabRequest,
    setTabResponse,
    markTabSaved,
    saveRequest,
    createCollection,
    collections,
    settings,
    responsePanelHeight,
    setResponsePanelHeight,
    sendingTabId,
    sendActiveRequest,
    toggleSidebar,
    refreshCollections,
    refreshHistory,
  } = useWorkspaceStore()

  const [searchOpen, setSearchOpen] = useState(false)
  const [runnerOpen, setRunnerOpen] = useState(false)
  const [toast, setToast] = useState<{ kind: "info" | "error"; text: string } | null>(null)

  const activeTab = tabs.find((t) => t.id === activeTabId)
  const activeRequest = activeTab?.request
  const activeResponse = activeTab?.response
  const isLoading = sendingTabId === activeTab?.id

  useEffect(() => {
    initialize()
  }, [initialize])

  // Backend events keep the UI in sync with script-driven changes.
  useEffect(() => {
    const unsubCollections = api.onEvent("collections:updated", () => refreshCollections())
    const unsubHistory = api.onEvent("history:updated", () => refreshHistory())
    const unsubNotify = api.onEvent<{ title: string; message: string }>("notification", (n) =>
      setToast({ kind: "info", text: `${n.title}: ${n.message}` }),
    )
    return () => {
      unsubCollections()
      unsubHistory()
      unsubNotify()
    }
  }, [refreshCollections, refreshHistory])

  useEffect(() => {
    if (!toast) return
    const timer = setTimeout(() => setToast(null), 3500)
    return () => clearTimeout(timer)
  }, [toast])

  // Theme
  useEffect(() => {
    const root = document.documentElement
    const dark =
      settings.theme === "dark" ||
      (settings.theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches)
    root.classList.toggle("dark", dark)
  }, [settings.theme])

  const handleNewTab = useCallback(() => {
    newTab(newRequestDraft())
  }, [newTab])

  const handleSave = useCallback(async () => {
    if (!activeTab?.request) return
    let request: Request = activeTab.request
    try {
      if (!request.id) {
        let collectionId = request.collectionId
        if (!collectionId) {
          if (collections.length === 0) {
            const created = await createCollection("My Collection")
            collectionId = created?.id ?? ""
          } else {
            collectionId = collections[0].id
          }
        }
        if (!collectionId) throw new Error("Could not determine a collection for this request")
        request = { ...request, collectionId, name: request.name || "Untitled Request" }
      }
      const saved = await saveRequest(request)
      if (saved) markTabSaved(activeTab.id, saved)
    } catch (err) {
      setToast({ kind: "error", text: api.errorMessage(err) })
    }
  }, [activeTab, collections, createCollection, saveRequest, markTabSaved])

  const handleImport = useCallback(async () => {
    try {
      const result = await useWorkspaceStore.getState().importFromFile()
      if (result) setToast({ kind: "info", text: result.message || `Imported ${result.format}` })
    } catch (err) {
      setToast({ kind: "error", text: api.errorMessage(err) })
    }
  }, [])

  // Global keyboard shortcuts
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      const mod = e.ctrlKey || e.metaKey
      if (!mod) return
      const key = e.key.toLowerCase()
      if (key === "k") {
        e.preventDefault()
        setSearchOpen(true)
      } else if (key === "t") {
        e.preventDefault()
        handleNewTab()
      } else if (key === "b") {
        e.preventDefault()
        toggleSidebar()
      } else if (key === "s") {
        e.preventDefault()
        handleSave()
      } else if (key === "w") {
        e.preventDefault()
        if (activeTabId) closeTab(activeTabId)
      } else if (e.key === "Enter") {
        e.preventDefault()
        sendActiveRequest()
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [handleNewTab, toggleSidebar, handleSave, activeTabId, closeTab, sendActiveRequest])

  if (!ready) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-background text-muted-foreground">
        <Loader2 className="h-6 w-6 animate-spin" />
      </div>
    )
  }

  if (initError) {
    return (
      <div className="flex h-screen w-screen items-center justify-center bg-background p-8">
        <div className="max-w-md text-center space-y-3">
          <AlertTriangle className="h-10 w-10 mx-auto text-destructive" />
          <h1 className="text-lg font-semibold">Backend unavailable</h1>
          <p className="text-sm text-muted-foreground">{initError}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-screen w-screen bg-background overflow-hidden">
      <Sidebar />

      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <TopBar
          onNewTab={handleNewTab}
          onSearch={() => setSearchOpen(true)}
          onOpenRunner={() => setRunnerOpen(true)}
          onImport={handleImport}
        />

        <RequestTabs
          tabs={tabs}
          activeTabId={activeTabId}
          onTabChange={setActiveTab}
          onTabClose={closeTab}
          onNewTab={handleNewTab}
          onTabReorder={reorderTabs}
        />

        <div className="flex-1 flex flex-col overflow-hidden relative min-h-0">
          {activeTab ? (
            <>
              <div className="flex-1 min-h-0 overflow-hidden">
                <RequestEditor
                  request={activeRequest || null}
                  onSend={sendActiveRequest}
                  onSave={handleSave}
                  onCancel={() => {}}
                  isLoading={isLoading}
                  onChange={(changes) => activeTabId && updateTabRequest(activeTabId, changes)}
                />
              </div>

              <div
                className="h-1 w-full bg-border hover:bg-primary/50 cursor-row-resize transition-colors shrink-0"
                onMouseDown={(e) => {
                  e.preventDefault()
                  const startY = e.clientY
                  const startHeight = responsePanelHeight
                  const handleMove = (moveEvent: MouseEvent) => {
                    const next = Math.max(120, Math.min(window.innerHeight - 220, startHeight + (startY - moveEvent.clientY)))
                    setResponsePanelHeight(next)
                  }
                  const handleUp = () => {
                    document.removeEventListener("mousemove", handleMove)
                    document.removeEventListener("mouseup", handleUp)
                  }
                  document.addEventListener("mousemove", handleMove)
                  document.addEventListener("mouseup", handleUp)
                }}
              />

              <div className="min-h-0 shrink-0" style={{ height: `${responsePanelHeight}px` }}>
                <ResponseViewer
                  response={activeResponse || null}
                  requestHeaders={activeRequest?.headers}
                  height={responsePanelHeight}
                  onHeightChange={setResponsePanelHeight}
                />
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground">
              <div className="text-center space-y-2">
                <p className="text-lg">No active request</p>
                <p className="text-sm">Create a new tab or open a saved request to get started</p>
                <button className="mt-2 pw-btn-primary inline-flex items-center px-4 py-2 rounded" onClick={handleNewTab}>
                  <Plus className="h-4 w-4 mr-2" />
                  New Request
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      <SearchPalette open={searchOpen} onOpenChange={setSearchOpen} />
      <RunnerDialog open={runnerOpen} onOpenChange={setRunnerOpen} />

      {toast && (
        <div
          className={
            "fixed bottom-4 right-4 z-50 px-4 py-2 rounded-md shadow-lg text-sm text-white " +
            (toast.kind === "error" ? "bg-destructive" : "bg-foreground/90")
          }
        >
          {toast.text}
        </div>
      )}
    </div>
  )
}

export default App
