import React from 'react';
import { PluginNavEntry } from '../themeAndTypes';

export interface HostSidebarProps {
  schoolURN: string;
  schoolName: string;
  configuredAcademicYear: string;
  installedPlugins: PluginNavEntry[];
  activePluginId: string;
  activeEntityContext: string;
  onActivatePlugin: (pluginId: string) => void;
}

export const HostSidebar: React.FC<HostSidebarProps> = ({
  schoolURN,
  schoolName,
  configuredAcademicYear,
  installedPlugins,
  activePluginId,
  activeEntityContext,
  onActivatePlugin,
}) => {
  return (
    <aside
      data-testid="host-sidebar"
      style={{
        backgroundColor: 'var(--mis-bg-sidebar)',
        color: 'var(--mis-text-sidebar)',
        borderRight: '1px solid var(--mis-border-subtle)',
        padding: '20px 16px',
        display: 'flex',
        flexDirection: 'column',
        gap: '16px',
      }}
    >
      <div>
        <div style={{ fontSize: '11px', textTransform: 'uppercase', letterSpacing: '0.08em', opacity: 0.8 }}>
          Optimis School MIS • URN {schoolURN}
        </div>
        <h1
          data-testid="host-school-name"
          style={{ fontSize: '16px', fontWeight: 700, margin: '4px 0 0 0', lineHeight: 1.3 }}
        >
          {schoolName}
        </h1>
        <div style={{ fontSize: '12px', marginTop: '4px', opacity: 0.85 }}>
          Academic Cycle: <strong>{configuredAcademicYear}</strong>
        </div>
      </div>

      <nav style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
        <div
          data-testid="nav-item-dashboard"
          style={{
            padding: '8px 12px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            backgroundColor: 'rgba(255, 255, 255, 0.1)',
            fontSize: '13px',
            fontWeight: 600,
            display: 'flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <span>Home Dashboard Grid</span>
        </div>

        {installedPlugins.map((plug) => (
          <button
            key={plug.id}
            type="button"
            data-testid={`nav-item-plugin-${plug.id}`}
            data-permitted-count={String(plug.permitted_methods.length)}
            data-permitted-methods={plug.permitted_methods.join(',')}
            onClick={() => onActivatePlugin(plug.id)}
            style={{
              padding: '8px 12px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              backgroundColor:
                activePluginId === plug.id ? 'var(--mis-brand-primary)' : 'transparent',
              color:
                activePluginId === plug.id
                  ? 'var(--mis-brand-primary-fg)'
                  : 'var(--mis-text-sidebar)',
              border: '1px solid var(--mis-border-subtle)',
              fontSize: '13px',
              fontWeight: 600,
              textAlign: 'left',
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '8px',
            }}
          >
            <span>{plug.title}</span>
            <span
              data-testid={`nav-item-plugin-methods-count-${plug.id}`}
              style={{
                marginLeft: 'auto',
                fontSize: '11px',
                padding: '1px 6px',
                borderRadius: '999px',
                backgroundColor: 'rgba(255, 255, 255, 0.16)',
              }}
            >
              {plug.permitted_methods.length}
            </span>
          </button>
        ))}
      </nav>

      <div
        style={{
          marginTop: 'auto',
          padding: '10px 12px',
          borderRadius: 'var(--mis-radius-sm, 4px)',
          border: '1px solid var(--mis-border-subtle)',
          fontSize: '12px',
        }}
      >
        <div style={{ opacity: 0.75, marginBottom: '4px' }}>Active Entity Context</div>
        <div data-testid="active-entity-context-pill" style={{ fontWeight: 700, wordBreak: 'break-all' }}>
          {activeEntityContext}
        </div>
      </div>
    </aside>
  );
};
