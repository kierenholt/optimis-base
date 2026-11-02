import React from 'react';

export interface WidgetFootprintDiagramProps {
  testId: string;
  width: number;
  height: number;
  cellWidthPx?: number;
  cellHeightPx?: number;
}

export const WidgetFootprintDiagram: React.FC<WidgetFootprintDiagramProps> = ({
  testId,
  width,
  height,
  cellWidthPx = 18,
  cellHeightPx = 14,
}) => {
  const spanW = width === 2 ? 2 : 1;
  const spanH = height === 2 ? 2 : 1;
  return (
    <div
      data-testid={testId}
      data-footprint-width={String(spanW)}
      data-footprint-height={String(spanH)}
      title={`Footprint: ${spanW} col × ${spanH} row`}
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(2, ${cellWidthPx}px)`,
        gridTemplateRows: `repeat(2, ${cellHeightPx}px)`,
        gap: '3px',
        padding: '5px',
        borderRadius: 'var(--mis-radius-sm, 4px)',
        border: '1px solid var(--mis-border-subtle)',
        backgroundColor: 'var(--mis-bg-surface)',
        flexShrink: 0,
      }}
    >
      {[1, 2].map((r) =>
        [1, 2].map((c) => {
          const activeCell = c <= spanW && r <= spanH;
          return (
            <div
              key={`fp-${c}-${r}`}
              style={{
                borderRadius: '2px',
                backgroundColor: activeCell
                  ? 'var(--mis-brand-primary)'
                  : 'var(--mis-border-subtle)',
                opacity: activeCell ? 1 : 0.45,
              }}
            />
          );
        })
      )}
    </div>
  );
};
