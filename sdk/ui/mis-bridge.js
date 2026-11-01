/**
 * Optimis School MIS — Official Sandboxed Plugin UI SDK (mis-bridge.js)
 * Architecture Blueprint: Sections 6C, 9.4 & 9.5
 *
 * Responsibilities:
 * 1. Initialization Handshake (mis:ready -> mis:init / mis:theme:update)
 * 2. CSS Custom Property (--mis-*) Injection & Light / High-Contrast Theme Sync
 * 3. Programmatic Promise RPC API: window.mis.rpc(method, args)
 * 4. Declarative DOM RPC Delegation: [data-mis-rpc] + [data-mis-args]
 * 5. Declarative Entity Deep-Linking: [data-mis-link] -> mis:link:navigate
 */
(function (global) {
  'use strict';

  var requestCounter = 0;
  var pendingRequests = Object.create(null);
  var initListeners = [];
  var routeChangedListeners = [];
  var personaChangedListeners = [];
  var invalidationListeners = [];
  var readyResolved = false;
  var readyResolveFn = null;

  var readyPromise = new Promise(function (resolve) {
    readyResolveFn = resolve;
  });

  var FONT_SIZE_PRESETS = {
    small: { px: '14px', scale: '0.875' },
    medium: { px: '16px', scale: '1' },
    large: { px: '18px', scale: '1.125' },
    'extra-large': { px: '20px', scale: '1.25' },
  };

  var state = {
    isReady: false,
    themeMode: 'light',
    baseFontSize: 'medium',
    themeTokens: Object.create(null),
    context: Object.create(null),
    route: {
      plug_id: '',
      view_id: 'home',
      path: '/',
      search: '',
      url: '/',
      tab: null,
      params: Object.create(null),
      query: Object.create(null),
    },
    permittedMethods: [],
    scopes: [],
    activeRoles: [],
  };

  function resolveBaseFontSizePreset(rawPreset, themeTokens) {
    if (typeof rawPreset === 'string') {
      var trimmed = rawPreset.trim().toLowerCase();
      if (FONT_SIZE_PRESETS[trimmed]) {
        return trimmed;
      }
    }
    if (themeTokens && typeof themeTokens['--mis-font-size-base'] === 'string') {
      var tokenPx = themeTokens['--mis-font-size-base'].trim();
      var presetKeys = Object.keys(FONT_SIZE_PRESETS);
      for (var i = 0; i < presetKeys.length; i++) {
        var k = presetKeys[i];
        if (FONT_SIZE_PRESETS[k].px === tokenPx) {
          return k;
        }
      }
    }
    return state.baseFontSize || 'medium';
  }

  function applyThemeTokens(themeMode, themeTokens, legacyTheme, baseFontSize) {
    var mode = typeof themeMode === 'string' && themeMode.trim() !== '' ? themeMode.trim() : 'light';
    state.themeMode = mode;

    var fontSizePreset = resolveBaseFontSizePreset(baseFontSize, themeTokens);
    state.baseFontSize = fontSizePreset;
    var presetMetrics = FONT_SIZE_PRESETS[fontSizePreset] || FONT_SIZE_PRESETS.medium;

    var root = document.documentElement;
    if (root) {
      root.setAttribute('data-mis-theme', mode);
      root.setAttribute('data-mis-font-size', fontSizePreset);
      if (themeTokens && typeof themeTokens === 'object') {
        var keys = Object.keys(themeTokens);
        for (var i = 0; i < keys.length; i++) {
          var key = keys[i];
          var val = themeTokens[key];
          if (typeof key === 'string' && key.indexOf('--mis-') === 0 && typeof val === 'string') {
            root.style.setProperty(key, val);
            state.themeTokens[key] = val;
          }
        }
      }
      var effectivePx = state.themeTokens['--mis-font-size-base'] || presetMetrics.px;
      var effectiveScale = state.themeTokens['--mis-font-scale'] || presetMetrics.scale;
      if (typeof baseFontSize === 'string' && FONT_SIZE_PRESETS[fontSizePreset]) {
        effectivePx = presetMetrics.px;
        effectiveScale = presetMetrics.scale;
      }
      root.style.setProperty('--mis-font-size-base', effectivePx);
      root.style.setProperty('--mis-font-scale', effectiveScale);
      root.style.fontSize = effectivePx;
      state.themeTokens['--mis-font-size-base'] = effectivePx;
      state.themeTokens['--mis-font-scale'] = effectiveScale;
      if (document.body) {
        document.body.style.fontSize = effectivePx;
        document.body.style.zoom = effectiveScale;
      }
      if (
        legacyTheme &&
        typeof legacyTheme === 'object' &&
        typeof legacyTheme.primary === 'string' &&
        legacyTheme.primary.trim() !== '' &&
        !state.themeTokens['--mis-brand-primary']
      ) {
        root.style.setProperty('--mis-brand-primary', legacyTheme.primary.trim());
        state.themeTokens['--mis-brand-primary'] = legacyTheme.primary.trim();
      }
    }
  }

  function buildSnapshot() {
    return {
      theme_mode: state.themeMode,
      base_font_size: state.baseFontSize,
      theme: Object.assign({}, state.themeTokens),
      context: Object.assign({}, state.context),
      route: {
        plug_id: state.route.plug_id || '',
        view_id: state.route.view_id || 'home',
        path: state.route.path || '/',
        search: state.route.search || '',
        url: state.route.url || '/',
        tab: state.route.tab || null,
        params: Object.assign({}, state.route.params || {}),
        query: Object.assign({}, state.route.query || {}),
      },
      permitted_methods: state.permittedMethods.slice(),
      scopes: state.scopes.slice(),
      active_roles: state.activeRoles.slice(),
    };
  }

  function notifyRouteChangedListeners() {
    var snapshot = buildSnapshot();
    for (var i = 0; i < routeChangedListeners.length; i++) {
      try {
        routeChangedListeners[i](snapshot);
      } catch (err) {
        console.error('[mis-bridge] Error in onRouteChanged listener:', err);
      }
    }
    if (typeof CustomEvent === 'function') {
      global.dispatchEvent(new CustomEvent('mis:route:changed', { detail: snapshot }));
    }
  }

  function notifyInitListeners() {
    renderIcons(document);
    var snapshot = buildSnapshot();
    if (!readyResolved && readyResolveFn) {
      readyResolved = true;
      readyResolveFn(snapshot);
    }
    for (var i = 0; i < initListeners.length; i++) {
      try {
        initListeners[i](snapshot);
      } catch (err) {
        console.error('[mis-bridge] Error in onInit listener:', err);
      }
    }
    if (typeof CustomEvent === 'function') {
      global.dispatchEvent(new CustomEvent('mis:initialized', { detail: snapshot }));
    }
  }

  function rpc(method, args, triggerElement) {
    return new Promise(function (resolve, reject) {
      if (typeof method !== 'string' || method.trim() === '') {
        var missingErr = new Error('MISSING_RPC_METHOD: method name must be a non-empty string');
        missingErr.code = 'MISSING_RPC_METHOD';
        reject(missingErr);
        return;
      }

      var normalizedMethod = method.trim();
      var payloadArgs = args === undefined ? {} : args;
      if (payloadArgs === null || typeof payloadArgs !== 'object' || Array.isArray(payloadArgs)) {
        var argsErr = new Error('INVALID_RPC_ARGS: args must be a plain JSON object');
        argsErr.code = 'INVALID_RPC_ARGS';
        reject(argsErr);
        return;
      }

      requestCounter += 1;
      var reqId = 'req_' + requestCounter;
      pendingRequests[reqId] = {
        method: normalizedMethod,
        args: payloadArgs,
        resolve: resolve,
        reject: reject,
        element: triggerElement || null,
      };

      if (triggerElement && typeof triggerElement.setAttribute === 'function') {
        triggerElement.setAttribute('data-mis-loading', 'true');
      }

      global.parent.postMessage(
        {
          type: 'mis:rpc:request',
          request_id: reqId,
          method: normalizedMethod,
          args: payloadArgs,
        },
        '*'
      );
    });
  }

  function setRouteState(nextState, options) {
    if (!nextState || typeof nextState !== 'object') {
      throw new Error('INVALID_ROUTE_STATE: nextState must be an object');
    }

    var entityType = typeof nextState.entity_type === 'string' ? nextState.entity_type.trim().toLowerCase() : '';
    var pupilId = typeof nextState.pupil_id === 'string' ? nextState.pupil_id.trim() : '';
    var groupId = typeof nextState.group_id === 'string' ? nextState.group_id.trim() : '';
    var staffId = typeof nextState.staff_id === 'string' ? nextState.staff_id.trim() : '';
    var entityId =
      (typeof nextState.entity_id === 'string' && nextState.entity_id.trim()) ||
      pupilId ||
      groupId ||
      staffId;
    var tab = typeof nextState.tab === 'string' && nextState.tab.trim() !== '' ? nextState.tab.trim() : '';
    var customPath = typeof nextState.path === 'string' && nextState.path.trim() !== '' ? nextState.path.trim() : '';

    if (pupilId) {
      state.context.pupil_id = pupilId;
      if (!entityType) {
        entityType = 'student';
      }
    }
    if (groupId) {
      state.context.group_id = groupId;
      if (!entityType) {
        entityType = 'group';
      }
    }
    if (staffId) {
      state.context.staff_id = staffId;
      if (!entityType) {
        entityType = 'staff';
      }
    }
    if (entityType) {
      state.context.entity_type = entityType;
    }
    if (entityId) {
      state.context.entity_id = entityId;
    }

    var nextQuery = Object.assign({}, state.route.query || {});
    if (nextState.query && typeof nextState.query === 'object') {
      var qKeys = Object.keys(nextState.query);
      for (var i = 0; i < qKeys.length; i++) {
        var k = qKeys[i];
        var v = nextState.query[k];
        if (v === null || v === undefined || v === '') {
          delete nextQuery[k];
        } else {
          nextQuery[k] = String(v);
        }
      }
    }
    if (tab) {
      nextQuery.tab = tab;
      state.context.tab = tab;
      state.route.tab = tab;
    } else if (nextState.tab === null || nextState.tab === '') {
      delete nextQuery.tab;
      state.context.tab = null;
      state.route.tab = null;
    }

    var resolvedPath = customPath;
    if (!resolvedPath) {
      var effType = entityType || state.context.entity_type || '';
      var effId =
        entityId ||
        (effType === 'student' ? state.context.pupil_id : '') ||
        (effType === 'group' ? state.context.group_id : '') ||
        state.context.entity_id ||
        '';
      if ((effType === 'student' || effType === 'group' || effType === 'staff') && effId && effId !== 'home') {
        resolvedPath = '/' + effType + '/' + effId;
      } else {
        resolvedPath = state.route.path || '/';
      }
    }

    state.route.path = resolvedPath;
    state.route.query = nextQuery;

    var message = {
      type: 'mis:route:state',
      path: resolvedPath,
      entity_type: entityType || state.context.entity_type || undefined,
      entity_id: entityId || state.context.entity_id || undefined,
      pupil_id: pupilId || state.context.pupil_id || undefined,
      group_id: groupId || state.context.group_id || undefined,
      staff_id: staffId || undefined,
      tab: state.route.tab || undefined,
      query: nextQuery,
      replace: Boolean(options && options.replace),
    };

    global.parent.postMessage(message, '*');
    notifyRouteChangedListeners();
    return message;
  }

  function navigate(entityType, params) {
    if (typeof entityType !== 'string' || entityType.trim() === '') {
      throw new Error('MISSING_DEEP_LINK_TYPE: entityType must be a non-empty string');
    }
    if (!params || typeof params !== 'object') {
      throw new Error('MISSING_DEEP_LINK_PARAMS: params must be an object');
    }

    var normalizedType = entityType.trim().toLowerCase();
    var pupilId = typeof params.pupil_id === 'string' ? params.pupil_id.trim() : '';
    var groupId = typeof params.group_id === 'string' ? params.group_id.trim() : '';
    var staffId = typeof params.staff_id === 'string' ? params.staff_id.trim() : '';
    var entityId =
      (typeof params.entity_id === 'string' && params.entity_id.trim()) ||
      pupilId ||
      groupId ||
      staffId;

    if (!entityId) {
      throw new Error(
        'MISSING_DEEP_LINK_TARGET: data-mis-link requires a non-empty entity identifier (pupil_id, group_id, staff_id, or entity_id)'
      );
    }

    var targetPlugin =
      (typeof params.target_plugin === 'string' && params.target_plugin.trim()) ||
      (typeof params.plugin_id === 'string' && params.plugin_id.trim()) ||
      '';
    var tab = typeof params.tab === 'string' && params.tab.trim() !== '' ? params.tab.trim() : '';

    var message = {
      type: 'mis:link:navigate',
      entity_type: normalizedType,
      entity_id: entityId,
      pupil_id: pupilId || (normalizedType === 'student' ? entityId : undefined),
      group_id: groupId || (normalizedType === 'group' ? entityId : undefined),
      staff_id: staffId || (normalizedType === 'staff' ? entityId : undefined),
      target_plugin: targetPlugin || undefined,
      tab: tab || undefined,
    };

    global.parent.postMessage(message, '*');
    return message;
  }

  global.addEventListener('message', function (event) {
    var msg = event.data;
    if (!msg || typeof msg !== 'object') {
      return;
    }

    if (msg.type === 'mis:init' || msg.type === 'mis:persona:changed' || msg.type === 'mis:route:update') {
      var prevStaffId =
        state.context && typeof state.context.staff_id === 'string'
          ? state.context.staff_id
          : '';
      var prevUrl = state.route && typeof state.route.url === 'string' ? state.route.url : '';
      var wasReady = state.isReady;
      state.isReady = true;
      state.context =
        msg.context && typeof msg.context === 'object'
          ? Object.assign({}, msg.context)
          : Object.create(null);
      if (msg.route && typeof msg.route === 'object') {
        state.route = {
          plug_id: typeof msg.route.plug_id === 'string' ? msg.route.plug_id : '',
          view_id: typeof msg.route.view_id === 'string' ? msg.route.view_id : 'home',
          path: typeof msg.route.path === 'string' ? msg.route.path : '/',
          search: typeof msg.route.search === 'string' ? msg.route.search : '',
          url: typeof msg.route.url === 'string' ? msg.route.url : '/',
          tab: typeof msg.route.tab === 'string' && msg.route.tab !== '' ? msg.route.tab : null,
          params:
            msg.route.params && typeof msg.route.params === 'object'
              ? Object.assign({}, msg.route.params)
              : Object.create(null),
          query:
            msg.route.query && typeof msg.route.query === 'object'
              ? Object.assign({}, msg.route.query)
              : Object.create(null),
        };
      }
      state.permittedMethods = Array.isArray(msg.permitted_methods)
        ? msg.permitted_methods.slice()
        : [];
      state.scopes = Array.isArray(msg.scopes)
        ? msg.scopes.slice()
        : msg.caller && Array.isArray(msg.caller.scopes)
          ? msg.caller.scopes.slice()
          : [];
      state.activeRoles = Array.isArray(msg.active_roles)
        ? msg.active_roles.slice()
        : msg.caller && Array.isArray(msg.caller.active_roles)
          ? msg.caller.active_roles.slice()
          : [];
      applyThemeTokens(msg.theme_mode, msg.theme_tokens, msg.theme, msg.base_font_size);
      notifyInitListeners();
      if (msg.type === 'mis:route:update' || (wasReady && prevUrl !== state.route.url)) {
        notifyRouteChangedListeners();
      }
      var nextStaffId =
        state.context && typeof state.context.staff_id === 'string'
          ? state.context.staff_id
          : '';
      if (msg.type === 'mis:persona:changed' || (wasReady && prevStaffId !== nextStaffId)) {
        var snapshot = buildSnapshot();
        for (var pIdx = 0; pIdx < personaChangedListeners.length; pIdx++) {
          try {
            personaChangedListeners[pIdx](snapshot);
          } catch (pErr) {
            console.error('[mis-bridge] Error in onPersonaChanged listener:', pErr);
          }
        }
        if (typeof CustomEvent === 'function') {
          global.dispatchEvent(new CustomEvent('mis:persona:changed', { detail: snapshot }));
        }
      }
      return;
    }

    if (msg.type === 'mis:theme:update') {
      applyThemeTokens(msg.theme_mode, msg.theme_tokens, msg.theme, msg.base_font_size);
      notifyInitListeners();
      return;
    }

    if (msg.type === 'mis:plugin:invalidated') {
      for (var j = 0; j < invalidationListeners.length; j++) {
        try {
          invalidationListeners[j](msg);
        } catch (err) {
          console.error('[mis-bridge] Error in onInvalidated listener:', err);
        }
      }
      if (typeof CustomEvent === 'function') {
        global.dispatchEvent(new CustomEvent('mis:plugin:invalidated', { detail: msg }));
      }
      return;
    }

    if (msg.type === 'mis:rpc:response') {
      var reqId = msg.request_id;
      var pending = reqId ? pendingRequests[reqId] : null;
      if (pending) {
        delete pendingRequests[reqId];
        if (pending.element && typeof pending.element.removeAttribute === 'function') {
          pending.element.removeAttribute('data-mis-loading');
        }
      }

      var invokedMethod = pending ? pending.method : msg.method || '';
      var dispatchTarget = (pending && pending.element) || document;

      if (msg.ok) {
        var detailOk = {
          request_id: reqId,
          method: invokedMethod,
          args: pending ? pending.args : {},
          data: msg.data,
        };
        if (typeof CustomEvent === 'function') {
          dispatchTarget.dispatchEvent(
            new CustomEvent('mis:rpc:success', { bubbles: true, detail: detailOk })
          );
        }
        if (pending) {
          pending.resolve(msg.data);
        }
      } else {
        var errCode =
          typeof msg.error_code === 'string' && msg.error_code.trim() !== ''
            ? msg.error_code.trim()
            : 'UNKNOWN_RPC_ERROR';
        var rpcErr = new Error(errCode);
        rpcErr.code = errCode;
        rpcErr.request_id = reqId;
        rpcErr.method = invokedMethod;

        var detailErr = {
          request_id: reqId,
          method: invokedMethod,
          args: pending ? pending.args : {},
          error_code: errCode,
        };
        if (typeof CustomEvent === 'function') {
          dispatchTarget.dispatchEvent(
            new CustomEvent('mis:rpc:error', { bubbles: true, detail: detailErr })
          );
        }
        if (pending) {
          pending.reject(rpcErr);
        }
      }
    }
  });

  document.addEventListener('click', function (event) {
    if (!event.target || typeof event.target.closest !== 'function') {
      return;
    }

    // 1. Declarative Route Tab Switching: [data-mis-route-tab]
    var tabTarget = event.target.closest('[data-mis-route-tab]');
    if (tabTarget) {
      event.preventDefault();
      var tabVal = tabTarget.getAttribute('data-mis-route-tab') || '';
      if (tabVal.trim() !== '') {
        setRouteState({ tab: tabVal.trim() });
      }
      return;
    }

    // 2. Declarative Entity Deep-Linking: [data-mis-link] (Section 9.5 / 9C)
    var linkTarget = event.target.closest('[data-mis-link]');
    if (linkTarget) {
      event.preventDefault();
      var entityType = linkTarget.getAttribute('data-mis-link');
      var pupilId = linkTarget.getAttribute('data-pupil-id') || '';
      var groupId = linkTarget.getAttribute('data-group-id') || '';
      var staffId = linkTarget.getAttribute('data-staff-id') || '';
      var entityId = linkTarget.getAttribute('data-entity-id') || '';
      var targetPlugin = linkTarget.getAttribute('data-mis-plugin') || '';
      var linkTab = linkTarget.getAttribute('data-mis-tab') || '';

      navigate(entityType || '', {
        entity_id: entityId,
        pupil_id: pupilId,
        group_id: groupId,
        staff_id: staffId,
        target_plugin: targetPlugin,
        tab: linkTab,
      });
      return;
    }

    // 3. Declarative RPC Invocation: [data-mis-rpc] (Sections 6C & 9.4)
    var rpcTarget = event.target.closest('[data-mis-rpc]');
    if (rpcTarget) {
      var method = rpcTarget.getAttribute('data-mis-rpc');
      var rawArgs = rpcTarget.getAttribute('data-mis-args');
      if (!method || method.trim() === '' || rawArgs === null || rawArgs.trim() === '') {
        throw new Error('Missing mandatory data-mis-rpc or data-mis-args attribute');
      }

      var parsedArgs;
      try {
        parsedArgs = JSON.parse(rawArgs);
      } catch (parseErr) {
        throw new Error('Invalid JSON in data-mis-args attribute: ' + parseErr.message);
      }

      rpc(method, parsedArgs, rpcTarget).catch(function () {
        // Declarative click callers handle errors via mis:rpc:error event or message listener;
        // prevent unhandled rejection pageerror on declarative clicks while preserving Promise rejection for programmatic callers.
      });
    }
  });

  var LUCIDE_ICONS = {
    award:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="award" aria-hidden="true"><circle cx="12" cy="8" r="6"/><path d="M15.477 12.89 17 22l-5-3-5 3 1.523-9.11"/></svg>',
    calendar:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="calendar" aria-hidden="true"><rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/></svg>',
    'book-open':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="book-open" aria-hidden="true"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>',
    'shield-alert':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="shield-alert" aria-hidden="true"><path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/><path d="M12 8v4"/><path d="M12 16h.01"/></svg>',
    'heart-pulse':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="heart-pulse" aria-hidden="true"><path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/><path d="M3.22 12H9.5l.5-1 2 4.5 2-7 1.5 3.5h5.27"/></svg>',
    bus:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="bus" aria-hidden="true"><path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/></svg>',
    users:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="users" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',
    user:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="user" aria-hidden="true"><path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>',
    'user-check':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="user-check" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><polyline points="16 11 18 13 22 9"/></svg>',
    calculator:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="calculator" aria-hidden="true"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',
    'file-check':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="file-check" aria-hidden="true"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="m9 15 2 2 4-4"/></svg>',
    box:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="box" aria-hidden="true"><path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/></svg>',
    stethoscope:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="stethoscope" aria-hidden="true"><path d="M4.8 2.3A.3.3 0 1 0 5 2H4a2 2 0 0 0-2 2v5a6 6 0 0 0 6 6v0a6 6 0 0 0 6-6V4a2 2 0 0 0-2-2h-1a.2.2 0 1 0 .3.3"/><path d="M8 15v1a6 6 0 0 0 6 6v0a6 6 0 0 0 6-6v-4"/><circle cx="20" cy="10" r="2"/></svg>',
    'refresh-cw':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="refresh-cw" aria-hidden="true"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>',
    inbox:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="inbox" aria-hidden="true"><polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/></svg>',
    house:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="house" aria-hidden="true"><path d="M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8"/><path d="M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>',
    home:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="home" aria-hidden="true"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>',
    search:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="search" aria-hidden="true"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',
    pin:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="pin" aria-hidden="true"><path d="M12 17v5"/><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H8a2 2 0 0 0 0 4 1 1 0 0 1 1 1z"/></svg>',
    settings:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="settings" aria-hidden="true"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',
    'log-out':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="log-out" aria-hidden="true"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" x2="9" y1="12" y2="12"/></svg>',
    'layout-grid':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="layout-grid" aria-hidden="true"><rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/></svg>',
    'clipboard-check':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="clipboard-check" aria-hidden="true"><rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="m9 14 2 2 4-4"/></svg>',
    'graduation-cap':
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="graduation-cap" aria-hidden="true"><path d="M21.42 10.922a1 1 0 0 0-.019-1.838L12.83 5.18a2 2 0 0 0-1.66 0L2.6 9.08a1 1 0 0 0 0 1.832l8.57 3.908a2 2 0 0 0 1.66 0z"/><path d="M22 10v6"/><path d="M6 12.5V16a6 3 0 0 0 12 0v-3.5"/></svg>',
    activity:
      '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="activity" aria-hidden="true"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>',
  };

  function getLucideSvg(iconName) {
    var key = typeof iconName === 'string' ? iconName.trim().toLowerCase() : '';
    if (!key || !LUCIDE_ICONS[key]) {
      throw new Error('UNKNOWN_LUCIDE_ICON: unsupported monochrome Lucide icon "' + iconName + '"');
    }
    return LUCIDE_ICONS[key];
  }

  function renderIcons(rootNode) {
    var scope = rootNode && typeof rootNode.querySelectorAll === 'function' ? rootNode : document;
    if (!scope || typeof scope.querySelectorAll !== 'function') {
      return;
    }
    var nodes = scope.querySelectorAll('[data-mis-icon]');
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      var iconName = el.getAttribute('data-mis-icon');
      if (iconName && iconName.trim() !== '') {
        var key = iconName.trim().toLowerCase();
        if (LUCIDE_ICONS[key]) {
          el.innerHTML = LUCIDE_ICONS[key];
        }
      }
    }
  }

  var misSDK = {
    version: '1.0.0',
    get isReady() {
      return state.isReady;
    },
    get themeMode() {
      return state.themeMode;
    },
    get theme() {
      return Object.assign({}, state.themeTokens);
    },
    get context() {
      return Object.assign({}, state.context);
    },
    get route() {
      return buildSnapshot().route;
    },
    get permittedMethods() {
      return state.permittedMethods.slice();
    },
    get scopes() {
      return state.scopes.slice();
    },
    get activeRoles() {
      return state.activeRoles.slice();
    },
    ready: function () {
      return readyPromise;
    },
    onInit: function (callback) {
      if (typeof callback !== 'function') {
        throw new Error('window.mis.onInit requires a function callback');
      }
      initListeners.push(callback);
      if (state.isReady) {
        callback(buildSnapshot());
      }
    },
    onRouteChanged: function (callback) {
      if (typeof callback !== 'function') {
        throw new Error('window.mis.onRouteChanged requires a function callback');
      }
      routeChangedListeners.push(callback);
    },
    onPersonaChanged: function (callback) {
      if (typeof callback !== 'function') {
        throw new Error('window.mis.onPersonaChanged requires a function callback');
      }
      personaChangedListeners.push(callback);
    },
    onInvalidated: function (callback) {
      if (typeof callback !== 'function') {
        throw new Error('window.mis.onInvalidated requires a function callback');
      }
      invalidationListeners.push(callback);
    },
    rpc: function (method, args) {
      return rpc(method, args, null);
    },
    useMisRpc: function (method, defaultArgs) {
      if (typeof method !== 'string' || method.trim() === '') {
        throw new Error('MISSING_RPC_METHOD: useMisRpc requires a non-empty method name');
      }
      var normalizedMethod = method.trim();
      var baseArgs =
        defaultArgs && typeof defaultArgs === 'object' && !Array.isArray(defaultArgs)
          ? Object.assign({}, defaultArgs)
          : {};
      return {
        method: normalizedMethod,
        call: function (overrideArgs) {
          var finalArgs =
            overrideArgs !== undefined
              ? overrideArgs
              : Object.assign({}, baseArgs);
          return rpc(normalizedMethod, finalArgs, null);
        },
      };
    },
    navigate: navigate,
    setRouteState: setRouteState,
    getLucideSvg: getLucideSvg,
    renderIcons: renderIcons,
  };

  Object.defineProperty(global, 'mis', {
    value: misSDK,
    writable: false,
    configurable: false,
  });

  function sendReady() {
    renderIcons(document);
    global.parent.postMessage({ type: 'mis:ready' }, '*');
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', sendReady);
  } else {
    sendReady();
  }
})(window);
