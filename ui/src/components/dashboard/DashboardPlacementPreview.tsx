import React from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { PluginWidgetDescriptor } from '../../themeAndTypes';

export interface DashboardPlacementPreviewProps {
  pendingWidget: PluginWidgetDescriptor;
  previewPos: { col: number; row: number };
  onConfirm: (col: number, row: number) => void;
}

export const DashboardPlacementPreview: React.FC<DashboardPlacementPreviewProps> = ({
  pendingWidget,
  previewPos,
  onConfirm,
}) => {
  const spanW = pendingWidget.width === 2 ? 2 : 1;
  const spanH = pendingWidget.height === 2 ? 2 : 1;
  return (
    <div
      data-testid="dashboard-widget-placement-preview"
      data-plug-id={pendingWidget.plugin_id}
      data-widget-id={pendingWidget.id}
      data-preview-col={String(previewPos.col)}
      data-preview-row={String(previewPos.row)}
      data-preview-width={String(spanW)}
      data-preview-height={String(spanH)}
      onClick={() => onConfirm(previewPos.col, previewPos.row)}
      style={{
        gridColumn: `${previewPos.col} / span ${spanW}`,
        gridRow: `${previewPos.row} / span ${spanH}`,
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
  );
};
