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
import { MonochromeLucideIcon } from './MonochromeLucideIcon';

export interface IndexedViewResult {
  pluginId: string;
  pluginTitle: string;
  pluginIcon?: string;
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
  enabled: boolean,
  allowEmptyQuery: boolean = false
) {
  const [pupils, setPupils] = useState<SearchPupilResult[]>([]);
  const [groups, setGroups] = useState<SearchGroupResult[]>([]);

  useEffect(() => {
    const trimmed = query.trim();
    if (!enabled || (!allowEmptyQuery && !trimmed)) {
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
  }, [query, activeStaffId, referenceDate, academicYearId, enabled, allowEmptyQuery]);

  return { pupils, groups };
}

export type CommandPaletteTab = 'all' | 'plugins' | 'widgets' | 'pupils' | 'groups';

export interface LandingPageTopHalfSearchProps extends GlobalSearchSharedProps {
  onOpenCommandPalette: () => void;
  pinnedRecents?: PinnedRecentEntry[];
  onTogglePinView?: (pluginId: string, view: PluginViewDescriptor, pluginTitle: string) => void;
}

export const LandingPageTopHalfSearch: React.FC<LandingPageTopHalfSearchProps> = ({
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
  onOpenCommandPalette,
}) => {
  const [query, setQuery] = useState<string>('');
  const [activeFilter, setActiveFilter] = useState<CommandPaletteTab>('all');
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
          pluginIcon: p.icon,
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

  const matchedWidgets = useMemo<Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }>>(() => {
    if (!qLower) {
      return [];
    }
    const results: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }> = [];
    for (const p of installedPlugins) {
      const widgets = Array.isArray(p.widgets) ? p.widgets : [];
      for (const w of widgets) {
        if (matchesWidgetItem(w, p.title, qLower)) {
          results.push({ widget: w, pluginTitle: p.title, pluginIcon: p.icon });
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

  const showViews = (activeFilter === 'all' || activeFilter === 'plugins') && matchedViews.length > 0;
  const showWidgets = (activeFilter === 'all' || activeFilter === 'widgets') && matchedWidgets.length > 0;
  const showPupils = (activeFilter === 'all' || activeFilter === 'pupils') && pupils.length > 0;
  const showGroups = (activeFilter === 'all' || activeFilter === 'groups') && groups.length > 0;

  const filterTabs: Array<{ id: CommandPaletteTab; label: string; icon: string }> = [
    { id: 'all', label: 'All', icon: 'search' },
    { id: 'plugins', label: 'Plugins', icon: 'box' },
    { id: 'widgets', label: 'Widgets', icon: 'layout-grid' },
    { id: 'pupils', label: 'Pupils', icon: 'user' },
    { id: 'groups', label: 'Groups', icon: 'users' },
  ];

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
          <h2 style={{ fontSize: '15px', fontWeight: 700, margin: 0, display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
            <MonochromeLucideIcon name="search" size={16} />
            <span>Global Search &amp; Discovery</span>
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
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
          }}
        >
          <MonochromeLucideIcon name="search" size={14} />
          <span>Full list</span>
        </button>
      </div>

      <div
        role="tablist"
        aria-label="Search record type filter"
        data-testid="landing-search-filter-tabs"
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: '6px',
          flexWrap: 'wrap',
        }}
      >
        {filterTabs.map((t) => {
          const isSelected = activeFilter === t.id;
          return (
            <button
              key={t.id}
              type="button"
              role="tab"
              aria-selected={isSelected}
              data-testid={`landing-search-filter-${t.id}`}
              onClick={() => setActiveFilter(t.id)}
              style={{
                padding: '4px 10px',
                borderRadius: 'var(--mis-radius-sm, 4px)',
                border: '1px solid var(--mis-border-subtle)',
                backgroundColor: isSelected ? 'var(--mis-brand-primary)' : 'var(--mis-bg-elevated)',
                color: isSelected ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-primary)',
                fontSize: '12px',
                fontWeight: 600,
                cursor: 'pointer',
                display: 'inline-flex',
                alignItems: 'center',
                gap: '5px',
              }}
            >
              <MonochromeLucideIcon name={t.icon} size={13} />
              <span>{t.label}</span>
            </button>
          );
        })}
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
          {showViews && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Plugin Views ({matchedViews.length})
              </div>
              {matchedViews.map(({ pluginId, pluginTitle, pluginIcon, view }) => {
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
                      data-testid={`landing-search-result-view-${pluginId}-${view.id}`}
                      onClick={() => onSelectViewResult(pluginId, view)}
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
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <MonochromeLucideIcon name={view.icon || pluginIcon || 'box'} fallback="box" size={16} />
                        <div>
                          <div style={{ fontSize: '13px', fontWeight: 700 }}>{view.title || pluginTitle}</div>
                          <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                            {pluginTitle} • /p/{pluginId}
                            {view.path === '/' ? '' : view.path}
                          </div>
                        </div>
                      </div>
                      <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                        <span>Open View</span>
                        <MonochromeLucideIcon name="arrow-right" size={12} />
                      </span>
                    </button>
                    {onTogglePinView && (
                      <button
                        type="button"
                        data-testid={`landing-search-pin-view-btn-${pluginId}-${view.id}`}
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
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}
                      >
                        <MonochromeLucideIcon name="pin" size={12} />
                        <span>{isPinned ? 'Pinned' : 'Pin'}</span>
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {showWidgets && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Dashboard Widgets ({matchedWidgets.length})
              </div>
              {matchedWidgets.map(({ widget, pluginTitle, pluginIcon }) => {
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
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <MonochromeLucideIcon name={widget.icon || pluginIcon || 'layout-grid'} fallback="layout-grid" size={16} />
                      <div>
                        <div style={{ fontSize: '13px', fontWeight: 700 }}>{widget.title}</div>
                        <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                          Widget • {pluginTitle}
                        </div>
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
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '4px',
                      }}
                    >
                      <MonochromeLucideIcon name={isPlaced ? 'check' : 'plus'} size={12} />
                      <span>{isPlaced ? 'Added to Dashboard' : '+ Add to Dashboard'}</span>
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {showPupils && (
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <MonochromeLucideIcon name="user" size={16} />
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>
                        {pupil.first_name} {pupil.last_name}
                      </div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        UPN {pupil.upn} • Year {pupil.nc_year_group || 'N/A'}
                      </div>
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                    <span>Open Student</span>
                    <MonochromeLucideIcon name="arrow-right" size={12} />
                  </span>
                </button>
              ))}
            </div>
          )}

          {showGroups && (
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <MonochromeLucideIcon name="users" size={16} />
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>
                        {group.code} — {group.name}
                      </div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        {group.group_type}
                      </div>
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                    <span>Open Group</span>
                    <MonochromeLucideIcon name="arrow-right" size={12} />
                  </span>
                </button>
              ))}
            </div>
          )}

          {!showViews && !showWidgets && !showPupils && !showGroups && (
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
  const [activeTab, setActiveTab] = useState<CommandPaletteTab>('all');
  const inputRef = useRef<HTMLInputElement | null>(null);
  const qLower = query.trim().toLowerCase();

  useEffect(() => {
    if (isOpen) {
      setQuery('');
      setActiveTab('all');
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
    isOpen,
    true
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
          pluginIcon: p.icon,
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

  const matchedWidgets = useMemo<Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }>>(() => {
    const results: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }> = [];
    for (const p of installedPlugins) {
      const widgets = Array.isArray(p.widgets) ? p.widgets : [];
      for (const w of widgets) {
        if (matchesWidgetItem(w, p.title, qLower)) {
          results.push({ widget: w, pluginTitle: p.title, pluginIcon: p.icon });
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

  const showViews = (activeTab === 'all' || activeTab === 'plugins') && matchedViews.length > 0;
  const showWidgets = (activeTab === 'all' || activeTab === 'widgets') && matchedWidgets.length > 0;
  const showPupils = (activeTab === 'all' || activeTab === 'pupils') && pupils.length > 0;
  const showGroups = (activeTab === 'all' || activeTab === 'groups') && groups.length > 0;

  const tabs: Array<{ id: CommandPaletteTab; label: string; icon: string }> = [
    {
      id: 'all',
      label: `All (${matchedViews.length + matchedWidgets.length + pupils.length + groups.length})`,
      icon: 'search',
    },
    {
      id: 'plugins',
      label: `Plugins (${matchedViews.length})`,
      icon: 'box',
    },
    {
      id: 'widgets',
      label: `Widgets (${matchedWidgets.length})`,
      icon: 'layout-grid',
    },
    {
      id: 'pupils',
      label: `Pupils (${pupils.length})`,
      icon: 'user',
    },
    {
      id: 'groups',
      label: `Groups (${groups.length})`,
      icon: 'users',
    },
  ];

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
          <span style={{ fontSize: '14px', fontWeight: 700, display: 'inline-flex', alignItems: 'center', gap: '6px' }}>
            <MonochromeLucideIcon name="search" size={16} />
            <span>Full list (Ctrl+K)</span>
          </span>
          <button
            type="button"
            aria-label="Close"
            title="Close (Esc)"
            data-testid="command-palette-close-btn"
            onClick={onClose}
            style={{
              padding: '4px',
              borderRadius: 'var(--mis-radius-sm, 4px)',
              border: '1px solid var(--mis-border-subtle)',
              backgroundColor: 'var(--mis-bg-elevated)',
              color: 'var(--mis-text-primary)',
              fontSize: '11px',
              fontWeight: 600,
              cursor: 'pointer',
              display: 'inline-flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <MonochromeLucideIcon name="x" size={14} />
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
          role="tablist"
          aria-label="Category filter tabs"
          data-testid="command-palette-tabs"
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: '6px',
            flexWrap: 'wrap',
            borderBottom: '1px solid var(--mis-border-subtle)',
            paddingBottom: '8px',
          }}
        >
          {tabs.map((t) => {
            const isSelected = activeTab === t.id;
            return (
              <button
                key={t.id}
                type="button"
                role="tab"
                aria-selected={isSelected}
                data-testid={`command-palette-tab-${t.id}`}
                onClick={() => setActiveTab(t.id)}
                style={{
                  padding: '5px 10px',
                  borderRadius: 'var(--mis-radius-sm, 4px)',
                  border: '1px solid var(--mis-border-subtle)',
                  backgroundColor: isSelected ? 'var(--mis-brand-primary)' : 'var(--mis-bg-elevated)',
                  color: isSelected ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-primary)',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '5px',
                }}
              >
                <MonochromeLucideIcon name={t.icon} size={13} />
                <span>{t.label}</span>
              </button>
            );
          })}
        </div>

        <div
          data-testid="command-palette-results"
          style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}
        >
          {showViews && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Views ({matchedViews.length})
              </div>
              {matchedViews.map(({ pluginId, pluginTitle, pluginIcon, view, permittedMethods }) => {
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
                      <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                        <MonochromeLucideIcon name={view.icon || pluginIcon || 'box'} fallback="box" size={16} />
                        <div>
                          <div style={{ fontSize: '13px', fontWeight: 700 }}>{view.title || pluginTitle}</div>
                          <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                            {pluginTitle} • /p/{pluginId}
                            {view.path === '/' ? '' : view.path} ({permittedMethods.length} methods)
                          </div>
                        </div>
                      </div>
                      <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                        <span>Jump</span>
                        <MonochromeLucideIcon name="arrow-right" size={12} />
                      </span>
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
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}
                      >
                        <MonochromeLucideIcon name="pin" size={12} />
                        <span>{isPinned ? 'Pinned' : 'Pin'}</span>
                      </button>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {showWidgets && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
              <div style={{ fontSize: '11px', fontWeight: 700, textTransform: 'uppercase', color: 'var(--mis-text-secondary)' }}>
                Dashboard Widgets ({matchedWidgets.length})
              </div>
              {matchedWidgets.map(({ widget, pluginTitle, pluginIcon }) => {
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
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <MonochromeLucideIcon name={widget.icon || pluginIcon || 'layout-grid'} fallback="layout-grid" size={16} />
                      <div>
                        <div style={{ fontSize: '13px', fontWeight: 700 }}>{widget.title}</div>
                        <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                          Widget • {pluginTitle}
                        </div>
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
                        display: 'inline-flex',
                        alignItems: 'center',
                        gap: '4px',
                      }}
                    >
                      <MonochromeLucideIcon name={isPlaced ? 'check' : 'plus'} size={12} />
                      <span>{isPlaced ? 'Added to Dashboard' : '+ Add to Dashboard'}</span>
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {showPupils && (
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <MonochromeLucideIcon name="user" size={16} />
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>
                        {pupil.first_name} {pupil.last_name}
                      </div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        UPN {pupil.upn} • Year {pupil.nc_year_group || 'N/A'}
                      </div>
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                    <span>Open Student</span>
                    <MonochromeLucideIcon name="arrow-right" size={12} />
                  </span>
                </button>
              ))}
            </div>
          )}

          {showGroups && (
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <MonochromeLucideIcon name="users" size={16} />
                    <div>
                      <div style={{ fontSize: '13px', fontWeight: 700 }}>
                        {group.code} — {group.name}
                      </div>
                      <div style={{ fontSize: '11px', color: 'var(--mis-text-secondary)' }}>
                        {group.group_type}
                      </div>
                    </div>
                  </div>
                  <span style={{ fontSize: '11px', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                    <span>Open Group</span>
                    <MonochromeLucideIcon name="arrow-right" size={12} />
                  </span>
                </button>
              ))}
            </div>
          )}

          {!showViews && !showWidgets && !showPupils && !showGroups && (
            <div
              data-testid="command-palette-empty-state"
              style={{ fontSize: '12px', color: 'var(--mis-text-secondary)', padding: '6px 0' }}
            >
              No matching items in this category.
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
