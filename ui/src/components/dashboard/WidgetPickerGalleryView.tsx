import React from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { getWidgetCompositeKey, PlacedDashboardWidget, PluginWidgetDescriptor } from '../../themeAndTypes';
import { WidgetFootprintDiagram } from './WidgetFootprintDiagram';

export interface WidgetPickerGalleryViewProps {
  widgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  onSelectWidget: (widget: PluginWidgetDescriptor, e: React.MouseEvent<HTMLButtonElement>) => void;
}

export const WidgetPickerGalleryView: React.FC<WidgetPickerGalleryViewProps> = ({
  widgets,
  placedWidgets,
  onSelectWidget,
}) => {
  return (
    <div
      data-testid="widget-picker-gallery-view"
      style={{
        padding: '16px 18px',
        overflowY: 'auto',
        flex: 1,
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))',
        alignContent: 'start',
        gap: '12px',
      }}
    >
      {widgets.map((w) => {
        const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
        const alreadyPlaced = placedWidgets.some(
          (pw) => pw.plugin_id === w.plugin_id && pw.id === w.id
        );
        return (
          <div
            key={compositeKey}
            data-testid={`picker-widget-card-${w.id}`}
            data-plug-id={w.plugin_id}
            data-widget-key={compositeKey}
            data-widget-size={`${w.width}x${w.height}`}
            style={{
              backgroundColor: 'var(--mis-bg-elevated)',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              padding: '12px',
              display: 'flex',
              flexDirection: 'column',
              justifyContent: 'space-between',
              gap: '10px',
            }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  justifyContent: 'space-between',
                  gap: '8px',
                }}
              >
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontSize: 'calc(13px * var(--mis-font-scale, 1))',
                    fontWeight: 700,
                    color: 'var(--mis-text-primary)',
                  }}
                >
                  <MonochromeLucideIcon name={w.icon || 'layout-grid'} size={15} />
                  <span>{w.title}</span>
                </span>
                <WidgetFootprintDiagram
                  testId={`picker-widget-footprint-diagram-${w.id}`}
                  width={w.width}
                  height={w.height}
                />
              </div>

              <div
                data-testid={`picker-widget-plugin-${w.id}`}
                style={{
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  fontWeight: 600,
                  color: 'var(--mis-text-secondary)',
                }}
              >
                {w.plugin_title || w.plugin_id}
              </div>

              <div
                data-testid={`picker-widget-description-${w.id}`}
                style={{
                  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                  lineHeight: 1.45,
                  color: 'var(--mis-text-primary)',
                }}
              >
                {w.description}
              </div>
            </div>

            <button
              type="button"
              data-testid={`place-widget-btn-${w.id}`}
              data-plug-id={w.plugin_id}
              data-widget-key={compositeKey}
              disabled={alreadyPlaced}
              onClick={(e) => onSelectWidget(w, e)}
              style={{
                padding: '8px 10px',
                backgroundColor: alreadyPlaced
                  ? 'var(--mis-bg-surface)'
                  : 'var(--mis-brand-primary)',
                color: alreadyPlaced
                  ? 'var(--mis-text-muted)'
                  : 'var(--mis-brand-primary-fg)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                fontWeight: 700,
                cursor: alreadyPlaced ? 'default' : 'pointer',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                gap: '8px',
                textAlign: 'left',
              }}
            >
              <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                <MonochromeLucideIcon name={alreadyPlaced ? 'check' : 'plus'} size={14} />
                <span>{alreadyPlaced ? `Placed: ${w.title}` : `+ ${w.title}`}</span>
              </span>
            </button>
          </div>
        );
      })}
    </div>
  );
};
