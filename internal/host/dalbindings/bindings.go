package dalbindings

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"optimis/internal/dal"
	"optimis/internal/host/types"
)

type Binder struct {
	coreDAL                 *dal.CoreDAL
	hostKMSKeyHandle        string
	listActivePlugins       func() []*types.LoadedPlugin
	ensureCloudKMSAvailable func(context.Context) error
}

func (b *Binder) ensureCloudKMS(ctx context.Context) error {
	if b.ensureCloudKMSAvailable != nil {
		return b.ensureCloudKMSAvailable(ctx)
	}
	return nil
}

func Bind(
	L *lua.LState,
	execCtx context.Context,
	coreDAL *dal.CoreDAL,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
	hostKMSKeyHandle string,
	listActivePlugins func() []*types.LoadedPlugin,
	ensureCloudKMS func(context.Context) error,
) *lua.LTable {
	b := &Binder{
		coreDAL:                 coreDAL,
		hostKMSKeyHandle:        hostKMSKeyHandle,
		listActivePlugins:       listActivePlugins,
		ensureCloudKMSAvailable: ensureCloudKMS,
	}
	return b.buildLuaDALTable(L, execCtx, endpointSpec, caller)
}

func hasDomainScope(scopes []string, prefix string) bool {
	for _, s := range scopes {
		if s == prefix || strings.HasPrefix(s, prefix+":") {
			return true
		}
	}
	return false
}

func markLuaTableAsArray(L *lua.LState, tbl *lua.LTable) {
	if tbl == nil {
		return
	}
	mt := L.NewTable()
	L.SetField(mt, "__is_array", lua.LTrue)
	L.SetMetatable(tbl, mt)
}

func mapSliceToLuaTable[T any](L *lua.LState, items []T, mapper func(*lua.LState, T) *lua.LTable) *lua.LTable {
	arrTbl := L.NewTable()
	markLuaTableAsArray(L, arrTbl)
	for i, item := range items {
		arrTbl.RawSetInt(i+1, mapper(L, item))
	}
	return arrTbl
}

func pushMappedSlice[T any](L *lua.LState, items []T, mapper func(*lua.LState, T) *lua.LTable) int {
	L.Push(mapSliceToLuaTable(L, items, mapper))
	return 1
}


func normalizeWidgetEntryRelPath(rawEntry string) (string, error) {
	trimmed := strings.TrimSpace(rawEntry)
	if trimmed == "" {
		return "", errors.New("empty widget entry path")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.HasPrefix(trimmed, "ui/dist/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/dist/")
	} else if strings.HasPrefix(trimmed, "ui/") {
		trimmed = strings.TrimPrefix(trimmed, "ui/")
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || clean == "" || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("disallowed widget entry path: %s", rawEntry)
	}
	return clean, nil
}

var piiScrubPattern = regexp.MustCompile(`(?i)\b(pup_[a-z0-9_]+|per_[a-z0-9_]+|[A-Z][0-9]{12})\b`)

func scrubDiagnosticToken(raw string) string {
	return piiScrubPattern.ReplaceAllString(raw, "[REDACTED]")
}

func callerHasAllRole(activeRoles []string) bool {
	for _, r := range activeRoles {
		switch strings.ToUpper(strings.TrimSpace(r)) {
		case "ADMIN", "HOD", "DSL", "SUPPORT", "LEADERSHIP", "DPO", "SYSTEM_ADMIN":
			return true
		}
	}
	return false
}

func resolveEffectiveDomainScopeMode(endpointScopes []string, caller types.CallerContext, domainPrefix string) dal.ScopeMode {
	for _, s := range endpointScopes {
		if s == domainPrefix+":all" || s == domainPrefix+":admin" {
			return dal.ScopeAll
		}
	}
	if caller.Type == "USER" {
		hasAll := false
		hasOwn := false
		for _, s := range caller.Scopes {
			if s == domainPrefix+":all" || s == domainPrefix+":admin" {
				hasAll = true
			}
			if s == domainPrefix+":own_groups" {
				hasOwn = true
			}
		}
		if hasAll && (!hasOwn || callerHasAllRole(caller.ActiveRoles)) {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func resolveDomainScopeMode(scopes []string, domainPrefix string) dal.ScopeMode {
	for _, s := range scopes {
		if s == domainPrefix+":all" || s == domainPrefix+":admin" {
			return dal.ScopeAll
		}
	}
	return dal.ScopeOwnGroups
}

func extractScopeFromLuaOpts(
	ctx context.Context,
	d *dal.CoreDAL,
	mode dal.ScopeMode,
	staffID string,
	optsTbl *lua.LTable,
) (dal.DALQueryScope, error) {
	scope := dal.DALQueryScope{
		Mode:    mode,
		StaffID: staffID,
	}
	if optsTbl != nil {
		if ay := optsTbl.RawGetString("academic_year_id"); ay != lua.LNil {
			scope.AcademicYearID = strings.TrimSpace(ay.String())
		}
		if rd := optsTbl.RawGetString("reference_date"); rd != lua.LNil {
			scope.ReferenceDate = strings.TrimSpace(rd.String())
		}
	}
	return d.ResolveScopeDefaults(ctx, scope)
}

func extractPupilIDsAndOptsFromLua(L *lua.LState) ([]string, *lua.LTable, error) {
	firstTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return nil, nil, errors.New("requires a table of pupil_ids")
	}

	var optsTbl *lua.LTable
	if L.GetTop() >= 2 {
		optsTbl, _ = L.Get(2).(*lua.LTable)
	}

	idsSourceTbl := firstTbl
	if nested, isTbl := firstTbl.RawGetString("pupil_ids").(*lua.LTable); isTbl {
		idsSourceTbl = nested
		if optsTbl == nil {
			optsTbl = firstTbl
		}
	}

	var pupilIDs []string
	maxn := idsSourceTbl.MaxN()
	for i := 1; i <= maxn; i++ {
		val := strings.TrimSpace(idsSourceTbl.RawGetInt(i).String())
		if val != "" {
			pupilIDs = append(pupilIDs, val)
		}
	}
	return pupilIDs, optsTbl, nil
}

func luaTableTrimmedString(tbl *lua.LTable, key string) string {
	if tbl == nil {
		return ""
	}
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return ""
	}
	return strings.TrimSpace(v.String())
}

func luaTableStringOr(tbl *lua.LTable, keys ...string) string {
	if tbl == nil {
		return ""
	}
	for _, k := range keys {
		if v := tbl.RawGetString(k); v != lua.LNil {
			trimmed := strings.TrimSpace(v.String())
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}


func luaTableInt(tbl *lua.LTable, key string, def int) int {
	if tbl == nil {
		return def
	}
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return def
	}
	if n, ok := v.(lua.LNumber); ok {
		return int(n)
	}
	return def
}

func luaTableFloat(tbl *lua.LTable, key string, def float64) float64 {
	if tbl == nil {
		return def
	}
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return def
	}
	if n, ok := v.(lua.LNumber); ok {
		return float64(n)
	}
	return def
}

func luaTableOptBoolPtr(tbl *lua.LTable, key string) *bool {
	if tbl == nil {
		return nil
	}
	v := tbl.RawGetString(key)
	if v == lua.LNil {
		return nil
	}
	b := v == lua.LTrue
	return &b
}

func (b *Binder) buildLuaDALTable(
	L *lua.LState,
	execCtx context.Context,
	endpointSpec types.EndpointSpec,
	caller types.CallerContext,
) *lua.LTable {
	dalTbl := L.NewTable()

	b.bindLuaDALAcademic(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALPupils(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALSen(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALMedical(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALGroups(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALStaff(L, execCtx, endpointSpec, dalTbl)
	b.bindLuaDALAttendance(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALGuardiansAndSafeguarding(L, execCtx, endpointSpec, caller, dalTbl)
	b.bindLuaDALCensus(L, execCtx, endpointSpec, dalTbl)
	b.bindLuaDALDashboard(L, execCtx, endpointSpec, caller, dalTbl)

	return dalTbl
}
