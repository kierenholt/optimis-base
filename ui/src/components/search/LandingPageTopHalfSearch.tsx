import React, { useState } from 'react';
import { PinnedRecentEntry, PluginViewDescriptor } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { SearchCategoryTabs } from './SearchCategoryTabs';
import { SearchResultsList } from './SearchResultsList';
import { searchInputStyle } from './searchStyles';
import { CommandPaletteTab, GlobalSearchSharedProps } from './searchTypes';
import { useGlobalSearchIndex } from './useGlobalSearchIndex';

export interface LandingPageTopHalfSearchProps extends GlobalSearchSharedProps {
  onOpenCommandPalette: () => void;
  pinnedRecents?: PinnedRecentEntry[];
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const LandingPageTopHalfSearch: React.FC<LandingPageTopHalfSearchProps> = ({
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
  onOpenCommandPalette,
}) => {
  const [query, setQuery] = useState<string>('');
  const [activeFilter, setActiveFilter] = useState<CommandPaletteTab>('all');

  const {
    qLower,
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
    activeTab: activeFilter,
    enabled: query.trim().length > 0,
    allowEmptyQuery: false,
    requireQueryForLocal: true,
  });

  return (
    <section
      data-testid="landing-page-search-panel"
      style={{
        backgroundColor: 'var(--mis-bg-surface)',
        border: '1px solid var(--mis-border-strong)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        padding: '16px',
        boxShadow: 'var(--mis-shadow-card)',
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '12px',
          flexWrap: 'wrap',
        }}
      >
        <div>
          <h2 style={{ fontSize: 'calc(15px * var(--mis-font-scale, 1))', fontWeight: 700, margin: 0, display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
            <MonochromeLucideIcon name="search" size={16} />
            <span>Global Search &amp; Discovery</span>
          </h2>
          <div style={{ fontSize: 'calc(12px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)', marginTop: '2px' }}>
            Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
          </div>
        </div>
        <button
          type="button"
          data-testid="landing-open-command-palette-btn"
          onClick={onOpenCommandPalette}
          style={{
            padding: '5px 10px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            border: '1px solid var(--mis-border-subtle)',
            backgroundColor: 'var(--mis-bg-elevated)',
            color: 'var(--mis-text-primary)',
            fontSize: 'calc(12px * var(--mis-font-scale, 1))',
            fontWeight: 600,
            cursor: 'pointer',
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
          }}
        >
          <MonochromeLucideIcon name="search" size={14} />
          <span>Full list</span>
        </button>
      </div>

      <SearchCategoryTabs
        ariaLabel="Search record type filter"
        testId="landing-search-filter-tabs"
        tabTestIdPrefix="landing-search-filter"
        activeTab={activeFilter}
        onSelectTab={setActiveFilter}
      />

      <input
        type="search"
        data-testid="landing-search-input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..."
        style={searchInputStyle}
      />

      {qLower.length > 0 && (
        <SearchResultsList
          containerTestId="landing-search-results"
          viewsTitle="Plugin Views"
          viewResultTestIdPrefix="landing-search-result-view"
          viewPinTestIdPrefix="landing-search-pin-view-btn"
          widgetResultTestIdPrefix="landing-search-result-widget"
          widgetAddBtnTestIdPrefix="landing-search-add-widget-btn"
          pupilResultTestIdPrefix="landing-search-result-pupil"
          groupResultTestIdPrefix="landing-search-result-group"
          emptyStateTestId="landing-search-empty-state"
          emptyStateMessage="No matching views, widgets, pupils, or groups found."
          viewActionLabel="Open View"
          showPermittedMethods={false}
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
      )}
    </section>
  );
};
