import { InstalledPlugin, PluginViewDescriptor } from '../themeAndTypes';

export interface UseHostSearchNavigationProps {
  activePupilId: string;
  setActivePupilId: (pupilId: string) => void;
  activeGroupId: string;
  setActiveGroupId: (groupId: string) => void;
  activeStaffId: string;
  setActiveViewPath: (path: string) => void;
  setActiveRouteQuery: (query: Record<string, string | number | boolean>) => void;
  setRouteError: (err: string) => void;
  setActiveEntityContext: (context: string) => void;
  setDeepLinkNavigationStatus: (status: string) => void;
  activatePluginInKeepAlivePool: (
    pluginId: string,
    viewPath?: string,
    routeQuery?: Record<string, string | number | boolean>
  ) => void;
  broadcastInitToMountedIframes: (overrides?: Record<string, unknown>) => void;
  entityLinkHandlers: Record<string, string>;
  installedPlugins: InstalledPlugin[];
  handlePinOrUnpinRoute: (route: string, title: string, viewId?: string) => Promise<void>;
}

export function useHostSearchNavigation({
  activePupilId,
  setActivePupilId,
  activeGroupId,
  setActiveGroupId,
  activeStaffId,
  setActiveViewPath,
  setActiveRouteQuery,
  setRouteError,
  setActiveEntityContext,
  setDeepLinkNavigationStatus,
  activatePluginInKeepAlivePool,
  broadcastInitToMountedIframes,
  entityLinkHandlers,
  installedPlugins,
  handlePinOrUnpinRoute,
}: UseHostSearchNavigationProps) {
  const resolveViewPathStrict = (rawPath: string | undefined): string => {
    let resolvedPath = rawPath || '/';
    if (resolvedPath.includes(':pupil_id')) {
      if (!activePupilId) {
        throw new Error('Missing mandatory activePupilId for :pupil_id view route');
      }
      resolvedPath = resolvedPath.replace(':pupil_id', activePupilId);
    }
    if (resolvedPath.includes(':group_id')) {
      if (!activeGroupId) {
        throw new Error('Missing mandatory activeGroupId for :group_id view route');
      }
      resolvedPath = resolvedPath.replace(':group_id', activeGroupId);
    }
    if (resolvedPath.includes(':staff_id')) {
      if (!activeStaffId) {
        throw new Error('Missing mandatory activeStaffId for :staff_id view route');
      }
      resolvedPath = resolvedPath.replace(':staff_id', activeStaffId);
    }
    return resolvedPath;
  };

  const handleTogglePinSearchView = (
    pluginId: string,
    view: PluginViewDescriptor,
    pluginTitle: string
  ) => {
    const resolvedPath = resolveViewPathStrict(view.path);
    const route = `/p/${pluginId}${resolvedPath === '/' ? '' : resolvedPath}`;
    void handlePinOrUnpinRoute(route, view.title || pluginTitle, view.id);
  };

  const handleSelectSearchViewResult = (pluginId: string, view: PluginViewDescriptor) => {
    const resolvedPath = resolveViewPathStrict(view.path);
    setActiveViewPath(resolvedPath);
    setActiveRouteQuery({});
    setRouteError('');
    activatePluginInKeepAlivePool(pluginId, resolvedPath, {});
  };

  const handleSelectSearchPupilResult = (pupilId: string) => {
    const targetPluginId = entityLinkHandlers.student || installedPlugins[0]?.id || '';
    if (!targetPluginId) {
      return;
    }
    const targetPath = `/student/${pupilId}`;
    setActivePupilId(pupilId);
    setActiveViewPath(targetPath);
    setActiveRouteQuery({});
    setRouteError('');
    setActiveEntityContext(`student:${pupilId}`);
    setDeepLinkNavigationStatus(`DEEP_LINK_OK:student:${pupilId}->${targetPluginId}`);
    activatePluginInKeepAlivePool(targetPluginId, targetPath, {});
    broadcastInitToMountedIframes({ pupil_id: pupilId });
  };

  const handleSelectSearchGroupResult = (groupId: string) => {
    const targetPluginId = entityLinkHandlers.group || installedPlugins[0]?.id || '';
    if (!targetPluginId) {
      return;
    }
    const targetPath = `/group/${groupId}`;
    setActiveGroupId(groupId);
    setActiveViewPath(targetPath);
    setActiveRouteQuery({});
    setRouteError('');
    setActiveEntityContext(`group:${groupId}`);
    setDeepLinkNavigationStatus(`DEEP_LINK_OK:group:${groupId}->${targetPluginId}`);
    activatePluginInKeepAlivePool(targetPluginId, targetPath, {});
    broadcastInitToMountedIframes({ group_id: groupId });
  };

  return {
    resolveViewPathStrict,
    handleTogglePinSearchView,
    handleSelectSearchViewResult,
    handleSelectSearchPupilResult,
    handleSelectSearchGroupResult,
  };
}
