/**
 * @ukmis/tailwind-preset — Official Tailwind CSS Preset for Optimis Plugins (Sections 9.5 & 10C)
 *
 * Maps school branding, surface, text, border, and semantic status tokens injected by the Host
 * via --mis-* CSS custom properties on mis:init.
 */
'use strict';

module.exports = {
  theme: {
    extend: {
      colors: {
        mis: {
          brand: {
            primary: 'var(--mis-brand-primary, #1d4ed8)',
            hover: 'var(--mis-brand-primary-hover, #1e40af)',
            contrast: 'var(--mis-brand-primary-contrast, #ffffff)',
          },
          surface: {
            bg: 'var(--mis-surface-bg, #f8fafc)',
            card: 'var(--mis-surface-card, #ffffff)',
            muted: 'var(--mis-surface-muted, #f1f5f9)',
          },
          text: {
            primary: 'var(--mis-text-primary, #0f172a)',
            secondary: 'var(--mis-text-secondary, #334155)',
            muted: 'var(--mis-text-muted, #64748b)',
          },
          border: {
            subtle: 'var(--mis-border-subtle, #e2e8f0)',
            strong: 'var(--mis-border-strong, #cbd5e1)',
            focus: 'var(--mis-focus-ring, #2563eb)',
          },
          status: {
            success: 'var(--mis-status-success-text, #166534)',
            warning: 'var(--mis-status-warning-text, #854d0e)',
            danger: 'var(--mis-status-danger-text, #991b1b)',
          },
        },
      },
      fontFamily: {
        sans: ['var(--mis-font-sans, Inter, system-ui, -apple-system, sans-serif)'],
        mono: ['var(--mis-font-mono, ui-monospace, SFMono-Regular, Menlo, monospace)'],
      },
      borderRadius: {
        'mis-sm': 'var(--mis-radius-sm, 0.375rem)',
        'mis-md': 'var(--mis-radius-md, 0.5rem)',
        'mis-lg': 'var(--mis-radius-lg, 0.75rem)',
      },
    },
  },
};
