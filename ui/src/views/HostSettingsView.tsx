import React from 'react';
import { MonochromeLucideIcon } from '../components/MonochromeLucideIcon';
import {
  BASE_FONT_SIZE_PRESETS,
  BaseFontSize,
  ThemeMode,
} from '../themeAndTypes';

export interface HostSettingsViewProps {
  schoolName: string;
  schoolURN: string;
  configuredAcademicYear: string;
  themeMode: ThemeMode;
  baseFontSize: BaseFontSize;
  pinnedViewsCount: number;
  onToggleThemeMode: () => void;
  onSelectBaseFontSize: (nextSize: BaseFontSize) => void;
}

export function HostSettingsView({
  schoolName,
  schoolURN,
  configuredAcademicYear,
  themeMode,
  baseFontSize,
  pinnedViewsCount,
  onToggleThemeMode,
  onSelectBaseFontSize,
}: HostSettingsViewProps) {
  return (
    <section
      data-testid="host-settings-view"
      aria-label="Host Settings and Preferences"
      style={{
        width: '100%',
        backgroundColor: 'var(--mis-bg-surface)',
        color: 'var(--mis-text-primary)',
        border: '1px solid var(--mis-border-strong)',
        borderRadius: 'var(--mis-radius-md, 8px)',
        boxShadow: 'var(--mis-shadow-card)',
        padding: '20px',
        display: 'flex',
        flexDirection: 'column',
        gap: '12px',
        boxSizing: 'border-box',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2
          style={{
            margin: 0,
            fontSize: 'calc(16px * var(--mis-font-scale, 1))',
            fontWeight: 700,
            display: 'inline-flex',
            alignItems: 'center',
            gap: '8px',
          }}
        >
          <MonochromeLucideIcon name="settings" size={16} />
          <span>Host Settings &amp; Preferences</span>
        </h2>
      </div>
      <div
        style={{
          fontSize: 'calc(13px * var(--mis-font-scale, 1))',
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
        }}
      >
        <div>
          <strong>School Name:</strong> {schoolName}
        </div>
        <div>
          <strong>School URN:</strong> {schoolURN}
        </div>
        <div>
          <strong>Academic Year:</strong> {configuredAcademicYear}
        </div>
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '10px',
            flexWrap: 'wrap',
          }}
        >
          <label
            htmlFor="theme-mode-toggle-switch"
            style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: '6px' }}
          >
            <strong>High-Contrast Mode:</strong>{' '}
            <span style={{ color: 'var(--mis-text-secondary)' }}>({themeMode})</span>
          </label>
          <button
            id="theme-mode-toggle-switch"
            type="button"
            role="switch"
            aria-checked={themeMode === 'high-contrast'}
            aria-label="Toggle High-Contrast Theme"
            data-testid="host-settings-theme-toggle-btn"
            onClick={onToggleThemeMode}
            style={{
              position: 'relative',
              display: 'inline-flex',
              alignItems: 'center',
              width: '44px',
              height: '24px',
              padding: '2px',
              borderRadius: '999px',
              border: '1px solid var(--mis-border-strong)',
              backgroundColor:
                themeMode === 'high-contrast'
                  ? 'var(--mis-brand-primary)'
                  : 'var(--mis-bg-elevated)',
              cursor: 'pointer',
              transition: 'background-color 150ms ease-in-out',
            }}
          >
            <span
              aria-hidden="true"
              style={{
                width: '18px',
                height: '18px',
                borderRadius: '50%',
                backgroundColor:
                  themeMode === 'high-contrast'
                    ? 'var(--mis-brand-primary-fg)'
                    : 'var(--mis-text-primary)',
                transform:
                  themeMode === 'high-contrast' ? 'translateX(20px)' : 'translateX(0px)',
                transition: 'transform 150ms ease-in-out',
              }}
            />
          </button>
        </div>
        <div
          data-testid="host-settings-font-size-control"
          style={{
            display: 'flex',
            flexDirection: 'column',
            gap: '6px',
          }}
        >
          <div>
            <strong>Base Font Size:</strong>{' '}
            <span style={{ color: 'var(--mis-text-secondary)' }}>
              ({BASE_FONT_SIZE_PRESETS[baseFontSize].label} — {BASE_FONT_SIZE_PRESETS[baseFontSize].px})
            </span>
          </div>
          <div
            role="group"
            aria-label="Base Font Size"
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(4, 1fr)',
              gap: '6px',
            }}
          >
            {(['small', 'medium', 'large', 'extra-large'] as BaseFontSize[]).map((presetKey) => {
              const preset = BASE_FONT_SIZE_PRESETS[presetKey];
              const isSelected = baseFontSize === presetKey;
              return (
                <button
                  key={presetKey}
                  type="button"
                  data-testid={`host-settings-font-size-btn-${presetKey}`}
                  aria-pressed={isSelected}
                  onClick={() => onSelectBaseFontSize(presetKey)}
                  style={{
                    padding: '6px 8px',
                    borderRadius: 'var(--mis-radius-sm, 4px)',
                    border: isSelected
                      ? '1px solid var(--mis-brand-primary)'
                      : '1px solid var(--mis-border-subtle)',
                    backgroundColor: isSelected
                      ? 'var(--mis-brand-primary)'
                      : 'var(--mis-bg-elevated)',
                    color: isSelected
                      ? 'var(--mis-brand-primary-fg)'
                      : 'var(--mis-text-primary)',
                    fontSize: 'calc(12px * var(--mis-font-scale, 1))',
                    fontWeight: isSelected ? 700 : 600,
                    cursor: 'pointer',
                  }}
                >
                  {preset.label}
                </button>
              );
            })}
          </div>
        </div>
        <div>
          <strong>Pinned Views:</strong> {pinnedViewsCount}
        </div>
      </div>
    </section>
  );
}
