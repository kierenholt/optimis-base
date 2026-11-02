import React from 'react';

export interface DashboardEmptyCellProps {
  col: number;
  row: number;
  isPlacementPending: boolean;
  onPointer: (e: React.MouseEvent<HTMLDivElement>, col: number, row: number) => void;
  onConfirm: (col: number, row: number) => void;
}

export const DashboardEmptyCell: React.FC<DashboardEmptyCellProps> = ({
  col,
  row,
  isPlacementPending,
  onPointer,
  onConfirm,
}) => {
  return (
    <div
      data-testid={`dashboard-empty-grid-cell-${col}-${row}`}
      data-dotted-cell="true"
      data-grid-col={String(col)}
      data-grid-row={String(row)}
      onMouseEnter={(e) => onPointer(e, col, row)}
      onMouseMove={(e) => onPointer(e, col, row)}
      onClick={() => {
        if (isPlacementPending) {
          onConfirm(col, row);
        }
      }}
      style={{
        position: 'relative',
        gridColumn: `${col} / span 1`,
        gridRow: `${row} / span 1`,
        border: '2px dashed var(--mis-border-strong)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        backgroundColor: 'transparent',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        boxSizing: 'border-box',
        cursor: isPlacementPending ? 'pointer' : 'default',
      }}
    />
  );
};
