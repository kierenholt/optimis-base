/**
 * Optimis School MIS — Official Sandboxed Plugin UI SDK (mis-bridge.js)
 * Architecture Blueprint: Sections 6C, 9.4 & 9.5
 * Responsibilities: Handshake, Theme/Tokens, Typography Scaling, RPC, Linking, Icons, Auto-Resize
 */
(function (global) {
  'use strict';

  var requestCounter = 0, pendingRequests = Object.create(null);
  var initListeners = [], routeChangedListeners = [], personaChangedListeners = [], invalidationListeners = [];
  var readyResolved = false, readyResolveFn = null;
  var readyPromise = new Promise(function (resolve) { readyResolveFn = resolve; });

  var FONT_SIZE_PRESETS = {
    small: { px: '14px', scale: '0.875' }, medium: { px: '16px', scale: '1' },
    large: { px: '18px', scale: '1.125' }, 'extra-large': { px: '20px', scale: '1.25' },
  };

  var DECLARATIVE_ATTRIBUTES = {
    RPC_METHOD: 'data-mis-rpc', RPC_ARGS: 'data-mis-args', ENTITY_LINK: 'data-mis-link',
    PUPIL_ID: 'data-pupil-id', GROUP_ID: 'data-group-id', STAFF_ID: 'data-staff-id',
    PLUGIN_TARGET: 'data-mis-plugin', ROUTE_TAB: 'data-mis-route-tab',
  };

  var state = {
    isReady: false, themeMode: 'light', baseFontSize: 'medium',
    themeTokens: Object.create(null), context: Object.create(null),
    route: { plug_id: '', view_id: 'home', path: '/', search: '', url: '/', tab: null, params: Object.create(null), query: Object.create(null) },
    permittedMethods: [], scopes: [], activeRoles: [],
  };

  function resolveBaseFontSizePreset(rawPreset, themeTokens) {
    if (typeof rawPreset === 'string' && FONT_SIZE_PRESETS[rawPreset.trim().toLowerCase()]) return rawPreset.trim().toLowerCase();
    if (themeTokens && typeof themeTokens['--mis-font-size-base'] === 'string') {
      var tokenPx = themeTokens['--mis-font-size-base'].trim(), keys = Object.keys(FONT_SIZE_PRESETS);
      for (var i = 0; i < keys.length; i++) { if (FONT_SIZE_PRESETS[keys[i]].px === tokenPx) return keys[i]; }
    }
    return state.baseFontSize || 'medium';
  }

  function scaleDocumentTypography(rootNode) {
    var scope = rootNode && typeof rootNode.querySelectorAll === 'function' ? rootNode : document;
    if (!scope || typeof scope.querySelectorAll !== 'function') return;
    var styleEls = scope.querySelectorAll('style:not([data-mis-font-scaled])');
    for (var i = 0; i < styleEls.length; i++) {
      var st = styleEls[i], css = st.textContent || '';
      if (css.indexOf('font-size') !== -1) st.textContent = css.replace(/font-size\s*:\s*(\d+(?:\.\d+)?)px/gi, 'font-size: calc($1px * var(--mis-font-scale, 1))');
      st.setAttribute('data-mis-font-scaled', 'true');
    }
    var inlineEls = scope.querySelectorAll('[style*="font-size"]:not(html):not(body)');
    for (var j = 0; j < inlineEls.length; j++) {
      var el = inlineEls[j]; if (!el.style) continue;
      var m = (el.style.fontSize || '').trim().match(/^(\d+(?:\.\d+)?)px$/i);
      if (m) el.style.fontSize = 'calc(' + m[1] + 'px * var(--mis-font-scale, 1))';
    }
  }

  function applyThemeTokens(themeMode, themeTokens, legacyTheme, baseFontSize) {
    var mode = typeof themeMode === 'string' && themeMode.trim() !== '' ? themeMode.trim() : 'light';
    state.themeMode = mode;
    var fontSizePreset = resolveBaseFontSizePreset(baseFontSize, themeTokens);
    state.baseFontSize = fontSizePreset;
    var presetMetrics = FONT_SIZE_PRESETS[fontSizePreset] || FONT_SIZE_PRESETS.medium, root = document.documentElement;
    if (!root) return;
    root.setAttribute('data-mis-theme', mode); root.setAttribute('data-mis-font-size', fontSizePreset);
    if (themeTokens && typeof themeTokens === 'object') {
      var keys = Object.keys(themeTokens);
      for (var i = 0; i < keys.length; i++) {
        var k = keys[i], val = themeTokens[k];
        if (typeof k === 'string' && k.indexOf('--mis-') === 0 && typeof val === 'string') {
          root.style.setProperty(k, val); state.themeTokens[k] = val;
        }
      }
    }
    var effPx = (typeof baseFontSize === 'string' && FONT_SIZE_PRESETS[fontSizePreset]) ? presetMetrics.px : (state.themeTokens['--mis-font-size-base'] || presetMetrics.px);
    var effScale = (typeof baseFontSize === 'string' && FONT_SIZE_PRESETS[fontSizePreset]) ? presetMetrics.scale : (state.themeTokens['--mis-font-scale'] || presetMetrics.scale);
    root.style.setProperty('--mis-font-size-base', effPx); root.style.setProperty('--mis-font-scale', effScale); root.style.fontSize = effPx;
    state.themeTokens['--mis-font-size-base'] = effPx; state.themeTokens['--mis-font-scale'] = effScale;
    if (document.body) document.body.style.fontSize = effPx;
    scaleDocumentTypography(document);
    if (legacyTheme && typeof legacyTheme === 'object' && typeof legacyTheme.primary === 'string' && legacyTheme.primary.trim() !== '' && !state.themeTokens['--mis-brand-primary']) {
      root.style.setProperty('--mis-brand-primary', legacyTheme.primary.trim());
      state.themeTokens['--mis-brand-primary'] = legacyTheme.primary.trim();
    }
  }

  var lastReportedHeight = 0, lastInnerWidth = global.innerWidth || 0;

  function computeRequiredContentHeight() {
    var docEl = document.documentElement, body = document.body;
    if (!docEl || !body || ((docEl.clientWidth || 0) <= 0 && (docEl.clientHeight || 0) <= 0)) return 0;
    var scale = parseFloat(state.themeTokens['--mis-font-scale'] || '1');
    if (!Number.isFinite(scale) || scale <= 0) scale = 1;
    var bStyle = global.getComputedStyle(body);
    var marginTop = parseFloat(bStyle.marginTop) || 0, marginBottom = parseFloat(bStyle.marginBottom) || 0;
    var paddingTop = parseFloat(bStyle.paddingTop) || 0, paddingBottom = parseFloat(bStyle.paddingBottom) || 0;
    var bodyRect = body.getBoundingClientRect(), maxChildRelBottom = 0, maxChildVpBottom = 0;
    var ignoredTags = { SCRIPT: true, STYLE: true, LINK: true, META: true, TEMPLATE: true, NOSCRIPT: true }, children = body.children;
    for (var i = 0; i < children.length; i++) {
      var ch = children[i];
      if (!ch || ignoredTags[ch.tagName]) continue;
      var cStyle = global.getComputedStyle(ch);
      if (cStyle.display === 'none' || cStyle.visibility === 'hidden') continue;
      var cRect = ch.getBoundingClientRect(), cMb = parseFloat(cStyle.marginBottom) || 0;
      var relBottom = cRect.bottom - bodyRect.top + cMb;
      if (relBottom > maxChildRelBottom) maxChildRelBottom = relBottom;
      var vpBottom = Math.ceil(cRect.bottom + cMb + marginBottom);
      if (vpBottom > maxChildVpBottom) maxChildVpBottom = vpBottom;
    }
    var intrinsic = maxChildRelBottom > 0 ? Math.ceil(maxChildRelBottom + paddingTop + paddingBottom + marginTop + marginBottom) : Math.ceil((bodyRect.height || 0) + marginTop + marginBottom);
    if (intrinsic <= 0) return 0;
    var zoom = Math.max(1, scale);
    return Math.max(Math.ceil(intrinsic * zoom), Math.ceil(maxChildVpBottom * zoom)) + 32;
  }

  function reportContentHeight() {
    try {
      var nextHeight = computeRequiredContentHeight();
      if (nextHeight <= 0 || Math.abs(nextHeight - lastReportedHeight) <= 2) return;
      lastReportedHeight = nextHeight;
      global.parent.postMessage({ type: 'mis:resize', height: nextHeight }, '*');
    } catch (_err) {}
  }

  function scheduleReportContentHeight() {
    reportContentHeight();
    if (typeof global.requestAnimationFrame === 'function') global.requestAnimationFrame(reportContentHeight);
    global.setTimeout(reportContentHeight, 30);
  }

  function buildSnapshot() {
    return {
      theme_mode: state.themeMode, base_font_size: state.baseFontSize,
      theme: Object.assign({}, state.themeTokens), context: Object.assign({}, state.context),
      route: {
        plug_id: state.route.plug_id || '', view_id: state.route.view_id || 'home',
        path: state.route.path || '/', search: state.route.search || '', url: state.route.url || '/',
        tab: state.route.tab || null, params: Object.assign({}, state.route.params || {}), query: Object.assign({}, state.route.query || {}),
      },
      permitted_methods: state.permittedMethods.slice(), scopes: state.scopes.slice(), active_roles: state.activeRoles.slice(),
    };
  }

  function notifyRouteChangedListeners() {
    var snapshot = buildSnapshot();
    for (var i = 0; i < routeChangedListeners.length; i++) {
      try { routeChangedListeners[i](snapshot); } catch (err) { console.error('[mis-bridge] route listener error:', err); }
    }
    if (typeof CustomEvent === 'function') global.dispatchEvent(new CustomEvent('mis:route:changed', { detail: snapshot }));
    scheduleReportContentHeight();
  }

  function notifyInitListeners() {
    renderIcons(document);
    var snapshot = buildSnapshot();
    if (!readyResolved && readyResolveFn) { readyResolved = true; readyResolveFn(snapshot); }
    for (var i = 0; i < initListeners.length; i++) {
      try { initListeners[i](snapshot); } catch (err) { console.error('[mis-bridge] init listener error:', err); }
    }
    if (typeof CustomEvent === 'function') global.dispatchEvent(new CustomEvent('mis:initialized', { detail: snapshot }));
    scheduleReportContentHeight();
  }

  function rpc(method, args, triggerElement) {
    return new Promise(function (resolve, reject) {
      if (typeof method !== 'string' || method.trim() === '') {
        var mErr = new Error('MISSING_RPC_METHOD: method name must be a non-empty string');
        mErr.code = 'MISSING_RPC_METHOD'; return reject(mErr);
      }
      var normMethod = method.trim(), pArgs = args === undefined ? {} : args;
      if (pArgs === null || typeof pArgs !== 'object' || Array.isArray(pArgs)) {
        var aErr = new Error('INVALID_RPC_ARGS: args must be a plain JSON object');
        aErr.code = 'INVALID_RPC_ARGS'; return reject(aErr);
      }
      requestCounter += 1;
      var reqId = 'req_' + requestCounter;
      pendingRequests[reqId] = { method: normMethod, args: pArgs, resolve: resolve, reject: reject, element: triggerElement || null };
      if (triggerElement && typeof triggerElement.setAttribute === 'function') triggerElement.setAttribute('data-mis-loading', 'true');
      global.parent.postMessage({ type: 'mis:rpc:request', request_id: reqId, method: normMethod, args: pArgs }, '*');
    });
  }

  function setRouteState(nextState, options) {
    if (!nextState || typeof nextState !== 'object') throw new Error('INVALID_ROUTE_STATE: nextState must be an object');
    var entityType = typeof nextState.entity_type === 'string' ? nextState.entity_type.trim().toLowerCase() : '';
    var pupilId = typeof nextState.pupil_id === 'string' ? nextState.pupil_id.trim() : '';
    var groupId = typeof nextState.group_id === 'string' ? nextState.group_id.trim() : '';
    var staffId = typeof nextState.staff_id === 'string' ? nextState.staff_id.trim() : '';
    var entityId = (typeof nextState.entity_id === 'string' && nextState.entity_id.trim()) || pupilId || groupId || staffId;
    var tab = typeof nextState.tab === 'string' && nextState.tab.trim() !== '' ? nextState.tab.trim() : '';
    var customPath = typeof nextState.path === 'string' && nextState.path.trim() !== '' ? nextState.path.trim() : '';

    if (pupilId) { state.context.pupil_id = pupilId; if (!entityType) entityType = 'student'; }
    if (groupId) { state.context.group_id = groupId; if (!entityType) entityType = 'group'; }
    if (staffId) { state.context.staff_id = staffId; if (!entityType) entityType = 'staff'; }
    if (entityType) state.context.entity_type = entityType;
    if (entityId) state.context.entity_id = entityId;

    var nextQuery = Object.assign({}, state.route.query || {});
    if (nextState.query && typeof nextState.query === 'object') {
      var qKeys = Object.keys(nextState.query);
      for (var i = 0; i < qKeys.length; i++) {
        var k = qKeys[i], v = nextState.query[k];
        if (v === null || v === undefined || v === '') delete nextQuery[k]; else nextQuery[k] = String(v);
      }
    }
    if (tab) { nextQuery.tab = tab; state.context.tab = tab; state.route.tab = tab; }
    else if (nextState.tab === null || nextState.tab === '') { delete nextQuery.tab; state.context.tab = null; state.route.tab = null; }

    var resolvedPath = customPath;
    if (!resolvedPath) {
      var effType = entityType || state.context.entity_type || '';
      var effId = entityId || (effType === 'student' ? state.context.pupil_id : '') || (effType === 'group' ? state.context.group_id : '') || state.context.entity_id || '';
      resolvedPath = ((effType === 'student' || effType === 'group' || effType === 'staff') && effId && effId !== 'home') ? ('/' + effType + '/' + effId) : (state.route.path || '/');
    }
    state.route.path = resolvedPath; state.route.query = nextQuery;

    var msg = {
      type: 'mis:route:state', path: resolvedPath,
      entity_type: entityType || state.context.entity_type || undefined,
      entity_id: entityId || state.context.entity_id || undefined,
      pupil_id: pupilId || state.context.pupil_id || undefined,
      group_id: groupId || state.context.group_id || undefined,
      staff_id: staffId || undefined, tab: state.route.tab || undefined,
      query: nextQuery, replace: Boolean(options && options.replace),
    };
    global.parent.postMessage(msg, '*');
    notifyRouteChangedListeners();
    return msg;
  }

  function navigate(entityType, params) {
    if (typeof entityType !== 'string' || entityType.trim() === '') throw new Error('MISSING_DEEP_LINK_TYPE: entityType must be a non-empty string');
    if (!params || typeof params !== 'object') throw new Error('MISSING_DEEP_LINK_PARAMS: params must be an object');
    var normType = entityType.trim().toLowerCase();
    var pId = typeof params.pupil_id === 'string' ? params.pupil_id.trim() : '';
    var gId = typeof params.group_id === 'string' ? params.group_id.trim() : '';
    var sId = typeof params.staff_id === 'string' ? params.staff_id.trim() : '';
    var eId = (typeof params.entity_id === 'string' && params.entity_id.trim()) || pId || gId || sId;
    if (!eId) throw new Error('MISSING_DEEP_LINK_TARGET: data-mis-link requires a non-empty entity identifier (pupil_id, group_id, staff_id, or entity_id)');
    var targetPlug = (typeof params.target_plugin === 'string' && params.target_plugin.trim()) || (typeof params.plugin_id === 'string' && params.plugin_id.trim()) || '';
    var tab = typeof params.tab === 'string' && params.tab.trim() !== '' ? params.tab.trim() : '';
    var msg = {
      type: 'mis:link:navigate', entity_type: normType, entity_id: eId,
      pupil_id: pId || (normType === 'student' ? eId : undefined),
      group_id: gId || (normType === 'group' ? eId : undefined),
      staff_id: sId || (normType === 'staff' ? eId : undefined),
      target_plugin: targetPlug || undefined, tab: tab || undefined,
    };
    global.parent.postMessage(msg, '*');
    return msg;
  }

  global.addEventListener('message', function (event) {
    var msg = event.data;
    if (!msg || typeof msg !== 'object') return;

    if (msg.type === 'mis:init' || msg.type === 'mis:persona:changed' || msg.type === 'mis:route:update') {
      var prevStaffId = (state.context && typeof state.context.staff_id === 'string') ? state.context.staff_id : '';
      var prevUrl = (state.route && typeof state.route.url === 'string') ? state.route.url : '', wasReady = state.isReady;
      state.isReady = true;
      state.context = msg.context && typeof msg.context === 'object' ? Object.assign({}, msg.context) : Object.create(null);
      if (msg.route && typeof msg.route === 'object') {
        var r = msg.route;
        state.route = {
          plug_id: typeof r.plug_id === 'string' ? r.plug_id : '', view_id: typeof r.view_id === 'string' ? r.view_id : 'home',
          path: typeof r.path === 'string' ? r.path : '/', search: typeof r.search === 'string' ? r.search : '',
          url: typeof r.url === 'string' ? r.url : '/', tab: typeof r.tab === 'string' && r.tab !== '' ? r.tab : null,
          params: r.params && typeof r.params === 'object' ? Object.assign({}, r.params) : Object.create(null),
          query: r.query && typeof r.query === 'object' ? Object.assign({}, r.query) : Object.create(null),
        };
      }
      state.permittedMethods = Array.isArray(msg.permitted_methods) ? msg.permitted_methods.slice() : [];
      state.scopes = Array.isArray(msg.scopes) ? msg.scopes.slice() : (msg.caller && Array.isArray(msg.caller.scopes) ? msg.caller.scopes.slice() : []);
      state.activeRoles = Array.isArray(msg.active_roles) ? msg.active_roles.slice() : (msg.caller && Array.isArray(msg.caller.active_roles) ? msg.caller.active_roles.slice() : []);
      applyThemeTokens(msg.theme_mode, msg.theme_tokens, msg.theme, msg.base_font_size);
      notifyInitListeners();
      if (msg.type === 'mis:route:update' || (wasReady && prevUrl !== state.route.url)) notifyRouteChangedListeners();
      var nextStaffId = (state.context && typeof state.context.staff_id === 'string') ? state.context.staff_id : '';
      if (msg.type === 'mis:persona:changed' || (wasReady && prevStaffId !== nextStaffId)) {
        var snapshot = buildSnapshot();
        for (var p = 0; p < personaChangedListeners.length; p++) {
          try { personaChangedListeners[p](snapshot); } catch (pErr) { console.error('[mis-bridge] persona listener error:', pErr); }
        }
        if (typeof CustomEvent === 'function') global.dispatchEvent(new CustomEvent('mis:persona:changed', { detail: snapshot }));
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
        try { invalidationListeners[j](msg); } catch (iErr) { console.error('[mis-bridge] invalidate listener error:', iErr); }
      }
      if (typeof CustomEvent === 'function') global.dispatchEvent(new CustomEvent('mis:plugin:invalidated', { detail: msg }));
      return;
    }

    if (msg.type === 'mis:rpc:response') {
      var reqId = msg.request_id, pending = reqId ? pendingRequests[reqId] : null;
      if (pending) {
        delete pendingRequests[reqId];
        if (pending.element && typeof pending.element.removeAttribute === 'function') pending.element.removeAttribute('data-mis-loading');
      }
      var invMethod = pending ? pending.method : (msg.method || '');
      var target = (pending && pending.element) || document;
      if (msg.ok) {
        var detailOk = { request_id: reqId, method: invMethod, args: pending ? pending.args : {}, data: msg.data };
        if (typeof CustomEvent === 'function') target.dispatchEvent(new CustomEvent('mis:rpc:success', { bubbles: true, detail: detailOk }));
        if (pending) pending.resolve(msg.data);
      } else {
        var errCode = (typeof msg.error_code === 'string' && msg.error_code.trim() !== '') ? msg.error_code.trim() : ((typeof msg.error === 'string' && msg.error.trim() !== '') ? msg.error.trim() : 'UNKNOWN_RPC_ERROR');
        var rpcErr = new Error(errCode);
        rpcErr.code = errCode; rpcErr.request_id = reqId; rpcErr.method = invMethod;
        var detailErr = { request_id: reqId, method: invMethod, args: pending ? pending.args : {}, error_code: errCode };
        if (typeof CustomEvent === 'function') target.dispatchEvent(new CustomEvent('mis:rpc:error', { bubbles: true, detail: detailErr }));
        if (pending) pending.reject(rpcErr);
      }
      scheduleReportContentHeight();
    }
  });

  document.addEventListener('click', function (event) {
    if (!event.target || typeof event.target.closest !== 'function') return;

    var tabTarget = event.target.closest('[' + DECLARATIVE_ATTRIBUTES.ROUTE_TAB + ']');
    if (tabTarget) {
      event.preventDefault();
      var tabVal = tabTarget.getAttribute(DECLARATIVE_ATTRIBUTES.ROUTE_TAB) || '';
      if (tabVal.trim() !== '') setRouteState({ tab: tabVal.trim() });
      return;
    }

    var linkTarget = event.target.closest('[' + DECLARATIVE_ATTRIBUTES.ENTITY_LINK + ']');
    if (linkTarget) {
      event.preventDefault();
      navigate(linkTarget.getAttribute(DECLARATIVE_ATTRIBUTES.ENTITY_LINK) || '', {
        entity_id: linkTarget.getAttribute('data-entity-id') || '',
        pupil_id: linkTarget.getAttribute(DECLARATIVE_ATTRIBUTES.PUPIL_ID) || '',
        group_id: linkTarget.getAttribute(DECLARATIVE_ATTRIBUTES.GROUP_ID) || '',
        staff_id: linkTarget.getAttribute(DECLARATIVE_ATTRIBUTES.STAFF_ID) || '',
        target_plugin: linkTarget.getAttribute(DECLARATIVE_ATTRIBUTES.PLUGIN_TARGET) || '',
        tab: linkTarget.getAttribute('data-mis-tab') || '',
      });
      return;
    }

    var rpcTarget = event.target.closest('[' + DECLARATIVE_ATTRIBUTES.RPC_METHOD + ']');
    if (rpcTarget) {
      var method = rpcTarget.getAttribute(DECLARATIVE_ATTRIBUTES.RPC_METHOD);
      var rawArgs = rpcTarget.getAttribute(DECLARATIVE_ATTRIBUTES.RPC_ARGS);
      if (!method || method.trim() === '' || rawArgs === null || rawArgs.trim() === '') {
        throw new Error('Missing mandatory data-mis-rpc or data-mis-args attribute');
      }
      var parsedArgs;
      try { parsedArgs = JSON.parse(rawArgs); } catch (pErr) { throw new Error('Invalid JSON in data-mis-args attribute: ' + pErr.message); }
      rpc(method, parsedArgs, rpcTarget).catch(function () {});
    }
  });

  var LUCIDE_ICONS = {
    activity: '<path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/>', 'alert-triangle': '<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
    award: '<circle cx="12" cy="8" r="6"/><path d="M15.477 12.89 17 22l-5-3-5 3 1.523-9.11"/>', 'bar-chart-3': '<path d="M3 3v18h18"/><path d="M18 17V9"/><path d="M13 17V5"/><path d="M8 17v-3"/>',
    bell: '<path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>', 'book-open': '<path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/>',
    box: '<path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"/><path d="m3.3 7 8.7 5 8.7-5"/><path d="M12 22V12"/>', briefcase: '<rect width="20" height="14" x="2" y="7" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>',
    bus: '<path d="M8 6v6"/><path d="M15 6v6"/><path d="M2 12h19.6"/><path d="M18 18h3s.5-1.7.8-2.8c.1-.4.2-.8.2-1.2 0-.4-.1-.8-.2-1.2l-1.4-5C20.1 6.8 19.1 6 18 6H4a2 2 0 0 0-2 2v10h3"/><circle cx="7" cy="18" r="2"/><path d="M9 18h5"/><circle cx="16" cy="18" r="2"/>', calculator: '<rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/>',
    calendar: '<rect width="18" height="18" x="3" y="4" rx="2" ry="2"/><line x1="16" x2="16" y1="2" y2="6"/><line x1="8" x2="8" y1="2" y2="6"/><line x1="3" x2="21" y1="10" y2="10"/>', 'calendar-clock': '<path d="M21 7.5V6a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h3.5"/><path d="M16 2v4"/><path d="M8 2v4"/><path d="M3 10h5"/><path d="M17.5 17.5 16 16.3V14"/><circle cx="16" cy="16" r="6"/>',
    'check-circle': '<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><path d="m9 11 3 3L22 4"/>', 'clipboard-check': '<rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="m9 14 2 2 4-4"/>',
    'clipboard-list': '<rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><path d="M12 11h4"/><path d="M12 16h4"/><path d="M8 11h.01"/><path d="M8 16h.01"/>', clock: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>',
    'credit-card': '<rect width="20" height="14" x="2" y="5" rx="2"/><line x1="2" x2="22" y1="10" y2="10"/>', database: '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/>',
    'file-check': '<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="m9 15 2 2 4-4"/>', 'file-text': '<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>',
    folder: '<path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/>', 'git-branch': '<line x1="6" x2="6" y1="3" y2="15"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>',
    'graduation-cap': '<path d="M21.42 10.922a1 1 0 0 0-.019-1.838L12.83 5.18a2 2 0 0 0-1.66 0L2.6 9.08a1 1 0 0 0 0 1.832l8.57 3.908a2 2 0 0 0 1.66 0z"/><path d="M22 10v6"/><path d="M6 12.5V16a6 3 0 0 0 12 0v-3.5"/>', 'heart-pulse': '<path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/><path d="M3.22 12H9.5l.5-1 2 4.5 2-7 1.5 3.5h5.27"/>',
    home: '<path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/>', house: '<path d="M15 21v-8a1 1 0 0 0-1-1h-4a1 1 0 0 0-1 1v8"/><path d="M3 10a2 2 0 0 1 .709-1.528l7-5.999a2 2 0 0 1 2.582 0l7 5.999A2 2 0 0 1 21 10v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    inbox: '<polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/><path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>', landmark: '<line x1="3" x2="21" y1="22" y2="22"/><line x1="6" x2="6" y1="18" y2="11"/><line x1="10" x2="10" y1="18" y2="11"/><line x1="14" x2="14" y1="18" y2="11"/><line x1="18" x2="18" y1="18" y2="11"/><polygon points="12 2 20 7 4 7"/>',
    layers: '<path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z"/><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65"/><path d="m22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65"/>', 'layout-grid': '<rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/>',
    'log-out': '<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" x2="9" y1="12" y2="12"/>', 'message-square': '<path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>',
    pin: '<path d="M12 17v5"/><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H8a2 2 0 0 0 0 4 1 1 0 0 1 1 1z"/>', 'refresh-cw': '<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/>',
    search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>', settings: '<path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/>',
    'shield-alert': '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="M12 8v4"/><path d="M12 16h.01"/>', 'shield-check': '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/>',
    stethoscope: '<path d="M4.8 2.3A.3.3 0 1 0 5 2H4a2 2 0 0 0-2 2v5a6 6 0 0 0 6 6v0a6 6 0 0 0 6-6V4a2 2 0 0 0-2-2h-1a.2.2 0 1 0 .3.3"/><path d="M8 15v1a6 6 0 0 0 6 6v0a6 6 0 0 0 6-6v-4"/><circle cx="20" cy="10" r="2"/>', user: '<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>',
    'user-check': '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><polyline points="16 11 18 13 22 9"/>', users: '<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>',
    utensils: '<path d="M3 2v7c0 1.1.9 2 2 2h4a2 2 0 0 0 2-2V2"/><path d="M7 2v20"/><path d="M21 15V2v0a5 5 0 0 0-5 5v6c0 1.1.9 2 2 2h3Zm0 0v7"/>', wallet: '<path d="M21 12V7H5a2 2 0 0 1 0-4h14v4"/><path d="M3 5v14a2 2 0 0 0 2 2h16v-5"/><path d="M18 12a2 2 0 0 0 0 4h4v-4Z"/>',
  };

  function getLucideSvg(iconName) {
    var key = typeof iconName === 'string' ? iconName.trim().toLowerCase() : '';
    if (!key || !LUCIDE_ICONS[key]) throw new Error('UNKNOWN_LUCIDE_ICON: unsupported monochrome Lucide icon "' + iconName + '"');
    return '<svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" data-lucide-icon="' + key + '" aria-hidden="true">' + LUCIDE_ICONS[key] + '</svg>';
  }

  function renderIcons(rootNode) {
    var scope = rootNode && typeof rootNode.querySelectorAll === 'function' ? rootNode : document;
    if (!scope || typeof scope.querySelectorAll !== 'function') return;
    var nodes = [];
    if (rootNode && typeof rootNode.getAttribute === 'function' && rootNode.getAttribute('data-mis-icon')) nodes.push(rootNode);
    var matched = scope.querySelectorAll('[data-mis-icon]');
    for (var k = 0; k < matched.length; k++) nodes.push(matched[k]);
    for (var i = 0; i < nodes.length; i++) {
      var iconName = nodes[i].getAttribute('data-mis-icon');
      if (iconName && iconName.trim() !== '') {
        var key = iconName.trim().toLowerCase();
        if (LUCIDE_ICONS[key]) nodes[i].innerHTML = getLucideSvg(key);
      }
    }
  }

  function addListener(list, fn, errMsg, invokeIfReady) {
    if (typeof fn !== 'function') throw new Error(errMsg);
    list.push(fn); if (invokeIfReady && state.isReady) fn(buildSnapshot());
    return function () { var i = list.indexOf(fn); if (i !== -1) list.splice(i, 1); };
  }

  var misSDK = {
    version: '1.0.0',
    get isReady() { return state.isReady; },
    get themeMode() { return state.themeMode; },
    get theme() { return Object.assign({}, state.themeTokens); },
    get context() { return Object.assign({}, state.context); },
    get route() { return buildSnapshot().route; },
    get permittedMethods() { return state.permittedMethods.slice(); },
    get scopes() { return state.scopes.slice(); },
    get activeRoles() { return state.activeRoles.slice(); },
    ready: function () { return readyPromise; },
    onInit: function (fn) { return addListener(initListeners, fn, 'window.mis.onInit requires a function callback', true); },
    onRouteChanged: function (fn) { return addListener(routeChangedListeners, fn, 'window.mis.onRouteChanged requires a function callback', false); },
    onPersonaChanged: function (fn) { return addListener(personaChangedListeners, fn, 'window.mis.onPersonaChanged requires a function callback', false); },
    onInvalidated: function (fn) { return addListener(invalidationListeners, fn, 'window.mis.onInvalidated requires a function callback', false); },
    rpc: function (method, args) { return rpc(method, args, null); },
    useMisRpc: function (method, defaultArgs) {
      if (typeof method !== 'string' || method.trim() === '') throw new Error('MISSING_RPC_METHOD: useMisRpc requires a non-empty method name');
      var normMethod = method.trim(), baseArgs = defaultArgs && typeof defaultArgs === 'object' && !Array.isArray(defaultArgs) ? Object.assign({}, defaultArgs) : {};
      return {
        method: normMethod,
        call: function (overrideArgs) {
          var finalArgs = overrideArgs !== undefined && typeof overrideArgs === 'object' && !Array.isArray(overrideArgs)
            ? Object.assign({}, baseArgs, overrideArgs) : (overrideArgs !== undefined ? overrideArgs : Object.assign({}, baseArgs));
          return rpc(normMethod, finalArgs, null);
        },
      };
    },
    navigate: navigate, setRouteState: setRouteState, getLucideSvg: getLucideSvg, renderIcons: renderIcons,
    DECLARATIVE_ATTRIBUTES: DECLARATIVE_ATTRIBUTES,
  };

  Object.defineProperty(global, 'mis', { value: misSDK, writable: false, configurable: false });

  var observersAttached = false;
  function attachAutoResizeObservers() {
    if (observersAttached) return;
    observersAttached = true;
    if (typeof global.MutationObserver === 'function' && document.documentElement) {
      var mo = new global.MutationObserver(function (mutations) {
        var hasChild = false;
        for (var i = 0; i < mutations.length; i++) {
          if (mutations[i].type === 'childList' && mutations[i].addedNodes && mutations[i].addedNodes.length > 0) { hasChild = true; break; }
        }
        if (hasChild) scaleDocumentTypography(document);
        scheduleReportContentHeight();
      });
      mo.observe(document.documentElement, { childList: true, subtree: true, attributes: true, characterData: true });
    }
    if (typeof global.ResizeObserver === 'function' && document.body) {
      var ro = new global.ResizeObserver(function () { scheduleReportContentHeight(); });
      ro.observe(document.body);
    }
    global.addEventListener('resize', function () {
      var w = global.innerWidth || 0;
      if (w !== lastInnerWidth) { lastInnerWidth = w; scheduleReportContentHeight(); }
    });
    global.addEventListener('load', scheduleReportContentHeight);
  }

  function sendReady() {
    scaleDocumentTypography(document);
    renderIcons(document);
    attachAutoResizeObservers();
    global.parent.postMessage({ type: 'mis:ready' }, '*');
    scheduleReportContentHeight();
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', sendReady); else sendReady();
})(window);
