import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  HOST_ALLOWED_QUERY_KEYS,
  parseBrowserRoute,
  validatePluginViewRoute,
} from '../components/routeUtils';
import {
  ActiveCallerIdentity,
  applyHostThemeToRoot,
  DashboardLayoutItem,
  DEFAULT_THEME_TOKENS,
  MAX_KEEP_ALIVE_POOL_SIZE,
  PinnedRecentEntry,
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginWidgetDescriptor,
  ResolvedDashboardWidgetSlot,
  ThemeMode,
} from '../themeAndTypes';

export function useHostWorkspaceState() {
  const initialBrowserRoute = useMemo(
    () => parseBrowserRoute(window.location.pathname, window.location.search),
    []
  );
  const params = useMemo(() => new URLSearchParams(window.location.search), []);
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
  const [pinnedRecents, setPinnedRecents] = useState<PinnedRecentEntry[]>([]);

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

  // Editable NxM Mini-Iframe Widget Grid & 1D core.db Persistence state (Section 9B)
  const [isDashboardEditMode, setIsDashboardEditMode] = useState<boolean>(false);
  const [placedWidgets, setPlacedWidgets] = useState<PlacedDashboardWidget[]>([]);
  const [dashboardLayoutSource, setDashboardLayoutSource] = useState<string>('SYSTEM_FALLBACK');
  const [dashboardSaveStatus, setDashboardSaveStatus] = useState<string>('');
  const [roleDefaultLayouts, setRoleDefaultLayouts] = useState<Record<string, DashboardLayoutItem[]>>({});
  const [selectedAdminRoleTarget, setSelectedAdminRoleTarget] = useState<string>('TEACHER');
  const hasLoadedServerLayoutRef = useRef<boolean>(false);

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

  const normalizePresetDim = (val: number | undefined, fallback: 1 | 2): 1 | 2 =>
    val === 1 || val === 2 ? val : fallback;

  const mapResolvedSlotsToPlacedWidgets = useCallback(
    (
      slots: ResolvedDashboardWidgetSlot[],
      _existingWidgets?: PlacedDashboardWidget[]
    ): PlacedDashboardWidget[] => {
      return slots.map((slot) => {
        const presetWidth = normalizePresetDim(slot.width, 2);
        const presetHeight = normalizePresetDim(slot.height, 1);
        return {
          id: slot.widget_id,
          plugin_id: slot.plug_id,
          title: slot.title || slot.widget_id,
          entry: slot.entry || '',
          entry_url: slot.entry_url || '',
          width: presetWidth,
          height: presetHeight,
          colSpan: presetWidth,
          rowSpan: presetHeight,
          permitted: slot.permitted !== false,
          error_code: slot.error_code,
          error_title: slot.error_title,
        };
      });
    },
    []
  );

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
        if (!isInitialLoad && !urlPlugId && prev === '') {
          return '';
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

  const lastFetchedStaffIdRef = useRef<string | null>(null);

  const refreshDashboardLayoutForStaff = useCallback(
    async (targetStaffId: string) => {
      try {
        const query = targetStaffId ? `?staff_id=${encodeURIComponent(targetStaffId)}` : '';
        const layoutRes = await fetch(`/api/dashboard/layout${query}`);
        if (!layoutRes.ok) {
          return;
        }
        const layoutBody = await layoutRes.json().catch(() => null);
        if (layoutBody && layoutBody.ok && Array.isArray(layoutBody.resolved_widgets)) {
          hasLoadedServerLayoutRef.current = true;
          const slots = layoutBody.resolved_widgets as ResolvedDashboardWidgetSlot[];
          setPlacedWidgets((prev) => mapResolvedSlotsToPlacedWidgets(slots, prev));
          if (typeof layoutBody.layout_source === 'string') {
            setDashboardLayoutSource(layoutBody.layout_source);
          }
          if (layoutBody.role_defaults && typeof layoutBody.role_defaults === 'object') {
            setRoleDefaultLayouts(
              layoutBody.role_defaults as Record<string, DashboardLayoutItem[]>
            );
          }
        }
      } catch {
        // Ignore network errors during dashboard layout refresh
      }
    },
    [mapResolvedSlotsToPlacedWidgets]
  );

  const refreshWorkspaceForStaff = useCallback(
    async (staffIdOverride?: string): Promise<PluginNavEntry[] | undefined> => {
      const targetStaffId = staffIdOverride !== undefined ? staffIdOverride : activeStaffId;
      lastFetchedStaffIdRef.current = targetStaffId;
      try {
        const query = targetStaffId ? `?staff_id=${encodeURIComponent(targetStaffId)}` : '';
        const [res, layoutRes] = await Promise.all([
          fetch(`/api/system/ui-config${query}`),
          fetch(`/api/dashboard/layout${query}`).catch(() => null),
        ]);

        if (layoutRes && layoutRes.ok) {
          const layoutBody = await layoutRes.json().catch(() => null);
          if (layoutBody && layoutBody.ok && Array.isArray(layoutBody.resolved_widgets)) {
            hasLoadedServerLayoutRef.current = true;
            const slots = layoutBody.resolved_widgets as ResolvedDashboardWidgetSlot[];
            setPlacedWidgets((prev) => mapResolvedSlotsToPlacedWidgets(slots, prev));
            if (typeof layoutBody.layout_source === 'string') {
              setDashboardLayoutSource(layoutBody.layout_source);
            }
            if (layoutBody.role_defaults && typeof layoutBody.role_defaults === 'object') {
              setRoleDefaultLayouts(
                layoutBody.role_defaults as Record<string, DashboardLayoutItem[]>
              );
            }
          }
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
    [activeStaffId, initialAcademicYearId, applyInstalledPluginsState, mapResolvedSlotsToPlacedWidgets]
  );

  useEffect(() => {
    if (lastFetchedStaffIdRef.current === activeStaffId && hasInitializedPluginsRef.current) {
      return;
    }
    void refreshWorkspaceForStaff(activeStaffId);
  }, [activeStaffId, refreshWorkspaceForStaff]);

  useEffect(() => {
    applyHostThemeToRoot(themeMode, themeCatalog);
  }, [themeMode, themeCatalog]);

  const handleNavigateHome = useCallback(() => {
    setActivePluginId('');
    setActiveViewPath('/');
    setActiveRouteQuery({});
    setRouteError('');
    if (window.location.pathname !== '/' || window.location.search !== '') {
      window.history.pushState(null, '', '/');
    }
    setActiveRouteUrl('/');
  }, []);

  const handlePinOrUnpinRoute = useCallback(
    async (route: string, title: string, viewId?: string) => {
      const trimmedRoute = route.trim();
      const trimmedTitle = title.trim();
      if (!trimmedRoute || !trimmedTitle) {
        throw new Error('Missing mandatory route or title for pinning view');
      }
      const isAlreadyPinned = pinnedRecents.some((r) => r.route === trimmedRoute);
      const action = isAlreadyPinned ? 'unpin' : 'pin';
      const res = await fetch('/api/system/pinned-recents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action,
          route: trimmedRoute,
          title: trimmedTitle,
          view_id: viewId || undefined,
          staff_id: activeStaffId || undefined,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        throw new Error(body?.error_code || 'Failed to update pinned recents');
      }
      if (Array.isArray(body.pinned_recents)) {
        setPinnedRecents(body.pinned_recents as PinnedRecentEntry[]);
      }
    },
    [activeStaffId, pinnedRecents]
  );

  const handleUnpinRoute = useCallback(
    async (route: string) => {
      const trimmedRoute = route.trim();
      if (!trimmedRoute) {
        throw new Error('Missing mandatory route for unpinning view');
      }
      const res = await fetch('/api/system/pinned-recents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'unpin',
          route: trimmedRoute,
          staff_id: activeStaffId || undefined,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        throw new Error(body?.error_code || 'Failed to unpin route');
      }
      if (Array.isArray(body.pinned_recents)) {
        setPinnedRecents(body.pinned_recents as PinnedRecentEntry[]);
      }
    },
    [activeStaffId]
  );

  const handleSelectPinnedRoute = useCallback(
    (item: PinnedRecentEntry) => {
      const qIdx = item.route.indexOf('?');
      const pathname = qIdx >= 0 ? item.route.slice(0, qIdx) : item.route;
      const search = qIdx >= 0 ? item.route.slice(qIdx) : '';
      const parsed = parseBrowserRoute(pathname, search);
      if (!parsed.plugId) {
        return;
      }
      if (parsed.pupilIdFromPath) {
        setActivePupilId(parsed.pupilIdFromPath);
        setActiveEntityContext(`student:${parsed.pupilIdFromPath}`);
      } else if (parsed.groupIdFromPath) {
        setActiveGroupId(parsed.groupIdFromPath);
        setActiveEntityContext(`group:${parsed.groupIdFromPath}`);
      }
      const nextViewPath = parsed.viewPath || '/';
      setActiveViewPath(nextViewPath);
      setActiveRouteQuery(parsed.pluginQuery);
      setActiveTab(parsed.tab || '');
      setRouteError('');
      activatePluginInKeepAlivePool(parsed.plugId, nextViewPath, parsed.pluginQuery);
    },
    [activatePluginInKeepAlivePool]
  );

  const persistStaffDashboardLayout = useCallback(
    async (nextWidgets: PlacedDashboardWidget[]) => {
      const layoutPayload: DashboardLayoutItem[] = nextWidgets.map((w) => ({
        plug_id: w.plugin_id,
        widget_id: w.id,
      }));
      try {
        const res = await fetch('/api/dashboard/layout', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            staff_id: activeStaffId || undefined,
            layout: layoutPayload,
          }),
        });
        const body = await res.json().catch(() => null);
        if (res.ok && body && body.ok) {
          hasLoadedServerLayoutRef.current = true;
          if (Array.isArray(body.resolved_widgets)) {
            setPlacedWidgets(
              mapResolvedSlotsToPlacedWidgets(
                body.resolved_widgets as ResolvedDashboardWidgetSlot[],
                nextWidgets
              )
            );
          } else {
            setPlacedWidgets(nextWidgets);
          }
          setDashboardLayoutSource('STAFF');
          setDashboardSaveStatus('STAFF_LAYOUT_SAVED');
          return;
        }
      } catch {
        // Fall through to local state update if network request fails
      }
      setPlacedWidgets(nextWidgets);
    },
    [activeStaffId, mapResolvedSlotsToPlacedWidgets]
  );

  const handlePlaceWidget = (widget: PluginWidgetDescriptor) => {
    if (placedWidgets.some((w) => w.plugin_id === widget.plugin_id && w.id === widget.id)) {
      return;
    }
    const presetWidth = normalizePresetDim(widget.width, 2);
    const presetHeight = normalizePresetDim(widget.height, 1);
    const nextWidgets: PlacedDashboardWidget[] = [
      ...placedWidgets,
      {
        ...widget,
        width: presetWidth,
        height: presetHeight,
        colSpan: presetWidth,
        rowSpan: presetHeight,
        permitted: true,
      },
    ];
    void persistStaffDashboardLayout(nextWidgets);
  };

  const handleRemoveWidget = (pluginId: string, widgetId: string) => {
    const nextWidgets = placedWidgets.filter(
      (w) => !(w.plugin_id === pluginId && w.id === widgetId)
    );
    void persistStaffDashboardLayout(nextWidgets);
  };

  const handleMoveWidgetUp = (pluginId: string, widgetId: string) => {
    const idx = placedWidgets.findIndex(
      (w) => w.plugin_id === pluginId && w.id === widgetId
    );
    if (idx <= 0) {
      return;
    }
    const nextWidgets = [...placedWidgets];
    const tmp = nextWidgets[idx - 1];
    nextWidgets[idx - 1] = nextWidgets[idx];
    nextWidgets[idx] = tmp;
    void persistStaffDashboardLayout(nextWidgets);
  };

  const handleMoveWidgetDown = (pluginId: string, widgetId: string) => {
    const idx = placedWidgets.findIndex(
      (w) => w.plugin_id === pluginId && w.id === widgetId
    );
    if (idx < 0 || idx >= placedWidgets.length - 1) {
      return;
    }
    const nextWidgets = [...placedWidgets];
    const tmp = nextWidgets[idx + 1];
    nextWidgets[idx + 1] = nextWidgets[idx];
    nextWidgets[idx] = tmp;
    void persistStaffDashboardLayout(nextWidgets);
  };

  const handleSaveRoleDefaultLayout = async (roleId: string) => {
    const normalizedRole = roleId.trim().toUpperCase();
    if (!normalizedRole) {
      throw new Error('Missing mandatory role_id for saving role default dashboard layout');
    }
    const layoutPayload: DashboardLayoutItem[] = placedWidgets.map((w) => ({
      plug_id: w.plugin_id,
      widget_id: w.id,
    }));
    const res = await fetch('/api/admin/dashboard/role-default', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        staff_id: activeStaffId || undefined,
        role_id: normalizedRole,
        layout: layoutPayload,
        append_to_existing_staff: true,
      }),
    });
    const body = await res.json().catch(() => null);
    if (!res.ok || !body || !body.ok) {
      setDashboardSaveStatus(`ERROR:${body?.error_code || res.status}`);
      return;
    }
    if (body.role_defaults && typeof body.role_defaults === 'object') {
      setRoleDefaultLayouts(body.role_defaults as Record<string, DashboardLayoutItem[]>);
    }
    setDashboardSaveStatus(`ROLE_DEFAULT_SAVED:${normalizedRole}`);
    await refreshDashboardLayoutForStaff(activeStaffId);
  };

  const handleAppendWidgetToRoleDefault = async (roleId: string, widget: PluginWidgetDescriptor) => {
    const normalizedRole = roleId.trim().toUpperCase();
    if (!normalizedRole || !widget.plugin_id || !widget.id) {
      throw new Error('Missing mandatory role_id, plug_id, or widget_id for role default append');
    }
    const res = await fetch('/api/admin/dashboard/role-default/append', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        staff_id: activeStaffId || undefined,
        role_id: normalizedRole,
        plug_id: widget.plugin_id,
        widget_id: widget.id,
      }),
    });
    const body = await res.json().catch(() => null);
    if (!res.ok || !body || !body.ok) {
      setDashboardSaveStatus(`ERROR:${body?.error_code || res.status}`);
      return;
    }
    if (body.role_defaults && typeof body.role_defaults === 'object') {
      setRoleDefaultLayouts(body.role_defaults as Record<string, DashboardLayoutItem[]>);
    }
    setDashboardSaveStatus(`ROLE_WIDGET_APPENDED:${normalizedRole}:${widget.id}`);
    await refreshDashboardLayoutForStaff(activeStaffId);
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
    pinnedRecents,
    setPinnedRecents,
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
    dashboardLayoutSource,
    dashboardSaveStatus,
    roleDefaultLayouts,
    selectedAdminRoleTarget,
    setSelectedAdminRoleTarget,
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
    refreshWorkspaceForStaff,
    activatePluginInKeepAlivePool,
    handleNavigateHome,
    handlePinOrUnpinRoute,
    handleUnpinRoute,
    handleSelectPinnedRoute,
    handlePlaceWidget,
    handleRemoveWidget,
    handleMoveWidgetUp,
    handleMoveWidgetDown,
    handleSaveRoleDefaultLayout,
    handleAppendWidgetToRoleDefault,
  };
}
