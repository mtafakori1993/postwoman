import React, { useMemo } from "react"
import CodeMirror from "@uiw/react-codemirror"
import { json } from "@codemirror/lang-json"
import { javascript } from "@codemirror/lang-javascript"
import { xml } from "@codemirror/lang-xml"
import { html } from "@codemirror/lang-html"
import { EditorView } from "@codemirror/view"
import type { Extension } from "@codemirror/state"
import { githubDark, githubLight } from "@uiw/codemirror-theme-github"
import { cn } from "@/lib/utils"
import { useIsDarkMode } from "@/lib/use-theme"
import { useWorkspaceStore } from "@/stores/workspace"
import type { CodeLanguage } from "@/lib/format"

interface SyntaxEditorProps {
  value: string
  onChange?: (value: string) => void
  language?: CodeLanguage
  readOnly?: boolean
  placeholder?: string
  lineNumbers?: boolean
  wrap?: boolean
  minHeight?: string
  maxHeight?: string
  height?: string
  className?: string
  ariaLabel?: string
  autoFocus?: boolean
}

function languageExtension(language: CodeLanguage) {
  switch (language) {
    case "json":
      return [json()]
    case "javascript":
      return [javascript()]
    case "xml":
      return [xml()]
    case "html":
      return [html()]
    default:
      return []
  }
}

/**
 * Thin CodeMirror wrapper used by both the request body editor and the
 * response viewer. Syntax highlighting is fully local (no CDN), which keeps
 * the desktop app usable offline.
 */
export function SyntaxEditor({
  value,
  onChange,
  language = "text",
  readOnly = false,
  placeholder,
  lineNumbers = true,
  wrap = true,
  minHeight,
  maxHeight,
  height,
  className,
  ariaLabel,
  autoFocus,
}: SyntaxEditorProps) {
  const dark = useIsDarkMode()
  const editorFontSize = useWorkspaceStore((s) => s.settings.editorFontSize || 13)
  const tabSize = useWorkspaceStore((s) => s.settings.editorTabSize || 2)
  const fontFamily = useWorkspaceStore((s) => s.settings.editorFontFamily || "mono")

  const extensions = useMemo(() => {
    const ext: Extension[] = [...languageExtension(language)]
    if (wrap) ext.push(EditorView.lineWrapping)
    ext.push(
      EditorView.theme({
        "&": { fontSize: `${editorFontSize}px`, height: "100%" },
        ".cm-scroller": { overflow: "auto" },
        ".cm-content": { fontFamily: "var(--pw-editor-font, 'Source Code Pro', ui-monospace, monospace)" },
        ".cm-gutters": { borderRight: "1px solid hsl(var(--border))" },
        "&.cm-focused": { outline: "none" },
      }),
    )
    return ext
  }, [language, wrap, editorFontSize, fontFamily])

  return (
    <div
      className={cn("pw-syntax-editor h-full min-h-0 overflow-hidden", className)}
      data-language={language}
      aria-label={ariaLabel}
    >
      <CodeMirror
        value={value}
        onChange={(next) => onChange?.(next)}
        theme={dark ? githubDark : githubLight}
        extensions={extensions}
        editable={!readOnly}
        readOnly={readOnly}
        placeholder={placeholder}
        autoFocus={autoFocus}
        height={height}
        minHeight={minHeight}
        maxHeight={maxHeight}
        basicSetup={{
          lineNumbers,
          foldGutter: !readOnly,
          highlightActiveLine: !readOnly,
          highlightActiveLineGutter: !readOnly,
          bracketMatching: true,
          closeBrackets: !readOnly,
          autocompletion: !readOnly,
          searchKeymap: true,
          tabSize,
        }}
        style={{ height: "100%" }}
      />
    </div>
  )
}
