import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { SkeletonList } from './Skeleton';

describe('SkeletonList', () => {
  it('화면 낭독기에는 불러오는 중으로 알리고, 지정한 행 수만큼 자리를 잡는다', () => {
    render(<SkeletonList rows={3} label="히스토리 불러오는 중" />);
    const status = screen.getByRole('status', { name: '히스토리 불러오는 중' });
    expect(status.children).toHaveLength(3);
  });
});
