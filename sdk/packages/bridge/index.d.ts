export interface MisRpcHandle<TArgs = Record<string, unknown>, TResult = unknown> {
  readonly method: string;
  call(overrideArgs?: TArgs): Promise<TResult>;
}

export function useMisRpc<TArgs = Record<string, unknown>, TResult = unknown>(
  method: string,
  defaultArgs?: TArgs
): MisRpcHandle<TArgs, TResult>;

export function rpc<TResult = unknown>(
  method: string,
  args?: Record<string, unknown>
): Promise<TResult>;

export function navigate(
  entityType: 'student' | 'group' | 'staff' | string,
  params: {
    entity_id?: string;
    pupil_id?: string;
    group_id?: string;
    staff_id?: string;
    target_plugin?: string;
    tab?: string;
  }
): void;

export function setRouteState(
  nextState: {
    path?: string;
    entity_type?: string;
    entity_id?: string;
    pupil_id?: string;
    group_id?: string;
    staff_id?: string;
    tab?: string | null;
    query?: Record<string, string | null | undefined>;
  },
  options?: { replace?: boolean }
): void;

export const DECLARATIVE_ATTRIBUTES: {
  readonly RPC_METHOD: 'data-mis-rpc';
  readonly RPC_ARGS: 'data-mis-args';
  readonly ENTITY_LINK: 'data-mis-link';
  readonly PUPIL_ID: 'data-pupil-id';
  readonly GROUP_ID: 'data-group-id';
  readonly STAFF_ID: 'data-staff-id';
  readonly PLUGIN_TARGET: 'data-mis-plugin';
  readonly ROUTE_TAB: 'data-mis-route-tab';
};
