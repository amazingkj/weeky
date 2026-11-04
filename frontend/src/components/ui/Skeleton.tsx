// 목록을 불러오는 동안 실제 행 모양 자리만 잡아 두는 표시 — 다 불러왔을 때 레이아웃이 덜 튐
export function SkeletonList({ rows = 4, label = '불러오는 중' }: { rows?: number; label?: string }) {
  return (
    <div role="status" aria-label={label} className="space-y-2 py-1">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="flex items-center gap-3 px-3 py-2.5 rounded-lg border border-neutral-100 bg-white">
          <div className="h-3 w-20 rounded bg-neutral-100 animate-pulse" />
          {/* 행마다 길이를 조금씩 달리해 실제 목록처럼 보이게 */}
          <div className="h-3 flex-1 rounded bg-neutral-100 animate-pulse" style={{ maxWidth: `${70 - (i % 3) * 15}%` }} />
        </div>
      ))}
    </div>
  );
}
