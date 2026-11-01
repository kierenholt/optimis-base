import React, { useEffect, useMemo, useRef, useState } from 'react';
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

const WidgetFootprintDiagram: React.FC<{
  testId: string;
  width: number;
  height: number;
  cellWidthPx?: number;
  cellHeightPx?: number;
}> = ({ testId, width, height, cellWidthPx = 18, cellHeightPx = 14 }) => {
  const spanW = width === 2 ? 2 : 1;
  const spanH = height === 2 ? 2 : 1;
  return (
    <div
      data-testid={testId}
      data-footprint-width={String(spanW)}
      data-footprint-height={String(spanH)}
      title={`Footprint: ${spanW} col × ${spanH} row`}
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(2, ${cellWidthPx}px)`,
        gridTemplateRows: `repeat(2, ${cellHeightPx}px)`,
        gap: '3px',
        padding: '5px',
        borderRadius: 'var(--mis-radius-sm, 4px)',
        border: '1px solid var(--mis-border-subtle)',
        backgroundColor: 'var(--mis-bg-surface)',
        flexShrink: 0,
      }}
    >
      {[1, 2].map((r) =>
        [1, 2].map((c) => {
          const activeCell = c <= spanW && r <= spanH;
          return (
            <div
              key={`fp-${c}-${r}`}
              style={{
                borderRadius: '2px',
                backgroundColor: activeCell
                  ? 'var(--mis-brand-primary)'
                  : 'var(--mis-border-subtle)',
                opacity: activeCell ? 1 : 0.45,
              }}
            />
          );
        })
      )}
    </div>
  );
};

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
  const [sizeFilter, setSizeFilter] = useState<'all' | '1x1' | '2x1' | '1x2' | '2x2'>('all');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [pluginFilter, setPluginFilter] = useState<string>('all');
  const [hidePlaced, setHidePlaced] = useState<boolean>(false);
  const [modalViewMode, setModalViewMode] = useState<'gallery' | 'split'>('gallery');
  const [selectedSplitWidgetKey, setSelectedSplitWidgetKey] = useState<string>('');
  const modalClickPosRef = useRef<{ x: number; y: number } | null>(null);

  const normalizedColCount = Math.min(MAX_COLUMNS, Math.max(MIN_COLUMNS, columnCount));

  const pickerPlugins = useMemo(() => {
    const map = new Map<string, string>();
    for (const w of availableWidgets) {
      if (!map.has(w.plugin_id)) {
        map.set(w.plugin_id, w.plugin_title || w.plugin_id);
      }
    }
    return Array.from(map.entries()).map(([id, title]) => ({ id, title }));
  }, [availableWidgets]);

  const filteredPickerWidgets = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return availableWidgets.filter((w) => {
      const sizeKey = `${w.width}x${w.height}`;
      if (sizeFilter !== 'all' && sizeKey !== sizeFilter) {
        return false;
      }
      if (pluginFilter !== 'all' && w.plugin_id !== pluginFilter) {
        return false;
      }
      if (hidePlaced) {
        const isPlaced = placedWidgets.some((pw) => pw.plugin_id === w.plugin_id && pw.id === w.id);
        if (isPlaced) {
          return false;
        }
      }
      if (q !== '') {
        const matchesTitle = (w.title || '').toLowerCase().includes(q);
        const matchesDesc = (w.description || '').toLowerCase().includes(q);
        const matchesPlugin = (w.plugin_title || w.plugin_id || '').toLowerCase().includes(q);
        const matchesId = (w.id || '').toLowerCase().includes(q);
        const matchesKw = Array.isArray(w.keywords)
          ? w.keywords.some((kw) => kw.toLowerCase().includes(q))
          : false;
        if (!matchesTitle && !matchesDesc && !matchesPlugin && !matchesId && !matchesKw) {
          return false;
        }
      }
      return true;
    });
  }, [availableWidgets, sizeFilter, pluginFilter, hidePlaced, searchQuery, placedWidgets]);

  const selectedSplitWidget = useMemo(() => {
    if (filteredPickerWidgets.length === 0) {
      return null;
    }
    if (selectedSplitWidgetKey) {
      const found = filteredPickerWidgets.find(
        (w) => getWidgetCompositeKey(w.plugin_id, w.id) === selectedSplitWidgetKey
      );
      if (found) {
        return found;
      }
    }
    return filteredPickerWidgets[0];
  }, [filteredPickerWidgets, selectedSplitWidgetKey]);

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

  const handleSelectWidgetFromModal = (w: PluginWidgetDescriptor, e: React.MouseEvent<HTMLButtonElement>) => {
    const spanW = w.width === 2 ? 2 : 1;
    const spanH = w.height === 2 ? 2 : 1;
    const firstSlot = findFirstFree2DSlot(placedWidgets, spanW, spanH, normalizedColCount);
    modalClickPosRef.current = { x: e.clientX, y: e.clientY };
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
              fontSize: 'calc(14px * var(--mis-font-scale, 1))',
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
                fontSize: 'calc(11px * var(--mis-font-scale, 1))',
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
                  setSizeFilter('all');
                  setSearchQuery('');
                  setPluginFilter('all');
                  setHidePlaced(false);
                  setIsPickerModalOpen(true);
                }}
                style={{
                  padding: '6px 12px',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  color: 'var(--mis-text-primary)',
                  border: '1px solid var(--mis-border-strong)',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
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
                    fontSize: 'calc(11px * var(--mis-font-scale, 1))',
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
                    fontSize: 'calc(11px * var(--mis-font-scale, 1))',
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
                  style={{ fontSize: 'calc(12px * var(--mis-font-scale, 1))', fontWeight: 700, padding: '0 4px' }}
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
                    fontSize: 'calc(11px * var(--mis-font-scale, 1))',
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
              fontSize: 'calc(12px * var(--mis-font-scale, 1))',
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
          data-testid="dashboard-widget-picker-backdrop"
          onClick={() => setIsPickerModalOpen(false)}
          style={{
            position: 'fixed',
            inset: 0,
            zIndex: 1000,
            backgroundColor: 'rgba(15, 23, 42, 0.55)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '20px',
            boxSizing: 'border-box',
          }}
        >
          <div
            data-testid="dashboard-widget-picker-modal"
            data-modal-view-mode={modalViewMode}
            role="dialog"
            aria-modal="true"
            aria-label="Add Dashboard Widget"
            onClick={(e) => e.stopPropagation()}
            style={{
              width: '94vw',
              maxWidth: '1240px',
              height: '86vh',
              maxHeight: '880px',
              backgroundColor: 'var(--mis-bg-surface)',
              border: '1px solid var(--mis-border-strong)',
              borderRadius: 'var(--mis-radius-lg, 12px)',
              boxShadow: '0 20px 50px rgba(0, 0, 0, 0.35)',
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
            }}
          >
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                gap: '12px',
                padding: '14px 18px',
                borderBottom: '1px solid var(--mis-border-subtle)',
                backgroundColor: 'var(--mis-bg-elevated)',
                flexWrap: 'wrap',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <MonochromeLucideIcon name="layout-grid" size={16} />
                <span style={{ fontSize: 'calc(14px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
                  Select a Widget to Place on Grid
                </span>
                <span
                  data-testid="widget-picker-result-count"
                  style={{
                    fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                    fontWeight: 600,
                    color: 'var(--mis-text-secondary)',
                    padding: '2px 8px',
                    borderRadius: '999px',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-surface)',
                  }}
                >
                  {filteredPickerWidgets.length} of {availableWidgets.length} widgets
                </span>
              </div>

              <div style={{ display: 'inline-flex', alignItems: 'center', gap: '10px' }}>
                <div
                  data-testid="widget-picker-view-mode-toggle"
                  role="group"
                  aria-label="Widget picker view mode"
                  style={{
                    display: 'inline-flex',
                    alignItems: 'center',
                    padding: '2px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-surface)',
                    gap: '2px',
                  }}
                >
                  <button
                    type="button"
                    data-testid="widget-picker-view-mode-gallery"
                    aria-pressed={modalViewMode === 'gallery'}
                    onClick={() => setModalViewMode('gallery')}
                    style={{
                      padding: '5px 10px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: 'none',
                      backgroundColor:
                        modalViewMode === 'gallery' ? 'var(--mis-brand-primary)' : 'transparent',
                      color:
                        modalViewMode === 'gallery'
                          ? 'var(--mis-brand-primary-fg)'
                          : 'var(--mis-text-primary)',
                      fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                      fontWeight: 700,
                      cursor: 'pointer',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '5px',
                    }}
                  >
                    <MonochromeLucideIcon name="layout-grid" size={13} />
                    <span>Gallery View</span>
                  </button>
                  <button
                    type="button"
                    data-testid="widget-picker-view-mode-split"
                    aria-pressed={modalViewMode === 'split'}
                    onClick={() => setModalViewMode('split')}
                    style={{
                      padding: '5px 10px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: 'none',
                      backgroundColor:
                        modalViewMode === 'split' ? 'var(--mis-brand-primary)' : 'transparent',
                      color:
                        modalViewMode === 'split'
                          ? 'var(--mis-brand-primary-fg)'
                          : 'var(--mis-text-primary)',
                      fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                      fontWeight: 700,
                      cursor: 'pointer',
                      display: 'inline-flex',
                      alignItems: 'center',
                      gap: '5px',
                    }}
                  >
                    <MonochromeLucideIcon name="columns" size={13} />
                    <span>Split Preview</span>
                  </button>
                </div>

                <button
                  type="button"
                  data-testid="dashboard-widget-picker-close-btn"
                  aria-label="Close widget picker modal"
                  onClick={() => setIsPickerModalOpen(false)}
                  style={{
                    background: 'none',
                    border: '1px solid var(--mis-border-subtle)',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    color: 'var(--mis-text-secondary)',
                    cursor: 'pointer',
                    padding: '4px 6px',
                    display: 'inline-flex',
                    alignItems: 'center',
                  }}
                >
                  <MonochromeLucideIcon name="x" size={14} />
                </button>
              </div>
            </div>

            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '10px',
                padding: '12px 18px',
                borderBottom: '1px solid var(--mis-border-subtle)',
                backgroundColor: 'var(--mis-bg-surface)',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  gap: '10px',
                  flexWrap: 'wrap',
                }}
              >
                <div
                  role="group"
                  aria-label="Filter widgets by footprint size"
                  style={{ display: 'inline-flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}
                >
                  <button
                    type="button"
                    data-testid="widget-picker-size-filter-all"
                    aria-pressed={sizeFilter === 'all'}
                    onClick={() => setSizeFilter('all')}
                    style={{
                      padding: '6px 12px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border:
                        sizeFilter === 'all'
                          ? '2px solid var(--mis-brand-primary)'
                          : '1px solid var(--mis-border-subtle)',
                      backgroundColor:
                        sizeFilter === 'all'
                          ? 'var(--mis-brand-primary)'
                          : 'var(--mis-bg-elevated)',
                      color:
                        sizeFilter === 'all'
                          ? 'var(--mis-brand-primary-fg)'
                          : 'var(--mis-text-primary)',
                      fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                      fontWeight: 700,
                      cursor: 'pointer',
                    }}
                  >
                    All Sizes
                  </button>
                  {(
                    [
                      { id: '1x1', w: 1, h: 1 },
                      { id: '2x1', w: 2, h: 1 },
                      { id: '1x2', w: 1, h: 2 },
                      { id: '2x2', w: 2, h: 2 },
                    ] as const
                  ).map((opt) => {
                    const active = sizeFilter === opt.id;
                    return (
                      <button
                        key={opt.id}
                        type="button"
                        data-testid={`widget-picker-size-filter-${opt.id}`}
                        aria-label={`Filter ${opt.w} by ${opt.h} footprint`}
                        aria-pressed={active}
                        onClick={() => setSizeFilter(opt.id)}
                        style={{
                          padding: '3px',
                          borderRadius: 'var(--mis-radius-sm, 4px)',
                          border: active
                            ? '2px solid var(--mis-brand-primary)'
                            : '1px solid var(--mis-border-subtle)',
                          backgroundColor: active
                            ? 'var(--mis-bg-elevated)'
                            : 'transparent',
                          cursor: 'pointer',
                          display: 'inline-flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <WidgetFootprintDiagram
                          testId={`size-filter-footprint-diagram-${opt.id}`}
                          width={opt.w}
                          height={opt.h}
                          cellWidthPx={14}
                          cellHeightPx={10}
                        />
                      </button>
                    );
                  })}
                </div>

                <div style={{ display: 'inline-flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                  <select
                    data-testid="widget-picker-plugin-filter"
                    aria-label="Filter widgets by plugin"
                    value={pluginFilter}
                    onChange={(e) => setPluginFilter(e.target.value)}
                    style={{
                      padding: '5px 8px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: '1px solid var(--mis-border-subtle)',
                      backgroundColor: 'var(--mis-bg-elevated)',
                      color: 'var(--mis-text-primary)',
                      fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                      fontWeight: 600,
                    }}
                  >
                    <option value="all">All Plugins</option>
                    {pickerPlugins.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.title}
                      </option>
                    ))}
                  </select>

                  <button
                    type="button"
                    data-testid="widget-picker-hide-placed-toggle"
                    aria-pressed={hidePlaced}
                    onClick={() => setHidePlaced((prev) => !prev)}
                    style={{
                      padding: '5px 10px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: hidePlaced
                        ? '1px solid var(--mis-brand-primary)'
                        : '1px solid var(--mis-border-subtle)',
                      backgroundColor: hidePlaced
                        ? 'var(--mis-brand-primary)'
                        : 'var(--mis-bg-elevated)',
                      color: hidePlaced
                        ? 'var(--mis-brand-primary-fg)'
                        : 'var(--mis-text-primary)',
                      fontSize: 'calc(11px * var(--mis-font-scale, 1))',
                      fontWeight: 700,
                      cursor: 'pointer',
                    }}
                  >
                    Hide already placed
                  </button>
                </div>
              </div>

              <div style={{ position: 'relative', display: 'flex', alignItems: 'center' }}>
                <input
                  type="search"
                  data-testid="widget-picker-search-input"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search widgets by title, description, plugin, or keyword..."
                  style={{
                    width: '100%',
                    padding: '7px 10px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                    boxSizing: 'border-box',
                  }}
                />
              </div>
            </div>

            {modalViewMode === 'gallery' ? (
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
                {filteredPickerWidgets.map((w) => {
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
                        onClick={(e) => handleSelectWidgetFromModal(w, e)}
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
            ) : (
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
                  {filteredPickerWidgets.map((w) => {
                    const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
                    const isSelected =
                      selectedSplitWidget !== null &&
                      getWidgetCompositeKey(
                        selectedSplitWidget.plugin_id,
                        selectedSplitWidget.id
                      ) === compositeKey;
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
                        onClick={() => setSelectedSplitWidgetKey(compositeKey)}
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
                  {selectedSplitWidget ? (
                    (() => {
                      const compositeKey = getWidgetCompositeKey(
                        selectedSplitWidget.plugin_id,
                        selectedSplitWidget.id
                      );
                      const alreadyPlaced = placedWidgets.some(
                        (pw) =>
                          pw.plugin_id === selectedSplitWidget.plugin_id &&
                          pw.id === selectedSplitWidget.id
                      );
                      const spanW = selectedSplitWidget.width === 2 ? 2 : 1;
                      const spanH = selectedSplitWidget.height === 2 ? 2 : 1;
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
                                  name={selectedSplitWidget.icon || 'layout-grid'}
                                  size={16}
                                />
                                <span
                                  data-testid="split-preview-widget-title"
                                  style={{ fontSize: 'calc(15px * var(--mis-font-scale, 1))', fontWeight: 700 }}
                                >
                                  {selectedSplitWidget.title}
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
                                {selectedSplitWidget.plugin_title || selectedSplitWidget.plugin_id}
                              </div>
                              <div
                                data-testid="split-preview-widget-description"
                                style={{
                                  fontSize: 'calc(13px * var(--mis-font-scale, 1))',
                                  lineHeight: 1.5,
                                  color: 'var(--mis-text-primary)',
                                }}
                              >
                                {selectedSplitWidget.description}
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
                                data-testid={`place-widget-btn-${selectedSplitWidget.id}`}
                                data-plug-id={selectedSplitWidget.plugin_id}
                                data-widget-key={compositeKey}
                                disabled={alreadyPlaced}
                                onClick={(e) => handleSelectWidgetFromModal(selectedSplitWidget, e)}
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
                                    ? `Placed: ${selectedSplitWidget.title}`
                                    : `+ Place ${selectedSplitWidget.title}`}
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
                                  `modal_preview_${selectedSplitWidget.id}`,
                                  selectedSplitWidget.plugin_id,
                                  el
                                )
                              }
                              data-testid="widget-picker-live-preview-iframe"
                              data-plug-id={selectedSplitWidget.plugin_id}
                              data-widget-id={selectedSplitWidget.id}
                              data-widget-key={compositeKey}
                              data-preview-width-px={String(previewWidthPx)}
                              data-preview-height-px={String(previewHeightPx)}
                              sandbox="allow-scripts"
                              src={selectedSplitWidget.entry_url}
                              title={`Preview: ${selectedSplitWidget.title}`}
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
            )}
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
                    fontSize: 'calc(12px * var(--mis-font-scale, 1))',
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
                    fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                    opacity: isEditMode ? 0.45 : 1,
                  }}
                >
                  <div style={{ fontWeight: 700 }}>
                    {w.error_title || 'Widget Access Restricted or Revoked'}
                  </div>
                  <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))' }}>
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
        })}

        {isEditMode &&
          emptyCells.map(({ col, row }) => {
            const handleCellPointer = (e: React.MouseEvent<HTMLDivElement>) => {
              if (!pendingWidget) {
                return;
              }
              if (
                modalClickPosRef.current &&
                Math.abs(e.clientX - modalClickPosRef.current.x) < 2 &&
                Math.abs(e.clientY - modalClickPosRef.current.y) < 2
              ) {
                return;
              }
              modalClickPosRef.current = null;
              const spanW = pendingWidget.width === 2 ? 2 : 1;
              const spanH = pendingWidget.height === 2 ? 2 : 1;
              if (is2DFootprintFree(placedWidgets, col, row, spanW, spanH, normalizedColCount)) {
                setPreviewPos({ col, row });
              } else {
                setPreviewPos(null);
              }
            };
            return (
              <div
                key={`empty-cell-${col}-${row}`}
                data-testid={`dashboard-empty-grid-cell-${col}-${row}`}
                data-dotted-cell="true"
                data-grid-col={String(col)}
                data-grid-row={String(row)}
                onMouseEnter={handleCellPointer}
                onMouseMove={handleCellPointer}
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
            );
          })}

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
                fontSize: 'calc(12px * var(--mis-font-scale, 1))',
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
            <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)', fontWeight: 600 }}>
              Click to place at Column {previewPos.col}, Row {previewPos.row}
            </div>
          </div>
        )}
      </div>
    </section>
  );
};
