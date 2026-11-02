import React from 'react';
import { PluginViewDescriptor } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { pinButtonStyle, resultItemButtonStyle } from './searchStyles';
import { IndexedViewResult } from './searchTypes';

export interface SearchResultViewItemProps {
  item: IndexedViewResult;
  testId: string;
  pinTestId: string;
  actionLabel: 'Open View' | 'Jump';
  showPermittedMethods: boolean;
  activePupilId?: string;
  activeGroupId?: string;
  activeStaffId?: string;
  isPinned: boolean;
  onSelect: (pluginId: string, view: PluginViewDescriptor) => void;
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const SearchResultViewItem: React.FC<SearchResultViewItemProps> = ({
  item,
  testId,
  pinTestId,
  actionLabel,
  showPermittedMethods,
  isPinned,
  onSelect,
  onTogglePinView,
}) => {
  const { pluginId, pluginTitle, pluginIcon, view, permittedMethods } = item;

  return (
    <div style={{ display: 'flex', alignItems: 'stretch', gap: '6px' }}>
      <button
        type="button"
        data-testid={testId}
        data-permitted-count={showPermittedMethods ? String(permittedMethods.length) : undefined}
        data-permitted-methods={showPermittedMethods ? permittedMethods.join(',') : undefined}
        onClick={() => onSelect(pluginId, view)}
        style={resultItemButtonStyle}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <MonochromeLucideIcon name={view.icon || pluginIcon || 'box'} fallback="box" size={16} />
          <div>
            <div style={{ fontSize: 'calc(13px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
              {view.title || pluginTitle}
            </div>
            <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)' }}>
              {pluginTitle} • /p/{pluginId}
              {view.path === '/' ? '' : view.path}
              {showPermittedMethods ? ` (${permittedMethods.length} methods)` : ''}
            </div>
          </div>
        </div>
        <span style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
          <span>{actionLabel}</span>
          <MonochromeLucideIcon name="arrow-right" size={12} />
        </span>
      </button>
      {onTogglePinView && (
        <button
          type="button"
          data-testid={pinTestId}
          aria-label={isPinned ? `Unpin ${view.title || pluginTitle}` : `Pin ${view.title || pluginTitle}`}
          onClick={() => onTogglePinView(pluginId, view, pluginTitle)}
          style={pinButtonStyle(isPinned)}
        >
          <MonochromeLucideIcon name="pin" size={12} />
          <span>{isPinned ? 'Pinned' : 'Pin'}</span>
        </button>
      )}
    </div>
  );
};
