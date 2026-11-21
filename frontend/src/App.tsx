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
import { Plus, AlertTriangle, Loader2, Save } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./components/ui/dialog"
import { Button } from "./components/ui/button"
import type { Request } from "./types"

function App() {
  const {
    ready,
    initError,
    initialize,
    tabs,
    activeTabId,
    setActiveTab,
    closeTabs,
    duplicateTab,
    newTab,
    reorderTabs,
    updateTabRequest,
    setTabResponse,
    settings,
    tabPreferences,
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
  const [pendingClose, setPendingClose] = useState<{
    ids: string[]
    dirtyCount: number
    dirtyTitles: string[]
    label: string
  } | null>(null)

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
    const media = window.matchMedia?.("(prefers-color-scheme: dark)")
    const apply = () => {
      const dark = settings.theme === "dark" || (settings.theme === "system" && !!media?.matches)
      root.classList.toggle("dark", dark)
    }
    apply()
    if (settings.theme === "system" && media) {
      media.addEventListener?.("change", apply)
      return () => media.removeEventListener?.("change", apply)
    }
  }, [settings.theme])

  const handleNewTab = useCallback(() => {
    newTab(newRequestDraft())
  }, [newTab])

  const requestCloseTabs = useCallback(
    (ids: string[], label: string) => {
      const currentTabs = useWorkspaceStore.getState().tabs
      const existing = new Set(currentTabs.map((tab) => tab.id))
      const targetIds = ids.filter((id) => existing.has(id))
      if (targetIds.length === 0) return
      const dirtyTabs = currentTabs.filter((tab) => targetIds.includes(tab.id) && tab.isDirty)
      if (tabPreferences.confirmCloseDirtyTabs && dirtyTabs.length > 0) {
        setPendingClose({
          ids: targetIds,
          dirtyCount: dirtyTabs.length,
          dirtyTitles: dirtyTabs.map((tab) => tab.title || "Untitled Request"),
          label,
        })
        return
      }
      closeTabs(targetIds)
    },
    [tabPreferences.confirmCloseDirtyTabs, closeTabs],
  )

  const handleCloseTab = useCallback(
    (id: string) => {
      const tab = useWorkspaceStore.getState().tabs.find((item) => item.id === id)
      requestCloseTabs([id], tab?.title || "this tab")
    },
    [requestCloseTabs],
  )

  const handleCloseOtherTabs = useCallback(
    (id: string) => {
      requestCloseTabs(
        useWorkspaceStore
          .getState()
          .tabs.filter((tab) => tab.id !== id)
          .map((tab) => tab.id),
        "the other tabs",
      )
    },
    [requestCloseTabs],
  )

  const handleCloseTabsToRight = useCallback(
    (id: string) => {
      const currentTabs = useWorkspaceStore.getState().tabs
      const index = currentTabs.findIndex((tab) => tab.id === id)
      if (index === -1) return
      requestCloseTabs(
        currentTabs.slice(index + 1).map((tab) => tab.id),
        "the tabs to the right",
      )
    },
    [requestCloseTabs],
  )

  const handleCloseAllTabs = useCallback(() => {
    requestCloseTabs(
      useWorkspaceStore.getState().tabs.map((tab) => tab.id),
      "all tabs",
    )
  }, [requestCloseTabs])

  const confirmPendingClose = useCallback(() => {
    if (!pendingClose) return
    closeTabs(pendingClose.ids)
    setPendingClose(null)
  }, [pendingClose, closeTabs])

  // Persist a single tab's request. Reads live store state (rather than closed
  // over values) so it stays correct when saving several tabs in one pass, and
  // so requests that were never saved get a collection on demand.
  const persistTabRequest = useCallback(async (tabId: string) => {
    const state = useWorkspaceStore.getState()
    const tab = state.tabs.find((item) => item.id === tabId)
    if (!tab?.request) return
    let request: Request = tab.request
    if (!request.id) {
      let collectionId = request.collectionId
      if (!collectionId) {
        if (state.collections.length === 0) {
          const created = await state.createCollection("My Collection")
          collectionId = created?.id ?? ""
        } else {
          collectionId = state.collections[0].id
        }
      }
      if (!collectionId) throw new Error("Could not determine a collection for this request")
      request = { ...request, collectionId, name: request.name || "Untitled Request" }
    }
    const saved = await state.saveRequest(request)
    if (saved) useWorkspaceStore.getState().markTabSaved(tabId, saved)
  }, [])

  const saveAndClosePending = useCallback(async () => {
    if (!pendingClose) return
    try {
      for (const id of pendingClose.ids) {
        const tab = useWorkspaceStore.getState().tabs.find((item) => item.id === id)
        if (tab?.isDirty) await persistTabRequest(id)
      }
    } catch (err) {
      setToast({ kind: "error", text: api.errorMessage(err) })
      return
    }
    closeTabs(pendingClose.ids)
    setPendingClose(null)
  }, [pendingClose, persistTabRequest, closeTabs])

  const handleSave = useCallback(async () => {
    if (!activeTab) return
    try {
      await persistTabRequest(activeTab.id)
    } catch (err) {
      setToast({ kind: "error", text: api.errorMessage(err) })
    }
  }, [activeTab, persistTabRequest])

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
        if (activeTabId) handleCloseTab(activeTabId)
      } else if (e.key === "Enter") {
        e.preventDefault()
        sendActiveRequest()
      }
    }
    window.addEventListener("keydown", onKeyDown)
    return () => window.removeEventListener("keydown", onKeyDown)
  }, [handleNewTab, toggleSidebar, handleSave, activeTabId, handleCloseTab, sendActiveRequest])

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
          onTabClose={handleCloseTab}
          onNewTab={handleNewTab}
          onTabReorder={reorderTabs}
          onTabCloseOthers={handleCloseOtherTabs}
          onTabCloseToRight={handleCloseTabsToRight}
          onTabCloseAll={handleCloseAllTabs}
          onTabDuplicate={duplicateTab}
          preferences={tabPreferences}
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

      <Dialog open={pendingClose !== null} onOpenChange={(open) => !open && setPendingClose(null)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-amber-500" />
              Unsaved changes
            </DialogTitle>
            <DialogDescription>
              {pendingClose?.dirtyCount === 1
                ? "One of the tabs you are closing has unsaved changes. Save it or discard the changes before closing."
                : `${pendingClose?.dirtyCount ?? 0} of the tabs you are closing have unsaved changes. Save them or discard the changes before closing.`}
              {pendingClose?.label ? ` (${pendingClose.label})` : ""}
            </DialogDescription>
          </DialogHeader>
          {pendingClose && pendingClose.dirtyTitles.length > 0 && (
            <ul className="max-h-32 space-y-0.5 overflow-y-auto rounded-md border border-border bg-muted/20 p-2 text-xs">
              {pendingClose.dirtyTitles.map((title, index) => (
                <li key={`${title}-${index}`} className="flex items-center gap-1.5">
                  <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-orange-500" aria-hidden />
                  <span className="truncate" title={title}>
                    {title}
                  </span>
                </li>
              ))}
            </ul>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setPendingClose(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={confirmPendingClose}>
              Discard and Close
            </Button>
            <Button onClick={saveAndClosePending}>
              <Save className="mr-1.5 h-4 w-4" />
              Save and Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

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
