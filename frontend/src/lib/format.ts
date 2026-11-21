// Formatting, validation and content-type helpers shared by the request body
// editor and the response viewer.
//
// Everything here is intentionally dependency-free: the desktop app has to
// work offline, so we only rely on the platform JSON parser and DOMParser.

import type { BodyType } from "@/types"

export type CodeLanguage = "json" | "xml" | "html" | "javascript" | "text"

export interface FormatResult {
  ok: boolean
  output: string
  error?: string
}

const CONTENT_TYPES: Record<string, string> = {
  raw: "text/plain",
  json: "application/json",
  xml: "application/xml",
  text: "text/plain",
  javascript: "application/javascript",
  html: "text/html",
  urlencoded: "application/x-www-form-urlencoded",
  binary: "application/octet-stream",
  graphql: "application/json",
}

/** The Content-Type that belongs to a body mode, or null when the backend
 * manages it itself (multipart needs a generated boundary). */
export function defaultContentTypeForBody(mode: BodyType | string): string | null {
  return CONTENT_TYPES[(mode || "").toLowerCase()] ?? null
}

/** CodeMirror language for a body mode. */
export function languageForBody(mode: BodyType | string): CodeLanguage {
  switch ((mode || "").toLowerCase()) {
    case "json":
    case "graphql":
      return "json"
    case "xml":
      return "xml"
    case "html":
      return "html"
    case "javascript":
      return "javascript"
    default:
      return "text"
  }
}

/** Best-effort language detection for a response body. */
export function detectLanguage(contentType = "", body = ""): CodeLanguage {
  const ct = contentType.toLowerCase()
  if (ct.includes("json") || ct.includes("+json")) return "json"
  if (ct.includes("html") || ct.includes("xhtml")) return "html"
  if (ct.includes("xml")) return "xml"
  if (ct.includes("javascript") || ct.includes("ecmascript")) return "javascript"

  // Content types such as text/plain or application/octet-stream often carry
  // structured payloads anyway. Sniff before falling back to plain text.
  const head = body.trimStart().slice(0, 512)
  if (!head) return "text"
  if (head.startsWith("{") || head.startsWith("[")) return "json"
  if (/^<!doctype\s+html/i.test(head) || /^<html[\s>]/i.test(head)) return "html"
  if (head.startsWith("<")) return "xml"
  return "text"
}

function indentation(level: number): string {
  return "  ".repeat(Math.max(0, level))
}

/**
 * Tolerant XML pretty-printer. It keeps text nodes on their own line and does
 * not try to be a full XML parser; the source is validated separately before
 * reaching this function.
 */
export function prettyXML(xml: string): string {
  const tokens = xml
    .replace(/>\s*</g, "><")
    .match(/<!--[\s\S]*?-->|<!\[CDATA\[[\s\S]*?\]\]>|<(?:[^>"']|"[^"]*"|'[^']*')+>|[^<]+/g)
  if (!tokens) return xml

  let depth = 0
  const lines: string[] = []

  for (const raw of tokens) {
    const token = raw.trim()
    if (!token) continue

    if (token.startsWith("<!--") || token.startsWith("<![CDATA[") || token.startsWith("<?") || token.startsWith("<!")) {
      lines.push(indentation(depth) + token)
      continue
    }
    if (token.startsWith("</")) {
      depth = Math.max(0, depth - 1)
      lines.push(indentation(depth) + token)
      continue
    }
    if (token.startsWith("<")) {
      const selfClosing = /\/>$/.test(token)
      lines.push(indentation(depth) + token)
      if (!selfClosing) depth += 1
      continue
    }
    lines.push(indentation(depth) + token)
  }

  return lines.join("\n")
}

const HTML_VOID_TAGS = new Set([
  "area", "base", "br", "col", "embed", "hr", "img", "input", "link",
  "meta", "param", "source", "track", "wbr",
])

/** Tolerant HTML pretty-printer with basic void/inline handling. */
export function prettyHTML(html: string): string {
  const tokens = html
    .replace(/>\s*</g, "><")
    .match(
      /<!--[\s\S]*?-->|<script\b[^>]*>[\s\S]*?<\/script>|<style\b[^>]*>[\s\S]*?<\/style>|<(?:[^>"']|"[^"]*"|'[^']*')+>|[^<]+/gi,
    )
  if (!tokens) return html

  let depth = 0
  const lines: string[] = []

  for (const raw of tokens) {
    const token = raw.trim()
    if (!token) continue

    if (token.startsWith("<!--") || token.startsWith("<!")) {
      lines.push(indentation(depth) + token)
      continue
    }
    if (token.startsWith("</")) {
      depth = Math.max(0, depth - 1)
      lines.push(indentation(depth) + token)
      continue
    }
    if (token.startsWith("<")) {
      const tag = token.match(/^<\s*([a-z0-9-]+)/i)?.[1]?.toLowerCase() ?? ""
      const selfClosing = /\/>$/.test(token) || HTML_VOID_TAGS.has(tag)
      const wholeElement = new RegExp(`^<${tag}\\b[^>]*>[\\s\\S]*</${tag}>$`, "i").test(token)
      lines.push(indentation(depth) + token)
      if (!selfClosing && !wholeElement) depth += 1
      continue
    }
    lines.push(indentation(depth) + token)
  }

  return lines.join("\n")
}

export function formatJSON(json: string): FormatResult {
  try {
    const parsed = JSON.parse(json)
    return { ok: true, output: JSON.stringify(parsed, null, 2) }
  } catch (error) {
    return { ok: false, output: json, error: jsonErrorMessage(error, json) }
  }
}

/** Validate XML with the platform parser when available. */
export function validateXML(xml: string): string | null {
  if (typeof DOMParser === "undefined") return null
  try {
    const doc = new DOMParser().parseFromString(xml, "application/xml")
    const parserError = doc.querySelector("parsererror")
    if (!parserError) return null
    const text = parserError.textContent?.replace(/\s+/g, " ").trim() ?? "Invalid XML"
    return text.replace(/^This page contains the following errors:\s*/i, "").slice(0, 400)
  } catch (error) {
    return error instanceof Error ? error.message : "Invalid XML"
  }
}

/** Validate content for the given language. Returns a human-readable error or null. */
export function validateContent(content: string, language: CodeLanguage): string | null {
  if (!content.trim()) return null
  if (language === "json") {
    try {
      JSON.parse(content)
      return null
    } catch (error) {
      return jsonErrorMessage(error, content)
    }
  }
  if (language === "xml") return validateXML(content)
  return null
}

export function formatContent(content: string, language: CodeLanguage): FormatResult {
  if (!content.trim()) return { ok: true, output: content }

  if (language === "json") return formatJSON(content)
  if (language === "xml") {
    const error = validateXML(content)
    if (error) return { ok: false, output: content, error }
    return { ok: true, output: prettyXML(content) }
  }
  if (language === "html") return { ok: true, output: prettyHTML(content) }
  return { ok: true, output: content }
}

/** Compact (minified) representation for whitespace-insensitive formats. Only
 * JSON is collapsed; other languages are returned unchanged. */
export function minifyContent(content: string, language: CodeLanguage): FormatResult {
  if (!content.trim()) return { ok: true, output: content }

  if (language === "json") {
    try {
      return { ok: true, output: JSON.stringify(JSON.parse(content)) }
    } catch (error) {
      return { ok: false, output: content, error: jsonErrorMessage(error, content) }
    }
  }
  return { ok: true, output: content }
}

/** Convert a JSON.parse SyntaxError into a message with line/column details. */
function jsonErrorMessage(error: unknown, source: string): string {
  const message = error instanceof Error ? error.message : String(error)
  const position = /position\s+(\d+)/i.exec(message)?.[1]
  if (!position) return message
  const index = Number(position)
  if (!Number.isFinite(index)) return message
  const before = source.slice(0, index)
  const line = before.split(/\r?\n/).length
  const column = index - before.lastIndexOf("\n")
  return `${message} (line ${line}, column ${column})`
}

export function extForContentType(contentType = ""): string {
  const ct = contentType.toLowerCase()
  if (ct.includes("json")) return "json"
  if (ct.includes("html")) return "html"
  if (ct.startsWith("image/png")) return "png"
  if (ct.startsWith("image/jpeg")) return "jpg"
  if (ct.startsWith("image/gif")) return "gif"
  if (ct.startsWith("image/webp")) return "webp"
  if (ct.includes("image/svg")) return "svg"
  if (ct.includes("xml")) return "xml"
  if (ct.includes("javascript")) return "js"
  if (ct.includes("csv")) return "csv"
  if (ct.includes("pdf")) return "pdf"
  if (ct.includes("octet-stream")) return "bin"
  return "txt"
}
