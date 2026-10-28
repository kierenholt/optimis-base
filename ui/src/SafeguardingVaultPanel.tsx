import React from 'react';
import { DecryptedSafeguardingNote, SafeguardingNoteEnvelope } from './themeAndTypes';

interface SafeguardingVaultPanelProps {
  safeguardingEnvelopes: SafeguardingNoteEnvelope[];
  cloudKeyToken: string;
  onCloudKeyTokenChange: (val: string) => void;
  onRenewCloudKey: () => void;
  onDecryptNote: (noteId: string) => void;
  decryptedNotes: Record<string, DecryptedSafeguardingNote>;
  dslVaultError: string;
}

export function SafeguardingVaultPanel({
  safeguardingEnvelopes,
  cloudKeyToken,
  onCloudKeyTokenChange,
  onRenewCloudKey,
  onDecryptNote,
  decryptedNotes,
  dslVaultError,
}: SafeguardingVaultPanelProps) {
  return (
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
            onChange={(e) => onCloudKeyTokenChange(e.target.value)}
            style={{ width: '280px', padding: '4px 8px' }}
          />
          <button
            type="button"
            data-testid="renew-dsl-cloud-key-btn"
            onClick={onRenewCloudKey}
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
                onClick={() => onDecryptNote(env.note_id)}
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
  );
}
