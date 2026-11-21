import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { KeyValueEditor } from "@/components/KeyValueEditor"
import { Check, Copy, Download, Plus, Save, Trash2, Upload } from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import * as api from "@/lib/api"
import type { Environment, GlobalVariable, KeyValuePair, Variable } from "@/types"

function toKeyValues(vars: Variable[]): KeyValuePair[] {
  return vars.map((v) => ({
    key: v.key,
    value: v.value,
    description: v.description,
    enabled: v.enabled !== false,
    type: v.type || "default",
  }))
}

function toVariables(rows: KeyValuePair[]): Variable[] {
  return rows.map((r) => ({
    key: r.key,
    value: r.value,
    description: r.description,
    enabled: r.enabled !== false,
    type: (r.type as string) || "default",
  }))
}

export function EnvironmentsPanel() {
  const {
    environments,
    activeEnvironmentId,
    globalVariables,
    createEnvironment,
    saveEnvironment,
    deleteEnvironment,
    setActiveEnvironment,
    setGlobalVariable,
    deleteGlobalVariable,
    collections,
    updateCollectionMeta,
    importFromFile,
  } = useWorkspaceStore()

  const [selectedId, setSelectedId] = useState<string | null>(activeEnvironmentId)
  const [draft, setDraft] = useState<Environment | null>(null)
  const [dirty, setDirty] = useState(false)
  const [globalRows, setGlobalRows] = useState<KeyValuePair[]>([])
  const [selectedCollectionId, setSelectedCollectionId] = useState<string | null>(null)
  const selectedCollection = collections.find((c) => c.id === selectedCollectionId) ?? collections[0] ?? null

  useEffect(() => {
    setGlobalRows(
      globalVariables.map((g: GlobalVariable) => ({
        key: g.key,
        value: g.value,
        description: g.description,
        enabled: g.enabled !== false,
        type: g.type || "default",
      })),
    )
  }, [globalVariables])

  useEffect(() => {
    const env = environments.find((e) => e.id === selectedId) ?? environments[0] ?? null
    if (env) {
      setSelectedId(env.id)
      setDraft({ ...env, values: env.values ?? [] })
      setDirty(false)
    } else {
      setDraft(null)
    }
  }, [environments, selectedId])

  const persistGlobals = async (rows: KeyValuePair[]) => {
    const previous = globalRows
    setGlobalRows(rows)
    const removed = previous.filter((p) => !rows.some((r) => r.key === p.key))
    for (const r of removed) {
      if (r.key) await deleteGlobalVariable(r.key)
    }
    for (const row of rows) {
      if (!row.key) continue
      await setGlobalVariable(row.key, row.value, (row.type as string) || "default", row.description)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">Environments</span>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          title="Duplicate environment"
          aria-label="Duplicate environment"
          disabled={!draft}
          onClick={async () => {
            if (!draft) return
            const created = await createEnvironment(`${draft.name} copy`)
            if (created) {
              await saveEnvironment({ ...created, values: draft.values })
              setSelectedId(created.id)
            }
          }}
        >
          <Copy className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          title="Import environments / collections"
          aria-label="Import"
          onClick={() => void importFromFile().catch(console.error)}
        >
          <Upload className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          title="New environment"
          aria-label="New environment"
          onClick={async () => {
            const env = await createEnvironment("New Environment")
            if (env) setSelectedId(env.id)
          }}
        >
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      <div className="max-h-40 overflow-y-auto border-b border-border py-1">
        {environments.length === 0 && (
          <div className="px-3 py-3 text-xs text-muted-foreground">No environments yet.</div>
        )}
        {environments.map((env) => (
          <div
            key={env.id}
            className={cn(
              "group flex items-center gap-2 px-2 py-1 cursor-pointer hover:bg-muted/50",
              selectedId === env.id && "bg-muted",
            )}
            onClick={() => setSelectedId(env.id)}
          >
            <button
              className={cn(
                "h-4 w-4 rounded-full border flex items-center justify-center shrink-0",
                activeEnvironmentId === env.id ? "bg-primary border-primary" : "border-border",
              )}
              title={activeEnvironmentId === env.id ? "Active" : "Set active"}
              aria-label="Set active environment"
              onClick={(e) => {
                e.stopPropagation()
                setActiveEnvironment(activeEnvironmentId === env.id ? null : env.id)
              }}
            >
              {activeEnvironmentId === env.id && <Check className="h-3 w-3 text-primary-foreground" />}
            </button>
            <span className="flex-1 text-sm truncate">{env.name}</span>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 p-0 opacity-0 group-hover:opacity-100"
              aria-label="Delete environment"
              onClick={(e) => {
                e.stopPropagation()
                deleteEnvironment(env.id)
              }}
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        ))}
      </div>

      <div className="flex-1 overflow-y-auto p-3 space-y-4">
        {draft ? (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Input
                value={draft.name}
                onChange={(e) => {
                  setDraft({ ...draft, name: e.target.value })
                  setDirty(true)
                }}
                className="h-8 text-sm"
                placeholder="Environment name"
              />
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                title="Export environment"
                aria-label="Export environment"
                onClick={() => api.exportEnvironmentToFile(draft.id).catch(console.error)}
              >
                <Download className="h-4 w-4" />
              </Button>
              <Button
                size="sm"
                className="h-8 gap-1"
                disabled={!dirty}
                onClick={async () => {
                  await saveEnvironment(draft)
                  setDirty(false)
                }}
              >
                <Save className="h-3.5 w-3.5" /> Save
              </Button>
            </div>

            <div>
              <Label className="text-xs font-medium text-muted-foreground mb-1 block">Variables</Label>
              <KeyValueEditor
                items={toKeyValues(draft.values)}
                onChange={(rows) => {
                  setDraft({ ...draft, values: toVariables(rows) })
                  setDirty(true)
                }}
                columns={[
                  { key: "enabled", header: "", width: "40px", type: "checkbox" },
                  { key: "key", header: "Variable", width: "28%" },
                  { key: "value", header: "Value", width: "40%" },
                  { key: "type", header: "Type (default / secret)", width: "24%" },
                ]}
                variableAutocomplete
              />
              <p className="mt-1 pw-text-2xs text-muted-foreground">
                Set the type to <code>secret</code> to mask the value here and in variable suggestions.
              </p>
            </div>
          </div>
        ) : (
          <div className="text-xs text-muted-foreground">Create an environment to store variables.</div>
        )}

        <div className="border-t border-border pt-3">
          <Label className="mb-1 block text-xs font-medium text-muted-foreground">Collection Variables</Label>
          {collections.length === 0 ? (
            <p className="pw-text-2xs text-muted-foreground">No collections yet.</p>
          ) : (
            <div className="space-y-2">
              <Select value={selectedCollection?.id ?? ""} onValueChange={setSelectedCollectionId}>
                <SelectTrigger className="h-8">
                  <SelectValue placeholder="Select a collection" />
                </SelectTrigger>
                <SelectContent>
                  {collections.map((collection) => (
                    <SelectItem key={collection.id} value={collection.id}>
                      {collection.info.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {selectedCollection && (
                <KeyValueEditor
                  items={toKeyValues(selectedCollection.variables ?? [])}
                  onChange={(rows) => {
                    void updateCollectionMeta({ ...selectedCollection, variables: toVariables(rows) })
                  }}
                  variableAutocomplete
                  columns={[
                    { key: "enabled", header: "", width: "40px", type: "checkbox" },
                    { key: "key", header: "Variable", width: "28%" },
                    { key: "value", header: "Value", width: "40%" },
                    { key: "type", header: "Type", width: "24%" },
                  ]}
                  emptyMessage="This collection has no variables."
                />
              )}
            </div>
          )}
        </div>

        <div className="pt-2 border-t border-border">
          <Label className="text-xs font-medium text-muted-foreground mb-1 block">Global Variables</Label>
          <KeyValueEditor
            items={globalRows}
            onChange={persistGlobals}
            columns={[
              { key: "enabled", header: "", width: "40px", type: "checkbox" },
              { key: "key", header: "Variable", width: "30%" },
              { key: "value", header: "Value", width: "45%" },
              { key: "type", header: "Type", width: "20%" },
            ]}
          />
        </div>
      </div>
    </div>
  )
}
