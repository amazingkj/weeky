import { useState, useEffect, useCallback, useMemo } from 'react';
import { Template, TemplateStyle, defaultTemplateStyle, parseTemplateStyle } from '../types';
import { getTemplates, createTemplate, updateTemplate, deleteTemplate } from '../services/api';

interface UseTemplatesReturn {
  templates: Template[];
  isLoading: boolean;
  error: string | null;
  createNewTemplate: (name: string, style: TemplateStyle) => Promise<boolean>;
  updateExistingTemplate: (id: number, name: string, style: TemplateStyle) => Promise<boolean>;
  removeTemplate: (id: number) => Promise<boolean>;
  getTemplateStyle: (templateId: number) => TemplateStyle;
  refetch: () => Promise<void>;
}

export function useTemplates(): UseTemplatesReturn {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTemplates = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await getTemplates();
      setTemplates(data);
    } catch (err) {
      setError('템플릿을 불러오는데 실패했습니다.');
      console.error('Failed to load templates:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  const createNewTemplate = useCallback(async (name: string, style: TemplateStyle): Promise<boolean> => {
    setError(null);
    try {
      await createTemplate(name, JSON.stringify(style));
      await fetchTemplates();
      return true;
    } catch (err) {
      setError('템플릿 생성에 실패했습니다.');
      console.error('Failed to create template:', err);
      return false;
    }
  }, [fetchTemplates]);

  const updateExistingTemplate = useCallback(async (id: number, name: string, style: TemplateStyle): Promise<boolean> => {
    setError(null);
    try {
      await updateTemplate(id, name, JSON.stringify(style));
      await fetchTemplates();
      return true;
    } catch (err) {
      setError('템플릿 수정에 실패했습니다.');
      console.error('Failed to update template:', err);
      return false;
    }
  }, [fetchTemplates]);

  const removeTemplate = useCallback(async (id: number): Promise<boolean> => {
    setError(null);
    try {
      await deleteTemplate(id);
      await fetchTemplates();
      return true;
    } catch (err) {
      setError('템플릿 삭제에 실패했습니다.');
      console.error('Failed to delete template:', err);
      return false;
    }
  }, [fetchTemplates]);

  // Build index map for O(1) lookups (js-index-maps)
  const templateById = useMemo(
    () => new Map(templates.map((t) => [t.id, t])),
    [templates]
  );

  const getTemplateStyle = useCallback((templateId: number): TemplateStyle => {
    if (templateId === 0) {
      return defaultTemplateStyle;
    }
    const template = templateById.get(templateId);
    return template ? parseTemplateStyle(template.style) : defaultTemplateStyle;
  }, [templateById]);

  return {
    templates,
    isLoading,
    error,
    createNewTemplate,
    updateExistingTemplate,
    removeTemplate,
    getTemplateStyle,
    refetch: fetchTemplates,
  };
}
