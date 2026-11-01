import React, { useEffect, useState } from 'react';
import { AdminDiagnosticsPanel } from './AdminDiagnosticsPanel';
import {
  GlobalCommandPaletteModal,
  LandingPageTopHalfSearch,
} from './components/GlobalSearchAndCommandPalette';
import { HostSidebar } from './components/HostSidebar';
import { MonochromeLucideIcon } from './components/MonochromeLucideIcon';
import { useHostIframeBridge } from './components/useHostIframeBridge';
import { DashboardWidgetGridPanel } from './DashboardWidgetGridPanel';
import { GdprSarPanel } from './GdprSarPanel';
import { SafeguardingVaultPanel } from './SafeguardingVaultPanel';
import {
  applyHostThemeToRoot,
  BASE_FONT_SIZE_PRESETS,
  BaseFontSize,
  PluginViewDescriptor,
  ThemeMode,
} from './themeAndTypes';
import { DashboardSummaryStrip } from './views/DashboardSummaryStrip';
import { KeepAlivePluginWorkspace } from './views/KeepAlivePluginWorkspace';
import { useHostAdminAndDslState } from './views/useHostAdminAndDslState';
import { useHostWorkspaceState } from './views/useHostWorkspaceState';

export function App() {
  const [isCommandPaletteOpen, setIsCommandPaletteOpen] = useState<boolean>(false);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState<boolean>(false);
  const [isLoggedOut, setIsLoggedOut] = useState<boolean>(false);

  const {
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
  } = useHostWorkspaceState();

  const {
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
    initialPersonaParam,
    setPinnedRecents,
    applyInstalledPluginsState,
    refreshWorkspaceForStaff,
    broadcastInitToMountedIframes,
  });

  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setIsCommandPaletteOpen((prev) => !prev);
      } else if (e.key === 'Escape') {
        setIsCommandPaletteOpen(false);
        setIsSettingsModalOpen(false);
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

  const handleLogout = async () => {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } finally {
      setIsLoggedOut(true);
    }
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

  return (
    <div
      data-testid="host-shell"
      data-mis-theme={themeMode}
      data-mis-font-size={baseFontSize}
      style={{
        display: 'grid',
        gridTemplateColumns: '88px 1fr',
        minHeight: 'calc(100vh / var(--mis-font-scale, 1))',
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
        activePluginId={activePluginId}
        activePlugin={activePlugin}
        isCurrentRoutePinned={isCurrentRoutePinned}
        onTogglePinCurrentRoute={handleTogglePinCurrentRoute}
        activeRouteUrl={canonicalActivePluginRoute || activeRouteUrl}
        activeEntityContext={activeEntityContext}
        pinnedRecents={pinnedRecents}
        onNavigateHome={handleNavigateHome}
        onOpenSearch={() => setIsCommandPaletteOpen(true)}
        onSelectPinnedRoute={handleSelectPinnedRoute}
        onUnpinRoute={(route) => void handleUnpinRoute(route)}
        onOpenSettings={() => setIsSettingsModalOpen(true)}
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
              fontSize: '13px',
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

          <LandingPageTopHalfSearch
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
            onOpenCommandPalette={() => setIsCommandPaletteOpen(true)}
          />

          <DashboardWidgetGridPanel
            isEditMode={isDashboardEditMode}
            availableWidgets={availableWidgets}
            placedWidgets={placedWidgets}
            onToggleEditMode={() => setIsDashboardEditMode((prev) => !prev)}
            onPlaceWidget={handlePlaceWidget}
            onRemoveWidget={handleRemoveWidget}
            onMoveWidgetUp={handleMoveWidgetUp}
            onMoveWidgetDown={handleMoveWidgetDown}
            isAdmin={hasAdminOrDpoAccess}
            dashboardLayoutSource={dashboardLayoutSource}
            dashboardSaveStatus={dashboardSaveStatus}
            selectedAdminRoleTarget={selectedAdminRoleTarget}
            onSelectAdminRoleTarget={setSelectedAdminRoleTarget}
            onSaveRoleDefaultLayout={handleSaveRoleDefaultLayout}
            onAppendWidgetToRoleDefault={handleAppendWidgetToRoleDefault}
            onRegisterWidgetIframeRef={handleRegisterWidgetIframeRef}
          />

          <KeepAlivePluginWorkspace
            mountedPluginIds={mountedPluginIds}
            lruPluginOrder={lruPluginOrder}
            installedPlugins={installedPlugins}
            activePluginId={activePluginId}
            onRegisterPluginIframeRef={handleRegisterPluginIframeRef}
            onIframeLoad={(pluginId, win) => {
              win?.postMessage(buildInitPayload(pluginId), '*');
            }}
          />

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

      {isSettingsModalOpen && (
        <div
          data-testid="host-settings-backdrop"
          onClick={() => setIsSettingsModalOpen(false)}
          style={{
            position: 'fixed',
            inset: 0,
            backgroundColor: 'rgba(15, 23, 42, 0.55)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            zIndex: 9998,
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Host Settings"
            data-testid="host-settings-modal"
            onClick={(e) => e.stopPropagation()}
            style={{
              width: '100%',
              maxWidth: '480px',
              backgroundColor: 'var(--mis-bg-surface)',
              color: 'var(--mis-text-primary)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              boxShadow: 'var(--mis-shadow-card)',
              padding: '20px',
              display: 'flex',
              flexDirection: 'column',
              gap: '12px',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <h2 style={{ margin: 0, fontSize: '16px', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '8px' }}>
                <MonochromeLucideIcon name="settings" size={16} />
                <span>Host Settings &amp; Preferences</span>
              </h2>
              <button
                type="button"
                aria-label="Close"
                title="Close"
                data-testid="close-settings-modal-btn"
                onClick={() => setIsSettingsModalOpen(false)}
                style={{
                  padding: '4px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: '1px solid var(--mis-border-subtle)',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  color: 'var(--mis-text-primary)',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <MonochromeLucideIcon name="x" size={14} />
              </button>
            </div>
            <div style={{ fontSize: '13px', display: 'flex', flexDirection: 'column', gap: '10px' }}>
              <div>
                <strong>School Name:</strong> {schoolName}
              </div>
              <div>
                <strong>School URN:</strong> {schoolURN}
              </div>
              <div>
                <strong>Academic Year:</strong> {configuredAcademicYear}
              </div>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: '10px',
                  flexWrap: 'wrap',
                }}
              >
                <label
                  htmlFor="theme-mode-toggle-switch"
                  style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
                >
                  <strong>High-Contrast Mode:</strong>{' '}
                  <span style={{ color: 'var(--mis-text-secondary)' }}>({themeMode})</span>
                </label>
                <button
                  id="theme-mode-toggle-switch"
                  type="button"
                  role="switch"
                  aria-checked={themeMode === 'high-contrast'}
                  aria-label="Toggle High-Contrast Theme"
                  data-testid="theme-mode-toggle-btn"
                  onClick={handleToggleThemeMode}
                  style={{
                    position: 'relative',
                    display: 'inline-flex',
                    alignItems: 'center',
                    width: '44px',
                    height: '24px',
                    padding: '2px',
                    borderRadius: '999px',
                    border: '1px solid var(--mis-border-strong)',
                    backgroundColor:
                      themeMode === 'high-contrast'
                        ? 'var(--mis-brand-primary)'
                        : 'var(--mis-bg-elevated)',
                    cursor: 'pointer',
                    transition: 'background-color 150ms ease-in-out',
                  }}
                >
                  <span
                    aria-hidden="true"
                    style={{
                      width: '18px',
                      height: '18px',
                      borderRadius: '50%',
                      backgroundColor:
                        themeMode === 'high-contrast'
                          ? 'var(--mis-brand-primary-fg)'
                          : 'var(--mis-text-primary)',
                      transform:
                        themeMode === 'high-contrast' ? 'translateX(20px)' : 'translateX(0px)',
                      transition: 'transform 150ms ease-in-out',
                    }}
                  />
                </button>
              </div>
              <div
                data-testid="base-font-size-control"
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: '6px',
                }}
              >
                <div>
                  <strong>Base Font Size:</strong>{' '}
                  <span style={{ color: 'var(--mis-text-secondary)' }}>
                    ({BASE_FONT_SIZE_PRESETS[baseFontSize].label} — {BASE_FONT_SIZE_PRESETS[baseFontSize].px})
                  </span>
                </div>
                <div
                  role="group"
                  aria-label="Base Font Size"
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(4, 1fr)',
                    gap: '6px',
                  }}
                >
                  {(['small', 'medium', 'large', 'extra-large'] as BaseFontSize[]).map((presetKey) => {
                    const preset = BASE_FONT_SIZE_PRESETS[presetKey];
                    const isSelected = baseFontSize === presetKey;
                    return (
                      <button
                        key={presetKey}
                        type="button"
                        data-testid={`font-size-btn-${presetKey}`}
                        aria-pressed={isSelected}
                        onClick={() => handleSelectBaseFontSize(presetKey)}
                        style={{
                          padding: '6px 8px',
                          borderRadius: 'var(--mis-radius-sm, 4px)',
                          border: isSelected
                            ? '1px solid var(--mis-brand-primary)'
                            : '1px solid var(--mis-border-subtle)',
                          backgroundColor: isSelected
                            ? 'var(--mis-brand-primary)'
                            : 'var(--mis-bg-elevated)',
                          color: isSelected
                            ? 'var(--mis-brand-primary-fg)'
                            : 'var(--mis-text-primary)',
                          fontSize: '12px',
                          fontWeight: isSelected ? 700 : 600,
                          cursor: 'pointer',
                        }}
                      >
                        {preset.label}
                      </button>
                    );
                  })}
                </div>
              </div>
              <div>
                <strong>Pinned Views:</strong> {pinnedRecents.length}
              </div>
            </div>
          </div>
        </div>
      )}

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
