/**
 * @ukmis/bridge — Official Frontend Bridge for Optimis Sandboxed Plugin Iframes (Section 10C)
 *
 * Auto-injects --mis-* CSS custom properties on mis:init, binds declarative
 * [data-mis-rpc] and [data-mis-link] DOM attributes, and exports useMisRpc(method, defaultArgs).
 */
'use strict';

function getGlobalMis() {
  if (typeof window !== 'undefined' && window.mis) {
    return window.mis;
  }
  throw new Error('MIS_BRIDGE_NOT_INITIALIZED: window.mis is not available in this context');
}

/**
 * Untyped JSON RPC helper hook / factory for invoking declared manifest.json endpoints.
 * @param {string} method - Lua endpoint name declared in manifest.json
 * @param {Record<string, any>} [defaultArgs] - Optional default JSON arguments
 */
function useMisRpc(method, defaultArgs) {
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
      var mis = getGlobalMis();
      if (typeof mis.useMisRpc === 'function') {
        return mis.useMisRpc(normalizedMethod, baseArgs).call(overrideArgs);
      }
      var finalArgs = overrideArgs !== undefined ? overrideArgs : Object.assign({}, baseArgs);
      return mis.rpc(normalizedMethod, finalArgs);
    },
  };
}

/**
 * Declarative attribute constants for [data-mis-rpc] and [data-mis-link].
 */
var DECLARATIVE_ATTRIBUTES = {
  RPC_METHOD: 'data-mis-rpc',
  RPC_ARGS: 'data-mis-args',
  ENTITY_LINK: 'data-mis-link',
  PUPIL_ID: 'data-pupil-id',
  GROUP_ID: 'data-group-id',
  STAFF_ID: 'data-staff-id',
  PLUGIN_TARGET: 'data-mis-plugin',
};

module.exports = {
  useMisRpc: useMisRpc,
  DECLARATIVE_ATTRIBUTES: DECLARATIVE_ATTRIBUTES,
  rpc: function (method, args) {
    return getGlobalMis().rpc(method, args);
  },
  navigate: function (entityType, params) {
    return getGlobalMis().navigate(entityType, params);
  },
};
