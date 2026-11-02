import { useMemo } from 'react';
import {
  getWidgetCompositeKey,
  PinnedRecentEntry,
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginWidgetDescriptor,
} from '../../themeAndTypes';
import { filterPluginViews, filterPluginWidgets } from './searchMatchingHelpers';
import { CommandPaletteTab, IndexedViewResult, SearchCategoryCounts } from './searchTypes';
import { useScopedEntitySearch } from './useScopedEntitySearch';

export function useGlobalSearchIndex({
  query,
  installedPlugins,
  placedWidgets,
  pinnedRecents = [],
  activePupilId,
  activeGroupId,
  activeStaffId,
  referenceDate,
  academicYearId,
  activeTab,
  enabled = true,
  allowEmptyQuery = false,
  requireQueryForLocal = false,
}: {
  query: string;
  installedPlugins: PluginNavEntry[];
  placedWidgets: PlacedDashboardWidget[];
  pinnedRecents?: PinnedRecentEntry[];
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  referenceDate: string;
  academicYearId: string;
  activeTab: CommandPaletteTab;
  enabled?: boolean;
  allowEmptyQuery?: boolean;
  requireQueryForLocal?: boolean;
}) {
  const qLower = query.trim().toLowerCase();

  const { pupils, groups } = useScopedEntitySearch(
    query,
    activeStaffId,
    referenceDate,
    academicYearId,
    enabled,
    allowEmptyQuery
  );

  const matchedViews = useMemo<IndexedViewResult[]>(() => {
    return filterPluginViews(
      installedPlugins,
      qLower,
      activePupilId,
      activeGroupId,
      activeStaffId,
      requireQueryForLocal
    );
  }, [installedPlugins, qLower, activePupilId, activeGroupId, activeStaffId, requireQueryForLocal]);

  const matchedWidgets = useMemo<Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }>>(() => {
    return filterPluginWidgets(installedPlugins, qLower, requireQueryForLocal);
  }, [installedPlugins, qLower, requireQueryForLocal]);

  const placedWidgetKeys = useMemo(
    () => new Set(placedWidgets.map((w) => getWidgetCompositeKey(w.plugin_id, w.id))),
    [placedWidgets]
  );

  const pinnedRoutesSet = useMemo(
    () => new Set(pinnedRecents.map((r) => r.route)),
    [pinnedRecents]
  );

  const showViews = (activeTab === 'all' || activeTab === 'plugins') && matchedViews.length > 0;
  const showWidgets = (activeTab === 'all' || activeTab === 'widgets') && matchedWidgets.length > 0;
  const showPupils = (activeTab === 'all' || activeTab === 'pupils') && pupils.length > 0;
  const showGroups = (activeTab === 'all' || activeTab === 'groups') && groups.length > 0;
  const hasAnyResults = showViews || showWidgets || showPupils || showGroups;

  const counts: SearchCategoryCounts = useMemo(
    () => ({
      all: matchedViews.length + matchedWidgets.length + pupils.length + groups.length,
      plugins: matchedViews.length,
      widgets: matchedWidgets.length,
      pupils: pupils.length,
      groups: groups.length,
    }),
    [matchedViews.length, matchedWidgets.length, pupils.length, groups.length]
  );

  return {
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
    hasAnyResults,
    counts,
  };
}
