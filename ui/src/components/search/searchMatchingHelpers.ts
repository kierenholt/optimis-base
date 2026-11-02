import { PluginNavEntry, PluginWidgetDescriptor } from '../../themeAndTypes';
import { IndexedViewResult } from './searchTypes';

export function matchesViewItem(item: IndexedViewResult, qLower: string): boolean {
  if (!qLower) {
    return true;
  }
  const vTitle = (item.view.title || item.pluginTitle || '').toLowerCase();
  const pTitle = (item.pluginTitle || '').toLowerCase();
  const vId = (item.view.id || '').toLowerCase();
  const vPath = (item.view.path || '').toLowerCase();
  if (
    vTitle.includes(qLower) ||
    pTitle.includes(qLower) ||
    vId.includes(qLower) ||
    vPath.includes(qLower)
  ) {
    return true;
  }
  if (Array.isArray(item.view.keywords)) {
    for (const kw of item.view.keywords) {
      if (kw.toLowerCase().includes(qLower)) {
        return true;
      }
    }
  }
  return false;
}

export function matchesWidgetItem(
  widget: PluginWidgetDescriptor,
  pluginTitle: string,
  qLower: string
): boolean {
  if (!qLower) {
    return true;
  }
  const wTitle = (widget.title || '').toLowerCase();
  const wId = (widget.id || '').toLowerCase();
  const pTitle = (pluginTitle || '').toLowerCase();
  if (wTitle.includes(qLower) || wId.includes(qLower) || pTitle.includes(qLower)) {
    return true;
  }
  if (Array.isArray(widget.keywords)) {
    for (const kw of widget.keywords) {
      if (kw.toLowerCase().includes(qLower)) {
        return true;
      }
    }
  }
  return false;
}

export function isParameterizedViewResolvable(
  path: string | undefined,
  activePupilId: string,
  activeGroupId: string,
  activeStaffId: string
): boolean {
  const p = path || '/';
  if (p.includes(':pupil_id') && !activePupilId) {
    return false;
  }
  if (p.includes(':group_id') && !activeGroupId) {
    return false;
  }
  if (p.includes(':staff_id') && !activeStaffId) {
    return false;
  }
  return true;
}

export function resolveParameterizedViewSubpath(
  path: string | undefined,
  activePupilId: string,
  activeGroupId: string,
  activeStaffId: string
): string {
  let resolved = path || '/';
  if (resolved.includes(':pupil_id')) {
    if (!activePupilId) {
      throw new Error('Missing mandatory activePupilId for :pupil_id view route');
    }
    resolved = resolved.replace(':pupil_id', activePupilId);
  }
  if (resolved.includes(':group_id')) {
    if (!activeGroupId) {
      throw new Error('Missing mandatory activeGroupId for :group_id view route');
    }
    resolved = resolved.replace(':group_id', activeGroupId);
  }
  if (resolved.includes(':staff_id')) {
    if (!activeStaffId) {
      throw new Error('Missing mandatory activeStaffId for :staff_id view route');
    }
    resolved = resolved.replace(':staff_id', activeStaffId);
  }
  return resolved;
}

export function filterPluginViews(
  installedPlugins: PluginNavEntry[],
  qLower: string,
  activePupilId: string,
  activeGroupId: string,
  activeStaffId: string,
  requireQuery: boolean
): IndexedViewResult[] {
  if (requireQuery && !qLower) {
    return [];
  }
  const results: IndexedViewResult[] = [];
  for (const p of installedPlugins) {
    const views = Array.isArray(p.views) ? p.views : [];
    const permittedMethods = Array.isArray(p.permitted_methods) ? p.permitted_methods : [];
    for (const v of views) {
      if (!isParameterizedViewResolvable(v.path, activePupilId, activeGroupId, activeStaffId)) {
        continue;
      }
      const candidate: IndexedViewResult = {
        pluginId: p.id,
        pluginTitle: p.title,
        pluginIcon: p.icon,
        view: v,
        permittedMethods,
      };
      if (matchesViewItem(candidate, qLower)) {
        results.push(candidate);
      }
    }
  }
  return results;
}

export function filterPluginWidgets(
  installedPlugins: PluginNavEntry[],
  qLower: string,
  requireQuery: boolean
): Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }> {
  if (requireQuery && !qLower) {
    return [];
  }
  const results: Array<{ widget: PluginWidgetDescriptor; pluginTitle: string; pluginIcon?: string }> = [];
  for (const p of installedPlugins) {
    const widgets = Array.isArray(p.widgets) ? p.widgets : [];
    for (const w of widgets) {
      if (matchesWidgetItem(w, p.title, qLower)) {
        results.push({ widget: w, pluginTitle: p.title, pluginIcon: p.icon });
      }
    }
  }
  return results;
}
