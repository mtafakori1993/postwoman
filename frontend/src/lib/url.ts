// URL ⇄ query-parameter synchronisation helpers.
//
// Post helps users keep the URL bar and the Params table in step. The URL is
// the source of truth: editing the URL re-parses the query string, and editing
// the table rewrites it. Descriptions and enabled flags are preserved for
// parameters that survive the round trip.

import type { KeyValuePair } from "@/types"

export interface SplitURL {
  base: string
  query: string
  hash: string
}

function safeDecode(value: string): string {
  try {
    return decodeURIComponent(value.replace(/\+/g, " "))
  } catch {
    return value
  }
}

export function splitUrl(url: string): SplitURL {
  const hashIndex = url.indexOf("#")
  const hash = hashIndex === -1 ? "" : url.slice(hashIndex)
  const withoutHash = hashIndex === -1 ? url : url.slice(0, hashIndex)
  const queryIndex = withoutHash.indexOf("?")
  if (queryIndex === -1) return { base: withoutHash, query: "", hash }
  return { base: withoutHash.slice(0, queryIndex), query: withoutHash.slice(queryIndex + 1), hash }
}

/** Parse a raw query string (without the leading `?`) into rows. */
export function parseQueryString(query: string): KeyValuePair[] {
  const cleaned = query.replace(/^\?/, "")
  if (!cleaned.trim()) return []
  const rows: KeyValuePair[] = []
  for (const part of cleaned.split("&")) {
    if (!part) continue
    const index = part.indexOf("=")
    const rawKey = index === -1 ? part : part.slice(0, index)
    const rawValue = index === -1 ? "" : part.slice(index + 1)
    rows.push({
      key: safeDecode(rawKey),
      value: safeDecode(rawValue),
      description: "",
      enabled: true,
      type: "text",
    })
  }
  return rows
}

/** Encode rows back into a query string (without the leading `?`). */
export function encodeQueryString(params: KeyValuePair[]): string {
  return params
    .filter((p) => p.enabled !== false && (p.key ?? "") !== "")
    .map((p) => `${encodeURIComponent(p.key)}=${encodeURIComponent(p.value ?? "")}`)
    .join("&")
}

/** Rebuild the full URL from its base plus the editable rows. */
export function composeUrl(url: string, params: KeyValuePair[]): string {
  const { base, hash } = splitUrl(url)
  const query = encodeQueryString(params)
  return `${base}${query ? `?${query}` : ""}${hash}`
}

/**
 * Re-parse the query string of `url` and merge it with the previous rows so
 * descriptions/enabled flags survive. Rows that are no longer present in the
 * URL are dropped, because the URL is authoritative.
 */
export function paramsFromUrl(url: string, previous: KeyValuePair[] = []): KeyValuePair[] {
  const parsed = parseQueryString(splitUrl(url).query)
  return parsed.map((row) => {
    const match = previous.find((p) => p.key === row.key && (p.value ?? "") === (row.value ?? ""))
    if (match) return { ...row, description: match.description ?? "", enabled: match.enabled !== false }
    return row
  })
}

/** True when the URL's query string differs from the rows. */
export function paramsDifferFromUrl(url: string, params: KeyValuePair[]): boolean {
  return encodeQueryString(paramsFromUrl(url)) !== encodeQueryString(params)
}

/**
 * Detect Postman-style path variables (`/:id`) and `{{placeholders}}` that sit
 * in the path portion of a URL (the query string is excluded).
 */
export function detectPathVariables(url: string): string[] {
  const { base } = splitUrl(url)
  const withoutScheme = base.replace(/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//, "")
  const slash = withoutScheme.indexOf("/")
  const path = slash === -1 ? "" : withoutScheme.slice(slash)
  const found = new Set<string>()
  for (const match of path.matchAll(/:([A-Za-z_][A-Za-z0-9_-]*)/g)) {
    found.add(match[1])
  }
  for (const match of path.matchAll(/{{([^}]+)}}/g)) {
    found.add(match[1].trim())
  }
  return [...found]
}

/** Ensure every detected path variable exists as a row (keeping existing values). */
export function syncPathVariables(url: string, previous: KeyValuePair[] = []): KeyValuePair[] {
  const detected = detectPathVariables(url)
  const rows: KeyValuePair[] = detected.map((key) => {
    const match = previous.find((p) => p.key === key)
    return match
      ? { ...match, key, enabled: match.enabled !== false }
      : { key, value: "", description: "", enabled: true, type: "text" }
  })
  return rows
}

/** Path variables detected in the URL that have no value configured. */
export function missingPathVariableNames(url: string, params: KeyValuePair[]): string[] {
  return detectPathVariables(url).filter((name) => {
    const row = params.find((p) => p.key === name && p.enabled !== false)
    return !row || !(row.value ?? "").trim()
  })
}
