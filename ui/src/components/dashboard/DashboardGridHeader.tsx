import React from 'react';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { PluginWidgetDescriptor } from '../../themeAndTypes';
import { FIXED_COLUMN_WIDTH_PX, MAX_COLUMNS } from './dashboardConstants';

export interface DashboardGridHeaderProps {
  isEditMode: boolean;
  dashboardSaveStatus?: string;
  pendingWidget: PluginWidgetDescriptor | null;
  normalizedColCount: number;
  minAllowedColumns: number;
  onToggleEditMode: () => void;
  onOpenAddWidgetModal: () => void;
  onCancelPlacement: () => void;
  onAddColumn?: () => void;
  onRemoveColumn?: () => void;
}

export const DashboardGridHeader: React.FC<DashboardGridHeaderProps> = ({
  isEditMode,
  dashboardSaveStatus,
  pendingWidget,
  normalizedColCount,
  minAllowedColumns,
  onToggleEditMode,
  onOpenAddWidgetModal,
  onCancelPlacement,
  onAddColumn,
  onRemoveColumn,
}) => {
  return (
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
              onClick={onOpenAddWidgetModal}
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
                onClick={onCancelPlacement}
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
  );
};
