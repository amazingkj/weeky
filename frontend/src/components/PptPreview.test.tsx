import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import PptPreview from './PptPreview';
import { Report } from '../types';

describe('PptPreview Component', () => {
  const emptyReport: Report = {
    team_name: '',
    author_name: '',
    report_date: '',
    this_week: [],
    next_week: [],
    issues: '',
    notes: '',
    next_issues: '',
    next_notes: '',
    template_id: 0,
  };

  const filledReport: Report = {
    team_name: '개발팀',
    author_name: '홍길동',
    report_date: '2024-01-15',
    this_week: [
      { title: 'API 개발', due_date: '2024-01-15', progress: 100 },
      { title: '테스트 작성', due_date: '2024-01-16', progress: 80 },
    ],
    next_week: [
      { title: '배포 준비', due_date: '2024-01-22', progress: 0 },
    ],
    issues: '특이사항 없음',
    notes: '',
    next_issues: '',
    next_notes: '',
    template_id: 0,
  };

  it('shows placeholder message for empty report', () => {
    render(<PptPreview report={emptyReport} />);

    // Shows message when no tasks
    expect(screen.getByText('금주실적을 입력하면 미리보기가 표시됩니다.')).toBeInTheDocument();
  });

  it('renders header with report info', () => {
    render(<PptPreview report={filledReport} />);

    // Header shows project name, date, author in table format (multiple slides)
    expect(screen.getAllByText('프로젝트명').length).toBeGreaterThan(0);
    expect(screen.getAllByText('개발팀').length).toBeGreaterThan(0);
    expect(screen.getAllByText('홍길동').length).toBeGreaterThan(0);
    expect(screen.getAllByText('보고일자').length).toBeGreaterThan(0);
  });

  it('renders this week tasks', () => {
    render(<PptPreview report={filledReport} />);

    expect(screen.getByText('금주실적')).toBeInTheDocument();
    expect(screen.getByText(/1\. API 개발/)).toBeInTheDocument();
    expect(screen.getByText(/2\. 테스트 작성/)).toBeInTheDocument();
  });

  it('renders next week tasks', () => {
    render(<PptPreview report={filledReport} />);

    expect(screen.getByText('차주계획')).toBeInTheDocument();
    expect(screen.getByText(/1\. 배포 준비/)).toBeInTheDocument();
  });

  it('renders issues section', () => {
    render(<PptPreview report={filledReport} />);

    // Template uses separate rows for "이슈/위험 사항" and "특이 사항"
    expect(screen.getAllByText('이슈/위험 사항').length).toBeGreaterThan(0);
    expect(screen.getAllByText('특이 사항').length).toBeGreaterThan(0);
    expect(screen.getByText('특이사항 없음')).toBeInTheDocument();
  });

  it('does not render issues slide when empty', () => {
    const reportWithoutIssues = { ...filledReport, issues: '' };
    render(<PptPreview report={reportWithoutIssues} />);

    // Only shows 2 slides (this week, next week) - no issues slide
    const issuesSections = screen.queryAllByText('이슈/특이사항');
    expect(issuesSections.length).toBe(0);
  });

  it('does not render this week slide when empty and no issues', () => {
    // Slide 1 renders if this_week has items OR issues exist
    const reportWithoutThisWeekAndIssues = { ...filledReport, this_week: [], issues: '' };
    render(<PptPreview report={reportWithoutThisWeekAndIssues} />);

    expect(screen.queryByText('금주실적')).not.toBeInTheDocument();
  });

  it('shows slide numbers with titles', () => {
    render(<PptPreview report={filledReport} />);

    expect(screen.getByText(/슬라이드 1.*금주실적/)).toBeInTheDocument();
    expect(screen.getByText(/슬라이드 2.*차주계획/)).toBeInTheDocument();
  });

  it('shows overflow indicator for many tasks', () => {
    const reportWithManyTasks: Report = {
      ...filledReport,
      this_week: Array(10).fill(null).map((_, i) => ({
        title: `Task ${i + 1}`,
        due_date: '2024-01-15',
        progress: 50,
      })),
    };

    render(<PptPreview report={reportWithManyTasks} />);

    // Should show indicator that there are more items (+5개 더)
    expect(screen.getByText(/\+\d+개 더/)).toBeInTheDocument();
  });
});
