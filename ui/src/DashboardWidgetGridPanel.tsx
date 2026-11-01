import React from 'react';
import {
  getWidgetCompositeKey,
  PlacedDashboardWidget,
  PluginWidgetDescriptor,
} from './themeAndTypes';

export interface DashboardWidgetGridPanelProps {
  isEditMode: boolean;
  availableWidgets: PluginWidgetDescriptor[];
  placedWidgets: PlacedDashboardWidget[];
  onToggleEditMode: () => void;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
  onRemoveWidget: (pluginId: string, widgetId: string) => void;
  onMoveWidgetUp?: (pluginId: string, widgetId: string) => void;
  onMoveWidgetDown?: (pluginId: string, widgetId: string) => void;
  isAdmin?: boolean;
  dashboardLayoutSource?: string;
  dashboardSaveStatus?: string;
  selectedAdminRoleTarget?: string;
  onSelectAdminRoleTarget?: (roleId: string) => void;
  onSaveRoleDefaultLayout?: (roleId: string) => void;
  onAppendWidgetToRoleDefault?: (roleId: string, widget: PluginWidgetDescriptor) => void;
  onRegisterWidgetIframeRef: (widgetId: string, pluginId: string, el: HTMLIFrameElement | null) => void;
}

const ADMIN_ROLE_TARGETS = ['TEACHER', 'HOD', 'DSL', 'ADMIN', 'SENCO', 'SUPPORT'];

export const DashboardWidgetGridPanel: React.FC<DashboardWidgetGridPanelProps> = ({
  isEditMode,
  availableWidgets,
  placedWidgets,
  onToggleEditMode,
  onPlaceWidget,
  onRemoveWidget,
  onMoveWidgetUp,
  onMoveWidgetDown,
  isAdmin = false,
  dashboardLayoutSource = 'SYSTEM_FALLBACK',
  dashboardSaveStatus = '',
  selectedAdminRoleTarget = 'TEACHER',
  onSelectAdminRoleTarget,
  onSaveRoleDefaultLayout,
  onAppendWidgetToRoleDefault,
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
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', flexWrap: 'wrap' }}>
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
          <span
            data-testid="dashboard-layout-source"
            style={{
              padding: '2px 8px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              backgroundColor: 'var(--mis-bg-elevated)',
              border: '1px solid var(--mis-border-subtle)',
              fontSize: '11px',
              fontWeight: 600,
              color: 'var(--mis-text-muted)',
            }}
          >
            {dashboardLayoutSource}
          </span>
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

      {isAdmin && (
        <div
          data-testid="admin-role-dashboard-controls"
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
          <span style={{ fontSize: '12px', fontWeight: 700 }}>Admin Role Defaults (core.db):</span>
          <label style={{ fontSize: '12px', display: 'flex', alignItems: 'center', gap: '4px' }}>
            Target Role:
            <select
              data-testid="admin-role-default-select"
              value={selectedAdminRoleTarget}
              onChange={(e) => onSelectAdminRoleTarget?.(e.target.value)}
              style={{ fontSize: '12px', padding: '2px 6px' }}
            >
              {ADMIN_ROLE_TARGETS.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            data-testid="admin-save-role-default-btn"
            onClick={() => onSaveRoleDefaultLayout?.(selectedAdminRoleTarget)}
            style={{
              padding: '4px 10px',
              backgroundColor: 'var(--mis-brand-primary)',
              color: 'var(--mis-brand-primary-fg)',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              fontSize: '11px',
              fontWeight: 700,
              cursor: 'pointer',
            }}
          >
            Save Current Grid as {selectedAdminRoleTarget} Default
          </button>
          {availableWidgets.map((w) => (
            <button
              key={`append:${getWidgetCompositeKey(w.plugin_id, w.id)}`}
              type="button"
              data-testid={`admin-append-to-role-btn-${w.id}`}
              data-plug-id={w.plugin_id}
              data-widget-key={getWidgetCompositeKey(w.plugin_id, w.id)}
              onClick={() => onAppendWidgetToRoleDefault?.(selectedAdminRoleTarget, w)}
              style={{
                padding: '4px 8px',
                backgroundColor: 'var(--mis-bg-surface)',
                color: 'var(--mis-text-primary)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                fontSize: '11px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              + Append {w.title} to {selectedAdminRoleTarget}
            </button>
          ))}
        </div>
      )}

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
          gridAutoFlow: 'row',
          gap: '12px',
        }}
      >
        {placedWidgets.map((w, idx) => {
          const compositeKey = getWidgetCompositeKey(w.plugin_id, w.id);
          const isPermitted = w.permitted !== false;
          const presetWidth = w.width === 1 || w.width === 2 ? w.width : w.colSpan;
          const presetHeight = w.height === 1 || w.height === 2 ? w.height : w.rowSpan;
          return (
            <div
              key={compositeKey}
              data-testid={`placed-widget-card-${w.id}`}
              data-order-index={String(idx)}
              data-plug-id={w.plugin_id}
              data-widget-id={w.id}
              data-widget-key={compositeKey}
              data-width={String(presetWidth)}
              data-height={String(presetHeight)}
              data-col-span={String(presetWidth)}
              data-row-span={String(presetHeight)}
              style={{
                gridColumn: `span ${presetWidth}`,
                gridRow: `span ${presetHeight}`,
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
                  {w.title} ({presetWidth}×{presetHeight})
                </span>

                {isEditMode && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', fontSize: '11px', flexWrap: 'wrap' }}>
                    <button
                      type="button"
                      data-testid={`move-widget-up-btn-${w.id}`}
                      data-plug-id={w.plugin_id}
                      data-widget-key={compositeKey}
                      disabled={idx === 0}
                      onClick={() => onMoveWidgetUp?.(w.plugin_id, w.id)}
                      style={{
                        padding: '2px 6px',
                        backgroundColor: 'var(--mis-bg-surface)',
                        color: 'var(--mis-text-primary)',
                        border: '1px solid var(--mis-border-subtle)',
                        borderRadius: 'var(--mis-radius-sm, 4px)',
                        fontSize: '11px',
                        fontWeight: 600,
                        cursor: idx === 0 ? 'default' : 'pointer',
                      }}
                    >
                      Move Up
                    </button>
                    <button
                      type="button"
                      data-testid={`move-widget-down-btn-${w.id}`}
                      data-plug-id={w.plugin_id}
                      data-widget-key={compositeKey}
                      disabled={idx === placedWidgets.length - 1}
                      onClick={() => onMoveWidgetDown?.(w.plugin_id, w.id)}
                      style={{
                        padding: '2px 6px',
                        backgroundColor: 'var(--mis-bg-surface)',
                        color: 'var(--mis-text-primary)',
                        border: '1px solid var(--mis-border-subtle)',
                        borderRadius: 'var(--mis-radius-sm, 4px)',
                        fontSize: '11px',
                        fontWeight: 600,
                        cursor: idx === placedWidgets.length - 1 ? 'default' : 'pointer',
                      }}
                    >
                      Move Down
                    </button>
                    <button
                      type="button"
                      data-testid={`remove-widget-btn-${w.id}`}
                      data-plug-id={w.plugin_id}
                      data-widget-key={compositeKey}
                      onClick={() => onRemoveWidget(w.plugin_id, w.id)}
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
                    minHeight: `${w.rowSpan * 110}px`,
                    border: '1px solid var(--mis-border-subtle)',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    backgroundColor: 'var(--mis-bg-surface)',
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
                    minHeight: `${w.rowSpan * 110}px`,
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
            </div>
          );
        })}
      </div>
    </section>
  );
};
