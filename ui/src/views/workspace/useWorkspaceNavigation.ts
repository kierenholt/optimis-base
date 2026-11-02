import { useCallback, useState } from 'react';
import {
  HOST_ALLOWED_QUERY_KEYS,
  HostViewMode,
  parseBrowserRoute,
  validatePluginViewRoute,
} from '../../components/routeUtils';
import { MAX_KEEP_ALIVE_POOL_SIZE } from '../../themeAndTypes';

export interface UseWorkspaceNavigationParams {
  initialBrowserRoute: ReturnType<typeof parseBrowserRoute>;
}

export function useWorkspaceNavigation({ initialBrowserRoute }: UseWorkspaceNavigationParams) {
  // Right-Hand Content Pane View Mode ('home' | 'search' | 'settings' | 'plugin')
  const [hostViewMode, setHostViewMode] = useState<HostViewMode>(initialBrowserRoute.hostViewMode);

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

  const activatePluginInKeepAlivePool = useCallback(
    (targetPluginId: string, overrideViewPath?: string, overridePluginQuery?: Record<string, string>) => {
      setHostViewMode('plugin');
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

  const handleNavigateHome = useCallback(() => {
    setHostViewMode('home');
    setActivePluginId('');
    setActiveViewPath('/');
    setActiveRouteQuery({});
    setRouteError('');
    if (window.location.pathname !== '/' || window.location.search !== '') {
      window.history.pushState(null, '', '/');
    }
    setActiveRouteUrl('/');
  }, []);

  const handleNavigateSearch = useCallback(() => {
    setHostViewMode('search');
    setActivePluginId('');
    setActiveViewPath('/');
    setActiveRouteQuery({});
    setRouteError('');
    if (window.location.pathname !== '/search' || window.location.search !== '') {
      window.history.pushState(null, '', '/search');
    }
    setActiveRouteUrl('/search');
  }, []);

  const handleNavigateSettings = useCallback(() => {
    setHostViewMode('settings');
    setActivePluginId('');
    setActiveViewPath('/');
    setActiveRouteQuery({});
    setRouteError('');
    if (window.location.pathname !== '/settings' || window.location.search !== '') {
      window.history.pushState(null, '', '/settings');
    }
    setActiveRouteUrl('/settings');
  }, []);

  const clearKeepAlivePool = useCallback(() => {
    setActivePluginId('');
    setLruPluginOrder([]);
    setMountedPluginIds([]);
  }, []);

  const syncPluginsWithKeepAlive = useCallback(
    (
      validIds: Set<string>,
      urlPlugId: string,
      defaultId: string,
      isInitialLoad: boolean,
      urlRouteHostViewMode: HostViewMode
    ) => {
      setActivePluginId((prev) => {
        if (urlPlugId && validIds.has(urlPlugId)) {
          return urlPlugId;
        }
        if (urlRouteHostViewMode === 'search' || urlRouteHostViewMode === 'settings') {
          return '';
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
    []
  );

  return {
    hostViewMode,
    setHostViewMode,
    activePluginId,
    setActivePluginId,
    lruPluginOrder,
    setLruPluginOrder,
    mountedPluginIds,
    setMountedPluginIds,
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
    syncBrowserRouteUrl,
    activatePluginInKeepAlivePool,
    handleNavigateHome,
    handleNavigateSearch,
    handleNavigateSettings,
    clearKeepAlivePool,
    syncPluginsWithKeepAlive,
  };
}
