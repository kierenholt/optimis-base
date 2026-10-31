package host

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	defaultMaxDBSizeMB  = 500
	defaultSQLitePageSz = 4096
)

var sqlSchemaPrefixRegex = regexp.MustCompile(`(?i)\b([a-z0-9_]+)\s*\.\s*([a-z0-9_]+)\b`)

func validatePluginDBSQLScope(sqlQuery string, alias string) error {
	trimmed := strings.TrimSpace(sqlQuery)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "attach ") || strings.HasPrefix(lower, "detach ") {
		return errors.New("ATTACH and DETACH statements are forbidden in ctx.plugin_db")
	}

	matches := sqlSchemaPrefixRegex.FindAllStringSubmatch(lower, -1)
	for _, m := range matches {
		schemaPrefix := m[1]
		if schemaPrefix == "excluded" || schemaPrefix == "new" || schemaPrefix == "old" {
			continue
		}
		if schemaPrefix == "main" || schemaPrefix == "temp" || strings.HasPrefix(schemaPrefix, "plugin_") {
			if schemaPrefix != strings.ToLower(alias) {
				return fmt.Errorf("ctx.plugin_db is scoped to %s; access to schema %q is forbidden", alias, schemaPrefix)
			}
		}
	}
	return nil
}

func luaTableInstanceToSQLArgs(tbl *lua.LTable) []any {
	maxn := tbl.MaxN()
	args := make([]any, 0, maxn)
	for i := 1; i <= maxn; i++ {
		item := tbl.RawGetInt(i)
		switch v := item.(type) {
		case *lua.LNilType:
			args = append(args, nil)
		case lua.LBool:
			args = append(args, bool(v))
		case lua.LString:
			args = append(args, string(v))
		case lua.LNumber:
			f := float64(v)
			if f == float64(int64(f)) {
				args = append(args, int64(f))
			} else {
				args = append(args, f)
			}
		default:
			args = append(args, v.String())
		}
	}
	return args
}

func luaTableToSQLArgs(L *lua.LState, argIdx int) ([]any, error) {
	if L.GetTop() < argIdx {
		return nil, nil
	}
	lv := L.Get(argIdx)
	if lv == lua.LNil {
		return nil, nil
	}
	tbl, ok := lv.(*lua.LTable)
	if !ok {
		return nil, errors.New("SQL params argument must be a Lua table")
	}
	return luaTableInstanceToSQLArgs(tbl), nil
}

func (h *Host) buildLuaPluginDBTable(
	L *lua.LState,
	execCtx context.Context,
	plugin *LoadedPlugin,
	endpointSpec EndpointSpec,
	writeTx **sql.Tx,
	lockTracker *sqlLockTracker,
) *lua.LTable {
	if plugin != nil && plugin.Manifest.Database != nil && strings.TrimSpace(plugin.Manifest.Database.Alias) != "" {
		alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
		if !h.coreDAL.IsPluginDBAttached(alias) {
			_ = h.provisionAndMigratePluginDB(execCtx, plugin)
		}
	}

	pluginDBTbl := L.NewTable()
	L.SetField(pluginDBTbl, "query", L.NewFunction(func(L *lua.LState) int {
		sqlQuery := strings.TrimSpace(L.OptString(1, ""))
		if sqlQuery == "" {
			L.RaiseError("plugin_db.query requires a non-empty SQL query")
			return 0
		}
		if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
			L.RaiseError("plugin %s does not declare database.alias in manifest.json", plugin.Manifest.ID)
			return 0
		}
		alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
		if err := validatePluginDBSQLScope(sqlQuery, alias); err != nil {
			L.RaiseError("plugin_db.query scope violation: %s", err.Error())
			return 0
		}
		sqlArgs, err := luaTableToSQLArgs(L, 2)
		if err != nil {
			L.RaiseError("plugin_db.query invalid params: %s", err.Error())
			return 0
		}

		var rows *sql.Rows
		if *writeTx != nil {
			rows, err = (*writeTx).QueryContext(execCtx, sqlQuery, sqlArgs...)
		} else {
			rows, err = h.coreDAL.ReadDB().QueryContext(execCtx, sqlQuery, sqlArgs...)
		}
		if err != nil {
			L.RaiseError("plugin_db.query failed: %s", err.Error())
			return 0
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			L.RaiseError("plugin_db.query failed to read columns: %s", err.Error())
			return 0
		}

		resultTbl := L.NewTable()
		markLuaTableAsArray(L, resultTbl)
		rowIndex := 1

		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				L.RaiseError("plugin_db.query row scan failed: %s", err.Error())
				return 0
			}

			rowTbl := L.NewTable()
			for i, colName := range cols {
				rawVal := vals[i]
				switch v := rawVal.(type) {
				case nil:
					L.SetField(rowTbl, colName, lua.LNil)
				case int64:
					L.SetField(rowTbl, colName, lua.LNumber(v))
				case float64:
					L.SetField(rowTbl, colName, lua.LNumber(v))
				case bool:
					L.SetField(rowTbl, colName, lua.LBool(v))
				case string:
					L.SetField(rowTbl, colName, lua.LString(v))
				case []byte:
					L.SetField(rowTbl, colName, lua.LString(string(v)))
				default:
					L.SetField(rowTbl, colName, lua.LString(fmt.Sprintf("%v", v)))
				}
			}
			resultTbl.RawSetInt(rowIndex, rowTbl)
			rowIndex++
		}
		if err := rows.Err(); err != nil {
			L.RaiseError("plugin_db.query iteration error: %s", err.Error())
			return 0
		}

		L.Push(resultTbl)
		return 1
	}))

	if endpointSpec.Mutates {
		ensureOpenWriteTx := func(L *lua.LState, opName string) bool {
			if *writeTx != nil {
				return true
			}
			waitStart := time.Now()
			tx, txErr := h.coreDAL.WriteDB().BeginTx(execCtx, nil)
			if lockTracker != nil {
				lockTracker.lockWaitDuration += time.Since(waitStart)
			}
			if txErr != nil {
				L.RaiseError("%s failed to start WriteDB transaction: %s", opName, txErr.Error())
				return false
			}
			if lockTracker != nil {
				lockTracker.writeHoldStart = time.Now()
			}
			*writeTx = tx
			return true
		}

		L.SetField(pluginDBTbl, "exec", L.NewFunction(func(L *lua.LState) int {
			sqlQuery := strings.TrimSpace(L.OptString(1, ""))
			if sqlQuery == "" {
				L.RaiseError("plugin_db.exec requires a non-empty SQL statement")
				return 0
			}
			if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
				L.RaiseError("plugin %s does not declare database.alias in manifest.json", plugin.Manifest.ID)
				return 0
			}
			alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
			if err := validatePluginDBSQLScope(sqlQuery, alias); err != nil {
				L.RaiseError("plugin_db.exec scope violation: %s", err.Error())
				return 0
			}
			sqlArgs, err := luaTableToSQLArgs(L, 2)
			if err != nil {
				L.RaiseError("plugin_db.exec invalid params: %s", err.Error())
				return 0
			}

			if !ensureOpenWriteTx(L, "plugin_db.exec") {
				return 0
			}

			res, execErr := (*writeTx).ExecContext(execCtx, sqlQuery, sqlArgs...)
			if execErr != nil {
				L.RaiseError("plugin_db.exec failed: %s", execErr.Error())
				return 0
			}

			rowsAffected, _ := res.RowsAffected()
			lastInsertID, _ := res.LastInsertId()

			resTbl := L.NewTable()
			L.SetField(resTbl, "rows_affected", lua.LNumber(rowsAffected))
			L.SetField(resTbl, "last_insert_id", lua.LNumber(lastInsertID))
			L.Push(resTbl)
			return 1
		}))

		L.SetField(pluginDBTbl, "exec_batch", L.NewFunction(func(L *lua.LState) int {
			sqlQuery := strings.TrimSpace(L.OptString(1, ""))
			if sqlQuery == "" {
				L.RaiseError("plugin_db.exec_batch requires a non-empty SQL statement")
				return 0
			}
			if plugin.Manifest.Database == nil || strings.TrimSpace(plugin.Manifest.Database.Alias) == "" {
				L.RaiseError("plugin %s does not declare database.alias in manifest.json", plugin.Manifest.ID)
				return 0
			}
			alias := strings.TrimSpace(plugin.Manifest.Database.Alias)
			if err := validatePluginDBSQLScope(sqlQuery, alias); err != nil {
				L.RaiseError("plugin_db.exec_batch scope violation: %s", err.Error())
				return 0
			}
			batchTbl, ok := L.Get(2).(*lua.LTable)
			if !ok {
				L.RaiseError("plugin_db.exec_batch requires a Lua table array of parameter rows as 2nd argument")
				return 0
			}

			if !ensureOpenWriteTx(L, "plugin_db.exec_batch") {
				return 0
			}

			stmt, prepErr := (*writeTx).PrepareContext(execCtx, sqlQuery)
			if prepErr != nil {
				L.RaiseError("plugin_db.exec_batch prepare failed: %s", prepErr.Error())
				return 0
			}
			defer stmt.Close()

			maxn := batchTbl.MaxN()
			var totalRowsAffected int64
			for i := 1; i <= maxn; i++ {
				rowTbl, isTbl := batchTbl.RawGetInt(i).(*lua.LTable)
				if !isTbl {
					L.RaiseError("plugin_db.exec_batch invalid row at index %d: must be a Lua table", i)
					return 0
				}
				rowArgs := luaTableInstanceToSQLArgs(rowTbl)
				res, execErr := stmt.ExecContext(execCtx, rowArgs...)
				if execErr != nil {
					L.RaiseError("plugin_db.exec_batch failed at row %d: %s", i, execErr.Error())
					return 0
				}
				affected, _ := res.RowsAffected()
				totalRowsAffected += affected
			}

			resTbl := L.NewTable()
			L.SetField(resTbl, "rows_affected", lua.LNumber(totalRowsAffected))
			L.SetField(resTbl, "batch_size", lua.LNumber(maxn))
			L.Push(resTbl)
			return 1
		}))
	}

	return pluginDBTbl
}
