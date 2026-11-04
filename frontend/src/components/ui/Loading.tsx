interface LoadingProps {
  size?: 'sm' | 'md' | 'lg';
  text?: string;
  className?: string;
}

const sizeClasses = {
  sm: { spinner: 'w-4 h-4', text: 'text-xs' },
  md: { spinner: 'w-6 h-6', text: 'text-xs' },
  lg: { spinner: 'w-8 h-8', text: 'text-sm' },
};

// 화면·섹션 전체를 불러올 때 쓰는 차분한 스피너. 목록은 SkeletonList를 사용
export default function Loading({ size = 'md', text, className = '' }: LoadingProps) {
  const { spinner, text: textSize } = sizeClasses[size];

  return (
    <div role="status" className={`flex flex-col items-center justify-center gap-3 ${className}`}>
      {/* Animate wrapper div instead of SVG for hardware acceleration (rendering-animate-svg-wrapper) */}
      <div className={`${spinner} animate-spin text-ink-600`}>
        <svg className="w-full h-full" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" className="opacity-15" />
          <path d="M22 12a10 10 0 0 0-10-10" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
        </svg>
      </div>
      {text ? <p className={`${textSize} text-neutral-500`}>{text}</p> : <span className="sr-only">불러오는 중</span>}
    </div>
  );
}
