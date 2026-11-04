import { describe, it, expect, beforeEach } from 'vitest';
import { saveDraft, loadDraft, clearDraft, clearAllDrafts } from './reportDraft';
import { Report } from '../types';

const report = (date: string, notes = ''): Report => ({
  team_name: '플랫폼팀', author_name: '홍길동', report_date: date,
  this_week: [], next_week: [], issues: '', notes, next_issues: '', next_notes: '', template_id: 0,
});

describe('reportDraft', () => {
  beforeEach(() => localStorage.clear());

  it('사용자·보고일자별로 저장하고 불러온다', () => {
    saveDraft(1, report('2026-10-06', 'A'));
    saveDraft(2, report('2026-10-06', 'B'));

    expect(loadDraft(1, '2026-10-06')?.report.notes).toBe('A');
    expect(loadDraft(2, '2026-10-06')?.report.notes).toBe('B');
    expect(loadDraft(1, '2026-10-13')).toBeNull();
    expect(loadDraft(1, '2026-10-06')?.savedAt).toMatch(/^\d{4}-\d{2}-\d{2}T/);
  });

  it('clearDraft는 해당 날짜만, clearAllDrafts는 해당 사용자 것만 지운다', () => {
    saveDraft(1, report('2026-10-06'));
    saveDraft(1, report('2026-10-13'));
    saveDraft(2, report('2026-10-06'));
    localStorage.setItem('jugan_author_name', '홍길동');

    clearDraft(1, '2026-10-06');
    expect(loadDraft(1, '2026-10-06')).toBeNull();
    expect(loadDraft(1, '2026-10-13')).not.toBeNull();

    clearAllDrafts(1);
    expect(loadDraft(1, '2026-10-13')).toBeNull();
    expect(loadDraft(2, '2026-10-06')).not.toBeNull();
    expect(localStorage.getItem('jugan_author_name')).toBe('홍길동');
  });

  it('깨진 값은 무시한다', () => {
    localStorage.setItem('jugan_draft_1_2026-10-06', '{not json');
    expect(loadDraft(1, '2026-10-06')).toBeNull();
  });
});
