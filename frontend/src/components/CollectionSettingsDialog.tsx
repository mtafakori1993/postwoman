import React, { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog"
import { KeyValueEditor } from "./KeyValueEditor"
import { AuthEditor } from "./AuthEditor"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import * as api from "../lib/api"
import { useWorkspaceStore } from "@/stores/workspace"
import type { AuthConfig, Collection, KeyValuePair, Variable } from "@/types"

interface CollectionSettingsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  collection: Collection | null
}

function toRows(variables: Variable[] | undefined): KeyValuePair[] {
  return (variables ?? []).map((v) => ({
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

/** Collection-level name, description, variables and inherited auth. */
export function CollectionSettingsDialog({ open, onOpenChange, collection }: CollectionSettingsDialogProps) {
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [rows, setRows] = useState<KeyValuePair[]>([])
  const [auth, setAuth] = useState<AuthConfig>({ type: "inherit" })
  const [dirty, setDirty] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!collection) return
    setName(collection.info.name)
    setDescription(collection.info.description || "")
    setRows(toRows(collection.variables))
    setAuth(collection.auth ?? { type: "inherit" })
    setDirty(false)
    setError(null)
  }, [collection])

  const save = async () => {
    if (!collection) return
    try {
      await useWorkspaceStore.getState().updateCollectionMeta({
        ...collection,
        info: { ...collection.info, name, description },
        variables: toVariables(rows),
        auth,
      })
      setDirty(false)
      onOpenChange(false)
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Collection settings</DialogTitle>
          <DialogDescription>Name, documentation, variables and the auth requests inherit by default.</DialogDescription>
        </DialogHeader>

        <Tabs defaultValue="general">
          <TabsList>
            <TabsTrigger value="general">General</TabsTrigger>
            <TabsTrigger value="variables">Variables</TabsTrigger>
            <TabsTrigger value="auth">Auth</TabsTrigger>
          </TabsList>

          <TabsContent value="general" className="space-y-2">
            <div>
              <Label className="text-xs">Name</Label>
              <Input
                value={name}
                onChange={(e) => {
                  setName(e.target.value)
                  setDirty(true)
                }}
              />
            </div>
            <div>
              <Label className="text-xs">Description / notes</Label>
              <Textarea
                value={description}
                onChange={(e) => {
                  setDescription(e.target.value)
                  setDirty(true)
                }}
                className="min-h-[110px]"
                placeholder="What is this API? Base URL, environments, auth notes…"
              />
            </div>
          </TabsContent>

          <TabsContent value="variables">
            <KeyValueEditor
              items={rows}
              onChange={(next) => {
                setRows(next)
                setDirty(true)
              }}
              columns={[
                { key: "enabled", header: "", width: "40px", type: "checkbox" },
                { key: "key", header: "Variable", width: "26%" },
                { key: "value", header: "Value", width: "42%" },
                { key: "type", header: "Type", width: "16%" },
                { key: "description", header: "Description", width: "18%" },
              ]}
              emptyMessage="No collection variables yet."
            />
            <p className="mt-2 text-[11px] text-muted-foreground">
              Rows with type <code>secret</code> are masked in the UI and excluded from exports where possible.
            </p>
          </TabsContent>

          <TabsContent value="auth">
            <AuthEditor
              auth={auth}
              onChange={(next) => {
                setAuth(next)
                setDirty(true)
              }}
            />
            <p className="mt-2 text-[11px] text-muted-foreground">
              Requests whose auth type is “Inherit from parent” use this configuration (folders take precedence over the collection).
            </p>
          </TabsContent>
        </Tabs>

        {error && <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div>}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={save} disabled={!dirty && !error}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
