import { useMemo, useState } from 'react';
import { parseBrowserRoute } from '../../components/routeUtils';
import { ActiveCallerIdentity } from '../../themeAndTypes';

export function useWorkspaceEntityContext() {
  const initialBrowserRoute = useMemo(
    () => parseBrowserRoute(window.location.pathname, window.location.search),
    []
  );
  const params = useMemo(() => new URLSearchParams(window.location.search), []);

  const initialPupilId = initialBrowserRoute.pupilIdFromPath || params.get('pupil_id') || '';
  const initialSecondPupilId = params.get('second_pupil_id') || '';
  const initialGroupId = initialBrowserRoute.groupIdFromPath || params.get('group_id') || '';
  const initialOtherGroupId = params.get('other_group_id') || '';
  const initialAbsentStaffId = params.get('absent_staff_id') || '';
  const initialCoverStaffId = params.get('cover_staff_id') || '';
  const initialSlotId = params.get('slot_id') || '';
  const initialReferenceDate = params.get('reference_date') || '';
  const initialAcademicYearId = params.get('academic_year_id') || '';
  const initialStaffId = initialBrowserRoute.staffIdFromPath || '';

  const [activePupilId, setActivePupilId] = useState<string>(initialPupilId);
  const [activeGroupId, setActiveGroupId] = useState<string>(initialGroupId);
  const [activeStaffId, setActiveStaffId] = useState<string>(initialStaffId);
  const [activeCaller, setActiveCaller] = useState<ActiveCallerIdentity | null>(null);
  const [activeEntityContext, setActiveEntityContext] = useState<string>(
    initialPupilId ? `student:${initialPupilId}` : initialGroupId ? `group:${initialGroupId}` : 'dashboard:home'
  );
  const [deepLinkNavigationStatus, setDeepLinkNavigationStatus] = useState<string>('IDLE');

  return {
    initialBrowserRoute,
    initialPupilId,
    initialSecondPupilId,
    initialGroupId,
    initialOtherGroupId,
    initialAbsentStaffId,
    initialCoverStaffId,
    initialSlotId,
    initialReferenceDate,
    initialAcademicYearId,
    initialStaffId,
    activePupilId,
    setActivePupilId,
    activeGroupId,
    setActiveGroupId,
    activeStaffId,
    setActiveStaffId,
    activeCaller,
    setActiveCaller,
    activeEntityContext,
    setActiveEntityContext,
    deepLinkNavigationStatus,
    setDeepLinkNavigationStatus,
  };
}
