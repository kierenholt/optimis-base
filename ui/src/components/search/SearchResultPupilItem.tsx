import React from 'react';
import { SearchPupilResult } from '../../themeAndTypes';
import { MonochromeLucideIcon } from '../MonochromeLucideIcon';
import { resultItemButtonStyle } from './searchStyles';

export interface SearchResultPupilItemProps {
  pupil: SearchPupilResult;
  testId: string;
  onSelect: (pupilId: string) => void;
}

export const SearchResultPupilItem: React.FC<SearchResultPupilItemProps> = ({
  pupil,
  testId,
  onSelect,
}) => (
  <button
    type="button"
    data-testid={testId}
    onClick={() => onSelect(pupil.id)}
    style={resultItemButtonStyle}
  >
    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
      <MonochromeLucideIcon name="user" size={16} />
      <div>
        <div style={{ fontSize: 'calc(13px * var(--mis-font-scale, 1))', fontWeight: 700 }}>
          {pupil.first_name} {pupil.last_name}
        </div>
        <div style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', color: 'var(--mis-text-secondary)' }}>
          UPN {pupil.upn} • Year {pupil.nc_year_group || 'N/A'}
        </div>
      </div>
    </div>
    <span style={{ fontSize: 'calc(11px * var(--mis-font-scale, 1))', fontWeight: 600, display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
      <span>Open Student</span>
      <MonochromeLucideIcon name="arrow-right" size={12} />
    </span>
  </button>
);
