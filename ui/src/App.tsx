import React from 'react';
import { AdminDiagnosticsPanel } from './AdminDiagnosticsPanel';
import { HostSidebar } from './components/HostSidebar';
import { HostTopbar } from './components/HostTopbar';
import { useHostIframeBridge } from './components/useHostIframeBridge';
import { DashboardWidgetGridPanel } from './DashboardWidgetGridPanel';
import { GdprSarPanel } from './GdprSarPanel';
import { SafeguardingVaultPanel } from './SafeguardingVaultPanel';
import { applyHostThemeToRoot, ThemeMode } from './themeAndTypes';
import { DashboardSummaryStrip } from './views/DashboardSummaryStrip';
import { KeepAlivePluginWorkspace } from './views/KeepAlivePluginWorkspace';
import { useHostAdminAndDslState } from './views/useHostAdminAndDslState';
import { useHostWorkspaceState } from './views/useHostWorkspaceState';

export function App() {
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
  } = useHostWorkspaceState();

  const {
    handleRegisterPluginIframeRef,
    handleRegisterWidgetIframeRef,
    buildInitPayload,
    broadcastInitToMountedIframes,
  } = useHostIframeBridge({
    themeMode,
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
    devPersonaSwitcherEnabled,
    availablePersonas,
    personaSwitchError,
    handleSelectPersona,
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
    applyInstalledPluginsState,
    broadcastInitToMountedIframes,
  });

  const handleToggleThemeMode = () => {
    const nextMode: ThemeMode = themeMode === 'light' ? 'high-contrast' : 'light';
    applyHostThemeToRoot(nextMode, themeCatalog);
    setThemeMode(nextMode);
    broadcastInitToMountedIframes(undefined, nextMode);
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
      <HostSidebar
        schoolURN={schoolURN}
        schoolName={schoolName}
        configuredAcademicYear={configuredAcademicYear}
        installedPlugins={installedPlugins}
        activePluginId={activePluginId}
        activeEntityContext={activeEntityContext}
        onActivatePlugin={activatePluginInKeepAlivePool}
      />

      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
        <HostTopbar
          deepLinkNavigationStatus={deepLinkNavigationStatus}
          activeRouteUrl={activeRouteUrl}
          devPersonaSwitcherEnabled={devPersonaSwitcherEnabled}
          availablePersonas={availablePersonas}
          activeCaller={activeCaller}
          activeStaffId={activeStaffId}
          personaSwitchError={personaSwitchError}
          themeMode={themeMode}
          onSelectPersona={handleSelectPersona}
          onToggleThemeMode={handleToggleThemeMode}
        />

        <main
          data-testid="host-dashboard-grid"
          style={{
            padding: '20px 24px',
            display: 'flex',
            flexDirection: 'column',
            gap: '20px',
          }}
        >
          <DashboardSummaryStrip
            routeError={routeError}
            studentEntityLinkHandler={entityLinkHandlers.student}
          />

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
    </div>
  );
}
