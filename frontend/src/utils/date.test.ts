import { describe, it, expect } from 'vitest';
import { toLocalYMD } from './date';

describe('toLocalYMD', () => {
  it('자정 직후(로컬)에도 로컬 날짜를 반환한다', () => {
    // KST 00:30은 UTC로 전날 15:30 — toISOString이면 하루 밀림
    expect(toLocalYMD(new Date(2026, 8, 28, 0, 30))).toBe('2026-09-28');
  });

  it('한 자리 월/일을 0으로 채운다', () => {
    expect(toLocalYMD(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05');
  });
});
