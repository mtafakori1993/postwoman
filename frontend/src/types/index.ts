// Frontend types.
//
// Persistent entities mirror the Wails-generated Go models so the TypeScript
// types can never drift from what the backend actually serialises. The
// generated classes carry a `convertValues` helper method, which is irrelevant
// to consumers of plain JSON, so `Plain<T>` strips it recursively and lets the
// UI build ordinary object literals.

import type { model } from "../../wailsjs/go/models"

/** Recursively drop the generated `convertValues` helper from a model type. */
export type Plain<T> = T extends (infer U)[]
  ? Plain<U>[]
  : T extends object
    ? { [K in keyof T as K extends "convertValues" ? never : K]: Plain<T[K]> }
    : T

// --- core entities (mirror internal/model) --------------------------------

export type Workspace = Plain<model.Workspace>
export type Collection = Plain<model.Collection>
export type CollectionInfo = Plain<model.CollectionInfo>
export type CollectionEvent = Plain<model.CollectionEvent>
export type Folder = Plain<model.Folder>
export type Request = Plain<model.Request>
export type RequestBody = Plain<model.Body>
export type GraphQLBody = Plain<model.GraphQLBody>
export type AuthConfig = Plain<model.Auth>
export type OAuth2Config = Plain<model.OAuth2Auth>
export type RequestSettings = Plain<model.Settings>
export type RequestScript = Plain<model.Script>
export type Environment = Plain<model.Environment>
export type Variable = Plain<model.Variable>
export type GlobalVariable = Plain<model.GlobalVariable>
export type Response = Plain<model.Response>
export type Cookie = Plain<model.Cookie>
export type TestResult = Plain<model.TestResult>
export type ConsoleEntry = Plain<model.ConsoleEntry>
export type RequestHistoryEntry = Plain<model.HistoryEntry>
export type HistoryEntry = Plain<model.HistoryEntry>
export type TabState = Plain<model.Tab> & { pinned?: boolean }
export type AppSettings = Plain<model.AppSettings>
export type MockServer = Plain<model.MockServer>
export type MockRoute = Plain<model.MockRoute>
export type MockRequestLog = Plain<model.MockRequestLog>
export type ImportResult = Plain<model.ImportSummary>
export type SearchResult = Plain<model.SearchResult>
export type WebSocketConnection = Plain<model.WSConnection>
export type WSConnection = Plain<model.WSConnection>
export type WebSocketMessage = Plain<model.WSMessage>
export type RunOptions = Plain<model.RunOptions>
export type RunResult = Plain<model.RunResult>
export type IterationResult = Plain<model.IterationResult>
export type RequestResult = Plain<model.RequestResult>
export type AppState = Plain<model.AppState>
export type ProxyConfig = Plain<model.Proxy>
export type TLSConfig = Plain<model.TLS>

// --- advanced tooling (mirrors internal/model additions) -------------------

export type StoredCookie = Plain<model.StoredCookie>
export type ExtractionResult = Plain<model.ExtractionResult>
export type JSONDiff = Plain<model.JSONDiff>
export type JSONDiffEntry = Plain<model.JSONDiffEntry>
export type AIRequest = Plain<model.AIRequest>
export type OAuth2StartResult = Plain<model.OAuth2StartResult>
export type CodeTarget = "curl" | "http" | "fetch" | "node" | "python" | "go" | "php"

// --- aliases used throughout the UI ---------------------------------------

export type KeyValuePair = Plain<model.KeyValue>
export type Header = Plain<model.KeyValue>
export type QueryParam = Plain<model.KeyValue>
export type CollectionVariable = Plain<model.Variable>
export type EnvironmentVariable = Plain<model.Variable>
export type RequestAuth = Plain<model.Auth>

export type HTTPMethod = string
export type BodyType = string
export type AuthType = string
export type VariableScope = "global" | "collection" | "environment" | "local"
export type VariableType = "default" | "secret"

export type SidebarTab =
  | "collections"
  | "environments"
  | "history"
  | "mocks"
  | "websocket"
  | "cookies"
  | "settings"

export type ResponseViewMode = "pretty" | "raw" | "preview"

/** User preferences that control tab-strip behaviour. Persisted locally. */
export interface TabPreferences {
  confirmCloseDirtyTabs: boolean
  closeTabOnMiddleClick: boolean
  tabCloseButton: "hover" | "always"
}
