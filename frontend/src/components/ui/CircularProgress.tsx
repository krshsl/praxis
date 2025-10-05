import * as Progress from '@radix-ui/react-progress'
import { cn } from 'lib/utils'

interface ProgressProps {
  value: number // 0-100
  className?: string
  label?: string
  showPercentage?: boolean
  size?: 'sm' | 'md' | 'lg'
}

export function ProgressBar({
  value,
  className = '',
  label,
  showPercentage = true,
  size = 'md'
}: ProgressProps) {
  const sizeClasses = {
    sm: 'h-2',
    md: 'h-3',
    lg: 'h-4'
  }

  const getColor = (value: number) => {
  if (value >= 80) return 'bg-primary'
  if (value >= 60) return 'bg-orange-9'
  if (value >= 40) return 'bg-orange-7'
  return 'bg-destructive'
  }

  return (
    <div className={`w-full ${className}`}>
      {(label || showPercentage) && (
        <div className="flex justify-between items-center mb-2">
          {label && <span className="text-sm font-medium text-foreground">{label}</span>}
          {showPercentage && (
            <span className="text-sm text-muted-foreground">
              {Math.round(value)}%
            </span>
          )}
        </div>
      )}
      <Progress.Root
        className={`relative overflow-hidden rounded-full bg-secondary ${sizeClasses[size]}`}
        value={value}
      >
        <Progress.Indicator
          className={`h-full w-full flex-1 transition-all duration-300 ease-in-out ${getColor(value)}`}
          style={{ transform: `translateX(-${100 - value}%)` }}
        />
      </Progress.Root>
    </div>
  )
}

// Keep the circular progress for backward compatibility
interface CircularProgressProps {
  value: number // 0-100
  size?: number
  strokeWidth?: number
  className?: string
  label?: string
  showPercentage?: boolean
}

export function CircularProgress({
  value,
  size = 120,
  strokeWidth = 8,
  className = '',
  label,
  showPercentage = true
}: CircularProgressProps) {
  const radius = (size - strokeWidth) / 2
  const circumference = radius * 2 * Math.PI
  const strokeDasharray = circumference
  const strokeDashoffset = circumference - (value / 100) * circumference

  const getColor = (value: number) => {
    if (value >= 80) return '#10b981' // green
    if (value >= 60) return '#f59e0b' // yellow
    if (value >= 40) return '#f97316' // orange
    return '#ef4444' // red
  }

  return (
    <div className={cn('flex flex-col items-center', className)}>
      <div className="relative" style={{ width: size, height: size }}>
        <svg
          width={size}
          height={size}
          className="transform -rotate-90"
        >
          {/* Background circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke="currentColor"
            strokeWidth={strokeWidth}
            fill="none"
            className="text-muted"
          />
          {/* Progress circle */}
          <circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={getColor(value)}
            strokeWidth={strokeWidth}
            fill="none"
            strokeDasharray={strokeDasharray}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
            className="transition-all duration-300 ease-in-out"
          />
        </svg>
        {/* Center content */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          {showPercentage && (
            <span className="text-2xl font-bold" style={{ color: getColor(value) }}>
              {Math.round(value)}%
            </span>
          )}
          {label && (
            <span className="text-xs text-muted-foreground mt-1 text-center">
              {label}
            </span>
          )}
        </div>
      </div>
    </div>
  )
}
