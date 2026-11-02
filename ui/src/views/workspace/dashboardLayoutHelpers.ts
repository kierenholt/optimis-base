import {
  findFirstFree2DSlot,
  is2DFootprintFree,
  PlacedDashboardWidget,
  ResolvedDashboardWidgetSlot,
} from '../../themeAndTypes';

export const normalizePresetDim = (val: number | undefined, fallback: 1 | 2): 1 | 2 =>
  val === 1 || val === 2 ? val : fallback;

export function mapResolvedSlotsToPlacedWidgets(
  slots: ResolvedDashboardWidgetSlot[],
  _existingWidgets?: PlacedDashboardWidget[],
  colCountOverride?: number
): PlacedDashboardWidget[] {
  const effectiveCols =
    typeof colCountOverride === 'number' && colCountOverride >= 2 && colCountOverride <= 8
      ? colCountOverride
      : 4;
  const acc: PlacedDashboardWidget[] = [];
  for (const slot of slots) {
    const presetWidth = normalizePresetDim(slot.width, 2);
    const presetHeight = normalizePresetDim(slot.height, 1);
    let col = typeof slot.col === 'number' && slot.col >= 1 ? slot.col : 0;
    let row = typeof slot.row === 'number' && slot.row >= 1 ? slot.row : 0;
    if (!is2DFootprintFree(acc, col, row, presetWidth, presetHeight, effectiveCols)) {
      const free = findFirstFree2DSlot(acc, presetWidth, presetHeight, effectiveCols);
      col = free.col;
      row = free.row;
    }
    acc.push({
      id: slot.widget_id,
      plugin_id: slot.plug_id,
      plugin_title: slot.plugin_title,
      title: slot.title || slot.widget_id,
      description: slot.description,
      icon: slot.icon,
      entry: slot.entry || '',
      entry_url: slot.entry_url || '',
      width: presetWidth,
      height: presetHeight,
      colSpan: presetWidth,
      rowSpan: presetHeight,
      col,
      row,
      permitted: slot.permitted !== false,
      error_code: slot.error_code,
      error_title: slot.error_title,
    });
  }
  return acc;
}
