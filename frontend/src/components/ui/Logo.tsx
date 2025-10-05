
interface LogoProps {
  className?: string
  size?: 'sm' | 'md' | 'lg'
}

export function Logo({ className = '', size = 'md' }: LogoProps) {
  const sizeClasses = {
    sm: 'w-6 h-6',
    md: 'w-8 h-8', 
    lg: 'w-12 h-12'
  }

  return (
    <svg 
  xmlns="http://www.w3.org/2000/svg" 
  viewBox="0 0 1200 600" 
  className={`${sizeClasses[size]} ${className}`}
>
  <defs>
    <mask id="cutout">
      {/* Everything in white is visible, black is transparent */}
      <rect x="0" y="0" width="1200" height="600" fill="white"/>
      <circle cx="600" cy="300" r="120" fill="black"/> {/* Inner circle cut-out */}
    </mask>
    <style>
      {`
        .fill { fill: #fefae0; }
        .counter-stroke { fill: none; stroke: #d4a373; stroke-width: 3; }
        .outline {
          fill: none;
          stroke: #d4a373;
          stroke-width: 3;
          stroke-linejoin: round;
          stroke-linecap: round;
        }
      `}
    </style>
  </defs>

  {/* Fill shapes (stem + outer circle) with mask applied to cut out inner circle */}
  <g mask="url(#cutout)">
    <rect className="fill" x="400" y="300" width="80" height="200" rx="40" />
    <circle className="fill" cx="600" cy="300" r="200"/>
  </g>

  {/* Inner circle stroke */}
  <circle className="counter-stroke" cx="600" cy="300" r="120"/>

  {/* ONE continuous outline around stem + outer circle */}
  <path className="outline" d="
    M 480 460
    A 200 200 0 1 0 400 300
    L 400 460
    A 40 40 0 0 0 480 460
    Z" />
</svg>

  )
}
