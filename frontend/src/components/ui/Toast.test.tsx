import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import { ToastProvider, useToast } from './Toast';

function Trigger() {
  const toast = useToast();
  return (
    <>
      <button onClick={() => toast.error('저장 실패')}>오류</button>
      <button onClick={() => toast.success('저장 완료')}>성공</button>
    </>
  );
}

describe('Toast', () => {
  afterEach(() => vi.useRealTimers());

  it('여러 개를 쌓아 보여주고 시간이 지나면 사라진다', () => {
    vi.useFakeTimers();
    render(<ToastProvider><Trigger /></ToastProvider>);

    fireEvent.click(screen.getByRole('button', { name: '성공' }));
    fireEvent.click(screen.getByRole('button', { name: '오류' }));
    expect(screen.getAllByRole('alert')).toHaveLength(2);

    // 성공은 3.5초, 오류는 6초 (+ 닫힘 애니메이션 150ms)
    act(() => { vi.advanceTimersByTime(3700); });
    expect(screen.queryByText('저장 완료')).not.toBeInTheDocument();
    expect(screen.getByText('저장 실패')).toBeInTheDocument();

    act(() => { vi.advanceTimersByTime(2500); });
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('닫기 버튼으로 바로 닫을 수 있다', () => {
    vi.useFakeTimers();
    render(<ToastProvider><Trigger /></ToastProvider>);
    fireEvent.click(screen.getByRole('button', { name: '오류' }));

    fireEvent.click(screen.getByRole('button', { name: '닫기' }));
    act(() => { vi.advanceTimersByTime(200); });
    expect(screen.queryByText('저장 실패')).not.toBeInTheDocument();
  });
});
