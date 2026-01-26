import { useState, useEffect, useCallback } from 'react';
import { ConfigMap } from '../types';
import { getConfig, updateConfig } from '../services/api';

interface UseConfigReturn {
  config: ConfigMap;
  isLoading: boolean;
  error: string | null;
  updateConfigValue: (key: string, value: string) => void;
  saveConfig: () => Promise<boolean>;
  refetch: () => Promise<void>;
}

export function useConfig(): UseConfigReturn {
  const [config, setConfig] = useState<ConfigMap>({});
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const data = await getConfig();
      setConfig(data);
    } catch (err) {
      setError('설정을 불러오는데 실패했습니다.');
      console.error('Failed to load config:', err);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  const updateConfigValue = useCallback((key: string, value: string) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  }, []);

  const saveConfig = useCallback(async (): Promise<boolean> => {
    setError(null);
    try {
      await updateConfig(config);
      return true;
    } catch (err) {
      setError('설정 저장에 실패했습니다.');
      console.error('Failed to save config:', err);
      return false;
    }
  }, [config]);

  return {
    config,
    isLoading,
    error,
    updateConfigValue,
    saveConfig,
    refetch: fetchConfig,
  };
}
