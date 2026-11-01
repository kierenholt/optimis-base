import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActiveCallerIdentity,
  BaseFontSize,
  getWidgetCompositeKey,
  PlacedDashboardWidget,
  PluginNavEntry,
  ThemeMode,
} from '../themeAndTypes';
import {
  createInitPayload,
  handleLinkNavigateMessage,
  handleRouteStateMessage,
  InitContextOverrides,
} from './iframeMessageHandlers';
import { parseBrowserRoute } from './routeUtils';

export interface UseHostIframeBridgeParams {
  themeMode: ThemeMode;
  baseFontSize: BaseFontSize;
  themeCatalog: Record<ThemeMode, Record<string, string>>;
  installedPlugins: PluginNavEntry[];
  activePluginId: string;
  setActivePluginId: React.Dispatch<React.SetStateAction<string>>;
  mountedPluginIds: string[];
  placedWidgets: PlacedDashboardWidget[];
  activeCaller: ActiveCallerIdentity | null;
  activePupilId: string;
  setActivePupilId: React.Dispatch<React.SetStateAction<string>>;
  initialSecondPupilId: string;
  activeGroupId: string;
  setActiveGroupId: React.Dispatch<React.SetStateAction<string>>;
  initialOtherGroupId: string;
  initialAbsentStaffId: string;
  initialCoverStaffId: string;
  initialSlotId: string;
  initialReferenceDate: string;
  initialAcademicYearId: string;
  configuredAcademicYear: string;
  activeStaffId: string;
  setActiveStaffId: React.Dispatch<React.SetStateAction<string>>;
  activeTab: string;
  setActiveTab: React.Dispatch<React.SetStateAction<string>>;
  activeViewPath: string;
  setActiveViewPath: React.Dispatch<React.SetStateAction<string>>;
  activeRouteQuery: Record<string, string>;
  setActiveRouteQuery: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  setActiveRouteUrl: React.Dispatch<React.SetStateAction<string>>;
  setRouteError: React.Dispatch<React.SetStateAction<string>>;
  setActiveEntityContext: React.Dispatch<React.SetStateAction<string>>;
  setDeepLinkNavigationStatus: React.Dispatch<React.SetStateAction<string>>;
  entityLinkHandlers: Record<string, string>;
  activatePluginInKeepAlivePool: (
    targetPluginId: string,
    overrideViewPath?: string,
    overridePluginQuery?: Record<string, string>
  ) => void;
  syncBrowserRouteUrl: (
    nextPlugId: string,
    nextViewPath: string,
    nextPluginQuery: Record<string, string>,
    replaceHistory?: boolean
  ) => string;
}

export function useHostIframeBridge(params: UseHostIframeBridgeParams) {
  const {
    themeMode,
    baseFontSize,
    themeCatalog,
    installedPlugins,
    activePluginId,
    setActivePluginId,
    mountedPluginIds,
    placedWidgets,
    activeCaller,
    activePupilId,
    setActivePupilId,
    initialSecondPupilId,
    activeGroupId,
    setActiveGroupId,
    initialOtherGroupId,
    initialAbsentStaffId,
    initialCoverStaffId,
    initialSlotId,
    initialReferenceDate,
    initialAcademicYearId,
    configuredAcademicYear,
    activeStaffId,
    setActiveStaffId,
    activeTab,
    setActiveTab,
    activeViewPath,
    setActiveViewPath,
    activeRouteQuery,
    setActiveRouteQuery,
    setActiveRouteUrl,
    setRouteError,
    setActiveEntityContext,
    setDeepLinkNavigationStatus,
    entityLinkHandlers,
    activatePluginInKeepAlivePool,
    syncBrowserRouteUrl,
  } = params;

  const pluginIframeRefs = useRef<Map<string, HTMLIFrameElement>>(new Map());
  const widgetIframeRefs = useRef<Map<string, { pluginId: string; el: HTMLIFrameElement }>>(new Map());
  const [pluginIframeHeights, setPluginIframeHeights] = useState<Record<string, number>>({});

  const handleRegisterPluginIframeRef = useCallback((pluginId: string, el: HTMLIFrameElement | null) => {
    if (el) {
      pluginIframeRefs.current.set(pluginId, el);
    } else {
      pluginIframeRefs.current.delete(pluginId);
    }
  }, []);

  const handleRegisterWidgetIframeRef = useCallback(
    (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => {
      const key = getWidgetCompositeKey(pluginId, widgetId);
      if (el) {
        widgetIframeRefs.current.set(key, { pluginId, el });
      } else {
        widgetIframeRefs.current.delete(key);
      }
    },
    []
  );

  const buildInitPayload = useCallback(
    (
      targetPluginId: string,
      overrideContext?: Partial<InitContextOverrides>,
      overrideThemeMode?: ThemeMode,
      overridePlugins?: PluginNavEntry[],
      overrideCaller?: ActiveCallerIdentity | null,
      overrideBaseFontSize?: BaseFontSize
    ) =>
      createInitPayload(
        {
          themeMode,
          baseFontSize,
          themeCatalog,
          installedPlugins,
          activeCaller,
          activePupilId,
          initialSecondPupilId,
          activeGroupId,
          initialOtherGroupId,
          initialAbsentStaffId,
          initialCoverStaffId,
          initialSlotId,
          initialReferenceDate,
          initialAcademicYearId,
          configuredAcademicYear,
          activeStaffId,
          activeTab,
          activeViewPath,
          activeRouteQuery,
        },
        targetPluginId,
        overrideContext,
        overrideThemeMode,
        overridePlugins,
        overrideCaller,
        overrideBaseFontSize
      ),
    [
      themeMode,
      baseFontSize,
      themeCatalog,
      installedPlugins,
      activeCaller,
      activePupilId,
      initialSecondPupilId,
      activeGroupId,
      initialOtherGroupId,
      initialAbsentStaffId,
      initialCoverStaffId,
      initialSlotId,
      initialReferenceDate,
      initialAcademicYearId,
      configuredAcademicYear,
      activeStaffId,
      activeTab,
      activeViewPath,
      activeRouteQuery,
    ]
  );

  const broadcastInitToMountedIframes = useCallback(
    (
      overrideContext?: Partial<InitContextOverrides>,
      overrideThemeMode?: ThemeMode,
      overridePlugins?: PluginNavEntry[],
      overrideCaller?: ActiveCallerIdentity | null,
      overrideBaseFontSize?: BaseFontSize
    ) => {
      for (const [pluginId, iframeEl] of pluginIframeRefs.current.entries()) {
        if (iframeEl?.contentWindow) {
          iframeEl.contentWindow.postMessage(
            buildInitPayload(
              pluginId,
              overrideContext,
              overrideThemeMode,
              overridePlugins,
              overrideCaller,
              overrideBaseFontSize
            ),
            '*'
          );
        }
      }
      for (const [, entry] of widgetIframeRefs.current.entries()) {
        if (entry.el?.contentWindow) {
          entry.el.contentWindow.postMessage(
            buildInitPayload(
              entry.pluginId,
              overrideContext,
              overrideThemeMode,
              overridePlugins,
              overrideCaller,
              overrideBaseFontSize
            ),
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
    const handlePopState = () => {
      const parsed = parseBrowserRoute(window.location.pathname, window.location.search);
      setActiveRouteUrl(`${window.location.pathname}${window.location.search}`);
      if (parsed.plugId) {
        setActivePluginId(parsed.plugId);
      }
      setActiveViewPath(parsed.viewPath || '/');
      setActiveTab(parsed.tab || '');
      setActiveRouteQuery(parsed.pluginQuery);
      if (parsed.pupilIdFromPath) {
        setActivePupilId(parsed.pupilIdFromPath);
        setActiveEntityContext(`student:${parsed.pupilIdFromPath}`);
      } else if (parsed.groupIdFromPath) {
        setActiveGroupId(parsed.groupIdFromPath);
        setActiveEntityContext(`group:${parsed.groupIdFromPath}`);
      }
      broadcastInitToMountedIframes({
        pupil_id: parsed.pupilIdFromPath || activePupilId,
        group_id: parsed.groupIdFromPath || activeGroupId,
        tab: parsed.tab || '',
        view_path: parsed.viewPath || '/',
        route_query: parsed.pluginQuery,
      });
    };
    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, [
    activePupilId,
    activeGroupId,
    broadcastInitToMountedIframes,
    setActivePluginId,
    setActiveRouteQuery,
    setActiveRouteUrl,
    setActiveTab,
    setActiveViewPath,
    setActivePupilId,
    setActiveGroupId,
    setActiveEntityContext,
  ]);

  const handleMessageRef = useRef<(event: MessageEvent) => void>(() => {});
  handleMessageRef.current = async (event: MessageEvent) => {
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

    if (msg.type === 'mis:resize') {
      const rawHeight = Number(msg.height);
      if (!Number.isFinite(rawHeight) || rawHeight <= 0 || !event.source) {
        return;
      }
      for (const [pid, el] of pluginIframeRefs.current.entries()) {
        if (el.contentWindow === event.source) {
          const nextHeight = Math.max(640, Math.ceil(rawHeight));
          el.style.height = `${nextHeight}px`;
          setPluginIframeHeights((prev) =>
            prev[pid] === nextHeight ? prev : { ...prev, [pid]: nextHeight }
          );
          return;
        }
      }
      return;
    }

    const routeDeps = {
      installedPlugins,
      activePluginId,
      activeViewPath,
      activePupilId,
      activeGroupId,
      activeStaffId,
      entityLinkHandlers,
      pluginIframeRefs,
      setRouteError,
      setActiveViewPath,
      setActiveTab,
      setActiveRouteQuery,
      setActivePupilId,
      setActiveGroupId,
      setActiveStaffId,
      setActiveEntityContext,
      setDeepLinkNavigationStatus,
      activatePluginInKeepAlivePool,
      syncBrowserRouteUrl,
      broadcastInitToMountedIframes,
    };

    if (msg.type === 'mis:route:state') {
      const { pluginId: sourcePluginId } = resolveSourceIframe(event.source);
      handleRouteStateMessage(msg as Record<string, unknown>, sourcePluginId, routeDeps);
      return;
    }

    if (msg.type === 'mis:link:navigate') {
      const { pluginId: sourcePluginId } = resolveSourceIframe(event.source);
      handleLinkNavigateMessage(msg as Record<string, unknown>, sourcePluginId, routeDeps);
      return;
    }

    if (msg.type === 'mis:rpc:request') {
      const { request_id, method, args } = msg;
      if (!request_id || !method) {
        throw new Error('Malformed mis:rpc:request: missing request_id or method');
      }

      const { pluginId, win } = resolveSourceIframe(event.source);
      const res = await fetch(`/api/p/${encodeURIComponent(pluginId)}/rpc/${encodeURIComponent(method)}`, {
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

      if (payload.ok && payload.mutated) {
        const invalidationEvent = {
          type: 'mis:plugin:invalidated',
          plugin_id: pluginId,
          method,
        };
        const mainPluginEl = pluginIframeRefs.current.get(pluginId);
        if (mainPluginEl?.contentWindow) {
          mainPluginEl.contentWindow.postMessage(invalidationEvent, '*');
        }
        for (const [, entry] of widgetIframeRefs.current.entries()) {
          if (entry.pluginId === pluginId && entry.el?.contentWindow) {
            entry.el.contentWindow.postMessage(invalidationEvent, '*');
          }
        }
      }
    }
  };

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      handleMessageRef.current(event);
    };
    window.addEventListener('message', onMessage);
    return () => {
      window.removeEventListener('message', onMessage);
    };
  }, []);

  return {
    pluginIframeHeights,
    handleRegisterPluginIframeRef,
    handleRegisterWidgetIframeRef,
    buildInitPayload,
    broadcastInitToMountedIframes,
  };
}
