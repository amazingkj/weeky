import { useState, useEffect, useCallback } from 'react';
import { ConsolidationRule, ConsolidationRuleType, CreateConsolidationRuleRequest } from '../types';
import {
  getConsolidationRules,
  createConsolidationRule,
  updateConsolidationRule,
  deleteConsolidationRule,
  reorderConsolidationRules,
} from '../services/api';

interface RulesEditorProps {
  teamId: number;
}

const RULE_TYPE_LABEL: Record<ConsolidationRuleType, string> = {
  rename_title: '업무제목 변경',
  virtual_client: '가상 고객사 묶기',
};

const RULE_TYPE_HINT: Record<ConsolidationRuleType, string> = {
  rename_title:
    '예: 패턴 "MyData" → "마이데이터" 로 일괄 변경. ' +
    '잡다 카테고리(MMS지원, 유지보수 지원 등)를 "유지보수"로 통합할 때도 사용.',
  virtual_client:
    '특정 업무제목 안에서 고객사가 비어 있는 작업을 가상 고객사로 묶어 그룹화. ' +
    '예: 업무제목="CruzAPIM", 가상 고객사="본사" → CruzAPIM의 고객사 없는 작업이 "본사"로 모임.',
};

export default function RulesEditor({ teamId }: RulesEditorProps) {
  const [rules, setRules] = useState<ConsolidationRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // form state for adding new rule
  const [newType, setNewType] = useState<ConsolidationRuleType>('rename_title');
  const [newPattern, setNewPattern] = useState('');
  const [newReplacement, setNewReplacement] = useState('');
  const [newScopeTitle, setNewScopeTitle] = useState('');
  const [creating, setCreating] = useState(false);

  // editing state per row
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editDraft, setEditDraft] = useState<CreateConsolidationRuleRequest | null>(null);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getConsolidationRules(teamId);
      setRules(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : '규칙 조회 실패');
    } finally {
      setLoading(false);
    }
  }, [teamId]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const resetNewForm = () => {
    setNewPattern('');
    setNewReplacement('');
    setNewScopeTitle('');
  };

  const handleCreate = async () => {
    setError(null);
    const req: CreateConsolidationRuleRequest = {
      rule_type: newType,
      pattern: newPattern.trim(),
      replacement: newReplacement.trim(),
      scope_title: newScopeTitle.trim(),
    };
    if (newType === 'rename_title' && (!req.pattern || !req.replacement)) {
      setError('변환 대상(pattern)과 결과(replacement)를 모두 입력해주세요.');
      return;
    }
    if (newType === 'virtual_client' && (!req.scope_title || !req.replacement)) {
      setError('적용 업무제목(scope_title)과 가상 고객사(replacement)를 입력해주세요.');
      return;
    }
    setCreating(true);
    try {
      await createConsolidationRule(teamId, req);
      resetNewForm();
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : '규칙 생성 실패');
    } finally {
      setCreating(false);
    }
  };

  const handleStartEdit = (r: ConsolidationRule) => {
    setEditingId(r.id);
    setEditDraft({
      rule_type: r.rule_type,
      pattern: r.pattern,
      replacement: r.replacement,
      scope_title: r.scope_title || '',
    });
  };

  const handleSaveEdit = async (rid: number) => {
    if (!editDraft) return;
    setError(null);
    try {
      await updateConsolidationRule(teamId, rid, editDraft);
      setEditingId(null);
      setEditDraft(null);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : '규칙 수정 실패');
    }
  };

  const handleDelete = async (rid: number) => {
    if (!confirm('이 규칙을 삭제하시겠습니까?')) return;
    setError(null);
    try {
      await deleteConsolidationRule(teamId, rid);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : '규칙 삭제 실패');
    }
  };

  const handleMove = async (index: number, direction: 'up' | 'down') => {
    const next = [...rules];
    const swap = direction === 'up' ? index - 1 : index + 1;
    if (swap < 0 || swap >= next.length) return;
    [next[index], next[swap]] = [next[swap], next[index]];
    setRules(next);
    try {
      await reorderConsolidationRules(teamId, next.map(r => r.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : '순서 변경 실패');
      await refresh();
    }
  };

  if (loading) return <div className="text-sm text-neutral-500">불러오는 중...</div>;

  return (
    <div className="space-y-4">
      <div className="text-xs text-neutral-500 leading-relaxed">
        취합 시 자동 적용되는 변환 규칙입니다. 위에서 아래 순서대로 적용됩니다.
        규칙은 원본 보고서에 적용되며, 사용자가 취합 편집을 저장한 경우에는 편집본이 우선됩니다.
      </div>

      {/* Add new rule */}
      <div className="border border-neutral-200 rounded-lg p-3 space-y-2 bg-neutral-50">
        <div className="text-xs font-medium text-neutral-700">새 규칙 추가</div>
        <div className="flex flex-wrap gap-2 items-end">
          <div>
            <label className="block text-[10px] text-neutral-500 mb-0.5">유형</label>
            <select
              value={newType}
              onChange={e => setNewType(e.target.value as ConsolidationRuleType)}
              className="text-xs px-2 py-1.5 border border-neutral-300 rounded">
              <option value="rename_title">{RULE_TYPE_LABEL.rename_title}</option>
              <option value="virtual_client">{RULE_TYPE_LABEL.virtual_client}</option>
            </select>
          </div>
          {newType === 'rename_title' ? (
            <>
              <div>
                <label className="block text-[10px] text-neutral-500 mb-0.5">원본 업무제목</label>
                <input
                  value={newPattern}
                  onChange={e => setNewPattern(e.target.value)}
                  placeholder="MyData"
                  className="text-xs px-2 py-1.5 border border-neutral-300 rounded w-40" />
              </div>
              <div>
                <label className="block text-[10px] text-neutral-500 mb-0.5">변경 결과</label>
                <input
                  value={newReplacement}
                  onChange={e => setNewReplacement(e.target.value)}
                  placeholder="마이데이터"
                  className="text-xs px-2 py-1.5 border border-neutral-300 rounded w-40" />
              </div>
            </>
          ) : (
            <>
              <div>
                <label className="block text-[10px] text-neutral-500 mb-0.5">적용 업무제목</label>
                <input
                  value={newScopeTitle}
                  onChange={e => setNewScopeTitle(e.target.value)}
                  placeholder="CruzAPIM"
                  className="text-xs px-2 py-1.5 border border-neutral-300 rounded w-40" />
              </div>
              <div>
                <label className="block text-[10px] text-neutral-500 mb-0.5">가상 고객사명</label>
                <input
                  value={newReplacement}
                  onChange={e => setNewReplacement(e.target.value)}
                  placeholder="본사"
                  className="text-xs px-2 py-1.5 border border-neutral-300 rounded w-40" />
              </div>
            </>
          )}
          <button
            onClick={handleCreate}
            disabled={creating}
            className="text-xs px-3 py-1.5 bg-neutral-900 text-white rounded hover:bg-neutral-800 disabled:opacity-40">
            {creating ? '추가 중...' : '추가'}
          </button>
        </div>
        <p className="text-[10px] text-neutral-500 leading-relaxed">{RULE_TYPE_HINT[newType]}</p>
      </div>

      {error && (
        <div className="text-xs text-red-600 p-2 bg-red-50 border border-red-200 rounded">{error}</div>
      )}

      {/* Rules list */}
      {rules.length === 0 ? (
        <div className="text-sm text-neutral-400 text-center py-8 border border-dashed border-neutral-200 rounded">
          등록된 규칙이 없습니다.
        </div>
      ) : (
        <div className="border border-neutral-200 rounded-lg overflow-hidden">
          <table className="w-full text-xs">
            <thead className="bg-neutral-50">
              <tr>
                <th className="w-10 px-2 py-2 text-center font-medium text-neutral-600">#</th>
                <th className="px-2 py-2 text-left font-medium text-neutral-600">유형</th>
                <th className="px-2 py-2 text-left font-medium text-neutral-600">조건</th>
                <th className="px-2 py-2 text-left font-medium text-neutral-600">결과</th>
                <th className="w-32 px-2 py-2 text-right font-medium text-neutral-600">작업</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((r, i) => {
                const isEditing = editingId === r.id;
                return (
                  <tr key={r.id} className="border-t border-neutral-100">
                    <td className="px-2 py-2 text-center text-neutral-400">{i + 1}</td>
                    {isEditing ? (
                      <>
                        <td className="px-2 py-2">
                          <select
                            value={editDraft?.rule_type}
                            onChange={e => setEditDraft(d => d && { ...d, rule_type: e.target.value as ConsolidationRuleType })}
                            className="text-xs px-1.5 py-1 border border-neutral-300 rounded">
                            <option value="rename_title">{RULE_TYPE_LABEL.rename_title}</option>
                            <option value="virtual_client">{RULE_TYPE_LABEL.virtual_client}</option>
                          </select>
                        </td>
                        <td className="px-2 py-2">
                          <input
                            value={editDraft?.rule_type === 'virtual_client' ? (editDraft?.scope_title || '') : (editDraft?.pattern || '')}
                            onChange={e => {
                              const v = e.target.value;
                              setEditDraft(d => d && (d.rule_type === 'virtual_client' ? { ...d, scope_title: v } : { ...d, pattern: v }));
                            }}
                            className="text-xs px-1.5 py-1 border border-neutral-300 rounded w-full" />
                        </td>
                        <td className="px-2 py-2">
                          <input
                            value={editDraft?.replacement || ''}
                            onChange={e => setEditDraft(d => d && { ...d, replacement: e.target.value })}
                            className="text-xs px-1.5 py-1 border border-neutral-300 rounded w-full" />
                        </td>
                        <td className="px-2 py-2 text-right space-x-1">
                          <button onClick={() => handleSaveEdit(r.id)}
                            className="text-xs px-2 py-1 bg-neutral-900 text-white rounded hover:bg-neutral-800">
                            저장
                          </button>
                          <button onClick={() => { setEditingId(null); setEditDraft(null); }}
                            className="text-xs px-2 py-1 border border-neutral-200 rounded hover:bg-neutral-50">
                            취소
                          </button>
                        </td>
                      </>
                    ) : (
                      <>
                        <td className="px-2 py-2 text-neutral-600">{RULE_TYPE_LABEL[r.rule_type]}</td>
                        <td className="px-2 py-2 font-mono text-neutral-700">
                          {r.rule_type === 'virtual_client'
                            ? `[${r.scope_title}] + 빈 고객사`
                            : `"${r.pattern}"`}
                        </td>
                        <td className="px-2 py-2 font-mono text-neutral-700">
                          {r.rule_type === 'virtual_client'
                            ? `고객사 → "${r.replacement}"`
                            : `→ "${r.replacement}"`}
                        </td>
                        <td className="px-2 py-2 text-right">
                          <div className="inline-flex gap-1">
                            <button onClick={() => handleMove(i, 'up')} disabled={i === 0}
                              className="text-neutral-400 hover:text-neutral-700 disabled:opacity-30 disabled:cursor-not-allowed px-1"
                              title="위로">↑</button>
                            <button onClick={() => handleMove(i, 'down')} disabled={i === rules.length - 1}
                              className="text-neutral-400 hover:text-neutral-700 disabled:opacity-30 disabled:cursor-not-allowed px-1"
                              title="아래로">↓</button>
                            <button onClick={() => handleStartEdit(r)}
                              className="text-xs px-2 py-1 border border-neutral-200 rounded hover:bg-neutral-50">
                              수정
                            </button>
                            <button onClick={() => handleDelete(r.id)}
                              className="text-xs px-2 py-1 text-red-600 border border-red-200 rounded hover:bg-red-50">
                              삭제
                            </button>
                          </div>
                        </td>
                      </>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
