import { useCallback, useRef, useState } from 'react';
import {
  DashboardLayoutItem,
  findFirstFree2DSlot,
  getWidgetCompositeKey,
  is2DFootprintFree,
  PlacedDashboardWidget,
  PluginWidgetDescriptor,
  ResolvedDashboardWidgetSlot,
} from '../../themeAndTypes';
import {
  mapResolvedSlotsToPlacedWidgets,
  normalizePresetDim,
} from './dashboardLayoutHelpers';

export interface UseWorkspaceDashboardLayoutParams {
  activeStaffId: string;
}

export function useWorkspaceDashboardLayout({
  activeStaffId,
}: UseWorkspaceDashboardLayoutParams) {
  // Editable 2D Mini-Iframe Widget Grid & core.db Persistence state (Section 9B)
  const [isDashboardEditMode, setIsDashboardEditMode] = useState<boolean>(false);
  const [placedWidgets, setPlacedWidgets] = useState<PlacedDashboardWidget[]>([]);
  const [dashboardColumnCount, setDashboardColumnCount] = useState<number>(4);
  const [dashboardLayoutSource, setDashboardLayoutSource] = useState<string>('SYSTEM_FALLBACK');
  const [dashboardSaveStatus, setDashboardSaveStatus] = useState<string>('');
  const [roleDefaultLayouts, setRoleDefaultLayouts] = useState<Record<string, DashboardLayoutItem[]>>({});
  const [selectedAdminRoleTarget, setSelectedAdminRoleTarget] = useState<string>('TEACHER');
  const hasLoadedServerLayoutRef = useRef<boolean>(false);

  const applyDashboardLayoutResponse = useCallback(
    (layoutBody: any, existingWidgets?: PlacedDashboardWidget[]) => {
      if (layoutBody && layoutBody.ok && Array.isArray(layoutBody.resolved_widgets)) {
        hasLoadedServerLayoutRef.current = true;
        const resolvedCols =
          typeof layoutBody.column_count === 'number' &&
          layoutBody.column_count >= 2 &&
          layoutBody.column_count <= 8
            ? layoutBody.column_count
            : 4;
        setDashboardColumnCount(resolvedCols);
        const slots = layoutBody.resolved_widgets as ResolvedDashboardWidgetSlot[];
        setPlacedWidgets((prev) =>
          mapResolvedSlotsToPlacedWidgets(slots, existingWidgets || prev, resolvedCols)
        );
        if (typeof layoutBody.layout_source === 'string') {
          setDashboardLayoutSource(layoutBody.layout_source);
        }
        if (layoutBody.role_defaults && typeof layoutBody.role_defaults === 'object') {
          setRoleDefaultLayouts(
            layoutBody.role_defaults as Record<string, DashboardLayoutItem[]>
          );
        }
      }
    },
    []
  );

  const refreshDashboardLayoutForStaff = useCallback(
    async (_targetStaffId?: string) => {
      try {
        const layoutRes = await fetch('/api/dashboard/layout');
        if (!layoutRes.ok) {
          return;
        }
        const layoutBody = await layoutRes.json().catch(() => null);
        applyDashboardLayoutResponse(layoutBody);
      } catch {
        // Ignore network errors during dashboard layout refresh
      }
    },
    [applyDashboardLayoutResponse]
  );

  const persistStaffDashboardLayout = useCallback(
    async (nextWidgets: PlacedDashboardWidget[], nextColumnCount?: number) => {
      const resolvedColCount =
        typeof nextColumnCount === 'number' && nextColumnCount >= 2 && nextColumnCount <= 8
          ? nextColumnCount
          : dashboardColumnCount;
      const layoutPayload: DashboardLayoutItem[] = nextWidgets.map((w) => ({
        plug_id: w.plugin_id,
        widget_id: w.id,
        col: w.col,
        row: w.row,
      }));
      setDashboardColumnCount(resolvedColCount);
      setPlacedWidgets(nextWidgets);
      try {
        const res = await fetch('/api/dashboard/layout', {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            staff_id: activeStaffId || undefined,
            layout: layoutPayload,
            column_count: resolvedColCount,
          }),
        });
        const body = await res.json().catch(() => null);
        if (res.ok && body && body.ok) {
          hasLoadedServerLayoutRef.current = true;
          const serverCols =
            typeof body.column_count === 'number' && body.column_count >= 2 && body.column_count <= 8
              ? body.column_count
              : resolvedColCount;
          setDashboardColumnCount(serverCols);
          if (Array.isArray(body.resolved_widgets)) {
            setPlacedWidgets(
              mapResolvedSlotsToPlacedWidgets(
                body.resolved_widgets as ResolvedDashboardWidgetSlot[],
                nextWidgets,
                serverCols
              )
            );
          }
          setDashboardLayoutSource('STAFF');
          setDashboardSaveStatus('STAFF_LAYOUT_SAVED');
          return;
        }
      } catch {
        // Keep optimistic local state if network request fails
      }
    },
    [activeStaffId, dashboardColumnCount]
  );

  const handlePlaceWidget = useCallback(
    (widget: PluginWidgetDescriptor, targetCol?: number, targetRow?: number) => {
      if (placedWidgets.some((w) => w.plugin_id === widget.plugin_id && w.id === widget.id)) {
        return;
      }
      const presetWidth = normalizePresetDim(widget.width, 2);
      const presetHeight = normalizePresetDim(widget.height, 1);
      let col = typeof targetCol === 'number' ? targetCol : 0;
      let row = typeof targetRow === 'number' ? targetRow : 0;
      if (
        !is2DFootprintFree(
          placedWidgets,
          col,
          row,
          presetWidth,
          presetHeight,
          dashboardColumnCount
        )
      ) {
        const freeSlot = findFirstFree2DSlot(
          placedWidgets,
          presetWidth,
          presetHeight,
          dashboardColumnCount
        );
        col = freeSlot.col;
        row = freeSlot.row;
      }
      const newItem: PlacedDashboardWidget = {
        ...widget,
        width: presetWidth,
        height: presetHeight,
        colSpan: presetWidth,
        rowSpan: presetHeight,
        col,
        row,
        permitted: true,
      };
      const nextWidgets: PlacedDashboardWidget[] = [...placedWidgets, newItem];
      void persistStaffDashboardLayout(nextWidgets, dashboardColumnCount);
    },
    [placedWidgets, dashboardColumnCount, persistStaffDashboardLayout]
  );

  const handleRemoveWidget = useCallback(
    (pluginId: string, widgetId: string) => {
      const nextWidgets = placedWidgets.filter(
        (w) => !(w.plugin_id === pluginId && w.id === widgetId)
      );
      void persistStaffDashboardLayout(nextWidgets, dashboardColumnCount);
    },
    [placedWidgets, dashboardColumnCount, persistStaffDashboardLayout]
  );

  const moveWidgetByDelta = useCallback(
    (pluginId: string, widgetId: string, dCol: number, dRow: number) => {
      const idx = placedWidgets.findIndex(
        (w) => w.plugin_id === pluginId && w.id === widgetId
      );
      if (idx < 0) {
        return;
      }
      const current = placedWidgets[idx];
      const wWidth = current.width === 1 || current.width === 2 ? current.width : current.colSpan;
      const wHeight = current.height === 1 || current.height === 2 ? current.height : current.rowSpan;
      const nextCol = current.col + dCol;
      const nextRow = current.row + dRow;
      const compositeKey = getWidgetCompositeKey(pluginId, widgetId);
      if (
        !is2DFootprintFree(
          placedWidgets,
          nextCol,
          nextRow,
          wWidth,
          wHeight,
          dashboardColumnCount,
          compositeKey
        )
      ) {
        return;
      }
      const nextWidgets = placedWidgets.map((w, i) =>
        i === idx ? { ...w, col: nextCol, row: nextRow } : w
      );
      void persistStaffDashboardLayout(nextWidgets, dashboardColumnCount);
    },
    [placedWidgets, dashboardColumnCount, persistStaffDashboardLayout]
  );

  const handleMoveWidgetUp = useCallback(
    (pluginId: string, widgetId: string) => {
      moveWidgetByDelta(pluginId, widgetId, 0, -1);
    },
    [moveWidgetByDelta]
  );

  const handleMoveWidgetDown = useCallback(
    (pluginId: string, widgetId: string) => {
      moveWidgetByDelta(pluginId, widgetId, 0, 1);
    },
    [moveWidgetByDelta]
  );

  const handleMoveWidgetLeft = useCallback(
    (pluginId: string, widgetId: string) => {
      moveWidgetByDelta(pluginId, widgetId, -1, 0);
    },
    [moveWidgetByDelta]
  );

  const handleMoveWidgetRight = useCallback(
    (pluginId: string, widgetId: string) => {
      moveWidgetByDelta(pluginId, widgetId, 1, 0);
    },
    [moveWidgetByDelta]
  );

  const handleAddDashboardColumn = useCallback(() => {
    if (dashboardColumnCount >= 8) {
      return;
    }
    const nextColCount = dashboardColumnCount + 1;
    void persistStaffDashboardLayout(placedWidgets, nextColCount);
  }, [dashboardColumnCount, placedWidgets, persistStaffDashboardLayout]);

  const handleRemoveDashboardColumn = useCallback(() => {
    if (dashboardColumnCount <= 2) {
      return;
    }
    const occupiesRightmost = placedWidgets.some((w) => {
      const wWidth = w.width === 1 || w.width === 2 ? w.width : w.colSpan;
      return w.col + wWidth - 1 >= dashboardColumnCount;
    });
    if (occupiesRightmost) {
      return;
    }
    const nextColCount = dashboardColumnCount - 1;
    void persistStaffDashboardLayout(placedWidgets, nextColCount);
  }, [dashboardColumnCount, placedWidgets, persistStaffDashboardLayout]);

  const handleSaveRoleDefaultLayout = useCallback(
    async (roleId: string) => {
      const normalizedRole = roleId.trim().toUpperCase();
      if (!normalizedRole) {
        throw new Error('Missing mandatory role_id for saving role default dashboard layout');
      }
      const layoutPayload: DashboardLayoutItem[] = placedWidgets.map((w) => ({
        plug_id: w.plugin_id,
        widget_id: w.id,
        col: w.col,
        row: w.row,
      }));
      const res = await fetch('/api/admin/dashboard/role-default', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          staff_id: activeStaffId || undefined,
          role_id: normalizedRole,
          layout: layoutPayload,
          append_to_existing_staff: true,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        setDashboardSaveStatus(`ERROR:${body?.error_code || res.status}`);
        return;
      }
      if (body.role_defaults && typeof body.role_defaults === 'object') {
        setRoleDefaultLayouts(body.role_defaults as Record<string, DashboardLayoutItem[]>);
      }
      setDashboardSaveStatus(`ROLE_DEFAULT_SAVED:${normalizedRole}`);
      await refreshDashboardLayoutForStaff();
    },
    [activeStaffId, placedWidgets, refreshDashboardLayoutForStaff]
  );

  const handleAppendWidgetToRoleDefault = useCallback(
    async (roleId: string, widget: PluginWidgetDescriptor) => {
      const normalizedRole = roleId.trim().toUpperCase();
      if (!normalizedRole || !widget.plugin_id || !widget.id) {
        throw new Error('Missing mandatory role_id, plug_id, or widget_id for role default append');
      }
      const res = await fetch('/api/admin/dashboard/role-default/append', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          staff_id: activeStaffId || undefined,
          role_id: normalizedRole,
          plug_id: widget.plugin_id,
          widget_id: widget.id,
        }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok || !body || !body.ok) {
        setDashboardSaveStatus(`ERROR:${body?.error_code || res.status}`);
        return;
      }
      if (body.role_defaults && typeof body.role_defaults === 'object') {
        setRoleDefaultLayouts(body.role_defaults as Record<string, DashboardLayoutItem[]>);
      }
      setDashboardSaveStatus(`ROLE_WIDGET_APPENDED:${normalizedRole}:${widget.id}`);
      await refreshDashboardLayoutForStaff();
    },
    [activeStaffId, refreshDashboardLayoutForStaff]
  );

  return {
    isDashboardEditMode,
    setIsDashboardEditMode,
    placedWidgets,
    setPlacedWidgets,
    dashboardColumnCount,
    setDashboardColumnCount,
    dashboardLayoutSource,
    setDashboardLayoutSource,
    dashboardSaveStatus,
    setDashboardSaveStatus,
    roleDefaultLayouts,
    setRoleDefaultLayouts,
    selectedAdminRoleTarget,
    setSelectedAdminRoleTarget,
    hasLoadedServerLayoutRef,
    mapResolvedSlotsToPlacedWidgets,
    applyDashboardLayoutResponse,
    refreshDashboardLayoutForStaff,
    persistStaffDashboardLayout,
    handlePlaceWidget,
    handleRemoveWidget,
    handleMoveWidgetUp,
    handleMoveWidgetDown,
    handleMoveWidgetLeft,
    handleMoveWidgetRight,
    handleAddDashboardColumn,
    handleRemoveDashboardColumn,
    handleSaveRoleDefaultLayout,
    handleAppendWidgetToRoleDefault,
  };
}
