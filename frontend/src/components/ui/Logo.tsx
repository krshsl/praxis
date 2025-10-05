
interface LogoProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export function Logo({ className = '', size = 'md' }: LogoProps) {
  const sizeClasses = {
    sm: 'w-8 h-8',
    md: 'w-10 h-10', 
    lg: 'w-16 h-16'
  }

  return (
    <svg 
      xmlns="http://www.w3.org/2000/svg" 
      viewBox="0 0 1200 600" 
      className={`${sizeClasses[size]} ${className}`}
      role="img" 
      aria-label="P logo with diagonal orange-to-green gradient"
    >
      <defs>
        {/* diagonal gradient spanning stem + circle evenly */}
        <linearGradient id="diagGrad" x1="360" y1="160" x2="760" y2="440" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#ff7b00"/>
          <stop offset="25%" stopColor="#ff9f1c"/>
          <stop offset="55%" stopColor="#ffd166"/>
          <stop offset="100%" stopColor="#06d6a0"/>
        </linearGradient>

        <mask id="cutout">
          <rect x="0" y="0" width="1200" height="600" fill="white"/>
          <circle cx="600" cy="300" r="120" fill="black"/>
        </mask>

        <filter id="softGlow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="6" result="blur"/>
          <feMerge>
            <feMergeNode in="blur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>

        <style>
          {`
            .fill { fill: url(#diagGrad); }
            .under-stroke { fill: none; stroke: rgba(255,255,255,0.92); stroke-linejoin: round; stroke-linecap: round; }
            .stroke-grad { fill: none; stroke-width: 4; stroke-linejoin: round; stroke-linecap: round; stroke: url(#diagGrad); }
            .counter-stroke { fill: none; stroke: rgba(255,255,255,0.9); stroke-width: 4; }
          `}
        </style>
      </defs>

      {/* masked group for stem + circle */}
      <g mask="url(#cutout)" filter="url(#softGlow)">
        <rect className="fill" x="400" y="300" width="80" height="200" rx="40"/>
        <circle className="fill" cx="600" cy="300" r="200"/>
      </g>

      {/* inner cutout counter */}
      <circle className="counter-stroke" cx="600" cy="300" r="120"/>

      {/* outline path */}
      <path d="M 480 460 A 200 200 0 1 0 400 300 L 400 460 A 40 40 0 0 0 480 460 Z"
            className="under-stroke" strokeWidth="6"/>
      <path d="M 480 460 A 200 200 0 1 0 400 300 L 400 460 A 40 40 0 0 0 480 460 Z"
            className="stroke-grad" strokeWidth="4"/>
    </svg>
  )
}
