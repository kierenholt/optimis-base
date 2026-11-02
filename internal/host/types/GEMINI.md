# Host Types & Manifest Specifications (`internal/host/types`)

This subpackage contains common domain, manifest, RPC, and event types shared across the Host runtime, DAL bindings, and subpackages without circular dependency issues.

## Key Types

* [`Config`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L7): Bootstrap configuration parameters (`DataDirPath`, `SchoolURN`, `HostKMSKeyHandle`, `PluginsDir`, `CoreDBPath`).
* [`Manifest`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L64): Hand-written plugin manifest definition containing reverse-DNS `id`, `version`, `core_dal_version`, `database`, `egress`, `endpoints`, `workers`, `views`, and `widgets`.
* [`LoadedPlugin`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L87): Active plugin memory instance including opaque 8-character `PlugID`, `Manifest`, `LuaSource`, and `PluginState`.
* [`CallerContext`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L99): Authenticated caller identity (`UserID`, `StaffID`, `ActiveRoles`, `Scopes`, `RequestID`).
* [`EndpointSpec`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L15): Scoping, role requirements, timeout, and mutation flags per endpoint or worker.
* [`RPCResponse`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L108): Standard `{ ok, data, error_code, error, mutated }` RPC return packet.
* [`Event`](file:///home/kieren/Documents/code/optimis/internal/host/types/types.go#L116): Event bus payload with hop count loop protection.
