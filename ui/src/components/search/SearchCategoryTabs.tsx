import React, { useMemo } from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { tabButtonStyle } from './searchStyles';
import { CommandPaletteTab, FILTER_TAB_DEFS, SearchCategoryCounts } from './searchTypes';

export interface SearchCategoryTabsProps {
  ariaLabel: string;
  testId: string;
  tabTestIdPrefix: string;
  activeTab: CommandPaletteTab;
  onSelectTab: (tab: CommandPaletteTab) => void;
  counts?: SearchCategoryCounts;
  borderBottom?: boolean;
}

export const SearchCategoryTabs: React.FC<SearchCategoryTabsProps> = ({
  ariaLabel,
  testId,
  tabTestIdPrefix,
  activeTab,
  onSelectTab,
  counts,
  borderBottom = false,
}) => {
  const tabs = useMemo(() => {
    return FILTER_TAB_DEFS.map((t) => {
      let label = t.label;
      if (counts) {
        if (t.id === 'all') label = `All (${counts.all})`;
        else if (t.id === 'plugins') label = `Pages (${counts.plugins})`;
        else if (t.id === 'widgets') label = `Widgets (${counts.widgets})`;
        else if (t.id === 'pupils') label = `Pupils (${counts.pupils})`;
        else if (t.id === 'groups') label = `Groups (${counts.groups})`;
      }
      return { ...t, displayLabel: label };
    });
  }, [counts]);

  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      data-testid={testId}
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '6px',
        flexWrap: 'wrap',
        borderBottom: borderBottom ? '1px solid var(--mis-border-subtle)' : undefined,
        paddingBottom: borderBottom ? '8px' : undefined,
      }}
    >
      {tabs.map((t) => {
        const isSelected = activeTab === t.id;
        return (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={isSelected}
            data-testid={`${tabTestIdPrefix}-${t.id}`}
            onClick={() => onSelectTab(t.id)}
            style={tabButtonStyle(isSelected)}
          >
            <MonochromeLucideIcon name={t.icon} size={13} />
            <span>{t.displayLabel}</span>
          </button>
        );
      })}
    </div>
  );
};
