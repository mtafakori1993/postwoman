import React, { useMemo, useState } from "react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { AlertTriangle, CheckCircle2, ExternalLink, Fingerprint, Globe, Key, Layers, Loader2, Lock, RefreshCw, Shield, Trash2, User } from "lucide-react"
import type { AuthConfig, AuthType, Request } from "@/types"
import { cn } from "@/lib/utils"
import * as api from "../lib/api"
import { decodeJWT, formatJWTExpiry, isJWT } from "@/lib/jwt"

const AUTH_TYPES: { value: AuthType; label: string; icon: React.ReactNode; description: string }[] = [
  { value: "inherit", label: "Inherit from parent", icon: <Layers className="h-4 w-4" />, description: "Use the folder or collection auth" },
  { value: "noauth", label: "No Auth", icon: <Shield className="h-4 w-4" />, description: "Do not send credentials" },
  { value: "apikey", label: "API Key", icon: <Key className="h-4 w-4" />, description: "Key/value in a header or query" },
  { value: "bearer", label: "Bearer Token", icon: <Fingerprint className="h-4 w-4" />, description: "Authorization: Bearer …" },
  { value: "basic", label: "Basic Auth", icon: <User className="h-4 w-4" />, description: "Username and password" },
  { value: "digest", label: "Digest Auth", icon: <Lock className="h-4 w-4" />, description: "Challenge/response digest" },
  { value: "oauth1", label: "OAuth 1.0", icon: <Globe className="h-4 w-4" />, description: "Signed OAuth 1.0 requests" },
  { value: "oauth2", label: "OAuth 2.0", icon: <Globe className="h-4 w-4" />, description: "Authorization code, PKCE, client credentials" },
  { value: "awsv4", label: "AWS Signature", icon: <Lock className="h-4 w-4" />, description: "AWS Signature v4" },
  { value: "hawk", label: "Hawk", icon: <Key className="h-4 w-4" />, description: "Hawk authentication" },
  { value: "ntlm", label: "NTLM", icon: <User className="h-4 w-4" />, description: "NTLM authentication" },
  { value: "jwt", label: "JWT Bearer", icon: <Fingerprint className="h-4 w-4" />, description: "JWT as a bearer token" },
]

interface AuthEditorProps {
  auth: AuthConfig
  onChange: (auth: AuthConfig) => void
  inheritedAuth?: AuthConfig
  inheritedSource?: string
  request?: Request | null
  environmentId?: string | null
}

function withDefaults<T extends object>(defaults: T, current?: Partial<T>): T {
  return { ...defaults, ...(current ?? {}) }
}

function mergeAuth<T extends object>(base: T | undefined, changes: Partial<T>): T {
  return { ...(base as T), ...changes }
}

function getAuthConfig(auth: AuthConfig, type: AuthType): AuthConfig {
  const base = { ...auth, type }
  switch (type) {
    case "apikey":
      return { ...base, apikey: withDefaults({ key: "", value: "", addTo: "header" }, auth.apikey) }
    case "bearer":
      return { ...base, bearer: withDefaults({ token: "" }, auth.bearer) }
    case "basic":
      return { ...base, basic: withDefaults({ username: "", password: "" }, auth.basic) }
    case "digest":
      return { ...base, digest: withDefaults({ username: "", password: "" }, auth.digest) }
    case "oauth1":
      return {
        ...base,
        oauth1: withDefaults(
          { consumerKey: "", consumerSecret: "", token: "", tokenSecret: "", signatureMethod: "HMAC-SHA1", version: "1.0" },
          auth.oauth1,
        ),
      }
    case "oauth2":
      return {
        ...base,
        oauth2: withDefaults(
          {
            grantType: "authorization_code_pkce",
            accessToken: "",
            refreshToken: "",
            clientId: "",
            clientSecret: "",
            authUrl: "",
            tokenUrl: "",
            scope: "",
            redirectUrl: "http://localhost",
            pkce: true,
          },
          auth.oauth2,
        ),
      }
    case "awsv4":
      return { ...base, awsv4: withDefaults({ accessKey: "", secretKey: "", region: "", service: "" }, auth.awsv4) }
    case "hawk":
      return { ...base, hawk: withDefaults({ id: "", key: "", algorithm: "sha256" }, auth.hawk) }
    case "ntlm":
      return { ...base, ntlm: withDefaults({ username: "", password: "" }, auth.ntlm) }
    case "jwt":
      return { ...base, jwt: withDefaults({ token: "" }, auth.jwt) }
    default:
      return { ...base, type: "noauth" }
  }
}

function JWTPreview({ token }: { token: string }) {
  const info = useMemo(() => (isJWT(token) ? decodeJWT(token) : null), [token])
  if (!info) return null
  return (
    <div className="space-y-1 rounded-md border border-border bg-muted/20 p-2 text-xs">
      <div className="flex items-center gap-2">
        <span className="font-semibold">JWT decoded</span>
        <span className={cn("rounded px-1.5 py-0.5 text-[10px] font-medium", info.expired ? "bg-destructive/15 text-destructive" : "bg-green-500/15 text-green-700 dark:text-green-400")}>
          {info.expired ? "expired" : "active"}
        </span>
        <span className="ml-auto text-muted-foreground" title={formatJWTExpiry(info)}>
          exp: {formatJWTExpiry(info)}
        </span>
      </div>
      {info.subject && <div className="text-muted-foreground">sub: {info.subject}</div>}
      <pre className="max-h-32 overflow-auto rounded bg-background p-2 font-mono text-[11px]">{JSON.stringify(info.payload, null, 2)}</pre>
    </div>
  )
}

function OAuth2Panel({
  oauth,
  onChange,
  request,
  environmentId,
}: {
  oauth: NonNullable<AuthConfig["oauth2"]>
  onChange: (oauth: NonNullable<AuthConfig["oauth2"]>) => void
  request?: Request | null
  environmentId?: string | null
}) {
  const [code, setCode] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const requestForCall = (): Request | null => (request ? { ...request, auth: { ...request.auth, type: "oauth2", oauth2: oauth } } : null)

  const run = async (fn: (req: Request) => Promise<Partial<NonNullable<AuthConfig["oauth2"]>>>) => {
    const withAuth = requestForCall()
    if (!withAuth) {
      setError("Save or open the request before running the OAuth flow.")
      return
    }
    setBusy(true)
    setError(null)
    setMessage(null)
    try {
      const updated = await fn(withAuth)
      onChange({ ...oauth, ...updated })
      setMessage("Token updated")
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const authorize = async () => {
    const withAuth = requestForCall()
    if (!withAuth) {
      setError("Save or open the request before running the OAuth flow.")
      return
    }
    setBusy(true)
    setError(null)
    setMessage(null)
    try {
      const result = await api.oauth2Start(withAuth, environmentId ?? "")
      if (result.auth) onChange({ ...oauth, ...result.auth })
      await api.openExternalURL(result.url)
      setMessage("Browser opened with the authorization URL. Paste the returned code or redirect URL below.")
    } catch (err) {
      setError(api.errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const exchange = () =>
    run((req) => api.oauth2Exchange(req, environmentId ?? "", code).then((updated) => updated as Partial<NonNullable<AuthConfig["oauth2"]>>))

  const fetchToken = () => run((req) => api.oauth2FetchToken(req, environmentId ?? "").then((updated) => updated as Partial<NonNullable<AuthConfig["oauth2"]>>))

  const expiry = oauth.expiresAt ? new Date(oauth.expiresAt) : null
  const expired = oauth.expiresAt ? oauth.expiresAt <= Date.now() : false
  const interactive = oauth.grantType === "authorization_code" || oauth.grantType === "authorization_code_pkce"

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-2">
        <Input placeholder="Client ID" value={oauth.clientId} onChange={(e) => onChange({ ...oauth, clientId: e.target.value })} />
        <Input type="password" placeholder="Client Secret" value={oauth.clientSecret} onChange={(e) => onChange({ ...oauth, clientSecret: e.target.value })} />
      </div>
      <div className="grid grid-cols-2 gap-2">
        <Input placeholder="Authorization URL" value={oauth.authUrl} onChange={(e) => onChange({ ...oauth, authUrl: e.target.value })} />
        <Input placeholder="Token URL" value={oauth.tokenUrl} onChange={(e) => onChange({ ...oauth, tokenUrl: e.target.value })} />
      </div>
      <div className="grid grid-cols-3 gap-2">
        <Select
          value={oauth.grantType}
          onValueChange={(value) => onChange({ ...oauth, grantType: value, pkce: value === "authorization_code_pkce" })}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="authorization_code_pkce">Authorization Code (PKCE)</SelectItem>
            <SelectItem value="authorization_code">Authorization Code</SelectItem>
            <SelectItem value="client_credentials">Client Credentials</SelectItem>
            <SelectItem value="password">Password</SelectItem>
            <SelectItem value="refresh_token">Refresh Token</SelectItem>
          </SelectContent>
        </Select>
        <Input placeholder="Scope" value={oauth.scope} onChange={(e) => onChange({ ...oauth, scope: e.target.value })} />
        <Input placeholder="Redirect URL" value={oauth.redirectUrl} onChange={(e) => onChange({ ...oauth, redirectUrl: e.target.value })} />
      </div>

      {oauth.grantType === "password" && (
        <div className="grid grid-cols-2 gap-2">
          <Input placeholder="Username" value={oauth.username || ""} onChange={(e) => onChange({ ...oauth, username: e.target.value })} />
          <Input type="password" placeholder="Password" value={oauth.password || ""} onChange={(e) => onChange({ ...oauth, password: e.target.value })} />
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2">
        {interactive && (
          <Button variant="outline" size="sm" className="gap-1.5" onClick={authorize} disabled={busy}>
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <ExternalLink className="h-3.5 w-3.5" />}
            Authorize
          </Button>
        )}
        {!interactive && (
          <Button variant="outline" size="sm" className="gap-1.5" onClick={fetchToken} disabled={busy}>
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
            Get token
          </Button>
        )}
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5"
          onClick={() => onChange({ ...oauth, accessToken: "", refreshToken: "", expiresAt: 0, tokenType: "" })}
          disabled={!oauth.accessToken && !oauth.refreshToken}
        >
          <Trash2 className="h-3.5 w-3.5" />
          Clear tokens
        </Button>
        <span className={cn("ml-auto flex items-center gap-1 text-[11px]", expired ? "text-destructive" : "text-muted-foreground")}>
          {oauth.accessToken ? (
            <>
              <CheckCircle2 className="h-3 w-3" />
              {expiry ? `expires ${expiry.toLocaleString()}` : "token stored"}
            </>
          ) : (
            "no token yet"
          )}
        </span>
      </div>

      {interactive && (
        <div className="flex items-center gap-2">
          <Input
            placeholder="Paste the authorization code or full redirect URL"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            className="font-mono text-xs"
          />
          <Button size="sm" variant="outline" onClick={exchange} disabled={busy || !code.trim()}>
            Exchange
          </Button>
        </div>
      )}

      {(error || message) && (
        <div className={cn("rounded-md border px-2.5 py-1.5 text-[11px]", error ? "border-destructive/30 bg-destructive/5 text-destructive" : "border-border bg-muted/30 text-muted-foreground")}>
          {error || message}
        </div>
      )}
    </div>
  )
}

function TokenField({ label, value, onChange, placeholder }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string }) {
  const [reveal, setReveal] = useState(false)
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <Label className="text-xs">{label}</Label>
        <button type="button" className="text-[11px] text-muted-foreground underline" onClick={() => setReveal((v) => !v)}>
          {reveal ? "hide" : "reveal"}
        </button>
      </div>
      <Input type={reveal ? "text" : "password"} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className="font-mono" />
    </div>
  )
}

export function AuthEditor({ auth, onChange, inheritedAuth, inheritedSource, request, environmentId }: AuthEditorProps) {
  const activeType = (auth.type || "inherit") as AuthType

  const handleTypeChange = (type: AuthType) => onChange(getAuthConfig(auth, type))

  const renderFields = () => {
    switch (activeType) {
      case "inherit":
        return (
          <div className="space-y-2 text-sm">
            <div className="flex items-center gap-2">
              <Layers className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium">Inherited authentication</span>
            </div>
            {inheritedAuth && inheritedAuth.type && inheritedAuth.type !== "noauth" && inheritedAuth.type !== "inherit" ? (
              <p className="text-xs text-muted-foreground">
                This request uses <span className="font-semibold text-foreground">{inheritedAuth.type}</span>
                {inheritedSource ? ` configured on ${inheritedSource}` : " from its collection or folder"}.
              </p>
            ) : (
              <p className="text-xs text-muted-foreground">
                No authentication is configured on the collection or folders, so this request is sent without credentials.
              </p>
            )}
          </div>
        )
      case "noauth":
        return (
          <div className="py-6 text-center text-muted-foreground">
            <Shield className="mx-auto mb-2 h-6 w-6 opacity-50" />
            <p className="text-sm">This request overrides any inherited auth and sends no credentials.</p>
          </div>
        )
      case "apikey":
        return (
          <div className="grid grid-cols-3 gap-2">
            <Input placeholder="Key" value={auth.apikey?.key || ""} onChange={(e) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { key: e.target.value }) })} />
            <Input placeholder="Value" value={auth.apikey?.value || ""} onChange={(e) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { value: e.target.value }) })} />
            <Select value={auth.apikey?.addTo || "header"} onValueChange={(value) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { addTo: value }) })}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="header">Header</SelectItem>
                <SelectItem value="query">Query Params</SelectItem>
              </SelectContent>
            </Select>
          </div>
        )
      case "bearer":
        return (
          <div className="space-y-2">
            <TokenField label="Token" value={auth.bearer?.token || ""} onChange={(token) => onChange({ ...auth, bearer: { token } })} placeholder="Enter bearer token" />
            <JWTPreview token={auth.bearer?.token || ""} />
          </div>
        )
      case "basic":
        return (
          <div className="grid grid-cols-2 gap-2">
            <Input placeholder="Username" value={auth.basic?.username || ""} onChange={(e) => onChange({ ...auth, basic: mergeAuth(auth.basic, { username: e.target.value }) })} />
            <Input type="password" placeholder="Password" value={auth.basic?.password || ""} onChange={(e) => onChange({ ...auth, basic: mergeAuth(auth.basic, { password: e.target.value }) })} />
          </div>
        )
      case "digest":
        return (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Username" value={auth.digest?.username || ""} onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { username: e.target.value }) })} />
              <Input type="password" placeholder="Password" value={auth.digest?.password || ""} onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { password: e.target.value }) })} />
            </div>
            <p className="text-[11px] text-muted-foreground">Realm and nonce are read from the server challenge automatically.</p>
          </div>
        )
      case "oauth1":
        return (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Consumer Key" value={auth.oauth1?.consumerKey || ""} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { consumerKey: e.target.value }) })} />
              <Input type="password" placeholder="Consumer Secret" value={auth.oauth1?.consumerSecret || ""} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { consumerSecret: e.target.value }) })} />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Token" value={auth.oauth1?.token || ""} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { token: e.target.value }) })} />
              <Input type="password" placeholder="Token Secret" value={auth.oauth1?.tokenSecret || ""} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { tokenSecret: e.target.value }) })} />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <Select value={auth.oauth1?.signatureMethod || "HMAC-SHA1"} onValueChange={(value) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { signatureMethod: value }) })}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="HMAC-SHA1">HMAC-SHA1</SelectItem>
                  <SelectItem value="HMAC-SHA256">HMAC-SHA256</SelectItem>
                  <SelectItem value="PLAINTEXT">PLAINTEXT</SelectItem>
                </SelectContent>
              </Select>
              <Input placeholder="Version" value={auth.oauth1?.version || "1.0"} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { version: e.target.value }) })} />
              <Input placeholder="Realm (optional)" value={auth.oauth1?.realm || ""} onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { realm: e.target.value }) })} />
            </div>
          </div>
        )
      case "oauth2":
        return (
          <OAuth2Panel
            oauth={getAuthConfig(auth, "oauth2").oauth2!}
            onChange={(oauth2) => onChange({ ...auth, type: "oauth2", oauth2 })}
            request={request}
            environmentId={environmentId}
          />
        )
      case "awsv4":
        return (
          <div className="space-y-2">
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Access Key" value={auth.awsv4?.accessKey || ""} onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { accessKey: e.target.value }) })} />
              <Input type="password" placeholder="Secret Key" value={auth.awsv4?.secretKey || ""} onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { secretKey: e.target.value }) })} />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <Input placeholder="Region" value={auth.awsv4?.region || ""} onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { region: e.target.value }) })} />
              <Input placeholder="Service" value={auth.awsv4?.service || ""} onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { service: e.target.value }) })} />
              <Input placeholder="Session token (optional)" value={auth.awsv4?.sessionToken || ""} onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { sessionToken: e.target.value }) })} />
            </div>
          </div>
        )
      case "hawk":
        return (
          <div className="space-y-2">
            <NotSupportedNotice kind="Hawk" />
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="ID" value={auth.hawk?.id || ""} onChange={(e) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { id: e.target.value }) })} />
              <Input type="password" placeholder="Key" value={auth.hawk?.key || ""} onChange={(e) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { key: e.target.value }) })} />
            </div>
          </div>
        )
      case "ntlm":
        return (
          <div className="space-y-2">
            <NotSupportedNotice kind="NTLM" />
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Username" value={auth.ntlm?.username || ""} onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { username: e.target.value }) })} />
              <Input type="password" placeholder="Password" value={auth.ntlm?.password || ""} onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { password: e.target.value }) })} />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input placeholder="Domain (optional)" value={auth.ntlm?.domain || ""} onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { domain: e.target.value }) })} />
              <Input placeholder="Workstation (optional)" value={auth.ntlm?.workstation || ""} onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { workstation: e.target.value }) })} />
            </div>
          </div>
        )
      case "jwt":
        return (
          <div className="space-y-2">
            <TokenField label="JWT" value={auth.jwt?.token || ""} onChange={(token) => onChange({ ...auth, jwt: { ...(auth.jwt || { token: "" }), token } })} placeholder="Paste a JWT" />
            <Input placeholder="Header prefix (default: Bearer)" value={auth.jwt?.headerPrefix || ""} onChange={(e) => onChange({ ...auth, jwt: { ...(auth.jwt || { token: "" }), headerPrefix: e.target.value } })} />
            <JWTPreview token={auth.jwt?.token || ""} />
          </div>
        )
      default:
        return <p className="py-6 text-center text-sm text-muted-foreground">Unknown authentication type.</p>
    }
  }

  const inheritedLabel = inheritedAuth?.type && inheritedAuth.type !== "noauth" && inheritedAuth.type !== "inherit"

  return (
    <div className="space-y-3">
      <div>
        <Label className="mb-1 block text-xs font-medium text-muted-foreground">Auth type</Label>
        <Select value={activeType} onValueChange={handleTypeChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {AUTH_TYPES.map((type) => (
              <SelectItem key={type.value} value={type.value}>
                <div className="flex items-center gap-2">
                  {type.icon}
                  <div>
                    <div className="font-medium">
                      {type.label}
                      {type.value === "inherit" && inheritedLabel && (
                        <span className="ml-1 text-[10px] text-muted-foreground">({inheritedAuth?.type})</span>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground">{type.description}</div>
                  </div>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="min-h-[150px] rounded-md border border-border bg-muted/30 p-3">{renderFields()}</div>
    </div>
  )
}

function NotSupportedNotice({ kind }: { kind: string }) {
  return (
    <div className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive">
      <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
      <span>{kind} signing is not implemented. Requests using this auth type fail with a clear error.</span>
    </div>
  )
}
