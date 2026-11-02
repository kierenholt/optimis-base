import React from 'react';
import { AdminDiagnosticsPanel } from '../AdminDiagnosticsPanel';
import { LandingPageTopHalfSearch } from '../components/GlobalSearchAndCommandPalette';
import { DashboardWidgetGridPanel } from '../DashboardWidgetGridPanel';
import { GdprSarPanel } from '../GdprSarPanel';
import { SafeguardingVaultPanel } from '../SafeguardingVaultPanel';
import {
  DecryptedSafeguardingNote,
  InstalledPlugin,
  PinnedRecentItem,
  PlacedDashboardWidget,
  PluginViewDescriptor,
  PluginWidgetDescriptor,
  SafeguardingNoteEnvelope,
  SARSection,
} from '../themeAndTypes';

export interface HostHomeDashboardViewProps {
  isVisible: boolean;
  installedPlugins: InstalledPlugin[];
  placedWidgets: PlacedDashboardWidget[];
  entityLinkHandlers: Record<string, string>;
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  referenceDate: string;
  academicYearId: string;
  pinnedRecents: PinnedRecentItem[];
  onPlaceWidget: (widget: PluginWidgetDescriptor, targetCol?: number, targetRow?: number) => void;
  onSelectSearchViewResult: (pluginId: string, view: PluginViewDescriptor) => void;
  onSelectSearchPupilResult: (pupilId: string) => void;
  onSelectSearchGroupResult: (groupId: string) => void;
  onTogglePinSearchView: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
  onOpenCommandPalette: () => void;
  isDashboardEditMode: boolean;
  dashboardColumnCount: number;
  availableWidgets: PluginWidgetDescriptor[];
  onToggleDashboardEditMode: () => void;
  onRemoveWidget: (compositeKey: string) => void;
  onMoveWidgetUp: (compositeKey: string) => void;
  onMoveWidgetDown: (compositeKey: string) => void;
  onMoveWidgetLeft: (compositeKey: string) => void;
  onMoveWidgetRight: (compositeKey: string) => void;
  onAddDashboardColumn: () => void;
  onRemoveDashboardColumn: () => void;
  dashboardSaveStatus: string;
  onRegisterWidgetIframeRef: (compositeKey: string, el: HTMLIFrameElement | null) => void;
  hasSafeguardingRead: boolean;
  safeguardingEnvelopes: SafeguardingNoteEnvelope[];
  cloudKeyToken: string;
  onCloudKeyTokenChange: (token: string) => void;
  onRenewCloudKey: () => void;
  onDecryptSafeguardingNote: (envelopeId: string) => void;
  decryptedNotes: Record<string, DecryptedSafeguardingNote>;
  dslVaultError: string;
  hasAdminOrDpoAccess: boolean;
  sarSections: SARSection[];
  sarExportStatus: string;
  sarFinalizedJson: string;
  sarPurgeStatus: string;
  onLoadSarExport: () => void;
  onFinalizeSarExport: () => void;
  onPurgePupilData: () => void;
  onUpdateRedactableField: (secIdx: number, recIdx: number, fieldKey: string, newValue: string) => void;
  diagnosticBundleStatus: string;
  diagnosticBundleJson: string;
  diagnosticBundleLog: string;
  onDownloadDiagnosticBundle: () => void;
}

export function HostHomeDashboardView({
  isVisible,
  installedPlugins,
  placedWidgets,
  entityLinkHandlers,
  activePupilId,
  activeGroupId,
  activeStaffId,
  referenceDate,
  academicYearId,
  pinnedRecents,
  onPlaceWidget,
  onSelectSearchViewResult,
  onSelectSearchPupilResult,
  onSelectSearchGroupResult,
  onTogglePinSearchView,
  onOpenCommandPalette,
  isDashboardEditMode,
  dashboardColumnCount,
  availableWidgets,
  onToggleDashboardEditMode,
  onRemoveWidget,
  onMoveWidgetUp,
  onMoveWidgetDown,
  onMoveWidgetLeft,
  onMoveWidgetRight,
  onAddDashboardColumn,
  onRemoveDashboardColumn,
  dashboardSaveStatus,
  onRegisterWidgetIframeRef,
  hasSafeguardingRead,
  safeguardingEnvelopes,
  cloudKeyToken,
  onCloudKeyTokenChange,
  onRenewCloudKey,
  onDecryptSafeguardingNote,
  decryptedNotes,
  dslVaultError,
  hasAdminOrDpoAccess,
  sarSections,
  sarExportStatus,
  sarFinalizedJson,
  sarPurgeStatus,
  onLoadSarExport,
  onFinalizeSarExport,
  onPurgePupilData,
  onUpdateRedactableField,
  diagnosticBundleStatus,
  diagnosticBundleJson,
  diagnosticBundleLog,
  onDownloadDiagnosticBundle,
}: HostHomeDashboardViewProps) {
  return (
    <div
      style={{
        display: isVisible ? 'flex' : 'none',
        flexDirection: 'column',
        gap: '20px',
      }}
    >
      <LandingPageTopHalfSearch
        installedPlugins={installedPlugins}
        placedWidgets={placedWidgets}
        entityLinkHandlers={entityLinkHandlers}
        activePupilId={activePupilId}
        activeGroupId={activeGroupId}
        activeStaffId={activeStaffId}
        referenceDate={referenceDate}
        academicYearId={academicYearId}
        pinnedRecents={pinnedRecents}
        onPlaceWidget={onPlaceWidget}
        onSelectViewResult={onSelectSearchViewResult}
        onSelectPupilResult={onSelectSearchPupilResult}
        onSelectGroupResult={onSelectSearchGroupResult}
        onTogglePinView={onTogglePinSearchView}
        onOpenCommandPalette={onOpenCommandPalette}
      />

      <DashboardWidgetGridPanel
        isEditMode={isDashboardEditMode}
        columnCount={dashboardColumnCount}
        availableWidgets={availableWidgets}
        placedWidgets={placedWidgets}
        onToggleEditMode={onToggleDashboardEditMode}
        onPlaceWidget={onPlaceWidget}
        onRemoveWidget={onRemoveWidget}
        onMoveWidgetUp={onMoveWidgetUp}
        onMoveWidgetDown={onMoveWidgetDown}
        onMoveWidgetLeft={onMoveWidgetLeft}
        onMoveWidgetRight={onMoveWidgetRight}
        onAddColumn={onAddDashboardColumn}
        onRemoveColumn={onRemoveDashboardColumn}
        dashboardSaveStatus={dashboardSaveStatus}
        onRegisterWidgetIframeRef={onRegisterWidgetIframeRef}
      />

      {hasSafeguardingRead && (
        <SafeguardingVaultPanel
          safeguardingEnvelopes={safeguardingEnvelopes}
          cloudKeyToken={cloudKeyToken}
          onCloudKeyTokenChange={onCloudKeyTokenChange}
          onRenewCloudKey={onRenewCloudKey}
          onDecryptNote={onDecryptSafeguardingNote}
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
          onLoadSarExport={onLoadSarExport}
          onFinalizeSarExport={onFinalizeSarExport}
          onPurgePupilData={onPurgePupilData}
          onUpdateRedactableField={onUpdateRedactableField}
        />
      )}

      {hasAdminOrDpoAccess && (
        <AdminDiagnosticsPanel
          diagnosticBundleStatus={diagnosticBundleStatus}
          diagnosticBundleJson={diagnosticBundleJson}
          diagnosticBundleLog={diagnosticBundleLog}
          onDownloadDiagnosticBundle={onDownloadDiagnosticBundle}
        />
      )}
    </div>
  );
}
