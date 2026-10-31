import React from 'react';
import {
  ActiveCallerIdentity,
  DEFAULT_PERMITTED_METHODS,
  DEFAULT_THEME_TOKENS,
  PluginNavEntry,
  ThemeMode,
} from '../themeAndTypes';
import { validatePluginViewRoute } from './routeUtils';

export interface InitContextOverrides {
  pupil_id: string;
  group_id: string;
  staff_id: string;
  entity_type: string;
  entity_id: string;
  tab: string;
  view_path: string;
  route_query: Record<string, string>;
}

export interface BuildInitPayloadContext {
  themeMode: ThemeMode;
  themeCatalog: Record<ThemeMode, Record<string, string>>;
  installedPlugins: PluginNavEntry[];
  activeCaller: ActiveCallerIdentity | null;
  activePupilId: string;
  initialSecondPupilId: string;
  activeGroupId: string;
  initialOtherGroupId: string;
  initialAbsentStaffId: string;
  initialCoverStaffId: string;
  initialSlotId: string;
  initialReferenceDate: string;
  initialAcademicYearId: string;
  configuredAcademicYear: string;
  activeStaffId: string;
  activeTab: string;
  activeViewPath: string;
  activeRouteQuery: Record<string, string>;
}

export function createInitPayload(
  ctx: BuildInitPayloadContext,
  targetPluginId: string,
  overrideContext?: Partial<InitContextOverrides>,
  overrideThemeMode?: ThemeMode,
  overridePlugins?: PluginNavEntry[],
  overrideCaller?: ActiveCallerIdentity | null
) {
  const mode = overrideThemeMode || ctx.themeMode;
  const activeTokens = ctx.themeCatalog[mode] || DEFAULT_THEME_TOKENS[mode];
  const effectivePlugins = overridePlugins || ctx.installedPlugins;
  const effectiveCaller = overrideCaller !== undefined ? overrideCaller : ctx.activeCaller;
  const resolvedPupilId = overrideContext?.pupil_id !== undefined ? overrideContext.pupil_id : ctx.activePupilId;
  const resolvedGroupId = overrideContext?.group_id !== undefined ? overrideContext.group_id : ctx.activeGroupId;
  const resolvedStaffId =
    overrideContext?.staff_id !== undefined
      ? overrideContext.staff_id
      : ctx.activeStaffId || effectiveCaller?.staff_id || '';
  const resolvedTab = overrideContext?.tab !== undefined ? overrideContext.tab : ctx.activeTab;
  const resolvedViewPath =
    overrideContext?.view_path !== undefined ? overrideContext.view_path : ctx.activeViewPath || '/';
  const resolvedRouteQuery =
    overrideContext?.route_query !== undefined ? overrideContext.route_query : ctx.activeRouteQuery;

  const pluginDescriptor = effectivePlugins.find((p) => p.id === targetPluginId);
  const permittedMethods = pluginDescriptor
    ? pluginDescriptor.permitted_methods
    : DEFAULT_PERMITTED_METHODS;

  const routeValidation = validatePluginViewRoute(
    pluginDescriptor,
    resolvedViewPath,
    resolvedRouteQuery
  );

  const cleanSubpath =
    !resolvedViewPath || resolvedViewPath === '/'
      ? ''
      : resolvedViewPath.startsWith('/')
        ? resolvedViewPath
        : `/${resolvedViewPath}`;
  const querySearch = new URLSearchParams(resolvedRouteQuery).toString();
  const routeUrl = `/p/${targetPluginId}${cleanSubpath}${querySearch ? `?${querySearch}` : ''}`;

  return {
    type: 'mis:init',
    theme_mode: mode,
    theme_tokens: activeTokens,
    theme: {
      primary: activeTokens['--mis-brand-primary'],
    },
    context: {
      pupil_id: resolvedPupilId || null,
      second_pupil_id: ctx.initialSecondPupilId || null,
      group_id: resolvedGroupId || null,
      other_group_id: ctx.initialOtherGroupId || null,
      absent_staff_id: ctx.initialAbsentStaffId || null,
      cover_staff_id: ctx.initialCoverStaffId || null,
      slot_id: ctx.initialSlotId || null,
      reference_date: ctx.initialReferenceDate || null,
      academic_year_id: ctx.initialAcademicYearId || ctx.configuredAcademicYear,
      staff_id: resolvedStaffId || null,
      tab: resolvedTab || null,
      entity_type:
        overrideContext?.entity_type || (resolvedPupilId ? 'student' : resolvedGroupId ? 'group' : 'dashboard'),
      entity_id: overrideContext?.entity_id || resolvedPupilId || resolvedGroupId || 'home',
    },
    route: {
      plug_id: targetPluginId,
      view_id: routeValidation.viewId || 'home',
      path: resolvedViewPath || '/',
      search: querySearch ? `?${querySearch}` : '',
      url: routeUrl,
      tab: resolvedTab || null,
      params: routeValidation.params,
      query: resolvedRouteQuery,
    },
    caller: {
      staff_id: resolvedStaffId || null,
      user_id: effectiveCaller?.user_id || null,
      active_roles: effectiveCaller?.active_roles || [],
      scopes: effectiveCaller?.scopes || [],
    },
    scopes: effectiveCaller?.scopes || [],
    active_roles: effectiveCaller?.active_roles || [],
    permitted_methods: permittedMethods,
  };
}

export interface RouteAndLinkHandlerDeps {
  installedPlugins: PluginNavEntry[];
  activePluginId: string;
  activeViewPath: string;
  activePupilId: string;
  activeGroupId: string;
  activeStaffId: string;
  entityLinkHandlers: Record<string, string>;
  pluginIframeRefs: React.MutableRefObject<Map<string, HTMLIFrameElement>>;
  setRouteError: React.Dispatch<React.SetStateAction<string>>;
  setActiveViewPath: React.Dispatch<React.SetStateAction<string>>;
  setActiveTab: React.Dispatch<React.SetStateAction<string>>;
  setActiveRouteQuery: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  setActivePupilId: React.Dispatch<React.SetStateAction<string>>;
  setActiveGroupId: React.Dispatch<React.SetStateAction<string>>;
  setActiveStaffId: React.Dispatch<React.SetStateAction<string>>;
  setActiveEntityContext: React.Dispatch<React.SetStateAction<string>>;
  setDeepLinkNavigationStatus: React.Dispatch<React.SetStateAction<string>>;
  activatePluginInKeepAlivePool: (
    targetPluginId: string,
    overrideViewPath?: string,
    overridePluginQuery?: Record<string, string>
  ) => void;
  syncBrowserRouteUrl: (
    nextPlugId: string,
    nextViewPath: string,
    nextPluginQuery: Record<string, string>,
    replaceHistory?: boolean
  ) => string;
  broadcastInitToMountedIframes: (overrideContext?: Partial<InitContextOverrides>) => void;
}

export function handleRouteStateMessage(
  msg: Record<string, unknown>,
  sourcePluginId: string,
  deps: RouteAndLinkHandlerDeps
) {
  const targetPlugId =
    (typeof msg.plug_id === 'string' && msg.plug_id.trim()) ||
    sourcePluginId ||
    deps.activePluginId;
  if (!targetPlugId) {
    return;
  }

  const targetPlugin = deps.installedPlugins.find((p) => p.id === targetPlugId);
  let candidatePath = deps.activeViewPath || '/';
  if (typeof msg.path === 'string' && msg.path.trim() !== '') {
    const rawPath = msg.path.trim();
    if (rawPath !== '/' || !(deps.activeViewPath && deps.activeViewPath !== '/')) {
      candidatePath = rawPath.startsWith('/') ? rawPath : `/${rawPath}`;
    }
  }
  if (
    (msg.entity_type === 'student' || (typeof msg.pupil_id === 'string' && msg.pupil_id.trim() !== '')) &&
    (!msg.path || msg.path === '/')
  ) {
    const pid = String(msg.pupil_id || msg.entity_id || '').trim();
    if (pid) {
      candidatePath = `/student/${encodeURIComponent(pid)}`;
    }
  } else if (
    (msg.entity_type === 'group' || (typeof msg.group_id === 'string' && msg.group_id.trim() !== '')) &&
    (!msg.path || msg.path === '/')
  ) {
    const gid = String(msg.group_id || msg.entity_id || '').trim();
    if (gid) {
      candidatePath = `/group/${encodeURIComponent(gid)}`;
    }
  }

  const candidateQuery: Record<string, string> = {};
  if (msg.query && typeof msg.query === 'object') {
    for (const [k, v] of Object.entries(msg.query as Record<string, unknown>)) {
      if (v !== undefined && v !== null && String(v) !== '') {
        candidateQuery[k] = String(v);
      }
    }
  }
  if (typeof msg.tab === 'string' && msg.tab.trim() !== '') {
    candidateQuery.tab = msg.tab.trim();
  }

  const validation = validatePluginViewRoute(targetPlugin, candidatePath, candidateQuery);
  if (!validation.ok) {
    deps.setRouteError(validation.errorCode);
    return;
  }

  deps.setRouteError('');
  const nextTab = candidateQuery.tab || '';
  deps.setActiveViewPath(validation.cleanPath);
  deps.setActiveTab(nextTab);
  deps.setActiveRouteQuery(candidateQuery);

  let nextPupilId = deps.activePupilId;
  let nextGroupId = deps.activeGroupId;
  if (validation.params.pupil_id) {
    nextPupilId = validation.params.pupil_id;
    deps.setActivePupilId(nextPupilId);
    deps.setActiveEntityContext(`student:${nextPupilId}`);
  } else if (typeof msg.pupil_id === 'string' && msg.pupil_id.trim()) {
    nextPupilId = msg.pupil_id.trim();
    deps.setActivePupilId(nextPupilId);
    deps.setActiveEntityContext(`student:${nextPupilId}`);
  }
  if (validation.params.group_id) {
    nextGroupId = validation.params.group_id;
    deps.setActiveGroupId(nextGroupId);
    deps.setActiveEntityContext(`group:${nextGroupId}`);
  } else if (typeof msg.group_id === 'string' && msg.group_id.trim()) {
    nextGroupId = msg.group_id.trim();
    deps.setActiveGroupId(nextGroupId);
    deps.setActiveEntityContext(`group:${nextGroupId}`);
  }

  deps.activatePluginInKeepAlivePool(targetPlugId, validation.cleanPath, candidateQuery);
  const fullUrl = deps.syncBrowserRouteUrl(targetPlugId, validation.cleanPath, candidateQuery);

  const targetEl = deps.pluginIframeRefs.current.get(targetPlugId);
  if (targetEl?.contentWindow) {
    const querySearch = new URLSearchParams(candidateQuery).toString();
    targetEl.contentWindow.postMessage(
      {
        type: 'mis:route:changed',
        route: {
          plug_id: targetPlugId,
          view_id: validation.viewId,
          path: validation.cleanPath,
          search: querySearch ? `?${querySearch}` : '',
          url: fullUrl,
          tab: nextTab || null,
          params: validation.params,
          query: candidateQuery,
        },
        context: {
          pupil_id: nextPupilId || null,
          group_id: nextGroupId || null,
          tab: nextTab || null,
        },
      },
      '*'
    );
  }
}

export function handleLinkNavigateMessage(
  msg: Record<string, unknown>,
  sourcePluginId: string,
  deps: RouteAndLinkHandlerDeps
) {
  const entityType = typeof msg.entity_type === 'string' ? msg.entity_type.trim().toLowerCase() : '';
  const entityId = typeof msg.entity_id === 'string' ? msg.entity_id.trim() : '';
  if (!entityType || !entityId) {
    throw new Error('Malformed mis:link:navigate payload: missing entity_type or entity_id');
  }

  const requestedPlugin = typeof msg.target_plugin === 'string' ? msg.target_plugin.trim() : '';
  let designatedPlugin =
    deps.entityLinkHandlers[entityType] || deps.activePluginId || (deps.installedPlugins[0]?.id ?? '');
  if (requestedPlugin === 'self') {
    if (sourcePluginId) {
      designatedPlugin = sourcePluginId;
    }
  } else if (requestedPlugin && deps.installedPlugins.some((p) => p.id === requestedPlugin)) {
    designatedPlugin = requestedPlugin;
  } else if (requestedPlugin && deps.entityLinkHandlers[requestedPlugin]) {
    designatedPlugin = deps.entityLinkHandlers[requestedPlugin];
  }

  let nextPupilId = deps.activePupilId;
  let nextGroupId = deps.activeGroupId;
  let nextStaffId = deps.activeStaffId;
  let nextViewPath = '/';

  if (entityType === 'student' || (typeof msg.pupil_id === 'string' && msg.pupil_id.trim() !== '')) {
    nextPupilId = (typeof msg.pupil_id === 'string' && msg.pupil_id.trim()) || entityId;
    deps.setActivePupilId(nextPupilId);
    nextViewPath = `/student/${encodeURIComponent(nextPupilId)}`;
  }
  if (entityType === 'group' || (typeof msg.group_id === 'string' && msg.group_id.trim() !== '')) {
    nextGroupId = (typeof msg.group_id === 'string' && msg.group_id.trim()) || entityId;
    deps.setActiveGroupId(nextGroupId);
    if (entityType === 'group') {
      nextViewPath = `/group/${encodeURIComponent(nextGroupId)}`;
    }
  }
  if (entityType === 'staff' || (typeof msg.staff_id === 'string' && msg.staff_id.trim() !== '')) {
    nextStaffId = (typeof msg.staff_id === 'string' && msg.staff_id.trim()) || entityId;
    deps.setActiveStaffId(nextStaffId);
    if (entityType === 'staff') {
      nextViewPath = `/staff/${encodeURIComponent(nextStaffId)}`;
    }
  }

  const nextLinkQuery: Record<string, string> = {};
  if (msg.query && typeof msg.query === 'object') {
    for (const [k, v] of Object.entries(msg.query as Record<string, unknown>)) {
      if (v !== undefined && v !== null && String(v) !== '') {
        nextLinkQuery[k] = String(v);
      }
    }
  }
  if (typeof msg.tab === 'string' && msg.tab.trim() !== '') {
    nextLinkQuery.tab = msg.tab.trim();
  }

  const targetPluginObj = deps.installedPlugins.find((p) => p.id === designatedPlugin);
  let routeCheck = validatePluginViewRoute(targetPluginObj, nextViewPath, nextLinkQuery);
  if (!routeCheck.ok && nextViewPath !== '/') {
    routeCheck = validatePluginViewRoute(targetPluginObj, '/', nextLinkQuery);
  }
  const resolvedViewPath = routeCheck.ok ? routeCheck.cleanPath : '/';
  const nextTab = nextLinkQuery.tab || '';

  deps.setRouteError('');
  deps.setActiveViewPath(resolvedViewPath);
  deps.setActiveTab(nextTab);
  deps.setActiveRouteQuery(nextLinkQuery);
  deps.activatePluginInKeepAlivePool(designatedPlugin, resolvedViewPath, nextLinkQuery);
  const fullUrl = deps.syncBrowserRouteUrl(designatedPlugin, resolvedViewPath, nextLinkQuery);

  deps.setActiveEntityContext(`${entityType}:${entityId}`);
  deps.setDeepLinkNavigationStatus(`${entityType}:${entityId} -> ${designatedPlugin}`);

  deps.broadcastInitToMountedIframes({
    pupil_id: nextPupilId,
    group_id: nextGroupId,
    staff_id: nextStaffId,
    entity_type: entityType,
    entity_id: entityId,
    tab: nextTab,
    view_path: resolvedViewPath,
    route_query: nextLinkQuery,
  });

  const targetEl = deps.pluginIframeRefs.current.get(designatedPlugin);
  if (targetEl?.contentWindow) {
    const querySearch = new URLSearchParams(nextLinkQuery).toString();
    targetEl.contentWindow.postMessage(
      {
        type: 'mis:route:changed',
        route: {
          plug_id: designatedPlugin,
          view_id: routeCheck.viewId || 'home',
          path: resolvedViewPath,
          search: querySearch ? `?${querySearch}` : '',
          url: fullUrl,
          tab: nextTab || null,
          params: routeCheck.params,
          query: nextLinkQuery,
        },
        context: {
          pupil_id: nextPupilId || null,
          group_id: nextGroupId || null,
          staff_id: nextStaffId || null,
          tab: nextTab || null,
          entity_type: entityType,
          entity_id: entityId,
        },
      },
      '*'
    );
  }
}
