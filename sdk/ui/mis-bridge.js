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
  var readyResolved = false;
  var readyResolveFn = null;

  var readyPromise = new Promise(function (resolve) {
    readyResolveFn = resolve;
  });

  var state = {
    isReady: false,
    themeMode: 'light',
    themeTokens: Object.create(null),
    context: Object.create(null),
    permittedMethods: [],
    scopes: [],
    activeRoles: [],
  };

  function applyThemeTokens(themeMode, themeTokens, legacyTheme) {
    var mode = typeof themeMode === 'string' && themeMode.trim() !== '' ? themeMode.trim() : 'light';
    state.themeMode = mode;

    var root = document.documentElement;
    if (root) {
      root.setAttribute('data-mis-theme', mode);
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
      theme: Object.assign({}, state.themeTokens),
      context: Object.assign({}, state.context),
      permitted_methods: state.permittedMethods.slice(),
      scopes: state.scopes.slice(),
      active_roles: state.activeRoles.slice(),
    };
  }

  function notifyInitListeners() {
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

    var message = {
      type: 'mis:link:navigate',
      entity_type: normalizedType,
      entity_id: entityId,
      pupil_id: pupilId || (normalizedType === 'student' ? entityId : undefined),
      group_id: groupId || (normalizedType === 'group' ? entityId : undefined),
      staff_id: staffId || (normalizedType === 'staff' ? entityId : undefined),
    };

    global.parent.postMessage(message, '*');
    return message;
  }

  global.addEventListener('message', function (event) {
    var msg = event.data;
    if (!msg || typeof msg !== 'object') {
      return;
    }

    if (msg.type === 'mis:init') {
      state.isReady = true;
      state.context =
        msg.context && typeof msg.context === 'object'
          ? Object.assign({}, msg.context)
          : Object.create(null);
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
      applyThemeTokens(msg.theme_mode, msg.theme_tokens, msg.theme);
      notifyInitListeners();
      return;
    }

    if (msg.type === 'mis:theme:update') {
      applyThemeTokens(msg.theme_mode, msg.theme_tokens, msg.theme);
      notifyInitListeners();
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

    // 1. Declarative Entity Deep-Linking: [data-mis-link] (Section 9.5 / 9C)
    var linkTarget = event.target.closest('[data-mis-link]');
    if (linkTarget) {
      event.preventDefault();
      var entityType = linkTarget.getAttribute('data-mis-link');
      var pupilId = linkTarget.getAttribute('data-pupil-id') || '';
      var groupId = linkTarget.getAttribute('data-group-id') || '';
      var staffId = linkTarget.getAttribute('data-staff-id') || '';
      var entityId = linkTarget.getAttribute('data-entity-id') || '';

      navigate(entityType || '', {
        entity_id: entityId,
        pupil_id: pupilId,
        group_id: groupId,
        staff_id: staffId,
      });
      return;
    }

    // 2. Declarative RPC Invocation: [data-mis-rpc] (Sections 6C & 9.4)
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
    rpc: function (method, args) {
      return rpc(method, args, null);
    },
    navigate: navigate,
  };

  Object.defineProperty(global, 'mis', {
    value: misSDK,
    writable: false,
    configurable: false,
  });

  function sendReady() {
    global.parent.postMessage({ type: 'mis:ready' }, '*');
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', sendReady);
  } else {
    sendReady();
  }
})(window);
