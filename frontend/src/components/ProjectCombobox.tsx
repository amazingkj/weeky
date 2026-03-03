import { useState, useRef, useEffect } from 'react';
import { TeamProject } from '../types';

interface ProjectComboboxProps {
  value: string;
  onChange: (value: string) => void;
  onSelectProject?: (project: TeamProject) => void;
  projects: TeamProject[];
  onAutoCreate?: (name: string) => void;
  placeholder?: string;
}

export default function ProjectCombobox({
  value,
  onChange,
  onSelectProject,
  projects,
  onAutoCreate,
  placeholder = '업무 제목',
}: ProjectComboboxProps) {
  const [open, setOpen] = useState(false);
  const [filter, setFilter] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const filtered = projects.filter(p =>
    p.is_active && p.name.toLowerCase().includes((filter || value).toLowerCase())
  );

  const showDropdown = open && (filtered.length > 0 || filter.length > 0);
  const isNew = filter.length > 0 && !projects.some(p => p.name === filter);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <div ref={wrapperRef} className="relative">
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={(e) => {
          onChange(e.target.value);
          setFilter(e.target.value);
          setOpen(true);
        }}
        onFocus={() => {
          setFilter(value);
          setOpen(true);
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            setOpen(false);
            inputRef.current?.blur();
          }
        }}
        placeholder={placeholder}
        className="w-full px-2.5 py-1.5 text-sm bg-white border border-neutral-200 rounded-md
                   focus:outline-none focus:ring-1 focus:ring-neutral-400 focus:border-neutral-400
                   text-neutral-900 placeholder:text-neutral-300 font-medium transition-colors"
      />
      {showDropdown && (
        <div className="absolute z-50 left-0 right-0 mt-0.5 bg-white border border-neutral-200 rounded-md shadow-lg max-h-48 overflow-auto">
          {filtered.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => {
                onChange(p.name);
                if (onSelectProject) onSelectProject(p);
                setFilter('');
                setOpen(false);
              }}
              className={`w-full text-left px-2.5 py-1.5 text-sm hover:bg-neutral-100 transition-colors flex items-center justify-between ${
                p.name === value && !p.client ? 'bg-neutral-50 font-medium' : ''
              }`}
            >
              <span>{p.name}</span>
              {p.client && (
                <span className="text-[10px] text-neutral-400 ml-2">{p.client}</span>
              )}
            </button>
          ))}
          {isNew && (
            <button
              type="button"
              onClick={() => {
                onChange(filter);
                if (onAutoCreate) onAutoCreate(filter);
                setFilter('');
                setOpen(false);
              }}
              className="w-full text-left px-2.5 py-1.5 text-sm text-blue-600 hover:bg-blue-50 transition-colors border-t border-neutral-100"
            >
              + "{filter}" 새 프로젝트로 추가
            </button>
          )}
        </div>
      )}
    </div>
  );
}
