import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  getWidgetCompositeKey,
  PinnedRecentEntry,
  PlacedDashboardWidget,
  PluginNavEntry,
  PluginViewDescriptor,
  PluginWidgetDescriptor,
  SearchGroupResult,
  SearchPupilResult,
} from '../themeAndTypes';

export interface IndexedViewResult {
  pluginId: string;
  pluginTitle: string;
  view: PluginViewDescriptor;
  permittedMethods: string[];
}

export interface GlobalSearchSharedProps {
  installedPlugins: PluginNavEntry[];
  placedWidgets: PlacedDashboardWidget[];
  entityLinkHandlers: Record<string, string>;
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  referenceDate: string;
  academicYearId: string;
  onPlaceWidget: (widget: PluginWidgetDescriptor) => void;
  onSelectViewResult: (pluginId: string, view: PluginViewDescriptor) => void;
  onSelectPupilResult: (pupilId: string) => void;
  onSelectGroupResult: (groupId: string) => void;
}

function matchesViewItem(item: IndexedViewResult, qLower: string): boolean {
  if (!qLower) {
    return true;
  }
  const vTitle = (item.view.title || item.pluginTitle || '').toLowerCase();
  const pTitle = (item.pluginTitle || '').toLowerCase();
  const vId = (item.view.id || '').toLowerCase();
  const vPath = (item.view.path || '').toLowerCase();
  if (
    vTitle.includes(qLower) ||
    pTitle.includes(qLower) ||
    vId.includes(qLower) ||
    vPath.includes(qLower)
  ) {
    return true;
  }
  if (Array.isArray(item.view.keywords)) {
    for (const kw of item.view.keywords) {
      if (kw.toLowerCase().includes(qLower)) {
        return true;
      }
    }
  }
  return false;
}

function matchesWidgetItem(widget: PluginWidgetDescriptor, pluginTitle: string, qLower: string): boolean {
  if (!qLower) {
    return true;
  }
  const wTitle = (widget.title || '').toLowerCase();
  const wId = (widget.id || '').toLowerCase();
  const pTitle = (pluginTitle || '').toLowerCase();
  if (wTitle.includes(qLower) || wId.includes(qLower) || pTitle.includes(qLower)) {
    return true;
  }
  if (Array.isArray(widget.keywords)) {
    for (const kw of widget.keywords) {
      if (kw.toLowerCase().includes(qLower)) {
        return true;
      }
    }
  }
  return false;
}

function isParameterizedViewResolvable(
  path: string | undefined,
  activePupilId: string,
  activeGroupId: string,
  activeStaffId: string
): boolean {
  const p = path || '/';
  if (p.includes(':pupil_id') && !activePupilId) {
    return false;
  }
  if (p.includes(':group_id') && !activeGroupId) {
    return false;
  }
  if (p.includes(':staff_id') && !activeStaffId) {
    return false;
  }
  return true;
}

function resolveParameterizedViewSubpath(
  path: string | undefined,
  activePupilId: string,
  activeGroupId: string,
  activeStaffId: string
): string {
  let resolved = path || '/';
  if (resolved.includes(':pupil_id')) {
    if (!activePupilId) {
      throw new Error('Missing mandatory activePupilId for :pupil_id view route');
    }
    resolved = resolved.replace(':pupil_id', activePupilId);
  }
  if (resolved.includes(':group_id')) {
    if (!activeGroupId) {
      throw new Error('Missing mandatory activeGroupId for :group_id view route');
    }
    resolved = resolved.replace(':group_id', activeGroupId);
  }
  if (resolved.includes(':staff_id')) {
    if (!activeStaffId) {
      throw new Error('Missing mandatory activeStaffId for :staff_id view route');
    }
    resolved = resolved.replace(':staff_id', activeStaffId);
  }
  return resolved;
}

function useScopedEntitySearch(
  query: string,
  activeStaffId: string,
  referenceDate: string,
  academicYearId: string,
  enabled: boolean
) {
  const [pupils, setPupils] = useState<SearchPupilResult[]>([]);
  const [groups, setGroups] = useState<SearchGroupResult[]>([]);

  useEffect(() => {
    const trimmed = query.trim();
    if (!enabled || !trimmed) {
      setPupils([]);
      setGroups([]);
      return;
    }

    const controller = new AbortController();
    const fetchEntities = async () => {
      try {
        const qs = new URLSearchParams();
        qs.set('q', trimmed);
        if (activeStaffId) {
          qs.set('staff_id', activeStaffId);
        }
        if (referenceDate) {
          qs.set('reference_date', referenceDate);
        }
        if (academicYearId) {
          qs.set('academic_year_id', academicYearId);
        }
        const res = await fetch(`/api/system/search?${qs.toString()}`, {
          signal: controller.signal,
        });
        if (!res.ok || controller.signal.aborted) {
          return;
        }
        const body = await res.json();
        if (controller.signal.aborted || !body || !body.ok) {
          return;
        }
        setPupils(Array.isArray(body.pupils) ? body.pupils : []);
        setGroups(Array.isArray(body.groups) ? body.groups : []);
      } catch {
        if (!controller.signal.aborted) {
          setPupils([]);
          setGroups([]);
        }
      }
    };

    void fetchEntities();
    return () => {
      controller.abort();
    };
  }, [query, activeStaffId, referenceDate, academicYearId, enabled]);

  return { pupils, groups };
}

export interface LandingPageTopHalfSearchProps extends GlobalSearchSharedProps {
  onOpenCommandPalette: () => void;
}

export const LandingPageTopHalfSearch: React.FC<LandingPageTopHalfSearchProps> = ({
  installedPlugins,
  placedWidgets,
  activePupilId,
  activeGroupId,
  activeStaffId,
  referenceDate,
  academicYearId,
  onPlaceWidget,
  onSelectViewResult,
  onSelectPupilResult,
  onSelectGroupResult,
  onOpenCommandPalette,
}) => {
  const [query, setQuery] = useState<string>('');
  const qLower = query.trim().toLowerCase();

  const { pupils, groups } = useScopedEntitySearch(
    query,
    activeStaffId,
    referenceDate,
    academicYearId,
    qLower.length > 0
  );

  const matchedViews = useMemo<IndexedViewResult[]>(() => {
    if (!qLower) {
      return [];
    }
    const results: IndexedViewResult[] = [];
    for (const p of installedPlugins) {
      const views = Array.isArray(p.views) ? p.views : [];
      const permittedMethods = Array.isArray(p.permitted_methods) ? p.permitted_methods : [];
      for (const v of views) {
        if (!isParameterizedViewResolvable(v.path, activePupilId, activeGroupId, activeStaffId)) {
          continue;
        }
        const candidate: IndexedViewResult = {
          pluginId: p.id,
          pluginTitle: p.title,
          view: v,
          permittedMethods,
        };
        if (matchesViewItem(candidate, qLower)) {
          results.push(candidate);
        }
      }
    }
    return results;
  }, [installedPlugins, qLower, activePupilId, activeGroupId, activeStaffId]);

  const matchedWidgets = useMemo<Array<{ widget: PluginWidgetDescriptor; pluginTitle: string }>>(() => {
    if (!qLower) {
      return [];
    }
    const results: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string }> = [];
    for (const p of installedPlugins) {
      const widgets = Array.isArray(p.widgets) ? p.widgets : [];
      for (const w of widgets) {
        if (matchesWidgetItem(w, p.title, qLower)) {
          results.push({ widget: w, pluginTitle: p.title });
        }
      }
    }
    return results;
  }, [installedPlugins, qLower]);

  const placedWidgetKeys = useMemo(
    () => new Set(placedWidgets.map((w) => getWidgetCompositeKey(w.plugin_id, w.id))),
    [placedWidgets]
  );

  return (
    <section
      data-testid="landing-page-search-panel"
      style={{
        backgroundColor: 'var(--mis-bg-surface)',
        border: '1px solid var(--mis-border-strong)',
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
        <div>
          <h2 style={{ fontSize: '15px', fontWeight: 700, margin: 0 }}>
            Global Search &amp; Discovery
          </h2>
          <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)', marginTop: '2px' }}>
            Search plugin views, dashboard widgets, pupils, and teaching groups by name, action verb, or synonym
          </div>
        </div>
        <button
          type="button"
          data-testid="landing-open-command-palette-btn"
          onClick={onOpenCommandPalette}
          style={{
            padding: '5px 10px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            border: '1px solid var(--mis-border-subtle)',
            backgroundColor: 'var(--mis-bg-elevated)',
            color: 'var(--mis-text-primary)',
            fontSize: '12px',
            fontWeight: 600,
            cursor: 'pointer',
          }}
        >
          Command Palette (Ctrl+K)
        </button>
      </div>

      <input
        type="search"
        data-testid="landing-search-input"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search views, widgets, pupils, or groups (e.g. detention, ehcp, register, Ada Lovelace)..."
        style={{
          width: '100%',
          boxSizing: 'border-box',
          padding: '10px 12px',
          borderRadius: 'var(--mis-radius-sm, 4px)',
          border: '1px solid var(--mis-border-strong)',
          backgroundColor: 'var(--mis-bg-elevated)',
          color: 'var(--mis-text-primary)',
          fontSize: '13px',
        }}
      />

      {qLower.length > 0 && (
        <div
          data-testid="landing-search-results"
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '10px',
            paddingTop: '4px',
          }}
        >
          {matchedViews.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Plugin Views ({matchedViews.length})
              </div>
              {matchedViews.map(({ pluginId, pluginTitle, view }) => (
                <button
                  key={`${pluginId}:${view.id}`}
                  type="button"
                  data-testid={`landing-search-result-view-${pluginId}-${view.id}`}
                  onClick={() => onSelectViewResult(pluginId, view)}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 12px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    textAlign: 'left',
                    cursor: 'pointer',
                  }}
                >
                  <div>
                    <div style={{ fontSize: '13px', fontWeight: 700 }}>{view.title || pluginTitle}</div>
                    <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                      {pluginTitle} • /p/{pluginId}
                      {view.path === '/' ? '' : view.path}
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600 }}>Open View →</span>
                </button>
              ))}
            </div>
          )}

          {matchedWidgets.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Dashboard Widgets ({matchedWidgets.length})
              </div>
              {matchedWidgets.map(({ widget, pluginTitle }) => {
                const compositeKey = getWidgetCompositeKey(widget.plugin_id, widget.id);
                const isPlaced = placedWidgetKeys.has(compositeKey);
                return (
                  <div
                    key={compositeKey}
                    data-testid={`landing-search-result-widget-${widget.id}`}
                    data-plug-id={widget.plugin_id}
                    data-widget-key={compositeKey}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      gap: '8px',
                      padding: '8px 12px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: '1px solid var(--mis-border-subtle)',
                      backgroundColor: 'var(--mis-bg-elevated)',
                    }}
                  >
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>{widget.title}</div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        Widget • {pluginTitle}
                      </div>
                    </div>
                    <button
                      type="button"
                      data-testid={`landing-search-add-widget-btn-${widget.id}`}
                      data-plug-id={widget.plugin_id}
                      data-widget-key={compositeKey}
                      disabled={isPlaced}
                      onClick={() => onPlaceWidget(widget)}
                      style={{
                        padding: '5px 10px',
                        borderRadius: 'var(--mis-radius-sm, 4px)',
                        border: '1px solid var(--mis-border-subtle)',
                        backgroundColor: isPlaced ? 'var(--mis-bg-surface)' : 'var(--mis-brand-primary)',
                        color: isPlaced ? 'var(--mis-text-muted)' : 'var(--mis-brand-primary-fg)',
                        fontSize: '12px',
                        fontWeight: 600,
                        cursor: isPlaced ? 'default' : 'pointer',
                      }}
                    >
                      {isPlaced ? 'Added to Dashboard' : '+ Add to Dashboard'}
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {pupils.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Pupils ({pupils.length})
              </div>
              {pupils.map((pupil) => (
                <button
                  key={pupil.id}
                  type="button"
                  data-testid={`landing-search-result-pupil-${pupil.id}`}
                  onClick={() => onSelectPupilResult(pupil.id)}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 12px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    textAlign: 'left',
                    cursor: 'pointer',
                  }}
                >
                  <div>
                    <div style={{ fontSize: '13px', fontWeight: 700 }}>
                      {pupil.first_name} {pupil.last_name}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                      UPN {pupil.upn} • Year {pupil.nc_year_group || 'N/A'}
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600 }}>Open Student →</span>
                </button>
              ))}
            </div>
          )}

          {groups.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Groups ({groups.length})
              </div>
              {groups.map((group) => (
                <button
                  key={group.id}
                  type="button"
                  data-testid={`landing-search-result-group-${group.id}`}
                  onClick={() => onSelectGroupResult(group.id)}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 12px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    textAlign: 'left',
                    cursor: 'pointer',
                  }}
                >
                  <div>
                    <div style={{ fontSize: '13px', fontWeight: 700 }}>
                      {group.code} — {group.name}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                      {group.group_type}
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600 }}>Open Group →</span>
                </button>
              ))}
            </div>
          )}

          {matchedViews.length === 0 &&
            matchedWidgets.length === 0 &&
            pupils.length === 0 &&
            groups.length === 0 && (
              <div
                data-testid="landing-search-empty-state"
                style={{ fontSize: '12px', color: 'var(--mis-text-secondary)', padding: '6px 0' }}
              >
                No matching views, widgets, pupils, or groups found.
              </div>
            )}
        </div>
      )}
    </section>
  );
};

export interface GlobalCommandPaletteModalProps extends GlobalSearchSharedProps {
  isOpen: boolean;
  onClose: () => void;
  pinnedRecents?: PinnedRecentEntry[];
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const GlobalCommandPaletteModal: React.FC<GlobalCommandPaletteModalProps> = ({
  isOpen,
  onClose,
  installedPlugins,
  placedWidgets,
  activePupilId,
  activeGroupId,
  activeStaffId,
  referenceDate,
  academicYearId,
  pinnedRecents = [],
  onPlaceWidget,
  onSelectViewResult,
  onSelectPupilResult,
  onSelectGroupResult,
  onTogglePinView,
}) => {
  const [query, setQuery] = useState<string>('');
  const inputRef = useRef<HTMLInputElement | null>(null);
  const qLower = query.trim().toLowerCase();

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    }
  }, [isOpen]);

  const { pupils, groups } = useScopedEntitySearch(
    query,
    activeStaffId,
    referenceDate,
    academicYearId,
    isOpen && qLower.length > 0
  );

  const matchedViews = useMemo<IndexedViewResult[]>(() => {
    const results: IndexedViewResult[] = [];
    for (const p of installedPlugins) {
      const views = Array.isArray(p.views) ? p.views : [];
      const permittedMethods = Array.isArray(p.permitted_methods) ? p.permitted_methods : [];
      for (const v of views) {
        if (!isParameterizedViewResolvable(v.path, activePupilId, activeGroupId, activeStaffId)) {
          continue;
        }
        const candidate: IndexedViewResult = {
          pluginId: p.id,
          pluginTitle: p.title,
          view: v,
          permittedMethods,
        };
        if (matchesViewItem(candidate, qLower)) {
          results.push(candidate);
        }
      }
    }
    return results;
  }, [installedPlugins, qLower, activePupilId, activeGroupId, activeStaffId]);

  const matchedWidgets = useMemo<Array<{ widget: PluginWidgetDescriptor; pluginTitle: string }>>(() => {
    const results: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string }> = [];
    for (const p of installedPlugins) {
      const widgets = Array.isArray(p.widgets) ? p.widgets : [];
      for (const w of widgets) {
        if (matchesWidgetItem(w, p.title, qLower)) {
          results.push({ widget: w, pluginTitle: p.title });
        }
      }
    }
    return results;
  }, [installedPlugins, qLower]);

  const placedWidgetKeys = useMemo(
    () => new Set(placedWidgets.map((w) => getWidgetCompositeKey(w.plugin_id, w.id))),
    [placedWidgets]
  );
  const pinnedRoutesSet = useMemo(() => new Set(pinnedRecents.map((r) => r.route)), [pinnedRecents]);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      data-testid="command-palette-backdrop"
      onClick={onClose}
      style={{
        position: 'fixed',
        inset: 0,
        backgroundColor: 'rgba(15, 23, 42, 0.55)',
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'flex-start',
        paddingTop: '80px',
        zIndex: 9999,
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label="Global Command Palette"
        data-testid="command-palette-modal"
        onClick={(e) => e.stopPropagation()}
        style={{
          width: '100%',
          maxWidth: '640px',
          maxHeight: '75vh',
          backgroundColor: 'var(--mis-bg-surface)',
          color: 'var(--mis-text-primary)',
          border: '1px solid var(--mis-border-strong)',
          borderRadius: 'var(--mis-radius-md, 8px)',
          boxShadow: 'var(--mis-shadow-card)',
          padding: '16px',
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          overflowY: 'auto',
        }}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '14px', fontWeight: 700 }}>
            Global Command Palette (Ctrl+K)
          </span>
          <button
            type="button"
            data-testid="command-palette-close-btn"
            onClick={onClose}
            style={{
              padding: '3px 8px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-subtle)',
              backgroundColor: 'var(--mis-bg-elevated)',
              color: 'var(--mis-text-primary)',
              fontSize: '11px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            Esc
          </button>
        </div>

        <input
          ref={inputRef}
          type="search"
          autoFocus
          data-testid="command-palette-input"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Type a command, synonym, pupil name, or group code..."
          style={{
            width: '100%',
            boxSizing: 'border-box',
            padding: '10px 12px',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            border: '1px solid var(--mis-border-strong)',
            backgroundColor: 'var(--mis-bg-elevated)',
            color: 'var(--mis-text-primary)',
            fontSize: '13px',
          }}
        />

        <div
          data-testid="command-palette-results"
          style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}
        >
          {matchedViews.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Views ({matchedViews.length})
              </div>
              {matchedViews.map(({ pluginId, pluginTitle, view, permittedMethods }) => {
                const candidateSubpath = resolveParameterizedViewSubpath(
                  view.path,
                  activePupilId,
                  activeGroupId,
                  activeStaffId
                );
                const candidateRoute = `/p/${pluginId}${candidateSubpath === '/' ? '' : candidateSubpath}`;
                const isPinned = pinnedRoutesSet.has(candidateRoute);

                return (
                  <div
                    key={`${pluginId}:${view.id}`}
                    style={{
                      display: 'flex',
                      alignItems: 'stretch',
                      gap: '6px',
                    }}
                  >
                    <button
                      type="button"
                      data-testid={`command-palette-result-view-${pluginId}-${view.id}`}
                      data-permitted-count={String(permittedMethods.length)}
                      data-permitted-methods={permittedMethods.join(',')}
                      onClick={() => {
                        onSelectViewResult(pluginId, view);
                        onClose();
                      }}
                      style={{
                        flex: 1,
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        gap: '8px',
                        padding: '8px 12px',
                        borderRadius: 'var(--mis-radius-sm, 4px)',
                        border: '1px solid var(--mis-border-subtle)',
                        backgroundColor: 'var(--mis-bg-elevated)',
                        color: 'var(--mis-text-primary)',
                        textAlign: 'left',
                        cursor: 'pointer',
                      }}
                    >
                      <div>
                        <div style={{ fontSize: '13px', fontWeight: 700 }}>{view.title || pluginTitle}</div>
                        <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                          {pluginTitle} • /p/{pluginId}
                          {view.path === '/' ? '' : view.path} ({permittedMethods.length} methods)
                        </div>
                      </div>
                      <span style={{ fontSize: '11px', fontWeight: 600 }}>Jump →</span>
                    </button>
                    {onTogglePinView && (
                      <button
                        type="button"
                        data-testid={`search-pin-view-btn-${pluginId}-${view.id}`}
                        aria-label={isPinned ? `Unpin ${view.title || pluginTitle}` : `Pin ${view.title || pluginTitle}`}
                        onClick={() => onTogglePinView(pluginId, view, pluginTitle)}
                        style={{
                          padding: '6px 10px',
                          borderRadius: 'var(--mis-radius-sm, 4px)',
                          border: '1px solid var(--mis-border-subtle)',
                          backgroundColor: isPinned ? 'var(--mis-brand-primary)' : 'var(--mis-bg-elevated)',
                          color: isPinned ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-primary)',
                          fontSize: '11px',
                          fontWeight: 600,
                          cursor: 'pointer',
                          whiteSpace: 'nowrap',
                        }}
                      >
                        {isPinned ? '★ Pinned' : '☆ Pin'}
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {matchedWidgets.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Dashboard Widgets ({matchedWidgets.length})
              </div>
              {matchedWidgets.map(({ widget, pluginTitle }) => {
                const compositeKey = getWidgetCompositeKey(widget.plugin_id, widget.id);
                const isPlaced = placedWidgetKeys.has(compositeKey);
                return (
                  <div
                    key={compositeKey}
                    data-testid={`command-palette-result-widget-${widget.id}`}
                    data-plug-id={widget.plugin_id}
                    data-widget-key={compositeKey}
                    style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      gap: '8px',
                      padding: '8px 12px',
                      borderRadius: 'var(--mis-radius-sm, 4px)',
                      border: '1px solid var(--mis-border-subtle)',
                      backgroundColor: 'var(--mis-bg-elevated)',
                    }}
                  >
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>{widget.title}</div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        Widget • {pluginTitle}
                      </div>
                    </div>
                    <button
                      type="button"
                      data-testid={`command-palette-add-widget-btn-${widget.id}`}
                      data-plug-id={widget.plugin_id}
                      data-widget-key={compositeKey}
                      disabled={isPlaced}
                      onClick={() => onPlaceWidget(widget)}
                      style={{
                        padding: '5px 10px',
                        borderRadius: 'var(--mis-radius-sm, 4px)',
                        border: '1px solid var(--mis-border-subtle)',
                        backgroundColor: isPlaced ? 'var(--mis-bg-surface)' : 'var(--mis-brand-primary)',
                        color: isPlaced ? 'var(--mis-text-muted)' : 'var(--mis-brand-primary-fg)',
                        fontSize: '12px',
                        fontWeight: 600,
                        cursor: isPlaced ? 'default' : 'pointer',
                      }}
                    >
                      {isPlaced ? 'Added to Dashboard' : '+ Add to Dashboard'}
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {pupils.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Pupils ({pupils.length})
              </div>
              {pupils.map((pupil) => (
                <button
                  key={pupil.id}
                  type="button"
                  data-testid={`command-palette-result-pupil-${pupil.id}`}
                  onClick={() => {
                    onSelectPupilResult(pupil.id);
                    onClose();
                  }}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 12px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    textAlign: 'left',
                    cursor: 'pointer',
                  }}
                >
                  <div>
                    <div style={{ fontSize: '13px', fontWeight: 700 }}>
                      {pupil.first_name} {pupil.last_name}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                      UPN {pupil.upn} • Year {pupil.nc_year_group || 'N/A'}
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600 }}>Open Student →</span>
                </button>
              ))}
            </div>
          )}

          {groups.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Groups ({groups.length})
              </div>
              {groups.map((group) => (
                <button
                  key={group.id}
                  type="button"
                  data-testid={`command-palette-result-group-${group.id}`}
                  onClick={() => {
                    onSelectGroupResult(group.id);
                    onClose();
                  }}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    gap: '8px',
                    padding: '8px 12px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: '1px solid var(--mis-border-subtle)',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    color: 'var(--mis-text-primary)',
                    textAlign: 'left',
                    cursor: 'pointer',
                  }}
                >
                  <div>
                    <div style={{ fontSize: '13px', fontWeight: 700 }}>
                      {group.code} — {group.name}
                    </div>
                    <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                      {group.group_type}
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600 }}>Open Group →</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
