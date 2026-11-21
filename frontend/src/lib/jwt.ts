// Minimal JWT decoder used to display token payloads and expiry in the auth
// editor. It never verifies signatures — decoding is purely informational.

export interface JWTInfo {
  header: Record<string, unknown>
  payload: Record<string, unknown>
  signature: string
  expiresAt?: number
  issuedAt?: number
  subject?: string
  expired: boolean
}

function base64UrlDecode(input: string): string {
  const padded = input.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(input.length / 4) * 4, "=")
  const binary = atob(padded)
  const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
  return new TextDecoder().decode(bytes)
}

export function isJWT(token: string): boolean {
  return token.split(".").length === 3 && token.length > 20
}

export function decodeJWT(token: string): JWTInfo | null {
  const trimmed = token.trim().replace(/^Bearer\s+/i, "")
  const parts = trimmed.split(".")
  if (parts.length !== 3) return null
  try {
    const header = JSON.parse(base64UrlDecode(parts[0])) as Record<string, unknown>
    const payload = JSON.parse(base64UrlDecode(parts[1])) as Record<string, unknown>
    const exp = typeof payload.exp === "number" ? payload.exp : undefined
    const iat = typeof payload.iat === "number" ? payload.iat : undefined
    return {
      header,
      payload,
      signature: parts[2],
      expiresAt: exp,
      issuedAt: iat,
      subject: typeof payload.sub === "string" ? payload.sub : undefined,
      expired: exp !== undefined ? exp * 1000 <= Date.now() : false,
    }
  } catch {
    return null
  }
}

export function formatJWTExpiry(info: JWTInfo): string {
  if (info.expiresAt === undefined) return "no expiry claim"
  const date = new Date(info.expiresAt * 1000)
  const relative = describeRelative(info.expiresAt * 1000 - Date.now())
  return `${date.toLocaleString()} (${relative})`
}

function describeRelative(deltaMs: number): string {
  const abs = Math.abs(deltaMs)
  const units: [number, string][] = [
    [1000, "second"],
    [60_000, "minute"],
    [3_600_000, "hour"],
    [86_400_000, "day"],
  ]
  let label = `${Math.round(abs / 1000)} seconds`
  for (const [size, name] of units) {
    if (abs >= size) label = `${Math.round(abs / size)} ${name}${Math.round(abs / size) === 1 ? "" : "s"}`
  }
  return deltaMs >= 0 ? `in ${label}` : `${label} ago`
}
