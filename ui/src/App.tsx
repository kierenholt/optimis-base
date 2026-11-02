import React, { useEffect, useState } from 'react';
import {
  GlobalCommandPaletteModal,
  HostFullPageSearchView,
} from './components/GlobalSearchAndCommandPalette';
import { HostSidebar } from './components/HostSidebar';
import { useHostIframeBridge } from './components/useHostIframeBridge';
import {
  applyHostThemeToRoot,
  BaseFontSize,
  ThemeMode,
} from './themeAndTypes';
import { DashboardSummaryStrip } from './views/DashboardSummaryStrip';
import { HostHomeDashboardView } from './views/HostHomeDashboardView';
import { HostSettingsView } from './views/HostSettingsView';
import { KeepAlivePluginWorkspace } from './views/KeepAlivePluginWorkspace';
import { useHostAdminAndDslState } from './views/useHostAdminAndDslState';
import { useHostSearchNavigation } from './views/useHostSearchNavigation';
import { useHostWorkspaceState } from './views/useHostWorkspaceState';

export function App() {
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState<boolean>(false);
  const [isLoggedOut, setIsLoggedOut] = useState<boolean>(false);

  const {
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
    dashboardSaveStatus,
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
  } = useHostWorkspaceState();

  const {
    pluginIframeHeights,
    handleRegisterPluginIframeRef,
    handleRegisterWidgetIframeRef,
    buildInitPayload,
    broadcastInitToMountedIframes,
  } = useHostIframeBridge({
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
  });

  const {
    hasSafeguardingRead,
    safeguardingEnvelopes,
    cloudKeyToken,
    setCloudKeyToken,
    decryptedNotes,
    dslVaultError,
    handleRenewCloudKey,
    handleDecryptSafeguardingNote,
    hasAdminOrDpoAccess,
    sarSections,
    sarExportStatus,
    sarFinalizedJson,
    sarPurgeStatus,
    handleLoadSarExport,
    handleUpdateRedactableField,
    handleFinalizeSarExport,
    handlePurgePupilData,
    diagnosticBundleStatus,
    diagnosticBundleJson,
    diagnosticBundleLog,
    handleDownloadDiagnosticBundle,
  } = useHostAdminAndDslState({
    activePupilId,
    activeStaffId,
    setActiveStaffId,
    activeCaller,
    setActiveCaller,
    setPinnedRecents,
    applyInstalledPluginsState,
    refreshWorkspaceForStaff,
    broadcastInitToMountedIframes,
  });

  const {
    handleTogglePinSearchView,
    handleSelectSearchViewResult,
    handleSelectSearchPupilResult,
    handleSelectSearchGroupResult,
  } = useHostSearchNavigation({
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
  });

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setIsCommandPaletteOpen((prev) => !prev);
      } else if (e.key === 'Escape') {
        setIsCommandPaletteOpen(false);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const handleToggleThemeMode = () => {
    const nextMode: ThemeMode = themeMode === 'light' ? 'high-contrast' : 'light';
    applyHostThemeToRoot(nextMode, themeCatalog, baseFontSize);
    setThemeMode(nextMode);
    broadcastInitToMountedIframes(undefined, nextMode, undefined, undefined, baseFontSize);
    void persistUserSettings({ theme_mode: nextMode });
  };

  const handleSelectBaseFontSize = (nextSize: BaseFontSize) => {
    applyHostThemeToRoot(themeMode, themeCatalog, nextSize);
    setBaseFontSize(nextSize);
    broadcastInitToMountedIframes(undefined, themeMode, undefined, undefined, nextSize);
    void persistUserSettings({ base_font_size: nextSize });
  };

  const activePlugin = installedPlugins.find((p) => p.id === activePluginId);

  const canonicalActivePluginRoute = (() => {
    if (!activePluginId) {
      return '';
    }
    const cleanSubpath =
      !activeViewPath || activeViewPath === '/'
        ? ''
        : activeViewPath.startsWith('/')
          ? activeViewPath
          : `/${activeViewPath}`;
    const qsParams = new URLSearchParams();
    for (const [k, v] of Object.entries(activeRouteQuery)) {
      if (v !== undefined && v !== null && String(v) !== '') {
        qsParams.set(k, String(v));
      }
    }
    const qs = qsParams.toString();
    return qs ? `/p/${activePluginId}${cleanSubpath}?${qs}` : `/p/${activePluginId}${cleanSubpath}`;
  })();

  const isCurrentRoutePinned = Boolean(
    canonicalActivePluginRoute && pinnedRecents.some((r) => r.route === canonicalActivePluginRoute)
  );

  const handleTogglePinCurrentRoute = () => {
    if (!canonicalActivePluginRoute || !activePlugin) {
      return;
    }
    void handlePinOrUnpinRoute(canonicalActivePluginRoute, activePlugin.title);
  };

  const handleLogout = async () => {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } finally {
      setIsLoggedOut(true);
    }
  };

  return (
    <div
      data-testid="host-shell"
      data-mis-theme={themeMode}
      data-mis-font-size={baseFontSize}
      style={{
        display: 'grid',
        gridTemplateColumns: '88px 1fr',
        minHeight: '100vh',
        fontFamily: 'var(--mis-font-sans, system-ui, sans-serif)',
        fontSize: 'var(--mis-font-size-base, 16px)',
        backgroundColor: 'var(--mis-bg-canvas)',
        color: 'var(--mis-text-primary)',
      }}
    >
      <HostSidebar
        schoolURN={schoolURN}
        schoolName={schoolName}
        configuredAcademicYear={configuredAcademicYear}
        hostViewMode={hostViewMode}
        activePluginId={activePluginId}
        activePlugin={activePlugin}
        isCurrentRoutePinned={isCurrentRoutePinned}
        onTogglePinCurrentRoute={handleTogglePinCurrentRoute}
        activeRouteUrl={canonicalActivePluginRoute || activeRouteUrl}
        activeEntityContext={activeEntityContext}
        pinnedRecents={pinnedRecents}
        onNavigateHome={handleNavigateHome}
        onOpenSearch={handleNavigateSearch}
        onSelectPinnedRoute={handleSelectPinnedRoute}
        onUnpinRoute={(route) => void handleUnpinRoute(route)}
        onOpenSettings={handleNavigateSettings}
        onLogout={() => void handleLogout()}
      />

      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        {isLoggedOut && (
          <div
            data-testid="host-logged-out-banner"
            style={{
              margin: '16px 24px 0',
              padding: '10px 14px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-strong)',
              backgroundColor: 'var(--mis-bg-elevated)',
              color: 'var(--mis-text-primary)',
              fontSize: 'calc(13px * var(--mis-font-scale, 1))',
              fontWeight: 700,
            }}
          >
            Signed out of active MIS session. Select a persona or refresh to sign back in.
          </div>
        )}

        <main
          data-testid="host-dashboard-grid"
          style={{
            padding: '20px 24px',
            display: 'flex',
            flexDirection: 'column',
            gap: '20px',
          }}
        >
          <DashboardSummaryStrip routeError={routeError} />

          <HostHomeDashboardView
            isVisible={hostViewMode === 'home'}
            installedPlugins={installedPlugins}
            placedWidgets={placedWidgets}
            entityLinkHandlers={entityLinkHandlers}
            activePupilId={activePupilId}
            activeGroupId={activeGroupId}
            activeStaffId={activeCaller?.staff_id || activeStaffId}
            referenceDate={initialReferenceDate}
            academicYearId={configuredAcademicYear}
            pinnedRecents={pinnedRecents}
            onPlaceWidget={handlePlaceWidget}
            onSelectSearchViewResult={handleSelectSearchViewResult}
            onSelectSearchPupilResult={handleSelectSearchPupilResult}
            onSelectSearchGroupResult={handleSelectSearchGroupResult}
            onTogglePinSearchView={handleTogglePinSearchView}
            onOpenCommandPalette={() => setIsCommandPaletteOpen(true)}
            isDashboardEditMode={isDashboardEditMode}
            dashboardColumnCount={dashboardColumnCount}
            availableWidgets={availableWidgets}
            onToggleDashboardEditMode={() => setIsDashboardEditMode((prev) => !prev)}
            onRemoveWidget={handleRemoveWidget}
            onMoveWidgetUp={handleMoveWidgetUp}
            onMoveWidgetDown={handleMoveWidgetDown}
            onMoveWidgetLeft={handleMoveWidgetLeft}
            onMoveWidgetRight={handleMoveWidgetRight}
            onAddDashboardColumn={handleAddDashboardColumn}
            onRemoveDashboardColumn={handleRemoveDashboardColumn}
            dashboardSaveStatus={dashboardSaveStatus}
            onRegisterWidgetIframeRef={handleRegisterWidgetIframeRef}
            hasSafeguardingRead={hasSafeguardingRead}
            safeguardingEnvelopes={safeguardingEnvelopes}
            cloudKeyToken={cloudKeyToken}
            onCloudKeyTokenChange={setCloudKeyToken}
            onRenewCloudKey={handleRenewCloudKey}
            onDecryptSafeguardingNote={handleDecryptSafeguardingNote}
            decryptedNotes={decryptedNotes}
            dslVaultError={dslVaultError}
            hasAdminOrDpoAccess={hasAdminOrDpoAccess}
            sarSections={sarSections}
            sarExportStatus={sarExportStatus}
            sarFinalizedJson={sarFinalizedJson}
            sarPurgeStatus={sarPurgeStatus}
            onLoadSarExport={handleLoadSarExport}
            onFinalizeSarExport={handleFinalizeSarExport}
            onPurgePupilData={handlePurgePupilData}
            onUpdateRedactableField={handleUpdateRedactableField}
            diagnosticBundleStatus={diagnosticBundleStatus}
            diagnosticBundleJson={diagnosticBundleJson}
            diagnosticBundleLog={diagnosticBundleLog}
            onDownloadDiagnosticBundle={handleDownloadDiagnosticBundle}
          />

          {hostViewMode === 'search' && (
            <HostFullPageSearchView
              installedPlugins={installedPlugins}
              placedWidgets={placedWidgets}
              entityLinkHandlers={entityLinkHandlers}
              activePupilId={activePupilId}
              activeGroupId={activeGroupId}
              activeStaffId={activeCaller?.staff_id || activeStaffId}
              referenceDate={initialReferenceDate}
              academicYearId={configuredAcademicYear}
              pinnedRecents={pinnedRecents}
              onPlaceWidget={handlePlaceWidget}
              onSelectViewResult={handleSelectSearchViewResult}
              onSelectPupilResult={handleSelectSearchPupilResult}
              onSelectGroupResult={handleSelectSearchGroupResult}
              onTogglePinView={handleTogglePinSearchView}
            />
          )}

          {hostViewMode === 'settings' && (
            <HostSettingsView
              schoolName={schoolName}
              schoolURN={schoolURN}
              configuredAcademicYear={configuredAcademicYear}
              themeMode={themeMode}
              baseFontSize={baseFontSize}
              pinnedViewsCount={pinnedRecents.length}
              onToggleThemeMode={handleToggleThemeMode}
              onSelectBaseFontSize={handleSelectBaseFontSize}
            />
          )}

          <KeepAlivePluginWorkspace
            mountedPluginIds={mountedPluginIds}
            lruPluginOrder={lruPluginOrder}
            installedPlugins={installedPlugins}
            activePluginId={activePluginId}
            isVisible={hostViewMode === 'plugin'}
            pluginIframeHeights={pluginIframeHeights}
            onRegisterPluginIframeRef={handleRegisterPluginIframeRef}
            onIframeLoad={(pluginId, win) => {
              win?.postMessage(buildInitPayload(pluginId), '*');
            }}
          />
        </main>
      </div>

      <GlobalCommandPaletteModal
        isOpen={isCommandPaletteOpen}
        installedPlugins={installedPlugins}
        placedWidgets={placedWidgets}
        entityLinkHandlers={entityLinkHandlers}
        activePupilId={activePupilId}
        activeGroupId={activeGroupId}
        activeStaffId={activeCaller?.staff_id || activeStaffId}
        referenceDate={initialReferenceDate}
        academicYearId={configuredAcademicYear}
        pinnedRecents={pinnedRecents}
        onClose={() => setIsCommandPaletteOpen(false)}
        onPlaceWidget={handlePlaceWidget}
        onSelectViewResult={handleSelectSearchViewResult}
        onSelectPupilResult={handleSelectSearchPupilResult}
        onSelectGroupResult={handleSelectSearchGroupResult}
        onTogglePinView={handleTogglePinSearchView}
      />
    </div>
  );
}
