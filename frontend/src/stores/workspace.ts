import { create } from "zustand"
import * as api from "../lib/api"
import type {
  AppSettings,
  Collection,
  Environment,
  Folder,
  GlobalVariable,
  HistoryEntry,
  MockServer,
  Request,
  RequestHistoryEntry,
  Response,
  SidebarTab,
  TabState,
  WebSocketConnection,
  Workspace,
} from "../types"

// A tiny id generator that works even when `crypto.randomUUID` is unavailable
// (the Wails webview is not always treated as a secure context).
function uid(): string {
  const c = globalThis.crypto
  if (c && typeof c.randomUUID === "function") return c.randomUUID()
  if (c && typeof c.getRandomValues === "function") {
    const bytes = c.getRandomValues(new Uint8Array(16))
    return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")
  }
  return `${Date.now().toString(16)}-${Math.random().toString(16).slice(2)}`
}

export function defaultSettings(): AppSettings {
  return {
    theme: "system",
    fontSize: 13,
    editorFontSize: 13,
    editorTabSize: 2,
    requestTimeout: 30000,
    sslVerification: true,
    defaultHeaders: [],
    historyRetentionDays: 30,
    scriptExecutionEnabled: true,
    maxResponseSize: 50 * 1024 * 1024,
    autoSave: true,
    showRequestHeaders: true,
    showResponseHeaders: true,
    keyboardShortcuts: {},
    followRedirects: true,
    sendNoCacheHeader: false,
  }
}

export function newRequestDraft(overrides: Partial<Request> = {}): Request {
  const now = Date.now()
  return {
    id: "",
    collectionId: "",
    name: "Untitled Request",
    method: "GET",
    url: "",
    headers: [],
    queryParams: [],
    pathVariables: [],
    body: { mode: "none" },
    auth: { type: "inherit" },
    scripts: [],
    settings: {
      followRedirects: true,
      followOriginalMethod: false,
      maxRedirects: 10,
      timeout: 0,
      encoding: "auto",
    },
    order: 0,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}

interface WorkspaceState {
  ready: boolean
  initError: string | null

  workspace: Workspace | null
  workspaces: Workspace[]

  collections: Collection[]
  folders: Folder[]
  requests: Request[]

  environments: Environment[]
  activeEnvironmentId: string | null
  globalVariables: GlobalVariable[]

  history: HistoryEntry[]

  tabs: TabState[]
  activeTabId: string | null
  sendingTabId: string | null

  settings: AppSettings

  mockServers: MockServer[]
  wsConnections: WebSocketConnection[]

  // UI state
  sidebarOpen: boolean
  sidebarWidth: number
  activeSidebarTab: SidebarTab
  responsePanelHeight: number

  // lifecycle
  initialize: () => Promise<void>
  refreshAll: () => Promise<void>
  refreshCollections: () => Promise<void>
  refreshEnvironments: () => Promise<void>
  refreshHistory: () => Promise<void>
  refreshTabs: () => Promise<void>

  // workspaces
  createWorkspace: (name: string, description?: string) => Promise<Workspace | null>
  setActiveWorkspace: (id: string) => Promise<void>

  // collections
  createCollection: (name: string, description?: string) => Promise<Collection | null>
  renameCollection: (id: string, name: string) => Promise<void>
  deleteCollection: (id: string) => Promise<void>
  duplicateCollection: (id: string) => Promise<void>
  importFromFile: () => Promise<api.ImportResult | null>
  importContent: (content: string) => Promise<api.ImportResult | null>
  exportCollectionToFile: (id: string) => Promise<void>

  // folders
  createFolder: (collectionId: string, parentId: string | undefined, name: string) => Promise<Folder | null>
  renameFolder: (id: string, name: string) => Promise<void>
  deleteFolder: (id: string) => Promise<void>

  // requests
  createRequest: (input: {
    collectionId: string
    folderId?: string
    name?: string
    method?: string
    url?: string
  }) => Promise<Request | null>
  saveRequest: (request: Request) => Promise<Request | null>
  deleteRequest: (id: string) => Promise<void>
  duplicateRequest: (id: string) => Promise<void>
  openRequest: (id: string) => Promise<void>
  openHistoryEntry: (entry: RequestHistoryEntry) => void

  // tabs
  newTab: (request?: Request, title?: string) => TabState
  setActiveTab: (id: string | null) => void
  closeTab: (id: string) => void
  closeOtherTabs: (id: string) => void
  reorderTabs: (tabs: TabState[]) => void
  updateTabRequest: (tabId: string, patch: Partial<Request>) => void
  setTabResponse: (tabId: string, response: Response | undefined) => void
  markTabSaved: (tabId: string, request: Request) => void

  // sending
  sendActiveRequest: () => Promise<void>
  cancelActiveRequest: () => void

  // environments & variables
  createEnvironment: (name: string) => Promise<Environment | null>
  saveEnvironment: (env: Environment) => Promise<void>
  deleteEnvironment: (id: string) => Promise<void>
  setActiveEnvironment: (id: string | null) => Promise<void>
  setGlobalVariable: (key: string, value: string, type?: string, description?: string) => Promise<void>
  deleteGlobalVariable: (key: string) => Promise<void>

  // history
  clearHistory: () => Promise<void>
  deleteHistoryEntry: (id: string) => Promise<void>

  // settings
  updateSettings: (patch: Partial<AppSettings>) => void
  resetSettings: () => void

  // mocks / ws
  setMockServers: (servers: MockServer[]) => void
  setWSConnections: (conns: WebSocketConnection[]) => void

  // ui
  toggleSidebar: () => void
  setSidebarWidth: (width: number) => void
  setActiveSidebarTab: (tab: SidebarTab) => void
  setResponsePanelHeight: (height: number) => void
}

const LAYOUT_KEY = "postwoman.layout"

function loadLayout(): { sidebarOpen: boolean; sidebarWidth: number; responsePanelHeight: number; activeSidebarTab: SidebarTab } {
  const fallback = {
    sidebarOpen: true,
    sidebarWidth: 280,
    responsePanelHeight: 320,
    activeSidebarTab: "collections" as SidebarTab,
  }
  try {
    const raw = localStorage.getItem(LAYOUT_KEY)
    if (!raw) return fallback
    return { ...fallback, ...JSON.parse(raw) }
  } catch {
    return fallback
  }
}

function saveLayout(state: Pick<WorkspaceState, "sidebarOpen" | "sidebarWidth" | "responsePanelHeight" | "activeSidebarTab">) {
  try {
    localStorage.setItem(
      LAYOUT_KEY,
      JSON.stringify({
        sidebarOpen: state.sidebarOpen,
        sidebarWidth: state.sidebarWidth,
        responsePanelHeight: state.responsePanelHeight,
        activeSidebarTab: state.activeSidebarTab,
      }),
    )
  } catch {
    /* layout persistence is best-effort */
  }
}

// Tabs and settings are persisted to the Go backend (SQLite). Writes are
// debounced so typing in the request editor does not hammer the database.
let tabSaveTimer: ReturnType<typeof setTimeout> | null = null
let settingsSaveTimer: ReturnType<typeof setTimeout> | null = null

export const useWorkspaceStore = create<WorkspaceState>()((set, get) => {
  const layout = loadLayout()

  const scheduleTabSave = () => {
    if (tabSaveTimer) clearTimeout(tabSaveTimer)
    tabSaveTimer = setTimeout(() => {
      const tabs = get().tabs.map((t) => ({ ...t, isActive: t.id === get().activeTabId }))
      api.saveTabs(tabs).catch((err) => console.error("Failed to persist tabs:", err))
    }, 600)
  }

  const scheduleSettingsSave = () => {
    if (settingsSaveTimer) clearTimeout(settingsSaveTimer)
    settingsSaveTimer = setTimeout(() => {
      api.saveSettings(get().settings).catch((err) => console.error("Failed to persist settings:", err))
    }, 400)
  }

  return {
    ready: false,
    initError: null,

    workspace: null,
    workspaces: [],

    collections: [],
    folders: [],
    requests: [],

    environments: [],
    activeEnvironmentId: null,
    globalVariables: [],

    history: [],

    tabs: [],
    activeTabId: null,
    sendingTabId: null,

    settings: defaultSettings(),

    mockServers: [],
    wsConnections: [],

    sidebarOpen: layout.sidebarOpen,
    sidebarWidth: layout.sidebarWidth,
    activeSidebarTab: layout.activeSidebarTab,
    responsePanelHeight: layout.responsePanelHeight,

    // ------------------------------------------------------------- lifecycle

    initialize: async () => {
      try {
        const state = await api.bootstrap()
        set({
          ready: true,
          initError: null,
          workspace: state.workspace ?? null,
          workspaces: state.workspaces ?? [],
          collections: state.collections ?? [],
          folders: state.folders ?? [],
          requests: state.requests ?? [],
          environments: state.environments ?? [],
          activeEnvironmentId: state.activeEnvironmentId || null,
          globalVariables: state.globalVariables ?? [],
          history: state.history ?? [],
          tabs: state.tabs ?? [],
          activeTabId: state.tabs?.find((t) => t.isActive)?.id ?? state.tabs?.[0]?.id ?? null,
          settings: { ...defaultSettings(), ...(state.settings ?? {}) },
          mockServers: state.mockServers ?? [],
        })
      } catch (error) {
        set({ ready: true, initError: api.errorMessage(error) })
      }
    },

    refreshAll: async () => {
      const { workspace } = get()
      if (!workspace) return
      await Promise.all([
        get().refreshCollections(),
        get().refreshEnvironments(),
        get().refreshHistory(),
      ])
    },

    refreshCollections: async () => {
      const ws = get().workspace
      if (!ws) return
      try {
        const [collections, folders, requests] = await Promise.all([
          api.getCollections(ws.id),
          api.getFolders(""),
          api.getRequests("", ""),
        ])
        set({ collections, folders, requests })
      } catch (err) {
        console.error("Failed to refresh collections:", err)
      }
    },

    refreshEnvironments: async () => {
      const ws = get().workspace
      if (!ws) return
      try {
        const [environments, globalVariables] = await Promise.all([
          api.getEnvironments(ws.id),
          api.getGlobalVariables(),
        ])
        set({ environments, globalVariables })
      } catch (err) {
        console.error("Failed to refresh environments:", err)
      }
    },

    refreshHistory: async () => {
      try {
        set({ history: await api.getHistory(200, 0) })
      } catch (err) {
        console.error("Failed to refresh history:", err)
      }
    },

    refreshTabs: async () => {
      try {
        const tabs = await api.getTabs()
        if (tabs.length > 0) {
          set({
            tabs,
            activeTabId: tabs.find((t) => t.isActive)?.id ?? tabs[0].id,
          })
        }
      } catch (err) {
        console.error("Failed to refresh tabs:", err)
      }
    },

    // ------------------------------------------------------------ workspaces

    createWorkspace: async (name, description = "") => {
      try {
        const ws = await api.createWorkspace(name, description)
        await get().refreshAll()
        set((s) => ({ workspaces: [...s.workspaces, ws], workspace: ws }))
        return ws
      } catch (err) {
        console.error(err)
        return null
      }
    },

    setActiveWorkspace: async (id) => {
      try {
        await api.setActiveWorkspace(id)
        const ws = get().workspaces.find((w) => w.id === id) ?? null
        set({ workspace: ws, activeEnvironmentId: null })
        await get().refreshAll()
      } catch (err) {
        console.error(err)
      }
    },

    // ----------------------------------------------------------- collections

    createCollection: async (name, description = "") => {
      const ws = get().workspace
      if (!ws) return null
      try {
        const collection = await api.createCollection(ws.id, name, description)
        set((s) => ({ collections: [...s.collections, collection] }))
        return collection
      } catch (err) {
        console.error(err)
        return null
      }
    },

    renameCollection: async (id, name) => {
      const collection = get().collections.find((c) => c.id === id)
      if (!collection) return
      const updated = { ...collection, info: { ...collection.info, name } }
      set((s) => ({ collections: s.collections.map((c) => (c.id === id ? updated : c)) }))
      try {
        await api.updateCollection(updated)
      } catch (err) {
        console.error(err)
      }
    },

    deleteCollection: async (id) => {
      try {
        await api.deleteCollection(id)
        set((s) => ({
          collections: s.collections.filter((c) => c.id !== id),
          folders: s.folders.filter((f) => f.collectionId !== id),
          requests: s.requests.filter((r) => r.collectionId !== id),
          tabs: s.tabs.filter((t) => t.request?.collectionId !== id),
        }))
      } catch (err) {
        console.error(err)
      }
    },

    duplicateCollection: async (id) => {
      try {
        const created = await api.duplicateCollection(id)
        await get().refreshCollections()
        if (created) set((s) => ({ collections: [...s.collections.filter((c) => c.id !== created.id), created] }))
      } catch (err) {
        console.error(err)
      }
    },

    importFromFile: async () => {
      const ws = get().workspace
      if (!ws) return null
      try {
        const result = await api.importFromFile(ws.id)
        await get().refreshCollections()
        await get().refreshEnvironments()
        return result
      } catch (err) {
        throw new Error(api.errorMessage(err))
      }
    },

    importContent: async (content) => {
      const ws = get().workspace
      if (!ws) return null
      try {
        const result = await api.importCollection(ws.id, content)
        await get().refreshCollections()
        await get().refreshEnvironments()
        return result
      } catch (err) {
        throw new Error(api.errorMessage(err))
      }
    },

    exportCollectionToFile: async (id) => {
      try {
        await api.exportCollectionToFile(id)
      } catch (err) {
        console.error(err)
      }
    },

    // --------------------------------------------------------------- folders

    createFolder: async (collectionId, parentId, name) => {
      try {
        const folder = await api.createFolder(collectionId, parentId ?? "", name)
        set((s) => ({ folders: [...s.folders, folder] }))
        return folder
      } catch (err) {
        console.error(err)
        return null
      }
    },

    renameFolder: async (id, name) => {
      const folder = get().folders.find((f) => f.id === id)
      if (!folder) return
      const updated = { ...folder, name }
      set((s) => ({ folders: s.folders.map((f) => (f.id === id ? updated : f)) }))
      try {
        await api.updateFolder(updated)
      } catch (err) {
        console.error(err)
      }
    },

    deleteFolder: async (id) => {
      try {
        await api.deleteFolder(id)
        const folderIds = new Set<string>([id])
        // The backend deletes nested folders and their requests; mirror that.
        let changed = true
        while (changed) {
          changed = false
          for (const f of get().folders) {
            if (f.parentId && folderIds.has(f.parentId) && !folderIds.has(f.id)) {
              folderIds.add(f.id)
              changed = true
            }
          }
        }
        set((s) => ({
          folders: s.folders.filter((f) => !folderIds.has(f.id)),
          requests: s.requests.filter((r) => !(r.folderId && folderIds.has(r.folderId))),
        }))
      } catch (err) {
        console.error(err)
      }
    },

    // -------------------------------------------------------------- requests

    createRequest: async ({ collectionId, folderId, name, method, url }) => {
      const draft = newRequestDraft({
        collectionId,
        folderId,
        name: name || "Untitled Request",
        method: method || "GET",
        url: url || "",
      })
      try {
        const created = await api.createRequest(draft)
        set((s) => ({ requests: [...s.requests, created] }))
        return created
      } catch (err) {
        console.error(err)
        return null
      }
    },

    saveRequest: async (request) => {
      try {
        if (!request.id) {
          if (!request.collectionId) return null
          const created = await api.createRequest(request)
          set((s) => ({ requests: [...s.requests, created] }))
          return created
        }
        const saved = { ...request, updatedAt: Date.now() }
        await api.updateRequest(saved)
        set((s) => ({
          requests: s.requests.map((r) => (r.id === saved.id ? saved : r)),
        }))
        return saved
      } catch (err) {
        console.error(err)
        throw new Error(api.errorMessage(err))
      }
    },

    deleteRequest: async (id) => {
      try {
        await api.deleteRequest(id)
        set((s) => ({
          requests: s.requests.filter((r) => r.id !== id),
          tabs: s.tabs.filter((t) => t.requestId !== id),
        }))
      } catch (err) {
        console.error(err)
      }
    },

    duplicateRequest: async (id) => {
      try {
        const created = await api.duplicateRequest(id)
        set((s) => ({ requests: [...s.requests, created] }))
      } catch (err) {
        console.error(err)
      }
    },

    openRequest: async (id) => {
      const existing = get().tabs.find((t) => t.requestId === id)
      if (existing) {
        set({ activeTabId: existing.id })
        return
      }
      try {
        const request = await api.getRequest(id)
        const tab = get().newTab(request, request.name)
        set((s) => ({
          tabs: s.tabs.map((t) => (t.id === tab.id ? { ...t, requestId: request.id } : t)),
          activeTabId: tab.id,
        }))
      } catch (err) {
        console.error(err)
      }
    },

    openHistoryEntry: (entry) => {
      const request = newRequestDraft({ ...entry.request, id: "" })
      const tab = get().newTab(request, `${request.name} (history)`)
      set({ activeTabId: tab.id })
    },

    // ------------------------------------------------------------------ tabs

    newTab: (request, title) => {
      const tab: TabState = {
        id: `tab-${uid()}`,
        requestId: request?.id || undefined,
        title: title || request?.name || "Untitled Request",
        isDirty: false,
        isActive: true,
        protocol: "http",
        request: request || newRequestDraft(),
        createdAt: Date.now(),
        updatedAt: Date.now(),
      }
      set((s) => ({ tabs: [...s.tabs, tab], activeTabId: tab.id }))
      scheduleTabSave()
      return tab
    },

    setActiveTab: (id) => {
      set({ activeTabId: id })
      scheduleTabSave()
    },

    closeTab: (id) => {
      set((s) => {
        const tabs = s.tabs.filter((t) => t.id !== id)
        let activeTabId = s.activeTabId
        if (activeTabId === id) {
          activeTabId = tabs.length > 0 ? tabs[tabs.length - 1].id : null
        }
        return { tabs, activeTabId }
      })
      scheduleTabSave()
    },

    closeOtherTabs: (id) => {
      set((s) => ({ tabs: s.tabs.filter((t) => t.id === id), activeTabId: id }))
      scheduleTabSave()
    },

    reorderTabs: (tabs) => {
      set({ tabs })
      scheduleTabSave()
    },

    updateTabRequest: (tabId, patch) => {
      set((s) => ({
        tabs: s.tabs.map((t) =>
          t.id === tabId && t.request
            ? {
                ...t,
                isDirty: true,
                title: patch.name ?? t.title,
                request: { ...t.request, ...patch },
              }
            : t,
        ),
      }))
      scheduleTabSave()
    },

    setTabResponse: (tabId, response) => {
      set((s) => ({
        tabs: s.tabs.map((t) => (t.id === tabId ? { ...t, response } : t)),
      }))
      scheduleTabSave()
    },

    markTabSaved: (tabId, request) => {
      set((s) => ({
        tabs: s.tabs.map((t) =>
          t.id === tabId
            ? { ...t, requestId: request.id, title: request.name, isDirty: false, request }
            : t,
        ),
      }))
      scheduleTabSave()
    },

    // ---------------------------------------------------------------- sending

    sendActiveRequest: async () => {
      const { activeTabId, tabs, activeEnvironmentId } = get()
      const tab = tabs.find((t) => t.id === activeTabId)
      if (!tab || !tab.request) return
      set({ sendingTabId: tab.id })
      try {
        const response = await api.sendRequest(tab.request, activeEnvironmentId ?? "")
        get().setTabResponse(tab.id, response)
      } catch (error) {
        const message = api.errorMessage(error)
        get().setTabResponse(tab.id, {
          status: 0,
          statusText: "Error",
          headers: [],
          cookies: [],
          body: "",
          bodySize: 0,
          headerSize: 0,
          responseTime: 0,
          timestamp: Date.now(),
          redirected: false,
          error: message,
          tests: [],
          console: [],
        })
      } finally {
        set({ sendingTabId: null })
        // Scripts may have mutated variables or produced history entries.
        get().refreshHistory()
        get().refreshEnvironments()
      }
    },

    cancelActiveRequest: () => {
      // The Go backend cancels by request id; the UI surfaces this via the
      // active tab's request id. Cancellation of ad-hoc requests is handled by
      // the timeout, so this is a best-effort no-op for unsaved requests.
      set({ sendingTabId: null })
    },

    // -------------------------------------------------- environments & vars

    createEnvironment: async (name) => {
      const ws = get().workspace
      if (!ws) return null
      try {
        const env = await api.createEnvironment(ws.id, name)
        set((s) => ({ environments: [...s.environments, env] }))
        return env
      } catch (err) {
        console.error(err)
        return null
      }
    },

    saveEnvironment: async (env) => {
      set((s) => ({ environments: s.environments.map((e) => (e.id === env.id ? env : e)) }))
      try {
        await api.updateEnvironment(env)
      } catch (err) {
        console.error(err)
      }
    },

    deleteEnvironment: async (id) => {
      try {
        await api.deleteEnvironment(id)
        set((s) => ({
          environments: s.environments.filter((e) => e.id !== id),
          activeEnvironmentId: s.activeEnvironmentId === id ? null : s.activeEnvironmentId,
        }))
      } catch (err) {
        console.error(err)
      }
    },

    setActiveEnvironment: async (id) => {
      const ws = get().workspace
      if (!ws) return
      set({ activeEnvironmentId: id })
      try {
        await api.setActiveEnvironment(ws.id, id ?? "")
      } catch (err) {
        console.error(err)
      }
    },

    setGlobalVariable: async (key, value, type = "default", description) => {
      try {
        await api.setGlobalVariable({ key, value, type, description, enabled: true, updatedAt: Date.now() })
        await get().refreshEnvironments()
      } catch (err) {
        console.error(err)
      }
    },

    deleteGlobalVariable: async (key) => {
      try {
        await api.deleteGlobalVariable(key)
        set((s) => ({ globalVariables: s.globalVariables.filter((v) => v.key !== key) }))
      } catch (err) {
        console.error(err)
      }
    },

    // ---------------------------------------------------------------- history

    clearHistory: async () => {
      try {
        await api.clearHistory()
        set({ history: [] })
      } catch (err) {
        console.error(err)
      }
    },

    deleteHistoryEntry: async (id) => {
      try {
        await api.deleteHistoryEntry(id)
        set((s) => ({ history: s.history.filter((h) => h.id !== id) }))
      } catch (err) {
        console.error(err)
      }
    },

    // --------------------------------------------------------------- settings

    updateSettings: (patch) => {
      set((s) => ({ settings: { ...s.settings, ...patch } }))
      scheduleSettingsSave()
    },

    resetSettings: () => {
      set({ settings: defaultSettings() })
      scheduleSettingsSave()
    },

    // ------------------------------------------------------------ mocks / ws

    setMockServers: (servers) => set({ mockServers: servers }),
    setWSConnections: (conns) => set({ wsConnections: conns }),

    // --------------------------------------------------------------------- ui

    toggleSidebar: () => {
      set((s) => {
        const next = { ...s, sidebarOpen: !s.sidebarOpen }
        saveLayout(next)
        return { sidebarOpen: next.sidebarOpen }
      })
    },

    setSidebarWidth: (width) => {
      set((s) => {
        const next = { ...s, sidebarWidth: width }
        saveLayout(next)
        return { sidebarWidth: width }
      })
    },

    setActiveSidebarTab: (tab) => {
      set((s) => {
        const next = { ...s, activeSidebarTab: tab }
        saveLayout(next)
        return { activeSidebarTab: tab }
      })
    },

    setResponsePanelHeight: (height) => {
      set((s) => {
        const next = { ...s, responsePanelHeight: height }
        saveLayout(next)
        return { responsePanelHeight: height }
      })
    },
  }
})
