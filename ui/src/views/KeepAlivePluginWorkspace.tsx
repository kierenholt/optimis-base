import React from 'react';
import { PluginNavEntry } from '../themeAndTypes';

export interface KeepAlivePluginWorkspaceProps {
  mountedPluginIds: string[];
  lruPluginOrder: string[];
  installedPlugins: PluginNavEntry[];
  activePluginId: string;
  onRegisterPluginIframeRef: (pluginId: string, el: HTMLIFrameElement | null) => void;
  onIframeLoad: (pluginId: string, win: Window | null) => void;
}

export const KeepAlivePluginWorkspace: React.FC<KeepAlivePluginWorkspaceProps> = ({
  mountedPluginIds,
  lruPluginOrder,
  installedPlugins,
  activePluginId,
  onRegisterPluginIframeRef,
  onIframeLoad,
}) => {
  return (
    <section
      data-testid="dashboard-workspace-panel"
      style={{
        display: activePluginId ? 'block' : 'none',
        backgroundColor: 'var(--mis-bg-surface)',
        border: '1px solid var(--mis-border-subtle)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        padding: '16px',
        boxShadow: 'var(--mis-shadow-card)',
      }}
    >
      <div
        data-testid="keep-alive-lru-pool"
        data-pool-size={String(mountedPluginIds.length)}
        data-lru-order={lruPluginOrder.join(',')}
      >
        {mountedPluginIds.map((pluginId) => {
          const descriptor = installedPlugins.find((p) => p.id === pluginId);
          const entryUrl = descriptor ? descriptor.entry_url : `/p/${pluginId}/ui/index.html`;
          const isActive = pluginId === activePluginId;
          return (
            <iframe
              key={pluginId}
              ref={(el) => onRegisterPluginIframeRef(pluginId, el)}
              onLoad={(e) => {
                onIframeLoad(pluginId, e.currentTarget.contentWindow);
              }}
              data-testid={`plugin-iframe-${pluginId}`}
              data-lru-active={isActive ? 'true' : 'false'}
              sandbox="allow-scripts"
              src={entryUrl}
              title={`Plugin Workspace: ${pluginId}`}
              style={{
                display: isActive ? 'block' : 'none',
                width: '100%',
                height: '320px',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                backgroundColor: 'var(--mis-bg-surface)',
              }}
            />
          );
        })}
      </div>
    </section>
  );
};
