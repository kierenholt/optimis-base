import React from 'react';
import { PinnedRecentEntry } from '../themeAndTypes';

export interface HostSidebarProps {
  schoolURN: string;
  schoolName: string;
  configuredAcademicYear: string;
  activePluginId: string;
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
  activePluginId,
  activeRouteUrl,
  activeEntityContext,
  pinnedRecents,
  onNavigateHome,
  onOpenSearch,
  onSelectPinnedRoute,
  onUnpinRoute,
  onOpenSettings,
  onLogout,
}) => {
  const isHomeActive = !activePluginId;

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

      {/* Narrow Icon Rail Primary Actions: Home & Global Search (🔍) */}
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
          <span aria-hidden="true" style={{ fontSize: '15px' }}>
            🏠
          </span>
          <span data-testid="nav-item-dashboard">Home</span>
        </button>

        <button
          type="button"
          data-testid="nav-rail-search"
          title="Global Search & Command Palette (Ctrl+K)"
          onClick={onOpenSearch}
          style={{
            width: '100%',
            padding: '8px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'rgba(255, 255, 255, 0.08)',
            color: 'var(--mis-text-sidebar)',
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
          <span aria-hidden="true" style={{ fontSize: '15px' }}>
            🔍
          </span>
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

        {pinnedRecents.map((item, idx) => {
          const isRouteActive =
            activeRouteUrl === item.route ||
            activeRouteUrl.startsWith(`${item.route}&`) ||
            (item.plug_id && activePluginId === item.plug_id && activeRouteUrl.startsWith(item.route));
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
                <span aria-hidden="true" style={{ fontSize: '12px' }}>
                  📌
                </span>
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
                }}
              >
                ×
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
          title="Host & Workspace Settings"
          onClick={onOpenSettings}
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
          <span aria-hidden="true" style={{ fontSize: '13px' }}>
            ⚙️
          </span>
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
          <span aria-hidden="true" style={{ fontSize: '13px' }}>
            🚪
          </span>
          <span>Logout</span>
        </button>

        <div
          title={`Active Entity Context: ${activeEntityContext}`}
          style={{
            padding: '6px 4px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            border: '1px solid rgba(255, 255, 255, 0.16)',
            fontSize: '9px',
            textAlign: 'center',
          }}
        >
          <div
            data-testid="active-entity-context-pill"
            style={{
              fontWeight: 700,
              wordBreak: 'break-all',
            }}
          >
            {activeEntityContext}
          </div>
        </div>
      </div>
    </aside>
  );
};
