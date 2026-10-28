import React, { useEffect, useRef, useState } from 'react';

const PLUGIN_ID = 'uk.org.optimis.test';

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

export function App() {
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const params = new URLSearchParams(window.location.search);
  const pupilId = params.get('pupil_id');
  const groupId = params.get('group_id');
  const otherGroupId = params.get('other_group_id');
  const referenceDate = params.get('reference_date');
  const academicYearId = params.get('academic_year_id');
  const staffId = params.get('staff_id');

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
          const query = pupilId ? `?pupil_id=${encodeURIComponent(pupilId)}` : '';
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
  }, [pupilId]);

  useEffect(() => {
    const sendInit = () => {
      const contentWin = iframeRef.current?.contentWindow;
      if (!contentWin || !pupilId) {
        return;
      }
      contentWin.postMessage(
        {
          type: 'mis:init',
          theme: { primary: '#2563eb' },
          context: {
            pupil_id: pupilId,
            group_id: groupId,
            other_group_id: otherGroupId,
            reference_date: referenceDate,
            academic_year_id: academicYearId,
            staff_id: staffId,
          },
          permitted_methods: [
            'get_pupil_record',
            'save_pupil_note',
            'list_pupil_notes',
            'get_group_roster_summary',
            'get_pupil_guardians',
            'record_group_am_attendance',
            'submit_safeguarding_postbox',
          ],
        },
        '*'
      );
    };

    const handleMessage = async (event: MessageEvent) => {
      const msg = event.data;
      if (!msg || typeof msg !== 'object') {
        return;
      }

      if (msg.type === 'mis:ready') {
        sendInit();
        return;
      }

      if (msg.type === 'mis:rpc:request') {
        const { request_id, method, args } = msg;
        if (!request_id || !method) {
          throw new Error('Malformed mis:rpc:request: missing request_id or method');
        }

        const res = await fetch(`/api/plugins/${encodeURIComponent(PLUGIN_ID)}/rpc/${encodeURIComponent(method)}`, {
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
  }, [pupilId, groupId, otherGroupId, referenceDate, academicYearId, staffId]);

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
    if (!pupilId) {
      throw new Error('Missing mandatory pupil_id for SAR export');
    }
    const res = await fetch('/api/admin/gdpr/sar/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pupil_id: pupilId, staff_id: staffId ?? undefined }),
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
    if (!pupilId) {
      throw new Error('Missing mandatory pupil_id for SAR finalization');
    }
    const res = await fetch('/api/admin/gdpr/sar/finalize', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: pupilId,
        staff_id: staffId ?? undefined,
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
    if (!pupilId) {
      throw new Error('Missing mandatory pupil_id for retention purge');
    }
    const res = await fetch('/api/admin/gdpr/purge', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pupil_id: pupilId,
        staff_id: staffId ?? undefined,
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
    <div data-testid="host-shell" style={{ fontFamily: 'system-ui, sans-serif', padding: '24px' }}>
      <header style={{ marginBottom: '16px' }}>
        <h1 style={{ fontSize: '20px', margin: 0 }}>Optimis School MIS — Host Shell</h1>
      </header>
      <main>
        <iframe
          ref={iframeRef}
          data-testid={`plugin-iframe-${PLUGIN_ID}`}
          sandbox="allow-scripts"
          src={`/plugins/${PLUGIN_ID}/ui/index.html`}
          title="Test Plugin Micro-Frontend"
          style={{ width: '100%', height: '240px', border: '1px solid #d1d5db', borderRadius: '8px' }}
        />

        {hasSafeguardingRead && (
          <section
            data-testid="dsl-safeguarding-vault-panel"
            style={{
              marginTop: '24px',
              padding: '16px',
              border: '1px solid #fca5a5',
              borderRadius: '8px',
              backgroundColor: '#fef2f2',
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
                    backgroundColor: '#ffffff',
                    border: '1px solid #fecaca',
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
                        backgroundColor: '#fff1f2',
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
            marginTop: '24px',
            padding: '16px',
            border: '1px solid #cbd5e1',
            borderRadius: '8px',
            backgroundColor: '#f8fafc',
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
                style={{ padding: '6px 12px', cursor: 'pointer', backgroundColor: '#dc2626', color: '#ffffff', border: 'none', borderRadius: '4px' }}
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
            <div data-testid="sar-purge-status" style={{ marginBottom: '8px', fontWeight: 600, color: '#b91c1c' }}>
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
                  backgroundColor: '#ffffff',
                  border: '1px solid #e2e8f0',
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
                      border: '1px solid #f1f5f9',
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
  );
}
