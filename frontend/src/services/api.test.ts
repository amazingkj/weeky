import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  getTemplates,
  createTemplate,
  deleteTemplate,
  getConfig,
  updateConfig,
  syncGitHub,
  syncJira,
} from './api';

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe('API Service', () => {
  beforeEach(() => {
    mockFetch.mockClear();
    // Clear token for tests
    localStorage.removeItem('token');
  });

  describe('getTemplates', () => {
    it('should fetch templates successfully', async () => {
      const mockTemplates = [
        { id: 1, name: 'Template 1', style: '{}', created_at: '2024-01-01' },
      ];
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockTemplates),
      });

      const result = await getTemplates();
      expect(result).toEqual(mockTemplates);
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/templates',
        expect.objectContaining({ headers: expect.any(Object) }),
      );
    });

    it('should throw error on failure', async () => {
      mockFetch.mockResolvedValueOnce({ ok: false, status: 500 });

      await expect(getTemplates()).rejects.toThrow('Failed to fetch templates');
    });
  });

  describe('createTemplate', () => {
    it('should create template successfully', async () => {
      const mockTemplate = { id: 1, name: 'New Template', style: '{}', created_at: '2024-01-01' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockTemplate),
      });

      const result = await createTemplate('New Template', '{}');
      expect(result).toEqual(mockTemplate);
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/templates',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ name: 'New Template', style: '{}' }),
        }),
      );
    });
  });

  describe('deleteTemplate', () => {
    it('should delete template successfully', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      await expect(deleteTemplate(1)).resolves.not.toThrow();
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/templates/1',
        expect.objectContaining({ method: 'DELETE' }),
      );
    });
  });

  describe('getConfig', () => {
    it('should fetch config successfully', async () => {
      const mockConfig = { gitlab_token: '***configured***' };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockConfig),
      });

      const result = await getConfig();
      expect(result).toEqual(mockConfig);
    });
  });

  describe('updateConfig', () => {
    it('should update config successfully', async () => {
      mockFetch.mockResolvedValueOnce({ ok: true });

      await expect(updateConfig({ gitlab_token: 'new_token' })).resolves.not.toThrow();
      expect(mockFetch).toHaveBeenCalledWith(
        '/api/v1/config',
        expect.objectContaining({
          method: 'PUT',
          body: JSON.stringify({ configs: { gitlab_token: 'new_token' } }),
        }),
      );
    });
  });

  describe('syncGitHub', () => {
    it('should sync GitHub data successfully', async () => {
      const mockResult = {
        source: 'github',
        items: [{ title: 'Commit 1', date: '2024-01-01', url: 'http://...', type: 'commit' }],
        synced_at: '2024-01-01T00:00:00Z',
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResult),
      });

      const result = await syncGitHub({
        owner: 'test',
        repo: 'repo',
        start_date: '2024-01-01',
        end_date: '2024-01-07',
      });

      expect(result).toEqual(mockResult);
    });

    it('should throw error with message on failure', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: () => Promise.resolve({ error: 'API rate limit exceeded' }),
      });

      await expect(
        syncGitHub({
          owner: 'test',
          repo: 'repo',
          start_date: '2024-01-01',
          end_date: '2024-01-07',
        })
      ).rejects.toThrow('API rate limit exceeded');
    });
  });

  describe('syncJira', () => {
    it('should sync Jira data successfully', async () => {
      const mockResult = {
        source: 'jira',
        items: [{ title: '[JIRA-1] Issue', date: '2024-01-01', url: 'http://...', type: 'issue' }],
        synced_at: '2024-01-01T00:00:00Z',
      };
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve(mockResult),
      });

      const result = await syncJira({
        base_url: 'https://test.atlassian.net',
        start_date: '2024-01-01',
        end_date: '2024-01-07',
      });

      expect(result).toEqual(mockResult);
    });
  });
});
