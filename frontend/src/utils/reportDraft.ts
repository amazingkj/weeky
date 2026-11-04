import { Report } from '../types';

// 저장 안 된 보고서 초안 — 브라우저가 꺼지거나 새로고침돼도 복원할 수 있도록 localStorage에 보관.
// 사용자·보고일자별로 하나. localStorage를 못 쓰는 환경(사생활 모드 등)에서는 조용히 무시한다.
const PREFIX = 'jugan_draft_';

export interface ReportDraft {
  report: Report;
  savedAt: string; // ISO
  // 초안을 쓰기 시작할 때의 서버 내용(JSON) — 그 뒤 다른 곳에서 저장돼 서버가 바뀌었는지 판단용
  baseKey?: string;
}

const key = (userId: number, reportDate: string) => `${PREFIX}${userId}_${reportDate}`;

export function saveDraft(userId: number, report: Report, baseKey?: string): void {
  try {
    const draft: ReportDraft = { report, savedAt: new Date().toISOString(), baseKey };
    localStorage.setItem(key(userId, report.report_date), JSON.stringify(draft));
  } catch {
    // 저장 공간 부족 등 — 초안은 보조 수단이라 무시
  }
}

export function loadDraft(userId: number, reportDate: string): ReportDraft | null {
  try {
    const raw = localStorage.getItem(key(userId, reportDate));
    if (!raw) return null;
    const draft = JSON.parse(raw) as ReportDraft;
    return draft?.report && draft.savedAt ? draft : null;
  } catch {
    return null;
  }
}

export function clearDraft(userId: number, reportDate: string): void {
  try {
    localStorage.removeItem(key(userId, reportDate));
  } catch {
    // ignore
  }
}

// 직접 로그아웃할 때 — 공용 PC에 업무 내용이 남지 않도록 해당 사용자의 초안을 모두 삭제
export function clearAllDrafts(userId: number): void {
  try {
    const prefix = `${PREFIX}${userId}_`;
    Object.keys(localStorage).filter((k) => k.startsWith(prefix)).forEach((k) => localStorage.removeItem(k));
  } catch {
    // ignore
  }
}
