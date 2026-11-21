// Variable discovery, resolution and masking.
//
// Variable precedence (highest first): local → environment → collection →
// global. This mirrors the backend resolver in internal/variables so previews
// in the UI match what actually gets sent.

import { useMemo } from "react"
import { useWorkspaceStore } from "@/stores/workspace"
import type { VariableScope } from "@/types"

export interface ScopedVariable {
  key: string
  value: string
  type: string
  description?: string
  scope: VariableScope
  secret: boolean
}

const SCOPE_ORDER: VariableScope[] = ["local", "environment", "collection", "global"]

const SECRET_KEY_HINTS = ["token", "secret", "password", "apikey", "api_key", "auth", "credential", "private"]

/** A value that should never be shown in plain text in the UI. */
export function isSecretValue(key: string, declaredType?: string): boolean {
  if (declaredType === "secret") return true
  const lower = key.toLowerCase()
  return SECRET_KEY_HINTS.some((hint) => lower.includes(hint))
}

export function maskValue(value: string): string {
  if (!value) return ""
  if (value.length <= 4) return "••••"
  return `${value.slice(0, 2)}••••${value.slice(-2)}`
}

/** Merge scopes into a flat list where higher-precedence entries win. */
export function mergeScopes(
  scopes: Partial<Record<VariableScope, { key: string; value: string; type?: string; description?: string }[]>>,
): ScopedVariable[] {
  const byKey = new Map<string, ScopedVariable>()
  for (const scope of [...SCOPE_ORDER].reverse()) {
    for (const variable of scopes[scope] ?? []) {
      if (!variable.key) continue
      byKey.delete(variable.key)
      byKey.set(variable.key, {
        key: variable.key,
        value: variable.value ?? "",
        type: variable.type || "default",
        description: variable.description,
        scope,
        secret: isSecretValue(variable.key, variable.type),
      })
    }
  }
  return [...byKey.values()].sort((a, b) => a.key.localeCompare(b.key))
}

const PLACEHOLDER = /{{\s*([^}\s]+)\s*}}/g

/** All {{placeholder}} names used in a string (without duplicates). */
export function findPlaceholders(text: string): string[] {
  if (!text) return []
  const found = new Set<string>()
  for (const match of text.matchAll(PLACEHOLDER)) {
    found.add(match[1])
  }
  return [...found]
}

/** Placeholders that have no value in the given scopes (dynamic ones excluded). */
export function unresolvedPlaceholders(text: string, variables: ScopedVariable[]): string[] {
  const known = new Set(variables.map((v) => v.key))
  return findPlaceholders(text).filter((name) => !name.startsWith("$") && !known.has(name))
}

/** Expand {{name}} using the merged scopes; unknown placeholders are kept. */
export function resolveTemplate(text: string, variables: ScopedVariable[]): string {
  if (!text || !text.includes("{{")) return text
  const map = new Map(variables.map((v) => [v.key, v.value]))
  return text.replace(PLACEHOLDER, (whole, name: string) => (map.has(name) ? map.get(name) ?? "" : whole))
}

/** Hook that returns the merged variables from the current workspace state. */
export function useVariableScopes(): ScopedVariable[] {
  const environments = useWorkspaceStore((s) => s.environments)
  const activeEnvironmentId = useWorkspaceStore((s) => s.activeEnvironmentId)
  const collections = useWorkspaceStore((s) => s.collections)
  const globals = useWorkspaceStore((s) => s.globalVariables)

  return useMemo(() => {
    const environment = environments.find((e) => e.id === activeEnvironmentId)
    const collectionVariables = collections.flatMap((c) =>
      (c.variables ?? []).map((v) => ({ key: v.key, value: v.value, type: v.type, description: v.description })),
    )
    return mergeScopes({
      environment: (environment?.values ?? []).map((v) => ({
        key: v.key,
        value: v.value,
        type: v.type,
        description: v.description,
      })),
      collection: collectionVariables,
      global: globals.map((g) => ({ key: g.key, value: g.value, type: g.type, description: g.description })),
    })
  }, [environments, activeEnvironmentId, collections, globals])
}
