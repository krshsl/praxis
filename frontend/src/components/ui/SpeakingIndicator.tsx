interface SpeakingIndicatorProps {
  isSpeaking: boolean
  className?: string
}

export function SpeakingIndicator({ isSpeaking, className }: SpeakingIndicatorProps) {
  if (!isSpeaking) return null

  return (
    <div className={`flex items-center space-x-1 ${className}`}>
      <div className="flex space-x-1">
        <div className="w-2 h-2 bg-primary rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
        <div className="w-2 h-2 bg-primary rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
        <div className="w-2 h-2 bg-primary rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
      </div>
      <span className="text-sm text-muted-foreground">Speaking...</span>
    </div>
  )
}
