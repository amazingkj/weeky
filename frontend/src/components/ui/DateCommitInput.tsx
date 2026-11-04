import { useEffect, useRef, useState } from 'react';

// 키보드로 날짜를 치는 동안 브라우저는 중간값으로 change를 낸다
// (연도 '0002-10-09', 일 '15'를 칠 때 첫 글자 '1'에서 '2026-10-01').
// 그런 값마다 보고서를 바꾸거나 확인창을 띄우지 않도록, 화면 값은 따로 들고 있다가
// 입력이 잠시 멈추거나(COMMIT_DELAY) 포커스가 빠지거나 Enter를 누를 때 완성된 날짜만 onCommit한다.
// onCommit이 false를 돌려주면(취소) 원래 값으로 되돌림.
const COMMIT_DELAY = 500;

const isCompleteDate = (v: string) => {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(v)) return false;
  const year = Number(v.slice(0, 4));
  return year >= 2000 && year <= 2100;
};

interface DateCommitInputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'value' | 'onChange' | 'type'> {
  value: string;
  onCommit: (value: string) => boolean | Promise<boolean>;
}

export default function DateCommitInput({ value, onCommit, onBlur, onKeyDown, ...rest }: DateCommitInputProps) {
  const [draft, setDraft] = useState(value);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();
  useEffect(() => { setDraft(value); }, [value]);
  useEffect(() => () => clearTimeout(timerRef.current), []);

  // onCommit이 확인창을 띄우면 포커스가 빠져 blur가 다시 commit을 부르므로 진행 중엔 무시
  const committingRef = useRef(false);
  const commit = async (next: string) => {
    clearTimeout(timerRef.current);
    if (committingRef.current) return;
    if (!isCompleteDate(next)) { setDraft(value); return; }
    if (next === value) return;
    committingRef.current = true;
    try {
      if (!(await onCommit(next))) setDraft(value);
    } finally {
      committingRef.current = false;
    }
  };

  return (
    <input
      {...rest}
      type="date"
      value={draft}
      onChange={(e) => {
        const next = e.target.value;
        setDraft(next);
        clearTimeout(timerRef.current);
        if (isCompleteDate(next) && next !== value) {
          timerRef.current = setTimeout(() => commit(next), COMMIT_DELAY);
        }
      }}
      onBlur={(e) => {
        if (draft !== value) commit(draft);
        onBlur?.(e);
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter') { e.preventDefault(); if (draft !== value) commit(draft); }
        onKeyDown?.(e);
      }}
    />
  );
}
