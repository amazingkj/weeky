import { describe, it, expect } from 'vitest';
import type { Task, Template, Report, SyncItem, SyncResult } from './index';

describe('Type Definitions', () => {
  describe('Task', () => {
    it('should have correct structure', () => {
      const task: Task = {
        title: 'Test Task',
        due_date: '2024-01-15',
        progress: 50,
      };

      expect(task.title).toBe('Test Task');
      expect(task.due_date).toBe('2024-01-15');
      expect(task.progress).toBe(50);
    });
  });

  describe('Template', () => {
    it('should have correct structure', () => {
      const template: Template = {
        id: 1,
        name: 'Test Template',
        style: '{"color": "blue"}',
        created_at: '2024-01-01T00:00:00Z',
      };

      expect(template.id).toBe(1);
      expect(template.name).toBe('Test Template');
    });
  });

  describe('Report', () => {
    it('should have correct structure with optional id', () => {
      const report: Report = {
        team_name: '개발팀',
        author_name: '홍길동',
        report_date: '2024-01-15',
        this_week: [],
        next_week: [],
        issues: '',
        template_id: 0,
      };

      expect(report.id).toBeUndefined();
      expect(report.team_name).toBe('개발팀');
    });

    it('should allow id when provided', () => {
      const report: Report = {
        id: 1,
        team_name: '개발팀',
        author_name: '홍길동',
        report_date: '2024-01-15',
        this_week: [],
        next_week: [],
        issues: '',
        template_id: 0,
      };

      expect(report.id).toBe(1);
    });
  });

  describe('SyncItem', () => {
    it('should have correct type values', () => {
      const commitItem: SyncItem = {
        title: 'Fix bug',
        date: '2024-01-15',
        url: 'https://github.com/...',
        type: 'commit',
      };

      const prItem: SyncItem = {
        title: 'Feature PR',
        date: '2024-01-15',
        url: 'https://github.com/...',
        type: 'pr',
      };

      const issueItem: SyncItem = {
        title: 'JIRA-123',
        date: '2024-01-15',
        url: 'https://jira.com/...',
        type: 'issue',
      };

      const emailItem: SyncItem = {
        title: 'Meeting notes',
        date: '2024-01-15',
        url: 'https://mail.com/...',
        type: 'email',
      };

      expect(commitItem.type).toBe('commit');
      expect(prItem.type).toBe('pr');
      expect(issueItem.type).toBe('issue');
      expect(emailItem.type).toBe('email');
    });
  });

  describe('SyncResult', () => {
    it('should have correct source values', () => {
      const githubResult: SyncResult = {
        source: 'github',
        items: [],
        synced_at: '2024-01-15T00:00:00Z',
      };

      const jiraResult: SyncResult = {
        source: 'jira',
        items: [],
        synced_at: '2024-01-15T00:00:00Z',
      };

      const hiworksResult: SyncResult = {
        source: 'hiworks',
        items: [],
        synced_at: '2024-01-15T00:00:00Z',
      };

      expect(githubResult.source).toBe('github');
      expect(jiraResult.source).toBe('jira');
      expect(hiworksResult.source).toBe('hiworks');
    });
  });
});
