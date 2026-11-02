import React from 'react';

export const searchInputStyle: React.CSSProperties = {
  width: '100%',
  boxSizing: 'border-box',
  padding: '10px 12px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-strong)',
  backgroundColor: 'var(--mis-bg-elevated)',
  color: 'var(--mis-text-primary)',
  fontSize: 'calc(13px * var(--mis-font-scale, 1))',
};

export const resultItemButtonStyle: React.CSSProperties = {
  flex: 1,
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  gap: '8px',
  padding: '8px 12px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-subtle)',
  backgroundColor: 'var(--mis-bg-elevated)',
  color: 'var(--mis-text-primary)',
  textAlign: 'left',
  cursor: 'pointer',
};

export const pinButtonStyle = (isPinned: boolean): React.CSSProperties => ({
  padding: '6px 10px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-subtle)',
  backgroundColor: isPinned ? 'var(--mis-brand-primary)' : 'var(--mis-bg-elevated)',
  color: isPinned ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-primary)',
  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
  fontWeight: 600,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
  display: 'inline-flex',
  alignItems: 'center',
  gap: '4px',
});

export const widgetItemContainerStyle: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  gap: '8px',
  padding: '8px 12px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-subtle)',
  backgroundColor: 'var(--mis-bg-elevated)',
};

export const addWidgetButtonStyle = (isPlaced: boolean): React.CSSProperties => ({
  padding: '5px 10px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-subtle)',
  backgroundColor: isPlaced ? 'var(--mis-bg-surface)' : 'var(--mis-brand-primary)',
  color: isPlaced ? 'var(--mis-text-muted)' : 'var(--mis-brand-primary-fg)',
  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
  fontWeight: 600,
  cursor: isPlaced ? 'default' : 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  gap: '4px',
});

export const sectionTitleStyle: React.CSSProperties = {
  fontSize: 'calc(11px * var(--mis-font-scale, 1))',
  fontWeight: 700,
  textTransform: 'uppercase',
  color: 'var(--mis-text-secondary)',
};

export const emptyStateStyle: React.CSSProperties = {
  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
  color: 'var(--mis-text-secondary)',
  padding: '6px 0',
};

export const tabButtonStyle = (isSelected: boolean): React.CSSProperties => ({
  padding: '4px 10px',
  borderRadius: 'var(--mis-radius-sm, 4px)',
  border: '1px solid var(--mis-border-subtle)',
  backgroundColor: isSelected ? 'var(--mis-brand-primary)' : 'var(--mis-bg-elevated)',
  color: isSelected ? 'var(--mis-brand-primary-fg)' : 'var(--mis-text-primary)',
  fontSize: 'calc(12px * var(--mis-font-scale, 1))',
  fontWeight: 600,
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  gap: '5px',
});
