import React from 'react';
import { PinnedRecentEntry, PluginNavEntry } from '../themeAndTypes';
import { MonochromeLucideIcon } from './MonochromeLucideIcon';
import { HostViewMode } from './routeUtils';

export interface HostSidebarProps {
  schoolURN: string;
  schoolName: string;
  configuredAcademicYear: string;
  hostViewMode: HostViewMode;
  activePluginId: string;
  activePlugin?: PluginNavEntry | null;
  isCurrentRoutePinned?: boolean;
  onTogglePinCurrentRoute?: () => void;
  activeRouteUrl: string;
  activeEntityContext: string;
  pinnedRecents: PinnedRecentEntry[];
  onNavigateHome: () => void;
  onOpenSearch: () => void;
  onSelectPinnedRoute: (item: PinnedRecentEntry) => void;
  onUnpinRoute: (route: string) => void;
  onOpenSettings: () => void;
  onLogout: () => void;
}

export const HostSidebar: React.FC<HostSidebarProps> = ({
  schoolURN,
  schoolName,
  configuredAcademicYear,
  hostViewMode,
  activePluginId,
  activePlugin,
  isCurrentRoutePinned,
  onTogglePinCurrentRoute,
  activeRouteUrl,
  pinnedRecents,
  onNavigateHome,
  onOpenSearch,
  onSelectPinnedRoute,
  onUnpinRoute,
  onOpenSettings,
  onLogout,
}) => {
  const isHomeActive = hostViewMode === 'home';
  const isSearchActive = hostViewMode === 'search';
  const isSettingsActive = hostViewMode === 'settings';

  return (
    <aside
      data-testid="host-sidebar"
      data-rail-mode="narrow-icon-rail"
      style={{
        backgroundColor: 'var(--mis-bg-sidebar)',
        color: 'var(--mis-text-sidebar)',
        borderRight: '1px solid var(--mis-border-subtle)',
        padding: '14px 8px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: '12px',
        width: '88px',
        position: 'sticky',
        top: 0,
        height: 'calc(100vh / var(--mis-font-scale, 1))',
        maxHeight: 'calc(100vh / var(--mis-font-scale, 1))',
        overflowY: 'auto',
        boxSizing: 'border-box',
      }}
    >
      {/* Compact School Identity Badge (inside Narrow Icon Rail) */}
      <div
        title={`${schoolName} • URN ${schoolURN} • ${configuredAcademicYear}`}
        style={{
          width: '100%',
          padding: '6px 4px',
          borderRadius: 'var(--mis-radius-sm, 4px)',
          border: '1px solid rgba(255, 255, 255, 0.16)',
          textAlign: 'center',
        }}
      >
        <div style={{ fontSize: '9px', textTransform: 'uppercase', opacity: 0.8 }}>
          URN {schoolURN}
        </div>
        <h1
          data-testid="host-school-name"
          style={{
            fontSize: '10px',
            fontWeight: 700,
            margin: '2px 0 0 0',
            lineHeight: 1.2,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
          }}
        >
          {schoolName}
        </h1>
        <div style={{ fontSize: '9px', marginTop: '2px', opacity: 0.75 }}>
          {configuredAcademicYear}
        </div>
      </div>

      {/* Narrow Icon Rail Primary Actions: Home & Global Search */}
      <nav
        aria-label="Primary Icon Rail"
        style={{
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          gap: '8px',
        }}
      >
        <button
          type="button"
          data-testid="nav-rail-home"
          aria-current={isHomeActive ? 'page' : undefined}
          title="Home Dashboard"
          onClick={onNavigateHome}
          style={{
            width: '100%',
            padding: '8px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: isHomeActive
              ? 'var(--mis-brand-primary)'
              : 'rgba(255, 255, 255, 0.08)',
            color: isHomeActive ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-sidebar)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '11px',
            fontWeight: 700,
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '2px',
          }}
        >
          <MonochromeLucideIcon name="house" size={16} />
          <span data-testid="nav-item-dashboard">Home</span>
        </button>

        <button
          type="button"
          data-testid="nav-rail-search"
          aria-current={isSearchActive ? 'page' : undefined}
          title="Global Search & Discovery"
          onClick={onOpenSearch}
          style={{
            width: '100%',
            padding: '8px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: isSearchActive
              ? 'var(--mis-brand-primary)'
              : 'rgba(255, 255, 255, 0.08)',
            color: isSearchActive ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-sidebar)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '11px',
            fontWeight: 700,
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '2px',
          }}
        >
          <MonochromeLucideIcon name="search" size={16} />
          <span>Search</span>
        </button>
      </nav>

      {/* Pinned Recents Rail Section */}
      <div
        data-testid="nav-rail-pinned-recents"
        style={{
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          gap: '6px',
          borderTop: '1px solid rgba(255, 255, 255, 0.16)',
          paddingTop: '10px',
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
        }}
      >
        <div
          style={{
            fontSize: '9px',
            fontWeight: 700,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            textAlign: 'center',
            opacity: 0.75,
          }}
        >
          Pinned
        </div>

        {hostViewMode === 'plugin' && activePlugin && onTogglePinCurrentRoute && (
          <button
            type="button"
            data-testid="topbar-pin-view-btn"
            onClick={onTogglePinCurrentRoute}
            style={{
              width: '100%',
              padding: '5px 4px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-strong)',
              backgroundColor: isCurrentRoutePinned
                ? 'var(--mis-brand-primary)'
                : 'rgba(255, 255, 255, 0.08)',
              color: isCurrentRoutePinned
                ? 'var(--mis-brand-primary-fg)'
                : 'var(--mis-text-sidebar)',
              fontSize: '10px',
              fontWeight: 600,
              cursor: 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: '4px',
            }}
          >
            <MonochromeLucideIcon name="pin" size={12} />
            <span>{isCurrentRoutePinned ? 'Pinned' : 'Pin Current View'}</span>
          </button>
        )}

        {pinnedRecents.map((item, idx) => {
          const isRouteActive =
            hostViewMode === 'plugin' &&
            (activeRouteUrl === item.route ||
              activeRouteUrl.startsWith(`${item.route}&`) ||
              (item.plug_id && activePluginId === item.plug_id && activeRouteUrl.startsWith(item.route)));
          return (
            <div
              key={item.route}
              style={{
                position: 'relative',
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
              }}
            >
              <button
                type="button"
                data-testid={`pinned-recent-item-${idx}`}
                data-pinned-route={item.route}
                data-plug-id={item.plug_id}
                aria-current={isRouteActive ? 'page' : undefined}
                title={`${item.title} (${item.route})`}
                onClick={() => onSelectPinnedRoute(item)}
                style={{
                  width: '100%',
                  padding: '6px 4px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  backgroundColor: isRouteActive
                    ? 'var(--mis-brand-primary)'
                    : 'rgba(255, 255, 255, 0.08)',
                  color: isRouteActive ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-sidebar)',
                  border: '1px solid var(--mis-border-subtle)',
                  fontSize: '10px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  gap: '2px',
                  overflow: 'hidden',
                }}
              >
                <MonochromeLucideIcon name={item.icon || 'pin'} fallback="pin" size={14} />
                <span
                  style={{
                    maxWidth: '100%',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap',
                  }}
                >
                  {item.title}
                </span>
              </button>
              <button
                type="button"
                data-testid={`unpin-recent-btn-${idx}`}
                title={`Unpin ${item.title}`}
                onClick={(e) => {
                  e.stopPropagation();
                  onUnpinRoute(item.route);
                }}
                style={{
                  position: 'absolute',
                  top: '2px',
                  right: '2px',
                  width: '16px',
                  height: '16px',
                  padding: 0,
                  borderRadius: '999px',
                  border: 'none',
                  backgroundColor: 'rgba(15, 23, 42, 0.75)',
                  color: '#ffffff',
                  fontSize: '9px',
                  lineHeight: 1,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <MonochromeLucideIcon name="x" size={10} />
              </button>
            </div>
          );
        })}
      </div>

      {/* Bottom Rail Actions: Settings, Logout, & Compact Active Entity Context Pill */}
      <div
        style={{
          width: '100%',
          marginTop: 'auto',
          display: 'flex',
          flexDirection: 'column',
          gap: '6px',
          borderTop: '1px solid rgba(255, 255, 255, 0.16)',
          paddingTop: '10px',
        }}
      >
        <button
          type="button"
          data-testid="nav-rail-settings"
          aria-current={isSettingsActive ? 'page' : undefined}
          title="Host & Workspace Settings"
          onClick={onOpenSettings}
          style={{
            width: '100%',
            padding: '6px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: isSettingsActive
              ? 'var(--mis-brand-primary)'
              : 'rgba(255, 255, 255, 0.08)',
            color: isSettingsActive ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-sidebar)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '10px',
            fontWeight: 600,
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '2px',
          }}
        >
          <MonochromeLucideIcon name="settings" size={14} />
          <span>Settings</span>
        </button>

        <button
          type="button"
          data-testid="nav-rail-logout"
          title="Sign Out"
          onClick={onLogout}
          style={{
            width: '100%',
            padding: '6px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'rgba(255, 255, 255, 0.08)',
            color: 'var(--mis-text-sidebar)',
            border: '1px solid var(--mis-border-subtle)',
            fontSize: '10px',
            fontWeight: 600,
            cursor: 'pointer',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '2px',
          }}
        >
          <MonochromeLucideIcon name="log-out" size={14} />
          <span>Logout</span>
        </button>
      </div>
    </aside>
  );
};

