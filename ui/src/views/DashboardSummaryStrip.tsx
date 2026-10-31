import React from 'react';

export interface DashboardSummaryStripProps {
  routeError: string;
  studentEntityLinkHandler: string;
}

export const DashboardSummaryStrip: React.FC<DashboardSummaryStripProps> = ({
  routeError,
  studentEntityLinkHandler,
}) => {
  return (
    <>
      {routeError && (
        <div
          data-testid="host-route-error-banner"
          style={{
            padding: '10px 14px',
            backgroundColor: 'var(--mis-status-danger-bg)',
            color: 'var(--mis-status-danger)',
            border: '1px solid var(--mis-border-strong)',
            borderRadius: 'var(--mis-radius-sm, 4px)',
            fontSize: '13px',
            fontWeight: 700,
          }}
        >
          {routeError}
        </div>
      )}

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
            {studentEntityLinkHandler}
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
    </>
  );
};
