import { create } from "zustand"
import * as api from "../lib/api"
import { deepClone } from "../lib/utils"
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
  StoredCookie,
  TabState,
  TabPreferences,
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
    fontSize: 15,
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
    accentColor: "orange",
    editorFontFamily: "mono",
    uiFont: "system",
    compactMode: false,
    layoutPreset: "default",
    aiEnabled: false,
    aiProvider: "openai",
    aiBaseUrl: "https://api.openai.com/v1",
    aiApiKey: "",
    aiModel: "gpt-4o-mini",
    aiTemperature: 0.2,
    aiMaxTokens: 2048,
    aiTimeoutSeconds: 60,
    aiProviders: [
      {
        id: "openai",
        name: "OpenAI",
        kind: "openai",
        baseUrl: "https://api.openai.com/v1",
        apiKey: "",
        apiKeySet: false,
        model: "gpt-4o-mini",
        temperature: 0.2,
        maxTokens: 2048,
        timeoutSeconds: 60,
      },
    ],
    aiProviderId: "openai",
    aiApiKeySet: false,
  }
}

export function defaultTabPreferences(): TabPreferences {
  return {
    confirmCloseDirtyTabs: true,
    closeTabOnMiddleClick: true,
    tabCloseButton: "hover",
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
  tabPreferences: TabPreferences

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
  closeTabs: (ids: string[]) => void
  closeOtherTabs: (id: string) => void
  duplicateTab: (id: string) => TabState | null
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
  setTabPreferences: (patch: Partial<TabPreferences>) => void

  // url focus signal (Ctrl+L)
  focusUrlSignal: number
  requestUrlFocus: () => void

  // tabs: pinning and recently closed
  pinnedTabIds: string[]
  togglePinTab: (id: string) => void
  closedTabs: TabState[]
  restoreLastClosedTab: () => TabState | null

  // favourites and recents (persisted locally)
  favoriteRequestIds: string[]
  toggleFavoriteRequest: (id: string) => void
  recentRequestIds: string[]
  openDraft: (request: Request, title?: string) => TabState

  // collections: organise and metadata
  moveRequest: (requestId: string, target: { collectionId: string; folderId?: string }) => Promise<void>
  moveFolder: (folderId: string, target: { collectionId?: string; parentId?: string }) => Promise<void>
  updateCollectionMeta: (collection: Collection) => Promise<void>
  copyActiveRequestCurl: () => Promise<void>
  importFromURL: (sourceURL: string) => Promise<api.ImportResult | null>

  // cookies
  cookies: StoredCookie[]
  refreshCookies: () => Promise<void>
  saveCookie: (input: StoredCookie) => Promise<void>
  deleteCookie: (url: string, name: string) => Promise<void>
  clearCookies: () => Promise<void>

  // layout presets
  applyLayoutPreset: (preset: string) => void

  // cancelling
  sendingRequestId: string | null
}

const LAYOUT_KEY = "postwoman.layout"
const PINNED_TABS_KEY = "postwoman.pinnedTabs"
const CLOSED_TABS_KEY = "postwoman.closedTabs"
const FAVORITES_KEY = "postwoman.favorites"
const RECENTS_KEY = "postwoman.recentRequests"

function loadStringArray(key: string): string[] {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed.filter((v) => typeof v === "string") : []
  } catch {
    return []
  }
}

function saveStringArray(key: string, values: string[]) {
  try {
    localStorage.setItem(key, JSON.stringify(values))
  } catch {
    /* best effort */
  }
}

function loadClosedTabs(): TabState[] {
  try {
    const raw = localStorage.getItem(CLOSED_TABS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? (parsed as TabState[]) : []
  } catch {
    return []
  }
}

function saveClosedTabs(tabs: TabState[]) {
  try {
    localStorage.setItem(CLOSED_TABS_KEY, JSON.stringify(tabs.slice(-15)))
  } catch {
    /* best effort */
  }
}

/** Stable ordering used when requests/folders are reordered. */
function compareOrder<T extends { order?: number; name?: string }>(a: T, b: T): number {
  const ao = a.order ?? 0
  const bo = b.order ?? 0
  if (ao !== bo) return ao - bo
  return (a.name || "").localeCompare(b.name || "")
}

/** Layout presets applied from the settings panel or the command palette. */
export function layoutPresetValues(preset: string): {
  sidebarOpen: boolean
  sidebarWidth: number
  responsePanelHeight: number
  activeSidebarTab: SidebarTab
} {
  switch (preset) {
    case "minimal":
      return { sidebarOpen: false, sidebarWidth: 280, responsePanelHeight: 240, activeSidebarTab: "collections" }
    case "testing":
      return { sidebarOpen: true, sidebarWidth: 320, responsePanelHeight: 380, activeSidebarTab: "history" }
    case "json":
      return { sidebarOpen: true, sidebarWidth: 240, responsePanelHeight: 520, activeSidebarTab: "collections" }
    case "wide":
      return {
        sidebarOpen: true,
        sidebarWidth: 220,
        responsePanelHeight: Math.max(360, Math.round((globalThis.innerHeight || 800) * 0.6)),
        activeSidebarTab: "collections",
      }
    default:
      return { sidebarOpen: true, sidebarWidth: 280, responsePanelHeight: 320, activeSidebarTab: "collections" }
  }
}

function loadLayout(): {
  sidebarOpen: boolean
  sidebarWidth: number
  responsePanelHeight: number
  activeSidebarTab: SidebarTab
  tabPreferences: TabPreferences
} {
  const fallback = {
    sidebarOpen: true,
    sidebarWidth: 280,
    responsePanelHeight: 320,
    activeSidebarTab: "collections" as SidebarTab,
    tabPreferences: defaultTabPreferences(),
  }
  try {
    const raw = localStorage.getItem(LAYOUT_KEY)
    if (!raw) return fallback
    const parsed = JSON.parse(raw) as Partial<typeof fallback>
    return {
      ...fallback,
      ...parsed,
      tabPreferences: { ...fallback.tabPreferences, ...(parsed.tabPreferences ?? {}) },
    }
  } catch {
    return fallback
  }
}

function saveLayout(
  state: Pick<
    WorkspaceState,
    "sidebarOpen" | "sidebarWidth" | "responsePanelHeight" | "activeSidebarTab" | "tabPreferences"
  >,
) {
  try {
    localStorage.setItem(
      LAYOUT_KEY,
      JSON.stringify({
        sidebarOpen: state.sidebarOpen,
        sidebarWidth: state.sidebarWidth,
        responsePanelHeight: state.responsePanelHeight,
        activeSidebarTab: state.activeSidebarTab,
        tabPreferences: state.tabPreferences,
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
    tabPreferences: layout.tabPreferences,

    focusUrlSignal: 0,
    pinnedTabIds: loadStringArray(PINNED_TABS_KEY),
    closedTabs: loadClosedTabs(),
    favoriteRequestIds: loadStringArray(FAVORITES_KEY),
    recentRequestIds: loadStringArray(RECENTS_KEY),
    cookies: [],
    sendingRequestId: null,

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
        const tabIds = get().tabs.filter((t) => t.requestId === id).map((t) => t.id)
        if (tabIds.length > 0) {
          get().closeTabs(tabIds)
        }
        set((s) => ({
          requests: s.requests.filter((r) => r.id !== id),
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
      const recents = [id, ...get().recentRequestIds.filter((item) => item !== id)].slice(0, 20)
      set({ recentRequestIds: recents })
      saveStringArray(RECENTS_KEY, recents)
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
      get().closeTabs([id])
    },

    closeTabs: (ids) => {
      const remove = new Set(ids)
      if (remove.size === 0) return
      const removed = get().tabs.filter((t) => remove.has(t.id) && t.request)
      if (removed.length > 0) {
        const closed = [...get().closedTabs, ...removed.map((t) => ({ ...deepClone(t), pinned: false }))]
        const trimmed = closed.slice(-15)
        set({ closedTabs: trimmed })
        saveClosedTabs(trimmed)
      }
      set((s) => {
        const currentIndex = s.activeTabId ? s.tabs.findIndex((t) => t.id === s.activeTabId) : -1
        const tabs = s.tabs.filter((t) => !remove.has(t.id))
        let activeTabId = s.activeTabId
        if (activeTabId && remove.has(activeTabId)) {
          const nextRight = s.tabs.slice(currentIndex + 1).find((t) => !remove.has(t.id))
          const nextLeft = currentIndex > 0
            ? [...s.tabs.slice(0, currentIndex)].reverse().find((t) => !remove.has(t.id))
            : undefined
          activeTabId = nextRight?.id ?? nextLeft?.id ?? null
        }
        if (activeTabId && !tabs.some((t) => t.id === activeTabId)) {
          activeTabId = tabs[0]?.id ?? null
        }
        return { tabs, activeTabId }
      })
      scheduleTabSave()
    },

    closeOtherTabs: (id) => {
      const tab = get().tabs.find((t) => t.id === id)
      if (!tab) return
      get().closeTabs(get().tabs.filter((t) => t.id !== id).map((t) => t.id))
      set({ activeTabId: id })
      scheduleTabSave()
    },

    duplicateTab: (id) => {
      const source = get().tabs.find((t) => t.id === id)
      if (!source) return null
      const baseRequest = source.request ? deepClone(source.request) : newRequestDraft()
      const title = `${source.title || baseRequest.name || "Untitled Request"} copy`
      const request: Request = {
        ...baseRequest,
        id: "",
        name: `${baseRequest.name || "Untitled Request"} copy`,
      }
      const tab: TabState = {
        id: `tab-${uid()}`,
        requestId: undefined,
        title,
        isDirty: Boolean(source.requestId || source.request?.id || source.isDirty),
        isActive: true,
        protocol: source.protocol || "http",
        request,
        response: source.response ? deepClone(source.response) : undefined,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      }
      set((s) => {
        const index = s.tabs.findIndex((t) => t.id === id)
        const tabs = [...s.tabs]
        tabs.splice(index === -1 ? tabs.length : index + 1, 0, tab)
        return { tabs, activeTabId: tab.id }
      })
      scheduleTabSave()
      return tab
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
      const requestId = `${tab.id}:${Date.now()}`
      set({ sendingTabId: tab.id, sendingRequestId: requestId })
      try {
        const response = await api.sendRequestWithID(requestId, tab.request, activeEnvironmentId ?? "")
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
        set((s) => ({
          sendingTabId: s.sendingTabId === tab.id ? null : s.sendingTabId,
          sendingRequestId: s.sendingRequestId === requestId ? null : s.sendingRequestId,
        }))
        // Scripts may have mutated variables or produced history entries.
        get().refreshHistory()
        get().refreshEnvironments()
      }
    },

    cancelActiveRequest: () => {
      const { sendingRequestId } = get()
      if (sendingRequestId) {
        api.cancelRequest(sendingRequestId).catch((err) => console.error("Failed to cancel request:", err))
      }
      set({ sendingTabId: null, sendingRequestId: null })
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

    setTabPreferences: (patch) => {
      set((s) => {
        const tabPreferences = { ...s.tabPreferences, ...patch }
        const next = { ...s, tabPreferences }
        saveLayout(next)
        return { tabPreferences }
      })
    },

    // ------------------------------------------------- focus / tabs / recents

    requestUrlFocus: () => set((s) => ({ focusUrlSignal: s.focusUrlSignal + 1 })),

    togglePinTab: (id) => {
      const current = get().pinnedTabIds
      const next = current.includes(id) ? current.filter((t) => t !== id) : [...current, id]
      saveStringArray(PINNED_TABS_KEY, next)
      set((s) => {
        // Pinned tabs are kept at the front of the strip, matching editors
        // like VS Code.
        const pinned = s.tabs.filter((t) => next.includes(t.id))
        const unpinned = s.tabs.filter((t) => !next.includes(t.id))
        const tabs = [...pinned, ...unpinned].map((t) => ({ ...t, pinned: next.includes(t.id) }))
        return { pinnedTabIds: next, tabs }
      })
      scheduleTabSave()
    },

    restoreLastClosedTab: () => {
      const closed = get().closedTabs
      if (closed.length === 0) return null
      const last = closed[closed.length - 1]
      const remaining = closed.slice(0, -1)
      set({ closedTabs: remaining })
      saveClosedTabs(remaining)
      const draft = last.request
        ? { ...deepClone(last.request), id: "", collectionId: last.request?.collectionId ?? "" }
        : newRequestDraft()
      return get().openDraft(draft, last.title)
    },

    openDraft: (request, title) => get().newTab(request, title),

    toggleFavoriteRequest: (id) => {
      const current = get().favoriteRequestIds
      const next = current.includes(id) ? current.filter((r) => r !== id) : [...current, id]
      saveStringArray(FAVORITES_KEY, next)
      set({ favoriteRequestIds: next })
    },

    // ---------------------------------------------- organise collections tree

    moveRequest: async (requestId, target) => {
      const request = get().requests.find((r) => r.id === requestId)
      if (!request) return
      const siblings = get().requests.filter(
        (r) => r.id !== requestId
          && r.collectionId === target.collectionId
          && (r.folderId || "") === (target.folderId || ""),
      )
      const updated: Request = {
        ...request,
        collectionId: target.collectionId,
        folderId: target.folderId,
        order: siblings.length,
        updatedAt: Date.now(),
      }
      set((s) => ({
        requests: [...s.requests.filter((r) => r.id !== requestId), updated].sort(compareOrder),
      }))
      try {
        await api.updateRequest(updated)
      } catch (err) {
        console.error("Failed to move request:", err)
      }
    },

    moveFolder: async (folderId, target) => {
      const folder = get().folders.find((f) => f.id === folderId)
      if (!folder) return
      const nextCollection = target.collectionId ?? folder.collectionId
      const updated: Folder = {
        ...folder,
        collectionId: nextCollection,
        parentId: target.parentId,
        updatedAt: Date.now(),
      }
      set((s) => ({ folders: s.folders.map((f) => (f.id === folderId ? updated : f)) }))
      try {
        await api.updateFolder(updated)
      } catch (err) {
        console.error("Failed to move folder:", err)
      }
    },

    updateCollectionMeta: async (collection) => {
      set((s) => ({ collections: s.collections.map((c) => (c.id === collection.id ? collection : c)) }))
      try {
        await api.updateCollection(collection)
      } catch (err) {
        console.error("Failed to update collection:", err)
        throw new Error(api.errorMessage(err))
      }
    },

    copyActiveRequestCurl: async () => {
      const { activeTabId, tabs } = get()
      const tab = tabs.find((t) => t.id === activeTabId)
      if (!tab?.request) return
      const curl = await api.exportRequestCurl(tab.request)
      await api.copyToClipboard(curl)
    },

    importFromURL: async (sourceURL) => {
      const ws = get().workspace
      if (!ws) return null
      const result = await api.importFromURL(ws.id, sourceURL)
      await get().refreshCollections()
      await get().refreshEnvironments()
      return result
    },

    // ---------------------------------------------------------------- cookies

    refreshCookies: async () => {
      try {
        set({ cookies: await api.listCookies() })
      } catch (err) {
        console.error("Failed to load cookies:", err)
      }
    },

    saveCookie: async (input) => {
      await api.saveCookie(input)
      await get().refreshCookies()
    },

    deleteCookie: async (url, name) => {
      await api.deleteCookie(url, name)
      await get().refreshCookies()
    },

    clearCookies: async () => {
      await api.clearCookies()
      set({ cookies: [] })
    },

    // ----------------------------------------------------------- layout presets

    applyLayoutPreset: (preset) => {
      const values = layoutPresetValues(preset)
      set((s) => {
        const next = { ...s, ...values }
        saveLayout(next)
        return values
      })
      get().updateSettings({ layoutPreset: preset })
    },
  }
})
