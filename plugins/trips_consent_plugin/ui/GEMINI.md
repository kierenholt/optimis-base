# Trip Consent Plugin Sandboxed UI (`plugins/trips_consent_plugin/ui`)

This directory contains the offline HTML/JS/CSS micro-frontend for `uk.org.optimis.ops.trips`, rendered inside `<iframe sandbox="allow-scripts">` with a strict `connect-src 'none'` Content Security Policy.

## Files

* [`dist/index.html`](file:///home/kieren/Documents/code/optimis/plugins/trips_consent_plugin/ui/dist/index.html):
  * Binds declarative `data-mis-rpc` actions (`create_trip`, `record_trip_consent`, `generate_trip_emergency_pack`) to the Host `window.postMessage` RPC bridge.
  * Renders the Trip Leader Visit Setup panel, Roster & PR Parental Consent Verification form, fail-fast error banner (`trips-error-banner`), and the Trip Emergency Pack Manifest (`trip-emergency-manifest-card`) with SEN, Medical/IHCP, dietary/medical notes, and PR emergency contact badges.
