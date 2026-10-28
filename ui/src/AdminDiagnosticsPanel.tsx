import React from 'react';

interface AdminDiagnosticsPanelProps {
  diagnosticBundleStatus: string;
  diagnosticBundleJson: string;
  diagnosticBundleLog: string;
  onDownloadDiagnosticBundle: () => void;
}

export function AdminDiagnosticsPanel({
  diagnosticBundleStatus,
  diagnosticBundleJson,
  diagnosticBundleLog,
  onDownloadDiagnosticBundle,
}: AdminDiagnosticsPanelProps) {
  return (
    <section
      data-testid="admin-diagnostics-panel"
      style={{
        padding: '16px',
        border: '1px solid var(--mis-border-subtle)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        backgroundColor: 'var(--mis-bg-surface)',
      }}
    >
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          gap: '12px',
          flexWrap: 'wrap',
          marginBottom: '12px',
        }}
      >
        <div>
          <h2 style={{ fontSize: '16px', margin: 0 }}>
            Day-2 Operations — Admin-Exported PII-Scrubbed Plugin Diagnostic Bundle
          </h2>
          <div style={{ fontSize: '12px', color: 'var(--mis-text-secondary)', marginTop: '4px' }}>
            Zero outbound telemetry. Exports sanitized Lua stack traces, error classes, and execution/SQLite lock durations with zero pupil IDs or local variables.
          </div>
        </div>
        <button
          type="button"
          data-testid="download-plugin-diagnostic-bundle-btn"
          onClick={onDownloadDiagnosticBundle}
          style={{
            padding: '8px 14px',
            cursor: 'pointer',
            backgroundColor: 'var(--mis-brand-primary)',
            color: 'var(--mis-brand-primary-fg)',
            border: '1px solid var(--mis-border-strong)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            fontSize: '13px',
            fontWeight: 700,
          }}
        >
          Download Plugin Diagnostic Bundle
        </button>
      </div>

      {diagnosticBundleStatus && (
        <div
          data-testid="diagnostic-bundle-status"
          style={{ marginBottom: '8px', fontWeight: 600 }}
        >
          {diagnosticBundleStatus}
        </div>
      )}

      {diagnosticBundleJson && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '8px' }}>
          <div>
            <div style={{ fontSize: '12px', fontWeight: 700, marginBottom: '4px' }}>
              Sanitized Diagnostic Bundle (.json)
            </div>
            <pre
              data-testid="diagnostic-bundle-json"
              style={{
                margin: 0,
                padding: '12px',
                backgroundColor: '#0f172a',
                color: '#f8fafc',
                borderRadius: '6px',
                overflowX: 'auto',
                fontSize: '12px',
              }}
            >
              {diagnosticBundleJson}
            </pre>
          </div>
          <div>
            <div style={{ fontSize: '12px', fontWeight: 700, marginBottom: '4px' }}>
              Sanitized Ring Buffer Log Archive (.log)
            </div>
            <pre
              data-testid="diagnostic-bundle-log"
              style={{
                margin: 0,
                padding: '12px',
                backgroundColor: '#0f172a',
                color: '#e2e8f0',
                borderRadius: '6px',
                overflowX: 'auto',
                fontSize: '12px',
              }}
            >
              {diagnosticBundleLog}
            </pre>
          </div>
        </div>
      )}
    </section>
  );
}
