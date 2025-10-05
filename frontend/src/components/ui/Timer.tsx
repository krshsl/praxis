import { useState, useEffect } from 'react'
import { cn } from 'lib/utils'
import Countdown from 'react-countdown'

interface TimerProps {
  duration: number // in milliseconds
  onComplete: () => void
  onTick?: (remaining: number) => void
  className?: string
  showProgress?: boolean
}

export function Timer({ 
  duration, 
  onComplete, 
  onTick, 
  className,
  showProgress = true
}: TimerProps) {
  const [targetTime, setTargetTime] = useState<number>(Date.now() + duration)
  const [isActive, setIsActive] = useState(true)

  // Reset timer when duration changes
  useEffect(() => {
    setTargetTime(Date.now() + duration)
    setIsActive(true)
  }, [duration])

  const handleTick = ({ total }: { total: number }) => {
    onTick?.(total)
  }

  const handleComplete = () => {
    setIsActive(false)
    onComplete()
  }

  const renderer = ({ total, completed }: { 
    total: number
    completed: boolean 
  }) => {
    if (completed) {
      return <div className="text-sm font-medium text-destructive">Time's up!</div>
    }

    const progress = (duration - total) / duration
    const displaySeconds = Math.ceil(total / 1000)

    return (
      <div className={cn("flex flex-col items-center space-y-2", className)}>
        {showProgress && (
          <div className="w-full bg-muted rounded-full h-2">
            <div 
              className="bg-primary h-2 rounded-full transition-all duration-100"
              style={{ width: `${progress * 100}%` }}
            />
          </div>
        )}
        <div className="text-sm font-medium">
          {displaySeconds}s remaining
        </div>
      </div>
    )
  }

  if (!isActive) {
    return (
      <div className={cn("flex flex-col items-center space-y-2", className)}>
        <div className="text-sm font-medium text-muted-foreground">
          Timer stopped
        </div>
      </div>
    )
  }

  return (
    <Countdown
      date={targetTime}
      onTick={handleTick}
      onComplete={handleComplete}
      renderer={renderer}
      intervalDelay={100}
    />
  )
}

// Export timer controls for external use
export type TimerControls = {
  start: () => void
  stop: () => void
  reset: () => void
}