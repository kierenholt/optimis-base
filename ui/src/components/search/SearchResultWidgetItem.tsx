import React from 'react';
import { getWidgetCompositeKey, PluginWidgetDescriptor } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { addWidgetButtonStyle, widgetItemContainerStyle } from './searchStyles';

export interface SearchResultWidgetItemProps {
  widget: PluginWidgetDescriptor;
  pluginTitle: string;
  pluginIcon?: string;
  isPlaced: boolean;
  containerTestId: string;
  buttonTestId: string;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
}

export const SearchResultWidgetItem: React.FC<SearchResultWidgetItemProps> = ({
  widget,
  pluginTitle,
  pluginIcon,
  isPlaced,
  containerTestId,
  buttonTestId,
  onPlaceWidget,
}) => {
  const compositeKey = getWidgetCompositeKey(widget.plugin_id, widget.id);
  return (
    <div
      key={compositeKey}
      data-testid={containerTestId}
      data-plug-id={widget.plugin_id}
      data-widget-key={compositeKey}
      style={widgetItemContainerStyle}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
        <MonochromeLucideIcon name={widget.icon || pluginIcon || 'layout-grid'} fallback="layout-grid" size={16} />
        <div>
          <div style={{ fontSize: 'calc(13px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
            {widget.title}
          </div>
          <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)' }}>
            Widget • {pluginTitle}
          </div>
        </div>
      </div>
      <button
        type="button"
        data-testid={buttonTestId}
        data-plug-id={widget.plugin_id}
        data-widget-key={compositeKey}
        disabled={isPlaced}
        onClick={() => onPlaceWidget(widget)}
        style={addWidgetButtonStyle(isPlaced)}
      >
        <MonochromeLucideIcon name={isPlaced ? 'check' : 'plus'} size={12} />
        <span>{isPlaced ? 'Added to Dashboard' : '+ Add to Dashboard'}</span>
      </button>
    </div>
  );
};
