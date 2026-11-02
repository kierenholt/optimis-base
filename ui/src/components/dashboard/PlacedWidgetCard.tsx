import React from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { getWidgetCompositeKey, is2DFootprintFree, PlacedDashboardWidget } from '../../themeAndTypes';

export interface PlacedWidgetCardProps {
  widget: PlacedDashboardWidget;
  index: number;
  isEditMode: boolean;
  placedWidgets: PlacedDashboardWidget[];
  normalizedColCount: number;
  onMoveUp?: (pluginId: string, widgetId: string) => void;
  onMoveDown?: (pluginId: string, widgetId: string) => void;
  onMoveLeft?: (pluginId: string, widgetId: string) => void;
  onMoveRight?: (pluginId: string, widgetId: string) => void;
  onRemove: (pluginId: string, widgetId: string) => void;
  onRegisterIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

export const PlacedWidgetCard: React.FC<PlacedWidgetCardProps> = ({
  widget,
  index,
  isEditMode,
  placedWidgets,
  normalizedColCount,
  onMoveUp,
  onMoveDown,
  onMoveLeft,
  onMoveRight,
  onRemove,
  onRegisterIframeRef,
}) => {
  const compositeKey = getWidgetCompositeKey(widget.plugin_id, widget.id);
  const isPermitted = widget.permitted !== false;
  const presetWidth = widget.width === 1 || widget.width === 2 ? widget.width : widget.colSpan;
  const presetHeight = widget.height === 1 || widget.height === 2 ? widget.height : widget.rowSpan;
  const wCol = typeof widget.col === 'number' && widget.col >= 1 ? widget.col : 1;
  const wRow = typeof widget.row === 'number' && widget.row >= 1 ? widget.row : 1;

  const canMoveUp = isEditMode
    ? is2DFootprintFree(
        placedWidgets,
        wCol,
        wRow - 1,
        presetWidth,
        presetHeight,
        normalizedColCount,
        compositeKey
      )
    : false;
  const canMoveDown = isEditMode
    ? is2DFootprintFree(
        placedWidgets,
        wCol,
        wRow + 1,
        presetWidth,
        presetHeight,
        normalizedColCount,
        compositeKey
      )
    : false;
  const canMoveLeft = isEditMode
    ? is2DFootprintFree(
        placedWidgets,
        wCol - 1,
        wRow,
        presetWidth,
        presetHeight,
        normalizedColCount,
        compositeKey
      )
    : false;
  const canMoveRight = isEditMode
    ? is2DFootprintFree(
        placedWidgets,
        wCol + 1,
        wRow,
        presetWidth,
        presetHeight,
        normalizedColCount,
        compositeKey
      )
    : false;

  return (
    <div
      data-testid={`placed-widget-card-${widget.id}`}
      data-order-index={String(index)}
      data-plug-id={widget.plugin_id}
      data-widget-id={widget.id}
      data-widget-key={compositeKey}
      data-grid-col={String(wCol)}
      data-grid-row={String(wRow)}
      data-width={String(presetWidth)}
      data-height={String(presetHeight)}
      data-col-span={String(presetWidth)}
      data-row-span={String(presetHeight)}
      data-edit-preview={isEditMode ? 'true' : undefined}
      style={{
        position: 'relative',
        gridColumn: `${wCol} / span ${presetWidth}`,
        gridRow: `${wRow} / span ${presetHeight}`,
        backgroundColor: 'var(--mis-bg-elevated)',
        border: isEditMode
          ? '2px dashed var(--mis-border-strong)'
          : '1px solid var(--mis-border-subtle)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        padding: '10px',
        display: 'flex',
        flexDirection: 'column',
        gap: '8px',
        boxSizing: 'border-box',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '8px',
          flexWrap: 'wrap',
          opacity: isEditMode ? 0.55 : 1,
        }}
      >
        <span
          style={{
            fontSize: 'calc(12px * var(--mis-font-scale, 1))',
            fontWeight: 700,
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
          }}
        >
          <MonochromeLucideIcon name={widget.icon || 'layout-grid'} size={14} />
          <span>
            {widget.title}
            {isEditMode ? ` (${presetWidth}×${presetHeight})` : ''}
          </span>
        </span>
      </div>

      {isPermitted ? (
        <iframe
          ref={(el) => onRegisterIframeRef(widget.id, widget.plugin_id, el)}
          data-testid={`widget-iframe-${widget.id}`}
          data-plug-id={widget.plugin_id}
          data-widget-key={compositeKey}
          sandbox="allow-scripts"
          src={widget.entry_url}
          title={`Widget: ${widget.title}`}
          style={{
            width: '100%',
            flex: 1,
            minHeight: '0',
            border: '1px solid var(--mis-border-subtle)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'var(--mis-bg-surface)',
            pointerEvents: isEditMode ? 'none' : 'auto',
            opacity: isEditMode ? 0.35 : 1,
            filter: isEditMode ? 'grayscale(90%)' : 'none',
          }}
        />
      ) : (
        <div
          data-testid={`widget-permission-error-card-${widget.id}`}
          data-plug-id={widget.plugin_id}
          data-widget-id={widget.id}
          data-error-code={widget.error_code || 'WIDGET_PERMISSION_DENIED'}
          style={{
            width: '100%',
            flex: 1,
            minHeight: '0',
            border: '1px solid var(--mis-status-danger, #b31412)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'var(--mis-status-danger-bg, #fce8e6)',
            color: 'var(--mis-status-danger, #a50e0e)',
            padding: '12px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            gap: '4px',
            fontSize: 'calc(12px * var(--mis-font-scale, 1))',
            opacity: isEditMode ? 0.45 : 1,
          }}
        >
          <div style={{ fontWeight: 700 }}>
            {widget.error_title || 'Widget Access Restricted or Revoked'}
          </div>
          <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))' }}>
            Widget <code>{widget.id}</code> (Plugin <code>{widget.plugin_id}</code>) is not permitted for your active role/scopes or has been revoked ({widget.error_code || 'WIDGET_PERMISSION_DENIED'}).
          </div>
        </div>
      )}

      {isEditMode && (
        <div
          data-testid={`widget-edit-overlay-${widget.id}`}
          data-plug-id={widget.plugin_id}
          data-widget-key={compositeKey}
          style={{
            position: 'absolute',
            inset: 0,
            borderRadius: 'var(--mis-radius-md, 8px)',
            backgroundColor: 'rgba(15, 23, 42, 0.14)',
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            justifyContent: 'center',
            gap: '6px',
            padding: '10px',
            zIndex: 5,
          }}
        >
          <button
            type="button"
            data-testid={`move-widget-up-btn-${widget.id}`}
            data-plug-id={widget.plugin_id}
            data-widget-key={compositeKey}
            disabled={!canMoveUp}
            onClick={() => onMoveUp?.(widget.plugin_id, widget.id)}
            style={{
              padding: '5px 8px',
              backgroundColor: 'var(--mis-bg-surface)',
              color: !canMoveUp ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
              fontWeight: 700,
              cursor: !canMoveUp ? 'default' : 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <MonochromeLucideIcon name="arrow-up" size={12} />
            <span>Up</span>
          </button>

          <button
            type="button"
            data-testid={`move-widget-down-btn-${widget.id}`}
            data-plug-id={widget.plugin_id}
            data-widget-key={compositeKey}
            disabled={!canMoveDown}
            onClick={() => onMoveDown?.(widget.plugin_id, widget.id)}
            style={{
              padding: '5px 8px',
              backgroundColor: 'var(--mis-bg-surface)',
              color: !canMoveDown ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
              fontWeight: 700,
              cursor: !canMoveDown ? 'default' : 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <MonochromeLucideIcon name="arrow-down" size={12} />
            <span>Down</span>
          </button>

          <button
            type="button"
            data-testid={`move-widget-left-btn-${widget.id}`}
            data-plug-id={widget.plugin_id}
            data-widget-key={compositeKey}
            disabled={!canMoveLeft}
            onClick={() => onMoveLeft?.(widget.plugin_id, widget.id)}
            style={{
              padding: '5px 8px',
              backgroundColor: 'var(--mis-bg-surface)',
              color: !canMoveLeft ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
              fontWeight: 700,
              cursor: !canMoveLeft ? 'default' : 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <MonochromeLucideIcon name="arrow-left" size={12} />
            <span>Left</span>
          </button>

          <button
            type="button"
            data-testid={`move-widget-right-btn-${widget.id}`}
            data-plug-id={widget.plugin_id}
            data-widget-key={compositeKey}
            disabled={!canMoveRight}
            onClick={() => onMoveRight?.(widget.plugin_id, widget.id)}
            style={{
              padding: '5px 8px',
              backgroundColor: 'var(--mis-bg-surface)',
              color: !canMoveRight ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
              fontWeight: 700,
              cursor: !canMoveRight ? 'default' : 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <span>Right</span>
            <MonochromeLucideIcon name="arrow-right" size={12} />
          </button>

          <button
            type="button"
            data-testid={`remove-widget-btn-${widget.id}`}
            data-plug-id={widget.plugin_id}
            data-widget-key={compositeKey}
            onClick={() => onRemove(widget.plugin_id, widget.id)}
            style={{
              padding: '5px 8px',
              backgroundColor: 'var(--mis-status-danger-bg)',
              color: 'var(--mis-status-danger)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: 'calc(11px * var(--mis-font-scale, 1))',
              fontWeight: 700,
              cursor: 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              gap: '3px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <MonochromeLucideIcon name="trash-2" size={12} />
            <span>Delete</span>
          </button>
        </div>
      )}
    </div>
  );
};
