import { PluginNavEntry } from '../themeAndTypes';

export function checkCallerAdminOrDpo(roles: string[], scopes: string[]): boolean {
  if (roles.some((r) => r.toUpperCase() === 'ADMIN' || r.toUpperCase() === 'DPO')) {
    return true;
  }
  return scopes.includes('dal.staff.write:admin');
}

export const HOST_ALLOWED_QUERY_KEYS = new Set([
  'pupil_id',
  'second_pupil_id',
  'group_id',
  'other_group_id',
  'absent_staff_id',
  'cover_staff_id',
  'slot_id',
  'reference_date',
  'academic_year_id',
  '__persona',
  'staff_id',
]);

export type HostViewMode = 'home' | 'search' | 'settings' | 'plugin';

export interface ParsedBrowserRoute {
  hostViewMode: HostViewMode;
  plugId: string;
  viewPath: string;
  pupilIdFromPath: string;
  groupIdFromPath: string;
  staffIdFromPath: string;
  tab: string;
  pluginQuery: Record<string, string>;
}

export function parseBrowserRoute(pathname: string, search: string): ParsedBrowserRoute {
  const params = new URLSearchParams(search);
  let hostViewMode: HostViewMode = 'home';
  let plugId = '';
  let viewPath = '/';
  let pupilIdFromPath = '';
  let groupIdFromPath = '';
  let staffIdFromPath = '';

  const normalizedPath = pathname.replace(/\/+$/, '') || '/';

  if (normalizedPath === '/search') {
    hostViewMode = 'search';
  } else if (normalizedPath === '/settings') {
    hostViewMode = 'settings';
  } else if (pathname === '/p' || pathname.startsWith('/p/')) {
    hostViewMode = 'plugin';
    const trimmed = pathname.replace(/^\/p\/?/, '').replace(/\/+$/, '');
    if (trimmed) {
      const segments = trimmed.split('/');
      plugId = segments[0] || '';
      const rest = segments.slice(1);
      if (rest.length > 0) {
        viewPath = '/' + rest.join('/');
        if (rest[0] === 'student' && rest[1]) {
          pupilIdFromPath = decodeURIComponent(rest[1]);
        } else if (rest[0] === 'group' && rest[1]) {
          groupIdFromPath = decodeURIComponent(rest[1]);
        } else if (rest[0] === 'staff' && rest[1]) {
          staffIdFromPath = decodeURIComponent(rest[1]);
        }
      }
    }
  }

  const pluginQuery: Record<string, string> = {};
  params.forEach((value, key) => {
    if (!HOST_ALLOWED_QUERY_KEYS.has(key)) {
      pluginQuery[key] = value;
    }
  });

  return {
    hostViewMode,
    plugId,
    viewPath,
    pupilIdFromPath,
    groupIdFromPath,
    staffIdFromPath,
    tab: params.get('tab') || '',
    pluginQuery,
  };
}

export const DEFAULT_PLUGIN_VIEWS: NonNullable<PluginNavEntry['views']> = [
  { id: 'home', title: 'Home', path: '/', entry: 'ui/dist/index.html', entry_url: '', query_params: ['tab'] },
  {
    id: 'student',
    title: 'Student View',
    path: '/student/:pupil_id',
    entry: 'ui/dist/index.html',
    entry_url: '',
    query_params: ['tab'],
  },
  {
    id: 'group',
    title: 'Group View',
    path: '/group/:group_id',
    entry: 'ui/dist/index.html',
    entry_url: '',
    query_params: ['tab'],
  },
  {
    id: 'staff',
    title: 'Staff View',
    path: '/staff/:staff_id',
    entry: 'ui/dist/index.html',
    entry_url: '',
    query_params: ['tab'],
  },
];

export function validatePluginViewRoute(
  plugin: PluginNavEntry | undefined,
  rawSubpath: string,
  queryObj: Record<string, unknown>
): {
  ok: boolean;
  viewId: string;
  cleanPath: string;
  params: Record<string, string>;
  errorCode: string;
} {
  let cleanPath = (rawSubpath || '/').trim();
  if (!cleanPath.startsWith('/')) {
    cleanPath = '/' + cleanPath;
  }
  if (cleanPath.length > 1) {
    cleanPath = cleanPath.replace(/\/+$/, '');
  }

  const views = plugin?.views && plugin.views.length > 0 ? plugin.views : DEFAULT_PLUGIN_VIEWS;

  const reqSegs = cleanPath === '/' ? [] : cleanPath.slice(1).split('/');
  let matchedView: NonNullable<PluginNavEntry['views']>[number] | null = null;
  let extractedParams: Record<string, string> = {};

  for (const v of views) {
    let vPath = (v.path || '/').trim();
    if (!vPath.startsWith('/')) {
      vPath = '/' + vPath;
    }
    if (vPath.length > 1) {
      vPath = vPath.replace(/\/+$/, '');
    }
    const patSegs = vPath === '/' ? [] : vPath.slice(1).split('/');
    if (patSegs.length !== reqSegs.length) {
      continue;
    }
    const candidateParams: Record<string, string> = {};
    let isMatch = true;
    for (let i = 0; i < patSegs.length; i++) {
      const pSeg = patSegs[i];
      const rSeg = reqSegs[i];
      if (pSeg.startsWith(':')) {
        const paramName = pSeg.slice(1);
        if (!paramName || !rSeg) {
          isMatch = false;
          break;
        }
        candidateParams[paramName] = decodeURIComponent(rSeg);
      } else if (pSeg !== rSeg) {
        isMatch = false;
        break;
      }
    }
    if (isMatch) {
      matchedView = v;
      extractedParams = candidateParams;
      break;
    }
  }

  if (!matchedView) {
    return {
      ok: false,
      viewId: '',
      cleanPath,
      params: {},
      errorCode: `UNDECLARED_PLUGIN_VIEW_PATH: ${cleanPath} is not declared in manifest.json views`,
    };
  }

  const allowedViewQuery = new Set<string>(matchedView.query_params || []);
  for (const key of Object.keys(queryObj)) {
    if (!HOST_ALLOWED_QUERY_KEYS.has(key) && !allowedViewQuery.has(key)) {
      return {
        ok: false,
        viewId: matchedView.id,
        cleanPath,
        params: extractedParams,
        errorCode: `UNDECLARED_PLUGIN_VIEW_QUERY_PARAM: query parameter "${key}" is not declared for view "${matchedView.id}"`,
      };
    }
  }

  return {
    ok: true,
    viewId: matchedView.id,
    cleanPath,
    params: extractedParams,
    errorCode: '',
  };
}
