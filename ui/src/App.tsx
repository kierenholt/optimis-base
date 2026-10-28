import React, { useCallback, useEffect, useRef, useState } from 'react';

const PLUGIN_ID = 'uk.org.optimis.test';

type ThemeMode = 'light' | 'high-contrast';

const DEFAULT_THEME_TOKENS: Record<ThemeMode, Record<string, string>> = {
  light: {
    '--mis-brand-primary': '#1d4ed8',
    '--mis-brand-primary-hover': '#1e40af',
    '--mis-brand-primary-fg': '#ffffff',
    '--mis-brand-accent': '#0f766e',
    '--mis-bg-canvas': '#f1f5f9',
    '--mis-bg-surface': '#ffffff',
    '--mis-bg-elevated': '#f8fafc',
    '--mis-bg-sidebar': '#0f172a',
    '--mis-text-sidebar': '#f8fafc',
    '--mis-text-primary': '#0f172a',
    '--mis-text-secondary': '#475569',
    '--mis-text-muted': '#64748b',
    '--mis-text-inverse': '#ffffff',
    '--mis-border-subtle': '#e2e8f0',
    '--mis-border-strong': '#cbd5e1',
    '--mis-focus-ring': '0 0 0 3px rgba(29, 78, 216, 0.35)',
    '--mis-shadow-card': '0 1px 3px 0 rgba(15, 23, 42, 0.08), 0 1px 2px -1px rgba(15, 23, 42, 0.06)',
    '--mis-status-success': '#15803d',
    '--mis-status-warning': '#b45309',
    '--mis-status-danger': '#b91c1c',
    '--mis-status-danger-bg': '#fef2f2',
    '--mis-font-sans': "'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    '--mis-radius-sm': '4px',
    '--mis-radius-md': '8px',
    '--mis-radius-lg': '12px',
  },
  'high-contrast': {
    '--mis-brand-primary': '#facc15',
    '--mis-brand-primary-hover': '#fde047',
    '--mis-brand-primary-fg': '#000000',
    '--mis-brand-accent': '#22d3ee',
    '--mis-bg-canvas': '#000000',
    '--mis-bg-surface': '#0a0a0a',
    '--mis-bg-elevated': '#171717',
    '--mis-bg-sidebar': '#000000',
    '--mis-text-sidebar': '#ffffff',
    '--mis-text-primary': '#ffffff',
    '--mis-text-secondary': '#fef08a',
    '--mis-text-muted': '#e2e8f0',
    '--mis-text-inverse': '#000000',
    '--mis-border-subtle': '#ffffff',
    '--mis-border-strong': '#facc15',
    '--mis-focus-ring': '0 0 0 3px #facc15',
    '--mis-shadow-card': 'none',
    '--mis-status-success': '#4ade80',
    '--mis-status-warning': '#facc15',
    '--mis-status-danger': '#ff6b6b',
    '--mis-status-danger-bg': '#2a0000',
    '--mis-font-sans': "'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    '--mis-radius-sm': '4px',
    '--mis-radius-md': '8px',
    '--mis-radius-lg': '12px',
  },
};

const DEFAULT_PERMITTED_METHODS = [
  'get_pupil_record',
  'save_pupil_note',
  'list_pupil_notes',
  'get_group_roster_summary',
  'get_pupil_guardians',
  'record_group_am_attendance',
  'submit_safeguarding_postbox',
  'sync_external_sparx_get',
  'push_external_sparx_post',
  'mutate_then_illegal_egress',
  'trigger_loop_relay',
  'list_worker_event_logs',
];

function applyHostThemeToRoot(mode: ThemeMode, tokensMap: Record<ThemeMode, Record<string, string>>) {
  if (typeof document === 'undefined' || !document.documentElement) {
    return;
  }
  const root = document.documentElement;
  root.setAttribute('data-mis-theme', mode);
  const tokens = tokensMap[mode] || DEFAULT_THEME_TOKENS[mode];
  for (const [k, v] of Object.entries(tokens)) {
    root.style.setProperty(k, v);
  }
}

interface SARSection {
  source_id: string;
  source_type: string;
  plugin_state?: string;
  title: string;
  records: Array<Record<string, unknown>>;
  redactable_fields: string[];
}

interface SafeguardingNoteEnvelope {
  note_id: string;
  pupil_id: string;
  reported_by_staff_id: string;
  assigned_dsl_staff_id: string;
  concern_category: string;
  status: string;
  incident_date: string;
  kms_key_id: string;
  created_at: string;
}

interface SafeguardingAuditEntry {
  audit_id: string;
  action_type: string;
  actor_person_id: string;
  request_id: string;
  timestamp: string;
}

interface DecryptedSafeguardingNote {
  note_id: string;
  pupil_id: string;
  concern_category: string;
  status: string;
  incident_date: string;
  summary: string;
  body_map: string;
  verbatim_account: string;
  action_taken: string;
  audit_logs?: SafeguardingAuditEntry[];
}

interface PluginNavEntry {
  id: string;
  title: string;
  version: string;
  entry_url: string;
  permitted_methods: string[];
}

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
        const res = await fetch('/api/system/ui-config');
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
        if (Array.isArray(body.plugins) && body.plugins.length > 0) {
          setInstalledPlugins(body.plugins);
        }
      } catch {
        // Keep default tokens if ui-config endpoint is unreachable
      }
    };
    void loadUIConfig();
  }, [initialAcademicYearId]);

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
          permitted_methods: DEFAULT_PERMITTED_METHODS,
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
            <section
              data-testid="dsl-safeguarding-vault-panel"
              style={{
                padding: '16px',
                border: '1px solid var(--mis-border-strong)',
                borderRadius: 'var(--mis-radius-md, 8px)',
                backgroundColor: 'var(--mis-status-danger-bg)',
              }}
            >
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
                <h2 style={{ fontSize: '16px', margin: 0 }}>
                  Designated Safeguarding Lead (DSL) Envelope-Encrypted Vault
                </h2>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
                  <input
                    type="text"
                    data-testid="dsl-cloud-key-input"
                    placeholder="Signed DSL OAuth Cloud Key Token"
                    value={cloudKeyToken}
                    onChange={(e) => setCloudKeyToken(e.target.value)}
                    style={{ width: '280px', padding: '4px 8px' }}
                  />
                  <button
                    type="button"
                    data-testid="renew-dsl-cloud-key-btn"
                    onClick={handleRenewCloudKey}
                    style={{ padding: '6px 12px', cursor: 'pointer' }}
                  >
                    Renew OAuth Cloud Key
                  </button>
                </div>
              </div>

              {dslVaultError && (
                <div
                  data-testid="dsl-vault-error-banner"
                  style={{
                    padding: '8px 12px',
                    marginBottom: '12px',
                    backgroundColor: '#fee2e2',
                    color: '#991b1b',
                    borderRadius: '4px',
                    fontWeight: 600,
                  }}
                >
                  {dslVaultError}
                </div>
              )}

              {safeguardingEnvelopes.map((env) => {
                const dec = decryptedNotes[env.note_id];
                return (
                  <div
                    key={env.note_id}
                    data-testid={`safeguarding-note-card-${env.note_id}`}
                    style={{
                      padding: '12px',
                      marginBottom: '8px',
                      backgroundColor: 'var(--mis-bg-surface)',
                      border: '1px solid var(--mis-border-subtle)',
                      borderRadius: '6px',
                    }}
                  >
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <strong>{env.note_id}</strong> — Category: <span>{env.concern_category}</span> — Status:{' '}
                        <span>{env.status}</span> — Incident Date: {env.incident_date}
                      </div>
                      <button
                        type="button"
                        data-testid={`decrypt-safeguarding-note-btn-${env.note_id}`}
                        onClick={() => handleDecryptSafeguardingNote(env.note_id)}
                        style={{ padding: '4px 10px', cursor: 'pointer' }}
                      >
                        Unlock &amp; Decrypt Note
                      </button>
                    </div>

                    {dec && (
                      <div
                        data-testid={`decrypted-safeguarding-note-${env.note_id}`}
                        style={{
                          marginTop: '10px',
                          padding: '10px',
                          backgroundColor: 'var(--mis-bg-elevated)',
                          borderRadius: '4px',
                        }}
                      >
                        <div>
                          <strong>Summary:</strong> {dec.summary}
                        </div>
                        <div>
                          <strong>Body Map:</strong> {dec.body_map}
                        </div>
                        <div>
                          <strong>Verbatim Account:</strong> {dec.verbatim_account}
                        </div>
                        <div>
                          <strong>Action Taken:</strong> {dec.action_taken}
                        </div>
                        {dec.audit_logs && dec.audit_logs.length > 0 && (
                          <div style={{ marginTop: '6px', fontSize: '12px' }}>
                            <strong>Audit Trail:</strong>{' '}
                            {dec.audit_logs.map((a) => a.action_type).join(' -> ')}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </section>
          )}

          <section
            data-testid="sar-redaction-panel"
            style={{
              padding: '16px',
              border: '1px solid var(--mis-border-subtle)',
              borderRadius: 'var(--mis-radius-md, 8px)',
              backgroundColor: 'var(--mis-bg-surface)',
            }}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
              <h2 style={{ fontSize: '16px', margin: 0 }}>
                UK GDPR Subject Access Request (SAR) Redaction &amp; Statutory Retention Purge
              </h2>
              <div style={{ display: 'flex', gap: '8px' }}>
                <button
                  type="button"
                  data-testid="sar-load-export-btn"
                  onClick={handleLoadSarExport}
                  style={{ padding: '6px 12px', cursor: 'pointer' }}
                >
                  Load SAR Package
                </button>
                <button
                  type="button"
                  data-testid="sar-finalize-export-btn"
                  onClick={handleFinalizeSarExport}
                  disabled={sarSections.length === 0}
                  style={{ padding: '6px 12px', cursor: 'pointer' }}
                >
                  Finalize &amp; Export Redacted SAR
                </button>
                <button
                  type="button"
                  data-testid="sar-purge-pupil-btn"
                  onClick={handlePurgePupilData}
                  style={{
                    padding: '6px 12px',
                    cursor: 'pointer',
                    backgroundColor: '#dc2626',
                    color: '#ffffff',
                    border: 'none',
                    borderRadius: '4px',
                  }}
                >
                  Execute Retention Purge
                </button>
              </div>
            </div>

            {sarExportStatus && (
              <div data-testid="sar-export-status" style={{ marginBottom: '8px', fontWeight: 600 }}>
                {sarExportStatus}
              </div>
            )}

            {sarPurgeStatus && (
              <div data-testid="sar-purge-status" style={{ marginBottom: '8px', fontWeight: 600, color: 'var(--mis-status-danger)' }}>
                {sarPurgeStatus}
              </div>
            )}

            {sarSections.map((sec, secIdx) => {
              const sectionKey = `${sec.source_id}-${sec.title}`;
              const redactableSet = new Set(sec.redactable_fields || []);
              return (
                <div
                  key={sectionKey}
                  data-testid={`sar-section-${sec.source_id}-${sec.title}`}
                  style={{
                    marginBottom: '12px',
                    padding: '12px',
                    backgroundColor: 'var(--mis-bg-elevated)',
                    border: '1px solid var(--mis-border-subtle)',
                    borderRadius: '6px',
                  }}
                >
                  <div style={{ fontWeight: 600, marginBottom: '8px' }}>
                    [{sec.source_type}: {sec.source_id}] {sec.title}
                  </div>
                  {sec.records.map((rec, recIdx) => (
                    <div
                      key={recIdx}
                      style={{
                        padding: '8px',
                        marginBottom: '6px',
                        border: '1px solid var(--mis-border-subtle)',
                        borderRadius: '4px',
                      }}
                    >
                      {Object.entries(rec).map(([fieldKey, val]) => {
                        const isRedactable = redactableSet.has(fieldKey);
                        const strVal = val === null || val === undefined ? '' : String(val);
                        if (isRedactable) {
                          return (
                            <div key={fieldKey} style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                              <label style={{ minWidth: '150px', fontSize: '13px', fontWeight: 600 }}>
                                {fieldKey} (Redactable):
                              </label>
                              <input
                                type="text"
                                data-testid={`sar-redact-input-${sec.source_id}-${sec.title}-${recIdx}-${fieldKey}`}
                                value={strVal}
                                onChange={(e) =>
                                  handleUpdateRedactableField(secIdx, recIdx, fieldKey, e.target.value)
                                }
                                style={{ flex: 1, padding: '4px 8px' }}
                              />
                              <button
                                type="button"
                                data-testid={`sar-redact-mask-btn-${sec.source_id}-${sec.title}-${recIdx}-${fieldKey}`}
                                onClick={() =>
                                  handleUpdateRedactableField(secIdx, recIdx, fieldKey, '[REDACTED]')
                                }
                                style={{ padding: '4px 8px', fontSize: '12px', cursor: 'pointer' }}
                              >
                                [REDACTED]
                              </button>
                            </div>
                          );
                        }
                        return (
                          <div key={fieldKey} style={{ fontSize: '13px', marginBottom: '2px' }}>
                            <strong>{fieldKey}:</strong> {strVal}
                          </div>
                        );
                      })}
                    </div>
                  ))}
                </div>
              );
            })}

            {sarFinalizedJson && (
              <pre
                data-testid="sar-finalized-json"
                style={{
                  marginTop: '12px',
                  padding: '12px',
                  backgroundColor: '#0f172a',
                  color: '#f8fafc',
                  borderRadius: '6px',
                  overflowX: 'auto',
                  fontSize: '12px',
                }}
              >
                {sarFinalizedJson}
              </pre>
            )}
          </section>
        </main>
      </div>
    </div>
  );
}
