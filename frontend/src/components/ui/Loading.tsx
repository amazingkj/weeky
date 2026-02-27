interface LoadingProps {
  size?: 'sm' | 'md' | 'lg';
  text?: string;
  className?: string;
}

const sizeClasses = {
  sm: { spinner: 'w-5 h-5', text: 'text-xs' },
  md: { spinner: 'w-8 h-8', text: 'text-sm' },
  lg: { spinner: 'w-12 h-12', text: 'text-base' },
};

export default function Loading({ size = 'md', text, className = '' }: LoadingProps) {
  const { spinner, text: textSize } = sizeClasses[size];

  return (
    <div className={`flex flex-col items-center justify-center gap-4 ${className}`}>
      {/* Animate wrapper div instead of SVG for hardware acceleration (rendering-animate-svg-wrapper) */}
      <div className="relative">
        {/* Outer glow */}
        <div className={`absolute inset-0 ${spinner} rounded-full bg-blue-500/20 blur-md animate-pulse`} />

        {/* Spinner */}
        <div className={`relative ${spinner} animate-spin`}>
          <svg className="w-full h-full" viewBox="0 0 24 24" fill="none">
            {/* Background circle */}
            <circle
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="3"
              className="text-neutral-200"
            />
            {/* Animated arc */}
            <circle
              cx="12"
              cy="12"
              r="10"
              stroke="url(#gradient)"
              strokeWidth="3"
              strokeLinecap="round"
              strokeDasharray="60 40"
            />
            <defs>
              <linearGradient id="gradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stopColor="#3B82F6" />
                <stop offset="100%" stopColor="#8B5CF6" />
              </linearGradient>
            </defs>
          </svg>
        </div>
      </div>

      {text && (
        <p className={`${textSize} text-neutral-500 font-medium animate-pulse`}>
          {text}
        </p>
      )}
    </div>
  );
}
