import { useEffect, useState } from 'react';
import { SearchGroupResult, SearchPupilResult } from '../../themeAndTypes';

export function useScopedEntitySearch(
  query: string,
  activeStaffId: string,
  referenceDate: string,
  academicYearId: string,
  enabled: boolean,
  allowEmptyQuery: boolean = false
) {
  const [pupils, setPupils] = useState<SearchPupilResult[]>([]);
  const [groups, setGroups] = useState<SearchGroupResult[]>([]);

  useEffect(() => {
    const trimmed = query.trim();
    if (!enabled || (!allowEmptyQuery && !trimmed)) {
      setPupils([]);
      setGroups([]);
      return;
    }

    const controller = new AbortController();
    const fetchEntities = async () => {
      try {
        const qs = new URLSearchParams();
        qs.set('q', trimmed);
        if (activeStaffId) {
          qs.set('staff_id', activeStaffId);
        }
        if (referenceDate) {
          qs.set('reference_date', referenceDate);
        }
        if (academicYearId) {
          qs.set('academic_year_id', academicYearId);
        }
        const res = await fetch(`/api/system/search?${qs.toString()}`, {
          signal: controller.signal,
        });
        if (!res.ok || controller.signal.aborted) {
          return;
        }
        const body = await res.json();
        if (controller.signal.aborted || !body || !body.ok) {
          return;
        }
        setPupils(Array.isArray(body.pupils) ? body.pupils : []);
        setGroups(Array.isArray(body.groups) ? body.groups : []);
      } catch {
        if (!controller.signal.aborted) {
          setPupils([]);
          setGroups([]);
        }
      }
    };

    void fetchEntities();
    return () => {
      controller.abort();
    };
  }, [query, activeStaffId, referenceDate, academicYearId, enabled, allowEmptyQuery]);

  return { pupils, groups };
}
