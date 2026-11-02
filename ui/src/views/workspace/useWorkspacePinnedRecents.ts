import { useCallback, useState } from 'react';
import { PinnedRecentEntry } from '../../themeAndTypes';

export interface UseWorkspacePinnedRecentsParams {
  activeStaffId: string;
  onSelectPinnedRoute?: (item: PinnedRecentEntry) => void;
}

export function useWorkspacePinnedRecents({
  activeStaffId,
  onSelectPinnedRoute,
}: UseWorkspacePinnedRecentsParams) {
  const [pinnedRecents, setPinnedRecents] = useState<PinnedRecentEntry[]>([]);

  const handlePinOrUnpinRoute = useCallback(
    async (route: string, title: string, viewId?: string) => {
      const trimmedRoute = route.trim();
      const trimmedTitle = title.trim();
      if (!trimmedRoute || !trimmedTitle) {
        throw new Error('Missing mandatory route or title for pinning view');
      }
      const isAlreadyPinned = pinnedRecents.some((r) => r.route === trimmedRoute);
      const action = isAlreadyPinned ? 'unpin' : 'pin';
      const res = await fetch('/api/system/pinned-recents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action,
          route: trimmedRoute,
          title: trimmedTitle,
          view_id: viewId || undefined,
          staff_id: activeStaffId || undefined,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        throw new Error(body?.error_code || 'Failed to update pinned recents');
      }
      if (Array.isArray(body.pinned_recents)) {
        setPinnedRecents(body.pinned_recents as PinnedRecentEntry[]);
      }
    },
    [activeStaffId, pinnedRecents]
  );

  const handleUnpinRoute = useCallback(
    async (route: string) => {
      const trimmedRoute = route.trim();
      if (!trimmedRoute) {
        throw new Error('Missing mandatory route for unpinning view');
      }
      const res = await fetch('/api/system/pinned-recents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'unpin',
          route: trimmedRoute,
          staff_id: activeStaffId || undefined,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        throw new Error(body?.error_code || 'Failed to unpin route');
      }
      if (Array.isArray(body.pinned_recents)) {
        setPinnedRecents(body.pinned_recents as PinnedRecentEntry[]);
      }
    },
    [activeStaffId]
  );

  const handleSelectPinnedRoute = useCallback(
    (item: PinnedRecentEntry) => {
      onSelectPinnedRoute?.(item);
    },
    [onSelectPinnedRoute]
  );

  return {
    pinnedRecents,
    setPinnedRecents,
    handlePinOrUnpinRoute,
    handleUnpinRoute,
    handleSelectPinnedRoute,
  };
}
