import React from 'react';
import { PlacedDashboardWidget, PluginWidgetDescriptor } from './themeAndTypes';

export interface DashboardWidgetGridPanelProps {
  isEditMode: boolean;
  availableWidgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  onToggleEditMode: () => void;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
  onResizeWidget: (widgetId: string, colSpan: number, rowSpan: number) => void;
  onRemoveWidget: (widgetId: string) => void;
  onRegisterWidgetIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

export const DashboardWidgetGridPanel: React.FC<DashboardWidgetGridPanelProps> = ({
  isEditMode,
  availableWidgets,
  placedWidgets,
  onToggleEditMode,
  onPlaceWidget,
  onResizeWidget,
  onRemoveWidget,
  onRegisterWidgetIframeRef,
}) => {
  return (
    <section
      data-testid="dashboard-widget-grid-section"
      style={{
        backgroundColor: 'var(--mis-bg-surface)',
        border: '1px solid var(--mis-border-subtle)',
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
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <h2 style={{ fontSize: '14px', fontWeight: 700, margin: 0 }}>
            Home Dashboard Mini-Iframe Widgets (N×M Grid)
          </h2>
          <span
            data-testid="dashboard-edit-mode-status"
            style={{
              padding: '2px 8px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              backgroundColor: 'var(--mis-bg-elevated)',
              border: '1px solid var(--mis-border-subtle)',
              fontSize: '11px',
              fontWeight: 700,
            }}
          >
            {isEditMode ? 'EDIT_MODE' : 'VIEW_MODE'}
          </span>
        </div>

        <button
          type="button"
          data-testid="dashboard-edit-mode-toggle-btn"
          onClick={onToggleEditMode}
          style={{
            padding: '6px 12px',
            backgroundColor: 'var(--mis-brand-primary)',
            color: 'var(--mis-brand-primary-fg)',
            border: '1px solid var(--mis-border-strong)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            fontSize: '12px',
            fontWeight: 700,
            cursor: 'pointer',
          }}
        >
          {isEditMode ? 'Finish Editing Grid' : 'Edit Dashboard Grid'}
        </button>
      </div>

      {isEditMode && (
        <div
          data-testid="dashboard-widget-catalog"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
            flexWrap: 'wrap',
            padding: '10px 12px',
            backgroundColor: 'var(--mis-bg-elevated)',
            border: '1px solid var(--mis-border-subtle)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
          }}
        >
          <span style={{ fontSize: '12px', fontWeight: 600 }}>Available Manifest Widgets:</span>
          {availableWidgets.map((w) => {
            const alreadyPlaced = placedWidgets.some((pw) => pw.id === w.id);
            return (
              <button
                key={`${w.plugin_id}:${w.id}`}
                type="button"
                data-testid={`place-widget-btn-${w.id}`}
                onClick={() => onPlaceWidget(w)}
                disabled={alreadyPlaced}
                style={{
                  padding: '5px 10px',
                  backgroundColor: alreadyPlaced ? 'var(--mis-bg-surface)' : 'var(--mis-brand-primary)',
                  color: alreadyPlaced ? 'var(--mis-text-muted)' : 'var(--mis-brand-primary-fg)',
                  border: '1px solid var(--mis-border-subtle)',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: alreadyPlaced ? 'default' : 'pointer',
                }}
              >
                {alreadyPlaced ? `Placed: ${w.title}` : `+ Place ${w.title}`}
              </button>
            );
          })}
        </div>
      )}

      <div
        data-testid="dashboard-widget-grid"
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, minmax(0, 1fr))',
          gridAutoRows: 'minmax(150px, auto)',
          gap: '12px',
        }}
      >
        {placedWidgets.map((w) => (
          <div
            key={w.id}
            data-testid={`placed-widget-card-${w.id}`}
            data-col-span={String(w.colSpan)}
            data-row-span={String(w.rowSpan)}
            style={{
              gridColumn: `span ${w.colSpan}`,
              gridRow: `span ${w.rowSpan}`,
              backgroundColor: 'var(--mis-bg-elevated)',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              padding: '10px',
              display: 'flex',
              flexDirection: 'column',
              gap: '8px',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: '8px',
                flexWrap: 'wrap',
              }}
            >
              <span style={{ fontSize: '12px', fontWeight: 700 }}>
                {w.title} ({w.colSpan}×{w.rowSpan})
              </span>

              {isEditMode && (
                <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '11px' }}>
                  <label>
                    Cols:{' '}
                    <select
                      data-testid={`resize-widget-col-select-${w.id}`}
                      value={String(w.colSpan)}
                      onChange={(e) => onResizeWidget(w.id, Number(e.target.value), w.rowSpan)}
                    >
                      <option value="1">1</option>
                      <option value="2">2</option>
                      <option value="3">3</option>
                      <option value="4">4</option>
                    </select>
                  </label>
                  <label>
                    Rows:{' '}
                    <select
                      data-testid={`resize-widget-row-select-${w.id}`}
                      value={String(w.rowSpan)}
                      onChange={(e) => onResizeWidget(w.id, w.colSpan, Number(e.target.value))}
                    >
                      <option value="1">1</option>
                      <option value="2">2</option>
                      <option value="3">3</option>
                      <option value="4">4</option>
                    </select>
                  </label>
                  <button
                    type="button"
                    data-testid={`remove-widget-btn-${w.id}`}
                    onClick={() => onRemoveWidget(w.id)}
                    style={{
                      padding: '2px 6px',
                      backgroundColor: 'var(--mis-status-danger-bg)',
                      color: 'var(--mis-status-danger)',
                      border: '1px solid var(--mis-border-subtle)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
                      fontWeight: 600,
                      cursor: 'pointer',
                    }}
                  >
                    Remove
                  </button>
                </div>
              )}
            </div>

            <iframe
              ref={(el) => onRegisterWidgetIframeRef(w.id, w.plugin_id, el)}
              data-testid={`widget-iframe-${w.id}`}
              sandbox="allow-scripts"
              src={w.entry_url}
              title={`Widget: ${w.title}`}
              style={{
                width: '100%',
                flex: 1,
                minHeight: `${w.rowSpan * 110}px`,
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                backgroundColor: 'var(--mis-bg-surface)',
              }}
            />
          </div>
        ))}
      </div>
    </section>
  );
};
