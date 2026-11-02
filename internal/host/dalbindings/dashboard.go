package dalbindings

import (
	"context"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

func dashboardWidgetItemsToLuaTable(L *lua.LState, items []dal.DashboardWidgetItem) *lua.LTable {
	arrTbl := L.NewTable()
	markLuaTableAsArray(L, arrTbl)
	for i, it := range items {
		iTbl := L.NewTable()
		L.SetField(iTbl, "plug_id", lua.LString(it.PlugID))
		L.SetField(iTbl, "widget_id", lua.LString(it.WidgetID))
		L.SetField(iTbl, "col", lua.LNumber(it.Col))
		L.SetField(iTbl, "row", lua.LNumber(it.Row))
		arrTbl.RawSetInt(i+1, iTbl)
	}
	return arrTbl
}

func parseDashboardWidgetItemsFromLua(tbl *lua.LTable) []dal.DashboardWidgetItem {
	if tbl == nil {
		return []dal.DashboardWidgetItem{}
	}
	var out []dal.DashboardWidgetItem
	for i := 1; i <= tbl.MaxN(); i++ {
		if itemTbl, ok := tbl.RawGetInt(i).(*lua.LTable); ok {
			var col, row int
			if v := itemTbl.RawGetString("col"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					col = int(n)
				}
			}
			if v := itemTbl.RawGetString("row"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					row = int(n)
				}
			}
			out = append(out, dal.DashboardWidgetItem{
				PlugID:   luaTableTrimmedString(itemTbl, "plug_id"),
				WidgetID: luaTableTrimmedString(itemTbl, "widget_id"),
				Col:      col,
				Row:      row,
			})
		}
	}
	return out
}

func (b *Binder) bindLuaDALDashboard(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	dalTbl *lua.LTable,
) {
	hasDashRead := hasDomainScope(endpointSpec.Scopes, "dal.dashboard.read")
	hasDashWrite := hasDomainScope(endpointSpec.Scopes, "dal.dashboard.write")
	hasConfigWrite := hasDomainScope(endpointSpec.Scopes, "plugin.config.write")
	if !hasDashRead && !hasDashWrite && !hasConfigWrite {
		return
	}

	dashTbl := L.NewTable()

	L.SetField(dashTbl, "list_available_widgets", L.NewFunction(func(L *lua.LState) int {
		arrTbl := L.NewTable()
		markLuaTableAsArray(L, arrTbl)
		idx := 1
		var plugins []*types.LoadedPlugin
		if b.listActivePlugins != nil {
			plugins = b.listActivePlugins()
		}
		for _, p := range plugins {
			if p.State != types.PluginStateActive {
				continue
			}
			plugID := strings.TrimSpace(p.PlugID)
			if plugID == "" {
				plugID = p.Manifest.ID
			}
			pTitle := strings.TrimSpace(p.Manifest.Title)
			if pTitle == "" {
				pTitle = p.Folder
			}
			for _, wSpec := range p.Manifest.Widgets {
				relEntry, err := normalizeWidgetEntryRelPath(wSpec.Entry)
				if err != nil {
					continue
				}
				wIcon := strings.TrimSpace(wSpec.Icon)
				if wIcon == "" {
					wIcon = strings.TrimSpace(p.Manifest.Icon)
				}
				wTbl := L.NewTable()
				L.SetField(wTbl, "plug_id", lua.LString(plugID))
				L.SetField(wTbl, "plugin_title", lua.LString(pTitle))
				L.SetField(wTbl, "widget_id", lua.LString(wSpec.ID))
				L.SetField(wTbl, "id", lua.LString(wSpec.ID))
				L.SetField(wTbl, "title", lua.LString(wSpec.Title))
				L.SetField(wTbl, "description", lua.LString(wSpec.Description))
				L.SetField(wTbl, "icon", lua.LString(wIcon))
				L.SetField(wTbl, "entry", lua.LString(wSpec.Entry))
				L.SetField(wTbl, "entry_url", lua.LString(fmt.Sprintf("/p/%s/ui/%s", plugID, relEntry)))
				L.SetField(wTbl, "width", lua.LNumber(wSpec.Width))
				L.SetField(wTbl, "height", lua.LNumber(wSpec.Height))
				arrTbl.RawSetInt(idx, wTbl)
				idx++
			}
		}
		L.Push(arrTbl)
		return 1
	}))

	L.SetField(dashTbl, "list_role_defaults", L.NewFunction(func(L *lua.LState) int {
		defaults, err := b.coreDAL.ListAllRoleDefaultDashboardLayouts(execCtx)
		if err != nil {
			L.RaiseError("dal.dashboard.list_role_defaults failed: %s", err.Error())
			return 0
		}
		resTbl := L.NewTable()
		for roleID, items := range defaults {
			L.SetField(resTbl, roleID, dashboardWidgetItemsToLuaTable(L, items))
		}
		L.Push(resTbl)
		return 1
	}))

	L.SetField(dashTbl, "get_role_default", L.NewFunction(func(L *lua.LState) int {
		var roleID string
		if L.GetTop() >= 1 {
			if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				roleID = luaTableTrimmedString(tbl, "role_id")
			} else {
				roleID = strings.TrimSpace(L.OptString(1, ""))
			}
		}
		if roleID == "" {
			L.RaiseError("missing mandatory parameter: role_id")
			return 0
		}
		items, _, err := b.coreDAL.GetRoleDefaultDashboardLayout(execCtx, roleID)
		if err != nil {
			L.RaiseError("dal.dashboard.get_role_default failed: %s", err.Error())
			return 0
		}
		L.Push(dashboardWidgetItemsToLuaTable(L, items))
		return 1
	}))

	L.SetField(dashTbl, "get_staff_layout", L.NewFunction(func(L *lua.LState) int {
		var staffID string
		if L.GetTop() >= 1 {
			if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
				staffID = luaTableTrimmedString(tbl, "staff_id")
			} else {
				staffID = strings.TrimSpace(L.OptString(1, ""))
			}
		}
		if staffID == "" {
			staffID = caller.StaffID
		}
		if staffID == "" {
			L.RaiseError("missing mandatory parameter: staff_id")
			return 0
		}
		items, colCount, hasOverride, err := b.coreDAL.GetStaffDashboardLayoutWithColumns(execCtx, staffID)
		if err != nil {
			L.RaiseError("dal.dashboard.get_staff_layout failed: %s", err.Error())
			return 0
		}
		resTbl := L.NewTable()
		L.SetField(resTbl, "staff_id", lua.LString(staffID))
		L.SetField(resTbl, "has_override", lua.LBool(hasOverride))
		L.SetField(resTbl, "column_count", lua.LNumber(colCount))
		L.SetField(resTbl, "layout", dashboardWidgetItemsToLuaTable(L, items))
		L.Push(resTbl)
		return 1
	}))

	if hasDashWrite || hasConfigWrite {
		L.SetField(dashTbl, "save_role_default", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.dashboard.save_role_default is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.dashboard.save_role_default requires an input table")
				return 0
			}
			roleID := luaTableTrimmedString(inTbl, "role_id")
			if roleID == "" {
				L.RaiseError("missing mandatory parameter: role_id")
				return 0
			}
			layoutTbl, _ := inTbl.RawGetString("layout").(*lua.LTable)
			items := parseDashboardWidgetItemsFromLua(layoutTbl)
			appendToStaff := true
			if v := inTbl.RawGetString("append_to_existing_staff"); v != lua.LNil {
				appendToStaff = v == lua.LTrue
			}
			saved, appendedCount, err := b.coreDAL.SaveRoleDefaultDashboardLayout(execCtx, roleID, items, appendToStaff, caller.StaffID)
			if err != nil {
				L.RaiseError("dal.dashboard.save_role_default failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "role_id", lua.LString(strings.ToUpper(roleID)))
			L.SetField(resTbl, "layout", dashboardWidgetItemsToLuaTable(L, saved))
			L.SetField(resTbl, "appended_staff_count", lua.LNumber(appendedCount))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dashTbl, "append_widget_to_role", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.dashboard.append_widget_to_role is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.dashboard.append_widget_to_role requires an input table")
				return 0
			}
			roleID := luaTableTrimmedString(inTbl, "role_id")
			plugID := luaTableTrimmedString(inTbl, "plug_id")
			widgetID := luaTableTrimmedString(inTbl, "widget_id")
			updated, appendedCount, err := b.coreDAL.AppendWidgetToRoleDashboardLayout(execCtx, roleID, dal.DashboardWidgetItem{
				PlugID:   plugID,
				WidgetID: widgetID,
			}, caller.StaffID)
			if err != nil {
				L.RaiseError("dal.dashboard.append_widget_to_role failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "role_id", lua.LString(strings.ToUpper(roleID)))
			L.SetField(resTbl, "layout", dashboardWidgetItemsToLuaTable(L, updated))
			L.SetField(resTbl, "appended_staff_count", lua.LNumber(appendedCount))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dashTbl, "save_staff_layout", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.dashboard.save_staff_layout is forbidden when endpoint mutates=false")
				return 0
			}
			inTbl, ok := L.Get(1).(*lua.LTable)
			if !ok {
				L.RaiseError("dal.dashboard.save_staff_layout requires an input table")
				return 0
			}
			staffID := luaTableTrimmedString(inTbl, "staff_id")
			if staffID == "" {
				staffID = caller.StaffID
			}
			var colCount int
			if v := inTbl.RawGetString("column_count"); v != lua.LNil {
				if n, isNum := v.(lua.LNumber); isNum {
					colCount = int(n)
				}
			}
			layoutTbl, _ := inTbl.RawGetString("layout").(*lua.LTable)
			items := parseDashboardWidgetItemsFromLua(layoutTbl)
			saved, savedCols, err := b.coreDAL.SaveStaffDashboardLayoutWithColumns(execCtx, staffID, items, colCount, caller.StaffID)
			if err != nil {
				L.RaiseError("dal.dashboard.save_staff_layout failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "staff_id", lua.LString(staffID))
			L.SetField(resTbl, "column_count", lua.LNumber(savedCols))
			L.SetField(resTbl, "layout", dashboardWidgetItemsToLuaTable(L, saved))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(dashTbl, "reset_staff_layout", L.NewFunction(func(L *lua.LState) int {
			if !endpointSpec.Mutates {
				L.RaiseError("dal.dashboard.reset_staff_layout is forbidden when endpoint mutates=false")
				return 0
			}
			var staffID string
			if L.GetTop() >= 1 {
				if tbl, isTbl := L.Get(1).(*lua.LTable); isTbl {
					staffID = luaTableTrimmedString(tbl, "staff_id")
				} else {
					staffID = strings.TrimSpace(L.OptString(1, ""))
				}
			}
			if staffID == "" {
				staffID = caller.StaffID
			}
			deleted, err := b.coreDAL.DeleteStaffDashboardLayout(execCtx, staffID)
			if err != nil {
				L.RaiseError("dal.dashboard.reset_staff_layout failed: %s", err.Error())
				return 0
			}
			resTbl := L.NewTable()
			L.SetField(resTbl, "staff_id", lua.LString(staffID))
			L.SetField(resTbl, "reset", lua.LBool(deleted))
			L.Push(resTbl)
			return 1
		}))
	}

	L.SetField(dalTbl, "dashboard", dashTbl)
}
