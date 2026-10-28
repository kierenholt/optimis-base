import React, { useCallback, useEffect, useRef, useState } from 'react';
import { GdprSarPanel } from './GdprSarPanel';
import { SafeguardingVaultPanel } from './SafeguardingVaultPanel';
import {
  applyHostThemeToRoot,
  DecryptedSafeguardingNote,
  DEFAULT_PERMITTED_METHODS,
  DEFAULT_THEME_TOKENS,
  PLUGIN_ID,
  PluginNavEntry,
  SafeguardingNoteEnvelope,
  SARSection,
  ThemeMode,
} from './themeAndTypes';

export function App() {
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const params = new URLSearchParams(window.location.search);
  const initialPupilId = params.get('pupil_id') || '';
  const initialSecondPupilId = params.get('second_pupil_id') || '';
  const initialGroupId = params.get('group_id') || '';
  const initialOtherGroupId = params.get('other_group_id') || '';
  const initialReferenceDate = params.get('reference_date') || '';
  const initialAcademicYearId = params.get('academic_year_id') || '';
  const initialStaffId = params.get('staff_id') || '';

  // Host Theme & School Branding state (Sections 9.4 & 9.5)
  const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
    applyHostThemeToRoot('light', DEFAULT_THEME_TOKENS);
    return 'light';
  });
  const [themeCatalog, setThemeCatalog] = useState<Record<ThemeMode, Record<string, string>>>(DEFAULT_THEME_TOKENS);
  const [schoolName, setSchoolName] = useState<string>('Oakridge All-Through Academy');
  const [schoolURN, setSchoolURN] = useState<string>('100001');
  const [configuredAcademicYear, setConfiguredAcademicYear] = useState<string>(
    initialAcademicYearId || 'ay_2026_2027'
  );
  const [entityLinkHandlers, setEntityLinkHandlers] = useState<Record<string, string>>({
    student: PLUGIN_ID,
    group: PLUGIN_ID,
    staff: PLUGIN_ID,
  });
  const [installedPlugins, setInstalledPlugins] = useState<PluginNavEntry[]>([
    {
      id: PLUGIN_ID,
      title: 'Pupil & Classroom Workspace',
      version: '0.2.0',
      entry_url: `/plugins/${PLUGIN_ID}/ui/index.html`,
      permitted_methods: DEFAULT_PERMITTED_METHODS,
    },
  ]);

  // Active Workspace & Entity Deep-Linking Context state (Sections 9A & 9C)
  const [activePluginId, setActivePluginId] = useState<string>(PLUGIN_ID);
  const [activePupilId, setActivePupilId] = useState<string>(initialPupilId);
  const [activeGroupId, setActiveGroupId] = useState<string>(initialGroupId);
  const [activeStaffId, setActiveStaffId] = useState<string>(initialStaffId);
  const [activeEntityContext, setActiveEntityContext] = useState<string>(
    initialPupilId ? `student:${initialPupilId}` : initialGroupId ? `group:${initialGroupId}` : 'dashboard:home'
  );
  const [deepLinkNavigationStatus, setDeepLinkNavigationStatus] = useState<string>('IDLE');

  // DSL Safeguarding Vault state (Section 5B)
  const [hasSafeguardingRead, setHasSafeguardingRead] = useState<boolean>(false);
  const [safeguardingEnvelopes, setSafeguardingEnvelopes] = useState<SafeguardingNoteEnvelope[]>([]);
  const [cloudKeyToken, setCloudKeyToken] = useState<string>('');
  const [decryptedNotes, setDecryptedNotes] = useState<Record<string, DecryptedSafeguardingNote>>({});
  const [dslVaultError, setDslVaultError] = useState<string>('');

  // UK GDPR SAR Redaction & Retention Purge state (Sections 5C, 7B & 9.3)
  const [sarSections, setSarSections] = useState<SARSection[]>([]);
  const [sarExportStatus, setSarExportStatus] = useState<string>('');
  const [sarFinalizedJson, setSarFinalizedJson] = useState<string>('');
  const [sarPurgeStatus, setSarPurgeStatus] = useState<string>('');

  useEffect(() => {
    const loadUIConfig = async () => {
      try {
        const query = activeStaffId ? `?staff_id=${encodeURIComponent(activeStaffId)}` : '';
        const res = await fetch(`/api/system/ui-config${query}`);
        if (!res.ok) {
          return;
        }
        const body = await res.json();
        if (!body || !body.ok) {
          return;
        }
        if (body.school_name) {
          setSchoolName(body.school_name);
        }
        if (body.school_urn) {
          setSchoolURN(body.school_urn);
        }
        if (body.academic_year_id && !initialAcademicYearId) {
          setConfiguredAcademicYear(body.academic_year_id);
        }
        if (body.themes && body.themes.light && body.themes['high-contrast']) {
          const merged: Record<ThemeMode, Record<string, string>> = {
            light: { ...DEFAULT_THEME_TOKENS.light, ...body.themes.light },
            'high-contrast': { ...DEFAULT_THEME_TOKENS['high-contrast'], ...body.themes['high-contrast'] },
          };
          setThemeCatalog(merged);
        }
        if (body.entity_link_handlers && typeof body.entity_link_handlers === 'object') {
          setEntityLinkHandlers((prev) => ({ ...prev, ...body.entity_link_handlers }));
        }
        if (Array.isArray(body.plugins)) {
          setInstalledPlugins(body.plugins);
        }
      } catch {
        // Keep default tokens if ui-config endpoint is unreachable
      }
    };
    void loadUIConfig();
  }, [initialAcademicYearId, activeStaffId]);

  useEffect(() => {
    applyHostThemeToRoot(themeMode, themeCatalog);
  }, [themeMode, themeCatalog]);

  const sendInitToIframe = useCallback(
    (overrideContext?: Partial<{ pupil_id: string; group_id: string; staff_id: string; entity_type: string; entity_id: string }>, overrideThemeMode?: ThemeMode) => {
      const contentWin = iframeRef.current?.contentWindow;
      if (!contentWin) {
        return;
      }
      const mode = overrideThemeMode || themeMode;
      const activeTokens = themeCatalog[mode] || DEFAULT_THEME_TOKENS[mode];
      const resolvedPupilId = overrideContext?.pupil_id !== undefined ? overrideContext.pupil_id : activePupilId;
      const resolvedGroupId = overrideContext?.group_id !== undefined ? overrideContext.group_id : activeGroupId;
      const resolvedStaffId = overrideContext?.staff_id !== undefined ? overrideContext.staff_id : activeStaffId;
      const activePluginDescriptor = installedPlugins.find((p) => p.id === activePluginId);
      const permittedMethods = activePluginDescriptor
        ? activePluginDescriptor.permitted_methods
        : DEFAULT_PERMITTED_METHODS;

      contentWin.postMessage(
        {
          type: 'mis:init',
          theme_mode: mode,
          theme_tokens: activeTokens,
          theme: {
            primary: activeTokens['--mis-brand-primary'],
          },
          context: {
            pupil_id: resolvedPupilId || null,
            second_pupil_id: initialSecondPupilId || null,
            group_id: resolvedGroupId || null,
            other_group_id: initialOtherGroupId || null,
            reference_date: initialReferenceDate || null,
            academic_year_id: initialAcademicYearId || configuredAcademicYear,
            staff_id: resolvedStaffId || null,
            entity_type: overrideContext?.entity_type || (resolvedPupilId ? 'student' : resolvedGroupId ? 'group' : 'dashboard'),
            entity_id: overrideContext?.entity_id || resolvedPupilId || resolvedGroupId || 'home',
          },
          permitted_methods: permittedMethods,
        },
        '*'
      );
    },
    [
      themeMode,
      themeCatalog,
      activePupilId,
      initialSecondPupilId,
      activeGroupId,
      initialOtherGroupId,
      initialReferenceDate,
      initialAcademicYearId,
      configuredAcademicYear,
      activeStaffId,
      installedPlugins,
      activePluginId,
    ]
  );

  useEffect(() => {
    sendInitToIframe();
  }, [sendInitToIframe]);

  useEffect(() => {
    const loadSessionAndSafeguarding = async () => {
      try {
        const sessRes = await fetch('/api/auth/session');
        if (!sessRes.ok) {
          return;
        }
        const sessBody = await sessRes.json();
        const scopes: string[] = sessBody?.caller?.scopes || [];
        if (scopes.includes('dal.safeguarding.read')) {
          setHasSafeguardingRead(true);
          const query = activePupilId ? `?pupil_id=${encodeURIComponent(activePupilId)}` : '';
          const notesRes = await fetch(`/api/safeguarding/notes${query}`);
          if (notesRes.ok) {
            const notesBody = await notesRes.json();
            if (notesBody.ok && Array.isArray(notesBody.notes)) {
              setSafeguardingEnvelopes(notesBody.notes);
            }
          }
        } else {
          setHasSafeguardingRead(false);
        }
      } catch {
        // Ignore session fetch errors if not in DSL flow
      }
    };
    void loadSessionAndSafeguarding();
  }, [activePupilId]);

  useEffect(() => {
    const handleMessage = async (event: MessageEvent) => {
      const msg = event.data;
      if (!msg || typeof msg !== 'object') {
        return;
      }

      if (msg.type === 'mis:ready') {
        sendInitToIframe();
        return;
      }

      if (msg.type === 'mis:link:navigate') {
        const entityType = typeof msg.entity_type === 'string' ? msg.entity_type.trim().toLowerCase() : '';
        const entityId = typeof msg.entity_id === 'string' ? msg.entity_id.trim() : '';
        if (!entityType || !entityId) {
          throw new Error('Malformed mis:link:navigate payload: missing entity_type or entity_id');
        }

        const designatedPlugin = entityLinkHandlers[entityType] || PLUGIN_ID;
        setActivePluginId(designatedPlugin);

        let nextPupilId = activePupilId;
        let nextGroupId = activeGroupId;
        let nextStaffId = activeStaffId;

        if (entityType === 'student' || (typeof msg.pupil_id === 'string' && msg.pupil_id.trim() !== '')) {
          nextPupilId = (typeof msg.pupil_id === 'string' && msg.pupil_id.trim()) || entityId;
          setActivePupilId(nextPupilId);
        }
        if (entityType === 'group' || (typeof msg.group_id === 'string' && msg.group_id.trim() !== '')) {
          nextGroupId = (typeof msg.group_id === 'string' && msg.group_id.trim()) || entityId;
          setActiveGroupId(nextGroupId);
        }
        if (entityType === 'staff' || (typeof msg.staff_id === 'string' && msg.staff_id.trim() !== '')) {
          nextStaffId = (typeof msg.staff_id === 'string' && msg.staff_id.trim()) || entityId;
          setActiveStaffId(nextStaffId);
        }

        setActiveEntityContext(`${entityType}:${entityId}`);
        setDeepLinkNavigationStatus(`${entityType}:${entityId} -> ${designatedPlugin}`);

        sendInitToIframe({
          pupil_id: nextPupilId,
          group_id: nextGroupId,
          staff_id: nextStaffId,
          entity_type: entityType,
          entity_id: entityId,
        });
        return;
      }

      if (msg.type === 'mis:rpc:request') {
        const { request_id, method, args } = msg;
        if (!request_id || !method) {
          throw new Error('Malformed mis:rpc:request: missing request_id or method');
        }

        const res = await fetch(`/api/plugins/${encodeURIComponent(activePluginId)}/rpc/${encodeURIComponent(method)}`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ args: args ?? {} }),
        });

        const payload = await res.json();
        iframeRef.current?.contentWindow?.postMessage(
          {
            type: 'mis:rpc:response',
            request_id,
            ok: Boolean(payload.ok),
            data: payload.data,
            error_code: payload.error_code,
          },
          '*'
        );
      }
    };

    window.addEventListener('message', handleMessage);
    return () => {
      window.removeEventListener('message', handleMessage);
    };
  }, [sendInitToIframe, entityLinkHandlers, activePluginId, activePupilId, activeGroupId, activeStaffId]);

  const handleToggleThemeMode = () => {
    const nextMode: ThemeMode = themeMode === 'light' ? 'high-contrast' : 'light';
    applyHostThemeToRoot(nextMode, themeCatalog);
    setThemeMode(nextMode);
    sendInitToIframe(undefined, nextMode);
  };

  const handleRenewCloudKey = async () => {
    setDslVaultError('');
    const res = await fetch('/api/safeguarding/cloud-key/renew', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ttl_seconds: 3600 }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to renew DSL Cloud Key');
      return;
    }
    setCloudKeyToken(body.cloud_key_token);
  };

  const handleDecryptSafeguardingNote = async (noteId: string) => {
    setDslVaultError('');
    const res = await fetch(`/api/safeguarding/notes/${encodeURIComponent(noteId)}/decrypt`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ cloud_key_token: cloudKeyToken }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setDslVaultError(body.error_code || 'Failed to decrypt safeguarding note');
      return;
    }
    setDecryptedNotes((prev) => ({
      ...prev,
      [noteId]: body.note,
    }));
  };

  const handleLoadSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR export');
    }
    const res = await fetch('/api/admin/gdpr/sar/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pupil_id: activePupilId, staff_id: activeStaffId || undefined }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarSections(body.bundle.sections || []);
    setSarExportStatus('SAR_LOADED');
    setSarFinalizedJson('');
  };

  const handleUpdateRedactableField = (
    secIdx: number,
    recIdx: number,
    fieldKey: string,
    newValue: string
  ) => {
    setSarSections((prev) =>
      prev.map((sec, sIdx) => {
        if (sIdx !== secIdx) {
          return sec;
        }
        return {
          ...sec,
          records: sec.records.map((rec, rIdx) => {
            if (rIdx !== recIdx) {
              return rec;
            }
            return {
              ...rec,
              [fieldKey]: newValue,
            };
          }),
        };
      })
    );
  };

  const handleFinalizeSarExport = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for SAR finalization');
    }
    const res = await fetch('/api/admin/gdpr/sar/finalize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
        staff_id: activeStaffId || undefined,
        sections: sarSections,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarExportStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarExportStatus(body.package.status || 'SAR_EXPORT_FINALIZED');
    setSarFinalizedJson(JSON.stringify(body.package, null, 2));
  };

  const handlePurgePupilData = async () => {
    if (!activePupilId) {
      throw new Error('Missing mandatory pupil_id for retention purge');
    }
    const res = await fetch('/api/admin/gdpr/purge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: activePupilId,
        staff_id: activeStaffId || undefined,
      }),
    });
    const body = await res.json();
    if (!res.ok || !body.ok) {
      setSarPurgeStatus(`ERROR:${body.error_code || res.status}`);
      return;
    }
    setSarPurgeStatus(body.result.status || 'RETENTION_PURGE_COMPLETED');
    setSarSections([]);
  };

  return (
    <div
      data-testid="host-shell"
      data-mis-theme={themeMode}
      style={{
        display: 'grid',
        gridTemplateColumns: '260px 1fr',
        minHeight: '100vh',
        fontFamily: 'var(--mis-font-sans, system-ui, sans-serif)',
        backgroundColor: 'var(--mis-bg-canvas)',
        color: 'var(--mis-text-primary)',
      }}
    >
      {/* Left Navigation Sidebar (Section 9.5) */}
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
              onClick={() => setActivePluginId(plug.id)}
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

      {/* Main Content Area: Topbar + Default Dashboard Grid */}
      <div style={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
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
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
              Theme:{' '}
              <strong data-testid="active-theme-mode-indicator">{themeMode}</strong>
            </span>
            <button
              type="button"
              data-testid="theme-mode-toggle-btn"
              onClick={handleToggleThemeMode}
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

        <main
          data-testid="host-dashboard-grid"
          style={{
            padding: '20px 24px',
            display: 'flex',
            flexDirection: 'column',
            gap: '20px',
          }}
        >
          {/* Default Dashboard Statutory Summary Strip (Section 9.5) */}
          <section
            data-testid="dashboard-summary-strip"
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
              gap: '12px',
            }}
          >
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Statutory Roll Call (AM / PM)
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                AM 08:45 • PM 13:15 Ready
              </div>
            </div>
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Entity Link Handler (student / group)
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                {entityLinkHandlers.student}
              </div>
            </div>
            <div
              style={{
                backgroundColor: 'var(--mis-bg-surface)',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                padding: '12px 16px',
                boxShadow: 'var(--mis-shadow-card)',
              }}
            >
              <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)' }}>
                Sandboxed Plugin CSP
              </div>
              <div style={{ fontSize: '14px', fontWeight: 700, marginTop: '4px' }}>
                connect-src &apos;none&apos; (Enforced)
              </div>
            </div>
          </section>

          {/* Active Keep-Alive Plugin Workspace Panel */}
          <section
            data-testid="dashboard-workspace-panel"
            style={{
              backgroundColor: 'var(--mis-bg-surface)',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              padding: '16px',
              boxShadow: 'var(--mis-shadow-card)',
            }}
          >
            <iframe
              ref={iframeRef}
              data-testid={`plugin-iframe-${PLUGIN_ID}`}
              sandbox="allow-scripts"
              src={`/plugins/${PLUGIN_ID}/ui/index.html`}
              title="Test Plugin Micro-Frontend"
              style={{
                width: '100%',
                height: '320px',
                border: '1px solid var(--mis-border-subtle)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                backgroundColor: 'var(--mis-bg-surface)',
              }}
            />
          </section>

          {hasSafeguardingRead && (
            <SafeguardingVaultPanel
              safeguardingEnvelopes={safeguardingEnvelopes}
              cloudKeyToken={cloudKeyToken}
              onCloudKeyTokenChange={setCloudKeyToken}
              onRenewCloudKey={handleRenewCloudKey}
              onDecryptNote={handleDecryptSafeguardingNote}
              decryptedNotes={decryptedNotes}
              dslVaultError={dslVaultError}
            />
          )}

          <GdprSarPanel
            sarSections={sarSections}
            sarExportStatus={sarExportStatus}
            sarFinalizedJson={sarFinalizedJson}
            sarPurgeStatus={sarPurgeStatus}
            onLoadSarExport={handleLoadSarExport}
            onFinalizeSarExport={handleFinalizeSarExport}
            onPurgePupilData={handlePurgePupilData}
            onUpdateRedactableField={handleUpdateRedactableField}
          />
        </main>
      </div>
    </div>
  );
}
