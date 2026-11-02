import React from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { getWidgetCompositeKey, PlacedDashboardWidget, PluginWidgetDescriptor } from '../../themeAndTypes';
import { WidgetFootprintDiagram } from './WidgetFootprintDiagram';
import { FIXED_COLUMN_WIDTH_PX, FIXED_ROW_HEIGHT_PX } from './dashboardConstants';

export interface WidgetPickerSplitViewProps {
  widgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  selectedWidget: PluginWidgetDescriptor | null;
  onSelectWidgetKey: (key: string) => void;
  onSelectWidget: (widget: PluginWidgetDescriptor, e: React.MouseEvent<HTMLButtonElement>) => void;
  onRegisterWidgetIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

export const WidgetPickerSplitView: React.FC<WidgetPickerSplitViewProps> = ({
  widgets,
  placedWidgets,
  selectedWidget,
  onSelectWidgetKey,
  onSelectWidget,
  onRegisterWidgetIframeRef,
}) => {
  return (
    <div
      data-testid="widget-picker-split-view"
      style={{
        display: 'grid',
        gridTemplateColumns: '370px 1fr',
        flex: 1,
        minHeight: 0,
        overflow: 'hidden',
      }}
    >
      <div
        data-testid="widget-picker-split-list"
        role="listbox"
        aria-label="Available dashboard widgets"
        style={{
          overflowY: 'auto',
          borderRight: '1px solid var(--mis-border-subtle)',
          padding: '12px',
          display: 'flex',
          flexDirection: 'column',
          gap: '8px',
          backgroundColor: 'var(--mis-bg-surface)',
        }}
      >
        {widgets.map((w) => {
          const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
          const isSelected =
            selectedWidget !== null &&
            getWidgetCompositeKey(selectedWidget.plugin_id, selectedWidget.id) === compositeKey;
          const alreadyPlaced = placedWidgets.some(
            (pw) => pw.plugin_id === w.plugin_id && pw.id === w.id
          );
          return (
            <button
              key={compositeKey}
              type="button"
              role="option"
              aria-selected={isSelected}
              data-testid={`split-widget-list-item-${w.id}`}
              data-plug-id={w.plugin_id}
              data-widget-key={compositeKey}
              data-widget-size={`${w.width}x${w.height}`}
              onClick={() => onSelectWidgetKey(compositeKey)}
              style={{
                padding: '10px 12px',
                borderRadius: 'var(--mis-radius-md, 8px)',
                border: isSelected
                  ? '2px solid var(--mis-brand-primary)'
                  : '1px solid var(--mis-border-subtle)',
                backgroundColor: isSelected
                  ? 'var(--mis-bg-elevated)'
                  : 'var(--mis-bg-surface)',
                color: 'var(--mis-text-primary)',
                cursor: 'pointer',
                textAlign: 'left',
                display: 'flex',
                flexDirection: 'column',
                gap: '5px',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: '8px',
                }}
              >
                <span
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '6px',
                    fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                    fontWeight: 700,
                  }}
                >
                  <MonochromeLucideIcon name={w.icon || 'layout-grid'} size={14} />
                  <span>{w.title}</span>
                </span>
                <WidgetFootprintDiagram
                  testId={`split-list-footprint-diagram-${w.id}`}
                  width={w.width}
                  height={w.height}
                  cellWidthPx={12}
                  cellHeightPx={9}
                />
              </div>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                  color: 'var(--mis-text-secondary)',
                }}
              >
                <span>{w.plugin_title || w.plugin_id}</span>
                {alreadyPlaced && (
                  <span style={{ fontWeight: 700, color: 'var(--mis-text-muted)' }}>
                    Placed
                  </span>
                )}
              </div>
            </button>
          );
        })}
      </div>

      <div
        data-testid="widget-picker-split-preview"
        style={{
          overflowY: 'auto',
          padding: '18px 22px',
          display: 'flex',
          flexDirection: 'column',
          gap: '16px',
          backgroundColor: 'var(--mis-bg-surface)',
        }}
      >
        {selectedWidget ? (
          (() => {
            const compositeKey = getWidgetCompositeKey(
              selectedWidget.plugin_id,
              selectedWidget.id
            );
            const alreadyPlaced = placedWidgets.some(
              (pw) =>
                pw.plugin_id === selectedWidget.plugin_id &&
                pw.id === selectedWidget.id
            );
            const spanW = selectedWidget.width === 2 ? 2 : 1;
            const spanH = selectedWidget.height === 2 ? 2 : 1;
            const previewWidthPx =
              spanW === 2 ? FIXED_COLUMN_WIDTH_PX * 2 + 12 : FIXED_COLUMN_WIDTH_PX;
            const previewHeightPx =
              spanH === 2 ? FIXED_ROW_HEIGHT_PX * 2 + 12 : FIXED_ROW_HEIGHT_PX;

            return (
              <>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                    gap: '16px',
                    padding: '14px 16px',
                    borderRadius: 'var(--mis-radius-md, 8px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    flexWrap: 'wrap',
                  }}
                >
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '6px', flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                      <MonochromeLucideIcon
                        name={selectedWidget.icon || 'layout-grid'}
                        size={16}
                      />
                      <span
                        data-testid="split-preview-widget-title"
                        style={{ fontSize: 'calc(15px * var(--mis-font-scale, 1))', fontWeight: 700 }}
                      >
                        {selectedWidget.title}
                      </span>
                    </div>
                    <div
                      data-testid="split-preview-widget-plugin"
                      style={{
                        fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                        fontWeight: 600,
                        color: 'var(--mis-text-secondary)',
                      }}
                    >
                      {selectedWidget.plugin_title || selectedWidget.plugin_id}
                    </div>
                    <div
                      data-testid="split-preview-widget-description"
                      style={{
                        fontSize: 'calc(13px * var(--mis-font-scale, 1))',
                        lineHeight: 1.5,
                        color: 'var(--mis-text-primary)',
                      }}
                    >
                      {selectedWidget.description}
                    </div>
                  </div>

                  <div
                    style={{
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'flex-end',
                      gap: '10px',
                    }}
                  >
                    <WidgetFootprintDiagram
                      testId="split-preview-footprint-diagram"
                      width={spanW}
                      height={spanH}
                    />

                    <button
                      type="button"
                      data-testid={`place-widget-btn-${selectedWidget.id}`}
                      data-plug-id={selectedWidget.plugin_id}
                      data-widget-key={compositeKey}
                      disabled={alreadyPlaced}
                      onClick={(e) => onSelectWidget(selectedWidget, e)}
                      style={{
                        padding: '8px 14px',
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
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '6px',
                      }}
                    >
                      <MonochromeLucideIcon
                        name={alreadyPlaced ? 'check' : 'plus'}
                        size={14}
                      />
                      <span>
                        {alreadyPlaced
                          ? `Placed: ${selectedWidget.title}`
                          : `+ Place ${selectedWidget.title}`}
                      </span>
                    </button>
                  </div>
                </div>

                <div
                  style={{
                    display: 'flex',
                    flexDirection: 'column',
                    gap: '8px',
                  }}
                >
                  <div
                    style={{
                      fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                      fontWeight: 700,
                      textTransform: 'uppercase',
                      letterSpacing: '0.04em',
                      color: 'var(--mis-text-secondary)',
                    }}
                  >
                    Live Sandbox Preview ({previewWidthPx}px × {previewHeightPx}px — {spanW}×{spanH} Preset)
                  </div>
                  <iframe
                    key={compositeKey}
                    ref={(el) =>
                      onRegisterWidgetIframeRef(
                        `modal_preview_${selectedWidget.id}`,
                        selectedWidget.plugin_id,
                        el
                      )
                    }
                    data-testid="widget-picker-live-preview-iframe"
                    data-plug-id={selectedWidget.plugin_id}
                    data-widget-id={selectedWidget.id}
                    data-widget-key={compositeKey}
                    data-preview-width-px={String(previewWidthPx)}
                    data-preview-height-px={String(previewHeightPx)}
                    sandbox="allow-scripts"
                    src={selectedWidget.entry_url}
                    title={`Preview: ${selectedWidget.title}`}
                    style={{
                      width: `${previewWidthPx}px`,
                      height: `${previewHeightPx}px`,
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-md, 8px)',
                      backgroundColor: 'var(--mis-bg-elevated)',
                      boxShadow: 'var(--mis-shadow-card)',
                    }}
                  />
                </div>
              </>
            );
          })()
        ) : (
          <div
            style={{
              padding: '24px',
              color: 'var(--mis-text-secondary)',
              fontSize: 'calc(13px * var(--mis-font-scale, 1))',
            }}
          >
            No widgets match the current filter criteria.
          </div>
        )}
      </div>
    </div>
  );
};
