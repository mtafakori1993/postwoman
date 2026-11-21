import React, { useState, useEffect } from "react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { KeyValuePair, AuthConfig, AuthType } from "@/types"
import { Key, Shield, User, Globe, Lock, Fingerprint } from "lucide-react"

const AUTH_TYPES: { value: AuthType; label: string; icon: React.ReactNode; description: string }[] = [
  { value: "noauth", label: "No Auth", icon: <Shield className="h-4 w-4" />, description: "No authentication" },
  { value: "apikey", label: "API Key", icon: <Key className="h-4 w-4" />, description: "API Key authentication" },
  { value: "bearer", label: "Bearer Token", icon: <Fingerprint className="h-4 w-4" />, description: "Bearer token authentication" },
  { value: "basic", label: "Basic Auth", icon: <User className="h-4 w-4" />, description: "Username and password" },
  { value: "digest", label: "Digest Auth", icon: <Lock className="h-4 w-4" />, description: "Digest authentication" },
  { value: "oauth1", label: "OAuth 1.0", icon: <Globe className="h-4 w-4" />, description: "OAuth 1.0 authentication" },
  { value: "oauth2", label: "OAuth 2.0", icon: <Globe className="h-4 w-4" />, description: "OAuth 2.0 authentication" },
  { value: "awsv4", label: "AWS Signature", icon: <Lock className="h-4 w-4" />, description: "AWS Signature v4" },
  { value: "hawk", label: "Hawk", icon: <Key className="h-4 w-4" />, description: "Hawk authentication" },
  { value: "ntlm", label: "NTLM", icon: <User className="h-4 w-4" />, description: "NTLM authentication" },
  { value: "jwt", label: "JWT Bearer", icon: <Fingerprint className="h-4 w-4" />, description: "JWT Bearer token" },
]

interface AuthEditorProps {
  auth: AuthConfig
  onChange: (auth: AuthConfig) => void
}

/** Fill missing fields from `defaults` so the result always has every field. */
function withDefaults<T extends object>(defaults: T, current?: Partial<T>): T {
  return { ...defaults, ...(current ?? {}) }
}

/** Merge a partial change into an existing (possibly absent) sub-config. */
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
          {
            consumerKey: "",
            consumerSecret: "",
            token: "",
            tokenSecret: "",
            signatureMethod: "HMAC-SHA1",
            version: "1.0",
          },
          auth.oauth1,
        ),
      }
    case "oauth2":
      return {
        ...base,
        oauth2: withDefaults(
          {
            grantType: "authorization_code",
            accessToken: "",
            refreshToken: "",
            clientId: "",
            clientSecret: "",
            authUrl: "",
            tokenUrl: "",
            scope: "",
            redirectUrl: "",
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

export function AuthEditor({ auth, onChange }: AuthEditorProps) {
  const [activeType, setActiveType] = useState<AuthType>(auth.type || "noauth")

  useEffect(() => {
    setActiveType(auth.type || "noauth")
  }, [auth])

  const handleTypeChange = (type: AuthType) => {
    setActiveType(type)
    onChange(getAuthConfig(auth, type))
  }

  const renderAuthFields = () => {
    switch (activeType) {
      case "apikey":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-3 gap-2">
              <Input
                placeholder="Key"
                value={auth.apikey?.key || ""}
                onChange={(e) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { key: e.target.value }) })}
              />
              <Input
                placeholder="Value"
                value={auth.apikey?.value || ""}
                onChange={(e) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { value: e.target.value }) })}
              />
              <Select
                value={auth.apikey?.addTo || "header"}
                onValueChange={(value) => onChange({ ...auth, apikey: mergeAuth(auth.apikey, { addTo: value as "header" | "query" }) })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="header">Header</SelectItem>
                  <SelectItem value="query">Query Params</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        )

      case "bearer":
        return (
          <div className="space-y-3">
            <Label>Token</Label>
            <Input
              type="password"
              placeholder="Enter bearer token"
              value={auth.bearer?.token || ""}
              onChange={(e) => onChange({ ...auth, bearer: { token: e.target.value } })}
            />
          </div>
        )

      case "basic":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Username"
                value={auth.basic?.username || ""}
                onChange={(e) => onChange({ ...auth, basic: mergeAuth(auth.basic, { username: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Password"
                value={auth.basic?.password || ""}
                onChange={(e) => onChange({ ...auth, basic: mergeAuth(auth.basic, { password: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "digest":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Username"
                value={auth.digest?.username || ""}
                onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { username: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Password"
                value={auth.digest?.password || ""}
                onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { password: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Realm (optional)"
                value={auth.digest?.realm || ""}
                onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { realm: e.target.value }) })}
              />
              <Input
                placeholder="Nonce (optional)"
                value={auth.digest?.nonce || ""}
                onChange={(e) => onChange({ ...auth, digest: mergeAuth(auth.digest, { nonce: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "oauth1":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Consumer Key"
                value={auth.oauth1?.consumerKey || ""}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { consumerKey: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Consumer Secret"
                value={auth.oauth1?.consumerSecret || ""}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { consumerSecret: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Token"
                value={auth.oauth1?.token || ""}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { token: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Token Secret"
                value={auth.oauth1?.tokenSecret || ""}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { tokenSecret: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <Select
                value={auth.oauth1?.signatureMethod || "HMAC-SHA1"}
                onValueChange={(value) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { signatureMethod: value as "HMAC-SHA1" | "HMAC-SHA256" | "PLAINTEXT" }) })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="HMAC-SHA1">HMAC-SHA1</SelectItem>
                  <SelectItem value="HMAC-SHA256">HMAC-SHA256</SelectItem>
                  <SelectItem value="PLAINTEXT">PLAINTEXT</SelectItem>
                </SelectContent>
              </Select>
              <Input
                placeholder="Version"
                value={auth.oauth1?.version || "1.0"}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { version: e.target.value as "1.0" }) })}
              />
              <Input
                placeholder="Realm (optional)"
                value={auth.oauth1?.realm || ""}
                onChange={(e) => onChange({ ...auth, oauth1: mergeAuth(auth.oauth1, { realm: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "oauth2":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Client ID"
                value={auth.oauth2?.clientId || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { clientId: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Client Secret"
                value={auth.oauth2?.clientSecret || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { clientSecret: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Auth URL"
                value={auth.oauth2?.authUrl || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { authUrl: e.target.value }) })}
              />
              <Input
                placeholder="Token URL"
                value={auth.oauth2?.tokenUrl || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { tokenUrl: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <Select
                value={auth.oauth2?.grantType || "authorization_code"}
                onValueChange={(value) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { grantType: value as "authorization_code" | "client_credentials" | "password" | "refresh_token" }) })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="authorization_code">Auth Code</SelectItem>
                  <SelectItem value="client_credentials">Client Credentials</SelectItem>
                  <SelectItem value="password">Password</SelectItem>
                  <SelectItem value="refresh_token">Refresh Token</SelectItem>
                </SelectContent>
              </Select>
              <Input
                placeholder="Scope (optional)"
                value={auth.oauth2?.scope || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { scope: e.target.value }) })}
              />
              <Input
                placeholder="Redirect URL (optional)"
                value={auth.oauth2?.redirectUrl || ""}
                onChange={(e) => onChange({ ...auth, oauth2: mergeAuth(auth.oauth2, { redirectUrl: e.target.value }) })}
              />
            </div>
            <Button variant="outline" onClick={() => alert("OAuth 2.0 flow not implemented yet")}>
              Get New Access Token
            </Button>
          </div>
        )

      case "awsv4":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Access Key"
                value={auth.awsv4?.accessKey || ""}
                onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { accessKey: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Secret Key"
                value={auth.awsv4?.secretKey || ""}
                onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { secretKey: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Region"
                value={auth.awsv4?.region || ""}
                onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { region: e.target.value }) })}
              />
              <Input
                placeholder="Service"
                value={auth.awsv4?.service || ""}
                onChange={(e) => onChange({ ...auth, awsv4: mergeAuth(auth.awsv4, { service: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "hawk":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="ID"
                value={auth.hawk?.id || ""}
                onChange={(e) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { id: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Key"
                value={auth.hawk?.key || ""}
                onChange={(e) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { key: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <Select
                value={auth.hawk?.algorithm || "sha256"}
                onValueChange={(value) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { algorithm: value as "sha256" | "sha1" }) })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="sha256">SHA-256</SelectItem>
                  <SelectItem value="sha1">SHA-1</SelectItem>
                </SelectContent>
              </Select>
              <Input
                placeholder="User (optional)"
                value={auth.hawk?.user || ""}
                onChange={(e) => onChange({ ...auth, hawk: mergeAuth(auth.hawk, { user: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "ntlm":
        return (
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Username"
                value={auth.ntlm?.username || ""}
                onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { username: e.target.value }) })}
              />
              <Input
                type="password"
                placeholder="Password"
                value={auth.ntlm?.password || ""}
                onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { password: e.target.value }) })}
              />
            </div>
            <div className="grid grid-cols-2 gap-2">
              <Input
                placeholder="Domain (optional)"
                value={auth.ntlm?.domain || ""}
                onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { domain: e.target.value }) })}
              />
              <Input
                placeholder="Workstation (optional)"
                value={auth.ntlm?.workstation || ""}
                onChange={(e) => onChange({ ...auth, ntlm: mergeAuth(auth.ntlm, { workstation: e.target.value }) })}
              />
            </div>
          </div>
        )

      case "jwt":
        return (
          <div className="space-y-3">
            <Label>JWT Token</Label>
            <Input
              type="password"
              placeholder="Enter JWT token"
              value={auth.jwt?.token || ""}
              onChange={(e) => onChange({ ...auth, jwt: { token: e.target.value } })}
            />
          </div>
        )

      default:
        return (
          <div className="text-center text-muted-foreground py-8">
            <p>No authentication required</p>
          </div>
        )
    }
  }

  return (
    <div className="space-y-3">
      <div>
        <Label className="text-xs font-medium text-muted-foreground mb-1 block">Auth Type</Label>
        <Select value={activeType} onValueChange={handleTypeChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {AUTH_TYPES.map(t => (
              <SelectItem key={t.value} value={t.value}>
                <div className="flex items-center gap-2">
                  {t.icon}
                  <div>
                    <div className="font-medium">{t.label}</div>
                    <div className="text-xs text-muted-foreground">{t.description}</div>
                  </div>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="border border-border rounded-md p-3 bg-muted/30 min-h-[150px]">
        {renderAuthFields()}
      </div>
    </div>
  )
}