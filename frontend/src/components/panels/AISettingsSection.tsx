import React, { useEffect, useState } from "react"
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
import {
  CheckCircle2,
  Loader2,
  Plus,
  RefreshCw,
  Sparkles,
  Trash2,
  XCircle,
  Zap,
} from "lucide-react"
import * as api from "@/lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { AIConnectionTest, AIProviderConfig, AIProviderDescriptor } from "@/types"

/**
 * Offline fallback used only when the provider descriptor endpoint cannot be
 * reached (e.g. the page is open outside the Wails runtime). The backend remains
 * the source of truth; this just keeps the form renderable.
 */
const FALLBACK_DESCRIPTORS: AIProviderDescriptor[] = [
  {
    kind: "openai",
    label: "OpenAI-compatible",
    description: "OpenAI, Azure-compatible gateways and most local servers.",
    defaultBaseUrl: "https://api.openai.com/v1",
    defaultModel: "gpt-4o-mini",
    suggestedModels: ["gpt-4o-mini", "gpt-4o", "gpt-4.1-mini"],
    requiresApiKey: true,
    supportsModelList: true,
    supportsTemperature: true,
    supportsMaxTokens: true,
  },
  {
    kind: "custom",
    label: "Custom",
    description: "Any endpoint that implements the OpenAI chat completions API.",
    defaultBaseUrl: "",
    defaultModel: "",
    suggestedModels: [],
    requiresApiKey: false,
    supportsModelList: false,
    supportsTemperature: true,
    supportsMaxTokens: true,
  },
]

function newProvider(descriptor: AIProviderDescriptor | undefined): AIProviderConfig {
  const id = `provider-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`
  return {
    id,
    name: descriptor?.label ?? "Custom provider",
    kind: descriptor?.kind ?? "custom",
    baseUrl: descriptor?.defaultBaseUrl ?? "",
    model: descriptor?.defaultModel ?? "",
    apiKey: "",
    apiKeySet: false,
    models: descriptor?.suggestedModels ?? [],
    temperature: 0.2,
    maxTokens: 2048,
    timeoutSeconds: 60,
    custom: (descriptor?.kind ?? "custom") === "custom",
  }
}

/**
 * agent, documentation …). Providers are stored in global settings so keys are
 * entered exactly once.
 */
export function AISettingsSection() {
  const settings = useWorkspaceStore((s) => s.settings)
  const updateSettings = useWorkspaceStore((s) => s.updateSettings)

  const [descriptors, setDescriptors] = useState<AIProviderDescriptor[]>(FALLBACK_DESCRIPTORS)
  const [descriptorError, setDescriptorError] = useState<string | null>(null)
  const [test, setTest] = useState<{ id: string; result: AIConnectionTest } | null>(null)
  const [testing, setTesting] = useState(false)
  const [fetchingModels, setFetchingModels] = useState(false)
  const [modelWarning, setModelWarning] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api
      .aiProviderDescriptors()
      .then((list) => {
        if (!cancelled && list.length > 0) setDescriptors(list)
      })
      .catch((err) => {
        if (!cancelled) setDescriptorError(api.errorMessage(err))
      })
    return () => {
      cancelled = true
    }
  }, [])

  const providers = settings.aiProviders ?? []
  const active =
    providers.find((p) => p.id === settings.aiProviderId) ?? providers[0] ?? null
  const descriptor = descriptors.find((d) => d.kind === active?.kind)
  const suggested = Array.from(
    new Set([...(descriptor?.suggestedModels ?? []), ...(active?.models ?? [])]),
  )

  const commit = (list: AIProviderConfig[], selectID?: string) => {
    const selected = list.find((p) => p.id === selectID) ?? list[0]
    updateSettings({
      aiProviders: list,
      aiProviderId: selected?.id ?? "",
      // Flat fields are kept in sync for older settings consumers.
      aiProvider: selected?.kind ?? "openai",
      aiBaseUrl: selected?.baseUrl ?? "",
      aiModel: selected?.model ?? "",
      aiTemperature: selected?.temperature ?? 0.2,
      aiMaxTokens: selected?.maxTokens ?? 2048,
      aiTimeoutSeconds: selected?.timeoutSeconds ?? 60,
      aiApiKey: selected?.apiKey ?? "",
      aiApiKeySet: !!selected?.apiKeySet || !!selected?.apiKey,
    })
  }

  const patchProvider = (patch: Partial<AIProviderConfig>) => {
    if (!active) return
    commit(providers.map((p) => (p.id === active.id ? { ...p, ...patch } : p)))
  }

  const changeKind = (kind: string) => {
    const next = descriptors.find((d) => d.kind === kind)
    patchProvider({
      kind,
      name: next?.label ?? kind,
      baseUrl: next?.defaultBaseUrl ?? "",
      model: next?.defaultModel ?? "",
      models: next?.suggestedModels ?? [],
      custom: kind === "custom",
    })
    setTest(null)
    setModelWarning(null)
  }

  const addProvider = () => {
    const provider = newProvider(
      descriptors.find((d) => d.kind === "custom") ?? descriptors[0],
    )
    commit([...providers, provider], provider.id)
    setTest(null)
  }

  const removeActive = () => {
    if (!active || providers.length <= 1) return
    const next = providers.filter((p) => p.id !== active.id)
    commit(next, next[0]?.id)
    setTest(null)
  }

  const runTest = async () => {
    if (!active) return
    setTesting(true)
    setTest(null)
    try {
      const result = await api.aiTestConnection(active)
      setTest({ id: active.id, result })
    } catch (err) {
      setTest({
        id: active.id,
        result: { ok: false, message: api.errorMessage(err), latencyMs: 0 },
      })
    } finally {
      setTesting(false)
    }
  }

  const fetchModels = async () => {
    if (!active) return
    setFetchingModels(true)
    setModelWarning(null)
    try {
      const list = await api.aiListModels(active)
      patchProvider({ models: list.models })
      if (list.warning) setModelWarning(list.warning)
      else if (list.models.length === 0) setModelWarning("The provider returned no models.")
    } catch (err) {
      setModelWarning(api.errorMessage(err))
    } finally {
      setFetchingModels(false)
    }
  }

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-2">
        <span className="flex items-center gap-1.5 text-xs font-semibold uppercase text-muted-foreground">
          <Sparkles className="h-3.5 w-3.5" /> AI providers
        </span>
        <Button size="sm" variant="outline" className="h-7 gap-1 px-2 text-xs" onClick={addProvider}>
          <Plus className="h-3.5 w-3.5" /> Add
        </Button>
      </div>

      <label className="flex items-center justify-between gap-2 cursor-pointer">
        <span>Enable AI features</span>
        <Checkbox
          checked={!!settings.aiEnabled}
          onCheckedChange={(c) => updateSettings({ aiEnabled: c === true })}
        />
      </label>

      {descriptorError && (
        <p className="pw-text-2xs text-amber-500">
          Provider descriptors unavailable: {descriptorError}
        </p>
      )}

      <div className="flex items-end gap-2">
        <div className="flex-1">
          <span className="text-xs text-muted-foreground">Active provider</span>
          <Select
            value={active?.id ?? ""}
            onValueChange={(id) => {
              if (!providers.some((p) => p.id === id)) return
              commit(providers, id)
              setTest(null)
            }}
          >
            <SelectTrigger className="mt-1 h-8">
              <SelectValue placeholder="Select a provider" />
            </SelectTrigger>
            <SelectContent>
              {providers.map((p) => (
                <SelectItem key={p.id} value={p.id}>
                  {p.name || p.kind}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <Button
          size="sm"
          variant="ghost"
          className="h-8 px-2 text-destructive"
          disabled={providers.length <= 1}
          onClick={removeActive}
          title="Remove provider"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      {active && (
        <div className="space-y-2 rounded-md border p-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-xs text-muted-foreground">
              {active.apiKeySet || active.apiKey ? "Credentials stored" : "No credentials yet"}
            </span>
            <div className="flex gap-1">
              <Button
                size="sm"
                variant="outline"
                className="h-7 gap-1 px-2 text-xs"
                onClick={runTest}
                disabled={testing}
              >
                {testing ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Zap className="h-3.5 w-3.5" />
                )}
                Test
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-7 gap-1 px-2 text-xs"
                onClick={fetchModels}
                disabled={fetchingModels || descriptor?.supportsModelList === false}
                title="Fetch the model list from the provider"
              >
                {fetchingModels ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="h-3.5 w-3.5" />
                )}
                Models
              </Button>
            </div>
          </div>

          <div className="grid grid-cols-2 gap-2">
            <div>
              <span className="text-xs text-muted-foreground">Protocol</span>
              <Select value={active.kind} onValueChange={changeKind}>
                <SelectTrigger className="mt-1 h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {descriptors.map((d) => (
                    <SelectItem key={d.kind} value={d.kind}>
                      {d.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Display name</span>
              <Input
                className="mt-1 h-8 text-xs"
                value={active.name}
                onChange={(e) => patchProvider({ name: e.target.value })}
                placeholder="My provider"
              />
            </div>
          </div>

          <div>
            <span className="text-xs text-muted-foreground">Base URL</span>
            <Input
              className="mt-1 h-8 font-mono text-xs"
              value={active.baseUrl}
              onChange={(e) => patchProvider({ baseUrl: e.target.value })}
              placeholder="https://api.openai.com/v1"
            />
          </div>

          <div>
            <span className="text-xs text-muted-foreground">Model</span>
            <Input
              className="mt-1 h-8 font-mono text-xs"
              list="ai-model-options"
              value={active.model}
              onChange={(e) => patchProvider({ model: e.target.value })}
              placeholder={descriptor?.defaultModel || "gpt-4o-mini"}
            />
            <datalist id="ai-model-options">
              {suggested.map((m) => (
                <option key={m} value={m} />
              ))}
            </datalist>
          </div>

          <div>
            <span className="text-xs text-muted-foreground">
              API key {active.apiKeySet && !active.apiKey ? "(stored — leave blank to keep)" : ""}
            </span>
            <Input
              type="password"
              className="mt-1 h-8 font-mono text-xs"
              value={active.apiKey ?? ""}
              onChange={(e) =>
                patchProvider({ apiKey: e.target.value, apiKeySet: e.target.value.length > 0 })
              }
              placeholder={active.apiKeySet ? "••••••••" : "sk-…"}
              autoComplete="off"
            />
          </div>

          <div className="grid grid-cols-3 gap-2">
            <div>
              <span className="text-xs text-muted-foreground">Temperature</span>
              <Input
                type="number"
                step="0.1"
                min="0"
                max="2"
                className="mt-1 h-8 text-xs"
                value={active.temperature}
                onChange={(e) => patchProvider({ temperature: Number(e.target.value) || 0 })}
              />
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Max tokens</span>
              <Input
                type="number"
                min="1"
                className="mt-1 h-8 text-xs"
                value={active.maxTokens}
                onChange={(e) => patchProvider({ maxTokens: parseInt(e.target.value) || 2048 })}
              />
            </div>
            <div>
              <span className="text-xs text-muted-foreground">Timeout (s)</span>
              <Input
                type="number"
                min="1"
                className="mt-1 h-8 text-xs"
                value={active.timeoutSeconds}
                onChange={(e) => patchProvider({ timeoutSeconds: parseInt(e.target.value) || 60 })}
              />
            </div>
          </div>

          {test && test.id === active.id && (
            <div
              className={`flex items-start gap-1.5 rounded px-2 py-1 text-xs ${
                test.result.ok
                  ? "bg-emerald-500/10 text-emerald-600"
                  : "bg-destructive/10 text-destructive"
              }`}
            >
              {test.result.ok ? (
                <CheckCircle2 className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              ) : (
                <XCircle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              )}
              <span className="break-words">
                {test.result.message}
                {test.result.ok ? ` (${test.result.latencyMs} ms)` : ""}
              </span>
            </div>
          )}

          {modelWarning && <p className="pw-text-2xs text-amber-500">{modelWarning}</p>}
        </div>
      )}

      <p className="pw-text-2xs text-muted-foreground">
        Keys are stored locally with your application settings and are only sent to the provider you
        configure. AI features call the provider only when you explicitly run them.
      </p>
    </section>
  )
}