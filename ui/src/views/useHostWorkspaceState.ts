import { useCallback } from 'react';
import { parseBrowserRoute } from '../components/routeUtils';
import { PinnedRecentEntry } from '../themeAndTypes';
import {
  useWorkspaceDashboardLayout,
  useWorkspaceEntityContext,
  useWorkspaceInstalledPlugins,
  useWorkspaceNavigation,
  useWorkspacePinnedRecents,
  useWorkspaceThemeAndBranding,
} from './workspace';

export function useHostWorkspaceState() {
  // 1. Entity context & initial route parsing
  const {
    initialBrowserRoute,
    initialSecondPupilId,
    initialOtherGroupId,
    initialAbsentStaffId,
    initialCoverStaffId,
    initialSlotId,
    initialReferenceDate,
    initialAcademicYearId,
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
  } = useWorkspaceEntityContext();

  // 2. Theme & School Branding state
  const {
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
  } = useWorkspaceThemeAndBranding({
    initialAcademicYearId,
    activeStaffId,
  });

  // 3. Navigation, Opaque Route URL & Keep-Alive Pool state
  const {
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
  } = useWorkspaceNavigation({
    initialBrowserRoute,
  });

  // 4. Personal Home Dashboard 2D Coordinate Grid state
  const {
    isDashboardEditMode,
    setIsDashboardEditMode,
    placedWidgets,
    setPlacedWidgets,
    dashboardColumnCount,
    dashboardLayoutSource,
    dashboardSaveStatus,
    roleDefaultLayouts,
    selectedAdminRoleTarget,
    setSelectedAdminRoleTarget,
    hasLoadedServerLayoutRef,
    applyDashboardLayoutResponse,
    handlePlaceWidget,
    handleRemoveWidget,
    handleMoveWidgetUp,
    handleMoveWidgetDown,
    handleMoveWidgetLeft,
    handleMoveWidgetRight,
    handleAddDashboardColumn,
    handleRemoveDashboardColumn,
    handleSaveRoleDefaultLayout,
    handleAppendWidgetToRoleDefault,
  } = useWorkspaceDashboardLayout({
    activeStaffId,
  });

  // 5. Pinned routes & recents
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
    [
      setActivePupilId,
      setActiveEntityContext,
      setActiveGroupId,
      setActiveViewPath,
      setActiveRouteQuery,
      setActiveTab,
      setRouteError,
      activatePluginInKeepAlivePool,
    ]
  );

  const {
    pinnedRecents,
    setPinnedRecents,
    handlePinOrUnpinRoute,
    handleUnpinRoute,
  } = useWorkspacePinnedRecents({
    activeStaffId,
    onSelectPinnedRoute: handleSelectPinnedRoute,
  });

  // 6. Installed Plugins, Widgets & Workspace Bootstrap Sync
  const {
    installedPlugins,
    entityLinkHandlers,
    availableWidgets,
    applyInstalledPluginsState,
    refreshWorkspaceForStaff,
  } = useWorkspaceInstalledPlugins({
    initialBrowserPlugId: initialBrowserRoute.plugId,
    initialAcademicYearId,
    initialStaffIdFromPath: initialBrowserRoute.staffIdFromPath,
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
  });

  return {
    initialSecondPupilId,
    initialOtherGroupId,
    initialAbsentStaffId,
    initialCoverStaffId,
    initialSlotId,
    initialReferenceDate,
    initialAcademicYearId,
    themeMode,
    setThemeMode,
    baseFontSize,
    setBaseFontSize,
    persistUserSettings,
    themeCatalog,
    schoolName,
    schoolURN,
    configuredAcademicYear,
    entityLinkHandlers,
    installedPlugins,
    pinnedRecents,
    setPinnedRecents,
    hostViewMode,
    setHostViewMode,
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
    dashboardColumnCount,
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
    handleNavigateSearch,
    handleNavigateSettings,
    handlePinOrUnpinRoute,
    handleUnpinRoute,
    handleSelectPinnedRoute,
    handlePlaceWidget,
    handleRemoveWidget,
    handleMoveWidgetUp,
    handleMoveWidgetDown,
    handleMoveWidgetLeft,
    handleMoveWidgetRight,
    handleAddDashboardColumn,
    handleRemoveDashboardColumn,
    handleSaveRoleDefaultLayout,
    handleAppendWidgetToRoleDefault,
  };
}
