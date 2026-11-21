import React, { useEffect, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { Plug, PlugZap, Send, Trash2, X } from "lucide-react"
import * as api from "@/lib/api"
import type { WebSocketMessage } from "@/types"

interface LiveConnection {
  id: string
  url: string
  connected: boolean
}

export function WebSocketPanel() {
  const [url, setUrl] = useState("wss://echo.websocket.org")
  const [connection, setConnection] = useState<LiveConnection | null>(null)
  const [messages, setMessages] = useState<WebSocketMessage[]>([])
  const [outgoing, setOutgoing] = useState("")
  const [error, setError] = useState<string | null>(null)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const unsub = api.onEvent<{ connectionId: string; message: WebSocketMessage }>("ws:message", (payload) => {
      setMessages((prev) => {
        if (connection && payload.connectionId !== connection.id) return prev
        return [...prev, payload.message]
      })
      if (payload.message.type === "system" && payload.message.data === "Disconnected") {
        setConnection((c) => (c ? { ...c, connected: false } : c))
      }
    })
    return unsub
  }, [connection])

  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight })
  }, [messages])

  const connect = async () => {
    setError(null)
    try {
      const conn = await api.wsConnect(url.trim())
      setConnection({ id: conn.id, url: conn.url, connected: true })
      setMessages(conn.messages ?? [])
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  const disconnect = async () => {
    if (!connection) return
    try {
      await api.wsDisconnect(connection.id)
    } catch (err) {
      setError(api.errorMessage(err))
    }
    setConnection((c) => (c ? { ...c, connected: false } : c))
  }

  const send = async () => {
    if (!connection || !outgoing) return
    setError(null)
    try {
      await api.wsSend(connection.id, outgoing, false)
      setOutgoing("")
    } catch (err) {
      setError(api.errorMessage(err))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-1 px-2 py-1.5 border-b border-border">
        <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground flex-1">WebSocket</span>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          title="Clear messages"
          aria-label="Clear messages"
          disabled={messages.length === 0}
          onClick={() => setMessages([])}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>

      <div className="p-2 space-y-2 border-b border-border">
        <div className="flex items-center gap-1">
          <Input
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="ws:// or wss:// URL"
            className="h-8 text-sm font-mono"
            disabled={connection?.connected}
          />
          {connection?.connected ? (
            <Button variant="destructive" size="sm" className="h-8 gap-1" onClick={disconnect}>
              <X className="h-3.5 w-3.5" /> Close
            </Button>
          ) : (
            <Button size="sm" className="h-8 gap-1" onClick={connect}>
              {connection ? <PlugZap className="h-3.5 w-3.5" /> : <Plug className="h-3.5 w-3.5" />}
              Connect
            </Button>
          )}
        </div>
        {error && <div className="text-xs text-destructive">{error}</div>}
        {connection && (
          <div className="text-[11px] text-muted-foreground flex items-center gap-1">
            <span className={cn("h-2 w-2 rounded-full", connection.connected ? "bg-green-500" : "bg-muted-foreground")} />
            {connection.connected ? "Connected" : "Disconnected"}
          </div>
        )}
      </div>

      <div ref={listRef} className="flex-1 overflow-y-auto p-2 space-y-1 font-mono text-xs">
        {messages.length === 0 && <div className="text-muted-foreground">No messages yet.</div>}
        {messages.map((m) => (
          <div
            key={m.id}
            className={cn(
              "rounded px-2 py-1 break-all",
              m.type === "sent" && "bg-primary/10 text-foreground",
              m.type === "received" && "bg-muted",
              m.type === "system" && "text-muted-foreground italic",
              m.type === "error" && "bg-destructive/10 text-destructive",
            )}
          >
            <span className="opacity-60 mr-1">{new Date(m.timestamp).toLocaleTimeString()}</span>
            {m.data}
          </div>
        ))}
      </div>

      <div className="p-2 border-t border-border flex items-center gap-1">
        <Input
          value={outgoing}
          onChange={(e) => setOutgoing(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && send()}
          placeholder="Message to send"
          className="h-8 text-sm font-mono"
          disabled={!connection?.connected}
        />
        <Button size="icon" className="h-8 w-8" onClick={send} disabled={!connection?.connected} aria-label="Send message">
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
