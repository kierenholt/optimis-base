import React from 'react';
import {
  ActiveCallerIdentity,
  PluginNavEntry,
  StaffPersonaSummary,
  ThemeMode,
} from '../themeAndTypes';

export interface HostTopbarProps {
  deepLinkNavigationStatus: string;
  activeRouteUrl: string;
  activePlugin?: PluginNavEntry | null;
  isCurrentRoutePinned?: boolean;
  onTogglePinCurrentRoute?: () => void;
  devPersonaSwitcherEnabled: boolean;
  availablePersonas: StaffPersonaSummary[];
  activeCaller: ActiveCallerIdentity | null;
  activeStaffId: string;
  personaSwitchError: string;
  themeMode: ThemeMode;
  onSelectPersona: (staffId: string) => Promise<void>;
  onToggleThemeMode: () => void;
  onOpenCommandPalette?: () => void;
}

export const HostTopbar: React.FC<HostTopbarProps> = ({
  deepLinkNavigationStatus,
  activeRouteUrl,
  activePlugin,
  isCurrentRoutePinned,
  onTogglePinCurrentRoute,
  devPersonaSwitcherEnabled,
  availablePersonas,
  activeCaller,
  activeStaffId,
  personaSwitchError,
  themeMode,
  onSelectPersona,
  onToggleThemeMode,
  onOpenCommandPalette,
}) => {
  return (
    <header
      data-testid="host-topbar"
      style={{
        backgroundColor: 'var(--mis-bg-surface)',
        borderBottom: '1px solid var(--mis-border-subtle)',
        padding: '12px 24px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        gap: '16px',
        flexWrap: 'wrap',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
        <span style={{ fontSize: '15px', fontWeight: 700 }}>
          Optimis School MIS — Host Shell
        </span>
        <span
          data-testid="deep-link-navigation-status"
          style={{
            padding: '3px 8px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'var(--mis-bg-elevated)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '12px',
            fontWeight: 600,
          }}
        >
          {deepLinkNavigationStatus}
        </span>
        <span
          data-testid="host-active-route-url"
          style={{
            padding: '3px 8px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'var(--mis-bg-elevated)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '12px',
            fontFamily: 'monospace',
            fontWeight: 600,
          }}
        >
          {activeRouteUrl}
        </span>
        {activePlugin && (
          <span
            data-testid="topbar-active-plugin-badge"
            data-plug-id={activePlugin.id}
            data-permitted-count={String(activePlugin.permitted_methods.length)}
            data-permitted-methods={activePlugin.permitted_methods.join(',')}
            style={{
              padding: '3px 8px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              backgroundColor: 'var(--mis-bg-elevated)',
              border: '1px solid var(--mis-border-subtle)',
              fontSize: '12px',
              fontWeight: 600,
              display: 'inline-flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            <span>{activePlugin.title}</span>
            <span
              data-testid="topbar-active-plugin-methods-count"
              style={{
                fontSize: '11px',
                padding: '1px 6px',
                borderRadius: '999px',
                backgroundColor: 'var(--mis-brand-primary)',
                color: 'var(--mis-brand-primary-fg)',
              }}
            >
              {activePlugin.permitted_methods.length}
            </span>
          </span>
        )}
        {activePlugin && onTogglePinCurrentRoute && (
          <button
            type="button"
            data-testid="topbar-pin-view-btn"
            onClick={onTogglePinCurrentRoute}
            style={{
              padding: '4px 10px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-strong)',
              backgroundColor: isCurrentRoutePinned
                ? 'var(--mis-brand-primary)'
                : 'var(--mis-bg-elevated)',
              color: isCurrentRoutePinned
                ? 'var(--mis-brand-primary-fg)'
                : 'var(--mis-text-primary)',
              fontSize: '12px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            {isCurrentRoutePinned ? '📌 Pinned' : '📌 Pin View'}
          </button>
        )}
      </div>

      <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flexWrap: 'wrap' }}>
        {onOpenCommandPalette && (
          <button
            type="button"
            data-testid="command-palette-trigger-btn"
            onClick={onOpenCommandPalette}
            style={{
              padding: '5px 10px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-strong)',
              backgroundColor: 'var(--mis-bg-elevated)',
              color: 'var(--mis-text-primary)',
              fontSize: '12px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            Search (Ctrl+K)
          </button>
        )}

        {devPersonaSwitcherEnabled && (
          <div
            data-testid="persona-switcher-container"
            style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}
          >
            <label
              htmlFor="persona-switcher-select"
              style={{ fontSize: '12px', fontWeight: 600, color: 'var(--mis-text-secondary)' }}
            >
              Persona:
            </label>
            <select
              id="persona-switcher-select"
              data-testid="persona-switcher-select"
              value={activeCaller?.staff_id || activeStaffId || ''}
              onChange={(e) => {
                void onSelectPersona(e.target.value);
              }}
              style={{
                padding: '5px 10px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                border: '1px solid var(--mis-border-strong)',
                backgroundColor: 'var(--mis-bg-elevated)',
                color: 'var(--mis-text-primary)',
                fontSize: '12px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              {activeCaller?.staff_id &&
                !availablePersonas.some((p) => p.staff_id === activeCaller.staff_id) && (
                  <option value={activeCaller.staff_id}>
                    {activeCaller.staff_id} [{activeCaller.active_roles.join(', ')}]
                  </option>
                )}
              {availablePersonas.map((p) => (
                <option
                  key={p.staff_id}
                  value={p.staff_id}
                  data-testid={`persona-option-${p.staff_id}`}
                >
                  {p.display_name} — {p.job_title} [{p.active_roles.join(', ')}]
                </option>
              ))}
            </select>
            {activeCaller && (
              <span
                data-testid="active-persona-badge"
                data-staff-id={activeCaller.staff_id}
                data-roles={activeCaller.active_roles.join(',')}
                data-scopes={activeCaller.scopes.join(',')}
                style={{
                  padding: '3px 8px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  backgroundColor: 'var(--mis-bg-elevated)',
                  border: '1px solid var(--mis-border-subtle)',
                  fontSize: '11px',
                  fontWeight: 700,
                }}
              >
                <span data-testid="topbar-active-persona">
                  {activeCaller.staff_id} ({activeCaller.active_roles.join(', ')})
                </span>
              </span>
            )}
            {personaSwitchError && (
              <span
                data-testid="persona-switcher-error"
                style={{ fontSize: '12px', color: 'var(--mis-status-danger)', fontWeight: 600 }}
              >
                {personaSwitchError}
              </span>
            )}
          </div>
        )}

        <span style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
          Theme:{' '}
          <strong data-testid="active-theme-mode-indicator">{themeMode}</strong>
        </span>
        <button
          type="button"
          data-testid="theme-mode-toggle-btn"
          onClick={onToggleThemeMode}
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
          {themeMode === 'light' ? 'Switch to High-Contrast' : 'Switch to Light Theme'}
        </button>
      </div>
    </header>
  );
};
