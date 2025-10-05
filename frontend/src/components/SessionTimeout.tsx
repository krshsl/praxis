import { useState, useEffect } from 'react'

interface SessionTimeoutProps {
  sessionId: string
  onTimeout: () => void
  timeoutMinutes?: number
}

export function SessionTimeout({ sessionId: _sessionId, onTimeout, timeoutMinutes = 5 }: SessionTimeoutProps) {
  const [timeRemaining, setTimeRemaining] = useState<number>(timeoutMinutes * 60) // seconds
  const [isWarning, setIsWarning] = useState(false)
  const [lastActivity, setLastActivity] = useState<number>(Date.now())

  useEffect(() => {
    const interval = setInterval(() => {
      const now = Date.now()
      const timeSinceActivity = (now - lastActivity) / 1000 // seconds
      const remaining = Math.max(0, (timeoutMinutes * 60) - timeSinceActivity)
      
      setTimeRemaining(remaining)
      
      // Show warning when 1 minute remaining
      if (remaining <= 60 && remaining > 0) {
        setIsWarning(true)
      }
      
      // Trigger timeout when time is up
      if (remaining <= 0) {
        onTimeout()
      }
    }, 1000)

    return () => clearInterval(interval)
  }, [lastActivity, timeoutMinutes, onTimeout])

  // Reset activity timer on user interaction
  useEffect(() => {
    const handleActivity = () => {
      setLastActivity(Date.now())
      setIsWarning(false)
    }

    // Listen for various user activities
    const events = ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart', 'click']
    
    events.forEach(event => {
      document.addEventListener(event, handleActivity, true)
    })

    return () => {
      events.forEach(event => {
        document.removeEventListener(event, handleActivity, true)
      })
    }
  }, [])

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins}:${secs.toString().padStart(2, '0')}`
  }

  if (timeRemaining <= 0) {
    return null // Session has timed out
  }

  if (timeRemaining <= 60) {
    return (
  <div className="fixed top-4 right-4 bg-destructive text-destructive-foreground px-4 py-2 rounded-lg shadow-lg z-50">
        <div className="flex items-center space-x-2">
          <div className="w-2 h-2 bg-foreground rounded-full animate-pulse"></div>
          <span className="font-semibold">
            Session will timeout in {formatTime(timeRemaining)}
          </span>
        </div>
        <div className="text-sm mt-1">
          Move your mouse or type to keep the session active
        </div>
      </div>
    )
  }

  if (isWarning) {
    return (
  <div className="fixed top-4 right-4 bg-accent text-accent-foreground px-4 py-2 rounded-lg shadow-lg z-50">
        <div className="flex items-center space-x-2">
          <div className="w-2 h-2 bg-foreground rounded-full animate-pulse"></div>
          <span className="font-semibold">
            Session timeout warning: {formatTime(timeRemaining)} remaining
          </span>
        </div>
      </div>
    )
  }

  return null
}
