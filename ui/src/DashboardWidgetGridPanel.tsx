import React, { useEffect, useMemo, useState } from 'react';
import { MonochromeLucideIcon } from './components/MonochromeLucideIcon';
import {
  findFirstFree2DSlot,
  getWidgetCompositeKey,
  is2DFootprintFree,
  PlacedDashboardWidget,
  PluginWidgetDescriptor,
} from './themeAndTypes';

export interface DashboardWidgetGridPanelProps {
  isEditMode: boolean;
  columnCount?: number;
  availableWidgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  onToggleEditMode: () => void;
  onPlaceWidget: (widget: PluginWidgetDescriptor, targetCol?: number, targetRow?: number) => void;
  onRemoveWidget: (pluginId: string, widgetId: string) => void;
  onMoveWidgetUp?: (pluginId: string, widgetId: string) => void;
  onMoveWidgetDown?: (pluginId: string, widgetId: string) => void;
  onMoveWidgetLeft?: (pluginId: string, widgetId: string) => void;
  onMoveWidgetRight?: (pluginId: string, widgetId: string) => void;
  onAddColumn?: () => void;
  onRemoveColumn?: () => void;
  dashboardSaveStatus?: string;
  onRegisterWidgetIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

const FIXED_COLUMN_WIDTH_PX = 300;
const FIXED_ROW_HEIGHT_PX = 180;
const MIN_COLUMNS = 2;
const MAX_COLUMNS = 8;

export const DashboardWidgetGridPanel: React.FC<DashboardWidgetGridPanelProps> = ({
  isEditMode,
  columnCount = 4,
  availableWidgets,
  placedWidgets,
  onToggleEditMode,
  onPlaceWidget,
  onRemoveWidget,
  onMoveWidgetUp,
  onMoveWidgetDown,
  onMoveWidgetLeft,
  onMoveWidgetRight,
  onAddColumn,
  onRemoveColumn,
  dashboardSaveStatus = '',
  onRegisterWidgetIframeRef,
}) => {
  const [isPickerModalOpen, setIsPickerModalOpen] = useState<boolean>(false);
  const [pendingWidget, setPendingWidget] = useState<PluginWidgetDescriptor | null>(null);
  const [previewPos, setPreviewPos] = useState<{ col: number; row: number } | null>(null);

  const normalizedColCount = Math.min(MAX_COLUMNS, Math.max(MIN_COLUMNS, columnCount));

  useEffect(() => {
    if (!isEditMode) {
      setIsPickerModalOpen(false);
      setPendingWidget(null);
      setPreviewPos(null);
    }
  }, [isEditMode]);

  useEffect(() => {
    if (!pendingWidget && !isPickerModalOpen) {
      return;
    }
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setIsPickerModalOpen(false);
        setPendingWidget(null);
        setPreviewPos(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [pendingWidget, isPickerModalOpen]);

  const minAllowedColumns = useMemo(() => {
    return placedWidgets.reduce((acc, w) => {
      const spanW = w.width === 2 || w.colSpan === 2 ? 2 : 1;
      const wCol = typeof w.col === 'number' && w.col >= 1 ? w.col : 1;
      const rightEdge = wCol + spanW - 1;
      return Math.max(acc, rightEdge);
    }, MIN_COLUMNS);
  }, [placedWidgets]);

  const visibleRowCount = useMemo(() => {
    const maxPlacedBottom = placedWidgets.reduce((maxRow, w) => {
      const spanH = w.height === 2 || w.rowSpan === 2 ? 2 : 1;
      const wRow = typeof w.row === 'number' && w.row >= 1 ? w.row : 1;
      return Math.max(maxRow, wRow + spanH - 1);
    }, 1);

    let maxBottom = maxPlacedBottom;
    if (pendingWidget && previewPos) {
      const pendingH = pendingWidget.height === 2 ? 2 : 1;
      maxBottom = Math.max(maxBottom, previewPos.row + pendingH - 1);
    }
    return isEditMode ? Math.max(2, maxBottom + 1) : Math.max(1, maxPlacedBottom);
  }, [isEditMode, placedWidgets, pendingWidget, previewPos]);

  const emptyCells = useMemo(() => {
    if (!isEditMode) {
      return [];
    }
    const cells: Array<{ col: number; row: number }> = [];
    for (let row = 1; row <= visibleRowCount; row += 1) {
      for (let col = 1; col <= normalizedColCount; col += 1) {
        if (is2DFootprintFree(placedWidgets, col, row, 1, 1, normalizedColCount)) {
          cells.push({ col, row });
        }
      }
    }
    return cells;
  }, [isEditMode, visibleRowCount, normalizedColCount, placedWidgets]);

  const handleSelectWidgetFromModal = (w: PluginWidgetDescriptor) => {
    const spanW = w.width === 2 ? 2 : 1;
    const spanH = w.height === 2 ? 2 : 1;
    const firstSlot = findFirstFree2DSlot(placedWidgets, spanW, spanH, normalizedColCount);
    setIsPickerModalOpen(false);
    setPendingWidget(w);
    setPreviewPos(firstSlot);
  };

  const handleConfirmPlacement = (col: number, row: number) => {
    if (!pendingWidget) {
      return;
    }
    const spanW = pendingWidget.width === 2 ? 2 : 1;
    const spanH = pendingWidget.height === 2 ? 2 : 1;
    if (!is2DFootprintFree(placedWidgets, col, row, spanW, spanH, normalizedColCount)) {
      return;
    }
    onPlaceWidget(pendingWidget, col, row);
    setPendingWidget(null);
    setPreviewPos(null);
  };

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
        overflowX: 'auto',
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
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
          <h2
            style={{
              fontSize: '14px',
              fontWeight: 700,
              margin: 0,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <MonochromeLucideIcon name="layout-grid" size={16} />
            <span>Home Dashboard Widgets</span>
          </h2>
          {dashboardSaveStatus && (
            <span
              data-testid="dashboard-layout-save-status"
              style={{
                padding: '2px 8px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                backgroundColor: 'var(--mis-bg-elevated)',
                border: '1px solid var(--mis-border-subtle)',
                fontSize: '11px',
                fontWeight: 700,
              }}
            >
              {dashboardSaveStatus}
            </span>
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
          {isEditMode && (
            <>
              <button
                type="button"
                data-testid="dashboard-add-widget-btn"
                onClick={() => {
                  setPendingWidget(null);
                  setPreviewPos(null);
                  setIsPickerModalOpen(true);
                }}
                style={{
                  padding: '6px 12px',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  color: 'var(--mis-text-primary)',
                  border: '1px solid var(--mis-border-strong)',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  fontSize: '12px',
                  fontWeight: 700,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '6px',
                }}
              >
                <MonochromeLucideIcon name="plus" size={14} />
                <span>Add Widget</span>
              </button>

              {pendingWidget && (
                <button
                  type="button"
                  data-testid="dashboard-cancel-placement-btn"
                  onClick={() => {
                    setPendingWidget(null);
                    setPreviewPos(null);
                  }}
                  style={{
                    padding: '6px 10px',
                    backgroundColor: 'var(--mis-status-danger-bg)',
                    color: 'var(--mis-status-danger)',
                    border: '1px solid var(--mis-border-strong)',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    fontSize: '11px',
                    fontWeight: 700,
                    cursor: 'pointer',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                  }}
                >
                  <MonochromeLucideIcon name="x" size={12} />
                  <span>Cancel Placement ({pendingWidget.title})</span>
                </button>
              )}

              <div
                data-testid="dashboard-column-controls"
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '4px 8px',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  border: '1px solid var(--mis-border-subtle)',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                }}
              >
                <button
                  type="button"
                  data-testid="dashboard-remove-column-btn"
                  disabled={normalizedColCount <= minAllowedColumns}
                  onClick={() => onRemoveColumn?.()}
                  style={{
                    padding: '3px 8px',
                    backgroundColor: 'var(--mis-bg-surface)',
                    color:
                      normalizedColCount <= minAllowedColumns
                        ? 'var(--mis-text-muted)'
                        : 'var(--mis-text-primary)',
                    border: '1px solid var(--mis-border-subtle)',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    fontSize: '11px',
                    fontWeight: 700,
                    cursor: normalizedColCount <= minAllowedColumns ? 'default' : 'pointer',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                  }}
                >
                  <MonochromeLucideIcon name="minus" size={12} />
                  <span>Remove Column</span>
                </button>
                <span
                  data-testid="dashboard-column-count-badge"
                  data-column-count={String(normalizedColCount)}
                  style={{ fontSize: '12px', fontWeight: 700, padding: '0 4px' }}
                >
                  {normalizedColCount} Columns ({FIXED_COLUMN_WIDTH_PX}px)
                </span>
                <button
                  type="button"
                  data-testid="dashboard-add-column-btn"
                  disabled={normalizedColCount >= MAX_COLUMNS}
                  onClick={() => onAddColumn?.()}
                  style={{
                    padding: '3px 8px',
                    backgroundColor: 'var(--mis-bg-surface)',
                    color:
                      normalizedColCount >= MAX_COLUMNS
                        ? 'var(--mis-text-muted)'
                        : 'var(--mis-text-primary)',
                    border: '1px solid var(--mis-border-subtle)',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    fontSize: '11px',
                    fontWeight: 700,
                    cursor: normalizedColCount >= MAX_COLUMNS ? 'default' : 'pointer',
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '4px',
                  }}
                >
                  <MonochromeLucideIcon name="plus" size={12} />
                  <span>Add Column</span>
                </button>
              </div>
            </>
          )}

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
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <MonochromeLucideIcon name={isEditMode ? 'check' : 'layout-grid'} size={14} />
            <span>{isEditMode ? 'Finish Editing Grid' : 'Edit Dashboard Grid'}</span>
          </button>
        </div>
      </div>

      {isEditMode && isPickerModalOpen && (
        <div
          data-testid="dashboard-widget-picker-modal"
          style={{
            backgroundColor: 'var(--mis-bg-elevated)',
            border: '1px solid var(--mis-border-strong)',
            borderRadius: 'var(--mis-radius-md, 8px)',
            boxShadow: 'var(--mis-shadow-card)',
            padding: '12px',
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
              fontSize: '12px',
              fontWeight: 700,
            }}
          >
            <span>Select a Widget to Place on Grid</span>
            <button
              type="button"
              data-testid="dashboard-widget-picker-close-btn"
              onClick={() => setIsPickerModalOpen(false)}
              style={{
                background: 'none',
                border: 'none',
                color: 'var(--mis-text-secondary)',
                cursor: 'pointer',
                padding: '4px',
                display: 'inline-flex',
                alignItems: 'center',
              }}
            >
              <MonochromeLucideIcon name="x" size={14} />
            </button>
          </div>

          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
              gap: '8px',
            }}
          >
            {availableWidgets.map((w) => {
              const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
              const alreadyPlaced = placedWidgets.some(
                (pw) => pw.plugin_id === w.plugin_id && pw.id === w.id
              );
              return (
                <button
                  key={compositeKey}
                  type="button"
                  data-testid={`place-widget-btn-${w.id}`}
                  data-plug-id={w.plugin_id}
                  data-widget-key={compositeKey}
                  disabled={alreadyPlaced}
                  onClick={() => handleSelectWidgetFromModal(w)}
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
                    fontSize: '12px',
                    fontWeight: 600,
                    cursor: alreadyPlaced ? 'default' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    gap: '8px',
                    textAlign: 'left',
                  }}
                >
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
                    <MonochromeLucideIcon name={w.icon || 'layout-grid'} size={14} />
                    <span>{alreadyPlaced ? `Placed: ${w.title}` : `+ ${w.title}`}</span>
                  </span>
                  <span style={{ fontSize: '11px', opacity: 0.85 }}>
                    {w.width}×{w.height}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}

      <div
        data-testid="dashboard-widget-grid"
        data-column-count={String(normalizedColCount)}
        style={{
          display: 'grid',
          gridTemplateColumns: `repeat(${normalizedColCount}, ${FIXED_COLUMN_WIDTH_PX}px)`,
          gridAutoRows: `${FIXED_ROW_HEIGHT_PX}px`,
          gridAutoFlow: 'row',
          gap: '12px',
        }}
      >
        {placedWidgets.map((w, idx) => {
          const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
          const isPermitted = w.permitted !== false;
          const presetWidth = w.width === 1 || w.width === 2 ? w.width : w.colSpan;
          const presetHeight = w.height === 1 || w.height === 2 ? w.height : w.rowSpan;
          const wCol = typeof w.col === 'number' && w.col >= 1 ? w.col : 1;
          const wRow = typeof w.row === 'number' && w.row >= 1 ? w.row : 1;

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
              key={compositeKey}
              data-testid={`placed-widget-card-${w.id}`}
              data-order-index={String(idx)}
              data-plug-id={w.plugin_id}
              data-widget-id={w.id}
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
                    fontSize: '12px',
                    fontWeight: 700,
                    display: 'inline-flex',
                    alignItems: 'center',
                    gap: '6px',
                  }}
                >
                  <MonochromeLucideIcon name={w.icon || 'layout-grid'} size={14} />
                  <span>
                    {w.title}
                    {isEditMode ? ` (${presetWidth}×${presetHeight})` : ''}
                  </span>
                </span>
              </div>

              {isPermitted ? (
                <iframe
                  ref={(el) => onRegisterWidgetIframeRef(w.id, w.plugin_id, el)}
                  data-testid={`widget-iframe-${w.id}`}
                  data-plug-id={w.plugin_id}
                  data-widget-key={compositeKey}
                  sandbox="allow-scripts"
                  src={w.entry_url}
                  title={`Widget: ${w.title}`}
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
                  data-testid={`widget-permission-error-card-${w.id}`}
                  data-plug-id={w.plugin_id}
                  data-widget-id={w.id}
                  data-error-code={w.error_code || 'WIDGET_PERMISSION_DENIED'}
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
                    fontSize: '12px',
                    opacity: isEditMode ? 0.45 : 1,
                  }}
                >
                  <div style={{ fontWeight: 700 }}>
                    {w.error_title || 'Widget Access Restricted or Revoked'}
                  </div>
                  <div style={{ fontSize: '11px' }}>
                    Widget <code>{w.id}</code> (Plugin <code>{w.plugin_id}</code>) is not permitted for your active role/scopes or has been revoked ({w.error_code || 'WIDGET_PERMISSION_DENIED'}).
                  </div>
                </div>
              )}

              {isEditMode && (
                <div
                  data-testid={`widget-edit-overlay-${w.id}`}
                  data-plug-id={w.plugin_id}
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
                    data-testid={`move-widget-up-btn-${w.id}`}
                    data-plug-id={w.plugin_id}
                    data-widget-key={compositeKey}
                    disabled={!canMoveUp}
                    onClick={() => onMoveWidgetUp?.(w.plugin_id, w.id)}
                    style={{
                      padding: '5px 8px',
                      backgroundColor: 'var(--mis-bg-surface)',
                      color: !canMoveUp ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
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
                    data-testid={`move-widget-down-btn-${w.id}`}
                    data-plug-id={w.plugin_id}
                    data-widget-key={compositeKey}
                    disabled={!canMoveDown}
                    onClick={() => onMoveWidgetDown?.(w.plugin_id, w.id)}
                    style={{
                      padding: '5px 8px',
                      backgroundColor: 'var(--mis-bg-surface)',
                      color: !canMoveDown ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
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
                    data-testid={`move-widget-left-btn-${w.id}`}
                    data-plug-id={w.plugin_id}
                    data-widget-key={compositeKey}
                    disabled={!canMoveLeft}
                    onClick={() => onMoveWidgetLeft?.(w.plugin_id, w.id)}
                    style={{
                      padding: '5px 8px',
                      backgroundColor: 'var(--mis-bg-surface)',
                      color: !canMoveLeft ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
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
                    data-testid={`move-widget-right-btn-${w.id}`}
                    data-plug-id={w.plugin_id}
                    data-widget-key={compositeKey}
                    disabled={!canMoveRight}
                    onClick={() => onMoveWidgetRight?.(w.plugin_id, w.id)}
                    style={{
                      padding: '5px 8px',
                      backgroundColor: 'var(--mis-bg-surface)',
                      color: !canMoveRight ? 'var(--mis-text-muted)' : 'var(--mis-text-primary)',
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
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
                    data-testid={`remove-widget-btn-${w.id}`}
                    data-plug-id={w.plugin_id}
                    data-widget-key={compositeKey}
                    onClick={() => onRemoveWidget(w.plugin_id, w.id)}
                    style={{
                      padding: '5px 8px',
                      backgroundColor: 'var(--mis-status-danger-bg)',
                      color: 'var(--mis-status-danger)',
                      border: '1px solid var(--mis-border-strong)',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      fontSize: '11px',
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
        })}

        {isEditMode &&
          emptyCells.map(({ col, row }) => (
            <div
              key={`empty-cell-${col}-${row}`}
              data-testid={`dashboard-empty-grid-cell-${col}-${row}`}
              data-dotted-cell="true"
              data-grid-col={String(col)}
              data-grid-row={String(row)}
              onMouseEnter={() => {
                if (!pendingWidget) {
                  return;
                }
                const spanW = pendingWidget.width === 2 ? 2 : 1;
                const spanH = pendingWidget.height === 2 ? 2 : 1;
                if (is2DFootprintFree(placedWidgets, col, row, spanW, spanH, normalizedColCount)) {
                  setPreviewPos({ col, row });
                } else {
                  setPreviewPos(null);
                }
              }}
              onClick={() => {
                if (pendingWidget) {
                  handleConfirmPlacement(col, row);
                }
              }}
              style={{
                position: 'relative',
                gridColumn: `${col} / span 1`,
                gridRow: `${row} / span 1`,
                border: '2px dashed var(--mis-border-strong)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                backgroundColor: 'transparent',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                boxSizing: 'border-box',
                cursor: pendingWidget ? 'pointer' : 'default',
              }}
            />
          ))}

        {isEditMode && pendingWidget && previewPos && (
          <div
            data-testid="dashboard-widget-placement-preview"
            data-plug-id={pendingWidget.plugin_id}
            data-widget-id={pendingWidget.id}
            data-preview-col={String(previewPos.col)}
            data-preview-row={String(previewPos.row)}
            data-preview-width={String(pendingWidget.width === 2 ? 2 : 1)}
            data-preview-height={String(pendingWidget.height === 2 ? 2 : 1)}
            onClick={() => handleConfirmPlacement(previewPos.col, previewPos.row)}
            style={{
              gridColumn: `${previewPos.col} / span ${pendingWidget.width === 2 ? 2 : 1}`,
              gridRow: `${previewPos.row} / span ${pendingWidget.height === 2 ? 2 : 1}`,
              border: '2px dashed var(--mis-brand-primary)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              backgroundColor: 'var(--mis-bg-elevated)',
              opacity: 0.9,
              padding: '12px',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '6px',
              cursor: 'pointer',
              zIndex: 4,
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <div
              style={{
                fontSize: '12px',
                fontWeight: 700,
                display: 'inline-flex',
                alignItems: 'center',
                gap: '6px',
              }}
            >
              <MonochromeLucideIcon name={pendingWidget.icon || 'layout-grid'} size={14} />
              <span>
                {pendingWidget.title} ({pendingWidget.width}×{pendingWidget.height})
              </span>
            </div>
            <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)', fontWeight: 600 }}>
              Click to place at Column {previewPos.col}, Row {previewPos.row}
            </div>
          </div>
        )}
      </div>
    </section>
  );
};
