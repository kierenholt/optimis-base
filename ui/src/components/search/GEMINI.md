# Search Component Architecture (`ui/src/components/search`)

Modularized search and command palette suite powering global school entity, plugin view, and dashboard widget discovery across the Host UI Shell.

## Architectural Structure

The search domain is decomposed into single-responsibility, cohesive modules:

1. **[`searchTypes.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/searchTypes.ts)**:
   Shared contracts and types:
   - `IndexedViewResult`: Model representing an indexed plugin view with caller permissions.
   - `GlobalSearchSharedProps`: Common input props passed down to search surfaces.
   - `CommandPaletteTab`: Active category filter (`'all' | 'plugins' | 'widgets' | 'pupils' | 'groups'`).
   - `FILTER_TAB_DEFS`: Configuration array containing monochrome Lucide icon names and human-readable category labels.
   - `SearchCategoryCounts`: Aggregated result counter structure for tab badges.

2. **[`searchStyles.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/searchStyles.ts)**:
   CSSProperties object literals and dynamic styling functions (`searchInputStyle`, `resultItemButtonStyle`, `pinButtonStyle`, `widgetItemContainerStyle`, `addWidgetButtonStyle`, `sectionTitleStyle`, `emptyStateStyle`, `tabButtonStyle`) consuming `--mis-*` CSS custom properties.

3. **[`searchMatchingHelpers.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/searchMatchingHelpers.ts)**:
   Pure matching and parameterized path resolution functions:
   - `matchesViewItem`: Matches search queries against view title, plugin title, view ID, path, and `keywords[]` synonyms/verbs.
   - `matchesWidgetItem`: Matches queries against widget title, widget ID, plugin title, and `keywords[]` synonyms.
   - `isParameterizedViewResolvable`: Validates whether a parameterized route (`:pupil_id`, `:group_id`, `:staff_id`) has active context so unreachable views are omitted.
   - `resolveParameterizedViewSubpath`: Resolves parameterized segments with strict fail-fast error checks if required entity IDs are missing.
   - `filterPluginViews`: Filters all installed plugin views against search term and active context.
   - `filterPluginWidgets`: Filters available widgets against search term.

4. **[`useScopedEntitySearch.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/useScopedEntitySearch.ts)**:
   Abortable entity search hook targeting `/api/system/search?q=...&staff_id=...&reference_date=...&academic_year_id=...` with `AbortController` cancellation on keystrokes.

5. **[`useGlobalSearchIndex.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/useGlobalSearchIndex.ts)**:
   Orchestration hook combining pure view/widget matchers and remote entity search, returning aggregated category counts, matched results, and visibility flags.

6. **Item Subcomponents**:
   - **[`SearchResultViewItem.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/SearchResultViewItem.tsx)**: View result row rendering monochrome Lucide icon, route subpath, permitted method badge count, and inline Pin toggle button.
   - **[`SearchResultWidgetItem.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/SearchResultWidgetItem.tsx)**: Widget result row with composite key (`plug_id:widget_id`) and inline "Add to Dashboard" placement button.
   - **[`SearchResultPupilItem.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/SearchResultPupilItem.tsx)**: Pupil result row displaying pupil full name, UPN, and NC Year Group.
   - **[`SearchResultGroupItem.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/SearchResultGroupItem.tsx)**: Teaching/pastoral group result row with group code and type.

7. **[`SearchResultsList.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/SearchResultsList.tsx)**:
   Unified results container grouping results by category sections with custom test IDs, action labels, and empty states.

8. **Top-Level Surfaces**:
   - **[`LandingPageTopHalfSearch.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/LandingPageTopHalfSearch.tsx)**: Embedded on the Home dashboard (`data-testid="landing-page-search-panel"`), featuring record-type filter tabs and the "Full list" palette launcher.
   - **[`GlobalCommandPaletteModal.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/GlobalCommandPaletteModal.tsx)**: Accessible modal dialog (`data-testid="command-palette-modal"`) opened via `Ctrl+K` or "Full list" button, with automatic input focus, backdrop dismiss, and category tabs with counts.
   - **[`HostFullPageSearchView.tsx`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/HostFullPageSearchView.tsx)**: Standalone right-hand workspace view (`data-testid="host-search-view"`) mounted at `/search`.

9. **[`index.ts`](file:///home/kieren/Documents/code/optimis/ui/src/components/search/index.ts)**:
   Clean barrel export for all sub-modules.
