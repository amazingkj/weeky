import { useState, useCallback, memo } from 'react';
import { Template, TemplateStyle, defaultTemplateStyle, parseTemplateStyle } from '../types';
import { useTemplates } from '../hooks';
// Direct imports to avoid barrel file overhead (bundle-barrel-imports)
import Loading from './ui/Loading';
import Alert from './ui/Alert';

// Preset templates
const PRESETS = [
  { name: '기본', primary: '#2563EB', secondary: '#64748B' },
  { name: '다크', primary: '#1E293B', secondary: '#475569' },
  { name: '그린', primary: '#059669', secondary: '#6B7280' },
  { name: '레드', primary: '#DC2626', secondary: '#6B7280' },
  { name: '퍼플', primary: '#7C3AED', secondary: '#6B7280' },
  { name: '오렌지', primary: '#EA580C', secondary: '#6B7280' },
] as const;

export default function TemplateManager() {
  const { templates, isLoading, error, createNewTemplate, updateExistingTemplate, removeTemplate } = useTemplates();
  const [newName, setNewName] = useState('');
  const [newStyle, setNewStyle] = useState<TemplateStyle>(defaultTemplateStyle);
  const [isCreating, setIsCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editStyle, setEditStyle] = useState<TemplateStyle>(defaultTemplateStyle);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const handleCreate = useCallback(async () => {
    if (!newName.trim()) {
      setMessage({ type: 'error', text: '템플릿 이름을 입력해주세요.' });
      return;
    }

    setIsCreating(true);
    const success = await createNewTemplate(newName.trim(), newStyle);
    if (success) {
      setNewName('');
      setNewStyle(defaultTemplateStyle);
      setMessage({ type: 'success', text: '템플릿이 생성되었습니다.' });
    } else {
      setMessage({ type: 'error', text: '템플릿 생성에 실패했습니다.' });
    }
    setIsCreating(false);
  }, [newName, newStyle, createNewTemplate]);

  const handleDelete = useCallback(async (id: number) => {
    if (!confirm('이 템플릿을 삭제하시겠습니까?')) return;

    const success = await removeTemplate(id);
    if (!success) {
      setMessage({ type: 'error', text: '템플릿 삭제에 실패했습니다.' });
    }
  }, [removeTemplate]);

  const handleUpdate = useCallback(async (template: Template) => {
    const success = await updateExistingTemplate(template.id, template.name, editStyle);
    if (success) {
      setEditingId(null);
      setEditStyle(defaultTemplateStyle);
      setMessage({ type: 'success', text: '템플릿이 수정되었습니다.' });
    } else {
      setMessage({ type: 'error', text: '템플릿 수정에 실패했습니다.' });
    }
  }, [editStyle, updateExistingTemplate]);

  const startEditing = useCallback((template: Template) => {
    setEditingId(template.id);
    setEditStyle(parseTemplateStyle(template.style));
  }, []);

  const cancelEditing = useCallback(() => {
    setEditingId(null);
    setEditStyle(defaultTemplateStyle);
  }, []);

  const applyPreset = useCallback((preset: typeof PRESETS[number]) => {
    setNewStyle({
      ...defaultTemplateStyle,
      primaryColor: preset.primary,
      secondaryColor: preset.secondary,
    });
  }, []);

  if (isLoading) {
    return (
      <div className="py-12">
        <Loading text="템플릿을 불러오는 중..." />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {message && (
        <Alert type={message.type} onClose={() => setMessage(null)}>
          {message.text}
        </Alert>
      )}

      {error && <Alert type="error">{error}</Alert>}

      {/* Create Template */}
      <section className="bg-white p-6 rounded-lg shadow-sm border">
        <h2 className="text-xl font-bold text-gray-800 mb-4">새 템플릿 등록</h2>
        <div className="space-y-4">
          <div className="flex gap-2">
            <input
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="템플릿 이름 (예: 회사 공식 양식)"
              className="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <button
              onClick={handleCreate}
              disabled={isCreating}
              className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {isCreating ? '생성 중...' : '등록'}
            </button>
          </div>

          <StyleEditor style={newStyle} onChange={setNewStyle} />
        </div>
      </section>

      {/* Preset Templates */}
      <section className="bg-white p-6 rounded-lg shadow-sm border">
        <h2 className="text-xl font-bold text-gray-800 mb-4">프리셋 템플릿</h2>
        <div className="grid grid-cols-3 gap-3">
          {PRESETS.map((preset) => (
            <button
              key={preset.name}
              onClick={() => applyPreset(preset)}
              className="p-3 border rounded-lg hover:border-blue-500 transition-colors"
            >
              <div
                className="h-8 rounded mb-2"
                style={{ backgroundColor: preset.primary }}
              />
              <span className="text-sm text-gray-700">{preset.name}</span>
            </button>
          ))}
        </div>
      </section>

      {/* Template List */}
      <section className="bg-white p-6 rounded-lg shadow-sm border">
        <h2 className="text-xl font-bold text-gray-800 mb-4">저장된 템플릿</h2>
        {templates.length === 0 ? (
          <p className="text-gray-500">등록된 템플릿이 없습니다.</p>
        ) : (
          <ul className="divide-y divide-gray-200">
            {templates.map((template) => (
              <TemplateItem
                key={template.id}
                template={template}
                isEditing={editingId === template.id}
                editStyle={editStyle}
                onEditStyleChange={setEditStyle}
                onStartEditing={startEditing}
                onCancelEditing={cancelEditing}
                onSave={handleUpdate}
                onDelete={handleDelete}
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

// Sub-components

interface TemplateItemProps {
  template: Template;
  isEditing: boolean;
  editStyle: TemplateStyle;
  onEditStyleChange: (style: TemplateStyle) => void;
  onStartEditing: (template: Template) => void;
  onCancelEditing: () => void;
  onSave: (template: Template) => void;
  onDelete: (id: number) => void;
}

const TemplateItem = memo(function TemplateItem({
  template,
  isEditing,
  editStyle,
  onEditStyleChange,
  onStartEditing,
  onCancelEditing,
  onSave,
  onDelete,
}: TemplateItemProps) {
  const style = parseTemplateStyle(template.style);

  return (
    <li className="py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div
            className="w-8 h-8 rounded"
            style={{ backgroundColor: style.primaryColor }}
          />
          <div>
            <p className="font-medium text-gray-800">{template.name}</p>
            <p className="text-sm text-gray-500">
              {new Date(template.created_at).toLocaleDateString('ko-KR')}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          {!isEditing && (
            <button
              onClick={() => onStartEditing(template)}
              className="text-blue-500 hover:text-blue-600 text-sm"
            >
              수정
            </button>
          )}
          <button
            onClick={() => onDelete(template.id)}
            className="text-red-500 hover:text-red-600 text-sm"
          >
            삭제
          </button>
        </div>
      </div>

      {isEditing && (
        <div className="mt-4">
          <StyleEditor style={editStyle} onChange={onEditStyleChange} />
          <div className="flex gap-2 mt-4">
            <button
              onClick={onCancelEditing}
              className="px-3 py-1 text-sm text-gray-600 border rounded hover:bg-gray-50"
            >
              취소
            </button>
            <button
              onClick={() => onSave(template)}
              className="px-3 py-1 text-sm text-white bg-blue-600 rounded hover:bg-blue-700"
            >
              저장
            </button>
          </div>
        </div>
      )}
    </li>
  );
});

// Style Editor Component

interface StyleEditorProps {
  style: TemplateStyle;
  onChange: (style: TemplateStyle) => void;
}

function StyleEditor({ style, onChange }: StyleEditorProps) {
  return (
    <div className="space-y-3 mt-4 p-4 bg-gray-50 rounded-lg">
      <h4 className="font-medium text-gray-700">스타일 설정</h4>

      <ColorPicker
        label="메인 색상"
        value={style.primaryColor}
        onChange={(v) => onChange({ ...style, primaryColor: v })}
      />

      <ColorPicker
        label="보조 색상"
        value={style.secondaryColor}
        onChange={(v) => onChange({ ...style, secondaryColor: v })}
      />

      <div className="flex items-center gap-2">
        <label className="text-sm text-gray-600 w-24">제목 크기</label>
        <input
          type="number"
          value={style.titleFontSize}
          onChange={(e) => onChange({ ...style, titleFontSize: parseInt(e.target.value) || 36 })}
          className="w-20 px-2 py-1 text-sm border rounded"
          min="20"
          max="60"
        />
        <span className="text-sm text-gray-500">pt</span>
      </div>

      <div className="flex items-center gap-2">
        <label className="text-sm text-gray-600 w-24">본문 크기</label>
        <input
          type="number"
          value={style.bodyFontSize}
          onChange={(e) => onChange({ ...style, bodyFontSize: parseInt(e.target.value) || 11 })}
          className="w-20 px-2 py-1 text-sm border rounded"
          min="8"
          max="20"
        />
        <span className="text-sm text-gray-500">pt</span>
      </div>

      <div className="flex items-center gap-2">
        <label className="text-sm text-gray-600 w-24">헤더 정렬</label>
        <select
          value={style.headerLayout}
          onChange={(e) => onChange({ ...style, headerLayout: e.target.value as 'left' | 'center' })}
          className="px-2 py-1 text-sm border rounded"
        >
          <option value="center">가운데</option>
          <option value="left">왼쪽</option>
        </select>
      </div>

      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="showProgressBar"
          checked={style.showProgressBar}
          onChange={(e) => onChange({ ...style, showProgressBar: e.target.checked })}
          className="rounded"
        />
        <label htmlFor="showProgressBar" className="text-sm text-gray-600">
          진척률 바 표시
        </label>
      </div>

      {/* Preview */}
      <div className="mt-4 p-3 border rounded-lg bg-white">
        <p className="text-xs text-gray-500 mb-2">미리보기</p>
        <div
          className="h-20 rounded flex items-center justify-center"
          style={{ backgroundColor: style.primaryColor }}
        >
          <span
            className="text-white font-bold"
            style={{ fontSize: `${style.titleFontSize / 3}px` }}
          >
            주간업무보고
          </span>
        </div>
      </div>
    </div>
  );
}

// Color Picker Component

interface ColorPickerProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
}

function ColorPicker({ label, value, onChange }: ColorPickerProps) {
  return (
    <div className="flex items-center gap-2">
      <label className="text-sm text-gray-600 w-24">{label}</label>
      <input
        type="color"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-10 h-8 rounded border cursor-pointer"
        aria-label={label}
      />
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-24 px-2 py-1 text-sm border rounded"
        placeholder="#000000"
      />
    </div>
  );
}
