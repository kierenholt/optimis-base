import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  HostViewMode,
  parseBrowserRoute,
  validatePluginViewRoute,
} from '../../components/routeUtils';
import {
  BaseFontSize,
  DEFAULT_THEME_TOKENS,
  normalizeBaseFontSize,
  PinnedRecentEntry,
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginWidgetDescriptor,
  ThemeMode,
} from '../../themeAndTypes';

export interface UseWorkspaceInstalledPluginsParams {
  initialBrowserPlugId: string;
  initialAcademicYearId: string;
  initialStaffIdFromPath: string;
  activeStaffId: string;
  setActiveStaffId: (updater: (prev: string) => string) => void;
  setBaseFontSize: (size: BaseFontSize) => void;
  setThemeMode: (mode: ThemeMode) => void;
  setThemeCatalog: (catalog: Record<ThemeMode, Record<string, string>>) => void;
  setSchoolName: (name: string) => void;
  setSchoolURN: (urn: string) => void;
  setConfiguredAcademicYear: (year: string) => void;
  setPinnedRecents: (recents: PinnedRecentEntry[]) => void;
  setPlacedWidgets: (widgets: PlacedDashboardWidget[]) => void;
  setRouteError: (err: string) => void;
  clearKeepAlivePool: () => void;
  syncPluginsWithKeepAlive: (
    validIds: Set<string>,
    urlPlugId: string,
    defaultId: string,
    isInitialLoad: boolean,
    urlRouteHostViewMode: HostViewMode
  ) => void;
  applyDashboardLayoutResponse: (layoutBody: any) => void;
  hasLoadedServerLayoutRef: React.MutableRefObject<boolean>;
}

export function useWorkspaceInstalledPlugins({
  initialBrowserPlugId,
  initialAcademicYearId,
  initialStaffIdFromPath,
  activeStaffId,
  setActiveStaffId,
  setBaseFontSize,
  setThemeMode,
  setThemeCatalog,
  setSchoolName,
  setSchoolURN,
  setConfiguredAcademicYear,
  setPinnedRecents,
  setPlacedWidgets,
  setRouteError,
  clearKeepAlivePool,
  syncPluginsWithKeepAlive,
  applyDashboardLayoutResponse,
  hasLoadedServerLayoutRef,
}: UseWorkspaceInstalledPluginsParams) {
  const [installedPlugins, setInstalledPlugins] = useState<PluginNavEntry[]>([]);
  const [entityLinkHandlers, setEntityLinkHandlers] = useState<Record<string, string>>({
    student: '',
    group: '',
    staff: '',
  });
  const hasInitializedPluginsRef = useRef<boolean>(false);
  const lastFetchedStaffIdRef = useRef<string | null>(null);

  const availableWidgets = useMemo<PluginWidgetDescriptor[]>(() => {
    const list: PluginWidgetDescriptor[] = [];
    for (const p of installedPlugins) {
      if (Array.isArray(p.widgets)) {
        for (const w of p.widgets) {
          list.push({
            ...w,
            plugin_title: w.plugin_title || p.title,
          });
        }
      }
    }
    return list;
  }, [installedPlugins]);

  const applyInstalledPluginsState = useCallback(
    (plugins: PluginNavEntry[], serverEntityHandlers?: Record<string, string>) => {
      const isInitialLoad = !hasInitializedPluginsRef.current;
      hasInitializedPluginsRef.current = true;
      setInstalledPlugins(plugins);

      const urlRoute = parseBrowserRoute(window.location.pathname, window.location.search);
      const urlPlugId = urlRoute.plugId;

      if (plugins.length === 0) {
        if (urlPlugId) {
          setRouteError(`PLUGIN_NOT_FOUND: unknown plug_id ${urlPlugId}`);
        }
        clearKeepAlivePool();
        if (!hasLoadedServerLayoutRef.current) {
          setPlacedWidgets([]);
        }
        return;
      }

      const validIds = new Set(plugins.map((p) => p.id));
      if (urlPlugId) {
        const matchedUrlPlugin = plugins.find((p) => p.id === urlPlugId);
        if (!matchedUrlPlugin) {
          setRouteError(`PLUGIN_NOT_FOUND: unknown plug_id ${urlPlugId}`);
        } else {
          const check = validatePluginViewRoute(
            matchedUrlPlugin,
            urlRoute.viewPath || '/',
            urlRoute.pluginQuery
          );
          setRouteError(check.ok ? '' : check.errorCode);
        }
      }

      const defaultId =
        urlPlugId && validIds.has(urlPlugId)
          ? urlPlugId
          : initialBrowserPlugId && validIds.has(initialBrowserPlugId)
            ? initialBrowserPlugId
            : plugins[0].id;

      setEntityLinkHandlers((prev) => {
        const base = serverEntityHandlers ? { ...prev, ...serverEntityHandlers } : { ...prev };
        return {
          ...base,
          student: base.student && validIds.has(base.student) ? base.student : plugins[0].id,
          group: base.group && validIds.has(base.group) ? base.group : plugins[0].id,
          staff: base.staff && validIds.has(base.staff) ? base.staff : plugins[0].id,
        };
      });

      syncPluginsWithKeepAlive(
        validIds,
        urlPlugId,
        defaultId,
        isInitialLoad,
        urlRoute.hostViewMode
      );
    },
    [
      clearKeepAlivePool,
      hasLoadedServerLayoutRef,
      initialBrowserPlugId,
      setPlacedWidgets,
      setRouteError,
      syncPluginsWithKeepAlive,
    ]
  );

  const refreshWorkspaceForStaff = useCallback(
    async (staffIdOverride?: string): Promise<PluginNavEntry[] | undefined> => {
      const targetStaffId = staffIdOverride !== undefined ? staffIdOverride : activeStaffId;
      lastFetchedStaffIdRef.current = targetStaffId;
      try {
        const [res, layoutRes] = await Promise.all([
          fetch('/api/system/ui-config'),
          fetch('/api/dashboard/layout').catch(() => null),
        ]);

        if (layoutRes && layoutRes.ok) {
          const layoutBody = await layoutRes.json().catch(() => null);
          applyDashboardLayoutResponse(layoutBody);
        }

        if (!res.ok) {
          return undefined;
        }
        const body = await res.json();
        if (!body || !body.ok) {
          return undefined;
        }
        if (body.school_name) {
          setSchoolName(body.school_name);
        }
        if (body.school_urn) {
          setSchoolURN(body.school_urn);
        }
        if (body.academic_year_id && !initialAcademicYearId) {
          setConfiguredAcademicYear(body.academic_year_id);
        }
        if (body.themes && body.themes.light && body.themes['high-contrast']) {
          const merged: Record<ThemeMode, Record<string, string>> = {
            light: { ...DEFAULT_THEME_TOKENS.light, ...body.themes.light },
            'high-contrast': { ...DEFAULT_THEME_TOKENS['high-contrast'], ...body.themes['high-contrast'] },
          };
          setThemeCatalog(merged);
        }
        const serverEntityHandlers =
          body.entity_link_handlers && typeof body.entity_link_handlers === 'object'
            ? (body.entity_link_handlers as Record<string, string>)
            : undefined;
        if (serverEntityHandlers) {
          setEntityLinkHandlers((prev) => ({ ...prev, ...serverEntityHandlers }));
        }
        if (body.user_settings && typeof body.user_settings === 'object') {
          if (
            typeof body.user_settings.staff_id === 'string' &&
            body.user_settings.staff_id &&
            !initialStaffIdFromPath
          ) {
            lastFetchedStaffIdRef.current = body.user_settings.staff_id;
            setActiveStaffId((prev) => prev || body.user_settings.staff_id);
          }
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
        if (Array.isArray(body.pinned_recents)) {
          setPinnedRecents(body.pinned_recents as PinnedRecentEntry[]);
        }
        if (Array.isArray(body.plugins)) {
          applyInstalledPluginsState(body.plugins, serverEntityHandlers);
          return body.plugins as PluginNavEntry[];
        }
        return undefined;
      } catch {
        return undefined;
      }
    },
    [
      activeStaffId,
      initialAcademicYearId,
      initialStaffIdFromPath,
      applyDashboardLayoutResponse,
      setSchoolName,
      setSchoolURN,
      setConfiguredAcademicYear,
      setThemeCatalog,
      setActiveStaffId,
      setBaseFontSize,
      setThemeMode,
      setPinnedRecents,
      applyInstalledPluginsState,
    ]
  );

  useEffect(() => {
    if (lastFetchedStaffIdRef.current === activeStaffId && hasInitializedPluginsRef.current) {
      return;
    }
    void refreshWorkspaceForStaff(activeStaffId);
  }, [activeStaffId, refreshWorkspaceForStaff]);

  return {
    installedPlugins,
    setInstalledPlugins,
    entityLinkHandlers,
    setEntityLinkHandlers,
    availableWidgets,
    applyInstalledPluginsState,
    refreshWorkspaceForStaff,
  };
}
