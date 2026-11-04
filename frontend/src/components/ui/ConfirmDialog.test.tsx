import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ConfirmProvider, useConfirm, ConfirmOptions } from './ConfirmDialog';

// 버튼을 누르면 confirm을 띄우고 결과를 화면에 표시하는 테스트용 컴포넌트
function Trigger({ options }: { options: ConfirmOptions }) {
  const confirm = useConfirm();
  return (
    <>
      <button
        onClick={async (e) => {
          const target = e.currentTarget;
          const ok = await confirm(options);
          target.dataset.result = String(ok);
        }}
      >
        열기
      </button>
    </>
  );
}

function setup(options: ConfirmOptions = { title: '삭제할까요?' }) {
  render(<ConfirmProvider><Trigger options={options} /></ConfirmProvider>);
  const trigger = screen.getByRole('button', { name: '열기' });
  trigger.focus();
  fireEvent.click(trigger);
  return trigger;
}

describe('ConfirmDialog', () => {
  it('확인을 누르면 true로 끝나고 닫힌다', async () => {
    const trigger = setup({ title: '삭제할까요?', message: '되돌릴 수 없습니다.', confirmLabel: '삭제' });
    const dialog = screen.getByRole('alertdialog', { name: '삭제할까요?' });
    expect(dialog).toHaveAccessibleDescription('되돌릴 수 없습니다.');

    fireEvent.click(screen.getByRole('button', { name: '삭제' }));
    await waitFor(() => expect(trigger.dataset.result).toBe('true'));
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('Esc와 바깥 클릭은 취소로 처리한다', async () => {
    const trigger = setup();
    fireEvent.keyDown(screen.getByRole('alertdialog'), { key: 'Escape' });
    await waitFor(() => expect(trigger.dataset.result).toBe('false'));

    fireEvent.click(trigger);
    const backdrop = screen.getByRole('alertdialog').parentElement!;
    fireEvent.mouseDown(backdrop);
    await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
    expect(trigger.dataset.result).toBe('false');
  });

  it('위험 동작은 취소 버튼에 먼저 포커스하고, 닫히면 원래 버튼으로 포커스를 돌려준다', async () => {
    const trigger = setup({ title: '팀을 삭제할까요?', danger: true });
    expect(screen.getByRole('button', { name: '취소' })).toHaveFocus();

    // Tab은 두 버튼 안에서만 순환
    fireEvent.keyDown(screen.getByRole('alertdialog'), { key: 'Tab' });
    expect(screen.getByRole('button', { name: '확인' })).toHaveFocus();
    fireEvent.keyDown(screen.getByRole('alertdialog'), { key: 'Tab' });
    expect(screen.getByRole('button', { name: '취소' })).toHaveFocus();

    fireEvent.click(screen.getByRole('button', { name: '취소' }));
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it('본문을 클릭해 포커스가 옮겨가도 Esc로 닫힌다', async () => {
    const trigger = setup({ title: '삭제할까요?', message: '되돌릴 수 없습니다.' });
    const message = screen.getByText('되돌릴 수 없습니다.');
    // 브라우저처럼: 포커스 불가 요소를 클릭하면 가장 가까운 포커스 가능한 조상으로 포커스
    const dialog = screen.getByRole('alertdialog');
    expect(dialog).toHaveAttribute('tabindex', '-1');
    dialog.focus();
    fireEvent.keyDown(message, { key: 'Escape' });
    await waitFor(() => expect(trigger.dataset.result).toBe('false'));
  });

  it('떠 있는 동안 페이지의 전역 단축키로 키 입력이 전달되지 않는다', () => {
    const onWindowKey = vi.fn();
    window.addEventListener('keydown', onWindowKey);
    setup({ title: '바꿀까요?' });

    fireEvent.keyDown(screen.getByRole('button', { name: '확인' }), { key: 's', code: 'KeyS', ctrlKey: true });
    expect(onWindowKey).not.toHaveBeenCalled();
    window.removeEventListener('keydown', onWindowKey);
  });

  it('일반 동작은 확인 버튼에 먼저 포커스한다', () => {
    setup({ title: '지정할까요?' });
    expect(screen.getByRole('button', { name: '확인' })).toHaveFocus();
  });
});
