export const PLUGIN_ID = 'uk.org.optimis.test';

export type ThemeMode = 'light' | 'high-contrast';

export const DEFAULT_THEME_TOKENS: Record<ThemeMode, Record<string, string>> = {
  light: {
    '--mis-brand-primary': '#1d4ed8',
    '--mis-brand-primary-hover': '#1e40af',
    '--mis-brand-primary-fg': '#ffffff',
    '--mis-brand-accent': '#0f766e',
    '--mis-bg-canvas': '#f1f5f9',
    '--mis-bg-surface': '#ffffff',
    '--mis-bg-elevated': '#f8fafc',
    '--mis-bg-sidebar': '#0f172a',
    '--mis-text-sidebar': '#f8fafc',
    '--mis-text-primary': '#0f172a',
    '--mis-text-secondary': '#475569',
    '--mis-text-muted': '#64748b',
    '--mis-text-inverse': '#ffffff',
    '--mis-border-subtle': '#e2e8f0',
    '--mis-border-strong': '#cbd5e1',
    '--mis-focus-ring': '0 0 0 3px rgba(29, 78, 216, 0.35)',
    '--mis-shadow-card': '0 1px 3px 0 rgba(15, 23, 42, 0.08), 0 1px 2px -1px rgba(15, 23, 42, 0.06)',
    '--mis-status-success': '#15803d',
    '--mis-status-warning': '#b45309',
    '--mis-status-danger': '#b91c1c',
    '--mis-status-danger-bg': '#fef2f2',
    '--mis-font-sans': "'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    '--mis-radius-sm': '4px',
    '--mis-radius-md': '8px',
    '--mis-radius-lg': '12px',
  },
  'high-contrast': {
    '--mis-brand-primary': '#facc15',
    '--mis-brand-primary-hover': '#fde047',
    '--mis-brand-primary-fg': '#000000',
    '--mis-brand-accent': '#22d3ee',
    '--mis-bg-canvas': '#000000',
    '--mis-bg-surface': '#0a0a0a',
    '--mis-bg-elevated': '#171717',
    '--mis-bg-sidebar': '#000000',
    '--mis-text-sidebar': '#ffffff',
    '--mis-text-primary': '#ffffff',
    '--mis-text-secondary': '#fef08a',
    '--mis-text-muted': '#e2e8f0',
    '--mis-text-inverse': '#000000',
    '--mis-border-subtle': '#ffffff',
    '--mis-border-strong': '#facc15',
    '--mis-focus-ring': '0 0 0 3px #facc15',
    '--mis-shadow-card': 'none',
    '--mis-status-success': '#4ade80',
    '--mis-status-warning': '#facc15',
    '--mis-status-danger': '#ff6b6b',
    '--mis-status-danger-bg': '#2a0000',
    '--mis-font-sans': "'Inter', system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    '--mis-radius-sm': '4px',
    '--mis-radius-md': '8px',
    '--mis-radius-lg': '12px',
  },
};

export const DEFAULT_PERMITTED_METHODS = [
  'get_pupil_record',
  'save_pupil_note',
  'list_pupil_notes',
  'get_group_roster_summary',
  'get_pupil_guardians',
  'record_group_am_attendance',
  'submit_safeguarding_postbox',
  'sync_external_sparx_get',
  'push_external_sparx_post',
  'mutate_then_illegal_egress',
  'trigger_loop_relay',
  'list_worker_event_logs',
];

export function applyHostThemeToRoot(mode: ThemeMode, tokensMap: Record<ThemeMode, Record<string, string>>) {
  if (typeof document === 'undefined' || !document.documentElement) {
    return;
  }
  const root = document.documentElement;
  root.setAttribute('data-mis-theme', mode);
  const tokens = tokensMap[mode] || DEFAULT_THEME_TOKENS[mode];
  for (const [k, v] of Object.entries(tokens)) {
    root.style.setProperty(k, v);
  }
}

export interface SARSection {
  source_id: string;
  source_type: string;
  plugin_state?: string;
  title: string;
  records: Array<Record<string, unknown>>;
  redactable_fields: string[];
}

export interface SafeguardingNoteEnvelope {
  note_id: string;
  pupil_id: string;
  reported_by_staff_id: string;
  assigned_dsl_staff_id: string;
  concern_category: string;
  status: string;
  incident_date: string;
  kms_key_id: string;
  created_at: string;
}

export interface SafeguardingAuditEntry {
  audit_id: string;
  action_type: string;
  actor_person_id: string;
  request_id: string;
  timestamp: string;
}

export interface DecryptedSafeguardingNote {
  note_id: string;
  pupil_id: string;
  concern_category: string;
  status: string;
  incident_date: string;
  summary: string;
  body_map: string;
  verbatim_account: string;
  action_taken: string;
  audit_logs?: SafeguardingAuditEntry[];
}

export const MAX_KEEP_ALIVE_POOL_SIZE = 5;

export interface PluginWidgetDescriptor {
  id: string;
  plugin_id: string;
  title: string;
  entry: string;
  entry_url: string;
}

export interface PlacedDashboardWidget {
  id: string;
  plugin_id: string;
  title: string;
  entry: string;
  entry_url: string;
  colSpan: number;
  rowSpan: number;
}

export const DEFAULT_PLUGIN_WIDGETS: PluginWidgetDescriptor[] = [
  {
    id: 'attendance_pulse',
    plugin_id: PLUGIN_ID,
    title: 'Statutory Attendance Pulse',
    entry: 'ui/dist/widget.html',
    entry_url: `/plugins/${PLUGIN_ID}/ui/widget.html`,
  },
  {
    id: 'pastoral_watch',
    plugin_id: PLUGIN_ID,
    title: 'Pastoral & Behaviour Watch',
    entry: 'ui/dist/widget.html',
    entry_url: `/plugins/${PLUGIN_ID}/ui/widget.html`,
  },
];

export interface PluginNavEntry {
  id: string;
  title: string;
  version: string;
  entry_url: string;
  permitted_methods: string[];
  widgets?: PluginWidgetDescriptor[];
}

export interface StaffPersonaSummary {
  staff_id: string;
  person_id: string;
  user_id: string;
  staff_code: string;
  display_name: string;
  job_title: string;
  is_dsl: boolean;
  active_roles: string[];
  scopes: string[];
}

export interface ActiveCallerIdentity {
  staff_id: string;
  user_id: string;
  active_roles: string[];
  scopes: string[];
}


