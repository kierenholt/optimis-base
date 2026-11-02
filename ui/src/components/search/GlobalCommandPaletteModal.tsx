import React, { useEffect, useRef, useState } from 'react';
import { PinnedRecentEntry, PluginViewDescriptor } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { SearchCategoryTabs } from './SearchCategoryTabs';
import { SearchResultsList } from './SearchResultsList';
import { searchInputStyle } from './searchStyles';
import { CommandPaletteTab, GlobalSearchSharedProps } from './searchTypes';
import { useGlobalSearchIndex } from './useGlobalSearchIndex';

export interface GlobalCommandPaletteModalProps extends GlobalSearchSharedProps {
  isOpen: boolean;
  onClose: () => void;
  pinnedRecents?: PinnedRecentEntry[];
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const GlobalCommandPaletteModal: React.FC<GlobalCommandPaletteModalProps> = ({
  isOpen,
  onClose,
  installedPlugins,
  placedWidgets,
  activePupilId,
  activeGroupId,
  activeStaffId,
  referenceDate,
  academicYearId,
  pinnedRecents = [],
  onPlaceWidget,
  onSelectViewResult,
  onSelectPupilResult,
  onSelectGroupResult,
  onTogglePinView,
}) => {
  const [query, setQuery] = useState<string>('');
  const [activeTab, setActiveTab] = useState<CommandPaletteTab>('all');
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setActiveTab('all');
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    }
  }, [isOpen]);

  const {
    pupils,
    groups,
    matchedViews,
    matchedWidgets,
    placedWidgetKeys,
    pinnedRoutesSet,
    showViews,
    showWidgets,
    showPupils,
    showGroups,
    counts,
  } = useGlobalSearchIndex({
    query,
    installedPlugins,
    placedWidgets,
    pinnedRecents,
    activePupilId,
    activeGroupId,
    activeStaffId,
    referenceDate,
    academicYearId,
    activeTab,
    enabled: isOpen,
    allowEmptyQuery: true,
    requireQueryForLocal: false,
  });

  if (!isOpen) {
    return null;
  }

  return (
    <div
      data-testid="command-palette-backdrop"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(15, 23, 42, 0.55)',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'flex-start',
        paddingTop: '80px',
        zIndex: 9999,
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Global Command Palette"
        data-testid="command-palette-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: '100%',
          maxWidth: '640px',
          maxHeight: '75vh',
          backgroundColor: 'var(--mis-bg-surface)',
          color: 'var(--mis-text-primary)',
          border: '1px solid var(--mis-border-strong)',
          borderRadius: 'var(--mis-radius-md, 8px)',
          boxShadow: 'var(--mis-shadow-card)',
          padding: '16px',
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          overflowY: 'auto',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: 'calc(14px * var(--mis-font-scale, 1))', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
            <MonochromeLucideIcon name="search" size={16} />
            <span>Full list (Ctrl+K)</span>
          </span>
          <button
            type="button"
            aria-label="Close"
            title="Close (Esc)"
            data-testid="command-palette-close-btn"
            onClick={onClose}
            style={{
              padding: '4px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-subtle)',
              backgroundColor: 'var(--mis-bg-elevated)',
              color: 'var(--mis-text-primary)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
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

        <input
          ref={inputRef}
          type="search"
          autoFocus
          data-testid="command-palette-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Type a command, synonym, pupil name, or group code..."
          style={searchInputStyle}
        />

        <SearchCategoryTabs
          ariaLabel="Category filter tabs"
          testId="command-palette-tabs"
          tabTestIdPrefix="command-palette-tab"
          activeTab={activeTab}
          onSelectTab={setActiveTab}
          counts={counts}
          borderBottom
        />

        <SearchResultsList
          containerTestId="command-palette-results"
          viewsTitle="Views"
          viewResultTestIdPrefix="command-palette-result-view"
          viewPinTestIdPrefix="search-pin-view-btn"
          widgetResultTestIdPrefix="command-palette-result-widget"
          widgetAddBtnTestIdPrefix="command-palette-add-widget-btn"
          pupilResultTestIdPrefix="command-palette-result-pupil"
          groupResultTestIdPrefix="command-palette-result-group"
          emptyStateTestId="command-palette-empty-state"
          emptyStateMessage="No matching items in this category."
          viewActionLabel="Jump"
          showPermittedMethods={true}
          matchedViews={matchedViews}
          matchedWidgets={matchedWidgets}
          pupils={pupils}
          groups={groups}
          placedWidgetKeys={placedWidgetKeys}
          pinnedRoutesSet={pinnedRoutesSet}
          showViews={showViews}
          showWidgets={showWidgets}
          showPupils={showPupils}
          showGroups={showGroups}
          activePupilId={activePupilId}
          activeGroupId={activeGroupId}
          activeStaffId={activeStaffId}
          onSelectViewResult={onSelectViewResult}
          onPlaceWidget={onPlaceWidget}
          onSelectPupilResult={onSelectPupilResult}
          onSelectGroupResult={onSelectGroupResult}
          onTogglePinView={onTogglePinView}
          onAfterItemSelect={onClose}
        />
      </div>
    </div>
  );
};
