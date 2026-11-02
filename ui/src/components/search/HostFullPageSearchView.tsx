import React, { useEffect, useRef, useState } from 'react';
import { PinnedRecentEntry, PluginViewDescriptor } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { SearchCategoryTabs } from './SearchCategoryTabs';
import { SearchResultsList } from './SearchResultsList';
import { searchInputStyle } from './searchStyles';
import { CommandPaletteTab, GlobalSearchSharedProps } from './searchTypes';
import { useGlobalSearchIndex } from './useGlobalSearchIndex';

export interface HostFullPageSearchViewProps extends GlobalSearchSharedProps {
  pinnedRecents?: PinnedRecentEntry[];
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const HostFullPageSearchView: React.FC<HostFullPageSearchViewProps> = ({
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
    inputRef.current?.focus();
  }, []);

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
    enabled: true,
    allowEmptyQuery: true,
    requireQueryForLocal: false,
  });

  return (
    <section
      data-testid="host-search-view"
      aria-label="Global Search and Discovery"
      style={{
        backgroundColor: 'var(--mis-bg-surface)',
        color: 'var(--mis-text-primary)',
        border: '1px solid var(--mis-border-strong)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        boxShadow: 'var(--mis-shadow-card)',
        padding: '20px',
        display: 'flex',
        flexDirection: 'column',
        gap: '14px',
      }}
    >
      <div>
        <h2 style={{ margin: 0, fontSize: 'calc(16px * var(--mis-font-scale, 1))', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '8px' }}>
          <MonochromeLucideIcon name="search" size={16} />
          <span>Global Search &amp; Discovery</span>
        </h2>
        <div style={{ fontSize: 'calc(12px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)', marginTop: '4px' }}>
          Search plugin views, dashboard widgets, pupils, and teaching groups across your school workspace
        </div>
      </div>

      <input
        ref={inputRef}
        type="search"
        autoFocus
        data-testid="host-search-input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Type a command, synonym, pupil name, or group code..."
        style={searchInputStyle}
      />

      <SearchCategoryTabs
        ariaLabel="Search category filter tabs"
        testId="host-search-tabs"
        tabTestIdPrefix="host-search-tab"
        activeTab={activeTab}
        onSelectTab={setActiveTab}
        counts={counts}
        borderBottom
      />

      <SearchResultsList
        containerTestId="host-search-results"
        viewsTitle="Views"
        viewResultTestIdPrefix="host-search-result-view"
        viewPinTestIdPrefix="host-search-pin-view-btn"
        widgetResultTestIdPrefix="host-search-result-widget"
        widgetAddBtnTestIdPrefix="host-search-add-widget-btn"
        pupilResultTestIdPrefix="host-search-result-pupil"
        groupResultTestIdPrefix="host-search-result-group"
        emptyStateTestId="host-search-empty-state"
        emptyStateMessage="No matching items in this category."
        viewActionLabel="Open View"
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
      />
    </section>
  );
};
