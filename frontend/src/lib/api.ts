// Typed wrapper around the Wails-generated Go bindings.
//
// Every call here goes through `frontend/wailsjs/go/main/App.js`, which in turn
// invokes `window.go.main.App.*` provided by the Wails v2 runtime. There is no
// mock or browser fallback: if the runtime is missing (e.g. the page is opened
// in a plain browser instead of the Wails webview) the calls reject with a
// clear message instead of silently pretending to work.

import * as App from "../../wailsjs/go/main/App"
import { EventsOn, EventsOff, EventsEmit } from "../../wailsjs/runtime/runtime"
import type { model } from "../../wailsjs/go/models"
import type {
  AppSettings,
  Collection,
  Environment,
  Folder,
  GlobalVariable,
  KeyValuePair,
  Request,
  RunOptions,
  RunResult,
  TabState,
  AppState,
  ImportResult,
  Response,
  SearchResult,
  HistoryEntry,
  Workspace,
  WSConnection,
  MockServer,
  MockRequestLog,
  StoredCookie,
  ExtractionResult,
  JSONDiff,
  OAuth2StartResult,
  AIRequest,
  AuthConfig,
  OAuth2Config,
} from "../types"

export type * from "../types"

/** True when running inside the Wails webview with the Go backend attached. */
export function hasBackend(): boolean {
  return typeof window !== "undefined" && !!(window as any).go?.main?.App
}

/** Normalise the various shapes Wails rejects with into a readable message. */
export function errorMessage(error: unknown): string {
  if (error == null) return "Unknown error"
  if (typeof error === "string") return error
  if (error instanceof Error) return error.message
  if (typeof error === "object" && "message" in error) {
    return String((error as { message: unknown }).message)
  }
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}

async function guard<T>(fn: () => Promise<T>): Promise<T> {
  if (!hasBackend()) {
    throw new Error(
      "The Postwoman backend is not available. Launch the desktop app (or `wails dev`) instead of opening the page in a browser.",
    )
  }
  try {
    return await fn()
  } catch (error) {
    throw new Error(errorMessage(error))
  }
}

// The generated bindings expect the generated class types. Our `Plain<T>`
// values are structurally identical minus the `convertValues` helper, so a
// single cast at the boundary is safe.
const asModel = <T>(value: unknown): T => value as T

// ------------------------------------------------------------------ lifecycle

export const bootstrap = () => guard<AppState>(() => App.Bootstrap())
export const getVersion = () => guard<string>(() => App.GetAppVersion())
export const getPlatform = () => guard<string>(() => App.GetPlatform())

// ----------------------------------------------------------------- workspaces

export const getWorkspaces = () => guard<Workspace[]>(() => App.GetWorkspaces())
export const createWorkspace = (name: string, description = "") =>
  guard<Workspace>(() => App.CreateWorkspace(name, description))
export const setActiveWorkspace = (id: string) => guard<void>(() => App.SetActiveWorkspace(id))
export const getActiveWorkspaceID = () => guard<string>(() => App.GetActiveWorkspaceID())

// ---------------------------------------------------------------- collections

export const getCollections = (workspaceID = "") =>
  guard<Collection[]>(() => App.GetCollections(workspaceID))
export const getCollection = (id: string) => guard<Collection>(() => App.GetCollection(id))
export const createCollection = (workspaceID: string, name: string, description = "") =>
  guard<Collection>(() => App.CreateCollection(workspaceID, name, description))
export const updateCollection = (collection: Collection) =>
  guard<void>(() => App.UpdateCollection(asModel<model.Collection>(collection)))
export const deleteCollection = (id: string) => guard<void>(() => App.DeleteCollection(id))
export const duplicateCollection = (id: string) =>
  guard<Collection>(() => App.DuplicateCollection(id))

// -------------------------------------------------------------------- folders

export const getFolders = (collectionID = "") => guard<Folder[]>(() => App.GetFolders(collectionID))
export const createFolder = (
  collectionID: string,
  parentID: string,
  name: string,
  description = "",
) => guard<Folder>(() => App.CreateFolder(collectionID, parentID, name, description))
export const updateFolder = (folder: Folder) =>
  guard<void>(() => App.UpdateFolder(asModel<model.Folder>(folder)))
export const deleteFolder = (id: string) => guard<void>(() => App.DeleteFolder(id))

// ------------------------------------------------------------------- requests

export const getRequests = (collectionID = "", folderID = "") =>
  guard<Request[]>(() => App.GetRequests(collectionID, folderID))
export const getRequest = (id: string) => guard<Request>(() => App.GetRequest(id))
export const createRequest = (request: Request) =>
  guard<Request>(() => App.CreateRequest(asModel<model.Request>(request)))
export const updateRequest = (request: Request) =>
  guard<void>(() => App.UpdateRequest(asModel<model.Request>(request)))
export const deleteRequest = (id: string) => guard<void>(() => App.DeleteRequest(id))
export const duplicateRequest = (id: string) => guard<Request>(() => App.DuplicateRequest(id))

// -------------------------------------------------------------------- sending

export const sendRequest = (request: Request, environmentID = "") =>
  guard<Response>(() =>
    App.SendRequest(asModel<model.Request>(request), {}, {}, {}, true, environmentID),
  )

/** Cancellable send used by the editor. Call `cancelRequest` to abort. */
export const sendRequestWithID = (requestID: string, request: Request, environmentID = "") =>
  guard<Response>(() =>
    App.SendRequestWithID(requestID, asModel<model.Request>(request), environmentID),
  )

export const cancelRequest = (requestID: string) => guard<void>(() => App.CancelRequest(requestID))

/** Render the request as code for cURL/HTTP/fetch/Node/Python/Go/PHP. */
export const exportRequestCode = (request: Request, target: string) =>
  guard<string>(() => App.ExportRequestCode(asModel<model.Request>(request), target))

/** Parse a cURL command into an editable request draft. */
export const parseCurl = (command: string) => guard<Request>(() => App.ParseCurl(command))

// -------------------------------------------------------- response tooling

export const extractJSONPath = (body: string, path: string) =>
  guard<ExtractionResult>(() => App.ExtractJSONPath(body, path))

export const compareJSON = (left: string, right: string) =>
  guard<JSONDiff>(() => App.CompareJSON(left, right))

export const generateJSONSchema = (body: string) =>
  guard<string>(() => App.GenerateJSONSchema(body))

export const validateJSONSchema = (body: string, schema: string) =>
  guard<string[]>(() => App.ValidateJSONSchema(body, schema))

// ----------------------------------------------------------------- OAuth 2.0

export const oauth2Start = (request: Request, environmentID = "") =>
  guard<OAuth2StartResult>(() =>
    App.OAuth2Start(asModel<model.Request>(request), environmentID),
  )

export const oauth2Exchange = (request: Request, environmentID: string, code: string) =>
  guard<OAuth2Config>(() =>
    App.OAuth2Exchange(asModel<model.Request>(request), environmentID, code),
  )

export const oauth2FetchToken = (request: Request, environmentID = "") =>
  guard<OAuth2Config>(() =>
    App.OAuth2FetchToken(asModel<model.Request>(request), environmentID),
  )

// ------------------------------------------------------------------- GraphQL

export const graphqlIntrospect = (request: Request, environmentID = "") =>
  guard<string>(() =>
    App.GraphQLIntrospect(asModel<model.Request>(request), environmentID),
  )

// ------------------------------------------------------------------- cookies

export const listCookies = () => guard<StoredCookie[]>(() => App.ListCookies())
export const saveCookie = (input: StoredCookie) =>
  guard<void>(() => App.SaveCookie(asModel<model.StoredCookie>(input)))
export const deleteCookie = (url: string, name: string) =>
  guard<void>(() => App.DeleteCookie(url, name))
export const clearCookies = () => guard<void>(() => App.ClearCookies())

// ------------------------------------------------------------- AI assistant

export const aiAssist = (input: AIRequest) =>
  guard<string>(() => App.AIAssist(asModel<model.AIRequest>(input)))

// -------------------------------------------------------------- import URL

export const importFromURL = (workspaceID: string, sourceURL: string) =>
  guard<ImportResult>(() => App.ImportFromURL(workspaceID, sourceURL))

// --------------------------------------------------------------- websocket+

export const wsPing = (connectionID: string) => guard<void>(() => App.WSPing(connectionID))
export const wsClear = (connectionID: string) => guard<void>(() => App.WSClear(connectionID))

export const resolveVariables = (input: string, scopes: Record<string, string>) =>
  guard<string>(() => App.ResolveVariables(input, scopes))

// --------------------------------------------------------------- environments

export const getEnvironments = (workspaceID = "") =>
  guard<Environment[]>(() => App.GetEnvironments(workspaceID))
export const getEnvironment = (id: string) => guard<Environment>(() => App.GetEnvironment(id))
export const createEnvironment = (workspaceID: string, name: string) =>
  guard<Environment>(() => App.CreateEnvironment(workspaceID, name, []))
export const updateEnvironment = (env: Environment) =>
  guard<void>(() => App.UpdateEnvironment(asModel<model.Environment>(env)))
export const deleteEnvironment = (id: string) => guard<void>(() => App.DeleteEnvironment(id))
export const setActiveEnvironment = (workspaceID: string, environmentID: string) =>
  guard<void>(() => App.SetActiveEnvironment(workspaceID, environmentID))
export const getActiveEnvironment = () => guard<Environment>(() => App.GetActiveEnvironment())

// ------------------------------------------------------------ global variables

export const getGlobalVariables = () => guard<GlobalVariable[]>(() => App.GetGlobalVariables())
export const setGlobalVariable = (variable: GlobalVariable) =>
  guard<void>(() => App.SetGlobalVariable(asModel<model.GlobalVariable>(variable)))
export const deleteGlobalVariable = (key: string) => guard<void>(() => App.DeleteGlobalVariable(key))

// -------------------------------------------------------------------- history

export const getHistory = (limit = 200, offset = 0) =>
  guard<HistoryEntry[]>(() => App.GetHistory(limit, offset))
export const clearHistory = () => guard<void>(() => App.ClearHistory())
export const deleteHistoryEntry = (id: string) => guard<void>(() => App.DeleteHistoryEntry(id))

// ---------------------------------------------------------- settings and tabs

export const getSettings = () => guard<AppSettings>(() => App.GetSettings())
export const saveSettings = (settings: AppSettings) =>
  guard<void>(() => App.SaveSettings(asModel<model.AppSettings>(settings)))
export const getTabs = () => guard<TabState[]>(() => App.GetTabs())
export const saveTabs = (tabs: TabState[]) =>
  guard<void>(() => App.SaveTabs(asModel<model.Tab[]>(tabs)))

// -------------------------------------------------------------- import/export

export const importCollection = (workspaceID: string, content: string) =>
  guard<ImportResult>(() => App.ImportCollection(workspaceID, content))
export const importFromFile = (workspaceID: string) =>
  guard<ImportResult>(() => App.ImportFromFile(workspaceID))
export const exportCollection = (id: string) => guard<string>(() => App.ExportCollection(id))
export const exportCollectionToFile = (id: string) =>
  guard<string>(() => App.ExportCollectionToFile(id))
export const exportEnvironment = (id: string) => guard<string>(() => App.ExportEnvironment(id))
export const exportEnvironmentToFile = (id: string) =>
  guard<string>(() => App.ExportEnvironmentToFile(id))
export const exportRequestCurl = (request: Request) =>
  guard<string>(() => App.ExportRequestCurl(asModel<model.Request>(request)))

// --------------------------------------------------------------------- runner

export const runCollection = (opts: RunOptions) =>
  guard<RunResult>(() => App.RunCollection(asModel<model.RunOptions>(opts)))

// ------------------------------------------------------------------ websocket

export const wsConnect = (url: string, headers: KeyValuePair[] = [], protocols: string[] = []) =>
  guard<WSConnection>(() => App.WSConnect(url, asModel<model.KeyValue[]>(headers), protocols))
export const wsSend = (connectionID: string, message: string, isBinary = false) =>
  guard<void>(() => App.WSSend(connectionID, message, isBinary))
export const wsDisconnect = (connectionID: string) => guard<void>(() => App.WSDisconnect(connectionID))
export const wsList = () => guard<WSConnection[]>(() => App.WSList())
export const wsGet = (connectionID: string) => guard<WSConnection>(() => App.WSGet(connectionID))

// -------------------------------------------------------------- mock servers

export const listMockServers = () => guard<MockServer[]>(() => App.ListMockServers())
export const saveMockServer = (server: MockServer) =>
  guard<MockServer>(() => App.SaveMockServer(asModel<model.MockServer>(server)))
export const deleteMockServer = (id: string) => guard<void>(() => App.DeleteMockServer(id))
export const startMockServer = (id: string) => guard<void>(() => App.StartMockServer(id))
export const stopMockServer = (id: string) => guard<void>(() => App.StopMockServer(id))
export const mockServerLogs = (id: string) =>
  guard<MockRequestLog[]>(() => App.MockServerLogs(id))
export const clearMockServerLogs = (id: string) => guard<void>(() => App.ClearMockServerLogs(id))

// ----------------------------------------------------------- dialogs and misc

export const openFileDialog = (title: string, pattern = "*") =>
  guard<string>(() => App.OpenFileDialog(title, pattern))
export const saveFileDialog = (title: string, defaultName: string) =>
  guard<string>(() => App.SaveFileDialog(title, defaultName))
export const readTextFile = (path: string) => guard<string>(() => App.ReadTextFile(path))
export const writeTextFile = (path: string, content: string) =>
  guard<void>(() => App.WriteTextFile(path, content))
export const saveTextFile = (defaultName: string, content: string) =>
  guard<string>(() => App.SaveTextFile(defaultName, content))
export const copyToClipboard = (text: string) => guard<void>(() => App.CopyToClipboard(text))
export const showNotification = (title: string, message: string) =>
  guard<void>(() => App.ShowNotification(title, message))
export const openExternalURL = (url: string) => guard<void>(() => App.OpenExternalURL(url))
export const search = (query: string) => guard<SearchResult[]>(() => App.Search(query))

// ---------------------------------------------------------------------- events

export type Unsubscribe = () => void

export function onEvent<T = unknown>(eventName: string, handler: (data: T) => void): Unsubscribe {
  if (!hasBackend()) return () => {}
  return EventsOn(eventName, (data: T) => handler(data))
}

export function emitEvent(eventName: string, data?: unknown): void {
  if (!hasBackend()) return
  EventsEmit(eventName, data)
}

export function offEvent(eventName: string): void {
  if (!hasBackend()) return
  EventsOff(eventName)
}
