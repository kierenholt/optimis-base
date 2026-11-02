import { useCallback, useEffect, useState } from 'react';
import {
  applyHostThemeToRoot,
  BaseFontSize,
  DEFAULT_THEME_TOKENS,
  normalizeBaseFontSize,
  ThemeMode,
} from '../../themeAndTypes';

export interface UseWorkspaceThemeAndBrandingParams {
  initialAcademicYearId?: string;
  activeStaffId: string;
}

export function useWorkspaceThemeAndBranding({
  initialAcademicYearId,
  activeStaffId,
}: UseWorkspaceThemeAndBrandingParams) {
  // Host Theme & School Branding state (Sections 9.4 & 9.5)
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    applyHostThemeToRoot('light', DEFAULT_THEME_TOKENS, 'medium');
    return 'light';
  });
  const [baseFontSize, setBaseFontSize] = useState<BaseFontSize>('medium');
  const [themeCatalog, setThemeCatalog] = useState<Record<ThemeMode, Record<string, string>>>(
    DEFAULT_THEME_TOKENS
  );
  const [schoolName, setSchoolName] = useState<string>('Oakridge All-Through Academy');
  const [schoolURN, setSchoolURN] = useState<string>('100001');
  const [configuredAcademicYear, setConfiguredAcademicYear] = useState<string>(
    initialAcademicYearId || 'ay_2026_2027'
  );

  useEffect(() => {
    applyHostThemeToRoot(themeMode, themeCatalog, baseFontSize);
  }, [themeMode, themeCatalog, baseFontSize]);

  const persistUserSettings = useCallback(
    async (updates: { base_font_size?: BaseFontSize; theme_mode?: ThemeMode }) => {
      try {
        const res = await fetch('/api/system/user-settings', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            staff_id: activeStaffId || undefined,
            ...updates,
          }),
        });
        const body = await res.json().catch(() => null);
        if (res.ok && body && body.ok && body.user_settings) {
          if (typeof body.user_settings.base_font_size === 'string') {
            setBaseFontSize(normalizeBaseFontSize(body.user_settings.base_font_size));
          }
          if (
            body.user_settings.theme_mode === 'light' ||
            body.user_settings.theme_mode === 'high-contrast'
          ) {
            setThemeMode(body.user_settings.theme_mode);
          }
        }
      } catch {
        // Ignore transient network errors after optimistic state update
      }
    },
    [activeStaffId]
  );

  return {
    themeMode,
    setThemeMode,
    baseFontSize,
    setBaseFontSize,
    themeCatalog,
    setThemeCatalog,
    schoolName,
    setSchoolName,
    schoolURN,
    setSchoolURN,
    configuredAcademicYear,
    setConfiguredAcademicYear,
    persistUserSettings,
  };
}
