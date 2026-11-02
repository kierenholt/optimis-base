import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  findFirstFree2DSlot,
  is2DFootprintFree,
  PlacedDashboardWidget,
  PluginWidgetDescriptor,
} from './themeAndTypes';
import {
  DashboardEmptyCell,
  DashboardGridHeader,
  DashboardPlacementPreview,
  FIXED_COLUMN_WIDTH_PX,
  FIXED_ROW_HEIGHT_PX,
  MAX_COLUMNS,
  MIN_COLUMNS,
  PlacedWidgetCard,
  WidgetFootprintDiagram,
  WidgetPickerModal,
} from './components/dashboard';

export { WidgetFootprintDiagram };
export { FIXED_COLUMN_WIDTH_PX, FIXED_ROW_HEIGHT_PX, MIN_COLUMNS, MAX_COLUMNS };

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
  const modalClickPosRef = useRef<{ x: number; y: number } | null>(null);

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

  const handleSelectWidgetFromModal = (
    w: PluginWidgetDescriptor,
    e: React.MouseEvent<HTMLButtonElement>
  ) => {
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

  const handleCellPointer = (
    e: React.MouseEvent<HTMLDivElement>,
    col: number,
    row: number
  ) => {
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
      <DashboardGridHeader
        isEditMode={isEditMode}
        dashboardSaveStatus={dashboardSaveStatus}
        pendingWidget={pendingWidget}
        normalizedColCount={normalizedColCount}
        minAllowedColumns={minAllowedColumns}
        onToggleEditMode={onToggleEditMode}
        onOpenAddWidgetModal={() => {
          setPendingWidget(null);
          setPreviewPos(null);
          setIsPickerModalOpen(true);
        }}
        onCancelPlacement={() => {
          setPendingWidget(null);
          setPreviewPos(null);
        }}
        onAddColumn={onAddColumn}
        onRemoveColumn={onRemoveColumn}
      />

      {isEditMode && (
        <WidgetPickerModal
          isOpen={isPickerModalOpen}
          availableWidgets={availableWidgets}
          placedWidgets={placedWidgets}
          onClose={() => setIsPickerModalOpen(false)}
          onSelectWidget={handleSelectWidgetFromModal}
          onRegisterWidgetIframeRef={onRegisterWidgetIframeRef}
        />
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
        {placedWidgets.map((w, idx) => (
          <PlacedWidgetCard
            key={w.plugin_id + ':' + w.id}
            widget={w}
            index={idx}
            isEditMode={isEditMode}
            placedWidgets={placedWidgets}
            normalizedColCount={normalizedColCount}
            onMoveUp={onMoveWidgetUp}
            onMoveDown={onMoveWidgetDown}
            onMoveLeft={onMoveWidgetLeft}
            onMoveRight={onMoveWidgetRight}
            onRemove={onRemoveWidget}
            onRegisterIframeRef={onRegisterWidgetIframeRef}
          />
        ))}

        {isEditMode &&
          emptyCells.map(({ col, row }) => (
            <DashboardEmptyCell
              key={`empty-cell-${col}-${row}`}
              col={col}
              row={row}
              isPlacementPending={pendingWidget !== null}
              onPointer={handleCellPointer}
              onConfirm={handleConfirmPlacement}
            />
          ))}

        {isEditMode && pendingWidget && previewPos && (
          <DashboardPlacementPreview
            pendingWidget={pendingWidget}
            previewPos={previewPos}
            onConfirm={handleConfirmPlacement}
          />
        )}
      </div>
    </section>
  );
};
