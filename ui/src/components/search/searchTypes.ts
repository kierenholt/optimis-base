import {
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginViewDescriptor,
  PluginWidgetDescriptor,
} from '../../themeAndTypes';

export interface IndexedViewResult {
  pluginId: string;
  pluginTitle: string;
  pluginIcon?: string;
  view: PluginViewDescriptor;
  permittedMethods: string[];
}

export interface GlobalSearchSharedProps {
  installedPlugins: PluginNavEntry[];
  placedWidgets: PlacedDashboardWidget[];
  entityLinkHandlers: Record<string, string>;
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  referenceDate: string;
  academicYearId: string;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
  onSelectViewResult: (pluginId: string, view: PluginViewDescriptor) => void;
  onSelectPupilResult: (pupilId: string) => void;
  onSelectGroupResult: (groupId: string) => void;
}

export type CommandPaletteTab = 'all' | 'plugins' | 'widgets' | 'pupils' | 'groups';

export const FILTER_TAB_DEFS: Array<{ id: CommandPaletteTab; label: string; icon: string }> = [
  { id: 'all', label: 'All', icon: 'search' },
  { id: 'plugins', label: 'Pages', icon: 'box' },
  { id: 'widgets', label: 'Widgets', icon: 'layout-grid' },
  { id: 'pupils', label: 'Pupils', icon: 'user' },
  { id: 'groups', label: 'Groups', icon: 'users' },
];

export interface SearchCategoryCounts {
  all: number;
  plugins: number;
  widgets: number;
  pupils: number;
  groups: number;
}
