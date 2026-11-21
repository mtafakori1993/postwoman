import { cn } from "@/lib/utils"

interface LogoProps {
  className?: string
  size?: "sm" | "md" | "lg"
  showText?: boolean
}

export function Logo({ className, size = "md", showText = true }: LogoProps) {
  const sizeClasses = {
    sm: "w-6 h-6",
    md: "w-8 h-8",
    lg: "w-10 h-10",
  }

  const textSizeClasses = {
    sm: "text-sm",
    md: "text-lg",
    lg: "text-xl",
  }

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <img
        src="/appicon.png"
        alt="Postwoman"
        className={cn(sizeClasses[size], "rounded-lg")}
      />
      {showText && (
        <span className={cn("font-semibold", textSizeClasses[size], "truncate")}>
          Postwoman
        </span>
      )}
    </div>
  )
}