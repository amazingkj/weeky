import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import MyHistoryPanel from './MyHistoryPanel';
import * as api from '../services/api';

vi.mock('../services/api', () => ({
  getMySubmissions: vi.fn(),
  getMySiteReports: vi.fn(),
  getReport: vi.fn(),
  getReports: vi.fn(),
  getSiteProjects: vi.fn(),
}));

const mocked = api as unknown as Record<string, ReturnType<typeof vi.fn>>;

describe('MyHistoryPanel 고객사 필터', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocked.getMySubmissions.mockResolvedValue([
      { id: 1, report_id: 10, team_id: 1, user_id: 1, status: 'submitted', created_at: '', report_date: '2026-09-25' },
      { id: 2, report_id: 11, team_id: 1, user_id: 1, status: 'submitted', created_at: '', report_date: '2026-09-18' },
    ]);
    mocked.getReports.mockResolvedValue([
      { id: 10, report_date: '2026-09-25', this_week: [{ title: 'CruzAPIM', client: '삼성카드', due_date: '', progress: 50 }], next_week: [] },
      { id: 11, report_date: '2026-09-18', this_week: [{ title: 'Mesh', client: '흥국화재', due_date: '', progress: 100 }], next_week: [] },
    ]);
    mocked.getMySiteReports.mockResolvedValue([
      { id: 5, team_id: 1, site_project_id: 7, author_user_id: 1, author_names: [], project_name: '도공 사이트', report_date: '2026-09-25', this_week: [], next_week: [], notes: '', created_at: '', updated_at: '' },
    ]);
    mocked.getSiteProjects.mockResolvedValue([
      { id: 7, team_id: 1, project_name: '도공 사이트', client_name: '도로교통공단', is_active: true, sort_order: 0, created_at: '' },
    ]);
  });

  it('본사 task의 client와 사이트 프로젝트 고객사로 필터링한다', async () => {
    render(<MyHistoryPanel teamId={1} />);
    const select = await screen.findByLabelText('고객사 필터');

    expect(screen.getByText('2026-09-18')).toBeInTheDocument();
    expect(screen.getByText('전체 3')).toBeInTheDocument();

    fireEvent.change(select, { target: { value: '흥국화재' } });
    expect(screen.getByText('전체 1')).toBeInTheDocument();
    expect(screen.getByText('2026-09-18')).toBeInTheDocument();
    expect(screen.queryByText('2026-09-25')).not.toBeInTheDocument();

    fireEvent.change(select, { target: { value: '도로교통공단' } });
    expect(screen.getByText('사이트 1')).toBeInTheDocument();
    expect(screen.getByText('도공 사이트')).toBeInTheDocument();
    expect(screen.queryByText('2026-09-18')).not.toBeInTheDocument();
  });
});
