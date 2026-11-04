import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
import DateCommitInput from './DateCommitInput';

function setup(onCommit: (v: string) => boolean | Promise<boolean>, value = '2026-10-06') {
  render(<DateCommitInput aria-label="날짜" value={value} onCommit={onCommit} />);
  return screen.getByLabelText('날짜') as HTMLInputElement;
}

const change = async (input: HTMLInputElement, value: string) => {
  await act(async () => { fireEvent.change(input, { target: { value } }); });
};
const wait = async (ms: number) => {
  await act(async () => { vi.advanceTimersByTime(ms); });
};

describe('DateCommitInput', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('키보드로 연도를 치는 중간값은 반영하지 않고, 완성된 날짜는 입력이 멈춘 뒤 반영한다', async () => {
    const onCommit = vi.fn().mockReturnValue(true);
    const input = setup(onCommit);

    for (const partial of ['0002-10-09', '0020-10-09', '0202-10-09']) {
      await change(input, partial);
      expect(input.value).toBe(partial);
    }
    await wait(1000);
    expect(onCommit).not.toHaveBeenCalled();

    await change(input, '2026-10-09');
    expect(onCommit).not.toHaveBeenCalled();
    await wait(600);
    expect(onCommit).toHaveBeenCalledWith('2026-10-09');
  });

  it('일을 두 자리로 칠 때 첫 글자의 날짜로 반영하지 않는다', async () => {
    const onCommit = vi.fn().mockReturnValue(true);
    const input = setup(onCommit, '2026-10-09');

    await change(input, '2026-10-01'); // '1'
    await wait(200);
    await change(input, '2026-10-15'); // '5'
    await wait(600);
    expect(onCommit).toHaveBeenCalledTimes(1);
    expect(onCommit).toHaveBeenCalledWith('2026-10-15');
  });

  it('포커스가 빠지거나 Enter를 누르면 기다리지 않고 반영한다', async () => {
    const onCommit = vi.fn().mockReturnValue(true);
    const input = setup(onCommit);

    await change(input, '2026-10-09');
    await act(async () => { fireEvent.blur(input); });
    expect(onCommit).toHaveBeenCalledWith('2026-10-09');
    await wait(600);
    expect(onCommit).toHaveBeenCalledTimes(1);
  });

  it('비운 값이나 완성하지 않은 값은 반영하지 않고 벗어나면 원래 값으로 되돌린다', async () => {
    const onCommit = vi.fn().mockReturnValue(true);
    const input = setup(onCommit);

    await change(input, '');
    await act(async () => { fireEvent.blur(input); });
    expect(input.value).toBe('2026-10-06');

    await change(input, '0002-10-09');
    await act(async () => { fireEvent.blur(input); });
    expect(input.value).toBe('2026-10-06');
    expect(onCommit).not.toHaveBeenCalled();
  });

  it('반영이 취소되면 원래 값으로 되돌리고, 확인 중 포커스가 빠져도 다시 반영하지 않는다', async () => {
    let resolve!: (ok: boolean) => void;
    const onCommit = vi.fn(() => new Promise<boolean>((r) => { resolve = r; }));
    const input = setup(onCommit);

    await change(input, '2026-01-05');
    await wait(600);
    expect(onCommit).toHaveBeenCalledTimes(1);
    // 확인창이 떠서 포커스가 빠짐
    await act(async () => { fireEvent.blur(input); });
    expect(onCommit).toHaveBeenCalledTimes(1);

    await act(async () => { resolve(false); });
    expect(input.value).toBe('2026-10-06');
  });
});
