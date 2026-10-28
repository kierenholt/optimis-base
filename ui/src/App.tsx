import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AdminDiagnosticsPanel } from './AdminDiagnosticsPanel';
import { DashboardWidgetGridPanel } from './DashboardWidgetGridPanel';
import { GdprSarPanel } from './GdprSarPanel';
import { SafeguardingVaultPanel } from './SafeguardingVaultPanel';
import {
  ActiveCallerIdentity,
  applyHostThemeToRoot,
  DecryptedSafeguardingNote,
  DEFAULT_PERMITTED_METHODS,
  DEFAULT_PLUGIN_WIDGETS,
  DEFAULT_THEME_TOKENS,
  MAX_KEEP_ALIVE_POOL_SIZE,
  PlacedDashboardWidget,
  PLUGIN_ID,
  PluginNavEntry,
  PluginWidgetDescriptor,
  SafeguardingNoteEnvelope,
  SARSection,
  StaffPersonaSummary,
  ThemeMode,
} from './themeAndTypes';

function checkCallerAdminOrDpo(roles: string[], scopes: string[]): boolean {
  if (roles.some((r) => r.toUpperCase() === 'ADMIN' || r.toUpperCase() === 'DPO')) {
    return true;
  }
  return scopes.includes('dal.staff.write:admin');
}

export function App() {
  const pluginIframeRefs = useRef<Map<string, HTMLIFrameElement>>(new Map());
  const widgetIframeRefs = useRef<Map<string, { pluginId: string; el: HTMLIFrameElement }>>(new Map());

  const params = new URLSearchParams(window.location.search);
  const initialPupilId = params.get('pupil_id') || '';
  const initialSecondPupilId = params.get('second_pupil_id') || '';
  const initialGroupId = params.get('group_id') || '';
  const initialOtherGroupId = params.get('other_group_id') || '';
  const initialReferenceDate = params.get('reference_date') || '';
  const initialAcademicYearId = params.get('academic_year_id') || '';
  const initialStaffId = params.get('staff_id') || '';

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
    student: PLUGIN_ID,
    group: PLUGIN_ID,
    staff: PLUGIN_ID,
  });
  const hasInitializedPluginsRef = useRef<boolean>(false);
  const [installedPlugins, setInstalledPlugins] = useState<PluginNavEntry[]>([]);

  // Cached Keep-Alive LRU Pool state (Section 9A: N=5 max mounted plugin iframes)
  const [activePluginId, setActivePluginId] = useState<string>('');
  const [lruPluginOrder, setLruPluginOrder] = useState<string[]>([]);
  const [mountedPluginIds, setMountedPluginIds] = useState<string[]>([]);

  // Editable NxM Mini-Iframe Widget Grid state (Section 9B)
  const [isDashboardEditMode, setIsDashboardEditMode] = useState<boolean>(false);
  const [placedWidgets, setPlacedWidgets] = useState<PlacedDashboardWidget[]>([]);

  // Active Entity Deep-Linking Context state (Section 9C)
  const [activePupilId, setActivePupilId] = useState<string>(initialPupilId);
  const [activeGroupId, setActiveGroupId] = useState<string>(initialGroupId);
  const [activeStaffId, setActiveStaffId] = useState<string>(initialStaffId);
  const [activeEntityContext, setActiveEntityContext] = useState<string>(
    initialPupilId ? `student:${initialPupilId}` : initialGroupId ? `group:${initialGroupId}` : 'dashboard:home'
  );
  const [deepLinkNavigationStatus, setDeepLinkNavigationStatus] = useState<string>('IDLE');

  // In-UI Topbar Persona Switcher state (Section 8A)
  const [devPersonaSwitcherEnabled, setDevPersonaSwitcherEnabled] = useState<boolean>(false);
  const [availablePersonas, setAvailablePersonas] = useState<StaffPersonaSummary[]>([]);
  const [activeCaller, setActiveCaller] = useState<ActiveCallerIdentity | null>(null);
  const [personaSwitchError, setPersonaSwitchError] = useState<string>('');

  // DSL Safeguarding Vault state (Section 5B)
  const [hasSafeguardingRead, setHasSafeguardingRead] = useState<boolean>(false);
  const [safeguardingEnvelopes, setSafeguardingEnvelopes] = useState<SafeguardingNoteEnvelope[]>([]);
  const [cloudKeyToken, setCloudKeyToken] = useState<string>('');
  const [decryptedNotes, setDecryptedNotes] = useState<Record<string, DecryptedSafeguardingNote>>({});
  const [dslVaultError, setDslVaultError] = useState<string>('');

  // UK GDPR SAR Redaction & Retention Purge state (Sections 5C, 7B & 9.3)
  const [hasAdminOrDpoAccess, setHasAdminOrDpoAccess] = useState<boolean>(false);
  const [sarSections, setSarSections] = useState<SARSection[]>([]);
  const [sarExportStatus, setSarExportStatus] = useState<string>('');
  const [sarFinalizedJson, setSarFinalizedJson] = useState<string>('');
  const [sarPurgeStatus, setSarPurgeStatus] = useState<string>('');

  // Day-2 Operations: Admin-Exported PII-Scrubbed Diagnostic Bundle state (Section 8 Gap 1)
  const [diagnosticBundleStatus, setDiagnosticBundleStatus] = useState<string>('');
  const [diagnosticBundleJson, setDiagnosticBundleJson] = useState<string>('');
  const [diagnosticBundleLog, setDiagnosticBundleLog] = useState<string>('');

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

  const applyInstalledPluginsState = useCallback((plugins: PluginNavEntry[]) => {
    const isInitialLoad = !hasInitializedPluginsRef.current;
    hasInitializedPluginsRef.current = true;
    setInstalledPlugins(plugins);

    if (plugins.length === 0) {
      setActivePluginId('');
      setLruPluginOrder([]);
      setMountedPluginIds([]);
      setPlacedWidgets([]);
      return;
    }

    const validIds = new Set(plugins.map((p) => p.id));
    const defaultId = validIds.has(PLUGIN_ID) ? PLUGIN_ID : plugins[0].id;
    setActivePluginId((prev) => (prev && validIds.has(prev) ? prev : defaultId));
    setLruPluginOrder((prev) => {
      const filtered = prev.filter((id) => validIds.has(id));
      return filtered.length > 0 ? filtered : [defaultId];
    });
    setMountedPluginIds((prev) => {
      const filtered = prev.filter((id) => validIds.has(id));
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
  }, []);

  const activatePluginInKeepAlivePool = useCallback((targetPluginId: string) => {
    setActivePluginId(targetPluginId);
    setLruPluginOrder((prevOrder) => {
      const nextOrder = [...prevOrder.filter((id) => id !== targetPluginId), targetPluginId];
      let evictedId: string | null = null;
      if (nextOrder.length > MAX_KEEP_ALIVE_POOL_SIZE) {
        evictedId = nextOrder[0];
        nextOrder.shift();
      }
      setMountedPluginIds((prevMounted) => {
        let nextMounted = evictedId ? prevMounted.filter((id) => id !== evictedId) : [...prevMounted];
        if (!nextMounted.includes(targetPluginId)) {
          nextMounted.push(targetPluginId);
        }
        return nextMounted;
      });
      return nextOrder;
    });
  }, []);

  const handleRegisterPluginIframeRef = useCallback((pluginId: string, el: HTMLIFrameElement | null) => {
    if (el) {
      pluginIframeRefs.current.set(pluginId, el);
    } else {
      pluginIframeRefs.current.delete(pluginId);
    }
  }, []);

  const handleRegisterWidgetIframeRef = useCallback(
    (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => {
      if (el) {
        widgetIframeRefs.current.set(widgetId, { pluginId, el });
      } else {
        widgetIframeRefs.current.delete(widgetId);
      }
    },
    []
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
        if (body.entity_link_handlers && typeof body.entity_link_handlers === 'object') {
          setEntityLinkHandlers((prev) => ({ ...prev, ...body.entity_link_handlers }));
        }
        if (Array.isArray(body.plugins)) {
          applyInstalledPluginsState(body.plugins);
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

  const buildInitPayload = useCallback(
    (
      targetPluginId: string,
      overrideContext?: Partial<{
        pupil_id: string;
        group_id: string;
        staff_id: string;
        entity_type: string;
        entity_id: string;
      }>,
      overrideThemeMode?: ThemeMode,
      overridePlugins?: PluginNavEntry[],
      overrideCaller?: ActiveCallerIdentity | null
    ) => {
      const mode = overrideThemeMode || themeMode;
      const activeTokens = themeCatalog[mode] || DEFAULT_THEME_TOKENS[mode];
      const effectivePlugins = overridePlugins || installedPlugins;
      const effectiveCaller = overrideCaller !== undefined ? overrideCaller : activeCaller;
      const resolvedPupilId = overrideContext?.pupil_id !== undefined ? overrideContext.pupil_id : activePupilId;
      const resolvedGroupId = overrideContext?.group_id !== undefined ? overrideContext.group_id : activeGroupId;
      const resolvedStaffId =
        overrideContext?.staff_id !== undefined
          ? overrideContext.staff_id
          : activeStaffId || effectiveCaller?.staff_id || '';
      const pluginDescriptor = effectivePlugins.find((p) => p.id === targetPluginId);
      const permittedMethods = pluginDescriptor
        ? pluginDescriptor.permitted_methods
        : DEFAULT_PERMITTED_METHODS;

      return {
        type: 'mis:init',
        theme_mode: mode,
        theme_tokens: activeTokens,
        theme: {
          primary: activeTokens['--mis-brand-primary'],
        },
        context: {
          pupil_id: resolvedPupilId || null,
          second_pupil_id: initialSecondPupilId || null,
          group_id: resolvedGroupId || null,
          other_group_id: initialOtherGroupId || null,
          reference_date: initialReferenceDate || null,
          academic_year_id: initialAcademicYearId || configuredAcademicYear,
          staff_id: resolvedStaffId || null,
          entity_type:
            overrideContext?.entity_type || (resolvedPupilId ? 'student' : resolvedGroupId ? 'group' : 'dashboard'),
          entity_id: overrideContext?.entity_id || resolvedPupilId || resolvedGroupId || 'home',
        },
        caller: {
          staff_id: resolvedStaffId || null,
          user_id: effectiveCaller?.user_id || null,
          active_roles: effectiveCaller?.active_roles || [],
          scopes: effectiveCaller?.scopes || [],
        },
        scopes: effectiveCaller?.scopes || [],
        active_roles: effectiveCaller?.active_roles || [],
        permitted_methods: permittedMethods,
      };
    },
    [
      themeMode,
      themeCatalog,
      activePupilId,
      initialSecondPupilId,
      activeGroupId,
      initialOtherGroupId,
      initialReferenceDate,
      initialAcademicYearId,
      configuredAcademicYear,
      activeStaffId,
      installedPlugins,
      activeCaller,
    ]
  );

  const broadcastInitToMountedIframes = useCallback(
    (
      overrideContext?: Partial<{
        pupil_id: string;
        group_id: string;
        staff_id: string;
        entity_type: string;
        entity_id: string;
      }>,
      overrideThemeMode?: ThemeMode,
      overridePlugins?: PluginNavEntry[],
      overrideCaller?: ActiveCallerIdentity | null
    ) => {
      for (const [pluginId, iframeEl] of pluginIframeRefs.current.entries()) {
        if (iframeEl?.contentWindow) {
          iframeEl.contentWindow.postMessage(
            buildInitPayload(pluginId, overrideContext, overrideThemeMode, overridePlugins, overrideCaller),
            '*'
          );
        }
      }
      for (const [, entry] of widgetIframeRefs.current.entries()) {
        if (entry.el?.contentWindow) {
          entry.el.contentWindow.postMessage(
            buildInitPayload(entry.pluginId, overrideContext, overrideThemeMode, overridePlugins, overrideCaller),
            '*'
          );
        }
      }
    },
    [buildInitPayload]
  );

  useEffect(() => {
    broadcastInitToMountedIframes();
  }, [broadcastInitToMountedIframes, mountedPluginIds, placedWidgets]);

  useEffect(() => {
    const loadPersonas = async () => {
      try {
        const res = await fetch('/api/auth/personas');
        if (!res.ok) {
          setDevPersonaSwitcherEnabled(false);
          setAvailablePersonas([]);
          return;
        }
        const body = await res.json();
        if (body && body.ok && Array.isArray(body.personas)) {
          setDevPersonaSwitcherEnabled(true);
          setAvailablePersonas(body.personas);
        } else {
          setDevPersonaSwitcherEnabled(false);
          setAvailablePersonas([]);
        }
      } catch {
        setDevPersonaSwitcherEnabled(false);
        setAvailablePersonas([]);
      }
    };
    void loadPersonas();
  }, []);

  useEffect(() => {
    const loadSessionAndSafeguarding = async () => {
      try {
        const sessQuery = activeStaffId ? `?staff_id=${encodeURIComponent(activeStaffId)}` : '';
        const sessRes = await fetch(`/api/auth/session${sessQuery}`);
        if (!sessRes.ok) {
          return;
        }
        const sessBody = await sessRes.json();
        const caller = sessBody?.caller;
        const scopes: string[] = Array.isArray(caller?.scopes) ? caller.scopes : [];
        const roles: string[] = Array.isArray(caller?.active_roles) ? caller.active_roles : [];
        if (caller && typeof caller.staff_id === 'string') {
          setActiveCaller({
            staff_id: caller.staff_id,
            user_id: caller.user_id || '',
            active_roles: roles,
            scopes,
          });
        }
        setHasAdminOrDpoAccess(checkCallerAdminOrDpo(roles, scopes));

        if (scopes.includes('dal.safeguarding.read')) {
          setHasSafeguardingRead(true);
          const query = activePupilId ? `?pupil_id=${encodeURIComponent(activePupilId)}` : '';
          const notesRes = await fetch(`/api/safeguarding/notes${query}`);
          if (notesRes.ok) {
            const notesBody = await notesRes.json();
            if (notesBody.ok && Array.isArray(notesBody.notes)) {
              setSafeguardingEnvelopes(notesBody.notes);
            }
          }
        } else {
          setHasSafeguardingRead(false);
          setSafeguardingEnvelopes([]);
          setDecryptedNotes({});
        }
      } catch {
        // Ignore session fetch errors if not in DSL flow
      }
    };
    void loadSessionAndSafeguarding();
  }, [activePupilId, activeStaffId]);

  const handleSelectPersona = async (nextStaffId: string) => {
    const trimmed = nextStaffId.trim();
    if (!trimmed) {
      throw new Error('Missing mandatory staff_id for persona switch');
    }
    setPersonaSwitchError('');

    const switchRes = await fetch('/api/auth/persona', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ staff_id: trimmed }),
    });
    const switchBody = await switchRes.json();
    if (!switchRes.ok || !switchBody.ok) {
      setPersonaSwitchError(switchBody.error_code || 'Failed to switch persona');
      return;
    }

    const caller = switchBody.caller;
    const resolvedStaffId: string = caller?.staff_id || trimmed;
    const roles: string[] = Array.isArray(caller?.active_roles) ? caller.active_roles : [];
    const scopes: string[] = Array.isArray(caller?.scopes) ? caller.scopes : [];
    const nextCaller: ActiveCallerIdentity = {
      staff_id: resolvedStaffId,
      user_id: caller?.user_id || '',
      active_roles: roles,
      scopes,
    };

    setActiveStaffId(resolvedStaffId);
    setActiveCaller(nextCaller);

    const isAdminOrDpo = checkCallerAdminOrDpo(roles, scopes);
    setHasAdminOrDpoAccess(isAdminOrDpo);
    if (!isAdminOrDpo) {
      setSarSections([]);
      setSarExportStatus('');
      setSarFinalizedJson('');
      setSarPurgeStatus('');
      setDiagnosticBundleStatus('');
      setDiagnosticBundleJson('');
      setDiagnosticBundleLog('');
    }

    if (scopes.includes('dal.safeguarding.read')) {
      setHasSafeguardingRead(true);
      setDslVaultError('');
      const query = activePupilId ? `?pupil_id=${encodeURIComponent(activePupilId)}` : '';
      const notesRes = await fetch(`/api/safeguarding/notes${query}`);
      if (notesRes.ok) {
        const notesBody = await notesRes.json();
        if (notesBody.ok && Array.isArray(notesBody.notes)) {
          setSafeguardingEnvelopes(notesBody.notes);
        }
      }
    } else {
      setHasSafeguardingRead(false);
      setSafeguardingEnvelopes([]);
      setDecryptedNotes({});
      setCloudKeyToken('');
      setDslVaultError('');
    }

    const uiRes = await fetch(`/api/system/ui-config?staff_id=${encodeURIComponent(resolvedStaffId)}`);
    if (uiRes.ok) {
      const uiBody = await uiRes.json();
      if (uiBody && uiBody.ok && Array.isArray(uiBody.plugins)) {
        applyInstalledPluginsState(uiBody.plugins);
        broadcastInitToMountedIframes(
          { staff_id: resolvedStaffId },
          undefined,
          uiBody.plugins,
          nextCaller
        );
        return;
      }
    }

    broadcastInitToMountedIframes({ staff_id: resolvedStaffId }, undefined, undefined, nextCaller);
  };

  useEffect(() => {
    const resolveSourceIframe = (source: MessageEventSource | null): { pluginId: string; win: Window | null } => {
      if (source) {
        for (const [pid, el] of pluginIframeRefs.current.entries()) {
          if (el.contentWindow === source) {
            return { pluginId: pid, win: el.contentWindow };
          }
        }
        for (const [, entry] of widgetIframeRefs.current.entries()) {
          if (entry.el.contentWindow === source) {
            return { pluginId: entry.pluginId, win: entry.el.contentWindow };
          }
        }
      }
      const fallbackEl = pluginIframeRefs.current.get(activePluginId);
      return {
        pluginId: activePluginId,
        win: (source as Window | null) || fallbackEl?.contentWindow || null,
      };
    };

    const handleMessage = async (event: MessageEvent) => {
      const msg = event.data;
      if (!msg || typeof msg !== 'object') {
        return;
      }

      if (msg.type === 'mis:ready') {
        const { pluginId, win } = resolveSourceIframe(event.source);
        if (win) {
          win.postMessage(buildInitPayload(pluginId), '*');
        }
        return;
      }

      if (msg.type === 'mis:link:navigate') {
        const entityType = typeof msg.entity_type === 'string' ? msg.entity_type.trim().toLowerCase() : '';
        const entityId = typeof msg.entity_id === 'string' ? msg.entity_id.trim() : '';
        if (!entityType || !entityId) {
          throw new Error('Malformed mis:link:navigate payload: missing entity_type or entity_id');
        }

        const designatedPlugin = entityLinkHandlers[entityType] || PLUGIN_ID;
        activatePluginInKeepAlivePool(designatedPlugin);

        let nextPupilId = activePupilId;
        let nextGroupId = activeGroupId;
        let nextStaffId = activeStaffId;

        if (entityType === 'student' || (typeof msg.pupil_id === 'string' && msg.pupil_id.trim() !== '')) {
          nextPupilId = (typeof msg.pupil_id === 'string' && msg.pupil_id.trim()) || entityId;
          setActivePupilId(nextPupilId);
        }
        if (entityType === 'group' || (typeof msg.group_id === 'string' && msg.group_id.trim() !== '')) {
          nextGroupId = (typeof msg.group_id === 'string' && msg.group_id.trim()) || entityId;
          setActiveGroupId(nextGroupId);
        }
        if (entityType === 'staff' || (typeof msg.staff_id === 'string' && msg.staff_id.trim() !== '')) {
          nextStaffId = (typeof msg.staff_id === 'string' && msg.staff_id.trim()) || entityId;
          setActiveStaffId(nextStaffId);
        }

        setActiveEntityContext(`${entityType}:${entityId}`);
        setDeepLinkNavigationStatus(`${entityType}:${entityId} -> ${designatedPlugin}`);

        broadcastInitToMountedIframes({
          pupil_id: nextPupilId,
          group_id: nextGroupId,
          staff_id: nextStaffId,
          entity_type: entityType,
          entity_id: entityId,
        });
        return;
      }

      if (msg.type === 'mis:rpc:request') {
        const { request_id, method, args } = msg;
        if (!request_id || !method) {
          throw new Error('Malformed mis:rpc:request: missing request_id or method');
        }

        const { pluginId, win } = resolveSourceIframe(event.source);
        const res = await fetch(`/api/plugins/${encodeURIComponent(pluginId)}/rpc/${encodeURIComponent(method)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ args: args ?? {} }),
        });

        const payload = await res.json();
        win?.postMessage(
          {
            type: 'mis:rpc:response',
            request_id,
            ok: Boolean(payload.ok),
            data: payload.data,
            error_code: payload.error_code,
          },
          '*'
        );
      }
    };

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [
    buildInitPayload,
    broadcastInitToMountedIframes,
    activatePluginInKeepAlivePool,
    entityLinkHandlers,
    activePluginId,
    activePupilId,
    activeGroupId,
    activeStaffId,
  ]);

  const handleToggleThemeMode = () => {
    const nextMode: ThemeMode = themeMode === 'light' ? 'high-contrast' : 'light';
    applyHostThemeToRoot(nextMode, themeCatalog);
    setThemeMode(nextMode);
    broadcastInitToMountedIframes(undefined, nextMode);
  };

  const handlePlaceWidget = (widget: PluginWidgetDescriptor) => {
    setPlacedWidgets((prev) => {
      if (prev.some((w) => w.id === widget.id)) {
        return prev;
      }
      return [
        ...prev,
        {
          ...widget,
          colSpan: 2,
          rowSpan: 1,
        },
      ];
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

  const handleRenewCloudKey = async () => {
    setDslVaultError('');
    const res = await fetch('/api/safeguarding/cloud-key/renew', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ttl_seconds: 3600 }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to renew DSL Cloud Key');
      return;
    }
    setCloudKeyToken(body.cloud_key_token);
  };

  const handleDecryptSafeguardingNote = async (noteId: string) => {
    setDslVaultError('');
    const res = await fetch(`/api/safeguarding/notes/${encodeURIComponent(noteId)}/decrypt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cloud_key_token: cloudKeyToken }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to decrypt safeguarding note');
      return;
    }
    setDecryptedNotes((prev) => ({
      ...prev,
      [noteId]: body.note,
    }));
  };

  const handleLoadSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR export');
    }
    const res = await fetch('/api/admin/gdpr/sar/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pupil_id: activePupilId, staff_id: activeStaffId || undefined }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarSections(body.bundle.sections || []);
    setSarExportStatus('SAR_LOADED');
    setSarFinalizedJson('');
  };

  const handleUpdateRedactableField = (
    secIdx: number,
    recIdx: number,
    fieldKey: string,
    newValue: string
  ) => {
    setSarSections((prev) =>
      prev.map((sec, sIdx) => {
        if (sIdx !== secIdx) {
          return sec;
        }
        return {
          ...sec,
          records: sec.records.map((rec, rIdx) => {
            if (rIdx !== recIdx) {
              return rec;
            }
            return {
              ...rec,
              [fieldKey]: newValue,
            };
          }),
        };
      })
    );
  };

  const handleFinalizeSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR finalization');
    }
    const res = await fetch('/api/admin/gdpr/sar/finalize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
        staff_id: activeStaffId || undefined,
        sections: sarSections,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarExportStatus(body.package.status || 'SAR_EXPORT_FINALIZED');
    setSarFinalizedJson(JSON.stringify(body.package, null, 2));
  };

  const handlePurgePupilData = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for retention purge');
    }
    const res = await fetch('/api/admin/gdpr/purge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
        staff_id: activeStaffId || undefined,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarPurgeStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarPurgeStatus(body.result.status || 'RETENTION_PURGE_COMPLETED');
    setSarSections([]);
  };

  const handleDownloadDiagnosticBundle = async () => {
    const query = activeStaffId ? `?staff_id=${encodeURIComponent(activeStaffId)}` : '';
    const res = await fetch(`/api/admin/diagnostics/bundle${query}`);
    const body = await res.json();
    if (!res.ok || !body.ok || !body.bundle) {
      setDiagnosticBundleStatus(`ERROR:${body?.error_code || res.status}`);
      return;
    }
    const formattedJson = JSON.stringify(body.bundle, null, 2);
    setDiagnosticBundleJson(formattedJson);
    setDiagnosticBundleLog(typeof body.bundle.log_archive === 'string' ? body.bundle.log_archive : '');
    setDiagnosticBundleStatus('DIAGNOSTIC_BUNDLE_EXPORTED');
  };

  return (
    <div
      data-testid="host-shell"
      data-mis-theme={themeMode}
      style={{
        display: 'grid',
        gridTemplateColumns: '260px 1fr',
        minHeight: '100vh',
        fontFamily: 'var(--mis-font-sans, system-ui, sans-serif)',
        backgroundColor: 'var(--mis-bg-canvas)',
        color: 'var(--mis-text-primary)',
      }}
    >
      {/* Left Navigation Sidebar (Section 9.5) */}
      <aside
        data-testid="host-sidebar"
        style={{
          backgroundColor: 'var(--mis-bg-sidebar)',
          color: 'var(--mis-text-sidebar)',
          borderRight: '1px solid var(--mis-border-subtle)',
          padding: '20px 16px',
          display: 'flex',
          flexDirection: 'column',
          gap: '16px',
        }}
      >
        <div>
          <div style={{ fontSize: '11px', textTransform: 'uppercase', letterSpacing: '0.08em', opacity: 0.8 }}>
            Optimis School MIS • URN {schoolURN}
          </div>
          <h1
            data-testid="host-school-name"
            style={{ fontSize: '16px', fontWeight: 700, margin: '4px 0 0 0', lineHeight: 1.3 }}
          >
            {schoolName}
          </h1>
          <div style={{ fontSize: '12px', marginTop: '4px', opacity: 0.85 }}>
            Academic Cycle: <strong>{configuredAcademicYear}</strong>
          </div>
        </div>

        <nav style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div
            data-testid="nav-item-dashboard"
            style={{
              padding: '8px 12px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              backgroundColor: 'rgba(255, 255, 255, 0.1)',
              fontSize: '13px',
              fontWeight: 600,
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <span>Home Dashboard Grid</span>
          </div>

          {installedPlugins.map((plug) => (
            <button
              key={plug.id}
              type="button"
              data-testid={`nav-item-plugin-${plug.id}`}
              data-permitted-count={String(plug.permitted_methods.length)}
              data-permitted-methods={plug.permitted_methods.join(',')}
              onClick={() => activatePluginInKeepAlivePool(plug.id)}
              style={{
                padding: '8px 12px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                backgroundColor:
                  activePluginId === plug.id ? 'var(--mis-brand-primary)' : 'transparent',
                color:
                  activePluginId === plug.id
                    ? 'var(--mis-brand-primary-fg)'
                    : 'var(--mis-text-sidebar)',
                border: '1px solid var(--mis-border-subtle)',
                fontSize: '13px',
                fontWeight: 600,
                textAlign: 'left',
                cursor: 'pointer',
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
              }}
            >
              <span>{plug.title}</span>
              <span
                data-testid={`nav-item-plugin-methods-count-${plug.id}`}
                style={{
                  marginLeft: 'auto',
                  fontSize: '11px',
                  padding: '1px 6px',
                  borderRadius: '999px',
                  backgroundColor: 'rgba(255, 255, 255, 0.16)',
                }}
              >
                {plug.permitted_methods.length}
              </span>
            </button>
          ))}
        </nav>

        <div
          style={{
            marginTop: 'auto',
            padding: '10px 12px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '12px',
          }}
        >
          <div style={{ opacity: 0.75, marginBottom: '4px' }}>Active Entity Context</div>
          <div data-testid="active-entity-context-pill" style={{ fontWeight: 700, wordBreak: 'break-all' }}>
            {activeEntityContext}
          </div>
        </div>
      </aside>

      {/* Main Content Area: Topbar + Default Dashboard Grid */}
      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <header
          data-testid="host-topbar"
          style={{
            backgroundColor: 'var(--mis-bg-surface)',
            borderBottom: '1px solid var(--mis-border-subtle)',
            padding: '12px 24px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            gap: '16px',
            flexWrap: 'wrap',
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
            <span style={{ fontSize: '15px', fontWeight: 700 }}>
              Optimis School MIS — Host Shell
            </span>
            <span
              data-testid="deep-link-navigation-status"
              style={{
                padding: '3px 8px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                backgroundColor: 'var(--mis-bg-elevated)',
                border: '1px solid var(--mis-border-subtle)',
                fontSize: '12px',
                fontWeight: 600,
              }}
            >
              {deepLinkNavigationStatus}
            </span>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
            {devPersonaSwitcherEnabled && (
              <div
                data-testid="persona-switcher-container"
                style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}
              >
                <label
                  htmlFor="persona-switcher-select"
                  style={{ fontSize: '12px', fontWeight: 600, color: 'var(--mis-text-secondary)' }}
                >
                  Persona:
                </label>
                <select
                  id="persona-switcher-select"
                  data-testid="persona-switcher-select"
                  value={activeCaller?.staff_id || activeStaffId || ''}
                  onChange={(e) => {
                    void handleSelectPersona(e.target.value);
                  }}
                  style={{
                    padding: '5px 10px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-strong)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    fontSize: '12px',
                    fontWeight: 600,
                    cursor: 'pointer',
                  }}
                >
                  {activeCaller?.staff_id &&
                    !availablePersonas.some((p) => p.staff_id === activeCaller.staff_id) && (
                      <option value={activeCaller.staff_id}>
                        {activeCaller.staff_id} [{activeCaller.active_roles.join(', ')}]
                      </option>
                    )}
                  {availablePersonas.map((p) => (
                    <option
                      key={p.staff_id}
                      value={p.staff_id}
                      data-testid={`persona-option-${p.staff_id}`}
                    >
                      {p.display_name} — {p.job_title} [{p.active_roles.join(', ')}]
                    </option>
                  ))}
                </select>
                {activeCaller && (
                  <span
                    data-testid="active-persona-badge"
                    data-staff-id={activeCaller.staff_id}
                    data-roles={activeCaller.active_roles.join(',')}
                    data-scopes={activeCaller.scopes.join(',')}
                    style={{
                      padding: '3px 8px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      backgroundColor: 'var(--mis-bg-elevated)',
                      border: '1px solid var(--mis-border-subtle)',
                      fontSize: '11px',
                      fontWeight: 700,
                    }}
                  >
                    {activeCaller.staff_id} ({activeCaller.active_roles.join(', ')})
                  </span>
                )}
                {personaSwitchError && (
                  <span
                    data-testid="persona-switcher-error"
                    style={{ fontSize: '12px', color: 'var(--mis-status-danger)', fontWeight: 600 }}
                  >
                    {personaSwitchError}
                  </span>
                )}
              </div>
            )}

            <span style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
              Theme:{' '}
              <strong data-testid="active-theme-mode-indicator">{themeMode}</strong>
            </span>
            <button
              type="button"
              data-testid="theme-mode-toggle-btn"
              onClick={handleToggleThemeMode}
              style={{
                padding: '6px 12px',
                backgroundColor: 'var(--mis-brand-primary)',
                color: 'var(--mis-brand-primary-fg)',
                border: '1px solid var(--mis-border-strong)',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                fontSize: '12px',
                fontWeight: 700,
                cursor: 'pointer',
              }}
            >
              {themeMode === 'light' ? 'Switch to High-Contrast' : 'Switch to Light Theme'}
            </button>
          </div>
        </header>

        <main
          data-testid="host-dashboard-grid"
          style={{
            padding: '20px 24px',
            display: 'flex',
            flexDirection: 'column',
            gap: '20px',
          }}
        >
          {/* Default Dashboard Statutory Summary Strip (Section 9.5) */}
          <section
            data-testid="dashboard-summary-strip"
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
              gap: '12px',
            }}
          >
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Statutory Roll Call (AM / PM)
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                AM 08:45 • PM 13:15 Ready
              </div>
            </div>
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Entity Link Handler (student / group)
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                {entityLinkHandlers.student}
              </div>
            </div>
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Sandboxed Plugin CSP
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                connect-src &apos;none&apos; (Enforced)
              </div>
            </div>
          </section>

          {/* Editable NxM Mini-Iframe Widget Grid (Section 9B) */}
          <DashboardWidgetGridPanel
            isEditMode={isDashboardEditMode}
            availableWidgets={availableWidgets}
            placedWidgets={placedWidgets}
            onToggleEditMode={() => setIsDashboardEditMode((prev) => !prev)}
            onPlaceWidget={handlePlaceWidget}
            onResizeWidget={handleResizeWidget}
            onRemoveWidget={handleRemoveWidget}
            onRegisterWidgetIframeRef={handleRegisterWidgetIframeRef}
          />

          {/* Cached Keep-Alive LRU Plugin Workspace Pool (Section 9A: N=5) */}
          <section
            data-testid="dashboard-workspace-panel"
            style={{
              backgroundColor: 'var(--mis-bg-surface)',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              padding: '16px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <div
              data-testid="keep-alive-lru-pool"
              data-pool-size={String(mountedPluginIds.length)}
              data-lru-order={lruPluginOrder.join(',')}
            >
              {mountedPluginIds.map((pluginId) => {
                const descriptor = installedPlugins.find((p) => p.id === pluginId);
                const entryUrl = descriptor ? descriptor.entry_url : `/plugins/${pluginId}/ui/index.html`;
                const isActive = pluginId === activePluginId;
                return (
                  <iframe
                    key={pluginId}
                    ref={(el) => handleRegisterPluginIframeRef(pluginId, el)}
                    data-testid={`plugin-iframe-${pluginId}`}
                    data-lru-active={isActive ? 'true' : 'false'}
                    sandbox="allow-scripts"
                    src={entryUrl}
                    title={`Plugin Workspace: ${pluginId}`}
                    style={{
                      display: isActive ? 'block' : 'none',
                      width: '100%',
                      height: '320px',
                      border: '1px solid var(--mis-border-subtle)',
                      borderRadius: 'var(--mis-radius-md, 8px)',
                      backgroundColor: 'var(--mis-bg-surface)',
                    }}
                  />
                );
              })}
            </div>
          </section>

          {hasSafeguardingRead && (
            <SafeguardingVaultPanel
              safeguardingEnvelopes={safeguardingEnvelopes}
              cloudKeyToken={cloudKeyToken}
              onCloudKeyTokenChange={setCloudKeyToken}
              onRenewCloudKey={handleRenewCloudKey}
              onDecryptNote={handleDecryptSafeguardingNote}
              decryptedNotes={decryptedNotes}
              dslVaultError={dslVaultError}
            />
          )}

          {hasAdminOrDpoAccess && (
            <GdprSarPanel
              sarSections={sarSections}
              sarExportStatus={sarExportStatus}
              sarFinalizedJson={sarFinalizedJson}
              sarPurgeStatus={sarPurgeStatus}
              onLoadSarExport={handleLoadSarExport}
              onFinalizeSarExport={handleFinalizeSarExport}
              onPurgePupilData={handlePurgePupilData}
              onUpdateRedactableField={handleUpdateRedactableField}
            />
          )}

          {hasAdminOrDpoAccess && (
            <AdminDiagnosticsPanel
              diagnosticBundleStatus={diagnosticBundleStatus}
              diagnosticBundleJson={diagnosticBundleJson}
              diagnosticBundleLog={diagnosticBundleLog}
              onDownloadDiagnosticBundle={handleDownloadDiagnosticBundle}
            />
          )}
        </main>
      </div>
    </div>
  );
}
