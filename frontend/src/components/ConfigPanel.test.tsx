import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ConfigPanel from './ConfigPanel';
import * as api from '../services/api';

// Mock the API module
vi.mock('../services/api', () => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
}));

describe('ConfigPanel Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (api.getConfig as ReturnType<typeof vi.fn>).mockResolvedValue({});
  });

  it('renders GitLab settings section', async () => {
    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('GitLab')).toBeInTheDocument();
    });
  });

  it('renders Jira settings section', async () => {
    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('Jira')).toBeInTheDocument();
    });
  });

  it('renders Hiworks settings section', async () => {
    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('Hiworks')).toBeInTheDocument();
    });
  });

  it('renders Claude AI settings section', async () => {
    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('Claude AI')).toBeInTheDocument();
    });
  });

  it('shows configured indicator when config is set', async () => {
    (api.getConfig as ReturnType<typeof vi.fn>).mockResolvedValue({
      gitlab_token: '***configured***',
    });

    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('설정됨')).toBeInTheDocument();
    });
  });

  it('calls updateConfig on save', async () => {
    (api.updateConfig as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    render(<ConfigPanel />);

    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByText('GitLab')).toBeInTheDocument();
    });

    // Find and fill the GitLab token input (GitLab section is expanded by default)
    const tokenInputs = screen.getAllByPlaceholderText(/glpat-|새 토큰/);
    if (tokenInputs.length > 0) {
      fireEvent.change(tokenInputs[0], { target: { value: 'test-token' } });
    }

    // Click save button
    const saveButton = screen.getByText('설정 저장');
    fireEvent.click(saveButton);

    await waitFor(() => {
      expect(api.updateConfig).toHaveBeenCalled();
    });
  });

  it('shows success message after save', async () => {
    (api.updateConfig as ReturnType<typeof vi.fn>).mockResolvedValue(undefined);

    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('설정 저장')).toBeInTheDocument();
    });

    // Fill in a field (GitLab namespace field is visible in expanded section)
    const namespaceInput = screen.getByPlaceholderText('group 또는 username');
    fireEvent.change(namespaceInput, { target: { value: 'test-org' } });

    // Save
    fireEvent.click(screen.getByText('설정 저장'));

    await waitFor(() => {
      expect(screen.getByText('설정이 저장되었습니다.')).toBeInTheDocument();
    });
  });

  it('shows error message on save failure', async () => {
    (api.updateConfig as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Save failed'));

    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('설정 저장')).toBeInTheDocument();
    });

    // Fill in a field
    const namespaceInput = screen.getByPlaceholderText('group 또는 username');
    fireEvent.change(namespaceInput, { target: { value: 'test-org' } });

    // Save
    fireEvent.click(screen.getByText('설정 저장'));

    await waitFor(() => {
      expect(screen.getByText('저장에 실패했습니다.')).toBeInTheDocument();
    });
  });

  it('disables save button while saving', async () => {
    // Make updateConfig return a promise that doesn't resolve immediately
    let resolveUpdate: (value?: unknown) => void;
    (api.updateConfig as ReturnType<typeof vi.fn>).mockImplementation(
      () => new Promise(resolve => { resolveUpdate = resolve; })
    );

    render(<ConfigPanel />);

    await waitFor(() => {
      expect(screen.getByText('설정 저장')).toBeInTheDocument();
    });

    // Fill in a field to enable saving
    const namespaceInput = screen.getByPlaceholderText('group 또는 username');
    fireEvent.change(namespaceInput, { target: { value: 'test-org' } });

    // Click save
    fireEvent.click(screen.getByText('설정 저장'));

    // Button should show loading state
    await waitFor(() => {
      expect(screen.getByText('저장 중...')).toBeInTheDocument();
    });

    // Resolve the promise
    resolveUpdate!();

    await waitFor(() => {
      expect(screen.getByText('설정 저장')).toBeInTheDocument();
    });
  });
});
