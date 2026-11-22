import React from "react"
import { Button } from "@/components/ui/button"
import { AlertTriangle, RotateCcw } from "lucide-react"

interface ErrorBoundaryProps {
  children: React.ReactNode
  title?: string
  description?: string
  onReset?: () => void
}

interface ErrorBoundaryState {
  error: Error | null
}

/**
 * Catches render-time errors so an unexpected payload (for example an unusually
 * large or malformed discovery result) can never blank the whole application.
 */
export class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("Postwoman UI error:", error, info)
  }

  private reset = () => {
    this.setState({ error: null })
    this.props.onReset?.()
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex h-full min-h-[240px] flex-col items-center justify-center gap-3 p-8 text-center">
          <AlertTriangle className="h-8 w-8 text-amber-500" />
          <p className="text-sm font-semibold">{this.props.title || "Something went wrong in this view"}</p>
          <p className="max-w-md pw-text-2xs text-muted-foreground">
            {this.props.description ||
              "The interface hit an unexpected error while rendering this data. Your saved work is unaffected."}
          </p>
          <p className="max-w-lg break-words rounded bg-muted/50 px-3 py-1.5 font-mono pw-text-2xs text-muted-foreground">
            {this.state.error.message}
          </p>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" className="gap-1.5" onClick={this.reset}>
              <RotateCcw className="h-3.5 w-3.5" /> Try again
            </Button>
            <Button variant="ghost" size="sm" onClick={() => window.location.reload()}>
              Reload application
            </Button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}