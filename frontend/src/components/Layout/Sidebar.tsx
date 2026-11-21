import React from "react"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip"
import { Logo } from "@/components/ui/logo"
import {
  ChevronLeft,
  ChevronRight,
  Folder,
  Globe,
  History,
  Server,
  Radio,
  Settings as SettingsIcon,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useWorkspaceStore } from "@/stores/workspace"
import type { SidebarTab } from "@/types"
import { CollectionsPanel } from "../panels/CollectionsPanel"
import { EnvironmentsPanel } from "../panels/EnvironmentsPanel"
import { HistoryPanel } from "../panels/HistoryPanel"
import { MocksPanel } from "../panels/MocksPanel"
import { WebSocketPanel } from "../panels/WebSocketPanel"
import { SettingsPanel } from "../panels/SettingsPanel"

const sidebarTabs: { id: SidebarTab; label: string; icon: React.ElementType }[] = [
  { id: "collections", label: "Collections", icon: Folder },
  { id: "environments", label: "Environments", icon: Globe },
  { id: "history", label: "History", icon: History },
  { id: "mocks", label: "Mock Servers", icon: Server },
  { id: "websocket", label: "WebSocket", icon: Radio },
  { id: "settings", label: "Settings", icon: SettingsIcon },
]

function Panel({ tab }: { tab: SidebarTab }) {
  switch (tab) {
    case "collections":
      return <CollectionsPanel />
    case "environments":
      return <EnvironmentsPanel />
    case "history":
      return <HistoryPanel />
    case "mocks":
      return <MocksPanel />
    case "websocket":
      return <WebSocketPanel />
    case "settings":
      return <SettingsPanel />
    default:
      return null
  }
}

export function Sidebar() {
  const {
    sidebarOpen,
    toggleSidebar,
    sidebarWidth,
    setSidebarWidth,
    activeSidebarTab,
    setActiveSidebarTab,
  } = useWorkspaceStore()

  if (!sidebarOpen) {
    return (
      <TooltipProvider>
        <div className="z-40 h-full w-12 bg-card border-r border-border flex flex-col items-center shrink-0">
          <div className="w-full flex items-center justify-center p-2 border-b border-border">
            <Logo size="sm" showText={false} />
          </div>
          <Button variant="ghost" size="icon" className="h-10 w-10 mt-2 mb-2" onClick={toggleSidebar} aria-label="Open sidebar">
            <ChevronRight className="h-5 w-5" />
          </Button>
          <div className="flex flex-col gap-1 px-2 flex-1 overflow-y-auto">
            {sidebarTabs.map((tab) => (
              <Tooltip key={tab.id}>
                <TooltipTrigger asChild>
                  <Button
                    variant={activeSidebarTab === tab.id ? "default" : "ghost"}
                    size="icon"
                    className="h-10 w-10"
                    onClick={() => {
                      setActiveSidebarTab(tab.id)
                      toggleSidebar()
                    }}
                    aria-label={tab.label}
                  >
                    <tab.icon className="h-5 w-5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="right">{tab.label}</TooltipContent>
              </Tooltip>
            ))}
          </div>
        </div>
      </TooltipProvider>
    )
  }

  return (
    <TooltipProvider>
      <div className="z-40 h-full bg-card border-r border-border flex flex-col shrink-0 relative" style={{ width: sidebarWidth }}>
        {/* Header */}
        <div className="flex items-center justify-between p-3 border-b border-border">
          <Logo size="md" showText={true} />
          <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={toggleSidebar} aria-label="Close sidebar">
            <ChevronLeft className="h-5 w-5" />
          </Button>
        </div>

        {/* Tab navigation */}
        <div className="flex gap-0.5 px-1.5 py-1.5 border-b border-border">
          {sidebarTabs.map((tab) => (
            <Tooltip key={tab.id}>
              <TooltipTrigger asChild>
                <Button
                  variant={activeSidebarTab === tab.id ? "default" : "ghost"}
                  size="icon"
                  className="h-8 w-8 flex-1"
                  onClick={() => setActiveSidebarTab(tab.id)}
                  aria-label={tab.label}
                >
                  <tab.icon className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{tab.label}</TooltipContent>
            </Tooltip>
          ))}
        </div>

        {/* Panel */}
        <div className="flex-1 min-h-0">
          <Panel tab={activeSidebarTab} />
        </div>

        {/* Resize handle */}
        <div
          className="absolute top-0 right-0 h-full w-1 cursor-col-resize hover:bg-primary/50 transition-colors"
          onMouseDown={(e) => {
            e.preventDefault()
            const startX = e.clientX
            const startWidth = sidebarWidth
            const handleMove = (moveEvent: MouseEvent) => {
              setSidebarWidth(Math.max(220, Math.min(600, startWidth + moveEvent.clientX - startX)))
            }
            const handleUp = () => {
              document.removeEventListener("mousemove", handleMove)
              document.removeEventListener("mouseup", handleUp)
            }
            document.addEventListener("mousemove", handleMove)
            document.addEventListener("mouseup", handleUp)
          }}
        />
      </div>
    </TooltipProvider>
  )
}
