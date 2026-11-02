import React from 'react';
import { SearchGroupResult } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { resultItemButtonStyle } from './searchStyles';

export interface SearchResultGroupItemProps {
  group: SearchGroupResult;
  testId: string;
  onSelect: (groupId: string) => void;
}

export const SearchResultGroupItem: React.FC<SearchResultGroupItemProps> = ({
  group,
  testId,
  onSelect,
}) => (
  <button
    type="button"
    data-testid={testId}
    onClick={() => onSelect(group.id)}
    style={resultItemButtonStyle}
  >
    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
      <MonochromeLucideIcon name="users" size={16} />
      <div>
        <div style={{ fontSize: 'calc(13px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
          {group.code} — {group.name}
        </div>
        <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)' }}>
          {group.group_type}
        </div>
      </div>
    </div>
    <span style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
      <span>Open Group</span>
      <MonochromeLucideIcon name="arrow-right" size={12} />
    </span>
  </button>
);
