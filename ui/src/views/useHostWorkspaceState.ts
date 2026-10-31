import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  HOST_ALLOWED_QUERY_KEYS,
  parseBrowserRoute,
  validatePluginViewRoute,
} from '../components/routeUtils';
import {
  ActiveCallerIdentity,
  applyHostThemeToRoot,
  DEFAULT_THEME_TOKENS,
  MAX_KEEP_ALIVE_POOL_SIZE,
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginWidgetDescriptor,
  ThemeMode,
} from '../themeAndTypes';

export function useHostWorkspaceState() {
  const initialBrowserRoute = useMemo(
    () => parseBrowserRoute(window.location.pathname, window.location.search),
    []
  );
  const params = new URLSearchParams(window.location.search);
  const initialPupilId = initialBrowserRoute.pupilIdFromPath || params.get('pupil_id') || '';
  const initialSecondPupilId = params.get('second_pupil_id') || '';
  const initialGroupId = initialBrowserRoute.groupIdFromPath || params.get('group_id') || '';
  const initialOtherGroupId = params.get('other_group_id') || '';
  const initialAbsentStaffId = params.get('absent_staff_id') || '';
  const initialCoverStaffId = params.get('cover_staff_id') || '';
  const initialSlotId = params.get('slot_id') || '';
  const initialReferenceDate = params.get('reference_date') || '';
  const initialAcademicYearId = params.get('academic_year_id') || '';
  const initialPersonaParam = params.get('__persona') || '';
  const initialStaffId =
    initialPersonaParam || initialBrowserRoute.staffIdFromPath || params.get('staff_id') || '';

  // Host Theme & School Branding state (Sections 9.4 & 9.5)
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    applyHostThemeToRoot('light', DEFAULT_THEME_TOKENS);
    return 'light';
  });
  const [themeCatalog, setThemeCatalog] = useState<Record<ThemeMode, Record<string, string>>>(DEFAULT_THEME_TOKENS);
  const [schoolName, setSchoolName] = useState<string>('Oakridge All-Through Academy');
  const [schoolURN, setSchoolURN] = useState<string>('100001');
  const [configuredAcademicYear, setConfiguredAcademicYear] = useState<string>(
    initialAcademicYearId || 'ay_2026_2027'
  );
  const [entityLinkHandlers, setEntityLinkHandlers] = useState<Record<string, string>>({
    student: '',
    group: '',
    staff: '',
  });
  const hasInitializedPluginsRef = useRef<boolean>(false);
  const [installedPlugins, setInstalledPlugins] = useState<PluginNavEntry[]>([]);

  // Cached Keep-Alive LRU Pool state (Section 9A: N=5 max mounted plugin iframes)
  const [activePluginId, setActivePluginId] = useState<string>(initialBrowserRoute.plugId);
  const [lruPluginOrder, setLruPluginOrder] = useState<string[]>(
    initialBrowserRoute.plugId ? [initialBrowserRoute.plugId] : []
  );
  const [mountedPluginIds, setMountedPluginIds] = useState<string[]>(
    initialBrowserRoute.plugId ? [initialBrowserRoute.plugId] : []
  );

  // Strict Opaque URL Routing & View State (/p/<plugid>/...)
  const [activeViewPath, setActiveViewPath] = useState<string>(initialBrowserRoute.viewPath || '/');
  const [activeTab, setActiveTab] = useState<string>(initialBrowserRoute.tab || '');
  const [activeRouteQuery, setActiveRouteQuery] = useState<Record<string, string>>(
    initialBrowserRoute.pluginQuery
  );
  const [activeRouteUrl, setActiveRouteUrl] = useState<string>(
    `${window.location.pathname}${window.location.search}`
  );
  const [routeError, setRouteError] = useState<string>(() => {
    if (!initialBrowserRoute.plugId) {
      return '';
    }
    const check = validatePluginViewRoute(
      undefined,
      initialBrowserRoute.viewPath || '/',
      initialBrowserRoute.pluginQuery
    );
    return check.ok ? '' : check.errorCode;
  });

  // Editable NxM Mini-Iframe Widget Grid state (Section 9B)
  const [isDashboardEditMode, setIsDashboardEditMode] = useState<boolean>(false);
  const [placedWidgets, setPlacedWidgets] = useState<PlacedDashboardWidget[]>([]);

  // Active Entity Deep-Linking Context state (Section 9C)
  const [activePupilId, setActivePupilId] = useState<string>(initialPupilId);
  const [activeGroupId, setActiveGroupId] = useState<string>(initialGroupId);
  const [activeStaffId, setActiveStaffId] = useState<string>(initialStaffId);
  const [activeCaller, setActiveCaller] = useState<ActiveCallerIdentity | null>(null);
  const [activeEntityContext, setActiveEntityContext] = useState<string>(
    initialPupilId ? `student:${initialPupilId}` : initialGroupId ? `group:${initialGroupId}` : 'dashboard:home'
  );
  const [deepLinkNavigationStatus, setDeepLinkNavigationStatus] = useState<string>('IDLE');

  const availableWidgets = useMemo<PluginWidgetDescriptor[]>(() => {
    const list: PluginWidgetDescriptor[] = [];
    for (const p of installedPlugins) {
      if (Array.isArray(p.widgets)) {
        for (const w of p.widgets) {
          list.push(w);
        }
      }
    }
    return list;
  }, [installedPlugins]);

  const syncBrowserRouteUrl = useCallback(
    (
      nextPlugId: string,
      nextViewPath: string,
      nextPluginQuery: Record<string, string>,
      replaceHistory = false
    ): string => {
      if (!nextPlugId) {
        return `${window.location.pathname}${window.location.search}`;
      }
      const cleanSubpath =
        !nextViewPath || nextViewPath === '/'
          ? ''
          : nextViewPath.startsWith('/')
            ? nextViewPath
            : `/${nextViewPath}`;
      const pathname = `/p/${encodeURIComponent(nextPlugId)}${cleanSubpath}`;

      const mergedSearch = new URLSearchParams();
      for (const [k, v] of Object.entries(nextPluginQuery)) {
        if (v !== undefined && v !== null && String(v) !== '') {
          mergedSearch.set(k, String(v));
        }
      }
      const existingSearch = new URLSearchParams(window.location.search);
      existingSearch.forEach((val, key) => {
        if (HOST_ALLOWED_QUERY_KEYS.has(key) && val !== '') {
          if (key === 'pupil_id' && cleanSubpath.startsWith('/student/')) {
            return;
          }
          if (key === 'group_id' && cleanSubpath.startsWith('/group/')) {
            return;
          }
          mergedSearch.set(key, val);
        }
      });

      const qs = mergedSearch.toString();
      const fullUrl = qs ? `${pathname}?${qs}` : pathname;
      if (`${window.location.pathname}${window.location.search}` !== fullUrl) {
        if (replaceHistory) {
          window.history.replaceState(null, '', fullUrl);
        } else {
          window.history.pushState(null, '', fullUrl);
        }
      }
      setActiveRouteUrl(fullUrl);
      return fullUrl;
    },
    []
  );

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
        setActivePluginId('');
        setLruPluginOrder([]);
        setMountedPluginIds([]);
        setPlacedWidgets([]);
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
          : initialBrowserRoute.plugId && validIds.has(initialBrowserRoute.plugId)
            ? initialBrowserRoute.plugId
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

      setActivePluginId((prev) => {
        if (urlPlugId && validIds.has(urlPlugId)) {
          return urlPlugId;
        }
        return prev && validIds.has(prev) ? prev : defaultId;
      });
      setLruPluginOrder((prev) => {
        const filtered = prev.filter((id) => validIds.has(id));
        if (urlPlugId && validIds.has(urlPlugId) && !filtered.includes(urlPlugId)) {
          return [...filtered, urlPlugId];
        }
        return filtered.length > 0 ? filtered : [defaultId];
      });
      setMountedPluginIds((prev) => {
        const filtered = prev.filter((id) => validIds.has(id));
        if (urlPlugId && validIds.has(urlPlugId) && !filtered.includes(urlPlugId)) {
          return [...filtered, urlPlugId];
        }
        return filtered.length > 0 ? filtered : [defaultId];
      });

      if (isInitialLoad) {
        const declaredWidgets: PluginWidgetDescriptor[] = [];
        for (const p of plugins) {
          if (Array.isArray(p.widgets)) {
            for (const w of p.widgets) {
              declaredWidgets.push(w);
            }
          }
        }
        setPlacedWidgets((prev) => {
          const filtered = prev.filter((w) => validIds.has(w.plugin_id));
          if (filtered.length > 0) {
            return filtered;
          }
          return declaredWidgets.length > 0 ? [{ ...declaredWidgets[0], colSpan: 2, rowSpan: 1 }] : [];
        });
      } else {
        setPlacedWidgets((prev) => prev.filter((w) => validIds.has(w.plugin_id)));
      }
    },
    [initialBrowserRoute.plugId]
  );

  const activatePluginInKeepAlivePool = useCallback(
    (targetPluginId: string, overrideViewPath?: string, overridePluginQuery?: Record<string, string>) => {
      setActivePluginId(targetPluginId);
      setLruPluginOrder((prevOrder) => {
        const nextOrder = [...prevOrder.filter((id) => id !== targetPluginId), targetPluginId];
        let evictedId: string | null = null;
        if (nextOrder.length > MAX_KEEP_ALIVE_POOL_SIZE) {
          evictedId = nextOrder[0];
          nextOrder.shift();
        }
        setMountedPluginIds((prevMounted) => {
          const nextMounted = evictedId ? prevMounted.filter((id) => id !== evictedId) : [...prevMounted];
          if (!nextMounted.includes(targetPluginId)) {
            nextMounted.push(targetPluginId);
          }
          return nextMounted;
        });
        return nextOrder;
      });

      const resolvedPath = overrideViewPath !== undefined ? overrideViewPath : activeViewPath;
      const resolvedQuery = overridePluginQuery !== undefined ? overridePluginQuery : activeRouteQuery;
      syncBrowserRouteUrl(targetPluginId, resolvedPath, resolvedQuery);
    },
    [activeViewPath, activeRouteQuery, syncBrowserRouteUrl]
  );

  useEffect(() => {
    const loadUIConfig = async () => {
      try {
        const query = activeStaffId ? `?staff_id=${encodeURIComponent(activeStaffId)}` : '';
        const res = await fetch(`/api/system/ui-config${query}`);
        if (!res.ok) {
          return;
        }
        const body = await res.json();
        if (!body || !body.ok) {
          return;
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
        if (Array.isArray(body.plugins)) {
          applyInstalledPluginsState(body.plugins, serverEntityHandlers);
        }
      } catch {
        // Keep default tokens if ui-config endpoint is unreachable
      }
    };
    void loadUIConfig();
  }, [initialAcademicYearId, activeStaffId, applyInstalledPluginsState]);

  useEffect(() => {
    applyHostThemeToRoot(themeMode, themeCatalog);
  }, [themeMode, themeCatalog]);

  const handlePlaceWidget = (widget: PluginWidgetDescriptor) => {
    setPlacedWidgets((prev) => {
      if (prev.some((w) => w.id === widget.id)) {
        return prev;
      }
      return [...prev, { ...widget, colSpan: 2, rowSpan: 1 }];
    });
  };

  const handleResizeWidget = (widgetId: string, colSpan: number, rowSpan: number) => {
    const safeCols = Math.max(1, Math.min(4, colSpan));
    const safeRows = Math.max(1, Math.min(4, rowSpan));
    setPlacedWidgets((prev) =>
      prev.map((w) => (w.id === widgetId ? { ...w, colSpan: safeCols, rowSpan: safeRows } : w))
    );
  };

  const handleRemoveWidget = (widgetId: string) => {
    setPlacedWidgets((prev) => prev.filter((w) => w.id !== widgetId));
  };

  return {
    initialSecondPupilId,
    initialOtherGroupId,
    initialAbsentStaffId,
    initialCoverStaffId,
    initialSlotId,
    initialReferenceDate,
    initialAcademicYearId,
    initialPersonaParam,
    themeMode,
    setThemeMode,
    themeCatalog,
    schoolName,
    schoolURN,
    configuredAcademicYear,
    entityLinkHandlers,
    installedPlugins,
    activePluginId,
    setActivePluginId,
    lruPluginOrder,
    mountedPluginIds,
    activeViewPath,
    setActiveViewPath,
    activeTab,
    setActiveTab,
    activeRouteQuery,
    setActiveRouteQuery,
    activeRouteUrl,
    setActiveRouteUrl,
    routeError,
    setRouteError,
    isDashboardEditMode,
    setIsDashboardEditMode,
    placedWidgets,
    availableWidgets,
    activePupilId,
    setActivePupilId,
    activeGroupId,
    setActiveGroupId,
    activeStaffId,
    setActiveStaffId,
    activeCaller,
    setActiveCaller,
    activeEntityContext,
    setActiveEntityContext,
    deepLinkNavigationStatus,
    setDeepLinkNavigationStatus,
    syncBrowserRouteUrl,
    applyInstalledPluginsState,
    activatePluginInKeepAlivePool,
    handlePlaceWidget,
    handleResizeWidget,
    handleRemoveWidget,
  };
}
