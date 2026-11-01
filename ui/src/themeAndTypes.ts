export type ThemeMode = 'light' | 'high-contrast';

export type BaseFontSize = 'small' | 'medium' | 'large' | 'extra-large';

export const BASE_FONT_SIZE_PRESETS: Record<
  BaseFontSize,
  { label: string; px: string; scale: string }
> = {
  small: { label: 'Small', px: '14px', scale: '0.875' },
  medium: { label: 'Medium', px: '16px', scale: '1' },
  large: { label: 'Large', px: '18px', scale: '1.125' },
  'extra-large': { label: 'Extra Large', px: '20px', scale: '1.25' },
};

export function normalizeBaseFontSize(raw: unknown): BaseFontSize {
  if (typeof raw === 'string') {
    const lower = raw.trim().toLowerCase();
    if (lower === 'small' || lower === 'medium' || lower === 'large' || lower === 'extra-large') {
      return lower;
    }
  }
  return 'medium';
}

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
    '--mis-font-size-base': '16px',
    '--mis-font-scale': '1',
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
    '--mis-font-size-base': '16px',
    '--mis-font-scale': '1',
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

export function applyHostThemeToRoot(
  mode: ThemeMode,
  tokensMap: Record<ThemeMode, Record<string, string>>,
  baseFontSize: BaseFontSize = 'medium'
) {
  if (typeof document === 'undefined' || !document.documentElement) {
    return;
  }
  const root = document.documentElement;
  const normalizedFontSize = normalizeBaseFontSize(baseFontSize);
  const preset = BASE_FONT_SIZE_PRESETS[normalizedFontSize];
  root.setAttribute('data-mis-theme', mode);
  root.setAttribute('data-mis-font-size', normalizedFontSize);
  const tokens = tokensMap[mode] || DEFAULT_THEME_TOKENS[mode];
  for (const [k, v] of Object.entries(tokens)) {
    root.style.setProperty(k, v);
  }
  root.style.setProperty('--mis-font-size-base', preset.px);
  root.style.setProperty('--mis-font-scale', preset.scale);
  root.style.fontSize = preset.px;
  if (document.body) {
    document.body.style.margin = '0';
    document.body.style.fontSize = preset.px;
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
  icon?: string;
  entry: string;
  entry_url: string;
  width: 1 | 2;
  height: 1 | 2;
  keywords?: string[];
}

export interface PluginViewDescriptor {
  id: string;
  plugin_id?: string;
  path: string;
  title?: string;
  icon?: string;
  entry: string;
  entry_url: string;
  query_params?: string[];
  keywords?: string[];
}

export interface SearchPupilResult {
  id: string;
  upn: string;
  first_name: string;
  last_name: string;
  preferred_first_name?: string;
  preferred_last_name?: string;
  nc_year_group?: string;
}

export interface SearchGroupResult {
  id: string;
  code: string;
  name: string;
  group_type: string;
}

export interface DashboardLayoutItem {
  plug_id: string;
  widget_id: string;
  col?: number;
  row?: number;
}

export interface ResolvedDashboardWidgetSlot {
  plug_id: string;
  widget_id: string;
  col?: number;
  row?: number;
  permitted: boolean;
  error_code?: string;
  error_title?: string;
  title?: string;
  icon?: string;
  entry?: string;
  entry_url?: string;
  width?: 1 | 2;
  height?: 1 | 2;
}

export interface PlacedDashboardWidget {
  id: string;
  plugin_id: string;
  title: string;
  icon?: string;
  entry: string;
  entry_url: string;
  width?: 1 | 2;
  height?: 1 | 2;
  colSpan: 1 | 2;
  rowSpan: 1 | 2;
  col: number;
  row: number;
  permitted?: boolean;
  error_code?: string;
  error_title?: string;
}

export function getWidgetCompositeKey(pluginId: string, widgetId: string): string {
  return `${pluginId}:${widgetId}`;
}

export function is2DFootprintFree(
  placedWidgets: PlacedDashboardWidget[],
  col: number,
  row: number,
  width: number,
  height: number,
  columnCount: number,
  ignoreCompositeKey?: string
): boolean {
  if (col < 1 || row < 1 || col + width - 1 > columnCount) {
    return false;
  }
  for (const w of placedWidgets) {
    const key = getWidgetCompositeKey(w.plugin_id, w.id);
    if (ignoreCompositeKey && key === ignoreCompositeKey) {
      continue;
    }
    const wWidth = w.width === 1 || w.width === 2 ? w.width : w.colSpan;
    const wHeight = w.height === 1 || w.height === 2 ? w.height : w.rowSpan;
    const overlapsCol = col < w.col + wWidth && col + width > w.col;
    const overlapsRow = row < w.row + wHeight && row + height > w.row;
    if (overlapsCol && overlapsRow) {
      return false;
    }
  }
  return true;
}

export function findFirstFree2DSlot(
  placedWidgets: PlacedDashboardWidget[],
  width: number,
  height: number,
  columnCount: number
): { col: number; row: number } {
  const effectiveWidth = Math.min(Math.max(1, width), columnCount);
  const effectiveHeight = Math.max(1, height);
  for (let r = 1; r <= 200; r++) {
    for (let c = 1; c <= columnCount - effectiveWidth + 1; c++) {
      if (is2DFootprintFree(placedWidgets, c, r, effectiveWidth, effectiveHeight, columnCount)) {
        return { col: c, row: r };
      }
    }
  }
  return { col: 1, row: 1 };
}

export interface PluginNavEntry {
  id: string;
  title: string;
  icon?: string;
  version: string;
  entry_url: string;
  permitted_methods: string[];
  widgets?: PluginWidgetDescriptor[];
  views?: PluginViewDescriptor[];
}

export interface PinnedRecentEntry {
  route: string;
  title: string;
  icon?: string;
  plug_id: string;
  view_id?: string;
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

export interface DiagnosticBundleEntry {
  timestamp: string;
  plugin_id: string;
  method: string;
  error_class: string;
  file: string;
  line: number;
  stack_trace: string;
  duration_ms: number;
  sqlite_lock_wait_ms: number;
  sqlite_write_hold_ms: number;
}

export interface DiagnosticBundlePayload {
  schema_version: string;
  school_urn: string;
  generated_at: string;
  generated_by_staff_id: string;
  telemetry_mode: string;
  ring_buffer_capacity: number;
  entry_count: number;
  pii_scrubbed: boolean;
  entries: DiagnosticBundleEntry[];
  log_archive: string;
}
