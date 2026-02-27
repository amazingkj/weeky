import { useState, useCallback, useRef, memo } from 'react';
import { Task } from '../types';

interface TaskListProps {
  title: string;
  description?: string;
  tasks: Task[];
  onChange: (tasks: Task[]) => void;
  showProgress?: boolean;
  emptyIcon?: React.ReactNode;
}

export default function TaskList({
  title,
  description,
  tasks,
  onChange,
  showProgress = true,
  emptyIcon
}: TaskListProps) {
  const idCounter = useRef(0);
  const taskIdsRef = useRef<number[]>([]);

  // Ensure every task has a stable ID (assign new IDs for newly appended tasks)
  while (taskIdsRef.current.length < tasks.length) {
    taskIdsRef.current.push(idCounter.current++);
  }
  taskIdsRef.current.length = tasks.length;

  const addTask = useCallback(() => {
    taskIdsRef.current.push(idCounter.current++);
    onChange([...tasks, { title: '', details: '', due_date: '', progress: 0 }]);
  }, [tasks, onChange]);

  const updateTask = useCallback((index: number, field: keyof Task, value: string | number) => {
    // IDs stay the same — no change needed
    const updated = tasks.map((task, i) =>
      i === index ? { ...task, [field]: value } : task
    );
    onChange(updated);
  }, [tasks, onChange]);

  const removeTask = useCallback((index: number) => {
    taskIdsRef.current.splice(index, 1);
    onChange(tasks.filter((_, i) => i !== index));
  }, [tasks, onChange]);

  const moveTask = useCallback((index: number, direction: 'up' | 'down') => {
    const newIndex = direction === 'up' ? index - 1 : index + 1;
    if (newIndex < 0 || newIndex >= tasks.length) return;

    // Swap IDs to match
    [taskIdsRef.current[index], taskIdsRef.current[newIndex]] =
      [taskIdsRef.current[newIndex], taskIdsRef.current[index]];

    const newTasks = [...tasks];
    [newTasks[index], newTasks[newIndex]] = [newTasks[newIndex], newTasks[index]];
    onChange(newTasks);
  }, [tasks, onChange]);

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
            {tasks.length > 0 ? (
              <span className="px-1.5 py-0.5 bg-neutral-100 text-neutral-600 text-xs font-medium rounded">
                {tasks.length}
              </span>
            ) : null}
          </div>
          {description ? (
            <p className="text-xs text-neutral-400 mt-0.5">{description}</p>
          ) : null}
        </div>
        <button
          type="button"
          onClick={addTask}
          className="flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium text-neutral-600
                     bg-neutral-100 hover:bg-neutral-200 rounded-lg transition-colors"
        >
          {plusIcon}
          추가
        </button>
      </div>

      {/* Task List */}
      {tasks.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center py-10 bg-neutral-50 rounded-lg border border-dashed border-neutral-200 cursor-pointer hover:border-neutral-300 transition-colors"
          onClick={addTask}
        >
          {emptyIcon || defaultEmptyIcon}
          <p className="text-neutral-400 text-sm mt-3">
            {showProgress ? '금주 업무를 추가해주세요' : '차주 계획을 추가해주세요'}
          </p>
          <p className="text-neutral-300 text-xs mt-1">클릭하여 첫 번째 항목 추가</p>
        </div>
      ) : (
        <div className="space-y-2">
          {tasks.map((task, index) => (
            <TaskItem
              key={taskIdsRef.current[index]}
              task={task}
              index={index}
              totalCount={tasks.length}
              showProgress={showProgress}
              onUpdate={(field, value) => updateTask(index, field, value)}
              onRemove={() => removeTask(index)}
              onMoveUp={() => moveTask(index, 'up')}
              onMoveDown={() => moveTask(index, 'down')}
            />
          ))}
        </div>
      )}
    </div>
  );
}

interface TaskItemProps {
  task: Task;
  index: number;
  totalCount: number;
  showProgress: boolean;
  onUpdate: (field: keyof Task, value: string | number) => void;
  onRemove: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
}

const TaskItem = memo(function TaskItem({
  task,
  index,
  totalCount,
  showProgress,
  onUpdate,
  onRemove,
  onMoveUp,
  onMoveDown
}: TaskItemProps) {
  const [showDescription, setShowDescription] = useState(() => !!task.description);

  const progressColor = task.progress === 100
    ? 'bg-neutral-900'
    : task.progress >= 50
      ? 'bg-neutral-500'
      : 'bg-neutral-300';

  return (
    <div className="group bg-neutral-50 hover:bg-neutral-100/80 p-4 rounded-lg border border-neutral-200 transition-colors">
      {/* Header Row */}
      <div className="flex items-center justify-between mb-2.5">
        <div className="flex items-center gap-2">
          <span className="w-5 h-5 flex items-center justify-center bg-neutral-200 text-neutral-600 text-xs font-mono rounded">
            {index + 1}
          </span>
          {task._carriedForward && (
            <span className="px-1.5 py-0.5 bg-blue-100 text-blue-600 text-[10px] font-medium rounded">
              이전 주
            </span>
          )}
          {showProgress ? (
            <span className={`px-1.5 py-0.5 text-xs font-medium rounded ${
              task.progress === 100
                ? 'bg-neutral-900 text-white'
                : task.progress >= 50
                  ? 'bg-neutral-200 text-neutral-700'
                  : 'bg-neutral-100 text-neutral-500'
            }`}>
              {task.progress === 100 ? '완료' : task.progress > 0 ? '진행중' : '예정'}
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
          <button type="button" onClick={onMoveUp} disabled={index === 0}
            className="p-1 text-neutral-400 hover:text-neutral-600 disabled:opacity-30" title="위로">
            {upIcon}
          </button>
          <button type="button" onClick={onMoveDown} disabled={index === totalCount - 1}
            className="p-1 text-neutral-400 hover:text-neutral-600 disabled:opacity-30" title="아래로">
            {downIcon}
          </button>
          <button type="button" onClick={onRemove}
            className="p-1 text-neutral-400 hover:text-red-500 transition-colors" title="삭제">
            {trashIcon}
          </button>
        </div>
      </div>

      {/* Title */}
      <input
        type="text"
        placeholder="업무 제목"
        value={task.title}
        onChange={(e) => onUpdate('title', e.target.value)}
        className="w-full px-2.5 py-1.5 text-sm bg-white border border-neutral-200 rounded-md
                   focus:outline-none focus:ring-1 focus:ring-neutral-400 focus:border-neutral-400
                   text-neutral-900 placeholder:text-neutral-300 font-medium transition-colors"
      />

      {/* Details */}
      <textarea
        placeholder={showProgress ? "진행 사항" : "계획 세부 내용"}
        value={task.details || ''}
        onChange={(e) => onUpdate('details', e.target.value)}
        rows={1}
        className="w-full mt-1.5 px-2.5 py-1.5 bg-white border border-neutral-200 rounded-md
                   focus:outline-none focus:ring-1 focus:ring-neutral-400 focus:border-neutral-400
                   text-neutral-700 text-xs placeholder:text-neutral-300 transition-colors resize-none"
      />

      {/* Description toggle + textarea */}
      <div className="mt-1.5">
        <button
          type="button"
          onClick={() => setShowDescription(!showDescription)}
          className="flex items-center gap-1 text-[11px] text-neutral-400 hover:text-neutral-600 transition-colors"
        >
          <svg className={`w-3 h-3 transition-transform ${showDescription ? 'rotate-90' : ''}`}
            fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
          상세내용
          {task.description ? <span className="w-1.5 h-1.5 bg-neutral-400 rounded-full" /> : null}
        </button>
        {showDescription && (
          <textarea
            placeholder="- 세부 작업 항목을 입력하세요&#10;- 여러 줄로 작성 가능합니다"
            value={task.description || ''}
            onChange={(e) => onUpdate('description', e.target.value)}
            rows={5}
            className="w-full mt-1 px-2.5 py-2 bg-white border border-neutral-200 rounded-md
                       focus:outline-none focus:ring-1 focus:ring-neutral-400 focus:border-neutral-400
                       text-neutral-700 text-sm leading-relaxed placeholder:text-neutral-300 transition-colors resize-y"
          />
        )}
      </div>

      {/* Bottom Row */}
      <div className="flex flex-wrap items-center gap-4 mt-2.5">
        <div className="flex items-center gap-2">
          <span className="text-xs text-neutral-400">{showProgress ? '완료일' : '예정일'}</span>
          <input
            type="date"
            value={task.due_date}
            onChange={(e) => onUpdate('due_date', e.target.value)}
            className="px-2 py-1 bg-white border border-neutral-200 rounded-md text-xs text-neutral-600
                       focus:outline-none focus:ring-1 focus:ring-neutral-400"
          />
        </div>

        {showProgress ? (
          <div className="flex items-center gap-2 flex-1">
            <span className="text-xs text-neutral-400">진척률</span>
            <div className="flex-1 max-w-[160px] flex items-center gap-2">
              <div className="flex-1 h-1.5 bg-neutral-200 rounded-full overflow-hidden">
                <div
                  className={`h-full ${progressColor} transition-all duration-200`}
                  style={{ width: `${task.progress}%` }}
                />
              </div>
              <input
                type="range" min="0" max="100" step="10"
                value={task.progress}
                onChange={(e) => onUpdate('progress', parseInt(e.target.value))}
                className="w-14 accent-neutral-900"
              />
              <span className="text-xs font-mono text-neutral-600 w-8 text-right">
                {task.progress}%
              </span>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
});

// Hoisted static SVG icons
const plusIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
  </svg>
);

const trashIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
  </svg>
);

const upIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M5 15l7-7 7 7" />
  </svg>
);

const downIcon = (
  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 9l-7 7-7-7" />
  </svg>
);

const defaultEmptyIcon = (
  <svg className="w-12 h-12 text-neutral-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
  </svg>
);
