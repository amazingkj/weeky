import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import TaskList from './TaskList';
import { Task } from '../types';

describe('TaskList Component', () => {
  const mockOnChange = vi.fn();

  const defaultTasks: Task[] = [
    { title: 'Task 1', due_date: '2024-01-15', progress: 50 },
    { title: 'Task 2', due_date: '2024-01-16', progress: 100 },
  ];

  beforeEach(() => {
    mockOnChange.mockClear();
  });

  it('renders with title', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={[]}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('금주실적')).toBeInTheDocument();
  });

  it('renders tasks correctly', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={defaultTasks}
        onChange={mockOnChange}
        showProgress={true}
      />
    );

    expect(screen.getByDisplayValue('Task 1')).toBeInTheDocument();
    expect(screen.getByDisplayValue('Task 2')).toBeInTheDocument();
  });

  it('calls onChange when adding a task', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={[]}
        onChange={mockOnChange}
      />
    );

    const addButton = screen.getByText('추가');
    fireEvent.click(addButton);

    expect(mockOnChange).toHaveBeenCalledWith([
      { title: '', details: '', due_date: '', progress: 0 }
    ]);
  });

  it('calls onChange when removing a task', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={defaultTasks}
        onChange={mockOnChange}
      />
    );

    const removeButtons = screen.getAllByTitle('삭제');
    fireEvent.click(removeButtons[0]);

    expect(mockOnChange).toHaveBeenCalledWith([defaultTasks[1]]);
  });

  it('calls onChange when editing task title', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={defaultTasks}
        onChange={mockOnChange}
      />
    );

    const titleInput = screen.getByDisplayValue('Task 1');
    fireEvent.change(titleInput, { target: { value: 'Updated Task 1' } });

    expect(mockOnChange).toHaveBeenCalled();
  });

  it('shows progress when showProgress is true', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={defaultTasks}
        onChange={mockOnChange}
        showProgress={true}
      />
    );

    // Progress text should be visible
    expect(screen.getByText('50%')).toBeInTheDocument();
    expect(screen.getByText('100%')).toBeInTheDocument();
  });

  it('hides progress when showProgress is false', () => {
    render(
      <TaskList
        title="차주계획"
        tasks={defaultTasks}
        onChange={mockOnChange}
        showProgress={false}
      />
    );

    // Progress text should not be visible
    expect(screen.queryByText('50%')).not.toBeInTheDocument();
  });

  it('renders empty state correctly', () => {
    render(
      <TaskList
        title="금주실적"
        tasks={[]}
        onChange={mockOnChange}
      />
    );

    // Should show add button even with no tasks
    expect(screen.getByText('추가')).toBeInTheDocument();
    expect(screen.getByText('금주 업무를 추가해주세요')).toBeInTheDocument();
  });
});
