import { cn } from 'lib/utils'

interface AvatarProps {
  name?: string
  role: 'user' | 'ai'
  isSpeaking?: boolean
  className?: string
}

interface AvatarFallbackProps {
  children: React.ReactNode
  className?: string
}

export function Avatar({ name, role, isSpeaking = false, className }: AvatarProps) {
  const getInitials = (name: string) => {
    if (!name || typeof name !== 'string') {
      return role === 'user' ? 'U' : 'AI'
    }
    return name
      .split(' ')
      .map(word => word[0])
      .join('')
      .toUpperCase()
      .slice(0, 2)
  }

  const getAvatarColor = (role: 'user' | 'ai') => {
    return role === 'user' 
      ? 'bg-primary text-primary-foreground' 
      : 'bg-secondary text-secondary-foreground'
  }

  return (
    <div className={cn(
      "relative flex items-center justify-center w-12 h-12 rounded-full border-2 transition-all duration-300",
      getAvatarColor(role),
      isSpeaking && "ring-4 ring-primary/30 scale-110",
      className
    )}>
      <AvatarFallback className="text-sm font-semibold">
        {getInitials(name || '')}
      </AvatarFallback>
      
      {/* Speaking indicator */}
      {isSpeaking && (
        <div className="absolute -bottom-1 -right-1 w-4 h-4 bg-green-500 rounded-full flex items-center justify-center">
          <div className="w-2 h-2 bg-white rounded-full animate-pulse" />
        </div>
      )}
    </div>
  )
}

export function AvatarFallback({ children, className }: AvatarFallbackProps) {
  return (
    <span className={cn("flex items-center justify-center", className)}>
      {children}
    </span>
  )
}