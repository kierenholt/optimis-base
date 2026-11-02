import React from 'react';
import {
  getWidgetCompositeKey,
  PluginViewDescriptor,
  PluginWidgetDescriptor,
  SearchGroupResult,
  SearchPupilResult,
} from '../../themeAndTypes';
import { resolveParameterizedViewSubpath } from './searchMatchingHelpers';
import { emptyStateStyle, sectionTitleStyle } from './searchStyles';
import { IndexedViewResult } from './searchTypes';
import { SearchResultGroupItem } from './SearchResultGroupItem';
import { SearchResultPupilItem } from './SearchResultPupilItem';
import { SearchResultViewItem } from './SearchResultViewItem';
import { SearchResultWidgetItem } from './SearchResultWidgetItem';

export interface SearchResultsListProps {
  containerTestId: string;
  viewsTitle: string;
  viewResultTestIdPrefix: string;
  viewPinTestIdPrefix: string;
  widgetResultTestIdPrefix: string;
  widgetAddBtnTestIdPrefix: string;
  pupilResultTestIdPrefix: string;
  groupResultTestIdPrefix: string;
  emptyStateTestId: string;
  emptyStateMessage: string;
  viewActionLabel: 'Open View' | 'Jump';
  showPermittedMethods: boolean;
  matchedViews: IndexedViewResult[];
  matchedWidgets: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }>;
  pupils: SearchPupilResult[];
  groups: SearchGroupResult[];
  placedWidgetKeys: Set<string>;
  pinnedRoutesSet: Set<string>;
  showViews: boolean;
  showWidgets: boolean;
  showPupils: boolean;
  showGroups: boolean;
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  onSelectViewResult: (pluginId: string, view: PluginViewDescriptor) => void;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
  onSelectPupilResult: (pupilId: string) => void;
  onSelectGroupResult: (groupId: string) => void;
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
  onAfterItemSelect?: () => void;
}

export const SearchResultsList: React.FC<SearchResultsListProps> = ({
  containerTestId,
  viewsTitle,
  viewResultTestIdPrefix,
  viewPinTestIdPrefix,
  widgetResultTestIdPrefix,
  widgetAddBtnTestIdPrefix,
  pupilResultTestIdPrefix,
  groupResultTestIdPrefix,
  emptyStateTestId,
  emptyStateMessage,
  viewActionLabel,
  showPermittedMethods,
  matchedViews,
  matchedWidgets,
  pupils,
  groups,
  placedWidgetKeys,
  pinnedRoutesSet,
  showViews,
  showWidgets,
  showPupils,
  showGroups,
  activePupilId,
  activeGroupId,
  activeStaffId,
  onSelectViewResult,
  onPlaceWidget,
  onSelectPupilResult,
  onSelectGroupResult,
  onTogglePinView,
  onAfterItemSelect,
}) => {
  const hasAny = showViews || showWidgets || showPupils || showGroups;

  return (
    <div
      data-testid={containerTestId}
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: '10px',
        paddingTop: containerTestId === 'landing-search-results' ? '4px' : undefined,
      }}
    >
      {showViews && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div style={sectionTitleStyle}>
            {viewsTitle} ({matchedViews.length})
          </div>
          {matchedViews.map((item) => {
            const candidateSubpath = resolveParameterizedViewSubpath(
              item.view.path,
              activePupilId,
              activeGroupId,
              activeStaffId
            );
            const candidateRoute = `/p/${item.pluginId}${candidateSubpath === '/' ? '' : candidateSubpath}`;
            return (
              <SearchResultViewItem
                key={`${item.pluginId}:${item.view.id}`}
                item={item}
                testId={`${viewResultTestIdPrefix}-${item.pluginId}-${item.view.id}`}
                pinTestId={`${viewPinTestIdPrefix}-${item.pluginId}-${item.view.id}`}
                actionLabel={viewActionLabel}
                showPermittedMethods={showPermittedMethods}
                activePupilId={activePupilId}
                activeGroupId={activeGroupId}
                activeStaffId={activeStaffId}
                isPinned={pinnedRoutesSet.has(candidateRoute)}
                onSelect={(pid, v) => {
                  onSelectViewResult(pid, v);
                  onAfterItemSelect?.();
                }}
                onTogglePinView={onTogglePinView}
              />
            );
          })}
        </div>
      )}

      {showWidgets && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div style={sectionTitleStyle}>
            Dashboard Widgets ({matchedWidgets.length})
          </div>
          {matchedWidgets.map(({ widget, pluginTitle, pluginIcon }) => {
            const compositeKey = getWidgetCompositeKey(widget.plugin_id, widget.id);
            return (
              <SearchResultWidgetItem
                key={compositeKey}
                widget={widget}
                pluginTitle={pluginTitle}
                pluginIcon={pluginIcon}
                isPlaced={placedWidgetKeys.has(compositeKey)}
                containerTestId={`${widgetResultTestIdPrefix}-${widget.id}`}
                buttonTestId={`${widgetAddBtnTestIdPrefix}-${widget.id}`}
                onPlaceWidget={onPlaceWidget}
              />
            );
          })}
        </div>
      )}

      {showPupils && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div style={sectionTitleStyle}>
            Pupils ({pupils.length})
          </div>
          {pupils.map((pupil) => (
            <SearchResultPupilItem
              key={pupil.id}
              pupil={pupil}
              testId={`${pupilResultTestIdPrefix}-${pupil.id}`}
              onSelect={(pid) => {
                onSelectPupilResult(pid);
                onAfterItemSelect?.();
              }}
            />
          ))}
        </div>
      )}

      {showGroups && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
          <div style={sectionTitleStyle}>
            Groups ({groups.length})
          </div>
          {groups.map((group) => (
            <SearchResultGroupItem
              key={group.id}
              group={group}
              testId={`${groupResultTestIdPrefix}-${group.id}`}
              onSelect={(gid) => {
                onSelectGroupResult(gid);
                onAfterItemSelect?.();
              }}
            />
          ))}
        </div>
      )}

      {!hasAny && (
        <div data-testid={emptyStateTestId} style={emptyStateStyle}>
          {emptyStateMessage}
        </div>
      )}
    </div>
  );
};
