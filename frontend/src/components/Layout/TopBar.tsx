import React from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Keyboard, Link, Menu, Search, Moon, Sun, Plus, Play, Sparkles, Terminal, Upload, Compass } from "lucide-react"
import { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from "@/components/ui/tooltip"
import { useWorkspaceStore } from "@/stores/workspace"
import { Logo } from "@/components/ui/logo"

interface TopBarProps {
  onNewTab: () => void
  onSearch: () => void
  onOpenRunner: () => void
  onImport: () => void
  onImportUrl: () => void
  onImportCurl: () => void
  onOpenShortcuts: () => void
  onOpenAI: () => void
  onOpenDiscovery: () => void
}

export function TopBar({ onNewTab, onSearch, onOpenRunner, onImport, onImportUrl, onImportCurl, onOpenShortcuts, onOpenAI, onOpenDiscovery }: TopBarProps) {
  const {
    settings,
    updateSettings,
    environments,
    activeEnvironmentId,
    setActiveEnvironment,
    workspaces,
    workspace,
    setActiveWorkspace,
    toggleSidebar,
  } = useWorkspaceStore()

  return (
    <TooltipProvider>
      <div className="flex items-center gap-2 h-12 px-3 bg-background border-b border-border shrink-0">
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" className="h-9 w-9" onClick={toggleSidebar} aria-label="Toggle sidebar">
              <Menu className="h-5 w-5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">Toggle Sidebar (Ctrl+B)</TooltipContent>
        </Tooltip>

        <div className="items-center gap-2 hidden sm:flex">
          <Logo size="md" />
        </div>

        <div className="flex-1 min-w-0 max-w-md hidden sm:block">
          <button
            className="relative w-full text-left"
            onClick={onSearch}
            aria-label="Open search"
          >
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              type="search"
              placeholder="Search (Ctrl+K)..."
              className="pl-10 h-9 w-full pointer-events-none"
              readOnly
              tabIndex={-1}
            />
          </button>
        </div>

        <Button variant="ghost" size="icon" className="h-9 w-9 sm:hidden" onClick={onSearch} aria-label="Search">
          <Search className="h-5 w-5" />
        </Button>

        {/* Workspace selector */}
        <div className="hidden lg:block shrink-0">
          <Select value={workspace?.id ?? ""} onValueChange={setActiveWorkspace}>
            <SelectTrigger className="w-[170px] h-9">
              <SelectValue placeholder="Workspace" />
            </SelectTrigger>
            <SelectContent>
              {workspaces.map((ws) => (
                <SelectItem key={ws.id} value={ws.id}>
                  {ws.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Environment selector */}
        <div className="hidden md:block shrink-0">
          <Tooltip>
            <TooltipTrigger asChild>
              <Select
                value={activeEnvironmentId ?? "no-env"}
                onValueChange={(v) => setActiveEnvironment(v === "no-env" ? null : v)}
              >
                <SelectTrigger className="w-[170px] h-9">
                  <SelectValue placeholder="No Environment" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="no-env">No Environment</SelectItem>
                  {environments.map((env) => (
                    <SelectItem key={env.id} value={env.id}>
                      {env.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </TooltipTrigger>
            <TooltipContent side="bottom">Active environment</TooltipContent>
          </Tooltip>
        </div>

        <div className="flex items-center gap-1 ml-auto shrink-0">
          <DropdownMenu>
            <Tooltip>
              <TooltipTrigger asChild>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-9 w-9" aria-label="Import">
                    <Upload className="h-5 w-5" />
                  </Button>
                </DropdownMenuTrigger>
              </TooltipTrigger>
              <TooltipContent side="bottom">Import</TooltipContent>
            </Tooltip>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onImport}>
                <Upload className="mr-2 h-3.5 w-3.5" /> From file (Postman / OpenAPI / HAR)
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onImportUrl}>
                <Link className="mr-2 h-3.5 w-3.5" /> From URL (OpenAPI / Swagger)
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onImportCurl}>
                <Terminal className="mr-2 h-3.5 w-3.5" /> Paste cURL command
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onOpenRunner} aria-label="Run collection">
                <Play className="h-5 w-5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Collection runner</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onOpenDiscovery} aria-label="API discovery agent">
                <Compass className="h-5 w-5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">API discovery agent</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onOpenAI} aria-label="AI assistant">
                <Sparkles className="h-5 w-5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">AI assistant</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onOpenShortcuts} aria-label="Keyboard shortcuts">
                <Keyboard className="h-5 w-5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Keyboard shortcuts (Ctrl+/)</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" onClick={onNewTab} aria-label="New tab">
                <Plus className="h-5 w-5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">New Tab (Ctrl+T)</TooltipContent>
          </Tooltip>

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9"
                onClick={() => updateSettings({ theme: settings.theme === "dark" ? "light" : "dark" })}
                aria-label="Toggle theme"
              >
                {settings.theme === "dark" ? <Sun className="h-5 w-5" /> : <Moon className="h-5 w-5" />}
              </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">Toggle theme</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </TooltipProvider>
  )
}
