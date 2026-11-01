import React from 'react';

export interface DashboardSummaryStripProps {
  routeError: string;
}

export const DashboardSummaryStrip: React.FC<DashboardSummaryStripProps> = ({
  routeError,
}) => {
  if (!routeError) {
    return null;
  }

  return (
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
  );
};

