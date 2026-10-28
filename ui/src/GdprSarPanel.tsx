import React from 'react';
import { SARSection } from './themeAndTypes';

interface GdprSarPanelProps {
  sarSections: SARSection[];
  sarExportStatus: string;
  sarFinalizedJson: string;
  sarPurgeStatus: string;
  onLoadSarExport: () => void;
  onFinalizeSarExport: () => void;
  onPurgePupilData: () => void;
  onUpdateRedactableField: (secIdx: number, recIdx: number, fieldKey: string, newValue: string) => void;
}

export function GdprSarPanel({
  sarSections,
  sarExportStatus,
  sarFinalizedJson,
  sarPurgeStatus,
  onLoadSarExport,
  onFinalizeSarExport,
  onPurgePupilData,
  onUpdateRedactableField,
}: GdprSarPanelProps) {
  return (
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
            onClick={onLoadSarExport}
            style={{ padding: '6px 12px', cursor: 'pointer' }}
          >
            Load SAR Package
          </button>
          <button
            type="button"
            data-testid="sar-finalize-export-btn"
            onClick={onFinalizeSarExport}
            disabled={sarSections.length === 0}
            style={{ padding: '6px 12px', cursor: 'pointer' }}
          >
            Finalize &amp; Export Redacted SAR
          </button>
          <button
            type="button"
            data-testid="sar-purge-pupil-btn"
            onClick={onPurgePupilData}
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
                            onUpdateRedactableField(secIdx, recIdx, fieldKey, e.target.value)
                          }
                          style={{ flex: 1, padding: '4px 8px' }}
                        />
                        <button
                          type="button"
                          data-testid={`sar-redact-mask-btn-${sec.source_id}-${sec.title}-${recIdx}-${fieldKey}`}
                          onClick={() =>
                            onUpdateRedactableField(secIdx, recIdx, fieldKey, '[REDACTED]')
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
  );
}
