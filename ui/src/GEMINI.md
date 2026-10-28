# Host UI Shell Components (`ui/src`)

* [`main.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/main.tsx): React 19 DOM root entrypoint mounting [`App`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx).
* [`App.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/App.tsx): Host UI Shell layout (Left Navigation Sidebar, Top Bar with Light / High-Contrast theme switcher, Statutory Summary Strip, Keep-Alive `<iframe sandbox="allow-scripts">` Plugin Workspace, and `window.postMessage` RPC & Entity Deep-Link coordinator).
* [`SafeguardingVaultPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/SafeguardingVaultPanel.tsx): Designated Safeguarding Lead (DSL) Envelope-Encrypted Vault panel (`Renew OAuth Cloud Key` & `Unlock & Decrypt Note`).
* [`GdprSarPanel.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/GdprSarPanel.tsx): Admin/DPO UK GDPR Subject Access Request (SAR) interactive `redactable_fields` redaction UI and statutory retention purge panel.
* [`themeAndTypes.ts`](file:///home/kieren/Documents/code/optimis/ui/src/themeAndTypes.ts): Default Light and WCAG AAA High-Contrast `--mis-*` CSS Custom Property catalogs, `applyHostThemeToRoot` helper, and shared TypeScript interfaces.
