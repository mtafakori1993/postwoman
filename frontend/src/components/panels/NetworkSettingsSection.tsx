import React, { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { CheckCircle2, Globe, Loader2, XCircle, Zap } from "lucide-react"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { ProxyConfig, ProxyTestResult } from "@/types"

const DEFAULT_PROXY: ProxyConfig = { host: "", port: 8080, protocol: "http", bypass: [] }

/**
 * Global proxy configuration shared by API requests and AI provider requests.
 * A single place to point both kinds of traffic at HTTP/HTTPS/SOCKS5 proxies,
 * with per-scope switches so a corporate proxy can cover API calls without
 * capturing local AI provider traffic (or the other way round).
 */
export function NetworkSettingsSection() {
  const settings = useWorkspaceStore((s) => s.settings)
  const updateSettings = useWorkspaceStore((s) => s.updateSettings)

  const [test, setTest] = useState<ProxyTestResult | null>(null)
  const [testing, setTesting] = useState(false)

  const proxy: ProxyConfig = { ...DEFAULT_PROXY, ...(settings.proxy ?? {}) }
  const enabled = !!settings.proxyEnabled

  const patchProxy = (patch: Partial<ProxyConfig>) => {
    updateSettings({ proxy: { ...proxy, ...patch } })
    setTest(null)
  }

  const toggleEnabled = (on: boolean) => {
    updateSettings({
      proxyEnabled: on,
      // Enable both scopes when switching the proxy on; the user can narrow it
      // afterwards. Keeping the configured host/port intact avoids re-typing.
      proxyForRequests: on ? true : settings.proxyForRequests,
      proxyForAI: on ? true : settings.proxyForAI,
      proxy: { ...DEFAULT_PROXY, ...(settings.proxy ?? {}) },
    })
    setTest(null)
  }

  const runTest = async () => {
    setTesting(true)
    setTest(null)
    try {
      setTest(await api.testProxy(proxy))
    } catch (err) {
      setTest({ ok: false, message: api.errorMessage(err), latencyMs: 0 })
    } finally {
      setTesting(false)
    }
  }

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <span className="flex items-center gap-1.5 text-xs font-semibold uppercase text-muted-foreground">
          <Globe className="h-3.5 w-3.5" /> Network &amp; proxy
        </span>
        <label className="flex cursor-pointer items-center gap-2">
          <span className="text-xs text-muted-foreground">{enabled ? "Enabled" : "Disabled"}</span>
          <Checkbox checked={enabled} onCheckedChange={(c) => toggleEnabled(c === true)} />
        </label>
      </div>

      {!enabled && (
        <p className="pw-text-2xs text-muted-foreground">
          Route API and AI provider traffic through a proxy. Off by default — requests use the
          system proxy environment when available.
        </p>
      )}

      {enabled && (
        <div className="space-y-2 rounded-md border p-2">
          <div className="grid grid-cols-4 gap-2">
            <div className="col-span-2">
              <span className="text-xs text-muted-foreground">Host</span>
              <Input
                className="mt-1 h-8 font-mono text-xs"
                placeholder="proxy.example.com"
                value={proxy.host ?? ""}
                onChange={(e) => patchProxy({ host: e.target.value })}
              />
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Port</span>
              <Input
                type="number"
                min="1"
                max="65535"
                className="mt-1 h-8 text-xs"
                value={proxy.port ?? ""}
                onChange={(e) => patchProxy({ port: parseInt(e.target.value) || 0 })}
              />
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Protocol</span>
              <Select
                value={proxy.protocol || "http"}
                onValueChange={(value) => patchProxy({ protocol: value })}
              >
                <SelectTrigger className="mt-1 h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="http">HTTP</SelectItem>
                  <SelectItem value="https">HTTPS</SelectItem>
                  <SelectItem value="socks5">SOCKS5</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <span className="text-xs text-muted-foreground">Username (optional)</span>
              <Input
                className="mt-1 h-8 font-mono text-xs"
                autoComplete="off"
                value={proxy.username ?? ""}
                onChange={(e) => patchProxy({ username: e.target.value })}
              />
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Password (optional)</span>
              <Input
                type="password"
                className="mt-1 h-8 font-mono text-xs"
                autoComplete="off"
                value={proxy.password ?? ""}
                onChange={(e) => patchProxy({ password: e.target.value })}
              />
            </div>
          </div>

          <div>
            <span className="text-xs text-muted-foreground">Bypass list</span>
            <Input
              className="mt-1 h-8 font-mono text-xs"
              placeholder="localhost, *.internal, 127.0.0.1"
              value={(proxy.bypass ?? []).join(", ")}
              onChange={(e) =>
                patchProxy({
                  bypass: e.target.value
                    .split(",")
                    .map((v) => v.trim())
                    .filter(Boolean),
                })
              }
            />
            <p className="mt-1 pw-text-2xs text-muted-foreground">
              Comma-separated hosts or wildcards that connect directly.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <label className="flex cursor-pointer items-center justify-between gap-2">
              <span className="text-xs">Use for API requests</span>
              <Checkbox
                checked={!!settings.proxyForRequests}
                onCheckedChange={(c) => updateSettings({ proxyForRequests: c === true })}
              />
            </label>
            <label className="flex cursor-pointer items-center justify-between gap-2">
              <span className="text-xs">Use for AI providers</span>
              <Checkbox
                checked={!!settings.proxyForAI}
                onCheckedChange={(c) => updateSettings({ proxyForAI: c === true })}
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="pw-text-2xs text-muted-foreground">
              A request-level proxy still overrides this.
            </span>
            <Button
              size="sm"
              variant="outline"
              className="h-7 gap-1 px-2 text-xs"
              onClick={runTest}
              disabled={testing}
            >
              {testing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Zap className="h-3.5 w-3.5" />}
              Test
            </Button>
          </div>

          {test && (
            <div
              className={`flex items-start gap-1.5 rounded px-2 py-1 text-xs ${
                test.ok ? "bg-emerald-500/10 text-emerald-600" : "bg-destructive/10 text-destructive"
              }`}
            >
              {test.ok ? (
                <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              ) : (
                <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              )}
              <span className="break-words">
                {test.message}
                {test.ok ? ` (${test.latencyMs} ms)` : ""}
              </span>
            </div>
          )}
        </div>
      )}

      <p className="pw-text-2xs text-muted-foreground">
        Proxy credentials are stored locally with your settings and are never logged.
      </p>
    </section>
  )
}
